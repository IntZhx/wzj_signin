(() => {
	const $id = (id) => document.getElementById(id);

	const HISTORY_KEY = "wzj.frontendHistory.v2";
	const HISTORY_MAX = 18;
	const SETTINGS_KEY = "wzj.settings.v1";
	const EVENTS_KEY = "wzj.events.v1";
	const POLL_KEY = "wzj.poll.v1";

	const HELP_TEXT =
		"快速开始\n" +
		"1. 设置默认邮箱（用于通知）\n" +
		"2. 设置 GPS 标签（标签名 + 坐标）\n" +
		"3. 提交页：选择 GPS 标签 + 填 OpenID 并提交\n" +
		"4. 保持页面打开，出现二维码签到提醒会弹窗\n" +
		"5. 历史页可查看记录并再次打开二维码\n\n" +
		"注意\n" +
		"- 主页统计基于本机浏览器历史，不会同步服务器\n" +
		"- 二维码签到（扫码）与 GPS 签到互不影响\n" +
		"- 二维码弹窗可能被浏览器拦截，可在历史页再次打开\n";

	// ===== modal =====
	const modal = $id("modal");
	const modalBody = $id("modalBody");
	const modalCloseBtn = $id("modalCloseBtn");
	const modalOkBtn = $id("modalOkBtn");
	const modalActionBtn = $id("modalActionBtn");

	function openModal(message) {
		if (!modal || !modalBody) {
			alert(message || "");
			return;
		}
		modalBody.textContent = message || "";
		if (modalActionBtn) {
			modalActionBtn.style.display = "none";
			modalActionBtn.onclick = null;
		}
		modal.setAttribute("aria-hidden", "false");
	}

	function closeModal() {
		if (!modal) return;
		modal.setAttribute("aria-hidden", "true");
	}

	if (modal) {
		modal.addEventListener("click", (e) => {
			if (e.target === modal) closeModal();
		});
	}
	if (modalCloseBtn) modalCloseBtn.addEventListener("click", closeModal);
	if (modalOkBtn) modalOkBtn.addEventListener("click", closeModal);
	document.addEventListener("keydown", (e) => {
		if (e.key === "Escape") closeModal();
	});

	// ===== helpers =====
	async function safeReadJson(response) {
		try {
			return await response.json();
		} catch {
			return null;
		}
	}

	function normalizeLocation(input) {
		return String(input || "")
			.trim()
			.replaceAll("，", ",")
			.replace(/\s+/g, "");
	}

	function loadSettings() {
		try {
			const raw = localStorage.getItem(SETTINGS_KEY);
			if (!raw) return { defaultEmail: "", gpsLabels: [] };
			const parsed = JSON.parse(raw);
			return {
				defaultEmail: typeof parsed.defaultEmail === "string" ? parsed.defaultEmail : "",
				gpsLabels: Array.isArray(parsed.gpsLabels) ? parsed.gpsLabels : [],
			};
		} catch {
			return { defaultEmail: "", gpsLabels: [] };
		}
	}

	function saveSettings(settings) {
		try {
			localStorage.setItem(
				SETTINGS_KEY,
				JSON.stringify(settings || { defaultEmail: "", gpsLabels: [] })
			);
		} catch {
			// ignore
		}
	}

	async function syncFrontendSettingsFromServer() {
		try {
			const resp = await fetch("/api/frontendsettings", { method: "GET", cache: "no-store" });
			if (!resp.ok) return false;
			const data = await safeReadJson(resp);
			if (!data) return false;

			const merged = loadSettings();
			if (typeof data.defaultEmail === "string") merged.defaultEmail = data.defaultEmail;
			if (Array.isArray(data.gpsLabels)) {
				merged.gpsLabels = data.gpsLabels
					.map((x) => ({
						label: String(x && x.label ? x.label : "").trim(),
						location: String(x && x.location ? x.location : "").trim(),
					}))
					.filter((x) => x.label && x.location);
			}
			saveSettings(merged);
			return true;
		} catch {
			return false;
		}
	}

	async function saveFrontendSettingsToServer(settings) {
		try {
			const payload = {
				defaultEmail: String((settings && settings.defaultEmail) || ""),
				gpsLabels: Array.isArray(settings && settings.gpsLabels)
					? settings.gpsLabels.map((x) => ({
							label: String(x && x.label ? x.label : "").trim(),
							location: String(x && x.location ? x.location : "").trim(),
						}))
					: [],
			};
			await fetch("/api/frontendsettings", {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify(payload),
			});
		} catch {
			// ignore
		}
	}

	function loadHistory() {
		try {
			const raw = localStorage.getItem(HISTORY_KEY);
			if (!raw) return { emailHistory: [], gpsHistory: [], lastOpenId: "" };
			const parsed = JSON.parse(raw);
			return {
				emailHistory: Array.isArray(parsed.emailHistory) ? parsed.emailHistory : [],
				gpsHistory: Array.isArray(parsed.gpsHistory) ? parsed.gpsHistory : [],
				lastOpenId: typeof parsed.lastOpenId === "string" ? parsed.lastOpenId : "",
			};
		} catch {
			return { emailHistory: [], gpsHistory: [], lastOpenId: "" };
		}
	}

	function saveHistory(history) {
		try {
			localStorage.setItem(
				HISTORY_KEY,
				JSON.stringify(history || { emailHistory: [], gpsHistory: [], lastOpenId: "" })
			);
		} catch {
			// ignore
		}
	}

	function rememberEmail(email) {
		const val = String(email || "").trim();
		if (!val) return;
		const history = loadHistory();
		history.emailHistory = [
			val,
			...history.emailHistory.filter((x) => String(x).trim() !== val),
		].slice(0, HISTORY_MAX);
		saveHistory(history);
	}

	function rememberLastOpenId(openId) {
		const val = String(openId || "").trim();
		if (!val) return;
		const history = loadHistory();
		history.lastOpenId = val;
		saveHistory(history);
	}

	function loadEvents() {
		try {
			const raw = localStorage.getItem(EVENTS_KEY);
			if (!raw) return [];
			const parsed = JSON.parse(raw);
			return Array.isArray(parsed) ? parsed : [];
		} catch {
			return [];
		}
	}

	function saveEvents(events) {
		try {
			localStorage.setItem(EVENTS_KEY, JSON.stringify((events || []).slice(0, 80)));
		} catch {
			// ignore
		}
	}

	function addEvent(evt) {
		const events = loadEvents();
		events.unshift({ ...(evt || {}), ts: Date.now() });
		saveEvents(events);
	}

	function formatTime(ts) {
		const d = new Date(ts);
		return d.toLocaleString("zh-CN", { hour12: false });
	}

	function toLocalDateKey(ts) {
		const d = new Date(ts);
		const y = d.getFullYear();
		const m = String(d.getMonth() + 1).padStart(2, "0");
		const day = String(d.getDate()).padStart(2, "0");
		return `${y}-${m}-${day}`;
	}

	function parseOpenIdFromText(text) {
		const raw = String(text || "").trim();
		if (!raw) return "";

		if (/^[0-9a-fA-F]{32}$/.test(raw)) return raw;

		try {
			const u = new URL(raw);
			const openId = u.searchParams.get("openid");
			if (openId && /^[0-9a-fA-F]{32}$/.test(openId)) return openId;
		} catch {
			// ignore
		}

		const m = raw.match(/[?&]openid=([0-9a-fA-F]{32})/i);
		if (m && m[1]) return m[1];

		const any = raw.match(/([0-9a-fA-F]{32})/);
		if (any && any[1]) return any[1];

		return raw;
	}

	// ===== polling multi-openId =====
	let pendingQrTimer = null;
	let monitoredOpenIds = [];
	let monitoredIndex = 0;
	let pendingQrLastShownByOpenId = Object.create(null);

	const monitorCount = $id("monitorCount");
	const pollHint = $id("pollHint");

	async function refreshMonitoredOpenIds() {
		try {
			const resp = await fetch("/openids", { method: "GET", cache: "no-store" });
			if (!resp.ok) return [];
			const data = await safeReadJson(resp);
			const list = Array.isArray(data && data.openIds)
				? data.openIds.map((x) => String(x || "").trim()).filter(Boolean)
				: [];
			monitoredOpenIds = list;
			if (monitorCount) monitorCount.textContent = String(list.length);
			return list;
		} catch {
			return [];
		}
	}

	function savePollState(enabled) {
		try {
			localStorage.setItem(POLL_KEY, JSON.stringify({ enabled: !!enabled }));
		} catch {
			// ignore
		}
	}

	function loadPollState() {
		try {
			const raw = localStorage.getItem(POLL_KEY);
			if (!raw) return { enabled: false };
			const p = JSON.parse(raw);
			return { enabled: !!(p && p.enabled) };
		} catch {
			return { enabled: false };
		}
	}

	function setPollHint() {
		if (!pollHint) return;
		const st = loadPollState();
		if (!st.enabled) {
			pollHint.textContent = "当前未轮询二维码。";
			return;
		}
		const n = Array.isArray(monitoredOpenIds) ? monitoredOpenIds.length : 0;
		pollHint.textContent = `正在轮询二维码：${n} 个 OpenID`;
	}

	function stopPendingQrPoll() {
		if (pendingQrTimer) {
			clearInterval(pendingQrTimer);
			pendingQrTimer = null;
		}
		savePollState(false);
		setPollHint();
	}

	async function startPendingQrPollAll() {
		stopPendingQrPoll();
		savePollState(true);

		const list = await refreshMonitoredOpenIds();
		if (!list || list.length === 0) {
			savePollState(false);
			setPollHint();
			return;
		}
		setPollHint();

		pendingQrTimer = setInterval(async () => {
			const st = loadPollState();
			if (!st.enabled) return;

			if (monitoredIndex % 15 === 0) {
				await refreshMonitoredOpenIds();
				setPollHint();
			}

			if (!monitoredOpenIds || monitoredOpenIds.length === 0) return;
			const openId = monitoredOpenIds[monitoredIndex % monitoredOpenIds.length];
			monitoredIndex++;
			if (!openId) return;

			try {
				// 1) QR pending
				{
					const resp = await fetch("/pendingqr/" + encodeURIComponent(openId), {
						method: "GET",
						cache: "no-store",
					});
					if (resp.ok) {
						const data = await safeReadJson(resp);
						if (data) {
							const url = data.url ? String(data.url) : "";
							const signId = data.signId ? String(data.signId) : "";
							if (url) {
								const last = pendingQrLastShownByOpenId[openId] || "";
								if (url !== last) {
									pendingQrLastShownByOpenId[openId] = url;
									addEvent({ type: "qr", url, signId, openId });

									// 尝试自动打开新标签页（可能会被浏览器拦截）
									let opened = false;
									try {
										const w = window.open(url, "_blank", "noopener,noreferrer");
										opened = !!w;
									} catch {
										opened = false;
									}

									if (modalActionBtn) {
										modalActionBtn.style.display = "inline-flex";
										modalActionBtn.textContent = "打开二维码页面";
										modalActionBtn.onclick = () =>
											window.open(url, "_blank", "noopener,noreferrer");
									}

									const tip =
										"检测到二维码签到，需要手动用微信扫一扫完成。\n" +
										(openId ? "openid：" + openId + "\n" : "") +
										(signId ? "signId：" + signId + "\n" : "") +
										"点击『打开二维码页面』即可查看二维码。\n" +
										(!opened ? "（若未自动打开新页面，可能被浏览器拦截弹窗）" : "");

									openModal(tip);
								}
							}
						}
					}
				}

				// 2) Sign-in pending events (GPS/normal)
				{
					const resp = await fetch("/pendingevent/" + encodeURIComponent(openId), {
						method: "GET",
						cache: "no-store",
					});
					if (resp.ok) {
						const data = await safeReadJson(resp);
						if (data && data.ok && data.type) {
							addEvent({
								type: String(data.type),
								mode: data.mode ? String(data.mode) : "",
								openId: data.openId ? String(data.openId) : openId,
								courseId: data.courseId,
								signId: data.signId,
								courseName: data.courseName ? String(data.courseName) : "",
								studentRank: data.studentRank,
								signRank: data.signRank,
							});
						}
					}
				}

				// 如果在历史页，顺便刷新
				if ($id("eventList")) renderEvents();
			} catch {
				// ignore
			}
		}, 1200);
	}

	// ===== render: home =====
	function renderHomeStats() {
		const statTotalQr = $id("statTotalQr");
		const statLastQr = $id("statLastQr");
		const statTotalSubmit = $id("statTotalSubmit");
		const statLastSubmit = $id("statLastSubmit");
		const statChart = $id("statChart");
		const statChartLabels = $id("statChartLabels");
		if (!statTotalQr || !statChart || !statChartLabels) return;

		const events = loadEvents();
		const qrEvents = events.filter((e) => e && e.type === "qr" && typeof e.ts === "number");
		const submitEvents = events.filter((e) => e && e.type === "submit" && typeof e.ts === "number");

		statTotalQr.textContent = String(qrEvents.length);
		statTotalSubmit.textContent = String(submitEvents.length);
		statLastQr.textContent = qrEvents.length ? "最近一次：" + formatTime(qrEvents[0].ts) : "最近一次：—";
		statLastSubmit.textContent = submitEvents.length ? "最近一次：" + formatTime(submitEvents[0].ts) : "最近一次：—";

		const days = [];
		const now = new Date();
		now.setHours(0, 0, 0, 0);
		for (let i = 6; i >= 0; i--) days.push(new Date(now.getTime() - i * 86400000));

		const dayCount = new Map();
		for (const e of qrEvents) {
			const k = toLocalDateKey(e.ts);
			dayCount.set(k, (dayCount.get(k) || 0) + 1);
		}

		const counts = days.map((d) => dayCount.get(toLocalDateKey(d.getTime())) || 0);
		const max = Math.max(1, ...counts);
		const sum7 = counts.reduce((a, b) => a + b, 0);

		statChart.innerHTML = "";
		statChartLabels.innerHTML = "";

		if (sum7 === 0) {
			const empty = document.createElement("div");
			empty.className = "chartEmpty";
			empty.textContent = "近 7 天暂无签到提醒记录";
			statChart.appendChild(empty);
		}

		counts.forEach((c, idx) => {
			const bar = document.createElement("div");
			const h = c === 0 ? 8 : Math.round((c / max) * 78);
			bar.style.height = `${h}px`;
			bar.style.borderRadius = "12px";
			bar.style.border = "1px solid var(--border)";
			bar.style.background = c === 0 ? "rgba(234, 210, 215, 0.35)" : "rgba(200, 154, 166, 0.22)";
			bar.style.opacity = c === 0 ? "0.55" : "1";
			bar.title = `${toLocalDateKey(days[idx].getTime())}：${c} 次`;
			statChart.appendChild(bar);

			const lab = document.createElement("div");
			const mm = String(days[idx].getMonth() + 1).padStart(2, "0");
			const dd = String(days[idx].getDate()).padStart(2, "0");
			lab.textContent = `${mm}/${dd}`;
			statChartLabels.appendChild(lab);
		});
	}

	function setHelpText() {
		const noticeBox = $id("noticeBox");
		const quickNoticeBtn = $id("quickNoticeBtn");
		if (noticeBox) {
			noticeBox.innerHTML =
				"1）到【设置】保存默认邮箱（提交时会自动使用）<br />" +
				"2）到【设置】添加 GPS 标签（标签名 + 经纬度 lng,lat）<br />" +
				"3）到【提交】粘贴 OpenID/链接，选择 GPS 标签，点击提交<br />" +
				"说明：二维码签到（扫码）与 GPS 签到互不影响<br />" +
				"4）提交后会加入监控池；保持页面打开即可接收二维码弹窗<br />" +
				"5）到【历史】可开始/停止轮询，查看记录，并可再次打开二维码<br /><br />" +
				"提示：二维码弹窗可能被浏览器拦截，拦截后可去【历史】再次打开。";
		}
		if (quickNoticeBtn) quickNoticeBtn.textContent = "使用说明";
	}

	// ===== render: selectors/settings =====
	function renderSelectors() {
		const emailHistoryDatalist = $id("emailHistory");
		const gpsLabelSelect = $id("gpsLabelSelect");

		if (emailHistoryDatalist) {
			const history = loadHistory();
			emailHistoryDatalist.innerHTML = "";
			for (const email of history.emailHistory) {
				const opt = document.createElement("option");
				opt.value = String(email);
				emailHistoryDatalist.appendChild(opt);
			}
		}

		if (gpsLabelSelect) {
			const settings = loadSettings();
			gpsLabelSelect.innerHTML = "";
			const p2 = document.createElement("option");
			p2.value = "";
			p2.textContent = "— 选择标签 —";
			gpsLabelSelect.appendChild(p2);

			settings.gpsLabels.forEach((it, idx) => {
				const label = it && it.label ? String(it.label) : "";
				const location = it && it.location ? String(it.location) : "";
				if (!label || !location) return;
				const opt = document.createElement("option");
				opt.value = String(idx);
				opt.textContent = `${label}  ·  ${location}`;
				gpsLabelSelect.appendChild(opt);
			});
		}
	}

	function renderSettings() {
		const defaultEmailInput = $id("defaultEmail");
		const gpsLabelList = $id("gpsLabelList");
		if (!defaultEmailInput && !gpsLabelList) return;

		const settings = loadSettings();
		if (defaultEmailInput) defaultEmailInput.value = settings.defaultEmail || "";

		if (gpsLabelList) {
			gpsLabelList.innerHTML = "";
			settings.gpsLabels.forEach((it) => {
				const label = it && it.label ? String(it.label) : "";
				const location = it && it.location ? String(it.location) : "";
				if (!label || !location) return;
				const row = document.createElement("div");
				row.className = "badge";
				row.innerHTML = `<span class="dot"></span><span><strong>${label}</strong> <span class="mono">${location}</span></span>`;
				gpsLabelList.appendChild(row);
			});
		}
	}

	// ===== render: events/history =====
	function renderEvents() {
		const eventList = $id("eventList");
		if (!eventList) return;

		const events = loadEvents();
		eventList.innerHTML = "";
		if (!events.length) {
			const empty = document.createElement("div");
			empty.className = "hint";
			empty.textContent = "暂无历史记录。";
			eventList.appendChild(empty);
			return;
		}

		for (const e of events.slice(0, 40)) {
			const card = document.createElement("div");
			card.className = "card";
			card.style.padding = "14px";
			card.style.background = "rgba(255,255,255,0.76)";

			const when = formatTime(e.ts);

			if (e.type === "qr") {
				const url = String(e.url || "");
				const signId = String(e.signId || "");
				const openId = String(e.openId || "");
				card.innerHTML = `
					<div style="display:flex;justify-content:space-between;gap:10px;align-items:center">
						<div>
							<div style="font-weight:800">二维码签到提醒</div>
							<div class="hint" style="margin-top:4px">${when}${signId ? ` · signId: <span class="mono">${signId}</span>` : ""}</div>
						</div>
						<button class="pill primary" type="button">再次打开</button>
					</div>
					${openId ? `<div class="hint mono" style="margin-top:10px">openid: ${openId}</div>` : ""}
					<div class="hint mono" style="margin-top:10px">${url}</div>
				`;
				card
					.querySelector("button")
					.addEventListener("click", () => window.open(url, "_blank", "noopener,noreferrer"));
			} else if (e.type === "submit") {
				const openId = String(e.openId || "");
				const gpsLabel = String(e.gpsLabel || "");
				const loc = String(e.location || "");
				const title = "提交 OpenID";
				card.innerHTML = `
					<div style="font-weight:800">${title}</div>
					<div class="hint" style="margin-top:4px">${when}</div>
					<div class="hint mono" style="margin-top:10px">openid: ${openId}</div>
					${gpsLabel ? `<div class="hint" style="margin-top:6px">GPS 标签：<strong>${gpsLabel}</strong></div>` : ""}
					${loc ? `<div class="hint mono" style="margin-top:6px">GPS 坐标：${loc}</div>` : ""}
				`;
			} else if (e.type === "signin") {
				const openId = String(e.openId || "");
				const mode = String(e.mode || "");
				const courseName = String(e.courseName || "");
				const courseId = e.courseId != null ? String(e.courseId) : "";
				const signId = e.signId != null ? String(e.signId) : "";
				const title = mode === "gps" ? "GPS 签到成功" : "普通签到成功";
				const rankLine =
					e.studentRank != null && e.signRank != null
						? `签到No.<span class="mono">${String(e.signRank)}</span> · 你是第 <span class="mono">${String(
								e.studentRank
							)}</span> 个`
						: "";
				card.innerHTML = `
					<div style="font-weight:800">${title}</div>
					<div class="hint" style="margin-top:4px">${when}${courseName ? ` · ${courseName}` : ""}</div>
					${openId ? `<div class="hint mono" style="margin-top:10px">openid: ${openId}</div>` : ""}
					${courseId || signId ? `<div class="hint" style="margin-top:6px">C${courseId || "?"} / S${signId || "?"}</div>` : ""}
					${rankLine ? `<div class="hint" style="margin-top:6px">${rankLine}</div>` : ""}
				`;
			} else {
				card.innerHTML = `<div style="font-weight:800">事件</div><div class="hint" style="margin-top:4px">${when}</div>`;
			}

			eventList.appendChild(card);
		}
	}

	// ===== submit page =====
	function wireSubmitPage() {
		const form = $id("submitForm");
		const submitBtn = $id("submitBtn");
		const statusText = $id("statusText");
		const statusDot = $id("statusDot");
		const gpsLabelSelect = $id("gpsLabelSelect");
		if (!form || !submitBtn) return;

		function setStatus(type, text) {
			if (statusText) statusText.textContent = text;
			if (!statusDot) return;
			statusDot.classList.remove("ok", "bad");
			if (type === "ok") statusDot.classList.add("ok");
			if (type === "bad") statusDot.classList.add("bad");
		}

		const openIdField = $id("inputOpenId");
		if (openIdField) {
			openIdField.addEventListener("paste", (e) => {
				const text = (e.clipboardData || window.clipboardData)?.getData("text") || "";
				const parsed = parseOpenIdFromText(text);
				if (parsed && parsed !== String(text || "").trim()) {
					e.preventDefault();
					openIdField.value = parsed;
					openIdField.dispatchEvent(new Event("input", { bubbles: true }));
				}
			});

			openIdField.addEventListener("blur", () => {
				openIdField.value = parseOpenIdFromText(openIdField.value);
			});
		}

		form.addEventListener("submit", async (event) => {
			event.preventDefault();

			const openId = parseOpenIdFromText(openIdField ? openIdField.value : "");
			if (openIdField) openIdField.value = openId;

			const settings = loadSettings();
			const email = String(settings.defaultEmail || "").trim();

			const labelIdx = gpsLabelSelect ? gpsLabelSelect.value : "";
			const labelItem = labelIdx ? settings.gpsLabels[Number(labelIdx)] : null;
			const gpsLabel = labelItem && labelItem.label ? String(labelItem.label) : "";
			const location = labelItem && labelItem.location ? String(labelItem.location) : "";

			if (openId.length !== 32) {
				setStatus("bad", "OpenID 必须为 32 位");
				openModal("OpenID 必须为 32 位，请检查后再提交。");
				return;
			}

			if (!email) {
				setStatus("bad", "请先设置默认邮箱");
				openModal("提交页已隐藏邮箱输入。请先到“设置”页保存默认邮箱，再回来提交。");
				return;
			}

			if (!labelIdx || !gpsLabel || !location) {
				setStatus("bad", "请选择 GPS 标签");
				openModal("请选择一个 GPS 标签（在“设置”页创建后可选）。");
				return;
			}

			const payload = { openId, value: email, location };

			submitBtn.disabled = true;
			submitBtn.textContent = "提交中...";
			setStatus("", "提交中…");

			try {
				const resp = await fetch("/register", {
					method: "POST",
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify(payload),
				});

				const data = await safeReadJson(resp);
				const msg =
					(data && (data.message || data.error) ? String(data.message || data.error) : "") ||
					(resp.ok
						? "提交完成，但服务器未返回明确消息，请检查后端日志。"
						: "提交失败，请检查后端日志。");

				if (resp.ok) {
					setStatus("ok", "已提交");
					rememberEmail(email);
					rememberLastOpenId(openId);
					addEvent({ type: "submit", openId, gpsLabel, location: normalizeLocation(location) });

					await refreshMonitoredOpenIds();
					startPendingQrPollAll();
				} else {
					setStatus("bad", "提交失败");
				}

				openModal(msg);
			} catch (err) {
				setStatus("bad", "网络错误");
				openModal(
					"提交失败，请检查网络或后端日志。错误信息：" +
						(err && err.message ? err.message : String(err))
				);
			} finally {
				submitBtn.disabled = false;
				submitBtn.textContent = "提交";
			}
		});
	}

	// ===== history page buttons =====
	function wireHistoryPage() {
		const startBtn = $id("remindStartPollBtn");
		const stopBtn = $id("remindStopPollBtn");

		if (startBtn) {
			startBtn.addEventListener("click", async () => {
				await startPendingQrPollAll();
				const list =
					monitoredOpenIds && monitoredOpenIds.length
						? monitoredOpenIds
						: await refreshMonitoredOpenIds();
				if (!list || list.length === 0) {
					openModal("监控池为空：请先去“提交”页提交至少一个 OpenID。");
					stopPendingQrPoll();
					return;
				}
				openModal("已开始轮询二维码提醒（多 OpenID 同时生效）。保持页面打开即可。");
			});
		}

		if (stopBtn) {
			stopBtn.addEventListener("click", () => {
				stopPendingQrPoll();
				openModal("已停止轮询二维码提醒。");
			});
		}
	}

	// ===== settings page =====
	function wireSettingsPage() {
		const saveDefaultEmailBtn = $id("saveDefaultEmailBtn");
		const defaultEmailInput = $id("defaultEmail");
		if (saveDefaultEmailBtn && defaultEmailInput) {
			saveDefaultEmailBtn.addEventListener("click", () => {
				const val = String(defaultEmailInput.value || "").trim();
				const settings = loadSettings();
				settings.defaultEmail = val;
				saveSettings(settings);
				saveFrontendSettingsToServer(settings);
				if (val) rememberEmail(val);
				renderAll();
				openModal("默认邮箱已保存。");
			});
		}

		const saveGpsLabelBtn = $id("saveGpsLabelBtn");
		const newGpsLabel = $id("newGpsLabel");
		const newGpsLocation = $id("newGpsLocation");
		if (saveGpsLabelBtn && newGpsLabel && newGpsLocation) {
			saveGpsLabelBtn.addEventListener("click", () => {
				const label = String(newGpsLabel.value || "").trim();
				const location = normalizeLocation(newGpsLocation.value);
				if (!label) return openModal("标签名不能为空。");
				if (!location) return openModal("坐标不能为空。");

				const settings = loadSettings();
				const remaining = settings.gpsLabels.filter(
					(x) => String(x && x.label).trim() !== label
				);
				settings.gpsLabels = [{ label, location }, ...remaining].slice(0, HISTORY_MAX);
				saveSettings(settings);
				saveFrontendSettingsToServer(settings);
				renderAll();
				openModal("GPS 标签已保存。");
			});
		}

		const saveServerConfigBtn = $id("saveServerConfigBtn");
		const normalDelay = $id("normalDelay");
		const mailEnabled = $id("mailEnabled");
		const mailHost = $id("mailHost");
		const mailPort = $id("mailPort");
		const mailUsername = $id("mailUsername");
		const mailFrom = $id("mailFrom");
		const mailPassword = $id("mailPassword");
		const mailPasswordHint = $id("mailPasswordHint");

		async function loadServerConfig() {
			if (!normalDelay || !mailEnabled) return;
			try {
				const resp = await fetch("/api/appconfig", { method: "GET", cache: "no-store" });
				if (!resp.ok) return;
				const data = await safeReadJson(resp);
				if (!data) return;

				normalDelay.value = typeof data.normal_delay === "number" ? String(data.normal_delay) : "";
				if (mailEnabled) {
					mailEnabled.value = data.mail && data.mail.enabled ? "on" : "off";
				}
				if (mailHost) mailHost.value = String((data.mail && data.mail.host) || "");
				if (mailPort) mailPort.value = String((data.mail && data.mail.port) || "");
				if (mailUsername) mailUsername.value = String((data.mail && data.mail.username) || "");
				if (mailFrom) mailFrom.value = String((data.mail && data.mail.from) || "");

				if (mailPasswordHint) {
					mailPasswordHint.textContent = data.passwordSet
						? "已保存（为安全不回显）。留空表示不修改"
						: "未设置。留空表示不修改";
				}
			} catch {
				// ignore
			}
		}

		loadServerConfig();
		// 同步一次本机持久化的前端设置（默认邮箱/GPS 标签），确保清浏览器缓存也能回填
		syncFrontendSettingsFromServer().then((ok) => {
			if (ok) renderAll();
		});

		if (saveServerConfigBtn) {
			saveServerConfigBtn.addEventListener("click", async () => {
				try {
					const delayVal = normalDelay ? Number(normalDelay.value || 0) : 0;
					if (!Number.isFinite(delayVal) || delayVal < 0 || delayVal > 600) {
						openModal("延迟时间范围不合法（0-600 秒）。");
						return;
					}

					const portVal = mailPort && String(mailPort.value).trim() !== "" ? Number(mailPort.value) : 0;
					if (!Number.isFinite(portVal) || portVal < 0 || portVal > 65535) {
						openModal("邮件端口范围不合法（0-65535）。");
						return;
					}

					const payload = {
						normal_delay: delayVal,
						mail: {
							enabled: mailEnabled ? mailEnabled.value === "on" : false,
							host: mailHost ? String(mailHost.value || "").trim() : "",
							port: portVal,
							username: mailUsername ? String(mailUsername.value || "").trim() : "",
							password: mailPassword ? String(mailPassword.value || "") : "",
							from: mailFrom ? String(mailFrom.value || "").trim() : "",
						},
					};

					const resp = await fetch("/api/appconfig", {
						method: "POST",
						headers: { "Content-Type": "application/json" },
						body: JSON.stringify(payload),
					});
					const data = await safeReadJson(resp);
					if (!resp.ok) {
						openModal((data && data.error) || "保存失败。请检查输入。");
						return;
					}

					if (mailPassword) mailPassword.value = "";
					if (mailPasswordHint) {
						mailPasswordHint.textContent = data && data.passwordSet
							? "已保存（为安全不回显）。留空表示不修改"
							: "未设置。留空表示不修改";
					}
					openModal("服务端配置已保存，并已立即生效。\n（密码留空表示不修改已保存的密码）");
				} catch {
					openModal("保存失败：网络或服务异常。");
				}
			});
		}
	}

	function wireHelpButtons() {
		const quickNoticeBtn = $id("quickNoticeBtn");
		const copyNoticeBtn = $id("copyNoticeBtn");
		const noticeBox = $id("noticeBox");

		if (quickNoticeBtn) quickNoticeBtn.addEventListener("click", () => openModal(HELP_TEXT));

		if (copyNoticeBtn) {
			copyNoticeBtn.addEventListener("click", async () => {
				try {
					await navigator.clipboard.writeText((noticeBox && noticeBox.textContent) || "");
					openModal("已复制到剪贴板。");
				} catch {
					openModal("复制失败：浏览器不允许或不支持剪贴板 API。你可以手动选中文本复制。");
				}
			});
		}
	}

	function renderAll() {
		renderSelectors();
		renderSettings();
		renderEvents();
		renderHomeStats();
	}

	function markActiveNav() {
		const page = document.body.getAttribute("data-page") || "";
		document.querySelectorAll(".navbtn[data-page]").forEach((el) => {
			el.classList.toggle("active", el.getAttribute("data-page") === page);
		});
	}

	// ===== init =====
	setHelpText();
	markActiveNav();
	renderAll();
	wireSubmitPage();
	wireHistoryPage();
	wireSettingsPage();
	wireHelpButtons();

	refreshMonitoredOpenIds();
	syncFrontendSettingsFromServer().then((ok) => {
		if (ok) renderAll();
	});
	const initPoll = loadPollState();
	if (initPoll.enabled) startPendingQrPollAll();
	else setPollHint();
})();
