<script lang="ts">
	import { onMount } from 'svelte';

	const RANKS = [
		'',
		'ブロンズ',
		'シルバー',
		'ゴールド',
		'プラチナ',
		'ダイヤ',
		'マスター',
		'プレデター'
	];

	// ---- 記録フォーム ----
	let date = '';
	let name = '';
	let nikki = '';
	let rank1 = '';
	let point1: number | null = null;
	let rank2 = '';
	let point2: number | null = null;

	let saving = false;
	let message = '';
	let messageType: 'success' | 'error' | '' = '';

	// ---- タブ ----
	let activeTab: 'write' | 'history' = 'write';

	// ---- 履歴 ----
	interface Entry {
		row_name: string;
		date: string;
		name: string;
		nikki: string;
		rank1: string;
		point1: string;
		rank2: string;
		point2: string;
		timestamp: string;
	}

	let entries: Entry[] = [];
	let loadingHistory = false;
	let historyError = '';
	let searchName = '';
	let searchDate = '';
	let sortOrder: 'newest' | 'oldest' = 'newest';

	onMount(() => {
		const today = new Date();
		const y = today.getFullYear();
		const m = String(today.getMonth() + 1).padStart(2, '0');
		const d = String(today.getDate()).padStart(2, '0');
		date = `${y}-${m}-${d}`;
	});

	const API_BASE = import.meta.env.VITE_API_BASE_URL || '/api';

	async function save() {
		if (!date) {
			message = '日付を入力してください';
			messageType = 'error';
			return;
		}

		saving = true;
		message = '';
		messageType = '';

		const payload: Record<string, unknown> = {
			date,
			name,
			nikki,
			rank1,
			rank2
		};
		if (point1 !== null && point1 !== undefined && String(point1) !== '') {
			payload.point1 = Number(point1);
		}
		if (point2 !== null && point2 !== undefined && String(point2) !== '') {
			payload.point2 = Number(point2);
		}

		try {
			const res = await fetch(`${API_BASE}/nikki`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(payload)
			});
			const data = await res.json();
			if (data.success) {
				message = '保存しました';
				messageType = 'success';
				// フォームリセット
				name = '';
				nikki = '';
				rank1 = '';
				point1 = null;
				rank2 = '';
				point2 = null;
				// 履歴キャッシュをリセット（次回タブ切り替え時に再取得）
				entries = [];
			} else {
				message = data.message || '保存失敗';
				messageType = 'error';
			}
		} catch (e) {
			message = `通信エラー: ${e instanceof Error ? e.message : String(e)}`;
			messageType = 'error';
		} finally {
			saving = false;
		}
	}

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

	function formatDate(d: string): string {
		if (!d || d.length !== 10) return d;
		const [y, m, day] = d.split('-');
		return `${y}年${m}月${day}日`;
	}

	$: filteredEntries = entries
		.filter((e) => {
			if (searchName && !e.name.toLowerCase().includes(searchName.toLowerCase())) return false;
			if (searchDate && e.date !== searchDate) return false;
			return true;
		})
		.sort((a, b) => {
			const cmp = a.date.localeCompare(b.date) || a.timestamp.localeCompare(b.timestamp);
			return sortOrder === 'newest' ? -cmp : cmp;
		});
</script>

