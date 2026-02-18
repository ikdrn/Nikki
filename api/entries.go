package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

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
	// YYYYMMDD で始まらない行はヘッダーや空行とみなしてスキップ
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
	// YYYYMMDD → YYYY-MM-DD
	e.Date = dateStr[0:4] + "-" + dateStr[4:6] + "-" + dateStr[6:8]
	// YYYYMMDD_name の name 部分を抽出（"nikki" はデフォルト名なので空扱い）
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

// Handler は Vercel サーバーレス関数のエントリーポイント（GET /api/entries）
func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	credJSON := os.Getenv("GOOGLE_SERVICE_ACCOUNT_JSON")
	sheetID := os.Getenv("GOOGLE_SHEET_ID")

	if credJSON == "" {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "GOOGLE_SERVICE_ACCOUNT_JSON is not set"})
		return
	}
	if sheetID == "" {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "GOOGLE_SHEET_ID is not set"})
		return
	}

	ctx := context.Background()
	srv, err := sheets.NewService(ctx, option.WithCredentialsJSON([]byte(credJSON)))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to create sheets service: " + err.Error()})
		return
	}

	resp, err := srv.Spreadsheets.Values.Get(sheetID, "シート1!A:G").Do()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to read sheet: " + err.Error()})
		return
	}

	entries := make([]Entry, 0, len(resp.Values))
	for _, row := range resp.Values {
		if e, ok := parseEntry(row); ok {
			entries = append(entries, e)
		}
	}

	json.NewEncoder(w).Encode(entries)
}
