package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const driveFolderName = "apex-diary-photos"

// PhotoUploadResponse は写真アップロードの結果を表す。
type PhotoUploadResponse struct {
	Success bool   `json:"success"`
	FileID  string `json:"file_id,omitempty"`
	URL     string `json:"url,omitempty"`
	Message string `json:"message,omitempty"`
}

func newDriveService(ctx context.Context) (*drive.Service, error) {
	credJSON := os.Getenv("GOOGLE_SERVICE_ACCOUNT_JSON")
	if credJSON == "" {
		return nil, fmt.Errorf("GOOGLE_SERVICE_ACCOUNT_JSON is not set")
	}
	srv, err := drive.NewService(ctx,
		option.WithCredentialsJSON([]byte(credJSON)),
		option.WithScopes(drive.DriveFileScope),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %w", err)
	}
	return srv, nil
}

func getOrCreateFolder(ctx context.Context, srv *drive.Service) (string, error) {
	query := fmt.Sprintf("mimeType='application/vnd.google-apps.folder' and name='%s' and trashed=false", driveFolderName)
	list, err := srv.Files.List().Q(query).Fields("files(id, name)").Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to list folders: %w", err)
	}
	if len(list.Files) > 0 {
		return list.Files[0].Id, nil
	}
	folder := &drive.File{
		Name:     driveFolderName,
		MimeType: "application/vnd.google-apps.folder",
	}
	created, err := srv.Files.Create(folder).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to create folder: %w", err)
	}
	return created.Id, nil
}

// Handler は /api/photo の Vercel サーバーレス関数エントリーポイント。
func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	switch r.Method {
	case http.MethodPost:
		handlePhotoUpload(w, r)
	case http.MethodDelete:
		handlePhotoDelete(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(PhotoUploadResponse{Success: false, Message: "method not allowed"})
	}
}

func handlePhotoUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(PhotoUploadResponse{Success: false, Message: "フォームの解析に失敗しました"})
		return
	}

	file, header, err := r.FormFile("photo")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(PhotoUploadResponse{Success: false, Message: "photoフィールドが必要です"})
		return
	}
	defer file.Close()

	ctx := r.Context()
	drvSrv, err := newDriveService(ctx)
	if err != nil {
		log.Printf("drive service error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(PhotoUploadResponse{Success: false, Message: err.Error()})
		return
	}

	folderID, err := getOrCreateFolder(ctx, drvSrv)
	if err != nil {
		log.Printf("folder error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(PhotoUploadResponse{Success: false, Message: err.Error()})
		return
	}

	f := &drive.File{
		Name:    header.Filename,
		Parents: []string{folderID},
	}
	created, err := drvSrv.Files.Create(f).Media(file).Context(ctx).Do()
	if err != nil {
		log.Printf("upload error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(PhotoUploadResponse{Success: false, Message: fmt.Sprintf("アップロード失敗: %v", err)})
		return
	}

	permission := &drive.Permission{
		Type: "anyone",
		Role: "reader",
	}
	if _, err := drvSrv.Permissions.Create(created.Id, permission).Context(ctx).Do(); err != nil {
		log.Printf("permission error: %v", err)
	}

	url := "https://drive.google.com/uc?export=view&id=" + created.Id
	json.NewEncoder(w).Encode(PhotoUploadResponse{
		Success: true,
		FileID:  created.Id,
		URL:     url,
	})
}

func handlePhotoDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FileIDs []string `json:"file_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(PhotoUploadResponse{Success: false, Message: "invalid JSON"})
		return
	}

	ctx := r.Context()
	drvSrv, err := newDriveService(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(PhotoUploadResponse{Success: false, Message: err.Error()})
		return
	}

	for _, id := range req.FileIDs {
		if err := drvSrv.Files.Delete(id).Context(ctx).Do(); err != nil {
			log.Printf("failed to delete Drive file %s: %v", id, err)
		}
	}

	json.NewEncoder(w).Encode(PhotoUploadResponse{Success: true, Message: "削除しました"})
}
