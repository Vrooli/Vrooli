/**
 * Performance capture integration test (REAL browser).
 *
 * Proves the Tier 0 path end-to-end: a real Chromium CDP trace on a plain
 * page, plus the ⚛ Tier-1 pass-through when the page emits a
 * performance.measure("⚛ …") mark. The trace must be a DevTools-loadable
 * document containing timeline / CPU-profile events.
 *
 * Mirrors injector-selectors.test.ts: launches headless Chromium directly.
 * If the browser cannot launch in this environment the suite is skipped
 * rather than failing (same posture as the rest of the integration suite).
 */

import { chromium, type Browser, type BrowserContext, type Page } from 'rebrowser-playwright';
import { mkdtemp, readFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { PerformanceTracer, injectWebVitalsObserver, PERF_TRACE_FILE } from '../../src/tracing';

interface TraceEvent {
  cat?: string;
  name?: string;
}
interface TraceDoc {
  traceEvents: TraceEvent[];
  metadata?: Record<string, unknown>;
}

// A tiny page that lays out content. The ⚛ user-timing measure (the Tier 1
// marker) is emitted post-load via page.evaluate so it lands while tracing
// is unambiguously recording — exactly how a real React component commit
// fires during interaction.
const REACT_LIKE_PAGE = `data:text/html,${encodeURIComponent(`
<!doctype html><html><head><title>perf</title></head>
<body><div id="app"></div>
<script>
  const el = document.getElementById('app');
  for (let i = 0; i < 200; i++) {
    const d = document.createElement('div');
    d.textContent = 'row ' + i;
    el.appendChild(d);
  }
</script></body></html>`)}`;

// Simulate react-dom/profiling onRender emitting a ⚛ component-commit
// measure. Returns a promise that resolves after the marks are flushed.
function emitReactCommitMarks(page: Page): Promise<void> {
  return page.evaluate(() => {
    performance.mark('react-start');
    performance.measure('⚛ Counter [mount]', 'react-start');
    performance.measure('⚛ App [update]', 'react-start');
  });
}

const PLAIN_PAGE = `data:text/html,${encodeURIComponent(`
<!doctype html><html><head><title>plain</title></head>
<body><h1>plain page</h1><p>no instrumentation</p></body></html>`)}`;

describe('Performance capture (real browser, Tier 0 + ⚛ pass-through)', () => {
  let browser: Browser | undefined;
  let dir: string;

  beforeAll(async () => {
    try {
      browser = await chromium.launch({ headless: true });
    } catch {
      browser = undefined; // Environment cannot launch Chromium; tests self-skip.
    }
  });

  afterAll(async () => {
    if (browser) {
      await browser.close();
    }
  });

  beforeEach(async () => {
    dir = await mkdtemp(path.join(tmpdir(), 'perf-capture-'));
  });

  afterEach(async () => {
    await rm(dir, { recursive: true, force: true });
  });

  async function readTrace(): Promise<TraceDoc> {
    return JSON.parse(await readFile(path.join(dir, PERF_TRACE_FILE), 'utf8')) as TraceDoc;
  }

  async function capture(context: BrowserContext, page: Page, url: string): Promise<TraceDoc> {
    await injectWebVitalsObserver(context);
    const tracer = new PerformanceTracer(dir);
    await tracer.start(page);
    await page.goto(url, { waitUntil: 'networkidle' });
    // Give the observer a tick to flush LCP/paint entries.
    await page.waitForTimeout(200);
    await tracer.stop(page);
    return readTrace();
  }

  it('captures a Tier 0 trace with timeline / CPU profile events on a plain page (no ⚛)', async () => {
    if (!browser) {
      // eslint-disable-next-line no-console
      console.warn('chromium unavailable — skipping real-browser perf capture test');
      return;
    }
    const context = await browser.newContext();
    const page = await context.newPage();
    try {
      const trace = await capture(context, page, PLAIN_PAGE);

      expect(Array.isArray(trace.traceEvents)).toBe(true);
      expect(trace.traceEvents.length).toBeGreaterThan(0);

      const cats = new Set<string>(trace.traceEvents.map((e) => e.cat ?? ''));
      // Timeline category present (the flame chart spine).
      const hasTimeline = [...cats].some((c) => c.includes('devtools.timeline'));
      expect(hasTimeline).toBe(true);

      // No ⚛ user-timing marks on a plain page → Tier 0 only.
      const reactMarks = trace.traceEvents.filter(
        (e) => (e.cat ?? '').includes('blink.user_timing') && (e.name ?? '').includes('⚛')
      );
      expect(reactMarks.length).toBe(0);
    } finally {
      await context.close();
    }
  }, 30000);

  it('passes through ⚛ marks when the page emits them (Tier 1 rides along)', async () => {
    if (!browser) {
      return;
    }
    const context = await browser.newContext();
    const page = await context.newPage();
    try {
      await injectWebVitalsObserver(context);
      const tracer = new PerformanceTracer(dir);
      await tracer.start(page);
      await page.goto(REACT_LIKE_PAGE, { waitUntil: 'networkidle' });
      await emitReactCommitMarks(page);
      await page.waitForTimeout(200);
      await tracer.stop(page);
      const trace = await readTrace();

      const reactMarks = trace.traceEvents.filter(
        (e) => (e.cat ?? '').includes('blink.user_timing') && (e.name ?? '').includes('⚛')
      );
      // The ⚛ component measures must appear untouched in the trace.
      expect(reactMarks.length).toBeGreaterThan(0);
      const names = reactMarks.map((e) => e.name ?? '');
      expect(names.some((n) => n.includes('Counter'))).toBe(true);
    } finally {
      await context.close();
    }
  }, 30000);
});
