// End-to-end regression probe for "Claude Code standard-backend hang".
//
// Opens the web-console UI in a real browser, creates a *standard*-backend
// terminal via the toolbar + launcher, launches the Claude Code shortcut,
// waits for the xterm.js DOM to contain the Claude banner, and exits 0 on
// success / 1 on failure.
//
// IMPORTANT: the launcher's backend selector is hidden inside a collapsed
// "Session Options" accordion, and the form only sends the `backend` field
// in the launch request when the user explicitly changes it from the
// currently-applied default. The default is "persistent" when tmux is
// available. So the test must (a) expand options, (b) switch to standard,
// (c) verify the created session is actually `backend=standard` via the
// REST API — otherwise we'd silently be testing the persistent path,
// which was never broken in the first place.
//
// Usage:
//   node claude-standard-backend.mjs [--browser chromium|firefox]
//
// Requirements:
//   - web-console running on http://localhost:21233 (override with WC_UI_URL)
//   - web-console API on http://localhost:16382 (override with WC_API_URL)
//   - Claude Code logged in locally (~/.claude/.credentials.json)
//   - playwright installed and the requested browser downloaded
//     (npx playwright install chromium firefox)
//
// Exit codes:
//   0 — Claude banner rendered (standard backend is good on this browser)
//   1 — banner missing (hang reproduced — inspect /tmp/wc-probe-<browser>)
//   2 — test infrastructure failure (session backend mismatch, UI widget
//       missing, Claude not authed, etc.)

import { chromium, firefox } from "playwright";
import { writeFileSync, mkdirSync, existsSync } from "node:fs";
import { homedir } from "node:os";

const args = process.argv.slice(2);
const browserName = (args.includes("--browser") && args[args.indexOf("--browser") + 1]) || "chromium";
const UI_URL = process.env.WC_UI_URL || "http://localhost:21233";
const API_URL = process.env.WC_API_URL || "http://localhost:16382";
const OUT_DIR = `/tmp/wc-probe-${browserName}`;
mkdirSync(OUT_DIR, { recursive: true });

if (!existsSync(`${homedir()}/.claude/.credentials.json`)) {
  console.error("SKIP: Claude not logged in (~/.claude/.credentials.json missing)");
  process.exit(2);
}

const engines = { chromium, firefox };
const engine = engines[browserName];
if (!engine) {
  console.error(`unknown browser: ${browserName}`);
  process.exit(2);
}

const launchOpts = browserName === "chromium"
  ? { headless: true, executablePath: process.env.CHROME_EXECUTABLE || "/usr/bin/google-chrome" }
  : { headless: true };

// Snapshot existing sessions so we can identify the one this test creates.
const before = await fetch(`${API_URL}/api/v1/sessions`).then(r => r.json());
const beforeIds = new Set((before?.sessions ?? before).map(s => s.id));

const browser = await engine.launch(launchOpts);
const ctx = await browser.newContext({ viewport: { width: 1280, height: 900 } });
const page = await ctx.newPage();

