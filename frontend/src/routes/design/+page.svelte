<script>
  const sections = [
    { id: 'section-a', label: 'A', title: '要件定義書', subtitle: 'ビジネスとシステムの架け橋' },
    { id: 'section-b', label: 'B', title: '基本設計書', subtitle: 'ユーザー視点の全体設計' },
    { id: 'section-c', label: 'C', title: '詳細設計書', subtitle: 'エンジニア向けの製造指針' },
    { id: 'section-d', label: 'D', title: '単体テスト仕様書', subtitle: 'プログラム単位の検証' },
    { id: 'section-e', label: 'E', title: '結合テスト仕様書', subtitle: 'インターフェースの検証' },
    { id: 'section-f', label: 'F', title: '総合テスト仕様書', subtitle: 'エンドツーエンドの検証' },
    { id: 'section-g', label: 'G', title: 'トレーサビリティマトリクス', subtitle: '管理の必須項目' },
  ];
</script>

<svelte:head>
  <title>Nikki システム設計書</title>
</svelte:head>

<div class="page-wrapper">

  <!-- ドキュメントヘッダー -->
  <header class="doc-header">
    <div class="doc-header-inner">
      <div class="doc-meta">
        <span class="doc-meta-item">文書番号: NIKKI-SPEC-001</span>
        <span class="doc-meta-sep">|</span>
        <span class="doc-meta-item">版: 1.0</span>
        <span class="doc-meta-sep">|</span>
        <span class="doc-meta-item">最終更新: 2026年2月</span>
      </div>
      <h1 class="doc-title">Nikki システム設計書</h1>
      <p class="doc-subtitle">Apex成長日記 — システム仕様・設計・テスト計画の総合ドキュメント</p>
      <a href="/" class="back-link">← アプリに戻る</a>
    </div>
  </header>

  <div class="doc-body">

    <!-- 目次 -->
    <section class="toc-section">
      <h2 class="toc-heading">目次</h2>
      <ol class="toc-list">
        {#each sections as s}
          <li><a href="#{s.id}" class="toc-link">{s.label}. {s.title}<span class="toc-subtitle"> — {s.subtitle}</span></a></li>
        {/each}
      </ol>
    </section>

    <hr class="divider" />

    <!-- A: 要件定義書 -->
    <section id="section-a" class="doc-section">
      <div class="section-header">
        <span class="section-label">A</span>
        <div>
          <h2 class="section-title">要件定義書</h2>
          <p class="section-subtitle">ビジネスとシステムの架け橋</p>
        </div>
      </div>
      <p class="section-intro">
        「何を作るか」を言葉で明確にするドキュメントです。ここで曖昧さを残すと、
        後の工程で手戻りが発生します。非エンジニアの方でも読めるよう、平易な言葉で記述します。
      </p>

      <div class="subsection">
        <h3 class="subsection-title">業務フロー（現状 / 導入後）</h3>
        <p class="subsection-body">
          <strong>現状（As-Is）:</strong> プレイ後の成績をメモ帳やLINEで記録。検索・集計が困難。<br />
          <strong>導入後（To-Be）:</strong> Nikkiアプリにアクセスし、日付・ランク・RP・日記テキストを入力して保存。
          Googleスプレッドシートに自動記録され、フィードバックの受け取りや過去ログの検索が可能になる。<br />
          <strong>例外ケース:</strong> ネットワーク障害時は画面にエラーを表示。下書き自動保存（IndexedDB）があるため入力内容は失われない。
        </p>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">機能要件一覧</h3>
        <table class="spec-table">
          <thead>
            <tr><th>ID</th><th>機能名</th><th>説明</th></tr>
          </thead>
          <tbody>
            <tr><td>FR-01</td><td>日記の新規作成</td><td>日付・タイトル・本文・ランク・RP・写真を入力して保存する</td></tr>
            <tr><td>FR-02</td><td>日記の一覧表示</td><td>過去のエントリを日付降順で一覧表示する</td></tr>
            <tr><td>FR-03</td><td>日記の検索・絞り込み</td><td>タイトル・日付・ランクでフィルタリングできる</td></tr>
            <tr><td>FR-04</td><td>日記の編集・削除</td><td>既存エントリを修正または削除できる（複数選択一括削除対応）</td></tr>
            <tr><td>FR-05</td><td>フィードバックの入力</td><td>パスワード認証後、8カテゴリ＋自由記述でフィードバックを付与できる</td></tr>
            <tr><td>FR-06</td><td>写真アップロード</td><td>1エントリにつき最大5枚をCloudinaryにアップロードして紐付ける</td></tr>
            <tr><td>FR-07</td><td>データエクスポート</td><td>PDF・Excel・Markdown・テキスト・写真ZIPの5形式でエクスポートできる</td></tr>
            <tr><td>FR-08</td><td>下書き自動保存</td><td>入力中の内容をブラウザ内（IndexedDB）に自動保存する</td></tr>
            <tr><td>FR-09</td><td>メール通知</td><td>日記・フィードバック保存時に管理者メールアドレスへ通知する</td></tr>
          </tbody>
        </table>
        <p class="note">将来的な拡張予定: 複数ユーザー対応（ログイン機能）、月次サマリーの自動生成、グラフ可視化。</p>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">非機能要件グレード</h3>
        <table class="spec-table">
          <thead>
            <tr><th>項目</th><th>目標値</th><th>備考</th></tr>
          </thead>
          <tbody>
            <tr><td>応答時間（95パーセンタイル）</td><td>3秒以内</td><td>Vercelエッジ + Google Sheets API</td></tr>
            <tr><td>同時接続数</td><td>〜5ユーザー</td><td>個人・少人数チーム用途</td></tr>
            <tr><td>稼働率（SLA）</td><td>99%以上</td><td>Vercel無料プランの稼働率に準拠</td></tr>
            <tr><td>RTO（復旧目標時間）</td><td>24時間以内</td><td>個人利用のためベストエフォート</td></tr>
            <tr><td>RPO（復旧目標時点）</td><td>直近の保存時点</td><td>Googleスプレッドシートに即時反映</td></tr>
          </tbody>
        </table>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">情報セキュリティ要件</h3>
        <ul class="spec-list">
          <li>データの重要度分類: プレイ記録（個人情報なし）— 一般データとして扱う</li>
          <li>フィードバック入力はパスワード（PIN）認証で保護（4桁数字）</li>
          <li>Google Service AccountキーはVercel環境変数として管理し、ソースコードには含めない</li>
          <li>Cloudinary APIキーも環境変数で管理</li>
          <li>通信はすべてHTTPS（Vercel標準）</li>
        </ul>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">システム境界図（外部サービス連携）</h3>
        <table class="spec-table">
          <thead>
            <tr><th>外部サービス</th><th>役割</th><th>APIレートリミット</th></tr>
          </thead>
          <tbody>
            <tr><td>Google Sheets API</td><td>データの永続化（DB代替）</td><td>100リクエスト/100秒/ユーザー</td></tr>
            <tr><td>Cloudinary</td><td>画像ストレージ</td><td>無料プラン: 25クレジット/月</td></tr>
            <tr><td>Vercel</td><td>ホスティング・サーバーレス実行</td><td>関数実行: 最大10〜30秒</td></tr>
            <tr><td>SMTP（Gmail等）</td><td>メール通知送信</td><td>Gmail: 500通/日（Googleアカウント）</td></tr>
          </tbody>
        </table>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">用語集</h3>
        <table class="spec-table">
          <thead>
            <tr><th>用語</th><th>定義</th></tr>
          </thead>
          <tbody>
            <tr><td>RP</td><td>Ranked Points。Apex Legendsのランクマッチで得られるポイント</td></tr>
            <tr><td>エントリ</td><td>1回分のプレイ記録（日記）</td></tr>
            <tr><td>フィードバック</td><td>コーチや仲間からのコメント。8カテゴリ＋自由記述で構成</td></tr>
            <tr><td>行名（RowName）</td><td>スプレッドシートの主キー。形式: YYYYMMDD_タイトル</td></tr>
            <tr><td>下書き</td><td>未保存の入力内容。ブラウザのIndexedDBに自動保存される</td></tr>
          </tbody>
        </table>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">ブラウザ・OSサポート範囲</h3>
        <ul class="spec-list">
          <li>対象ブラウザ: Chrome最新版、Safari最新版（iOS含む）、Edge最新版</li>
          <li>切り捨て基準: ES2020非対応ブラウザ（Internet Explorer等）は対象外</li>
          <li>スマートフォン: iOS 15以上、Android 10以上を動作確認対象とする</li>
        </ul>
      </div>
    </section>

    <hr class="divider" />

    <!-- B: 基本設計書 -->
    <section id="section-b" class="doc-section">
      <div class="section-header">
        <span class="section-label">B</span>
        <div>
          <h2 class="section-title">基本設計書</h2>
          <p class="section-subtitle">ユーザー視点の全体設計</p>
        </div>
      </div>
      <p class="section-intro">
        アプリの「骨格」を定義します。画面の流れ、データの持ち方、外部サービスとの連携方法など、
        システム全体の構造を非エンジニアでも理解できるよう図と表で説明します。
      </p>

      <div class="subsection">
        <h3 class="subsection-title">システム構成図</h3>
        <pre class="ascii-diagram">
ユーザーのブラウザ
       │
       │ HTTPS
       ▼
  ┌─────────────────────────┐
  │       Vercel CDN         │ ← フロントエンド配信（SvelteKit静的ビルド）
  │  /frontend/build/        │
  └─────────────────────────┘
       │
       │ API呼び出し（/api/nikki, /api/photo）
       ▼
  ┌─────────────────────────┐
  │  Vercel Serverless Fn    │ ← Goで記述されたAPIハンドラー
  │  api/nikki.go            │
  │  api/photo.go            │
  └──────────┬──────────────┘
             │
     ┌───────┴───────┐
     ▼               ▼
Google Sheets      Cloudinary
（データ保存）     （画像保存）
        </pre>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">画面遷移図</h3>
        <pre class="ascii-diagram">
[トップページ /]
    │
    ├── タブ: 書く
    │       └── フォーム入力 → 保存 → 「書く」タブに戻る
    │
    ├── タブ: 履歴
    │       ├── エントリ一覧
    │       │       ├── 編集ボタン → インライン編集フォーム
    │       │       ├── フィードバックボタン → フィードバックモーダル
    │       │       └── 削除ボタン（複数選択可）
    │       └── エクスポート → プレビューモーダル → ダウンロード
    │
    └── フッター
            └── [システム設計書 /design] ← 本ページ
        </pre>
        <p class="note">権限制御: フィードバック入力はPINコード（4桁）入力後のみ有効。入力欄はPIN未入力時に非活性。</p>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">データモデル（Googleスプレッドシート）</h3>
        <table class="spec-table">
          <thead>
            <tr><th>列</th><th>論理名</th><th>物理名</th><th>型・形式</th><th>備考</th></tr>
          </thead>
          <tbody>
            <tr><td>A</td><td>行名</td><td>name</td><td>文字列</td><td>YYYYMMDD_タイトル（主キー相当）</td></tr>
            <tr><td>B</td><td>日記本文</td><td>nikki</td><td>文字列（改行可）</td><td>必須</td></tr>
            <tr><td>C</td><td>ランク1</td><td>rank1</td><td>文字列</td><td>例: プラチナ</td></tr>
            <tr><td>D</td><td>RP1</td><td>point1</td><td>文字列</td><td>例: 1000RP</td></tr>
            <tr><td>E</td><td>ランク2</td><td>rank2</td><td>文字列</td><td>省略可</td></tr>
            <tr><td>F</td><td>RP2</td><td>point2</td><td>文字列</td><td>省略可</td></tr>
            <tr><td>G</td><td>保存日時</td><td>timestamp</td><td>文字列</td><td>JST: YYYY/MM/DD HH:MM:SS</td></tr>
            <tr><td>H</td><td>写真URL一覧</td><td>photos</td><td>文字列（カンマ区切り）</td><td>CloudinaryのHTTPS URL</td></tr>
            <tr><td>I</td><td>フィードバック</td><td>feedback</td><td>JSON文字列</td><td>8カテゴリ＋署名者</td></tr>
          </tbody>
        </table>
        <p class="note">インデックス戦略: 現在はシーケンシャルスキャン（行数〜数百件を想定）。1万件超の場合はBigQueryやFirestoreへの移行を検討する。</p>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">外部I/F定義（APIエンドポイント）</h3>
        <table class="spec-table">
          <thead>
            <tr><th>メソッド</th><th>パス</th><th>処理</th><th>タイムアウト</th></tr>
          </thead>
          <tbody>
            <tr><td>GET</td><td>/api/nikki</td><td>全エントリ取得</td><td>10秒</td></tr>
            <tr><td>POST</td><td>/api/nikki</td><td>新規エントリ作成</td><td>10秒</td></tr>
            <tr><td>PUT</td><td>/api/nikki</td><td>既存エントリ更新</td><td>10秒</td></tr>
            <tr><td>PATCH</td><td>/api/nikki</td><td>フィードバックのみ更新</td><td>10秒</td></tr>
            <tr><td>DELETE</td><td>/api/nikki</td><td>エントリ削除（複数可）</td><td>10秒</td></tr>
            <tr><td>POST</td><td>/api/photo</td><td>写真アップロード（Cloudinary）</td><td>30秒</td></tr>
            <tr><td>DELETE</td><td>/api/photo</td><td>写真削除（Cloudinary）</td><td>30秒</td></tr>
          </tbody>
        </table>
        <p class="note">再送制御: フロントエンドはリトライなし（ユーザーが手動で再試行）。エラー時はトースト通知で案内する。</p>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">認証・認可設計</h3>
        <ul class="spec-list">
          <li>現バージョン: フィードバック入力のみPINコード（4桁）で認証。セッション管理なし（リロードで再入力が必要）</li>
          <li>日記の読み書きはURL知っている者全員が可能（クローズドURL運用を前提）</li>
          <li>将来拡張: Googleアカウント連携（OAuth 2.0）による本人確認</li>
        </ul>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">共通エラーハンドリング</h3>
        <table class="spec-table">
          <thead>
            <tr><th>HTTPステータス</th><th>画面表示メッセージ</th><th>ログレベル</th></tr>
          </thead>
          <tbody>
            <tr><td>400</td><td>「入力内容に誤りがあります」</td><td>WARN</td></tr>
            <tr><td>500</td><td>「サーバーエラーが発生しました」</td><td>ERROR</td></tr>
            <tr><td>ネットワーク断</td><td>「通信エラー。ネットワークを確認してください」</td><td>WARN</td></tr>
          </tbody>
        </table>
      </div>
    </section>

    <hr class="divider" />

    <!-- C: 詳細設計書 -->
    <section id="section-c" class="doc-section">
      <div class="section-header">
        <span class="section-label">C</span>
        <div>
          <h2 class="section-title">詳細設計書</h2>
          <p class="section-subtitle">エンジニア向けの製造指針</p>
        </div>
      </div>
      <p class="section-intro">
        「どう作るか」の具体的な指針です。関数・クラスの設計、データの流れ、バリデーションルールを
        明記することで、コードの保守性を高めます。
      </p>

      <div class="subsection">
        <h3 class="subsection-title">モジュール構成</h3>
        <table class="spec-table">
          <thead>
            <tr><th>ファイル</th><th>役割</th><th>主な関数・処理</th></tr>
          </thead>
          <tbody>
            <tr><td>frontend/src/routes/+page.svelte</td><td>メインUI（全機能）</td><td>saveNikki, loadEntries, openFeedback, exportPDF 等</td></tr>
            <tr><td>frontend/src/routes/design/+page.svelte</td><td>設計書ページ</td><td>静的コンテンツのみ</td></tr>
            <tr><td>api/nikki.go</td><td>日記CRUD API（Vercel）</td><td>Handler, handlePost, handlePatch, sendNotification</td></tr>
            <tr><td>api/photo.go</td><td>写真API（Vercel）</td><td>Handler（写真アップロード・削除）</td></tr>
            <tr><td>backend/main.go</td><td>ローカル開発用サーバー</td><td>handleNikki, sendNotification</td></tr>
          </tbody>
        </table>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">主要シーケンス図（日記保存）</h3>
        <pre class="ascii-diagram">
ブラウザ          SvelteKit         Vercel Fn        Google Sheets
   │                 │                  │                   │
   │ 保存ボタン押下  │                  │                   │
   │────────────────►│                  │                   │
   │                 │ POST /api/nikki  │                   │
   │                 │─────────────────►│                   │
   │                 │                  │ Sheets.Append     │
   │                 │                  │──────────────────►│
   │                 │                  │      OK           │
   │                 │                  │◄──────────────────│
   │                 │                  │ go sendNotification()
   │                 │ 200 [success]    │ （非同期・メール送信）
   │                 │◄─────────────────│                   │
   │ トースト表示    │                  │                   │
   │◄────────────────│                  │                   │
        </pre>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">フィードバックJSON構造（I列）</h3>
        <pre class="code-block">{`{
  "free": "自由記述コメント",
  "aim": "エイム評価",
  "positioning": "ポジショニング評価",
  "judgment": "判断力評価",
  "teamwork": "チームワーク評価",
  "ability": "アビリティ使用評価",
  "movement": "動き評価",
  "resources": "リソース管理評価",
  "mental": "メンタル評価",
  "signer": "署名者名"
}`}</pre>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">バリデーション設計</h3>
        <table class="spec-table">
          <thead>
            <tr><th>フィールド</th><th>フロント側検証</th><th>サーバー側検証</th></tr>
          </thead>
          <tbody>
            <tr><td>日付</td><td>空欄NG（必須）</td><td>YYYY-MM-DD形式チェック</td></tr>
            <tr><td>日記本文</td><td>空欄NG（必須）</td><td>空文字・空白のみNG</td></tr>
            <tr><td>写真</td><td>4MB以下・最大5枚</td><td>Cloudinary側でサイズ検証</td></tr>
            <tr><td>フィードバックPIN</td><td>4桁数字一致チェック</td><td>サーバー非検証（クライアント完結）</td></tr>
            <tr><td>RowIndex</td><td>—</td><td>1以上の整数であること</td></tr>
          </tbody>
        </table>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">CRUD図（どの機能がどのデータを操作するか）</h3>
        <table class="spec-table">
          <thead>
            <tr><th>機能</th><th>シート読み込み（R）</th><th>追記（C）</th><th>更新（U）</th><th>削除（D）</th></tr>
          </thead>
          <tbody>
            <tr><td>日記一覧表示</td><td>●</td><td></td><td></td><td></td></tr>
            <tr><td>日記新規保存</td><td></td><td>●</td><td></td><td></td></tr>
            <tr><td>日記編集</td><td>●（タイムスタンプ取得）</td><td></td><td>●</td><td></td></tr>
            <tr><td>日記削除</td><td></td><td></td><td></td><td>●</td></tr>
            <tr><td>フィードバック保存</td><td></td><td></td><td>●（I列のみ）</td><td></td></tr>
          </tbody>
        </table>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">キャッシュ設計</h3>
        <ul class="spec-list">
          <li>現バージョン: キャッシュなし（毎回APIから全件取得）</li>
          <li>下書き: IndexedDB（apex-diary DB / draft ストア）にキャッシュ。1500msデバウンス後に自動保存</li>
          <li>写真: Cloudinary CDNのキャッシュに依存（TTL: Cloudinary標準設定）</li>
        </ul>
      </div>
    </section>

    <hr class="divider" />

    <!-- D: 単体テスト仕様書 -->
    <section id="section-d" class="doc-section">
      <div class="section-header">
        <span class="section-label">D</span>
        <div>
          <h2 class="section-title">単体テスト仕様書</h2>
          <p class="section-subtitle">プログラム単位の検証</p>
        </div>
      </div>
      <p class="section-intro">
        個々の関数・部品が「仕様通りに動くか」を確認するテストの計画です。
        問題を早期に発見し、修正コストを最小化します。
      </p>

      <div class="subsection">
        <h3 class="subsection-title">テスト対象関数・カバレッジ目標</h3>
        <table class="spec-table">
          <thead>
            <tr><th>関数名</th><th>ファイル</th><th>C0（命令）</th><th>主な検証観点</th></tr>
          </thead>
          <tbody>
            <tr><td>generateRowName</td><td>api/nikki.go</td><td>100%</td><td>日付フォーマット、タイトル有無、無効日付</td></tr>
            <tr><td>formatPoint</td><td>api/nikki.go</td><td>100%</td><td>nil入力、0、正の整数</td></tr>
            <tr><td>parseEntry</td><td>api/nikki.go</td><td>80%以上</td><td>正常行、空行、列数不足</td></tr>
            <tr><td>parseFeedback</td><td>+page.svelte</td><td>90%以上</td><td>正常JSON、空文字、不正JSON</td></tr>
            <tr><td>sendNotification</td><td>api/nikki.go</td><td>80%以上</td><td>環境変数未設定時のスキップ動作</td></tr>
          </tbody>
        </table>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">境界値分析</h3>
        <table class="spec-table">
          <thead>
            <tr><th>テスト対象</th><th>境界値</th><th>期待結果</th></tr>
          </thead>
          <tbody>
            <tr><td>写真枚数</td><td>0枚、5枚、6枚</td><td>0〜5枚: 正常。6枚目: UIで追加ブロック</td></tr>
            <tr><td>ファイルサイズ</td><td>4MB、4MB+1byte</td><td>4MB: アップロード可。超過: エラー表示</td></tr>
            <tr><td>RP入力値</td><td>0、負数、整数最大値</td><td>0以上の整数のみ正常</td></tr>
            <tr><td>日記本文</td><td>空白のみ、1文字、10000文字</td><td>空白: バリデーションエラー。それ以外: 正常</td></tr>
            <tr><td>絵文字・特殊文字</td><td>🎮😀 等サロゲートペア</td><td>文字化けなく保存・表示されること</td></tr>
          </tbody>
        </table>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">異常系・例外処理テスト</h3>
        <ul class="spec-list">
          <li>Google Sheets API接続不可: 500エラーが返却されトースト通知が表示されること</li>
          <li>Cloudinaryアップロード失敗: エラーメッセージが表示され日記保存はキャンセルされること</li>
          <li>IndexedDB書き込み失敗: コンソールエラーのみ（UIには影響しない）</li>
          <li>SMTP接続失敗: ログ出力のみ。APIレスポンスには影響しないこと</li>
          <li>不正なJSONリクエスト: 400 Bad Requestが返却されること</li>
        </ul>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">静的解析・品質チェック</h3>
        <ul class="spec-list">
          <li>Go: <code>go vet ./...</code> でコード検査</li>
          <li>Go: <code>staticcheck</code> による静的解析</li>
          <li>Svelte/JS: <code>npm run check</code>（svelte-check）による型チェック</li>
        </ul>
      </div>
    </section>

    <hr class="divider" />

    <!-- E: 結合テスト仕様書 -->
    <section id="section-e" class="doc-section">
      <div class="section-header">
        <span class="section-label">E</span>
        <div>
          <h2 class="section-title">結合テスト仕様書</h2>
          <p class="section-subtitle">インターフェースの検証</p>
        </div>
      </div>
      <p class="section-intro">
        複数の機能を組み合わせたときに「継ぎ目」で問題が起きないかを確認するテストです。
        単体では動いても連携すると壊れる、という問題をここで潰します。
      </p>

      <div class="subsection">
        <h3 class="subsection-title">データ受け渡し検証</h3>
        <table class="spec-table">
          <thead>
            <tr><th>テストケース</th><th>検証内容</th></tr>
          </thead>
          <tbody>
            <tr><td>日記保存 → 一覧表示</td><td>保存後に一覧リロードして、保存した内容が正しく表示されること</td></tr>
            <tr><td>編集 → 保存 → 表示</td><td>編集内容がシートに反映されており、UI上も更新されること</td></tr>
            <tr><td>フィードバック保存 → 表示</td><td>フィードバックJSON解析後、カテゴリ別に正しく表示されること</td></tr>
            <tr><td>エクスポート（PDF）</td><td>写真を含むエントリがPDFプレビューに正しく反映されること</td></tr>
          </tbody>
        </table>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">イベントハンドリング</h3>
        <ul class="spec-list">
          <li>保存ボタン高速連打: 二重送信されないこと（ボタン無効化で制御）</li>
          <li>ブラウザバック: 下書きが残っていれば復元バナーが表示されること</li>
          <li>フィードバックモーダルを開いたまま別タブで操作: 状態が混在しないこと</li>
        </ul>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">エラーメッセージ表示</h3>
        <ul class="spec-list">
          <li>必須項目未入力で保存: バリデーションエラートーストが表示され入力値は消えないこと</li>
          <li>APIエラー後の再送信: 再度送信操作が行えること</li>
        </ul>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">外部システム連携（障害時）</h3>
        <ul class="spec-list">
          <li>Google Sheets APIがダウン: 「保存に失敗しました」トーストが表示され、下書きは消えないこと</li>
          <li>Cloudinaryが遅延: タイムアウト後にエラーメッセージが表示されること</li>
          <li>SMTP送信失敗: APIレスポンスは200を返し、ユーザー体験に影響しないこと</li>
        </ul>
      </div>
    </section>

    <hr class="divider" />

    <!-- F: 総合テスト仕様書 -->
    <section id="section-f" class="doc-section">
      <div class="section-header">
        <span class="section-label">F</span>
        <div>
          <h2 class="section-title">総合テスト仕様書</h2>
          <p class="section-subtitle">エンドツーエンドの検証</p>
        </div>
      </div>
      <p class="section-intro">
        本番環境に近い状態で「実際の使い方」を通してシステム全体を検証します。
        性能・運用・セキュリティの観点を網羅します。
      </p>

      <div class="subsection">
        <h3 class="subsection-title">業務シナリオテスト</h3>
        <table class="spec-table">
          <thead>
            <tr><th>シナリオ</th><th>手順</th><th>合否基準</th></tr>
          </thead>
          <tbody>
            <tr><td>正常系: 日記登録〜エクスポート</td><td>日記入力 → 写真追加 → 保存 → 履歴確認 → Excelエクスポート</td><td>全手順がエラーなく完了すること</td></tr>
            <tr><td>フィードバック付与</td><td>履歴のエントリを選択 → PIN入力 → 全カテゴリ記入 → 保存 → 表示確認</td><td>フィードバック内容が正しく保存・表示されること</td></tr>
            <tr><td>一括削除</td><td>複数エントリをチェック → 削除 → 一覧から消えることを確認</td><td>選択した件数分だけ削除されること</td></tr>
            <tr><td>下書き復元</td><td>日記を途中まで入力 → ページリロード → 復元バナーを確認 → 復元</td><td>入力内容が復元されること</td></tr>
          </tbody>
        </table>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">性能・負荷テスト</h3>
        <ul class="spec-list">
          <li>目標スループット: 同時5ユーザーが同時保存操作を行っても応答時間3秒以内</li>
          <li>限界負荷: Vercel Serverless関数のタイムアウト（10秒）に達する条件の確認</li>
          <li>エントリ件数: 1000件登録時の一覧表示速度を計測（目標: 5秒以内）</li>
        </ul>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">脆弱性診断（DAST）</h3>
        <ul class="spec-list">
          <li>XSSチェック: 日記本文・タイトルへのスクリプトタグ入力 → エスケープされていること</li>
          <li>直接オブジェクト参照: 他ユーザーの行番号を指定した削除APIが拒否されること（現バージョンはシングルユーザー前提）</li>
          <li>環境変数漏洩: Goサーバーログに秘密鍵・APIキーが出力されないこと</li>
        </ul>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">切り戻し試験</h3>
        <ul class="spec-list">
          <li>Vercelの以前のデプロイメントへのロールバック手順が機能することを確認</li>
          <li>Googleスプレッドシートのバージョン履歴からデータを復元できることを確認</li>
        </ul>
      </div>
    </section>

    <hr class="divider" />

    <!-- G: トレーサビリティマトリクス -->
    <section id="section-g" class="doc-section">
      <div class="section-header">
        <span class="section-label">G</span>
        <div>
          <h2 class="section-title">トレーサビリティマトリクス</h2>
          <p class="section-subtitle">管理の必須項目</p>
        </div>
      </div>
      <p class="section-intro">
        要件・設計・テストの「つながり」を可視化する表です。「この要件はどのテストで検証しているか？」
        が一目でわかるようにします。見直し・変更時の影響範囲の特定に役立ちます。
      </p>

      <div class="subsection">
        <h3 class="subsection-title">要件紐付け表</h3>
        <table class="spec-table">
          <thead>
            <tr><th>要件ID</th><th>機能名</th><th>設計書参照</th><th>単体テスト</th><th>結合テスト</th><th>総合テスト</th></tr>
          </thead>
          <tbody>
            <tr><td>FR-01</td><td>日記新規作成</td><td>B-API, C-CRUD</td><td>generateRowName, buildRow</td><td>保存→一覧</td><td>正常系シナリオ</td></tr>
            <tr><td>FR-02</td><td>日記一覧表示</td><td>B-画面遷移</td><td>parseEntry</td><td>保存→一覧</td><td>1000件表示</td></tr>
            <tr><td>FR-03</td><td>検索・絞り込み</td><td>B-画面遷移</td><td>フィルタ関数</td><td>—</td><td>業務シナリオ</td></tr>
            <tr><td>FR-04</td><td>編集・削除</td><td>C-CRUD</td><td>—</td><td>編集→保存→表示</td><td>一括削除</td></tr>
            <tr><td>FR-05</td><td>フィードバック</td><td>C-JSON構造</td><td>parseFeedback</td><td>FB保存→表示</td><td>フィードバック付与</td></tr>
            <tr><td>FR-06</td><td>写真アップロード</td><td>B-API</td><td>ファイルサイズ境界値</td><td>写真→PDF</td><td>正常系シナリオ</td></tr>
            <tr><td>FR-07</td><td>エクスポート</td><td>C-モジュール</td><td>—</td><td>エクスポート</td><td>正常系シナリオ</td></tr>
            <tr><td>FR-08</td><td>下書き自動保存</td><td>C-キャッシュ</td><td>IndexedDB</td><td>ブラウザバック</td><td>下書き復元</td></tr>
            <tr><td>FR-09</td><td>メール通知</td><td>C-シーケンス</td><td>sendNotification</td><td>SMTP障害時</td><td>—</td></tr>
          </tbody>
        </table>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">変更履歴管理</h3>
        <table class="spec-table">
          <thead>
            <tr><th>日付</th><th>版</th><th>変更内容</th><th>意思決定の背景（ADR）</th></tr>
          </thead>
          <tbody>
            <tr><td>2026-02</td><td>1.0</td><td>初版作成</td><td>個人利用のシンプルな構成でMVPとしてリリース</td></tr>
            <tr><td>2026-02</td><td>1.0</td><td>フィードバック欄スタイル修正</td><td>青一色で視認性が低かったため、日記欄と統一したグレー系に変更</td></tr>
            <tr><td>2026-02</td><td>1.0</td><td>メール通知機能追加</td><td>保存イベントを外部から把握するため、SMTP通知を実装（環境変数で制御）</td></tr>
          </tbody>
        </table>
      </div>

      <div class="subsection">
        <h3 class="subsection-title">回帰テスト区分</h3>
        <table class="spec-table">
          <thead>
            <tr><th>テスト項目</th><th>自動化推奨</th><th>理由</th></tr>
          </thead>
          <tbody>
            <tr><td>generateRowName / formatPoint</td><td>○</td><td>純粋関数。テーブル駆動テストが容易</td></tr>
            <tr><td>parseEntry</td><td>○</td><td>シートデータ形式変更時の回帰リスクあり</td></tr>
            <tr><td>日記保存〜一覧表示</td><td>△（E2Eツール推奨）</td><td>Playwright/Cypressで自動化可能</td></tr>
            <tr><td>エクスポートPDF</td><td>×（目視確認）</td><td>PDF描画の正確性は自動検証が困難</td></tr>
          </tbody>
        </table>
      </div>
    </section>

  </div><!-- /.doc-body -->

  <footer class="doc-footer">
    <div class="doc-footer-inner">
      <p>Nikki システム設計書 v1.0 &copy; 2026 Nikki Project</p>
      <a href="/" class="footer-back-link">← アプリに戻る</a>
    </div>
  </footer>

</div><!-- /.page-wrapper -->

<style>
  :global(body) {
    font-family: 'Hiragino Sans', 'Hiragino Kaku Gothic ProN', 'Noto Sans JP', 'Yu Gothic', sans-serif;
    background: #f5f5f5;
    color: #1a1a1a;
    margin: 0;
    padding: 0;
    box-sizing: border-box;
  }

  :global(*, *::before, *::after) {
    box-sizing: border-box;
  }

  .page-wrapper {
    min-height: 100vh;
    background: #f5f5f5;
  }

  /* ── ドキュメントヘッダー ── */
  .doc-header {
    background: #1e293b;
    color: #f8fafc;
    padding: 2rem 1rem;
    border-bottom: 4px solid #334155;
  }

  .doc-header-inner {
    max-width: 900px;
    margin: 0 auto;
    position: relative;
  }

  .doc-meta {
    font-size: 0.75rem;
    color: #94a3b8;
    margin-bottom: 0.75rem;
    letter-spacing: 0.04em;
  }

  .doc-meta-item {
    display: inline;
  }

  .doc-meta-sep {
    margin: 0 0.5rem;
    color: #475569;
  }

  .doc-title {
    font-size: 1.75rem;
    font-weight: 800;
    color: #f1f5f9;
    margin: 0 0 0.4rem;
    letter-spacing: -0.02em;
  }

  .doc-subtitle {
    font-size: 0.9rem;
    color: #94a3b8;
    margin: 0 0 1.25rem;
  }

  .back-link {
    display: inline-block;
    font-size: 0.8rem;
    color: #7dd3fc;
    text-decoration: none;
    border: 1px solid #334155;
    padding: 0.3rem 0.75rem;
    border-radius: 4px;
    transition: background 0.15s;
  }

  .back-link:hover {
    background: #334155;
  }

  /* ── ドキュメント本文 ── */
  .doc-body {
    max-width: 900px;
    margin: 0 auto;
    padding: 2rem 1.5rem;
  }

  /* ── 目次 ── */
  .toc-section {
    background: #fff;
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    padding: 1.5rem 2rem;
    margin-bottom: 2rem;
  }

  .toc-heading {
    font-size: 0.8rem;
    font-weight: 700;
    color: #64748b;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    margin-bottom: 1rem;
  }

  .toc-list {
    padding-left: 1.5rem;
    margin: 0;
  }

  .toc-list li {
    margin-bottom: 0.45rem;
  }

  .toc-link {
    font-size: 0.88rem;
    color: #1e293b;
    text-decoration: none;
    font-weight: 600;
  }

  .toc-link:hover {
    text-decoration: underline;
    color: #0284c7;
  }

  .toc-subtitle {
    font-weight: 400;
    color: #64748b;
  }

  /* ── 区切り線 ── */
  .divider {
    border: none;
    border-top: 1px solid #e2e8f0;
    margin: 2.5rem 0;
  }

  /* ── セクション ── */
  .doc-section {
    margin-bottom: 1rem;
  }

  .section-header {
    display: flex;
    align-items: flex-start;
    gap: 1rem;
    margin-bottom: 1rem;
    padding-bottom: 0.75rem;
    border-bottom: 2px solid #e2e8f0;
  }

  .section-label {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 2rem;
    height: 2rem;
    background: #1e293b;
    color: #f1f5f9;
    font-size: 0.95rem;
    font-weight: 800;
    border-radius: 4px;
    flex-shrink: 0;
    margin-top: 0.15rem;
    letter-spacing: 0;
  }

  .section-title {
    font-size: 1.25rem;
    font-weight: 800;
    color: #1e293b;
    margin: 0 0 0.15rem;
  }

  .section-subtitle {
    font-size: 0.8rem;
    color: #64748b;
    margin: 0;
  }

  .section-intro {
    font-size: 0.9rem;
    color: #475569;
    line-height: 1.7;
    margin-bottom: 1.5rem;
    background: #f8fafc;
    border-left: 3px solid #cbd5e1;
    padding: 0.75rem 1rem;
    border-radius: 0 4px 4px 0;
  }

  /* ── サブセクション ── */
  .subsection {
    margin-bottom: 1.75rem;
  }

  .subsection-title {
    font-size: 0.9rem;
    font-weight: 700;
    color: #334155;
    margin-bottom: 0.75rem;
    padding-bottom: 0.35rem;
    border-bottom: 1px solid #e2e8f0;
  }

  .subsection-body {
    font-size: 0.88rem;
    color: #374151;
    line-height: 1.75;
    margin: 0;
  }

  /* ── テーブル ── */
  .spec-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.82rem;
    margin-bottom: 0.5rem;
  }

  .spec-table th {
    background: #f1f5f9;
    color: #334155;
    font-weight: 700;
    padding: 0.5rem 0.75rem;
    text-align: left;
    border: 1px solid #e2e8f0;
    font-size: 0.78rem;
    white-space: nowrap;
  }

  .spec-table td {
    padding: 0.5rem 0.75rem;
    border: 1px solid #e2e8f0;
    color: #374151;
    vertical-align: top;
    line-height: 1.55;
  }

  .spec-table tr:nth-child(even) td {
    background: #fafafa;
  }

  /* ── リスト ── */
  .spec-list {
    font-size: 0.88rem;
    color: #374151;
    line-height: 1.8;
    padding-left: 1.5rem;
    margin: 0;
  }

  .spec-list li {
    margin-bottom: 0.25rem;
  }

  /* ── 注記 ── */
  .note {
    font-size: 0.78rem;
    color: #64748b;
    margin-top: 0.5rem;
    font-style: italic;
  }

  /* ── AAscii図 ── */
  .ascii-diagram {
    font-family: 'Courier New', Courier, monospace;
    font-size: 0.78rem;
    background: #1e293b;
    color: #94a3b8;
    padding: 1rem 1.25rem;
    border-radius: 6px;
    overflow-x: auto;
    line-height: 1.6;
    margin: 0;
  }

  /* ── コードブロック ── */
  .code-block {
    font-family: 'Courier New', Courier, monospace;
    font-size: 0.8rem;
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    color: #1e293b;
    padding: 1rem 1.25rem;
    border-radius: 6px;
    overflow-x: auto;
    line-height: 1.6;
    margin: 0;
  }

  /* ── フッター ── */
  .doc-footer {
    background: #1e293b;
    color: #94a3b8;
    padding: 1.5rem 1rem;
    margin-top: 3rem;
    border-top: 1px solid #334155;
  }

  .doc-footer-inner {
    max-width: 900px;
    margin: 0 auto;
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: 0.75rem;
  }

  .doc-footer-inner p {
    font-size: 0.78rem;
    margin: 0;
  }

  .footer-back-link {
    font-size: 0.78rem;
    color: #7dd3fc;
    text-decoration: none;
  }

  .footer-back-link:hover {
    text-decoration: underline;
  }

  /* ── レスポンシブ ── */
  @media (max-width: 600px) {
    .doc-title { font-size: 1.3rem; }
    .doc-body { padding: 1.25rem 1rem; }
    .spec-table { font-size: 0.75rem; }
    .spec-table th, .spec-table td { padding: 0.4rem 0.5rem; }
    .doc-footer-inner { flex-direction: column; align-items: flex-start; }
  }
</style>
