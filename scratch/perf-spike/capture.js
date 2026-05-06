#!/usr/bin/env node
/* eslint-disable no-console */

// Borrow the rebrowser-playwright that BAS already installed so the spike
// doesn't pull a fresh copy. If/when this graduates into BAS, this require
// becomes a normal package dep.
const path = require("node:path");
const fs = require("node:fs");

function resolveBASNodeModules() {
  if (process.env.BAS_NODE_MODULES) return process.env.BAS_NODE_MODULES;
  const root = process.env.VROOLI_ROOT || process.env.VROOLI_SOURCE_ROOT;
  if (root) {
    return path.join(root, "scenarios", "browser-automation-studio", "playwright-driver", "node_modules");
  }
  throw new Error("Set BAS_NODE_MODULES or VROOLI_ROOT/VROOLI_SOURCE_ROOT so capture.js can load rebrowser-playwright.");
}

const BAS_NODE_MODULES = resolveBASNodeModules();
const { chromium } = require(path.join(BAS_NODE_MODULES, "rebrowser-playwright"));

function parseArgs(argv) {
  const opts = { url: "http://localhost:21234", out: "/tmp/swarm-trace.json", headless: false };
  for (let i = 2; i < argv.length; i++) {
    const a = argv[i];
    if (a === "--url") opts.url = argv[++i];
    else if (a === "--out") opts.out = argv[++i];
    else if (a === "--headless") opts.headless = true;
  }
  return opts;
}

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

// Injected into every page before scripts run. Captures web-vitals + long-tasks
// without needing the npm package — just bare PerformanceObserver.
const WEB_VITALS_INIT = `
(() => {
  if (window.__perfSpikeInstalled) return;
  window.__perfSpikeInstalled = true;
  const data = { longTasks: [], paint: [], lcp: null, navigation: null, marks: [] };
  window.__perfSpikeData = data;

  try {
    new PerformanceObserver((list) => {
      for (const e of list.getEntries()) {
        data.longTasks.push({ start: e.startTime, duration: e.duration, name: e.name });
      }
    }).observe({ type: "longtask", buffered: true });
  } catch {}

  try {
    new PerformanceObserver((list) => {
      for (const e of list.getEntries()) {
        data.paint.push({ name: e.name, start: e.startTime });
      }
    }).observe({ type: "paint", buffered: true });
  } catch {}

  try {
    new PerformanceObserver((list) => {
      const entries = list.getEntries();
      const last = entries[entries.length - 1];
      if (last) data.lcp = { start: last.startTime, size: last.size };
    }).observe({ type: "largest-contentful-paint", buffered: true });
  } catch {}

  try {
    new PerformanceObserver((list) => {
      for (const e of list.getEntries()) {
        data.marks.push({ name: e.name, start: e.startTime, detail: e.detail ?? null });
      }
    }).observe({ type: "measure", buffered: true });
  } catch {}

  addEventListener("load", () => {
    const nav = performance.getEntriesByType("navigation")[0];
    if (nav) data.navigation = nav.toJSON();
  });
})();
`;

function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms));
}

async function dragHandle(page, fromX, fromY, deltas) {
  await page.mouse.move(fromX, fromY);
  await page.mouse.down();
  for (const [dx, dy] of deltas) {
    await page.mouse.move(fromX + dx, fromY + dy, { steps: 12 });
    await sleep(60);
  }
  await page.mouse.up();
}