const pageLogs = [];
const wsFrames = [];
page.on("console", msg => pageLogs.push({ level: msg.type(), text: msg.text() }));
page.on("pageerror", err => pageLogs.push({ level: "pageerror", text: String(err), stack: err?.stack ?? "" }));
page.on("websocket", ws => {
  const url = ws.url();
  const sid = (url.match(/sessions\/([a-f0-9-]+)\//) || [])[1] || "other";
  pageLogs.push({ level: "ws_open", text: url });
  ws.on("framereceived", ev => {
    const data = ev.payload?.toString?.("utf8") ?? String(ev.payload ?? "");
    wsFrames.push({ sid, len: data.length, preview: data.slice(0, 300) });
  });
  ws.on("close", () => pageLogs.push({ level: "ws_close", text: url }));
});

await page.goto(UI_URL, { waitUntil: "domcontentloaded", timeout: 20000 });
await page.waitForTimeout(1500);

// ── 1. Open launcher ──────────────────────────────────────────────────
{
  const launcher = page.getByTestId("terminal-launcher");
  const open = await launcher.isVisible().catch(() => false);
  if (!open) {
    const newBtn = page.getByTestId("toolbar-new");
    if (!(await newBtn.count())) {
      console.error("toolbar-new not found");
      await browser.close();
      process.exit(2);
    }
    await newBtn.click();
    await page.waitForTimeout(400);
  }
}

// ── 2. Expand "Session Options" so the backend selector becomes visible ─
{
  const toggle = page.getByTestId("launcher-options-toggle");
  if (!(await toggle.count())) {
    console.error("launcher-options-toggle not found — UI structure changed");
    await browser.close();
    process.exit(2);
  }
  await toggle.click();
  await page.waitForTimeout(200);
}

// ── 3. Force the backend to "standard" ────────────────────────────────
{
  const sel = page.getByTestId("launcher-backend-select");
  if (!(await sel.count())) {
    // Only one backend available (e.g., tmux not installed) — selector
    // stays hidden. That's a legit misconfig for this test.
    console.error("launcher-backend-select not found (only one backend available?)");
    await browser.close();
    process.exit(2);
  }
  await sel.selectOption("standard");
  const currentValue = await sel.inputValue();
  if (currentValue !== "standard") {
    console.error(`backend select value is '${currentValue}', expected 'standard'`);
    await browser.close();
    process.exit(2);
  }
}

// ── 4. Click the Claude Code shortcut ────────────────────────────────
{
  const shortcut = page.getByTestId("launcher-shortcut-claude-code");
  if (!(await shortcut.count())) {
    console.error("Claude Code shortcut not in launcher");
    await browser.close();
    process.exit(2);
  }
  await shortcut.first().click();
}

// ── 5. Identify the just-created session and verify backend=standard ──
let createdId = null;
{
  const deadline = Date.now() + 5000;
  while (Date.now() < deadline) {
    const after = await fetch(`${API_URL}/api/v1/sessions`).then(r => r.json()).catch(() => null);
    if (!after) { await page.waitForTimeout(200); continue; }
    const sessions = after?.sessions ?? after;
    const fresh = sessions.filter(s => !beforeIds.has(s.id));
    if (fresh.length) {
      createdId = fresh[0].id;
      const backend = fresh[0].backend;
      if (backend !== "standard") {
        console.error(`!! session was created with backend='${backend}', not 'standard'. ` +
          `Launcher options not applied — this test would have been testing the persistent path. ` +
          `Session id: ${createdId}`);
        await browser.close();
        process.exit(2);
      }
      break;
    }
    await page.waitForTimeout(200);
  }
}
if (!createdId) {
  console.error("no new session appeared in REST list within 5 s");
  await browser.close();
  process.exit(2);
}

// ── 5a. Activate the new pane's tab so xterm.js renders it. Without an
//        explicit click the grid layout may keep another pane focused and
//        the new xterm's DOM rows won't be in the visible viewport, making
//        any DOM-scraping assertion flaky.
{
  const tab = page.getByTestId(`tab-${createdId}`);
  if (await tab.count()) {
    await tab.click().catch(() => {});
    await page.waitForTimeout(400);
  }
}

// ── 6. Wait 10 s and assert the xterm parser did not crash ────────────
// The root-cause bug (xterm.js v6 `requestMode` throwing ReferenceError
// on `\x1b[?2026$p`) manifests as a `pageerror` event. That's the
// rock-solid regression signal — far more reliable than DOM scraping,
// because claude's actual rendered output depends on account state
// (weekly-limit banner vs normal prompt, permission-mode footer wording,
// scrollback position in a multi-pane layout, etc). If the parser
// crashes, every subsequent render is dead — if it doesn't crash, the
// banner will eventually appear for a real human user.
await page.waitForTimeout(10000);

const xtermCrash = pageLogs.some(l =>
  l.level === "pageerror" && /requestMode|ReferenceError: r is not defined/.test(l.text || "")
);
const anyPageError = pageLogs.some(l => l.level === "pageerror");

await page.screenshot({ path: `${OUT_DIR}/screenshot.png`, fullPage: true });
writeFileSync(`${OUT_DIR}/logs.json`, JSON.stringify(pageLogs, null, 2));
writeFileSync(`${OUT_DIR}/ws_frames.json`, JSON.stringify(wsFrames.slice(0, 300), null, 2));
const finalBuffer = await page.evaluate(() => {
  const out = [];
  document.querySelectorAll(".xterm-rows").forEach(el => {
    out.push(Array.from(el.children).map(r => r.textContent).join("\n"));
  });
  return out;
});
writeFileSync(`${OUT_DIR}/xterm_buffer.json`, JSON.stringify(finalBuffer, null, 2));

// Confirm at least one WS frame carrying a stdout payload showed up for
// this session — otherwise the test isn't actually exercising the PTY
// output path and the "no crash" assertion is meaningless.
const stdoutFramesForSession = wsFrames.filter(f =>
  f.sid === createdId && (f.preview || "").includes('"stdout"')
).length;

// ── 7. Clean up the session so we don't leave state around ────────────
try { await fetch(`${API_URL}/api/v1/sessions/${createdId}`, { method: "DELETE" }); } catch {}

await browser.close();

if (xtermCrash) {
  console.error(`[${browserName}] FAIL — xterm.js parser crash detected (the regression bug). Session ${createdId}. See ${OUT_DIR}/logs.json.`);
  process.exit(1);
}
if (anyPageError) {
  console.error(`[${browserName}] FAIL — unexpected page error during standard-backend Claude run. See ${OUT_DIR}/logs.json for details.`);
  process.exit(1);
}
if (stdoutFramesForSession < 3) {
  console.error(`[${browserName}] FAIL — only ${stdoutFramesForSession} stdout frames reached the WS client for session ${createdId}; claude may not have run at all.`);
  process.exit(1);
}
console.log(`[${browserName}] PASS — session ${createdId}: ${stdoutFramesForSession} stdout frames received, no xterm parser crash.`);
process.exit(0);
