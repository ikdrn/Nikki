package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
)

// PhotoUploadResponse は写真アップロードの結果を表す。
type PhotoUploadResponse struct {
	Success bool   `json:"success"`
	FileID  string `json:"file_id,omitempty"`
	URL     string `json:"url,omitempty"`
	Message string `json:"message,omitempty"`
}

type cloudinaryUploadResp struct {
	SecureURL string `json:"secure_url"`
	PublicID  string `json:"public_id"`
	Error     *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Handler は /api/photo の Vercel サーバーレス関数エントリーポイント。
// Cloudinary を使って画像をアップロードする。
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
		// Cloudinary 上のファイルは Cloudinary ダッシュボードから管理。
		// シート側のIDは nikki.go の DELETE で削除済みのため、ここでは成功を返す。
		json.NewEncoder(w).Encode(PhotoUploadResponse{Success: true, Message: "削除しました"})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(PhotoUploadResponse{Success: false, Message: "method not allowed"})
	}
}

func handlePhotoUpload(w http.ResponseWriter, r *http.Request) {
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	uploadPreset := os.Getenv("CLOUDINARY_UPLOAD_PRESET")
	if cloudName == "" || uploadPreset == "" {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(PhotoUploadResponse{
			Success: false,
			Message: "環境変数 CLOUDINARY_CLOUD_NAME と CLOUDINARY_UPLOAD_PRESET を設定してください",
		})
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(PhotoUploadResponse{Success: false, Message: "フォームの解析に失敗しました"})
		return
	}

	file, _, err := r.FormFile("photo")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(PhotoUploadResponse{Success: false, Message: "photoフィールドが必要です"})
		return
	}
	defer file.Close()

	// Cloudinary 用のマルチパートリクエストを構築
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("upload_preset", uploadPreset)
	fw, err := mw.CreateFormFile("file", "photo.jpg")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(PhotoUploadResponse{Success: false, Message: "リクエスト構築失敗"})
		return
	}
	if _, err := io.Copy(fw, file); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(PhotoUploadResponse{Success: false, Message: "ファイル読込失敗"})
		return
	}
	mw.Close()

	uploadURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/upload", cloudName)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, uploadURL, &buf)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(PhotoUploadResponse{Success: false, Message: "リクエスト作成失敗"})
		return
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(PhotoUploadResponse{
			Success: false,
			Message: fmt.Sprintf("Cloudinaryへの送信失敗: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	var cldResp cloudinaryUploadResp
	if err := json.NewDecoder(resp.Body).Decode(&cldResp); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(PhotoUploadResponse{Success: false, Message: "レスポンス解析失敗"})
		return
	}

	if cldResp.Error != nil {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(PhotoUploadResponse{
			Success: false,
			Message: "アップロード失敗: " + cldResp.Error.Message,
		})
		return
	}

	// file_id としてフルURLを保存することで、将来のストレージ変更にも柔軟に対応できる
	json.NewEncoder(w).Encode(PhotoUploadResponse{
		Success: true,
		FileID:  cldResp.SecureURL,
		URL:     cldResp.SecureURL,
	})
}
