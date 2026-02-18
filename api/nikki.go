package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type NikkiRequest struct {
	Date   string `json:"date"`
	Name   string `json:"name"`
	Nikki  string `json:"nikki"`
	Rank1  string `json:"rank1"`
	Point1 *int   `json:"point1"`
	Rank2  string `json:"rank2"`
	Point2 *int   `json:"point2"`
}

type NikkiResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func generateName(date, name string) (string, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return "", fmt.Errorf("invalid date format: %w", err)
	}
	yyyyMMdd := t.Format("20060102")
	if name == "" {
		return yyyyMMdd + "_nikki", nil
	}
	return yyyyMMdd + "_" + name, nil
}

func formatPoint(point *int) string {
	if point == nil {
		return ""
	}
	return fmt.Sprintf("%dRP", *point)
}

func appendToSheet(row []interface{}) error {
	ctx := context.Background()

	credJSON := os.Getenv("GOOGLE_SERVICE_ACCOUNT_JSON")
	sheetID := os.Getenv("GOOGLE_SHEET_ID")

	if credJSON == "" {
		return fmt.Errorf("GOOGLE_SERVICE_ACCOUNT_JSON is not set")
	}
	if sheetID == "" {
		return fmt.Errorf("GOOGLE_SHEET_ID is not set")
	}

	srv, err := sheets.NewService(ctx, option.WithCredentialsJSON([]byte(credJSON)))
	if err != nil {
		return fmt.Errorf("failed to create sheets service: %w", err)
	}

	rangeStr := "シート1!A:G"
	vr := &sheets.ValueRange{
		Values: [][]interface{}{row},
	}

	_, err = srv.Spreadsheets.Values.Append(sheetID, rangeStr, vr).
		ValueInputOption("USER_ENTERED").
		Do()
	if err != nil {
		return fmt.Errorf("failed to append row: %w", err)
	}

	return nil
}

// Handler is the Vercel serverless function entry point.
func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// GET: env変数の設定状況を確認するヘルスチェック
	if r.Method == http.MethodGet {
		type HealthResponse struct {
			CredSet   bool   `json:"cred_set"`
			CredValid bool   `json:"cred_valid"`
			SheetSet  bool   `json:"sheet_set"`
			SheetID   string `json:"sheet_id,omitempty"`
			Status    string `json:"status"`
		}
		credJSON := os.Getenv("GOOGLE_SERVICE_ACCOUNT_JSON")
		sheetID := os.Getenv("GOOGLE_SHEET_ID")
		credValid := credJSON != "" && json.Valid([]byte(credJSON))
		var status string
		switch {
		case credJSON == "":
			status = "ERROR: GOOGLE_SERVICE_ACCOUNT_JSON が未設定"
		case !credValid:
			status = "ERROR: GOOGLE_SERVICE_ACCOUNT_JSON が不正なJSON"
		case sheetID == "":
			status = "ERROR: GOOGLE_SHEET_ID が未設定"
		default:
			status = "OK: env変数はすべて設定されています"
		}
		json.NewEncoder(w).Encode(HealthResponse{
			CredSet:   credJSON != "",
			CredValid: credValid,
			SheetSet:  sheetID != "",
			SheetID:   sheetID,
			Status:    status,
		})
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: "method not allowed"})
		return
	}

	var req NikkiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: "invalid JSON"})
		return
	}

	if req.Date == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: "date is required"})
		return
	}

	rowName, err := generateName(req.Date, req.Name)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: err.Error()})
		return
	}

	jst, _ := time.LoadLocation("Asia/Tokyo")
	timestamp := time.Now().In(jst).Format("2006/01/02 15:04:05")

	row := []interface{}{
		rowName,
		req.Nikki,
		req.Rank1,
		formatPoint(req.Point1),
		req.Rank2,
		formatPoint(req.Point2),
		timestamp,
	}

	if err := appendToSheet(row); err != nil {
		log.Printf("sheets error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: err.Error()})
		return
	}

	json.NewEncoder(w).Encode(NikkiResponse{Success: true, Message: "保存しました"})
}
