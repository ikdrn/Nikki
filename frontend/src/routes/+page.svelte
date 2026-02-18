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

	onMount(() => {
		// デフォルト日付を今日に設定
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
</script>

<main>
	<header>
		<h1>🎮 Apex成長日記</h1>
		<p class="subtitle">ひらふうの記録帳</p>
	</header>

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
		margin-bottom: 2rem;
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
