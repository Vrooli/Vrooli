#!/usr/bin/env node
/**
 * Perf capture template for React + Vite scenarios.
 *
 * Copy this file to /tmp/<scenario>/perf/capture.js and edit `exerciseTarget`
 * to drive the specific user interaction the audit is investigating. The full
 * methodology is in the `scenario-performance-audit` skill — read that first:
 *
 *   prompt-manager skill read scenario-performance-audit
 *
 * Usage:
 *   node /tmp/<scenario>/perf/capture.js \
 *     "http://localhost:${UI_PORT}" /tmp/<scenario>/perf/trace.json
 *
 * Prerequisites:
 *   - Browser-Automation-Studio (BAS) is set up (provides rebrowser-playwright).
 *   - Scenario is running with VROOLI_BUILD_MODE=profile, so the served bundle
 *     emits ⚛ user_timing entries through onProfilerRender.
 */

const path = require("node:path");
const fs = require("node:fs");
function resolveBASNodeModules() {
  const basNodeModules = process.env["BAS_NODE_MODULES"];
  if (basNodeModules) return basNodeModules;
  const root = process.env["VROOLI_ROOT"] || process.env["VROOLI_SOURCE_ROOT"];
  if (root) {
    return path.join(root, "scenarios", "browser-automation-studio", "playwright-driver", "node_modules");
  }
  throw new Error("Set BAS_NODE_MODULES or VROOLI_ROOT/VROOLI_SOURCE_ROOT so capture.js can load rebrowser-playwright.");
}

const BAS = resolveBASNodeModules();
const { chromium } = require(path.join(BAS, "rebrowser-playwright"));

// 120s covers multi-phase scripts (resize + scroll + click sequences).
// Bump higher only if your `exerciseTarget` deliberately runs >90s.
const TRACING_COMPLETE_TIMEOUT_MS = 120000;

const TRACING_CATEGORIES = [
  "devtools.timeline",
  "disabled-by-default-devtools.timeline",
  "disabled-by-default-devtools.timeline.frame",
  "disabled-by-default-devtools.timeline.stack",
  "disabled-by-default-v8.cpu_profiler",
  "disabled-by-default-devtools.screenshot",
  "blink.user_timing",
  "loading",
  "v8.execute",
];

const WEB_VITALS_INIT = `
(() => {
  if (window.__perfInstalled) return;
  window.__perfInstalled = true;
  const data = { longTasks: [], paint: [], lcp: null, marks: [] };
  window.__perfData = data;
  try { new PerformanceObserver((l) => { for (const e of l.getEntries()) data.longTasks.push({start:e.startTime,duration:e.duration,name:e.name}); }).observe({type:"longtask",buffered:true}); } catch {}
  try { new PerformanceObserver((l) => { for (const e of l.getEntries()) data.paint.push({name:e.name,start:e.startTime}); }).observe({type:"paint",buffered:true}); } catch {}
  try { new PerformanceObserver((l) => { const e = l.getEntries(); const last = e[e.length-1]; if (last) data.lcp = {start:last.startTime,size:last.size}; }).observe({type:"largest-contentful-paint",buffered:true}); } catch {}
})();
`;

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

// --- Reusable interaction helpers ---

// Drag a target horizontally by `dx` over N steps. The stepwise move settles
// transition queues better than a one-shot move, making resize-lag traces
// reproducible.
async function dragHorizontalOnce(page, selector, dx, steps = 20) {
  const handle = await page.locator(selector).boundingBox();
  if (!handle) throw new Error(`dragHorizontalOnce: ${selector} not found`);
  const startX = handle.x + handle.width / 2;
  const startY = handle.y + handle.height / 2;
  await page.mouse.move(startX, startY);
  await page.mouse.down();
  for (let i = 1; i <= steps; i++) {
    await page.mouse.move(startX + (dx * i) / steps, startY, { steps: 1 });
  }
  await page.mouse.up();
}

