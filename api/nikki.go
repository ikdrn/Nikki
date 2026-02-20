package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const sheetTab = "シート1"

type NikkiRequest struct {
	Date   string   `json:"date"`
	Name   string   `json:"name"`
	Nikki  string   `json:"nikki"`
	Rank1  string   `json:"rank1"`
	Point1 *int     `json:"point1"`
	Rank2  string   `json:"rank2"`
	Point2 *int     `json:"point2"`
	Photos []string `json:"photos"`
}

type UpdateRequest struct {
	RowIndex int      `json:"row_index"`
	Date     string   `json:"date"`
	Name     string   `json:"name"`
	Nikki    string   `json:"nikki"`
	Rank1    string   `json:"rank1"`
	Point1   *int     `json:"point1"`
	Rank2    string   `json:"rank2"`
	Point2   *int     `json:"point2"`
	Photos   []string `json:"photos"`
}

type DeleteRequest struct {
	RowIndices []int `json:"row_indices"`
}

type NikkiResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type Entry struct {
	RowIndex  int      `json:"row_index"`
	RowName   string   `json:"row_name"`
	Date      string   `json:"date"`
	Name      string   `json:"name"`
	Nikki     string   `json:"nikki"`
	Rank1     string   `json:"rank1"`
	Point1    string   `json:"point1"`
	Rank2     string   `json:"rank2"`
	Point2    string   `json:"point2"`
	Timestamp string   `json:"timestamp"`
	Photos    []string `json:"photos"`
	Feedback  string   `json:"feedback"`
}

func generateRowName(date, name string) (string, error) {
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

// parseEntry はシートの行データを Entry に変換する。
// sheetRow は 1 始まりのシート行番号。
func parseEntry(sheetRow int, row []interface{}) (Entry, bool) {
	if len(row) == 0 {
		return Entry{}, false
	}
	rowName := cellStr(row[0])
	if len(rowName) < 8 {
		return Entry{}, false
	}
	dateStr := rowName[:8]
	for _, c := range dateStr {
		if c < '0' || c > '9' {
			return Entry{}, false
		}
	}
	e := Entry{RowIndex: sheetRow, RowName: rowName}
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
	if len(row) > 7 {
		photosStr := cellStr(row[7])
		if photosStr != "" {
			e.Photos = strings.Split(photosStr, ",")
		}
	}
	if len(row) > 8 {
		e.Feedback = cellStr(row[8])
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

// getInternalSheetID はスプレッドシートから「シート1」の数値IDを取得する。
func getInternalSheetID(ctx context.Context, srv *sheets.Service, spreadsheetID string) (int64, error) {
	meta, err := srv.Spreadsheets.Get(spreadsheetID).Context(ctx).Do()
	if err != nil {
		return 0, fmt.Errorf("failed to get spreadsheet metadata: %w", err)
	}
	for _, s := range meta.Sheets {
		if s.Properties.Title == sheetTab {
			return s.Properties.SheetId, nil
		}
	}
	return 0, fmt.Errorf("sheet %q not found", sheetTab)
}

func readEntries(ctx context.Context) ([]Entry, error) {
	srv, spreadsheetID, err := newSheetsService(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetID, sheetTab+"!A:I").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to read sheet: %w", err)
	}
	entries := make([]Entry, 0, len(resp.Values))
	for i, row := range resp.Values {
		if e, ok := parseEntry(i+1, row); ok {
			entries = append(entries, e)
		}
	}
	return entries, nil
}

func appendToSheet(row []interface{}) error {
	ctx := context.Background()
	srv, spreadsheetID, err := newSheetsService(ctx)
	if err != nil {
		return err
	}
	vr := &sheets.ValueRange{Values: [][]interface{}{row}}
	_, err = srv.Spreadsheets.Values.Append(spreadsheetID, sheetTab+"!A1", vr).
		ValueInputOption("USER_ENTERED").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to append row: %w", err)
	}
	return nil
}

func buildRow(date, name, nikki, rank1, rank2 string, point1, point2 *int, photos []string) ([]interface{}, string, error) {
	rowName, err := generateRowName(date, name)
	if err != nil {
		return nil, "", err
	}
	jst, _ := time.LoadLocation("Asia/Tokyo")
	timestamp := time.Now().In(jst).Format("2006/01/02 15:04:05")
	row := []interface{}{
		rowName,
		nikki,
		rank1,
		formatPoint(point1),
		rank2,
		formatPoint(point2),
		timestamp,
		strings.Join(photos, ","),
	}
	return row, rowName, nil
}

// Handler は Vercel サーバーレス関数のエントリーポイント。
func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	switch r.Method {
	case http.MethodGet:
		handleGet(w, r)
	case http.MethodPost:
		handlePost(w, r)
	case http.MethodPut:
		handlePut(w, r)
	case http.MethodPatch:
		handlePatch(w, r)
	case http.MethodDelete:
		handleDelete(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: "method not allowed"})
	}
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	entries, err := readEntries(r.Context())
	if err != nil {
		log.Printf("read entries error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(entries)
}

func handlePost(w http.ResponseWriter, r *http.Request) {
	var req NikkiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: "invalid JSON"})
		return
	}

	if strings.TrimSpace(req.Nikki) == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: "日記の内容は必須です"})
		return
	}
	if req.Date == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: "日付は必須です"})
		return
	}

	row, _, err := buildRow(req.Date, req.Name, req.Nikki, req.Rank1, req.Rank2, req.Point1, req.Point2, req.Photos)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: err.Error()})
		return
	}

	if err := appendToSheet(row); err != nil {
		log.Printf("sheets error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: err.Error()})
		return
	}

	json.NewEncoder(w).Encode(NikkiResponse{Success: true, Message: "保存しました"})
}