async function exerciseSwarmManager(page, log) {
  log("waiting for sidebar to mount…");
  await page.waitForSelector("[data-testid='sidebar']", { timeout: 15000 });
  await page.waitForSelector("[data-testid='sidebar-resize-handle']", { timeout: 5000 });

  // Settle so initial fetches don't dominate the trace.
  await sleep(800);

  log("phase 1: resize sidebar on /graph");
  const handle = await page.$("[data-testid='sidebar-resize-handle']");
  const box = await handle.boundingBox();
  const startX = box.x + box.width / 2;
  const startY = box.y + box.height / 2;
  await dragHandle(page, startX, startY, [
    [60, 0], [-100, 0], [80, 0], [-40, 0], [120, 0], [-90, 0],
  ]);
  await sleep(300);

  log("phase 2: switch to backlog tab");
  const backlogTab = await page.$("[data-testid='sidebar-tab-backlog']");
  if (backlogTab) {
    await backlogTab.click();
    await sleep(700);
  } else {
    log("  (backlog tab not found — skipping)");
  }

  log("phase 3: type in sidebar search");
  const search = await page.$("[data-testid='sidebar-search']");
  if (search) {
    await search.click();
    await page.keyboard.type("perf", { delay: 80 });
    await sleep(500);
    await page.keyboard.press("Backspace");
    await page.keyboard.press("Backspace");
    await page.keyboard.press("Backspace");
    await page.keyboard.press("Backspace");
    await sleep(400);
  }

  log("phase 4: navigate to /command-post");
  await page.goto(page.url().replace(/\/[^/]*(\?.*)?$/, "/command-post"), { waitUntil: "domcontentloaded" });
  await page.waitForSelector("[data-testid='command-post-page']", { timeout: 10000 }).catch(() => {});
  await sleep(800);

  log("phase 5: resize sidebar on /command-post");
  const handle2 = await page.$("[data-testid='sidebar-resize-handle']");
  if (handle2) {
    const box2 = await handle2.boundingBox();
    if (box2) {
      const sx = box2.x + box2.width / 2;
      const sy = box2.y + box2.height / 2;
      await dragHandle(page, sx, sy, [
        [80, 0], [-120, 0], [60, 0], [-30, 0], [100, 0], [-70, 0],
      ]);
    }
  }
  await sleep(400);
}

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
  const opts = parseArgs(process.argv);
  console.log(`[spike] target=${opts.url} out=${opts.out} headless=${opts.headless}`);

  const browser = await chromium.launch({ headless: opts.headless });
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    deviceScaleFactor: 1,
  });
  await context.addInitScript({ content: WEB_VITALS_INIT });
  const page = await context.newPage();

  page.on("pageerror", (err) => console.error("[page error]", err.message));
  page.on("console", (msg) => {
    if (msg.type() === "error") console.error("[console]", msg.text());
  });

  console.log("[spike] navigating…");
  await page.goto(opts.url, { waitUntil: "domcontentloaded", timeout: 30000 });

  const client = await context.newCDPSession(page);

  console.log("[spike] starting Tracing");
  await client.send("Tracing.start", {
    transferMode: "ReturnAsStream",
    streamFormat: "json",
    streamCompression: "none",
    traceConfig: {
      includedCategories: TRACING_CATEGORIES,
      recordMode: "recordContinuously",
    },
  });

  const tracingComplete = new Promise((resolve, reject) => {
    client.once("Tracing.tracingComplete", (event) => resolve(event));
    setTimeout(() => reject(new Error("Tracing.tracingComplete timeout (60s)")), 60000);
  });

  const t0 = Date.now();
  await exerciseSwarmManager(page, (m) => console.log(`[spike] ${m}`));
  const elapsed = Date.now() - t0;
  console.log(`[spike] interactions complete in ${elapsed}ms; stopping Tracing`);

  await client.send("Tracing.end");
  const event = await tracingComplete;

  if (!event.stream) {
    throw new Error("Tracing complete but no stream handle returned");
  }
  console.log("[spike] streaming trace data…");
  const buf = await streamCdpIo(client, event.stream);
  fs.writeFileSync(opts.out, buf);
  console.log(`[spike] wrote ${buf.byteLength.toLocaleString()} bytes → ${opts.out}`);

  // Pull the web-vitals sidecar from the page.
  const vitals = await page.evaluate(() => window.__perfSpikeData ?? null);
  const vitalsPath = opts.out.replace(/\.json$/, "") + ".web-vitals.json";
  fs.writeFileSync(vitalsPath, JSON.stringify(vitals, null, 2));
  console.log(`[spike] wrote web-vitals sidecar → ${vitalsPath}`);

  await context.close();
  await browser.close();
}

main().catch((err) => {
  console.error("[spike] FAILED:", err);
  process.exit(1);
});
