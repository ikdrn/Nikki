# Apex成長日記 🎮

ひらふうのApex Legendsプレイ記録をGoogleスプレッドシートに保存するWebアプリ。

## 技術スタック

- **フロントエンド**: SvelteKit (static build)
- **バックエンド**: Go (Vercel Serverless Function)
- **DB**: Google Sheets API

---

## ディレクトリ構成

```
Nikki/
├── api/
│   ├── nikki.go        # Vercel Serverless Function (本番)
│   └── go.mod
├── backend/
│   ├── main.go         # ローカル開発用サーバー
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── routes/
│   │   │   ├── +layout.ts
│   │   │   └── +page.svelte  # メインUI
│   │   └── app.html
│   ├── package.json
│   ├── svelte.config.js
│   └── vite.config.js
├── vercel.json
├── .env.example
└── README.md
```

---

## セットアップ手順

### 1. Google Cloud設定

#### Service Accountの作成

1. [Google Cloud Console](https://console.cloud.google.com/) を開く
2. プロジェクトを作成（または既存のものを選択）
3. 「APIとサービス」→「ライブラリ」→「Google Sheets API」を有効化
4. 「IAMと管理」→「サービスアカウント」→「サービスアカウントを作成」
5. 名前を入力（例: `apex-diary-sa`）
6. 作成後、「キー」タブ→「鍵を追加」→「JSONキーを作成」でJSONをダウンロード

#### Googleスプレッドシートの設定

1. [Googleスプレッドシート](https://sheets.google.com) で新しいシートを作成
2. **1行目にヘッダーを追加**（A1〜G1）:
   ```
   name | nikki | rank1 | point1 | rank2 | point2 | timestamp
   ```
3. URLから**スプレッドシートID**をコピー
   - 例: `https://docs.google.com/spreadsheets/d/【ここがID】/edit`
4. 右上「共有」ボタンで、Service AccountのメールアドレスをEditorとして共有

---

### 2. ローカル開発

#### バックエンド起動

```bash
# 依存関係インストール
cd backend
go mod tidy

# .envファイルを作成
cp ../.env.example .env
# .envを編集してGOOGLE_SERVICE_ACCOUNT_JSONとGOOGLE_SHEET_IDを設定

# サーバー起動
source .env && go run main.go
# → http://localhost:8080 で起動
```

#### フロントエンド起動

```bash
cd frontend
npm install

# .env.localを作成
echo "VITE_API_BASE_URL=http://localhost:8080/api" > .env.local

# 開発サーバー起動
npm run dev
# → http://localhost:5173 で起動
```

---

### 3. Vercelへのデプロイ

#### 前提条件

- [Vercel CLI](https://vercel.com/cli) インストール済み: `npm i -g vercel`
- Vercelアカウント作成済み

#### デプロイ手順

```bash
# リポジトリルートで実行
vercel

# 初回はプロジェクト設定を聞かれる
# - Project name: apex-diary (任意)
# - Root directory: . (そのままEnter)
```

#### 環境変数の設定

Vercel Dashboard → プロジェクト → Settings → Environment Variables で以下を追加:

| 変数名 | 値 |
|--------|-----|
| `GOOGLE_SERVICE_ACCOUNT_JSON` | Service AccountのJSON（1行に圧縮）|
| `GOOGLE_SHEET_ID` | スプレッドシートID |

**JSONの1行化方法:**
```bash
# macOS/Linux
cat your-service-account.json | tr -d '\n'

# またはjqを使う
cat your-service-account.json | jq -c .
```

#### 本番デプロイ

```bash
vercel --prod
```

---

## 動作確認

### APIテスト (curl)

```bash
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

## name生成ルール

| 日付 | 名前入力 | 生成されるname |
|------|----------|---------------|
| 2026-02-17 | (空) | `20260217_nikki` |
| 2026-02-17 | night | `20260217_night` |
| 2026-02-17 | しろごま | `20260217_しろごま` |

---

## スプレッドシートの列構造

| A | B | C | D | E | F | G |
|---|---|---|---|---|---|---|
| name | nikki | rank1 | point1 | rank2 | point2 | timestamp |
| 20260217_nikki | 今日は... | プラチナ | 1000RP | ダイヤ | 1200RP | 2026/02/17 10:13:46 |