func handlePut(w http.ResponseWriter, r *http.Request) {
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: "invalid JSON"})
		return
	}
	if req.RowIndex <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: "invalid row_index"})
		return
	}
	if strings.TrimSpace(req.Nikki) == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: "日記の内容は必須です"})
		return
	}
	if req.Date == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: "日付は必須です"})
		return
	}

	rowName, err := generateRowName(req.Date, req.Name)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: err.Error()})
		return
	}

	ctx := r.Context()
	srv, spreadsheetID, err := newSheetsService(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: err.Error()})
		return
	}

	// 既存行のタイムスタンプを保持する
	rangeStr := fmt.Sprintf("%s!A%d:H%d", sheetTab, req.RowIndex, req.RowIndex)
	existResp, err := srv.Spreadsheets.Values.Get(spreadsheetID, rangeStr).Context(ctx).Do()
	var timestamp string
	if err == nil && len(existResp.Values) > 0 && len(existResp.Values[0]) > 6 {
		timestamp = cellStr(existResp.Values[0][6])
	}
	if timestamp == "" {
		jst, _ := time.LoadLocation("Asia/Tokyo")
		timestamp = time.Now().In(jst).Format("2006/01/02 15:04:05")
	}

	row := []interface{}{
		rowName,
		req.Nikki,
		req.Rank1,
		formatPoint(req.Point1),
		req.Rank2,
		formatPoint(req.Point2),
		timestamp,
		strings.Join(req.Photos, ","),
	}
	vr := &sheets.ValueRange{Values: [][]interface{}{row}}
	_, err = srv.Spreadsheets.Values.Update(spreadsheetID, rangeStr, vr).
		ValueInputOption("USER_ENTERED").Context(ctx).Do()
	if err != nil {
		log.Printf("update error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: "更新失敗"})
		return
	}

	json.NewEncoder(w).Encode(NikkiResponse{Success: true, Message: "更新しました"})
}

// handlePatch はフィードバックのみを更新する（I列）。
func handlePatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RowIndex int    `json:"row_index"`
		Feedback string `json:"feedback"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: "invalid JSON"})
		return
	}
	if req.RowIndex <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: "invalid row_index"})
		return
	}

	ctx := r.Context()
	srv, spreadsheetID, err := newSheetsService(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: err.Error()})
		return
	}

	// I列（index 8）= feedback のみを更新
	rangeStr := fmt.Sprintf("%s!I%d", sheetTab, req.RowIndex)
	vr := &sheets.ValueRange{Values: [][]interface{}{{req.Feedback}}}
	_, err = srv.Spreadsheets.Values.Update(spreadsheetID, rangeStr, vr).
		ValueInputOption("USER_ENTERED").Context(ctx).Do()
	if err != nil {
		log.Printf("feedback update error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: "フィードバック保存失敗"})
		return
	}

	json.NewEncoder(w).Encode(NikkiResponse{Success: true, Message: "フィードバックを保存しました"})
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	var req DeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: "invalid JSON"})
		return
	}
	if len(req.RowIndices) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: "no rows specified"})
		return
	}

	ctx := r.Context()
	srv, spreadsheetID, err := newSheetsService(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: err.Error()})
		return
	}

	internalSheetID, err := getInternalSheetID(ctx, srv, spreadsheetID)
	if err != nil {
		log.Printf("get sheet id error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: "シート情報の取得に失敗しました"})
		return
	}

	// 行番号を降順にソートして上から削除するとインデックスがずれない
	sort.Sort(sort.Reverse(sort.IntSlice(req.RowIndices)))

	requests := make([]*sheets.Request, 0, len(req.RowIndices))
	for _, idx := range req.RowIndices {
		requests = append(requests, &sheets.Request{
			DeleteDimension: &sheets.DeleteDimensionRequest{
				Range: &sheets.DimensionRange{
					SheetId:    internalSheetID,
					Dimension:  "ROWS",
					StartIndex: int64(idx - 1), // 0-based
					EndIndex:   int64(idx),
				},
			},
		})
	}

	batchReq := &sheets.BatchUpdateSpreadsheetRequest{Requests: requests}
	_, err = srv.Spreadsheets.BatchUpdate(spreadsheetID, batchReq).Context(ctx).Do()
	if err != nil {
		log.Printf("delete error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NikkiResponse{Success: false, Message: "削除失敗"})
		return
	}

	json.NewEncoder(w).Encode(NikkiResponse{Success: true, Message: "削除しました"})
}
