package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
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

type Entry struct {
	RowName   string `json:"row_name"`
	Date      string `json:"date"`
	Name      string `json:"name"`
	Nikki     string `json:"nikki"`
	Rank1     string `json:"rank1"`
	Point1    string `json:"point1"`
	Rank2     string `json:"rank2"`
	Point2    string `json:"point2"`
	Timestamp string `json:"timestamp"`
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

func cellStr(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func parseEntry(row []interface{}) (Entry, bool) {
	if len(row) == 0 {
		return Entry{}, false
	}
	rowName := cellStr(row[0])
	// YYYYMMDD で始まらない行はスキップ（ヘッダーや空行）
	if len(rowName) < 8 {
		return Entry{}, false
	}
	dateStr := rowName[:8]
	for _, c := range dateStr {
		if c < '0' || c > '9' {
			return Entry{}, false
		}
	}
	e := Entry{RowName: rowName}
	e.Date = dateStr[0:4] + "-" + dateStr[4:6] + "-" + dateStr[6:8]
	parts := strings.SplitN(rowName, "_", 2)
	if len(parts) == 2 && parts[1] != "nikki" {
		e.Name = parts[1]
	}
	if len(row) > 1 {
		e.Nikki = cellStr(row[1])
	}
	if len(row) > 2 {
		e.Rank1 = cellStr(row[2])
	}
	if len(row) > 3 {
		e.Point1 = cellStr(row[3])
	}
	if len(row) > 4 {
		e.Rank2 = cellStr(row[4])
	}
	if len(row) > 5 {
		e.Point2 = cellStr(row[5])
	}
	if len(row) > 6 {
		e.Timestamp = cellStr(row[6])
	}
	return e, true
}

func newSheetsService(ctx context.Context) (*sheets.Service, string, error) {
	credJSON := os.Getenv("GOOGLE_SERVICE_ACCOUNT_JSON")
	if credJSON == "" {
		return nil, "", fmt.Errorf("GOOGLE_SERVICE_ACCOUNT_JSON is not set")
	}
	sheetID := os.Getenv("GOOGLE_SHEET_ID")
	if sheetID == "" {
		return nil, "", fmt.Errorf("GOOGLE_SHEET_ID is not set")
	}
	srv, err := sheets.NewService(ctx, option.WithCredentialsJSON([]byte(credJSON)))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create sheets service: %w", err)
	}
	return srv, sheetID, nil
}

func appendToSheet(row []interface{}) error {
	ctx := context.Background()
	srv, sheetID, err := newSheetsService(ctx)
	if err != nil {
		return err
	}
	vr := &sheets.ValueRange{
		Values: [][]interface{}{row},
	}
	// A1 を起点にすることで必ずA列から追記される
	_, err = srv.Spreadsheets.Values.Append(sheetID, "シート1!A1", vr).
		ValueInputOption("USER_ENTERED").
		Do()
	if err != nil {
		return fmt.Errorf("failed to append row: %w", err)
	}
	return nil
}

func readEntries(ctx context.Context) ([]Entry, error) {
	srv, sheetID, err := newSheetsService(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := srv.Spreadsheets.Values.Get(sheetID, "シート1!A:G").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to read sheet: %w", err)
	}
	entries := make([]Entry, 0, len(resp.Values))
	for _, row := range resp.Values {
		if e, ok := parseEntry(row); ok {
			entries = append(entries, e)
		}
	}
	return entries, nil
}

// Handler は Vercel サーバーレス関数のエントリーポイント。
// GET  /api/nikki → 履歴一覧を返す
// POST /api/nikki → 日記を保存する
func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// GET: 履歴一覧を返す
	if r.Method == http.MethodGet {
		entries, err := readEntries(r.Context())
		if err != nil {
			log.Printf("read entries error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(entries)
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: "method not allowed"})
		return
	}

	// POST: 日記を保存する
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
