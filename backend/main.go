package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
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

// sendNotification はメール通知を非同期で送信する。
// SMTP環境変数が未設定の場合は静かにスキップする。
func sendNotification(subject, body string) {
	to := os.Getenv("NOTIFICATION_EMAIL")
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")

	if to == "" || host == "" || user == "" || pass == "" {
		return
	}
	if port == "" {
		port = "587"
	}

	addr := host + ":" + port
	msg := []byte("To: " + to + "\r\n" +
		"From: " + user + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		body + "\r\n")

	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         host,
	}
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		// STARTTLSにフォールバック
		auth := smtp.PlainAuth("", user, pass, host)
		if smtpErr := smtp.SendMail(addr, auth, user, []string{to}, msg); smtpErr != nil {
			log.Printf("sendNotification error: %v", smtpErr)
		}
		return
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		log.Printf("sendNotification smtp client error: %v", err)
		return
	}
	defer client.Quit()

	auth := smtp.PlainAuth("", user, pass, host)
	if err := client.Auth(auth); err != nil {
		log.Printf("sendNotification auth error: %v", err)
		return
	}
	if err := client.Mail(user); err != nil {
		log.Printf("sendNotification MAIL error: %v", err)
		return
	}
	if err := client.Rcpt(to); err != nil {
		log.Printf("sendNotification RCPT error: %v", err)
		return
	}
	w, err := client.Data()
	if err != nil {
		log.Printf("sendNotification DATA error: %v", err)
		return
	}
	if _, err := w.Write(msg); err != nil {
		log.Printf("sendNotification write error: %v", err)
		return
	}
	if err := w.Close(); err != nil {
		log.Printf("sendNotification close error: %v", err)
		return
	}
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func handleNikki(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

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
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: "保存失敗"})
		return
	}

	go sendNotification(
		"[Nikki] 新しい日記が保存されました",
		fmt.Sprintf("日付: %s\nタイトル: %s\n\n日記が保存されました。", req.Date, req.Name),
	)

	json.NewEncoder(w).Encode(NikkiResponse{Success: true, Message: "保存しました"})
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

	rangeStr := "Sheet1!A:G"
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

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/nikki", corsMiddleware(handleNikki))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	log.Printf("Server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
