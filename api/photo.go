package handler

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
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

// Cloudinary URL から public_id を抽出する正規表現
// 例: https://res.cloudinary.com/{cloud}/image/upload/v1234567890/folder/image.jpg
//
//	→ public_id = "folder/image"
var rePublicID = regexp.MustCompile(`/v\d+/(.+)\.[^./]+$`)

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

	// file_id としてフルURLを保存することで将来のストレージ変更にも対応できる
	json.NewEncoder(w).Encode(PhotoUploadResponse{
		Success: true,
		FileID:  cldResp.SecureURL,
		URL:     cldResp.SecureURL,
	})
}

// handlePhotoDelete は Cloudinary から指定した写真を削除する。
// リクエストボディ: {"urls": ["https://res.cloudinary.com/..."]}
func handlePhotoDelete(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URLs []string `json:"urls"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.URLs) == 0 {
		json.NewEncoder(w).Encode(PhotoUploadResponse{Success: true})
		return
	}

	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")
	if cloudName == "" || apiKey == "" || apiSecret == "" {
		// 認証情報が未設定の場合はスキップ（アップロードプリセットのみで運用中など）
		json.NewEncoder(w).Encode(PhotoUploadResponse{Success: true})
		return
	}

	for _, imgURL := range body.URLs {
		m := rePublicID.FindStringSubmatch(imgURL)
		if len(m) < 2 {
			continue
		}
		deleteFromCloudinary(r.Context(), cloudName, apiKey, apiSecret, m[1])
	}

	json.NewEncoder(w).Encode(PhotoUploadResponse{Success: true})
}

// deleteFromCloudinary は Cloudinary の destroy API を呼び出して画像を削除する。
// 失敗してもサイレントにスキップする（ベストエフォート）。
func deleteFromCloudinary(ctx context.Context, cloudName, apiKey, apiSecret, publicID string) {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	// 署名 = SHA1("public_id={id}&timestamp={ts}{secret}")
	sigStr := fmt.Sprintf("public_id=%s&timestamp=%s%s", publicID, timestamp, apiSecret)
	h := sha1.New()
	h.Write([]byte(sigStr))
	signature := fmt.Sprintf("%x", h.Sum(nil))

	form := url.Values{}
	form.Set("public_id", publicID)
	form.Set("timestamp", timestamp)
	form.Set("api_key", apiKey)
	form.Set("signature", signature)

	destroyURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/destroy", cloudName)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, destroyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}