// Walk up from a target element to find its nearest scrollable ancestor.
// Useful when scrolling a virtualized list whose scroll-parent isn't the
// element you started from. Returns a CSS-locatable selector or null.
async function findScrollableAncestor(page, selector) {
  return await page.evaluate((sel) => {
    let el = document.querySelector(sel);
    while (el && el !== document.body) {
      const cs = window.getComputedStyle(el);
      if (cs.overflowY === "auto" || cs.overflowY === "scroll") {
        if (el.id) return `#${el.id}`;
        const parent = el.parentElement;
        if (parent) {
          const idx = Array.from(parent.children).indexOf(el);
          return `${parent.tagName.toLowerCase()} > :nth-child(${idx + 1})`;
        }
      }
      el = el.parentElement;
    }
    return null;
  }, selector);
}

// --- Customise this for the audit ---
//
// Each phase logs a label so the trace is annotated and analysis can correlate
// timestamps to phases. DOM-count logs around structural changes are a
// hard-to-fake confirmation that the change you're auditing actually shipped.
//
// Replace the placeholder `[data-testid='item']` selector with whatever the
// audit cares about. Keep total wall-clock under 30s for clean traces.

async function exerciseTarget(page, log) {
  log("waiting for shell to mount");
  await page.waitForSelector("[data-testid='app-shell']", { timeout: 15000 }).catch(() => {});
  await sleep(800);
  log(`pre-phase DOM count: ${await page.locator("[data-testid='item']").count()}`);

  log("phase 1: <describe what the user complained about>");
  // Examples:
  //   await dragHorizontalOnce(page, "[data-testid='resize-handle']", 80);
  //   const scrollSel = await findScrollableAncestor(page, "[data-testid='list']");
  //   if (scrollSel) await page.evaluate((s) => document.querySelector(s).scrollBy(0, 400), scrollSel);
  //   await page.click("[data-testid='tab-2']");
  //   await page.keyboard.type("hello");
  await sleep(500);

  log(`post-phase DOM count: ${await page.locator("[data-testid='item']").count()}`);
}
// ----------------------------------------------------------------------

async function streamCdpIo(client, handle) {
  const chunks = [];
  while (true) {
    const { data, eof, base64Encoded } = await client.send("IO.read", { handle, size: 1 << 20 });
    if (data) chunks.push(base64Encoded ? Buffer.from(data, "base64") : Buffer.from(data, "utf8"));
    if (eof) break;
  }
  await client.send("IO.close", { handle });
  return Buffer.concat(chunks);
}

async function main() {
  const url = process.argv[2] || (() => { throw new Error("usage: capture.js <url> [out]"); })();
  const out = process.argv[3] || path.join(__dirname, "trace.json");
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  await context.addInitScript({ content: WEB_VITALS_INIT });
  const page = await context.newPage();
  page.on("pageerror", (e) => console.error("[page]", e.message));

  await page.goto(url, { waitUntil: "domcontentloaded", timeout: 30000 });
  const client = await context.newCDPSession(page);

  await client.send("Tracing.start", {
    transferMode: "ReturnAsStream",
    streamFormat: "json",
    streamCompression: "none",
    traceConfig: { includedCategories: TRACING_CATEGORIES, recordMode: "recordContinuously" },
  });
  const traceComplete = new Promise((res, rej) => {
    client.once("Tracing.tracingComplete", res);
    setTimeout(() => rej(new Error("tracingComplete timeout")), TRACING_COMPLETE_TIMEOUT_MS);
  });

  await exerciseTarget(page, (m) => console.log(`[capture] ${m}`));
  await client.send("Tracing.end");
  const ev = await traceComplete;
  fs.writeFileSync(out, await streamCdpIo(client, ev.stream));
  fs.writeFileSync(
    out.replace(/\.json$/, ".web-vitals.json"),
    JSON.stringify(await page.evaluate(() => window.__perfData), null, 2),
  );
  console.log(`[capture] wrote ${out}`);
  await context.close();
  await browser.close();
}

main().catch((e) => { console.error(e); process.exit(1); });
