<script lang="ts">
	import { onMount, onDestroy, tick } from 'svelte';
	import { fade, fly, slide } from 'svelte/transition';
	import { quintOut } from 'svelte/easing';

	const RANKS = ['', 'ブロンズ', 'シルバー', 'ゴールド', 'プラチナ', 'ダイヤ', 'マスター', 'プレデター'];
	const API_BASE = import.meta.env.VITE_API_BASE_URL || '/api';

	const FEEDBACK_CATEGORIES = [
		{ key: 'aim', label: 'エイム・撃ち合い', desc: '命中率、リコイルコントロール、被弾の多さ' },
		{ key: 'positioning', label: '立ち回り・位置取り', desc: '有利ポジションの確保、遮蔽物の利用、射線管理' },
		{ key: 'judgment', label: '状況判断', desc: '交戦・撤退のタイミング、漁夫の警戒、リング移動のルート' },
		{ key: 'teamwork', label: '連携・カバー', desc: '味方との距離感、フォーカス（狙い）合わせ、情報共有・報告' },
		{ key: 'ability', label: 'アビリティ・アルティメット', desc: '使用タイミング、スキルの無駄撃ち' },
		{ key: 'movement', label: 'キャラクターコントロール', desc: '被弾を抑える動き、移動・展開のスピード' },
		{ key: 'resources', label: 'リソース管理', desc: '回復アイテム・弾薬のバランス、漁るスピード' },
		{ key: 'mental', label: 'メンタル', desc: '焦り、判断の迷い、集中力の低下' }
	] as const;

	const SIGNER_OPTIONS = ['もぐ太'] as const;

	// ── フォーム ──
	let date = '';
	let title = '';
	let nikki = '';
	let rank1 = '';
	let point1: number | null = null;
	let rank2 = '';
	let point2: number | null = null;

	// ── 保存状態 ──
	let saving = false;
	let lastSaveTime = 0;
	let message = '';
	let messageType: 'success' | 'error' | '' = '';
	let messageCountdown = 0;
	let messageTimer: ReturnType<typeof setInterval> | null = null;

	// ── タブ ──
	let activeTab: 'write' | 'history' = 'write';

	// ── 下書き (IndexedDB) ──
	let hasDraft = false;
	let showDraftBanner = false;
	let draftDebounce: ReturnType<typeof setTimeout> | null = null;
	const IDB_NAME = 'apex-diary';
	const IDB_STORE = 'draft';

	// ── 履歴 ──
	interface Entry {
		row_index: number;
		row_name: string;
		date: string;
		name: string;
		nikki: string;
		rank1: string;
		point1: string;
		rank2: string;
		point2: string;
		timestamp: string;
		photos: string[];
		feedback: string;
	}

	let entries: Entry[] = [];
	let loadingHistory = false;
	let historyError = '';
	let searchName = '';
	let searchDate = '';
	let searchRank = '';
	let sortOrder: 'newest' | 'oldest' = 'newest';

	// ── 選択 ──
	let selectedIndices = new Set<number>();

	// ── 編集モーダル ──
	let editingEntry: Entry | null = null;
	let editDate = '';
	let editTitle = '';
	let editNikki = '';
	let editRank1 = '';
	let editPoint1: number | null = null;
	let editRank2 = '';
	let editPoint2: number | null = null;
	let editPhotoIds: string[] = [];
	let editNewPhotoFiles: File[] = [];
	let editSaving = false;

	// ── 写真 ──
	let photoFiles: File[] = [];
	let uploadingPhotos = false;

	// ── エクスポートメニュー ──
	let showExportMenu = false;

	// ── フィードバックモーダル ──
	let feedbackEntry: Entry | null = null;
	let feedbackPwInput = '';
	let feedbackData: Record<string, string> = {};
	let feedbackSaving = false;

	// ── 出力プレビュー ──
	let previewType: 'pdf' | 'excel' | 'txt' | 'md' | 'photos' | null = null;
	let previewList: Entry[] = [];

	// ── リアクティブ ──
	$: filteredEntries = entries
		.filter((e) => {
			if (searchName && !e.name.toLowerCase().includes(searchName.toLowerCase())) return false;
			if (searchDate && e.date !== searchDate) return false;
			if (searchRank && e.rank1 !== searchRank && e.rank2 !== searchRank) return false;
			return true;
		})
		.sort((a, b) => {
			const cmp = a.date.localeCompare(b.date) || a.timestamp.localeCompare(b.timestamp);
			return sortOrder === 'newest' ? -cmp : cmp;
		});

	$: selectedCount = selectedIndices.size;
	$: allSelected =
		filteredEntries.length > 0 && filteredEntries.every((e) => selectedIndices.has(e.row_index));
	$: feedbackUnlocked = feedbackPwInput === '8569';

	// ── 初期化 ──
	onMount(async () => {
		const today = new Date();
		date = today.toISOString().split('T')[0];
		const draft = await loadDraftFromIDB();
		if (draft && (draft.nikki || draft.title)) {
			hasDraft = true;
			showDraftBanner = true;
		}
	});

	onDestroy(() => {
		if (messageTimer) clearInterval(messageTimer);
		if (draftDebounce) clearTimeout(draftDebounce);
	});

	// ── メッセージ ──
	function showMsg(msg: string, type: 'success' | 'error') {
		if (messageTimer) clearInterval(messageTimer);
		message = msg;
		messageType = type;
		if (type === 'success') {
			messageCountdown = 5;
			messageTimer = setInterval(() => {
				messageCountdown--;
				if (messageCountdown <= 0) {
					clearInterval(messageTimer!);
					messageTimer = null;
					message = '';
					messageType = '';
					messageCountdown = 0;
				}
			}, 1000);
		}
	}

	// ── IndexedDB 下書き ──
	function openIDB(): Promise<IDBDatabase> {
		return new Promise((resolve, reject) => {
			const req = indexedDB.open(IDB_NAME, 1);
			req.onupgradeneeded = () => {
				req.result.createObjectStore(IDB_STORE, { keyPath: 'id' });
			};
			req.onsuccess = () => resolve(req.result);
			req.onerror = () => reject(req.error);
		});
	}

	async function saveDraftToIDB(data: object) {
		try {
			const db = await openIDB();
			const tx = db.transaction(IDB_STORE, 'readwrite');
			tx.objectStore(IDB_STORE).put({ id: 'current', ...data });
			await new Promise<void>((res, rej) => {
				tx.oncomplete = () => res();
				tx.onerror = () => rej(tx.error);
			});
			db.close();
			hasDraft = true;
		} catch (e) {
			console.warn('Draft save failed:', e);
		}
	}

	async function loadDraftFromIDB(): Promise<Record<string, unknown> | null> {
		try {
			const db = await openIDB();
			const tx = db.transaction(IDB_STORE, 'readonly');
			const req = tx.objectStore(IDB_STORE).get('current');
			const result = await new Promise<Record<string, unknown> | null>((res, rej) => {
				req.onsuccess = () => res((req.result as Record<string, unknown>) ?? null);
				req.onerror = () => rej(req.error);
			});
			db.close();
			return result;
		} catch (e) {
			console.warn('Draft load failed:', e);
			return null;
		}
	}

	async function clearDraftFromIDB() {
		try {
			const db = await openIDB();
			const tx = db.transaction(IDB_STORE, 'readwrite');
			tx.objectStore(IDB_STORE).delete('current');
			await new Promise<void>((res, rej) => {
				tx.oncomplete = () => res();
				tx.onerror = () => rej(tx.error);
			});
			db.close();
			hasDraft = false;
			showDraftBanner = false;
		} catch (e) {
			console.warn('Draft clear failed:', e);
		}
	}

	async function applyDraft() {
		const draft = await loadDraftFromIDB();
		if (draft) {
			if (draft.date) date = draft.date as string;
			title = (draft.title as string) || '';
			nikki = (draft.nikki as string) || '';
			rank1 = (draft.rank1 as string) || '';
			point1 = draft.point1 != null ? (draft.point1 as number) : null;
			rank2 = (draft.rank2 as string) || '';
			point2 = draft.point2 != null ? (draft.point2 as number) : null;
		}
		showDraftBanner = false;
	}

	function scheduleDraftSave() {
		if (draftDebounce) clearTimeout(draftDebounce);
		draftDebounce = setTimeout(() => {
			if (nikki || title) {
				saveDraftToIDB({ date, title, nikki, rank1, point1, rank2, point2 });
			}
		}, 1500);
	}

	// ── 保存 ──
	async function save() {
		if (!nikki.trim()) {
			showMsg('日記の内容を入力してください', 'error');
			return;
		}
		const now = Date.now();
		if (saving || now - lastSaveTime < 500) return;

		saving = true;
		try {
			let uploadedPhotoIds: string[] = [];
			if (photoFiles.length > 0) {
				uploadingPhotos = true;
				uploadedPhotoIds = await uploadPhotos(photoFiles);
				uploadingPhotos = false;
			}

			const payload: Record<string, unknown> = {
				date,
				name: title,
				nikki: nikki.trim(),
				rank1,
				rank2,
				photos: uploadedPhotoIds
			};
			if (point1 !== null && String(point1) !== '') payload.point1 = Number(point1);
			if (point2 !== null && String(point2) !== '') payload.point2 = Number(point2);

			const res = await fetch(`${API_BASE}/nikki`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(payload)
			});
			const data = await safeJson(res);
			if (data.success) {
				lastSaveTime = Date.now();
				showMsg('保存しました', 'success');
				await clearDraftFromIDB();
				title = '';
				nikki = '';
				rank1 = '';
				point1 = null;
				rank2 = '';
				point2 = null;
				photoFiles = [];
				entries = [];
			} else {
				showMsg(String(data.message || '保存失敗'), 'error');
			}
		} catch (e) {
			uploadingPhotos = false;
			showMsg(`通信エラー: ${e instanceof Error ? e.message : String(e)}`, 'error');
		} finally {
			saving = false;
		}
	}

	// ── 履歴読み込み ──
	async function loadHistory() {
		loadingHistory = true;
		historyError = '';
		try {
			const res = await fetch(`${API_BASE}/nikki`);
			if (!res.ok) {
				const text = await res.text().catch(() => '');
				let msg = `HTTP ${res.status}`;
				try { msg = (JSON.parse(text) as { error?: string }).error || msg; } catch { /* ignore */ }
				throw new Error(msg);
			}
			const text = await res.text();
			entries = JSON.parse(text) as Entry[];
		} catch (e) {
			historyError = e instanceof Error ? e.message : String(e);
		} finally {
			loadingHistory = false;
		}
	}

	function switchTab(tab: 'write' | 'history') {
		activeTab = tab;
		if (tab === 'history' && entries.length === 0 && !loadingHistory) {
			loadHistory();
		}
	}

	// ── 削除 ──
	async function deleteEntries(indices: number[]) {
		if (!confirm(`${indices.length}件を削除しますか？この操作は取り消せません。`)) return;
		// 削除対象エントリーの写真URLを事前に収集
		const photoUrls: string[] = entries
			.filter((e) => indices.includes(e.row_index))
			.flatMap((e) => e.photos || []);
		try {
			const res = await fetch(`${API_BASE}/nikki`, {
				method: 'DELETE',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ row_indices: indices })
			});
			const data = await safeJson(res);
			if (data.success) {
				// Cloudinaryからも写真を削除（バックグラウンド実行・エラー無視）
				if (photoUrls.length > 0) {
					fetch(`${API_BASE}/photo`, {
						method: 'DELETE',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({ urls: photoUrls })
					}).catch(() => {});
				}
				selectedIndices = new Set();
				await loadHistory();
			} else {
				showMsg(data.message || '削除失敗', 'error');
			}
		} catch (e) {
			showMsg(`通信エラー: ${e instanceof Error ? e.message : String(e)}`, 'error');
		}
	}

	// ── 編集 ──
	function startEdit(entry: Entry) {
		editingEntry = entry;
		editDate = entry.date;
		editTitle = entry.name;
		editNikki = entry.nikki;
		editRank1 = entry.rank1;
		editPoint1 = parsePointStr(entry.point1);
		editRank2 = entry.rank2;
		editPoint2 = parsePointStr(entry.point2);
		editPhotoIds = entry.photos ? [...entry.photos] : [];
		editNewPhotoFiles = [];
	}

	function parsePointStr(s: string): number | null {
		const m = s.match(/^(\d+)RP$/);
		return m ? parseInt(m[1], 10) : null;
	}

	function cancelEdit() {
		editingEntry = null;
	}

	async function saveEdit() {
		if (!editNikki.trim()) {
			showMsg('日記の内容を入力してください', 'error');
			return;
		}
		editSaving = true;
		try {
			let newPhotoIds: string[] = [];
			if (editNewPhotoFiles.length > 0) {
				newPhotoIds = await uploadPhotos(editNewPhotoFiles);
			}
			const allPhotoIds = [...editPhotoIds, ...newPhotoIds];

			const payload: Record<string, unknown> = {
				row_index: editingEntry!.row_index,
				date: editDate,
				name: editTitle,
				nikki: editNikki.trim(),
				rank1: editRank1,
				rank2: editRank2,
				photos: allPhotoIds
			};
			if (editPoint1 !== null && String(editPoint1) !== '') payload.point1 = Number(editPoint1);
			if (editPoint2 !== null && String(editPoint2) !== '') payload.point2 = Number(editPoint2);

			const res = await fetch(`${API_BASE}/nikki`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(payload)
			});
			const data = await safeJson(res);
			if (data.success) {
				editingEntry = null;
				await loadHistory();
			} else {
				showMsg(data.message || '更新失敗', 'error');
			}
		} catch (e) {
			showMsg(`通信エラー: ${e instanceof Error ? e.message : String(e)}`, 'error');
		} finally {
			editSaving = false;
		}
	}

	// ── 選択 ──
	function toggleSelect(index: number) {
		const next = new Set(selectedIndices);
		next.has(index) ? next.delete(index) : next.add(index);
		selectedIndices = next;
	}

	function toggleSelectAll() {
		if (allSelected) {
			selectedIndices = new Set();
		} else {
			selectedIndices = new Set(filteredEntries.map((e) => e.row_index));
		}
	}

	// ── エクスポート ──
	function fmtDate(d: string): string {
		if (!d || d.length !== 10) return d;
		const [y, m, day] = d.split('-');
		return `${y}年${m}月${day}日`;
	}

	// ランクに対応するバッジカラーを返す
	function rankStyle(rank: string): string {
		const map: Record<string, string> = {
			ブロンズ: 'background:#fef3c7;color:#92400e;border-color:#f59e0b',
			シルバー: 'background:#f1f5f9;color:#475569;border-color:#94a3b8',
			ゴールド: 'background:#fefce8;color:#a16207;border-color:#eab308',
			プラチナ: 'background:#f0fdf4;color:#166534;border-color:#86efac',
			ダイヤ: 'background:#eff6ff;color:#1d4ed8;border-color:#60a5fa',
			マスター: 'background:#faf5ff;color:#7e22ce;border-color:#c084fc',
			プレデター: 'background:#fff1f2;color:#be123c;border-color:#fb7185'
		};
		return map[rank] ?? '';
	}

	function getEntriesToExport(): Entry[] {
		if (selectedCount > 0) return filteredEntries.filter((e) => selectedIndices.has(e.row_index));
		return filteredEntries;
	}

	function downloadBlob(content: string | Uint8Array, filename: string, type: string) {
		const blob =
			typeof content === 'string'
				? new Blob(['\uFEFF' + content], { type })
				: new Blob([content], { type });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = filename;
		a.click();
		URL.revokeObjectURL(url);
	}

	function todayStr() {
		return new Date().toISOString().split('T')[0];
	}

	function blobToDataUrl(blob: Blob): Promise<string> {
		return new Promise((resolve, reject) => {
			const reader = new FileReader();
			reader.onload = () => resolve(reader.result as string);
			reader.onerror = reject;
			reader.readAsDataURL(blob);
		});
	}

	// ── コンテンツ生成ヘルパー（プレビューと実出力で共有） ──

	function buildTxtContent(list: Entry[]): string {
		return list
			.map((e) => {
				const fbData = parseFeedback(e.feedback);
				const hasFb = hasFeedback(fbData);
				const fbSection: (string | null)[] = hasFb
					? [
						'',
						'【フィードバック】',
						...(fbData['free'] ? [fbData['free'], ''] : []),
						...FEEDBACK_CATEGORIES.filter((c) => fbData[c.key]).flatMap((c) => [
							`[${c.label}]`,
							fbData[c.key]
						]),
						...(fbData['signer'] ? ['', `— ${fbData['signer']}`] : [])
					  ]
					: [];
				const lines = [
					'='.repeat(40),
					`日付: ${fmtDate(e.date)}`,
					e.name ? `タイトル: ${e.name}` : null,
					'',
					e.nikki,
					'',
					e.rank1 ? `ランク1: ${e.rank1}${e.point1 ? ' ' + e.point1 : ''}` : null,
					e.rank2 ? `ランク2: ${e.rank2}${e.point2 ? ' ' + e.point2 : ''}` : null,
					e.timestamp ? `記録日時: ${e.timestamp}` : null,
					...fbSection
				].filter((l) => l !== null);
				return lines.join('\n');
			})
			.join('\n\n');
	}

	function buildMdContent(list: Entry[]): string {
		const header = `# Apex成長日記\n\n出力日: ${fmtDate(todayStr())}\n\n---\n\n`;
		const body = list
			.map((e) => {
				const parts: string[] = [
					`## ${fmtDate(e.date)}${e.name ? ' — ' + e.name : ''}`,
					'',
					e.nikki,
					''
				];
				if (e.rank1 || e.rank2) {
					parts.push('**ランク情報**');
					if (e.rank1) parts.push(`- ランク1: ${e.rank1}${e.point1 ? ' ' + e.point1 : ''}`);
					if (e.rank2) parts.push(`- ランク2: ${e.rank2}${e.point2 ? ' ' + e.point2 : ''}`);
					parts.push('');
				}
				if (e.timestamp) parts.push(`*記録日時: ${e.timestamp}*`);
				const fbData = parseFeedback(e.feedback);
				if (hasFeedback(fbData)) {
					parts.push('');
					parts.push('**フィードバック**');
					if (fbData['free']) parts.push(`> ${fbData['free']}`);
					FEEDBACK_CATEGORIES.filter((c) => fbData[c.key]).forEach((c) =>
						parts.push(`- **${c.label}**: ${fbData[c.key]}`)
					);
					if (fbData['signer']) parts.push(`*— ${fbData['signer']}*`);
				}
				parts.push('---');
				return parts.join('\n');
			})
			.join('\n\n');
		return header + body;
	}

	function photoFilename(e: Entry, idx: number): string {
		const dateStr = e.date.replace(/-/g, '');
		const title = e.name ? e.name.replace(/[\\/:*?"<>|]/g, '_') : '';
		const prefix = title ? `${dateStr}_${title}` : dateStr;
		const url = e.photos![idx];
		const ext = url.split('?')[0].split('.').pop() || 'jpg';
		return e.photos!.length === 1 ? `${prefix}.${ext}` : `${prefix}_${idx + 1}.${ext}`;
	}

	function previewFormatName(type: string): string {
		const map: Record<string, string> = { pdf: 'PDF', excel: 'Excel', txt: 'テキスト', md: 'Markdown', photos: '写真ZIP' };
		return map[type] ?? type;
	}

	function previewFormatIcon(type: string): string {
		const map: Record<string, string> = { pdf: '📄', excel: '📊', txt: '📝', md: '📋', photos: '🖼' };
		return map[type] ?? '⬇';
	}

	// ── 写真ZIP共通ヘルパー ──

	async function exportPhotosForList(list: Entry[]) {
		const allPhotos: { url: string; filename: string }[] = [];
		for (const e of list) {
			if (!e.photos || e.photos.length === 0) continue;
			e.photos.forEach((url, i) => {
				allPhotos.push({ url, filename: photoFilename(e, i) });
			});
		}
		if (allPhotos.length === 0) return;
		const { default: JSZip } = await import('jszip');
		const zip = new JSZip();
		await Promise.all(
			allPhotos.map(async ({ url, filename }) => {
				const res = await fetch(url);
				const blob = await res.blob();
				zip.file(filename, blob);
			})
		);
		const zipBlob = await zip.generateAsync({ type: 'blob' });
		const zipUrl = URL.createObjectURL(zipBlob);
		const a = document.createElement('a');
		a.href = zipUrl;
		a.download = `apex-diary-photos-${todayStr()}.zip`;
		a.click();
		URL.revokeObjectURL(zipUrl);
	}

	// ── 実出力関数（list を引数で受け取る） ──

	async function exportTxt(list: Entry[]) {
		downloadBlob(buildTxtContent(list), `apex-diary-${todayStr()}.txt`, 'text/plain;charset=utf-8');
		await exportPhotosForList(list);
	}

	async function exportMd(list: Entry[]) {
		downloadBlob(buildMdContent(list), `apex-diary-${todayStr()}.md`, 'text/markdown;charset=utf-8');
		await exportPhotosForList(list);
	}

	async function exportExcel(list: Entry[]) {
		// ExcelJS を使ってスタイル付きで出力
		const ExcelJSMod = await import('exceljs');
		const ExcelJS = ExcelJSMod.default ?? ExcelJSMod;
		const wb = new ExcelJS.Workbook();
		wb.creator = 'Apex成長日記';

		const ws = wb.addWorksheet('日記');

		// カラム定義
		const COL_DEFS = [
			{ header: '日付', key: 'date' },
			{ header: 'タイトル', key: 'name' },
			{ header: '日記', key: 'nikki' },
			{ header: 'ランク1', key: 'rank1' },
			{ header: 'RP1', key: 'point1' },
			{ header: 'ランク2', key: 'rank2' },
			{ header: 'RP2', key: 'point2' },
			{ header: '記録日時', key: 'timestamp' },
			{ header: 'フィードバック', key: 'feedback' }
		] as const;
		const NCOLS = COL_DEFS.length;

		// CJK文字を2幅としてカラム幅を計算するヘルパー
		const charW = (s: string) =>
			[...s].reduce((w, c) => w + (c.charCodeAt(0) > 0x7f ? 2 : 1), 0);

		// 初期幅（ヘッダー文字数+余白）でカラムを定義
		ws.columns = COL_DEFS.map((c) => ({
			header: c.header,
			key: c.key,
			width: charW(c.header) + 4
		}));

		// ── ヘッダー行スタイル ──
		const HEADER_ROW = ws.getRow(1);
		HEADER_ROW.height = 22;
		HEADER_ROW.eachCell((cell, colIdx) => {
			cell.fill = { type: 'pattern', pattern: 'solid', fgColor: { argb: 'FF0284C7' } };
			cell.font = { bold: true, color: { argb: 'FFFFFFFF' }, size: 10 };
			cell.alignment = { vertical: 'middle', horizontal: 'center', wrapText: false };
			cell.border = {
				top: { style: 'medium' },
				bottom: { style: 'medium' },
				left: colIdx === 1 ? { style: 'medium' } : { style: 'thin' },
				right: colIdx === NCOLS ? { style: 'medium' } : { style: 'thin' }
			};
		});

		// ── データ行を追加 ──
		const dataRows: (string | number | null)[][] = list.map((e) => {
			const fbData = parseFeedback(e.feedback);
			const fbParts: string[] = [];
			if (fbData['free']) fbParts.push(fbData['free']);
			FEEDBACK_CATEGORIES.filter((c) => fbData[c.key]).forEach((c) =>
				fbParts.push(`[${c.label}] ${fbData[c.key]}`)
			);
			if (fbData['signer']) fbParts.push(`— ${fbData['signer']}`);
			return [
				e.date ?? null, e.name ?? null, e.nikki ?? null,
				e.rank1 ?? null, e.point1 ? Number(e.point1) : null,
				e.rank2 ?? null, e.point2 ? Number(e.point2) : null,
				e.timestamp ?? null, fbParts.join('\n') || null
			];
		});

		dataRows.forEach((rowData, ri) => {
			const row = ws.addRow(rowData);
			const isLast = ri === dataRows.length - 1;

			row.eachCell({ includeEmpty: true }, (cell, colIdx) => {
				const isWrap = colIdx === 3 || colIdx === 9; // 日記 / フィードバック
				const isCenter = colIdx >= 4 && colIdx <= 8;
				cell.alignment = {
					wrapText: isWrap,
					vertical: 'top',
					horizontal: isCenter ? 'center' : 'left'
				};
				cell.border = {
					top: { style: 'thin' },
					bottom: isLast ? { style: 'medium' } : { style: 'thin' },
					left: colIdx === 1 ? { style: 'medium' } : { style: 'thin' },
					right: colIdx === NCOLS ? { style: 'medium' } : { style: 'thin' }
				};
			});

			// 日記・フィードバックの行数に合わせて行高さを設定
			const nikkiLines = String(rowData[2] ?? '').split('\n').length;
			const fbLines = String(rowData[8] ?? '').split('\n').length;
			row.height = Math.max(nikkiLines, fbLines, 1) * 15 + 4;
		});

		// ── カラム幅の自動フィット ──
		ws.columns.forEach((col, i) => {
			let maxW = charW(COL_DEFS[i].header) + 4;
			col.eachCell({ includeEmpty: false }, (cell) => {
				const val = String(cell.value ?? '');
				const longestLine = val
					.split('\n')
					.reduce((m, l) => Math.max(m, charW(l)), 0);
				maxW = Math.max(maxW, longestLine + 2);
			});
			col.width = Math.min(maxW, 60);
		});

		// ── バッファ生成 → ダウンロード ──
		const buffer = await wb.xlsx.writeBuffer();
		const blob = new Blob([buffer as ArrayBuffer], {
			type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
		});
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `apex-diary-${todayStr()}.xlsx`;
		a.click();
		URL.revokeObjectURL(url);
		await exportPhotosForList(list);
	}

	async function exportPDF(list: Entry[]) {
		const today = fmtDate(todayStr());
		const escHtml = (s: string) =>
			s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

		// 写真を全て base64 data URL に変換してから HTML に埋め込む
		const imageCache = new Map<string, string>();
		const allPhotoUrls = [...new Set(list.flatMap((e) => e.photos || []))];
		await Promise.all(
			allPhotoUrls.map(async (url) => {
				try {
					const res = await fetch(url);
					const blob = await res.blob();
					imageCache.set(url, await blobToDataUrl(blob));
				} catch {
					imageCache.set(url, url);
				}
			})
		);

		const html = `<!DOCTYPE html>
<html lang="ja">
<head>
<meta charset="utf-8">
<title>Apex成長日記</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: 'Hiragino Sans', 'Noto Sans JP', sans-serif; padding: 32px; color: #111827; }
  h1 { font-size: 22px; font-weight: 700; color: #0284c7; margin-bottom: 4px; }
  .meta { font-size: 12px; color: #6b7280; margin-bottom: 28px; }
  .entry { margin-bottom: 28px; border-bottom: 1px solid #e5e7eb; padding-bottom: 20px; }
  .entry:last-child { border-bottom: none; }
  .entry-head { display: flex; align-items: baseline; gap: 10px; margin-bottom: 10px; }
  .date { font-size: 15px; font-weight: 700; color: #0284c7; }
  .badge { font-size: 11px; background: #e0f2fe; color: #0369a1; padding: 2px 8px; border-radius: 20px; }
  .nikki { white-space: pre-wrap; line-height: 1.75; font-size: 13.5px; color: #1f2937; }
  .ranks { display: flex; gap: 6px; margin-top: 10px; flex-wrap: wrap; }
  .rank-badge { font-size: 12px; background: #f3f4f6; border: 1px solid #e5e7eb; padding: 2px 10px; border-radius: 6px; }
  .ts { font-size: 11px; color: #9ca3af; margin-top: 8px; text-align: right; }
  .photos { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 12px; }
  .photos img { max-width: 220px; max-height: 180px; object-fit: cover; border-radius: 6px; border: 1px solid #e5e7eb; }
  .feedback { margin-top: 12px; background: #eff6ff; border-left: 3px solid #3b82f6; padding: 8px 12px; border-radius: 0 6px 6px 0; }
  .feedback-title { font-size: 11px; font-weight: 700; color: #1d4ed8; margin-bottom: 5px; }
  .feedback-free { font-size: 13px; color: #1f2937; white-space: pre-wrap; margin-bottom: 6px; }
  .feedback-cat { margin-bottom: 4px; font-size: 12px; color: #1f2937; white-space: pre-wrap; }
  .feedback-cat-label { font-weight: 700; color: #1e40af; }
  .feedback-signer { font-size: 12px; color: #4b5563; text-align: right; margin-top: 6px; font-style: italic; }
  @media print { body { padding: 20px; } .photos img { max-width: 180px; max-height: 150px; } }
</style>
</head>
<body>
<h1>Apex成長日記</h1>
<p class="meta">出力日: ${today} ／ ${list.length}件</p>
${list
	.map(
		(e) => `
<div class="entry">
  <div class="entry-head">
    <span class="date">${fmtDate(e.date)}</span>
    ${e.name ? `<span class="badge">${escHtml(e.name)}</span>` : ''}
  </div>
  <p class="nikki">${escHtml(e.nikki)}</p>
  ${
		e.rank1 || e.rank2
			? `<div class="ranks">
    ${e.rank1 ? `<span class="rank-badge">${escHtml(e.rank1)}${e.point1 ? ' ' + escHtml(e.point1) : ''}</span>` : ''}
    ${e.rank2 ? `<span class="rank-badge">${escHtml(e.rank2)}${e.point2 ? ' ' + escHtml(e.point2) : ''}</span>` : ''}
  </div>`
			: ''
	}
  ${
		e.photos && e.photos.length > 0
			? `<div class="photos">${e.photos.map((p) => `<img src="${imageCache.get(p) || p}">`).join('')}</div>`
			: ''
	}
  ${e.timestamp ? `<p class="ts">${escHtml(e.timestamp)}</p>` : ''}
  ${(() => { const fbData = (() => { try { return JSON.parse(e.feedback || '{}'); } catch { return {}; } })();
    const cats = [{ key: 'aim', label: 'エイム・撃ち合い' }, { key: 'positioning', label: '立ち回り・位置取り' },
      { key: 'judgment', label: '状況判断' }, { key: 'teamwork', label: '連携・カバー' },
      { key: 'ability', label: 'アビリティ・アルティメット' }, { key: 'movement', label: 'キャラクターコントロール' },
      { key: 'resources', label: 'リソース管理' }, { key: 'mental', label: 'メンタル' }];
    const filled = cats.filter(c => fbData[c.key]);
    const hasFb = fbData.free || filled.length > 0 || fbData.signer;
    if (!hasFb) return '';
    const freeHtml = fbData.free ? `<p class="feedback-free">${escHtml(fbData.free)}</p>` : '';
    const catsHtml = filled.map(c => `<div class="feedback-cat"><span class="feedback-cat-label">${escHtml(c.label)}: </span>${escHtml(fbData[c.key])}</div>`).join('');
    const signerHtml = fbData.signer ? `<p class="feedback-signer">— ${escHtml(fbData.signer)}</p>` : '';
    return `<div class="feedback"><div class="feedback-title">💬 フィードバック</div>${freeHtml}${catsHtml}${signerHtml}</div>`;
  })()}
</div>`
	)
	.join('')}
</body>
</html>`;
		const win = window.open('', '_blank');
		if (win) {
			win.document.write(html);
			win.document.close();
			setTimeout(() => win.print(), 300);
		}
	}

	// ── プレビューハンドラー ──

	function handleExport(type: 'pdf' | 'excel' | 'txt' | 'md' | 'photos') {
		showExportMenu = false;
		previewList = getEntriesToExport();
		previewType = type;
	}

	function closePreview() {
		previewType = null;
		previewList = [];
	}

	async function confirmExport() {
		const type = previewType;
		const list = [...previewList];
		previewType = null;
		previewList = [];
		if (!type) return;
		if (type === 'pdf') await exportPDF(list);
		else if (type === 'excel') await exportExcel(list);
		else if (type === 'txt') await exportTxt(list);
		else if (type === 'md') await exportMd(list);
		else await exportPhotosForList(list);
	}

	// ── フィードバック ──

	function parseFeedback(raw: string): Record<string, string> {
		if (!raw) return {};
		try {
			return JSON.parse(raw) as Record<string, string>;
		} catch {
			return {};
		}
	}

	function serializeFeedback(data: Record<string, string>): string {
		const filtered = Object.fromEntries(Object.entries(data).filter(([, v]) => v && v.trim()));
		return Object.keys(filtered).length > 0 ? JSON.stringify(filtered) : '';
	}

	function hasFeedback(fbData: Record<string, string>): boolean {
		return !!(
			fbData['free'] ||
			FEEDBACK_CATEGORIES.some((c) => fbData[c.key]) ||
			fbData['signer']
		);
	}

	function openFeedback(entry: Entry) {
		feedbackEntry = entry;
		feedbackPwInput = '';
		feedbackData = parseFeedback(entry.feedback);
		feedbackSaving = false;
		tick().then(() => {
			const el = document.getElementById('fb-pw') as HTMLInputElement | null;
			el?.focus();
		});
	}

	function closeFeedback() {
		feedbackEntry = null;
		feedbackPwInput = '';
	}

	async function saveFeedback() {
		if (!feedbackEntry || !feedbackUnlocked) return;
		feedbackSaving = true;
		const feedbackJson = serializeFeedback(feedbackData);
		try {
			const res = await fetch(`${API_BASE}/nikki`, {
				method: 'PATCH',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ row_index: feedbackEntry.row_index, feedback: feedbackJson })
			});
			const data = await safeJson(res);
			if (data.success) {
				const idx = feedbackEntry.row_index;
				entries = entries.map((e) => (e.row_index === idx ? { ...e, feedback: feedbackJson } : e));
				showMsg('フィードバックを保存しました', 'success');
				closeFeedback();
			} else {
				showMsg(String(data.message || '保存失敗'), 'error');
			}
		} catch (e) {
			showMsg(`通信エラー: ${e instanceof Error ? e.message : String(e)}`, 'error');
		} finally {
			feedbackSaving = false;
		}
	}

	// ── 写真 ──
	function photoUrl(fileId: string): string {
		// 新形式: Cloudinary のフル URL がそのまま格納されている
		if (fileId.startsWith('http')) return fileId;
		// 旧形式（後方互換）: Google Drive ファイルID
		return `https://drive.google.com/uc?export=view&id=${fileId}`;
	}

	async function safeJson(res: Response): Promise<Record<string, unknown>> {
		const text = await res.text();
		try {
			return JSON.parse(text) as Record<string, unknown>;
		} catch {
			// Vercel が 413 等をプレーンテキストで返した場合
			const label = text.slice(0, 120).trim() || `HTTP ${res.status}`;
			throw new Error(label);
		}
	}

	async function uploadPhotos(files: File[]): Promise<string[]> {
		const fileIds: string[] = [];
		for (const file of files) {
			const formData = new FormData();
			formData.append('photo', file);
			const res = await fetch(`${API_BASE}/photo`, { method: 'POST', body: formData });
			const data = await safeJson(res);
			if (data.success && data.file_id) {
				fileIds.push(data.file_id as string);
			} else {
				throw new Error((data.message as string) || '写真のアップロードに失敗しました');
			}
		}
		return fileIds;
	}

	function removeNewPhoto(index: number) {
		photoFiles = photoFiles.filter((_, i) => i !== index);
	}

	function removeEditExistingPhoto(index: number) {
		editPhotoIds = editPhotoIds.filter((_, i) => i !== index);
	}

	function removeEditNewPhoto(index: number) {
		editNewPhotoFiles = editNewPhotoFiles.filter((_, i) => i !== index);
	}

	const MAX_PHOTO_BYTES = 4 * 1024 * 1024; // 4MB (Vercel limit: 4.5MB)

	function validatePhotoFiles(files: File[]): File[] {
		const oversized = files.filter((f) => f.size > MAX_PHOTO_BYTES);
		if (oversized.length > 0) {
			showMsg(
				`ファイルサイズは4MB以下にしてください（超過: ${oversized.map((f) => f.name).join(', ')}）`,
				'error'
			);
			return files.filter((f) => f.size <= MAX_PHOTO_BYTES);
		}
		return files;
	}

	function handlePhotoInput(e: Event) {
		const input = e.target as HTMLInputElement;
		const files = validatePhotoFiles(Array.from(input.files ?? []));
		photoFiles = [...photoFiles, ...files].slice(0, 5);
		input.value = '';
	}

	function handleEditPhotoInput(e: Event) {
		const input = e.target as HTMLInputElement;
		const files = validatePhotoFiles(Array.from(input.files ?? []));
		const remaining = 5 - editPhotoIds.length - editNewPhotoFiles.length;
		editNewPhotoFiles = [...editNewPhotoFiles, ...files.slice(0, remaining)];
		input.value = '';
	}

	// click-outside アクション
	function clickOutside(node: HTMLElement, cb: () => void) {
		const handler = (e: MouseEvent) => {
			if (!node.contains(e.target as Node)) cb();
		};
		document.addEventListener('click', handler, true);
		return { destroy: () => document.removeEventListener('click', handler, true) };
	}