<main>
	<header>
		<h1>🎮 Apex成長日記</h1>
		<p class="subtitle">ひらふうの記録帳</p>
	</header>

	<!-- タブ -->
	<div class="tabs">
		<button
			class="tab-btn"
			class:active={activeTab === 'write'}
			on:click={() => switchTab('write')}
		>
			📝 記録する
		</button>
		<button
			class="tab-btn"
			class:active={activeTab === 'history'}
			on:click={() => switchTab('history')}
		>
			📋 履歴
		</button>
	</div>

	<!-- 記録フォーム -->
	{#if activeTab === 'write'}
		<form on:submit|preventDefault={save}>
			<!-- 日付 + 名前 -->
			<div class="row date-name-row">
				<div class="field date-field">
					<label for="date">日付</label>
					<input id="date" type="date" bind:value={date} required />
				</div>
				<div class="field name-field">
					<label for="name">名前 <span class="optional">(任意)</span></label>
					<input
						id="name"
						type="text"
						bind:value={name}
						placeholder="例: night"
						maxlength="50"
					/>
				</div>
			</div>

			<!-- 日記 -->
			<div class="field">
				<label for="nikki">日記</label>
				<textarea
					id="nikki"
					bind:value={nikki}
					placeholder="今日のApexを振り返ろう..."
					rows="5"
				></textarea>
			</div>

			<!-- ランク1 -->
			<div class="row rank-row">
				<div class="field rank-field">
					<label for="rank1">ランク1 <span class="optional">(任意)</span></label>
					<select id="rank1" bind:value={rank1}>
						{#each RANKS as r}
							<option value={r}>{r === '' ? '-- 未選択 --' : r}</option>
						{/each}
					</select>
				</div>
				<div class="field rp-field">
					<label for="point1">RP1 <span class="optional">(任意)</span></label>
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

			<!-- ランク2 -->
			<div class="row rank-row">
				<div class="field rank-field">
					<label for="rank2">ランク2 <span class="optional">(任意)</span></label>
					<select id="rank2" bind:value={rank2}>
						{#each RANKS as r}
							<option value={r}>{r === '' ? '-- 未選択 --' : r}</option>
						{/each}
					</select>
				</div>
				<div class="field rp-field">
					<label for="point2">RP2 <span class="optional">(任意)</span></label>
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

			<!-- メッセージ -->
			{#if message}
				<div class="message {messageType}">
					{message}
				</div>
			{/if}

			<!-- 保存ボタン -->
			<button type="submit" class="save-btn" disabled={saving}>
				{saving ? '保存中...' : '💾 保存'}
			</button>
		</form>
	{/if}

	<!-- 履歴タブ -->
	{#if activeTab === 'history'}
		<div class="history">
			<!-- 検索・ソートコントロール -->
			<div class="history-controls">
				<div class="field">
					<label for="search-name">名前で検索</label>
					<input
						id="search-name"
						type="text"
						bind:value={searchName}
						placeholder="例: night"
					/>
				</div>
				<div class="field">
					<label for="search-date">日付で絞り込み</label>
					<input id="search-date" type="date" bind:value={searchDate} />
				</div>
				<div class="sort-row">
					<button
						class="sort-btn"
						class:active={sortOrder === 'newest'}
						on:click={() => (sortOrder = 'newest')}
					>
						新しい順
					</button>
					<button
						class="sort-btn"
						class:active={sortOrder === 'oldest'}
						on:click={() => (sortOrder = 'oldest')}
					>
						古い順
					</button>
					<button class="reload-btn" on:click={loadHistory} disabled={loadingHistory}>
						{loadingHistory ? '読込中...' : '🔄'}
					</button>
				</div>
			</div>

			{#if loadingHistory}
				<p class="loading-msg">読み込み中...</p>
			{:else if historyError}
				<div class="message error">エラー: {historyError}</div>
			{:else if filteredEntries.length === 0}
				<p class="empty-msg">
					{entries.length === 0 ? 'まだ記録がありません' : '条件に一致する記録がありません'}
				</p>
			{:else}
				<p class="entry-count">{filteredEntries.length} 件</p>
				<div class="entry-list">
					{#each filteredEntries as entry}
						<div class="entry-card">
							<div class="entry-header">
								<span class="entry-date">{formatDate(entry.date)}</span>
								{#if entry.name}
									<span class="entry-name">{entry.name}</span>
								{/if}
							</div>

							{#if entry.nikki}
								<p class="entry-nikki">{entry.nikki}</p>
							{/if}

							{#if entry.rank1 || entry.rank2}
								<div class="entry-ranks">
									{#if entry.rank1}
										<span class="rank-badge">
											{entry.rank1}{entry.point1 ? ' ' + entry.point1 : ''}
										</span>
									{/if}
									{#if entry.rank2}
										<span class="rank-badge">
											{entry.rank2}{entry.point2 ? ' ' + entry.point2 : ''}
										</span>
									{/if}
								</div>
							{/if}

							{#if entry.timestamp}
								<p class="entry-timestamp">記録: {entry.timestamp}</p>
							{/if}
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</main>

<style>
	:global(*, *::before, *::after) {
		box-sizing: border-box;
		margin: 0;
		padding: 0;
	}

	:global(body) {
		font-family: 'Hiragino Sans', 'Hiragino Kaku Gothic ProN', 'Noto Sans JP', sans-serif;
		background: #0f0f1a;
		color: #e0e0f0;
		min-height: 100vh;
	}

	main {
		max-width: 640px;
		margin: 0 auto;
		padding: 1.5rem 1rem 4rem;
	}

	header {
		text-align: center;
		margin-bottom: 1.5rem;
	}

	h1 {
		font-size: 1.8rem;
		color: #a78bfa;
		font-weight: 700;
		letter-spacing: 0.05em;
	}

	.subtitle {
		font-size: 0.85rem;
		color: #6b7280;
		margin-top: 0.3rem;
	}

	/* ---- タブ ---- */
	.tabs {
		display: flex;
		gap: 0.5rem;
		margin-bottom: 1.5rem;
		border-bottom: 1px solid #374151;
		padding-bottom: 0;
	}

	.tab-btn {
		background: none;
		border: none;
		border-bottom: 2px solid transparent;
		color: #6b7280;
		cursor: pointer;
		font-family: inherit;
		font-size: 0.95rem;
		font-weight: 600;
		padding: 0.5rem 1rem 0.6rem;
		transition: color 0.2s, border-color 0.2s;
		margin-bottom: -1px;
	}

	.tab-btn:hover {
		color: #a78bfa;
	}

	.tab-btn.active {
		border-bottom-color: #7c3aed;
		color: #a78bfa;
	}

	/* ---- フォーム ---- */
	form {
		display: flex;
		flex-direction: column;
		gap: 1.2rem;
	}

	.row {
		display: flex;
		gap: 0.8rem;
	}

	.date-name-row .date-field {
		flex: 0 0 160px;
	}

	.date-name-row .name-field {
		flex: 1;
	}

	.rank-row .rank-field {
		flex: 1;
	}

	.rank-row .rp-field {
		flex: 0 0 130px;
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
	}

	label {
		font-size: 0.85rem;
		color: #9ca3af;
		font-weight: 500;
	}

	.optional {
		font-size: 0.75rem;
		color: #6b7280;
	}

	input[type='text'],
	input[type='date'],
	input[type='number'],
	select,
	textarea {
		background: #1e1e2e;
		border: 1px solid #374151;
		border-radius: 8px;
		color: #e0e0f0;
		font-size: 1rem;
		padding: 0.6rem 0.8rem;
		width: 100%;
		transition: border-color 0.2s;
		font-family: inherit;
	}

	input[type='text']:focus,
	input[type='date']:focus,
	input[type='number']:focus,
	select:focus,
	textarea:focus {
		outline: none;
		border-color: #7c3aed;
		box-shadow: 0 0 0 2px rgba(124, 58, 237, 0.2);
	}

	input[type='date']::-webkit-calendar-picker-indicator {
		filter: invert(1);
		opacity: 0.6;
		cursor: pointer;
	}

	select {
		cursor: pointer;
		appearance: none;
		background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' fill='none' viewBox='0 0 20 20'%3E%3Cpath stroke='%236b7280' stroke-linecap='round' stroke-linejoin='round' stroke-width='1.5' d='M6 8l4 4 4-4'/%3E%3C/svg%3E");
		background-repeat: no-repeat;
		background-position: right 0.6rem center;
		background-size: 1.2em;
		padding-right: 2rem;
	}

	select option {
		background: #1e1e2e;
		color: #e0e0f0;
	}

	textarea {
		resize: vertical;
		min-height: 120px;
		line-height: 1.6;
	}

	.message {
		padding: 0.75rem 1rem;
		border-radius: 8px;
		font-size: 0.95rem;
		font-weight: 500;
		text-align: center;
	}

	.message.success {
		background: rgba(16, 185, 129, 0.15);
		border: 1px solid #059669;
		color: #34d399;
	}

	.message.error {
		background: rgba(239, 68, 68, 0.15);
		border: 1px solid #dc2626;
		color: #f87171;
	}

	.save-btn {
		background: #7c3aed;
		border: none;
		border-radius: 10px;
		color: #fff;
		cursor: pointer;
		font-family: inherit;
		font-size: 1.05rem;
		font-weight: 700;
		letter-spacing: 0.05em;
		padding: 0.85rem;
		transition: background 0.2s, transform 0.1s;
		width: 100%;
	}

	.save-btn:hover:not(:disabled) {
		background: #6d28d9;
	}

	.save-btn:active:not(:disabled) {
		transform: scale(0.98);
	}

	.save-btn:disabled {
		background: #4b5563;
		cursor: not-allowed;
	}

	/* ---- 履歴 ---- */
	.history {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.history-controls {
		display: flex;
		flex-direction: column;
		gap: 0.8rem;
		background: #1a1a2e;
		border: 1px solid #374151;
		border-radius: 10px;
		padding: 1rem;
	}

	.sort-row {
		display: flex;
		gap: 0.5rem;
		align-items: center;
	}

	.sort-btn {
		background: #1e1e2e;
		border: 1px solid #374151;
		border-radius: 6px;
		color: #9ca3af;
		cursor: pointer;
		font-family: inherit;
		font-size: 0.85rem;
		font-weight: 600;
		padding: 0.4rem 0.9rem;
		transition: all 0.2s;
	}

	.sort-btn.active {
		background: #4c1d95;
		border-color: #7c3aed;
		color: #c4b5fd;
	}

	.sort-btn:not(.active):hover {
		border-color: #6b7280;
		color: #e0e0f0;
	}

	.reload-btn {
		background: none;
		border: 1px solid #374151;
		border-radius: 6px;
		color: #6b7280;
		cursor: pointer;
		font-size: 1rem;
		margin-left: auto;
		padding: 0.4rem 0.7rem;
		transition: color 0.2s, border-color 0.2s;
	}

	.reload-btn:hover:not(:disabled) {
		border-color: #6b7280;
		color: #e0e0f0;
	}

	.reload-btn:disabled {
		cursor: not-allowed;
		opacity: 0.5;
	}

	.loading-msg,
	.empty-msg {
		color: #6b7280;
		font-size: 0.95rem;
		text-align: center;
		padding: 2rem 0;
	}

	.entry-count {
		color: #6b7280;
		font-size: 0.8rem;
		text-align: right;
	}

	.entry-list {
		display: flex;
		flex-direction: column;
		gap: 0.8rem;
	}

	.entry-card {
		background: #1a1a2e;
		border: 1px solid #374151;
		border-radius: 10px;
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
		padding: 1rem;
		transition: border-color 0.2s;
	}

	.entry-card:hover {
		border-color: #4c1d95;
	}

	.entry-header {
		display: flex;
		align-items: center;
		gap: 0.7rem;
		flex-wrap: wrap;
	}

	.entry-date {
		color: #a78bfa;
		font-size: 0.95rem;
		font-weight: 700;
	}

	.entry-name {
		background: #2d1b69;
		border-radius: 4px;
		color: #c4b5fd;
		font-size: 0.8rem;
		font-weight: 600;
		padding: 0.15rem 0.5rem;
	}

	.entry-nikki {
		color: #d1d5db;
		font-size: 0.95rem;
		line-height: 1.6;
		white-space: pre-wrap;
		word-break: break-word;
	}

	.entry-ranks {
		display: flex;
		gap: 0.5rem;
		flex-wrap: wrap;
	}

	.rank-badge {
		background: #1e3a5f;
		border: 1px solid #1e40af;
		border-radius: 6px;
		color: #93c5fd;
		font-size: 0.85rem;
		font-weight: 600;
		padding: 0.25rem 0.7rem;
	}

	.entry-timestamp {
		color: #4b5563;
		font-size: 0.75rem;
		text-align: right;
	}

	@media (max-width: 480px) {
		h1 {
			font-size: 1.5rem;
		}

		.date-name-row {
			flex-direction: column;
		}

		.date-name-row .date-field {
			flex: unset;
		}

		.rank-row .rp-field {
			flex: 0 0 110px;
		}
	}
</style>
