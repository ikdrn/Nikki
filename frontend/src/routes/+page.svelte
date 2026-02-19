<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { fade, fly, slide } from 'svelte/transition';
	import { quintOut } from 'svelte/easing';

	const RANKS = ['', 'ブロンズ', 'シルバー', 'ゴールド', 'プラチナ', 'ダイヤ', 'マスター', 'プレデター'];
	const API_BASE = import.meta.env.VITE_API_BASE_URL || '/api';

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
			const data = await res.json();
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
				showMsg(data.message || '保存失敗', 'error');
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
				const data = await res.json().catch(() => ({}));
				throw new Error(data.error || `HTTP ${res.status}`);
			}
			entries = await res.json();
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
		try {
			const res = await fetch(`${API_BASE}/nikki`, {
				method: 'DELETE',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ row_indices: indices })
			});
			const data = await res.json();
			if (data.success) {
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
			const data = await res.json();
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

	function exportTxt() {
		const list = getEntriesToExport();
		const content = list
			.map((e) => {
				const lines = [
					'='.repeat(40),
					`日付: ${fmtDate(e.date)}`,
					e.name ? `タイトル: ${e.name}` : null,
					'',
					e.nikki,
					'',
					e.rank1 ? `ランク1: ${e.rank1}${e.point1 ? ' ' + e.point1 : ''}` : null,
					e.rank2 ? `ランク2: ${e.rank2}${e.point2 ? ' ' + e.point2 : ''}` : null,
					e.timestamp ? `記録日時: ${e.timestamp}` : null
				].filter((l) => l !== null);
				return lines.join('\n');
			})
			.join('\n\n');
		downloadBlob(content, `apex-diary-${todayStr()}.txt`, 'text/plain;charset=utf-8');
	}

	function exportMd() {
		const list = getEntriesToExport();
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
				parts.push('---');
				return parts.join('\n');
			})
			.join('\n\n');
		downloadBlob(header + body, `apex-diary-${todayStr()}.md`, 'text/markdown;charset=utf-8');
	}

	async function exportExcel() {
		const list = getEntriesToExport();
		const xlsxMod = await import('xlsx');
		const XLSX = xlsxMod.default ?? xlsxMod;
		const wsData = [
			['日付', 'タイトル', '日記', 'ランク1', 'RP1', 'ランク2', 'RP2', '記録日時'],
			...list.map((e) => [e.date, e.name, e.nikki, e.rank1, e.point1, e.rank2, e.point2, e.timestamp])
		];
		const ws = XLSX.utils.aoa_to_sheet(wsData);
		const wb = XLSX.utils.book_new();
		XLSX.utils.book_append_sheet(wb, ws, '日記');
		XLSX.writeFile(wb, `apex-diary-${todayStr()}.xlsx`);
	}

	function exportPDF() {
		const list = getEntriesToExport();
		const today = fmtDate(todayStr());
		const escHtml = (s: string) =>
			s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
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
  @media print { body { padding: 20px; } }
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
  ${e.timestamp ? `<p class="ts">${escHtml(e.timestamp)}</p>` : ''}
</div>`
	)
	.join('')}
</body>
</html>`;
		const win = window.open('', '_blank');
		if (win) {
			win.document.write(html);
			win.document.close();
			setTimeout(() => win.print(), 400);
		}
	}

	function handleExport(type: 'pdf' | 'excel' | 'txt' | 'md') {
		showExportMenu = false;
		if (type === 'pdf') exportPDF();
		else if (type === 'excel') exportExcel();
		else if (type === 'txt') exportTxt();
		else exportMd();
	}

	// ── 写真 ──
	function photoUrl(fileId: string): string {
		return `https://drive.google.com/uc?export=view&id=${fileId}`;
	}

	async function uploadPhotos(files: File[]): Promise<string[]> {
		const fileIds: string[] = [];
		for (const file of files) {
			const formData = new FormData();
			formData.append('photo', file);
			const res = await fetch(`${API_BASE}/photo`, { method: 'POST', body: formData });
			const data = await res.json();
			if (data.success && data.file_id) {
				fileIds.push(data.file_id);
			} else {
				throw new Error(data.message || '写真のアップロードに失敗しました');
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

	function handlePhotoInput(e: Event) {
		const input = e.target as HTMLInputElement;
		const files = Array.from(input.files ?? []);
		photoFiles = [...photoFiles, ...files].slice(0, 5);
		input.value = '';
	}

	function handleEditPhotoInput(e: Event) {
		const input = e.target as HTMLInputElement;
		const files = Array.from(input.files ?? []);
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
												class="action-btn edit-btn"
												title="編集"
												on:click|stopPropagation={() => startEdit(entry)}
											>✏️</button>
											<button
												class="action-btn delete-btn"
												title="削除"
												on:click|stopPropagation={() => deleteEntries([entry.row_index])}
											>🗑️</button>
										</div>
									</div>

									{#if entry.nikki}
										<p class="entry-nikki">{entry.nikki}</p>
									{/if}

									{#if entry.rank1 || entry.rank2}
										<div class="entry-ranks">
											{#if entry.rank1}
												<span class="rank-badge"
													>{entry.rank1}{entry.point1 ? ' ' + entry.point1 : ''}</span
												>
											{/if}
											{#if entry.rank2}
												<span class="rank-badge"
													>{entry.rank2}{entry.point2 ? ' ' + entry.point2 : ''}</span
												>
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
		border: 1.5px solid #f3f4f6;
		border-radius: 14px;
		padding: 1rem 1.125rem;
		transition: border-color 0.15s, box-shadow 0.15s;
		display: flex;
		gap: 0.75rem;
		align-items: flex-start;
	}

	.entry-card:hover {
		border-color: #bae6fd;
		box-shadow: 0 2px 8px rgba(2, 132, 199, 0.08);
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

	/* 入力時と参照時のスタイルを一致させる */
	.entry-nikki {
		color: #1f2937;
		font-size: 0.95rem;
		line-height: 1.75;
		white-space: pre-wrap;
		word-break: break-word;
		font-family: inherit;
	}

	.entry-ranks {
		display: flex;
		gap: 0.5rem;
		flex-wrap: wrap;
	}

	.rank-badge {
		background: #f3f4f6;
		border: 1px solid #e5e7eb;
		border-radius: 7px;
		color: #374151;
		font-size: 0.8rem;
		font-weight: 600;
		padding: 0.2rem 0.65rem;
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
</style>