</script>

<main>
	<!-- ヘッダー -->
	<header>
		<div class="logo">🎮</div>
		<h1>Apex 成長日記</h1>
		<p class="subtitle">ひらふうの記録帳</p>
	</header>

	<!-- タブ -->
	<nav class="tabs">
		<button
			class="tab-btn"
			class:active={activeTab === 'write'}
			on:click={() => switchTab('write')}
		>
			<span class="tab-icon">✏️</span> 記録する
		</button>
		<button
			class="tab-btn"
			class:active={activeTab === 'history'}
			on:click={() => switchTab('history')}
		>
			<span class="tab-icon">📋</span> 履歴
		</button>
	</nav>

	<!-- 記録フォーム -->
	{#if activeTab === 'write'}
		<div transition:fade={{ duration: 200 }}>
			<!-- 下書きバナー -->
			{#if showDraftBanner}
				<div class="draft-banner" transition:slide={{ duration: 250 }}>
					<span class="draft-icon">📝</span>
					<span class="draft-text">保存されていない下書きがあります</span>
					<div class="draft-actions">
						<button class="draft-restore-btn" on:click={applyDraft}>復元する</button>
						<button class="draft-discard-btn" on:click={() => clearDraftFromIDB()}>破棄</button>
					</div>
				</div>
			{/if}

			<form class="write-form" on:submit|preventDefault={save}>
				<!-- 日付 + タイトル -->
				<div class="row">
					<div class="field date-field">
						<label for="date">日付</label>
						<input id="date" type="date" bind:value={date} required />
					</div>
					<div class="field title-field">
						<label for="title">タイトル <span class="optional">（任意）</span></label>
						<input
							id="title"
							type="text"
							bind:value={title}
							placeholder="今日の一言..."
							maxlength="50"
							on:input={scheduleDraftSave}
						/>
					</div>
				</div>

				<!-- 日記本文 -->
				<div class="field">
					<label for="nikki">
						日記の内容 <span class="required-mark">*</span>
					</label>
					<textarea
						id="nikki"
						bind:value={nikki}
						placeholder="今日のApexを振り返ろう..."
						rows="7"
						on:input={scheduleDraftSave}
					></textarea>
					{#if hasDraft && !showDraftBanner}
						<span class="draft-indicator">下書き自動保存済み</span>
					{/if}
				</div>

				<!-- ランク -->
				<div class="rank-section">
					<p class="rank-section-label">ランク情報 <span class="optional">（任意）</span></p>
					<div class="row rank-row">
						<div class="field rank-field">
							<label for="rank1">ランク1</label>
							<select id="rank1" bind:value={rank1}>
								{#each RANKS as r}
									<option value={r}>{r === '' ? '-- 未選択 --' : r}</option>
								{/each}
							</select>
						</div>
						<div class="field rp-field">
							<label for="point1">RP1</label>
							<input
								id="point1"
								type="number"
								bind:value={point1}
								placeholder="例: 1000"
								min="0"
								max="99999"
							/>
						</div>
					</div>
					<div class="row rank-row">
						<div class="field rank-field">
							<label for="rank2">ランク2</label>
							<select id="rank2" bind:value={rank2}>
								{#each RANKS as r}
									<option value={r}>{r === '' ? '-- 未選択 --' : r}</option>
								{/each}
							</select>
						</div>
						<div class="field rp-field">
							<label for="point2">RP2</label>
							<input
								id="point2"
								type="number"
								bind:value={point2}
								placeholder="例: 1200"
								min="0"
								max="99999"
							/>
						</div>
					</div>
				</div>

				<!-- 写真 -->
				<div class="photo-section">
					<p class="photo-section-label">写真を追加 <span class="optional">（任意・最大5枚）</span></p>
					{#if photoFiles.length > 0}
						<div class="photo-preview-grid">
							{#each photoFiles as file, i}
								<div class="photo-preview-item">
									<img src={URL.createObjectURL(file)} alt="プレビュー" />
									<button type="button" class="photo-remove-btn" on:click={() => removeNewPhoto(i)}>×</button>
								</div>
							{/each}
						</div>
					{/if}
					{#if photoFiles.length < 5}
						<label class="photo-upload-btn" for="photo-input">
							📷 写真を選択
						</label>
						<input
							id="photo-input"
							type="file"
							accept="image/*"
							multiple
							style="display:none"
							on:change={handlePhotoInput}
						/>
					{/if}
				</div>

				<!-- 保存ボタン -->
				<button type="submit" class="save-btn" disabled={saving || uploadingPhotos}>
					{#if uploadingPhotos}
						<span class="spinner"></span> 写真をアップロード中...
					{:else if saving}
						<span class="spinner"></span> 保存中...
					{:else}
						💾 保存する
					{/if}
				</button>
			</form>
		</div>
	{/if}

	<!-- 履歴タブ -->
	{#if activeTab === 'history'}
		<div transition:fade={{ duration: 200 }}>
			<!-- 検索・ソートコントロール -->
			<div class="history-controls">
				<div class="controls-row">
					<div class="field search-field">
						<label for="search-name">タイトル検索</label>
						<input id="search-name" type="text" bind:value={searchName} placeholder="タイトルで絞り込み..." />
					</div>
					<div class="field date-filter-field">
						<label for="search-date">日付で絞り込み</label>
						<input id="search-date" type="date" bind:value={searchDate} />
					</div>
					<div class="field rank-filter-field">
						<label for="search-rank">ランクで絞り込み</label>
						<select id="search-rank" bind:value={searchRank}>
							<option value="">全てのランク</option>
							{#each RANKS as r}
								{#if r}
									<option value={r}>{r}</option>
								{/if}
							{/each}
						</select>
					</div>
				</div>
				<div class="controls-bottom">
					<div class="sort-group">
						<button
							class="sort-btn"
							class:active={sortOrder === 'newest'}
							on:click={() => (sortOrder = 'newest')}>新しい順</button
						>
						<button
							class="sort-btn"
							class:active={sortOrder === 'oldest'}
							on:click={() => (sortOrder = 'oldest')}>古い順</button
						>
					</div>
					<div class="controls-actions">
						<button
							class="icon-btn"
							title="再読み込み"
							on:click={loadHistory}
							disabled={loadingHistory}
						>
							{loadingHistory ? '⌛' : '🔄'}
						</button>
					</div>
				</div>
			</div>

			<!-- 選択ツールバー -->
			{#if selectedCount > 0}
				<div class="selection-toolbar" transition:slide={{ duration: 200 }}>
					<button class="toolbar-check-btn" on:click={toggleSelectAll}>
						{allSelected ? '☑ 全解除' : '☐ 全選択'}
					</button>
					<span class="selection-count">{selectedCount}件選択中</span>
					<div class="toolbar-actions">
						{#if selectedCount > 0}
							<button
								class="toolbar-delete-btn"
								on:click={() => deleteEntries([...selectedIndices])}
							>
								🗑 削除
							</button>
						{/if}
						<!-- エクスポートドロップダウン -->
						<div
							class="export-wrap"
							use:clickOutside={() => (showExportMenu = false)}
						>
							<button
								class="toolbar-export-btn"
								on:click|stopPropagation={() => (showExportMenu = !showExportMenu)}
							>
								⬇ 出力{selectedCount > 0 ? `（${selectedCount}件）` : '（全件）'}
							</button>
							{#if showExportMenu}
								<div class="export-menu" transition:fly={{ y: -6, duration: 150 }}>
									<button on:click={() => handleExport('pdf')}>📄 PDF</button>
									<button on:click={() => handleExport('photos')}>🖼 写真ZIP</button>
									<button on:click={() => handleExport('excel')}>📊 Excel</button>
									<button on:click={() => handleExport('txt')}>📝 テキスト</button>
									<button on:click={() => handleExport('md')}>📋 Markdown</button>
								</div>
							{/if}
						</div>
					</div>
				</div>
			{/if}

			<!-- 非選択時のエクスポートボタン -->
			{#if selectedCount === 0 && filteredEntries.length > 0}
				<div class="export-bar">
					<span class="entry-count">{filteredEntries.length}件</span>
					<div
						class="export-wrap"
						use:clickOutside={() => (showExportMenu = false)}
					>
						<button
							class="export-btn-ghost"
							on:click|stopPropagation={() => (showExportMenu = !showExportMenu)}
						>
							⬇ 出力
						</button>
						{#if showExportMenu}
							<div class="export-menu export-menu-right" transition:fly={{ y: -6, duration: 150 }}>
								<button on:click={() => handleExport('pdf')}>📄 PDF</button>
								<button on:click={() => handleExport('photos')}>🖼 写真ZIP</button>
								<button on:click={() => handleExport('excel')}>📊 Excel</button>
								<button on:click={() => handleExport('txt')}>📝 テキスト</button>
								<button on:click={() => handleExport('md')}>📋 Markdown</button>
							</div>
						{/if}
					</div>
				</div>
			{/if}

			<!-- エントリー一覧 -->
			{#if loadingHistory}
				<div class="loading-state">
					<div class="loading-spinner"></div>
					<p>読み込み中...</p>
				</div>
			{:else if historyError}
				<div class="error-state">⚠️ エラー: {historyError}</div>
			{:else if filteredEntries.length === 0}
				<div class="empty-state">
					{entries.length === 0 ? 'まだ記録がありません' : '条件に一致する記録がありません'}
				</div>
			{:else}
				<div class="entry-list">
					{#each filteredEntries as entry (entry.row_index)}
						<div
							class="entry-card"
							class:selected={selectedIndices.has(entry.row_index)}
							transition:fade={{ duration: 150 }}
						>
							{#if editingEntry?.row_index === entry.row_index}
								<!-- 編集フォーム -->
								<div class="edit-form" transition:slide={{ duration: 200 }}>
									<p class="edit-title">編集</p>
									<div class="row">
										<div class="field date-field">
											<label>日付</label>
											<input type="date" bind:value={editDate} required />
										</div>
										<div class="field title-field">
											<label>タイトル <span class="optional">（任意）</span></label>
											<input type="text" bind:value={editTitle} placeholder="今日の一言..." maxlength="50" />
										</div>
									</div>
									<div class="field">
										<label>日記の内容 <span class="required-mark">*</span></label>
										<textarea bind:value={editNikki} rows="6"></textarea>
									</div>
									<div class="row rank-row">
										<div class="field rank-field">
											<label>ランク1</label>
											<select bind:value={editRank1}>
												{#each RANKS as r}
													<option value={r}>{r === '' ? '-- 未選択 --' : r}</option>
												{/each}
											</select>
										</div>
										<div class="field rp-field">
											<label>RP1</label>
											<input type="number" bind:value={editPoint1} min="0" max="99999" />
										</div>
									</div>
									<div class="row rank-row">
										<div class="field rank-field">
											<label>ランク2</label>
											<select bind:value={editRank2}>
												{#each RANKS as r}
													<option value={r}>{r === '' ? '-- 未選択 --' : r}</option>
												{/each}
											</select>
										</div>
										<div class="field rp-field">
											<label>RP2</label>
											<input type="number" bind:value={editPoint2} min="0" max="99999" />
										</div>
									</div>
									<!-- 写真編集 -->
									<div class="photo-section">
										<p class="photo-section-label">写真 <span class="optional">（任意・最大5枚）</span></p>
										{#if editPhotoIds.length > 0 || editNewPhotoFiles.length > 0}
											<div class="photo-preview-grid">
												{#each editPhotoIds as photoId, i}
													<div class="photo-preview-item">
														<img src={photoUrl(photoId)} alt="写真" />
														<button type="button" class="photo-remove-btn" on:click={() => removeEditExistingPhoto(i)}>×</button>
													</div>
												{/each}
												{#each editNewPhotoFiles as file, i}
													<div class="photo-preview-item">
														<img src={URL.createObjectURL(file)} alt="プレビュー" />
														<button type="button" class="photo-remove-btn" on:click={() => removeEditNewPhoto(i)}>×</button>
													</div>
												{/each}
											</div>
										{/if}
										{#if editPhotoIds.length + editNewPhotoFiles.length < 5}
											<label class="photo-upload-btn" for="edit-photo-input">
												📷 写真を追加
											</label>
											<input
												id="edit-photo-input"
												type="file"
												accept="image/*"
												multiple
												style="display:none"
												on:change={handleEditPhotoInput}
											/>
										{/if}
									</div>

									<div class="edit-actions">
										<button class="cancel-btn" on:click={cancelEdit} disabled={editSaving}>
											キャンセル
										</button>
										<button class="update-btn" on:click={saveEdit} disabled={editSaving}>
											{editSaving ? '更新中...' : '✓ 更新する'}
										</button>
									</div>
								</div>
							{:else}
								<!-- 通常表示 -->
								<button
									class="check-circle"
									class:checked={selectedIndices.has(entry.row_index)}
									on:click={() => toggleSelect(entry.row_index)}
									aria-label="選択"
								>
									{#if selectedIndices.has(entry.row_index)}✓{/if}
								</button>

								<div class="entry-body">
									<div class="entry-header">
										<span class="entry-date">{fmtDate(entry.date)}</span>
										{#if entry.name}
											<span class="entry-title-badge">{entry.name}</span>
										{/if}
										<div class="entry-actions">
											<button
												class="action-btn feedback-btn"
												class:has-feedback={!!entry.feedback}
												title="フィードバック"
												on:click|stopPropagation={() => openFeedback(entry)}
											>💬 <span class="btn-label">FB</span></button>
											<button
												class="action-btn edit-btn"
												title="編集"
												on:click|stopPropagation={() => startEdit(entry)}
											>✏️ <span class="btn-label">編集</span></button>
											<button
												class="action-btn delete-btn"
												title="削除"
												on:click|stopPropagation={() => deleteEntries([entry.row_index])}
											>🗑️ <span class="btn-label">削除</span></button>
										</div>
									</div>

									{#if entry.nikki}
										<p class="entry-nikki">{entry.nikki}</p>
									{/if}

									{#if entry.rank1 || entry.rank2}
										<div class="entry-ranks">
											{#if entry.rank1}
												<span class="rank-badge" style={rankStyle(entry.rank1)}>
													{entry.rank1}{entry.point1 ? ' ' + entry.point1 + 'pt' : ''}
												</span>
											{/if}
											{#if entry.rank2}
												<span class="rank-badge" style={rankStyle(entry.rank2)}>
													{entry.rank2}{entry.point2 ? ' ' + entry.point2 + 'pt' : ''}
												</span>
											{/if}
										</div>
									{/if}

									{#if entry.photos && entry.photos.length > 0}
										<div class="entry-photos">
											{#each entry.photos as photoId}
												<a href={photoUrl(photoId)} target="_blank" rel="noopener noreferrer">
													<img
														class="entry-photo-thumb"
														src={photoUrl(photoId)}
														alt="写真"
														loading="lazy"
													/>
												</a>
											{/each}
										</div>
									{/if}

									{#if entry.feedback}
										{@const fbData = parseFeedback(entry.feedback)}
										{#if hasFeedback(fbData)}
											<div class="entry-feedback" on:click|stopPropagation={() => openFeedback(entry)} role="button" tabindex="0" on:keypress|stopPropagation>
												<span class="entry-feedback-label">💬 フィードバック</span>
												{#if fbData['free']}
													<p class="entry-feedback-free">{fbData['free']}</p>
												{/if}
												{#each FEEDBACK_CATEGORIES as cat}
													{#if fbData[cat.key]}
														<div class="entry-feedback-cat">
															<span class="entry-feedback-cat-label">{cat.label}</span>
															<p class="entry-feedback-cat-text">{fbData[cat.key]}</p>
														</div>
													{/if}
												{/each}
												{#if fbData['signer']}
													<p class="entry-feedback-signer">— {fbData['signer']}</p>
												{/if}
											</div>
										{/if}
									{/if}

									{#if entry.timestamp}
										<p class="entry-timestamp">{entry.timestamp}</p>
									{/if}
								</div>
							{/if}
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</main>

<!-- 出力プレビューモーダル -->
{#if previewType !== null}
	<!-- svelte-ignore a11y-click-events-have-key-events a11y-no-static-element-interactions -->
	<div class="preview-overlay" on:click|self={closePreview}>
		<div class="preview-modal" role="dialog" aria-modal="true">
			<div class="preview-header">
				<div class="preview-title-row">
					<span class="preview-title">出力プレビュー</span>
					<span class="preview-format-badge">{previewFormatName(previewType)}</span>
					<span class="preview-count">{previewList.length}件</span>
				</div>
				<button class="preview-close-btn" on:click={closePreview} aria-label="閉じる">✕</button>
			</div>

			<div class="preview-body">
				{#if previewType === 'pdf'}
					<!-- PDF プレビュー: 実際のレイアウトに近い形で表示 -->
					{#each previewList as entry}
						<div class="pv-entry">
							<div class="pv-entry-head">
								<span class="pv-date">{fmtDate(entry.date)}</span>
								{#if entry.name}<span class="pv-badge">{entry.name}</span>{/if}
							</div>
							<p class="pv-nikki">{entry.nikki}</p>
							{#if entry.rank1 || entry.rank2}
								<div class="pv-ranks">
									{#if entry.rank1}<span class="pv-rank">{entry.rank1}{entry.point1 ? ' ' + entry.point1 : ''}</span>{/if}
									{#if entry.rank2}<span class="pv-rank">{entry.rank2}{entry.point2 ? ' ' + entry.point2 : ''}</span>{/if}
								</div>
							{/if}
							{#if entry.photos && entry.photos.length > 0}
								<div class="pv-photos">
									{#each entry.photos as photo}
										<img class="pv-photo" src={photoUrl(photo)} alt="写真" loading="lazy" />
									{/each}
								</div>
							{/if}
							{#if entry.feedback}
								{@const pvFbData = parseFeedback(entry.feedback)}
								{#if hasFeedback(pvFbData)}
									<div class="pv-feedback">
										<span class="pv-feedback-label">💬 フィードバック</span>
										{#if pvFbData['free']}
											<p class="pv-feedback-free">{pvFbData['free']}</p>
										{/if}
										{#each FEEDBACK_CATEGORIES as cat}
											{#if pvFbData[cat.key]}
												<div class="pv-feedback-cat">
													<span class="pv-feedback-cat-label">{cat.label}</span>
													<p class="pv-feedback-cat-text">{pvFbData[cat.key]}</p>
												</div>
											{/if}
										{/each}
										{#if pvFbData['signer']}
											<p class="pv-feedback-signer">— {pvFbData['signer']}</p>
										{/if}
									</div>
								{/if}
							{/if}
						</div>
					{/each}
				{:else if previewType === 'txt'}
					<!-- テキストプレビュー -->
					<pre class="pv-pre">{buildTxtContent(previewList)}</pre>
				{:else if previewType === 'md'}
					<!-- Markdown プレビュー -->
					<pre class="pv-pre">{buildMdContent(previewList)}</pre>
				{:else if previewType === 'excel'}
					<!-- Excel プレビュー: テーブル表示 -->
					<div class="pv-table-wrap">
						<table class="pv-table">
							<thead>
								<tr>
									<th>日付</th><th>タイトル</th><th>日記</th>
									<th>ランク1</th><th>RP1</th><th>ランク2</th><th>RP2</th><th>記録日時</th><th>フィードバック</th>
								</tr>
							</thead>
							<tbody>
								{#each previewList as e}
									{@const pvExFb = parseFeedback(e.feedback)}
									<tr>
										<td>{e.date}</td>
										<td>{e.name}</td>
										<td class="pv-td-nikki">{e.nikki}</td>
										<td>{e.rank1}</td>
										<td>{e.point1}</td>
										<td>{e.rank2}</td>
										<td>{e.point2}</td>
										<td>{e.timestamp}</td>
										<td class="pv-td-feedback">
											{#if pvExFb['free']}<span class="pv-fb-cat">{pvExFb['free']}</span>{/if}
											{#each FEEDBACK_CATEGORIES as cat}{#if pvExFb[cat.key]}<span class="pv-fb-cat"><strong>{cat.label}</strong>: {pvExFb[cat.key]}</span>{/if}{/each}
											{#if pvExFb['signer']}<span class="pv-fb-cat">— {pvExFb['signer']}</span>{/if}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{:else if previewType === 'photos'}
					<!-- 写真プレビュー: サムネイル + ファイル名 -->
					{#if previewList.some((e) => e.photos && e.photos.length > 0)}
						{#each previewList as entry}
							{#if entry.photos && entry.photos.length > 0}
								<div class="pv-photo-group">
									<h4 class="pv-photo-group-title">
										{fmtDate(entry.date)}{entry.name ? ' — ' + entry.name : ''}
									</h4>
									<div class="pv-photo-grid">
										{#each entry.photos as photo, i}
											<div class="pv-photo-item">
												<img
													src={photoUrl(photo)}
													alt="写真"
													class="pv-photo-thumb"
													loading="lazy"
												/>
												<span class="pv-photo-name">{photoFilename(entry, i)}</span>
											</div>
										{/each}
									</div>
								</div>
							{/if}
						{/each}
					{:else}
						<p class="pv-empty">写真がありません</p>
					{/if}
				{/if}
			</div>

			<div class="preview-footer">
				<button class="pv-btn-cancel" on:click={closePreview}>キャンセル</button>
				<button class="pv-btn-export" on:click={confirmExport}>
					{previewFormatIcon(previewType)} このまま出力する
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- フィードバックモーダル -->
{#if feedbackEntry !== null}
	<!-- svelte-ignore a11y-click-events-have-key-events a11y-no-static-element-interactions -->
	<div class="preview-overlay" on:click|self={closeFeedback}>
		<div class="preview-modal fb-modal" role="dialog" aria-modal="true">
			<div class="preview-header">
				<div class="preview-title-row">
					<span class="preview-title">フィードバック</span>
					<span class="preview-format-badge">
						{fmtDate(feedbackEntry.date)}{feedbackEntry.name ? ' — ' + feedbackEntry.name : ''}
					</span>
				</div>
				<button class="preview-close-btn" on:click={closeFeedback} aria-label="閉じる">✕</button>
			</div>

			<div class="preview-body fb-body">
				<!-- 左: 日記参照 -->
				<div class="fb-ref">
					<p class="fb-ref-label">📖 日記</p>
					{#if feedbackEntry.nikki}
						<p class="fb-ref-nikki">{feedbackEntry.nikki}</p>
					{/if}
					{#if feedbackEntry.rank1 || feedbackEntry.rank2}
						<div class="fb-ref-ranks">
							{#if feedbackEntry.rank1}
								<span class="pv-rank">{feedbackEntry.rank1}{feedbackEntry.point1 ? ' ' + feedbackEntry.point1 : ''}</span>
							{/if}
							{#if feedbackEntry.rank2}
								<span class="pv-rank">{feedbackEntry.rank2}{feedbackEntry.point2 ? ' ' + feedbackEntry.point2 : ''}</span>
							{/if}
						</div>
					{/if}
					{#if feedbackEntry.photos && feedbackEntry.photos.length > 0}
						<div class="fb-ref-photos">
							{#each feedbackEntry.photos as photo}
								<img src={photoUrl(photo)} alt="写真" class="fb-ref-photo" loading="lazy" />
							{/each}
						</div>
					{/if}
				</div>

				<!-- 右: フィードバック入力 -->
				<div class="fb-input-col">
					<div class="fb-pw-row">
						<label class="fb-pw-label" for="fb-pw">パスワード</label>
						<div class="fb-pw-wrap">
							<input
								id="fb-pw"
								type="password"
								class="fb-pw-input"
								bind:value={feedbackPwInput}
								placeholder="パスワードを入力"
								maxlength="10"
							/>
							{#if feedbackPwInput.length > 0}
								<span class="fb-pw-status" class:ok={feedbackUnlocked}>
									{feedbackUnlocked ? '✓ 解除' : '✗'}
								</span>
							{/if}
						</div>
					</div>

					<div class="fb-free-group">
						<label class="fb-cat-label">📝 自由欄</label>
						<textarea
							class="fb-cat-textarea"
							bind:value={feedbackData['free']}
							disabled={!feedbackUnlocked}
							placeholder={feedbackUnlocked ? '総評や自由コメントを入力...' : ''}
							rows="3"
						></textarea>
					</div>

					<div class="fb-categories">
						{#each FEEDBACK_CATEGORIES as cat}
							<div class="fb-cat-group">
								<label class="fb-cat-label">{cat.label}</label>
								<p class="fb-cat-desc">{cat.desc}</p>
								<textarea
									class="fb-cat-textarea"
									bind:value={feedbackData[cat.key]}
									disabled={!feedbackUnlocked}
									placeholder={feedbackUnlocked ? cat.label + 'についてのフィードバック...' : ''}
									rows="2"
								></textarea>
							</div>
						{/each}
					</div>

					<div class="fb-signer-group">
						<label class="fb-signer-label" for="fb-signer">✍️ 署名</label>
						<div class="fb-signer-wrap">
							<input
								id="fb-signer"
								list="signer-list"
								class="fb-signer-input"
								bind:value={feedbackData['signer']}
								disabled={!feedbackUnlocked}
								placeholder={feedbackUnlocked ? '名前を入力または選択...' : ''}
							/>
							<datalist id="signer-list">
								{#each SIGNER_OPTIONS as opt}
									<option value={opt} />
								{/each}
							</datalist>
						</div>
					</div>
				</div>
			</div>

			<div class="preview-footer">
				<button class="pv-btn-cancel" on:click={closeFeedback}>キャンセル</button>
				{#if feedbackUnlocked}
					<button
						class="pv-btn-export"
						on:click={saveFeedback}
						disabled={feedbackSaving}
					>
						{feedbackSaving ? '保存中...' : '💾 保存'}
					</button>
				{/if}
			</div>
		</div>
	</div>
{/if}

<!-- トースト通知 -->
{#if message}
	<div
		class="toast {messageType}"
		role="alert"
		transition:fly={{ y: 80, duration: 300, easing: quintOut }}
	>
		<span class="toast-icon">{messageType === 'success' ? '✓' : '⚠'}</span>
		<span class="toast-text">{message}</span>
		{#if messageType === 'success' && messageCountdown > 0}
			<span class="toast-countdown">{messageCountdown}</span>
		{/if}
	</div>
{/if}

<style>
	/* ── リセット ── */
	:global(*, *::before, *::after) {
		box-sizing: border-box;
		margin: 0;
		padding: 0;
	}

	:global(body) {
		font-family: 'Hiragino Sans', 'Hiragino Kaku Gothic ProN', 'Noto Sans JP', sans-serif;
		background: #f1f5f9;
		color: #0c2340;
		min-height: 100vh;
	}

	/* ── レイアウト ── */
	main {
		max-width: 680px;
		margin: 0 auto;
		padding: 1.5rem 1rem 6rem;
	}

	/* ── ヘッダー ── */
	header {
		text-align: center;
		margin-bottom: 2rem;
		padding: 1.5rem 1rem 1.25rem;
		background: #fff;
		border-radius: 16px;
		box-shadow: 0 1px 4px rgba(2, 132, 199, 0.08), 0 0 0 1px rgba(2, 132, 199, 0.06);
	}

	.logo {
		font-size: 2.2rem;
		line-height: 1;
		margin-bottom: 0.4rem;
	}

	h1 {
		font-size: 1.6rem;
		font-weight: 800;
		color: #0284c7;
		letter-spacing: -0.01em;
	}

	.subtitle {
		font-size: 0.82rem;
		color: #9ca3af;
		margin-top: 0.25rem;
	}

	/* ── タブ ── */
	.tabs {
		display: flex;
		gap: 0;
		margin-bottom: 1.5rem;
		background: #fff;
		border-radius: 12px;
		padding: 0.25rem;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
	}

	.tab-btn {
		flex: 1;
		background: none;
		border: none;
		border-radius: 9px;
		color: #6b7280;
		cursor: pointer;
		font-family: inherit;
		font-size: 0.9rem;
		font-weight: 600;
		padding: 0.65rem 1rem;
		transition: all 0.2s;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.4rem;
	}

	.tab-btn:hover:not(.active) {
		background: #f3f4f6;
		color: #374151;
	}

	.tab-btn.active {
		background: #0284c7;
		color: #fff;
		box-shadow: 0 2px 8px rgba(2, 132, 199, 0.35);
	}

	.tab-icon {
		font-size: 1rem;
	}

	/* ── 下書きバナー ── */
	.draft-banner {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		flex-wrap: wrap;
		background: #fffbeb;
		border: 1px solid #fcd34d;
		border-radius: 12px;
		padding: 0.75rem 1rem;
		margin-bottom: 1rem;
		font-size: 0.875rem;
	}

	.draft-icon {
		font-size: 1.1rem;
	}

	.draft-text {
		flex: 1;
		color: #92400e;
		font-weight: 500;
	}

	.draft-actions {
		display: flex;
		gap: 0.5rem;
	}

	.draft-restore-btn {
		background: #f59e0b;
		border: none;
		border-radius: 6px;
		color: #fff;
		cursor: pointer;
		font-family: inherit;
		font-size: 0.8rem;
		font-weight: 600;
		padding: 0.3rem 0.75rem;
	}

	.draft-discard-btn {
		background: none;
		border: 1px solid #fcd34d;
		border-radius: 6px;
		color: #92400e;
		cursor: pointer;
		font-family: inherit;
		font-size: 0.8rem;
		padding: 0.3rem 0.75rem;
	}

	/* ── フォーム ── */
	.write-form {
		background: #fff;
		border-radius: 16px;
		padding: 1.5rem;
		box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
		display: flex;
		flex-direction: column;
		gap: 1.25rem;
	}

	.row {
		display: flex;
		gap: 0.75rem;
	}

	.date-field {
		flex: 0 0 160px;
	}

	.title-field {
		flex: 1;
	}

	.rank-section {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		background: #fafafa;
		border: 1px solid #f3f4f6;
		border-radius: 12px;
		padding: 1rem;
	}

	.rank-section-label {
		font-size: 0.82rem;
		font-weight: 600;
		color: #6b7280;
	}

	.rank-row .rank-field {
		flex: 1;
	}

	.rank-row .rp-field {
		flex: 0 0 120px;
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
	}

	label {
		font-size: 0.8rem;
		color: #6b7280;
		font-weight: 600;
		letter-spacing: 0.02em;
	}

	.optional {
		font-size: 0.72rem;
		color: #9ca3af;
		font-weight: 400;
	}

	.required-mark {
		color: #ef4444;
		font-weight: 700;
	}

	.draft-indicator {
		font-size: 0.72rem;
		color: #9ca3af;
		text-align: right;
	}

	input[type='text'],
	input[type='date'],
	input[type='number'],
	select,
	textarea {
		background: #fff;
		border: 1.5px solid #e5e7eb;
		border-radius: 9px;
		color: #0c2340;
		font-size: 0.95rem;
		padding: 0.6rem 0.85rem;
		width: 100%;
		transition: border-color 0.15s, box-shadow 0.15s;
		font-family: inherit;
		line-height: 1.5;
	}

	input[type='text']:focus,
	input[type='date']:focus,
	input[type='number']:focus,
	select:focus,
	textarea:focus {
		outline: none;
		border-color: #0369a1;
		box-shadow: 0 0 0 3px rgba(2, 132, 199, 0.12);
	}

	textarea {
		resize: vertical;
		min-height: 160px;
		line-height: 1.75;
		font-size: 0.95rem;
	}

	select {
		cursor: pointer;
		appearance: none;
		background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' fill='none' viewBox='0 0 20 20'%3E%3Cpath stroke='%239ca3af' stroke-linecap='round' stroke-linejoin='round' stroke-width='1.5' d='M6 8l4 4 4-4'/%3E%3C/svg%3E");
		background-repeat: no-repeat;
		background-position: right 0.6rem center;
		background-size: 1.1em;
		padding-right: 2rem;
	}

	/* ── 保存ボタン ── */
	.save-btn {
		background: linear-gradient(135deg, #0369a1, #0284c7);
		border: none;
		border-radius: 12px;
		color: #fff;
		cursor: pointer;
		font-family: inherit;
		font-size: 1rem;
		font-weight: 700;
		letter-spacing: 0.03em;
		padding: 0.9rem;
		transition: opacity 0.2s, transform 0.1s, box-shadow 0.2s;
		width: 100%;
		box-shadow: 0 4px 14px rgba(2, 132, 199, 0.4);
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
	}

	.save-btn:hover:not(:disabled) {
		opacity: 0.9;
		box-shadow: 0 6px 20px rgba(2, 132, 199, 0.45);
	}

	.save-btn:active:not(:disabled) {
		transform: scale(0.98);
	}

	.save-btn:disabled {
		background: #d1d5db;
		box-shadow: none;
		cursor: not-allowed;
	}

	.spinner {
		width: 16px;
		height: 16px;
		border: 2px solid rgba(255, 255, 255, 0.4);
		border-top-color: #fff;
		border-radius: 50%;
		animation: spin 0.7s linear infinite;
		display: inline-block;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}

	/* ── 履歴コントロール ── */
	.history-controls {
		background: #fff;
		border-radius: 14px;
		padding: 1rem 1.25rem;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.07);
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		margin-bottom: 0.75rem;
	}

	.controls-row {
		display: flex;
		gap: 0.75rem;
	}

	.search-field {
		flex: 1;
	}

	.date-filter-field {
		flex: 0 0 180px;
	}

	.controls-bottom {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.5rem;
	}

	.sort-group {
		display: flex;
		gap: 0.35rem;
	}

	.sort-btn {
		background: #f3f4f6;
		border: 1.5px solid transparent;
		border-radius: 8px;
		color: #6b7280;
		cursor: pointer;
		font-family: inherit;
		font-size: 0.8rem;
		font-weight: 600;
		padding: 0.35rem 0.75rem;
		transition: all 0.15s;
	}

	.sort-btn.active {
		background: #e0f2fe;
		border-color: #7dd3fc;
		color: #0284c7;
	}

	.sort-btn:not(.active):hover {
		background: #e5e7eb;
		color: #374151;
	}

	.controls-actions {
		display: flex;
		gap: 0.5rem;
		align-items: center;
	}

	.icon-btn {
		background: #f3f4f6;
		border: none;
		border-radius: 8px;
		cursor: pointer;
		font-size: 1rem;
		padding: 0.35rem 0.6rem;
		transition: background 0.15s;
	}

	.icon-btn:hover:not(:disabled) {
		background: #e5e7eb;
	}

	.select-toggle-btn {
		background: #f3f4f6;
		border: 1.5px solid transparent;
		border-radius: 8px;
		color: #6b7280;
		cursor: pointer;
		font-family: inherit;
		font-size: 0.8rem;
		font-weight: 600;
		padding: 0.35rem 0.75rem;
		transition: all 0.15s;
	}

	.select-toggle-btn.active {
		background: #e0f2fe;
		border-color: #7dd3fc;
		color: #0284c7;
	}

	.select-toggle-btn:not(.active):hover {
		background: #e5e7eb;
		color: #374151;
	}

	/* ── 選択ツールバー ── */
	.selection-toolbar {
		background: #0284c7;
		border-radius: 12px;
		padding: 0.75rem 1rem;
		margin-bottom: 0.75rem;
		display: flex;
		align-items: center;
		gap: 0.75rem;
		flex-wrap: wrap;
	}

	.toolbar-check-btn {
		background: rgba(255, 255, 255, 0.2);
		border: 1px solid rgba(255, 255, 255, 0.3);
		border-radius: 7px;
		color: #fff;
		cursor: pointer;
		font-family: inherit;
		font-size: 0.8rem;
		font-weight: 600;
		padding: 0.3rem 0.75rem;
	}

	.selection-count {
		color: rgba(255, 255, 255, 0.9);
		font-size: 0.85rem;
		font-weight: 600;
		flex: 1;
	}

	.toolbar-actions {
		display: flex;
		gap: 0.5rem;
		align-items: center;
	}

	.toolbar-delete-btn {
		background: rgba(239, 68, 68, 0.85);
		border: none;
		border-radius: 7px;
		color: #fff;
		cursor: pointer;
		font-family: inherit;
		font-size: 0.8rem;
		font-weight: 600;
		padding: 0.35rem 0.85rem;
		transition: background 0.15s;
	}

	.toolbar-delete-btn:hover {
		background: rgba(239, 68, 68, 1);
	}

	.toolbar-export-btn {
		background: rgba(255, 255, 255, 0.15);
		border: 1px solid rgba(255, 255, 255, 0.3);
		border-radius: 7px;
		color: #fff;
		cursor: pointer;
		font-family: inherit;
		font-size: 0.8rem;
		font-weight: 600;
		padding: 0.35rem 0.85rem;
	}

	/* ── エクスポートバー (非選択時) ── */
	.export-bar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 0.75rem;
	}

	.entry-count {
		font-size: 0.8rem;
		color: #9ca3af;
	}

	.export-btn-ghost {
		background: none;
		border: 1.5px solid #e5e7eb;
		border-radius: 8px;
		color: #6b7280;
		cursor: pointer;
		font-family: inherit;
		font-size: 0.8rem;
		font-weight: 600;
		padding: 0.35rem 0.85rem;
		transition: all 0.15s;
	}

	.export-btn-ghost:hover {
		border-color: #0369a1;
		color: #0284c7;
	}

	/* ── エクスポートドロップダウン ── */
	.export-wrap {
		position: relative;
	}

	.export-menu {
		position: absolute;
		top: calc(100% + 6px);
		left: 0;
		background: #fff;
		border: 1px solid #e5e7eb;
		border-radius: 10px;
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
		overflow: hidden;
		z-index: 50;
		min-width: 140px;
	}

	.export-menu-right {
		left: auto;
		right: 0;
	}

	.export-menu button {
		background: none;
		border: none;
		color: #374151;
		cursor: pointer;
		display: block;
		font-family: inherit;
		font-size: 0.875rem;
		padding: 0.6rem 1rem;
		text-align: left;
		transition: background 0.1s;
		width: 100%;
	}

	.export-menu button:hover {
		background: #f3f4f6;
	}

	/* ── ローディング / エラー / 空 ── */
	.loading-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.75rem;
		padding: 3rem 0;
		color: #9ca3af;
		font-size: 0.9rem;
	}

	.loading-spinner {
		width: 32px;
		height: 32px;
		border: 3px solid #e5e7eb;
		border-top-color: #0369a1;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	.error-state {
		background: #fef2f2;
		border: 1px solid #fecaca;
		border-radius: 12px;
		color: #dc2626;
		font-size: 0.9rem;
		padding: 1rem 1.25rem;
		text-align: center;
	}

	.empty-state {
		color: #9ca3af;
		font-size: 0.95rem;
		text-align: center;
		padding: 3rem 0;
	}

	/* ── エントリーカード ── */
	.entry-list {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.entry-card {
		background: #fff;
		border: 1.5px solid #e5e7eb;
		border-radius: 14px;
		padding: 1rem 1.25rem;
		transition: border-color 0.2s, box-shadow 0.2s;
		display: flex;
		gap: 0.75rem;
		align-items: flex-start;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
	}

	.entry-card:hover {
		border-color: #93c5fd;
		box-shadow: 0 4px 14px rgba(2, 132, 199, 0.12);
	}

	.entry-card.selected {
		border-color: #0369a1;
		background: #f0f9ff;
	}

	.entry-body {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
		min-width: 0;
	}

	.entry-header {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		flex-wrap: wrap;
	}

	.entry-date {
		font-size: 0.95rem;
		font-weight: 700;
		color: #0284c7;
	}

	.entry-title-badge {
		background: #e0f2fe;
		border-radius: 20px;
		color: #0369a1;
		font-size: 0.75rem;
		font-weight: 600;
		padding: 0.15rem 0.6rem;
	}

	.entry-actions {
		margin-left: auto;
		display: flex;
		gap: 0.25rem;
	}

	.action-btn {
		background: none;
		border: 1px solid #e5e7eb;
		border-radius: 7px;
		cursor: pointer;
		font-size: 0.85rem;
		line-height: 1;
		padding: 0.3rem 0.5rem;
		transition: all 0.15s;
	}

	.action-btn:hover {
		background: #f3f4f6;
	}

	.delete-btn:hover {
		background: #fef2f2;
		border-color: #fecaca;
	}

	.btn-label {
		font-size: 0.75rem;
		font-weight: 500;
	}

	/* 入力時と参照時のスタイルを一致させる */
	.entry-nikki {
		color: #1f2937;
		font-size: 0.93rem;
		line-height: 1.8;
		white-space: pre-wrap;
		word-break: break-word;
		font-family: inherit;
		background: #fffbeb;
		border-left: 3px solid #fbbf24;
		border-radius: 0 8px 8px 0;
		padding: 0.5rem 0.75rem;
	}

	.entry-ranks {
		display: flex;
		gap: 0.5rem;
		flex-wrap: wrap;
	}

	.rank-badge {
		background: #f3f4f6;
		border: 1.5px solid #e5e7eb;
		border-radius: 20px;
		color: #374151;
		font-size: 0.78rem;
		font-weight: 700;
		padding: 0.2rem 0.7rem;
		letter-spacing: 0.02em;
	}

	.entry-timestamp {
		color: #d1d5db;
		font-size: 0.72rem;
		text-align: right;
	}

	/* ── チェックサークル ── */
	.check-circle {
		flex-shrink: 0;
		width: 22px;
		height: 22px;
		border-radius: 50%;
		border: 2px solid #d1d5db;
		background: none;
		cursor: pointer;
		transition: all 0.15s;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 0.75rem;
		color: transparent;
		margin-top: 2px;
		font-weight: 700;
	}

	.check-circle.checked {
		background: #0369a1;
		border-color: #0369a1;
		color: #fff;
	}

	.check-circle:hover:not(.checked) {
		border-color: #0369a1;
		background: #e0f2fe;
	}

	/* ── 編集フォーム ── */
	.edit-form {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 0.9rem;
	}

	.edit-title {
		font-size: 0.82rem;
		font-weight: 700;
		color: #0284c7;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.edit-actions {
		display: flex;
		gap: 0.5rem;
		justify-content: flex-end;
		margin-top: 0.25rem;
	}

	.cancel-btn {
		background: #f3f4f6;
		border: none;
		border-radius: 8px;
		color: #6b7280;
		cursor: pointer;
		font-family: inherit;
		font-size: 0.875rem;
		font-weight: 600;
		padding: 0.5rem 1.1rem;
		transition: background 0.15s;
	}

	.cancel-btn:hover:not(:disabled) {
		background: #e5e7eb;
	}

	.update-btn {
		background: #0284c7;
		border: none;
		border-radius: 8px;
		color: #fff;
		cursor: pointer;
		font-family: inherit;
		font-size: 0.875rem;
		font-weight: 700;
		padding: 0.5rem 1.25rem;
		transition: opacity 0.15s;
	}

	.update-btn:hover:not(:disabled) {
		opacity: 0.88;
	}

	.update-btn:disabled,
	.cancel-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	/* ── トースト通知 ── */
	.toast {
		position: fixed;
		bottom: 1.75rem;
		left: 50%;
		transform: translateX(-50%);
		display: flex;
		align-items: center;
		gap: 0.6rem;
		padding: 0.75rem 1.25rem;
		border-radius: 12px;
		font-size: 0.9rem;
		font-weight: 600;
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
		z-index: 200;
		white-space: nowrap;
	}

	.toast.success {
		background: #059669;
		color: #fff;
	}

	.toast.error {
		background: #dc2626;
		color: #fff;
	}

	.toast-icon {
		font-size: 1rem;
		font-weight: 800;
	}

	.toast-countdown {
		background: rgba(255, 255, 255, 0.25);
		border-radius: 20px;
		font-size: 0.75rem;
		min-width: 22px;
		padding: 0.1rem 0.4rem;
		text-align: center;
	}

	/* ── 写真セクション ── */
	.photo-section {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		background: #fafafa;
		border: 1px solid #f3f4f6;
		border-radius: 12px;
		padding: 1rem;
	}

	.photo-section-label {
		font-size: 0.82rem;
		font-weight: 600;
		color: #6b7280;
	}

	.photo-upload-btn {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		background: #fff;
		border: 1.5px dashed #d1d5db;
		border-radius: 9px;
		color: #6b7280;
		cursor: pointer;
		font-family: inherit;
		font-size: 0.875rem;
		font-weight: 600;
		padding: 0.6rem 1rem;
		transition: all 0.15s;
		width: fit-content;
	}

	.photo-upload-btn:hover {
		border-color: #0369a1;
		color: #0284c7;
		background: #f0f9ff;
	}

	.photo-preview-grid {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}

	.photo-preview-item {
		position: relative;
		width: 80px;
		height: 80px;
		border-radius: 8px;
		overflow: hidden;
		flex-shrink: 0;
	}

	.photo-preview-item img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.photo-remove-btn {
		position: absolute;
		top: 2px;
		right: 2px;
		background: rgba(0, 0, 0, 0.6);
		border: none;
		border-radius: 50%;
		color: #fff;
		cursor: pointer;
		font-size: 0.75rem;
		font-weight: 700;
		width: 18px;
		height: 18px;
		display: flex;
		align-items: center;
		justify-content: center;
		line-height: 1;
		padding: 0;
	}

	.photo-remove-btn:hover {
		background: rgba(239, 68, 68, 0.9);
	}

	/* ── エントリー写真表示 ── */
	.entry-photos {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}

	.entry-photo-thumb {
		width: 80px;
		height: 80px;
		object-fit: cover;
		border-radius: 8px;
		border: 1px solid #e5e7eb;
		cursor: pointer;
		transition: opacity 0.15s;
		display: block;
	}

	.entry-photo-thumb:hover {
		opacity: 0.85;
	}

	/* ── レスポンシブ ── */
	@media (max-width: 520px) {
		h1 {
			font-size: 1.35rem;
		}

		.row {
			flex-direction: column;
		}

		.date-field,
		.title-field {
			flex: unset;
		}

		.controls-row {
			flex-direction: column;
		}

		.date-filter-field {
			flex: unset;
		}

		.rank-row .rp-field {
			flex: 0 0 100px;
		}
	}

	/* ── 出力プレビューモーダル ── */
	.preview-overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.55);
		z-index: 200;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 16px;
	}

	.preview-modal {
		background: #fff;
		border-radius: 14px;
		width: 100%;
		max-width: 800px;
		max-height: 90vh;
		display: flex;
		flex-direction: column;
		box-shadow: 0 24px 64px rgba(0, 0, 0, 0.28);
	}

	.preview-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 14px 18px;
		border-bottom: 1px solid #e5e7eb;
		flex-shrink: 0;
	}

	.preview-title-row {
		display: flex;
		align-items: center;
		gap: 10px;
	}

	.preview-title {
		font-size: 15px;
		font-weight: 700;
		color: #0c2340;
	}

	.preview-format-badge {
		font-size: 11px;
		background: #e0f2fe;
		color: #0369a1;
		padding: 2px 10px;
		border-radius: 20px;
		font-weight: 600;
	}

	.preview-count {
		font-size: 12px;
		color: #6b7280;
	}

	.preview-close-btn {
		background: none;
		border: none;
		cursor: pointer;
		font-size: 15px;
		color: #9ca3af;
		padding: 4px 6px;
		border-radius: 6px;
		line-height: 1;
		transition: background 0.15s;
	}

	.preview-close-btn:hover {
		background: #f3f4f6;
		color: #374151;
	}

	.preview-body {
		flex: 1;
		overflow-y: auto;
		padding: 18px 20px;
		min-height: 0;
	}

	.preview-footer {
		display: flex;
		justify-content: flex-end;
		gap: 10px;
		padding: 12px 18px;
		border-top: 1px solid #e5e7eb;
		flex-shrink: 0;
		background: #f9fafb;
		border-radius: 0 0 14px 14px;
	}

	.pv-btn-cancel {
		padding: 8px 18px;
		border: 1px solid #d1d5db;
		background: #fff;
		border-radius: 7px;
		cursor: pointer;
		font-size: 13px;
		color: #374151;
		transition: background 0.15s;
	}

	.pv-btn-cancel:hover {
		background: #f3f4f6;
	}

	.pv-btn-export {
		padding: 8px 20px;
		background: #0284c7;
		color: #fff;
		border: none;
		border-radius: 7px;
		cursor: pointer;
		font-size: 13px;
		font-weight: 600;
		transition: background 0.15s;
	}

	.pv-btn-export:hover {
		background: #0369a1;
	}

	/* PDF プレビュー */
	.pv-entry {
		margin-bottom: 20px;
		padding-bottom: 20px;
		border-bottom: 1px solid #e5e7eb;
	}

	.pv-entry:last-child {
		border-bottom: none;
	}

	.pv-entry-head {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-bottom: 8px;
	}

	.pv-date {
		font-size: 14px;
		font-weight: 700;
		color: #0284c7;
	}

	.pv-badge {
		font-size: 11px;
		background: #e0f2fe;
		color: #0369a1;
		padding: 2px 8px;
		border-radius: 20px;
	}

	.pv-nikki {
		white-space: pre-wrap;
		font-size: 13px;
		line-height: 1.75;
		color: #1f2937;
	}

	.pv-ranks {
		display: flex;
		gap: 6px;
		margin-top: 8px;
		flex-wrap: wrap;
	}

	.pv-rank {
		font-size: 12px;
		background: #f3f4f6;
		border: 1px solid #e5e7eb;
		padding: 2px 8px;
		border-radius: 6px;
	}

	.pv-photos {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		margin-top: 10px;
	}

	.pv-photo {
		max-width: 180px;
		max-height: 140px;
		object-fit: cover;
		border-radius: 6px;
		border: 1px solid #e5e7eb;
	}

	/* テキスト / Markdown プレビュー */
	.pv-pre {
		font-family: 'Courier New', Courier, monospace;
		font-size: 12px;
		white-space: pre-wrap;
		line-height: 1.65;
		color: #1f2937;
	}

	/* Excel プレビュー */
	.pv-table-wrap {
		overflow-x: auto;
	}

	.pv-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 12px;
	}

	.pv-table th {
		background: #f1f5f9;
		padding: 6px 10px;
		text-align: left;
		border: 1px solid #e5e7eb;
		font-weight: 600;
		white-space: nowrap;
		color: #374151;
	}

	.pv-table td {
		padding: 6px 10px;
		border: 1px solid #e5e7eb;
		color: #374151;
		max-width: 200px;
		vertical-align: top;
	}

	.pv-td-nikki {
		max-width: 260px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	/* 写真 ZIP プレビュー */
	.pv-photo-group {
		margin-bottom: 22px;
	}

	.pv-photo-group-title {
		font-size: 13px;
		font-weight: 600;
		color: #374151;
		margin-bottom: 10px;
	}

	.pv-photo-grid {
		display: flex;
		flex-wrap: wrap;
		gap: 12px;
	}

	.pv-photo-item {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 5px;
	}

	.pv-photo-thumb {
		width: 120px;
		height: 100px;
		object-fit: cover;
		border-radius: 6px;
		border: 1px solid #e5e7eb;
	}

	.pv-photo-name {
		font-size: 10px;
		color: #6b7280;
		max-width: 120px;
		text-align: center;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.pv-empty {
		color: #9ca3af;
		font-size: 14px;
		text-align: center;
		padding: 48px 0;
	}

	/* ── フィードバック ── */
	.feedback-btn {
		opacity: 0.45;
		transition: opacity 0.15s;
	}

	.feedback-btn:hover,
	.feedback-btn.has-feedback {
		opacity: 1;
	}

	.entry-feedback {
		margin-top: 10px;
		padding: 10px 14px;
		background: linear-gradient(135deg, #eff6ff 0%, #f0f9ff 100%);
		border-left: 3px solid #3b82f6;
		border-radius: 0 8px 8px 0;
		cursor: pointer;
		transition: background 0.15s, box-shadow 0.15s;
		box-shadow: 0 1px 4px rgba(59, 130, 246, 0.08);
	}

	.entry-feedback:hover {
		background: linear-gradient(135deg, #dbeafe 0%, #e0f2fe 100%);
		box-shadow: 0 2px 8px rgba(59, 130, 246, 0.14);
	}

	.entry-feedback-label {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 10.5px;
		font-weight: 700;
		color: #2563eb;
		text-transform: uppercase;
		letter-spacing: 0.06em;
		margin-bottom: 6px;
		padding-bottom: 5px;
		border-bottom: 1px dashed rgba(59, 130, 246, 0.3);
	}

	.entry-feedback-cat {
		margin-top: 4px;
	}

	.entry-feedback-cat-label {
		display: inline-block;
		font-size: 10px;
		font-weight: 700;
		color: #1d4ed8;
		background: rgba(59, 130, 246, 0.1);
		padding: 1px 7px;
		border-radius: 10px;
		margin-bottom: 2px;
	}

	.entry-feedback-free {
		font-size: 13px;
		color: #1e3a5f;
		white-space: pre-wrap;
		line-height: 1.6;
		margin: 4px 0 6px;
	}

	.entry-feedback-cat-text {
		font-size: 12px;
		color: #1e3a5f;
		white-space: pre-wrap;
		line-height: 1.5;
		margin: 1px 0 0;
	}

	.entry-feedback-signer {
		font-size: 12px;
		color: #4b5563;
		text-align: right;
		margin-top: 8px;
		padding-top: 6px;
		border-top: 1px solid rgba(59, 130, 246, 0.2);
		font-style: italic;
		font-weight: 500;
	}

	/* フィードバックモーダル */
	.fb-modal {
		max-width: 860px;
	}

	.fb-body {
		display: flex;
		gap: 18px;
		align-items: flex-start;
		padding: 16px 20px;
	}

	/* 左: 日記参照パネル */
	.fb-ref {
		flex: 1 1 0;
		min-width: 0;
		max-height: 60vh;
		overflow-y: auto;
		padding: 14px 16px;
		background: #f8fafc;
		border: 1px solid #e5e7eb;
		border-radius: 8px;
	}

	.fb-ref-label {
		font-size: 11px;
		font-weight: 700;
		color: #6b7280;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		margin-bottom: 10px;
	}

	.fb-ref-nikki {
		font-size: 13px;
		line-height: 1.75;
		color: #1f2937;
		white-space: pre-wrap;
	}

	.fb-ref-ranks {
		display: flex;
		gap: 6px;
		flex-wrap: wrap;
		margin-top: 10px;
	}

	.fb-ref-photos {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
		margin-top: 10px;
	}

	.fb-ref-photo {
		width: 90px;
		height: 72px;
		object-fit: cover;
		border-radius: 5px;
		border: 1px solid #e5e7eb;
	}

	/* 右: フィードバック入力パネル */
	.fb-input-col {
		flex: 1 1 0;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.fb-pw-row {
		margin-bottom: 14px;
	}

	.fb-pw-label {
		display: block;
		font-size: 12px;
		font-weight: 600;
		color: #374151;
		margin-bottom: 6px;
	}

	.fb-pw-wrap {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.fb-pw-input {
		flex: 1;
		padding: 8px 12px;
		border: 1px solid #d1d5db;
		border-radius: 7px;
		font-size: 14px;
		outline: none;
		transition: border-color 0.15s;
	}

	.fb-pw-input:focus {
		border-color: #3b82f6;
		box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.15);
	}

	.fb-pw-status {
		font-size: 13px;
		font-weight: 600;
		color: #ef4444;
		white-space: nowrap;
	}

	.fb-pw-status.ok {
		color: #22c55e;
	}

	.fb-textarea {
		width: 100%;
		padding: 10px 12px;
		border: 1px solid #d1d5db;
		border-radius: 7px;
		font-size: 14px;
		font-family: inherit;
		line-height: 1.65;
		resize: vertical;
		outline: none;
		transition: border-color 0.15s, background 0.15s;
	}

	.fb-textarea:focus {
		border-color: #3b82f6;
		box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.15);
	}

	.fb-textarea:disabled {
		background: #f9fafb;
		color: #9ca3af;
		cursor: not-allowed;
	}

/* ── フィードバックカテゴリ ── */
	.fb-categories {
		display: flex;
		flex-direction: column;
		gap: 10px;
		overflow-y: auto;
		flex: 1;
	}

	.fb-cat-group {
		display: flex;
		flex-direction: column;
		gap: 3px;
	}

	.fb-cat-label {
		font-size: 12px;
		font-weight: 700;
		color: #1e40af;
	}

	.fb-cat-desc {
		font-size: 11px;
		color: #6b7280;
		margin: 0;
	}

	.fb-cat-textarea {
		width: 100%;
		padding: 6px 10px;
		border: 1px solid #d1d5db;
		border-radius: 6px;
		font-size: 13px;
		font-family: inherit;
		line-height: 1.55;
		resize: vertical;
		outline: none;
		transition: border-color 0.15s;
	}

	.fb-cat-textarea:focus {
		border-color: #3b82f6;
		box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.12);
	}

	.fb-cat-textarea:disabled {
		background: #f9fafb;
		color: #9ca3af;
		cursor: not-allowed;
	}

	.fb-free-group {
		padding-bottom: 10px;
		border-bottom: 1px solid #e5e7eb;
		margin-bottom: 2px;
	}

	.fb-signer-group {
		padding-top: 10px;
		border-top: 1px solid #e5e7eb;
		margin-top: 2px;
	}

	.fb-signer-label {
		display: block;
		font-size: 12px;
		font-weight: 700;
		color: #374151;
		margin-bottom: 4px;
	}

	.fb-signer-wrap {
		display: flex;
		gap: 6px;
		align-items: center;
	}

	.fb-signer-input {
		width: 100%;
		padding: 6px 10px;
		border: 1px solid #d1d5db;
		border-radius: 6px;
		font-size: 14px;
		font-family: inherit;
		outline: none;
		transition: border-color 0.15s;
	}

	.fb-signer-input:focus {
		border-color: #3b82f6;
		box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.12);
	}

	.fb-signer-input:disabled {
		background: #f9fafb;
		color: #9ca3af;
		cursor: not-allowed;
	}

/* ── プレビュー内フィードバック ── */
	.pv-feedback {
		margin-top: 10px;
		padding: 8px 12px;
		background: #eff6ff;
		border-left: 3px solid #3b82f6;
		border-radius: 0 6px 6px 0;
	}

	.pv-feedback-label {
		display: block;
		font-size: 11px;
		font-weight: 700;
		color: #1d4ed8;
		margin-bottom: 6px;
	}

	.pv-feedback-cat {
		margin-bottom: 4px;
	}

	.pv-feedback-cat-label {
		font-size: 11px;
		font-weight: 700;
		color: #1e40af;
	}

	.pv-feedback-free {
		font-size: 13px;
		color: #1f2937;
		white-space: pre-wrap;
		margin-bottom: 6px;
	}

	.pv-feedback-cat-text {
		font-size: 12px;
		color: #1f2937;
		white-space: pre-wrap;
		line-height: 1.5;
		margin: 1px 0 0;
	}

	.pv-feedback-signer {
		font-size: 12px;
		color: #6b7280;
		text-align: right;
		margin-top: 6px;
		font-style: italic;
	}

	.pv-td-feedback {
		max-width: 200px;
		white-space: pre-wrap;
		font-size: 12px;
	}

	.pv-fb-cat {
		display: block;
		margin-bottom: 2px;
		font-size: 11px;
	}
</style>