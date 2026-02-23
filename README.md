# Apex成長日記 🎮

ひらふうのApex Legendsプレイ記録をGoogleスプレッドシートに保存するWebアプリ。

> **システム仕様・設計書については `/design` ページを参照してください。**
> 本READMEはローカル環境のセットアップ・デプロイ手順に特化しています。

## 技術スタック

- **フロントエンド**: SvelteKit (static build)
- **バックエンド**: Go (Vercel Serverless Function)
- **DB**: Google Sheets API
- **画像**: Cloudinary
- **メール通知**: SMTP（Gmailアプリパスワード等）

---

## ディレクトリ構成

```
Nikki/
├── api/
│   ├── nikki.go        # Vercel Serverless Function（本番）
│   ├── photo.go        # 写真アップロード・削除（Cloudinary）
│   └── go.mod
├── backend/
│   ├── main.go         # ローカル開発用サーバー
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── routes/
│   │   │   ├── +layout.ts
│   │   │   ├── +page.svelte      # メインUI
│   │   │   └── design/
│   │   │       └── +page.svelte  # システム設計書ページ
│   │   └── app.html
│   ├── package.json
│   ├── svelte.config.js
│   └── vite.config.js
├── vercel.json
├── .env.example
└── README.md
```

---

## ローカル開発のセットアップ

### 1. Google Cloud 設定

#### Service Account の作成

1. [Google Cloud Console](https://console.cloud.google.com/) を開く
2. プロジェクトを作成（または既存のものを選択）
3. 「APIとサービス」→「ライブラリ」→「Google Sheets API」を有効化
4. 「IAMと管理」→「サービスアカウント」→「サービスアカウントを作成」
5. 名前を入力（例: `apex-diary-sa`）
6. 作成後、「キー」タブ→「鍵を追加」→「JSONキーを作成」でJSONをダウンロード

#### Google スプレッドシートの設定

1. [Google スプレッドシート](https://sheets.google.com) で新しいシートを作成
2. **1行目にヘッダーを追加**（A1〜I1）:
   ```
   name | nikki | rank1 | point1 | rank2 | point2 | timestamp | photos | feedback
   ```
3. URLから**スプレッドシートID**をコピー
   - 例: `https://docs.google.com/spreadsheets/d/【ここがID】/edit`
4. 右上「共有」ボタンで、Service AccountのメールアドレスをEditorとして共有

---

### 2. 環境変数の準備

`.env.example` をコピーして `.env` を作成し、各値を設定します:

```bash
cp .env.example backend/.env
```

| 変数名 | 必須 | 説明 |
|--------|------|------|
| `GOOGLE_SERVICE_ACCOUNT_JSON` | ○ | Service AccountのJSONを1行に圧縮した文字列 |
| `GOOGLE_SHEET_ID` | ○ | スプレッドシートID |
| `PORT` | — | ローカルサーバーポート（デフォルト: 8080）|
| `NOTIFICATION_EMAIL` | — | 通知先メールアドレス（例: ikere105@gmail.com）|
| `SMTP_HOST` | — | SMTPサーバー（例: smtp.gmail.com）|
| `SMTP_PORT` | — | SMTPポート（例: 465）|
| `SMTP_USER` | — | 送信元メールアドレス |
| `SMTP_PASS` | — | Gmailアプリパスワード（16桁）|

> **メール通知を使わない場合**: `NOTIFICATION_EMAIL` 〜 `SMTP_PASS` の4変数は未設定でも動作します。

**JSONの1行化方法:**
```bash
# macOS/Linux
cat your-service-account.json | jq -c .
```

---

### 3. バックエンド起動（Go）

```bash
cd backend
go mod tidy

# .envを読み込んでサーバー起動
source .env && go run main.go
# → http://localhost:8080 で起動
```

### 4. フロントエンド起動（SvelteKit）

```bash
cd frontend
npm install

# APIエンドポイントをローカルに向ける
echo "VITE_API_BASE_URL=http://localhost:8080/api" > .env.local

# 開発サーバー起動
npm run dev
# → http://localhost:5173 で起動
```

---

## Vercel へのデプロイ

### 前提条件

- [Vercel CLI](https://vercel.com/cli) インストール済み: `npm i -g vercel`
- Vercelアカウント作成済み

### デプロイ手順

```bash
# リポジトリルートで実行
vercel

# 本番デプロイ
vercel --prod
```

### Vercel 環境変数の設定

Vercel Dashboard → プロジェクト → Settings → Environment Variables で以下を追加:

| 変数名 | 値 |
|--------|-----|
| `GOOGLE_SERVICE_ACCOUNT_JSON` | Service AccountのJSON（1行に圧縮）|
| `GOOGLE_SHEET_ID` | スプレッドシートID |
| `NOTIFICATION_EMAIL` | 通知先メール（任意）|
| `SMTP_HOST` | SMTPサーバー（任意）|
| `SMTP_PORT` | SMTPポート（任意）|
| `SMTP_USER` | 送信元メール（任意）|
| `SMTP_PASS` | SMTPパスワード（任意）|

---

## スプレッドシートの列構造

| 列 | 論理名 | 形式 | 備考 |
|---|---|---|---|
| A | 行名 | `YYYYMMDD_タイトル` | 主キー相当 |
| B | 日記本文 | 文字列 | 必須 |
| C | ランク1 | 文字列 | 例: プラチナ |
| D | RP1 | 文字列 | 例: 1000RP |
| E | ランク2 | 文字列 | 省略可 |
| F | RP2 | 文字列 | 省略可 |
| G | 保存日時 | `YYYY/MM/DD HH:MM:SS` | JST |
| H | 写真URL | カンマ区切り文字列 | CloudinaryのHTTPS URL |
| I | フィードバック | JSON文字列 | 8カテゴリ＋署名者 |

---

## APIテスト (curl)

```bash
# 新規日記保存
curl -X POST http://localhost:8080/api/nikki \
  -H "Content-Type: application/json" \
  -d '{
    "date": "2026-02-17",
    "name": "night",
    "nikki": "今日はしろごまともぐ太とランクをした。",
    "rank1": "プラチナ",
    "point1": 1000,
    "rank2": "ダイヤ",
    "point2": 1200
  }'
```

成功時レスポンス:
```json
{"success": true, "message": "保存しました"}
```

---

## name 生成ルール

| 日付 | 名前入力 | 生成される name |
|------|----------|----------------|
| 2026-02-17 | (空) | `20260217_nikki` |
| 2026-02-17 | night | `20260217_night` |
| 2026-02-17 | しろごま | `20260217_しろごま` |
