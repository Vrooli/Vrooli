/**
 * End-to-End Timeline Validation Tests
 *
 * These tests verify the complete recording flow from user interaction
 * to timeline entry generation. They ensure that:
 *
 * 1. All interaction types produce timeline entries
 * 2. Timeline entries have correct structure (proto schema compliance)
 * 3. Event ordering is preserved (action before resulting navigation)
 * 4. Events persist across navigation
 *
 * CRITICAL: These tests will FAIL if timeline events are lost, which is
 * exactly what happened with the navigation race bug. The goal is to
 * catch such issues before they reach production.
 */

import { describe, beforeAll, afterAll, beforeEach, afterEach, it, expect } from '@jest/globals';
import { chromium, Browser, BrowserContext, Page } from 'rebrowser-playwright';
import * as http from 'http';
import {
  createRecordingContextInitializer,
  createRecordingPipelineManager,
  RecordingContextInitializer,
  RecordingPipelineManager,
  TimelineEntry,
} from '../../src/recording';
import { waitForScriptReady } from '../../src/recording';
import { ActionType } from '../../src/proto/recording';

// Increase timeout for comprehensive E2E testing
jest.setTimeout(120000);

/**
 * Helper to get action type from TimelineEntry.
 */
function getActionType(entry: TimelineEntry): ActionType | undefined {
  return entry.action?.type;
}

/**
 * Test server providing pages with various interactive elements.
 */
class TimelineTestServer {
  private server: http.Server | null = null;
  private port = 0;

  async start(): Promise<number> {
    return new Promise((resolve) => {
      this.server = http.createServer((req, res) => {
        const path = req.url || '/';

        if (path === '/interactive-page') {
          res.writeHead(200, { 'Content-Type': 'text/html' });
          res.end(`
            <!DOCTYPE html>
            <html>
            <head>
              <style>
                body { height: 2000px; padding: 20px; font-family: sans-serif; }
                button, input, select, textarea { margin: 10px; padding: 10px; }
                .container { margin-bottom: 500px; }
                #hover-target { width: 100px; height: 100px; background: #ccc; }
              </style>
            </head>
            <body>
              <div class="container">
                <h1>Interactive Test Page</h1>

                <section id="buttons">
                  <button id="btn-1">Click Me</button>
                  <button id="btn-2">Double Click</button>
                  <button id="btn-3" oncontextmenu="return false;">Right Click</button>
                </section>

                <section id="inputs">
                  <input type="text" id="text-input" placeholder="Type here" />
                  <input type="password" id="password-input" placeholder="Password" />
                  <textarea id="textarea" placeholder="Multiline input"></textarea>
                </section>

                <section id="selects">
                  <select id="dropdown">
                    <option value="">Select...</option>
                    <option value="opt1">Option 1</option>
                    <option value="opt2">Option 2</option>
                  </select>
                </section>

                <section id="hover-focus">
                  <div id="hover-target">Hover Here</div>
                  <input type="text" id="focus-target" placeholder="Focus here" />
                </section>

                <section id="navigation">
                  <a id="nav-link" href="/page-2">Navigate to Page 2</a>
                  <button id="nav-btn" onclick="window.location.href='/page-2'">JS Navigate</button>
                </section>
              </div>

              <div class="container">
                <p>Scroll content - this page is tall to test scroll events</p>
              </div>
            </body>
            </html>
          `);
          return;
        }

        if (path === '/page-2') {
          res.writeHead(200, { 'Content-Type': 'text/html' });
          res.end(`
            <!DOCTYPE html>
            <html>
            <head></head>
            <body>
              <h1>Page 2</h1>
              <button id="page-2-btn">Button on Page 2</button>
              <a id="back-link" href="/interactive-page">Back</a>
            </body>
            </html>
          `);
          return;
        }

        res.writeHead(404);
        res.end('Not Found');
      });
      this.server.listen(0, () => {
        this.port = (this.server!.address() as { port: number }).port;
        resolve(this.port);
      });
    });
  }

  async stop(): Promise<void> {
    return new Promise((resolve) => {
      if (this.server) {
        this.server.close(() => resolve());
      } else {
        resolve();
      }
    });
  }

  getUrl(path: string = '/'): string {
    return `http://localhost:${this.port}${path}`;
  }
}

/**
 * Simulate a real scroll using CDP Input.dispatchMouseEvent with wheel type.
 */
async function simulateRealScroll(page: Page, deltaY: number = 200): Promise<void> {
  const client = await page.context().newCDPSession(page);
  try {
    const viewport = page.viewportSize();
    const x = (viewport?.width || 800) / 2;
    const y = (viewport?.height || 600) / 2;

    await client.send('Input.dispatchMouseEvent', {
      type: 'mouseWheel',
      x,
      y,
      deltaX: 0,
      deltaY,
    });
  } finally {
    await client.detach().catch(() => {});
  }
}

describe('End-to-End Timeline Validation (Integration)', () => {
  let browser: Browser;
  let server: TimelineTestServer;

  beforeAll(async () => {
    browser = await chromium.launch({ headless: true });
    server = new TimelineTestServer();
    await server.start();
  });

  afterAll(async () => {
    await browser.close();
    await server.stop();
  });

  describe('complete recording flow', () => {
    let context: BrowserContext;
    let page: Page;
    let initializer: RecordingContextInitializer;
    let pipelineManager: RecordingPipelineManager;
    let capturedEntries: TimelineEntry[];

    beforeEach(async () => {
      context = await browser.newContext();
      initializer = createRecordingContextInitializer({});
      await initializer.initialize(context);
      page = await context.newPage();

      pipelineManager = createRecordingPipelineManager(page, context, initializer, {
        sessionId: 'e2e-test-session',
      });
      await pipelineManager.initialize();

      capturedEntries = [];
    });

    afterEach(async () => {
      if (pipelineManager.isRecording()) {
        await pipelineManager.stopRecording();
      }
      await context.close();
    });

    it('[CRITICAL] should capture click events in timeline', async () => {
      await page.goto(server.getUrl('/interactive-page'));
      await waitForScriptReady(page, 5000);

      await pipelineManager.startRecording({
        sessionId: 'e2e-test-session',
        recordingId: 'click-test',
        onEntry: (entry) => capturedEntries.push(entry),
      });

      // Perform click
      await page.click('#btn-1');
      await page.waitForTimeout(500);

      // Stop recording
      await pipelineManager.stopRecording();

      // Validate click event was captured
      const clickEvents = capturedEntries.filter((e) => getActionType(e) === ActionType.CLICK);

      expect(clickEvents.length).toBeGreaterThan(0);
      expect(clickEvents[0].action).toBeDefined();
      expect(clickEvents[0].sequenceNum).toBeDefined();
      expect(clickEvents[0].timestamp).toBeDefined();
    });

    it('[CRITICAL] should capture input/type events in timeline', async () => {
      await page.goto(server.getUrl('/interactive-page'));
      await waitForScriptReady(page, 5000);

      await pipelineManager.startRecording({
        sessionId: 'e2e-test-session',
        recordingId: 'input-test',
        onEntry: (entry) => capturedEntries.push(entry),
      });

      // Use page.type() instead of fill() - type() dispatches individual keystrokes
      // which triggers the browser script's input handler
      await page.type('#text-input', 'hello');
      // Wait for input debounce (CONFIG.INPUT_DEBOUNCE_MS = 300ms in browser script)
      await page.waitForTimeout(800);

      await pipelineManager.stopRecording();

      // Validate input event was captured (browser script sends 'type' -> ActionType.INPUT)
      const inputEvents = capturedEntries.filter((e) => {
        const type = getActionType(e);
        return type === ActionType.INPUT;
      });

      console.log('Input test - captured entries:', capturedEntries.map((e) => ({
        type: getActionType(e),
        seq: e.sequenceNum,
      })));

      expect(inputEvents.length).toBeGreaterThan(0);
    });

    it('[CRITICAL] should capture scroll events in timeline', async () => {
      await page.goto(server.getUrl('/interactive-page'));
      await waitForScriptReady(page, 5000);

      await pipelineManager.startRecording({
        sessionId: 'e2e-test-session',
        recordingId: 'scroll-test',
        onEntry: (entry) => capturedEntries.push(entry),
      });

      // Simulate real scroll
      await simulateRealScroll(page, 300);
      await page.waitForTimeout(700); // Wait for scroll debounce

      await pipelineManager.stopRecording();

      // Validate scroll event was captured
      const scrollEvents = capturedEntries.filter((e) => getActionType(e) === ActionType.SCROLL);

      expect(scrollEvents.length).toBeGreaterThan(0);
    });

    // DESIGN DECISION: Focus events are disabled by default in RECORDING_EVENT_CATEGORIES
    // (selector-config.ts: focus.enabled = false). This is intentional because:
    // 1. Focus events can be noisy (triggered frequently during normal interaction)
    // 2. Input events work without focus events being captured
    // To enable focus capture, set RECORDING_EVENT_CATEGORIES.focus.enabled = true
    // For now, this test validates the expected behavior (no focus events captured)
    it('should NOT capture focus events when focus category is disabled (default)', async () => {
      await page.goto(server.getUrl('/interactive-page'));
      await waitForScriptReady(page, 5000);

      await pipelineManager.startRecording({
        sessionId: 'e2e-test-session',
        recordingId: 'focus-test',
        onEntry: (entry) => capturedEntries.push(entry),
      });

      // Focus an input - this should NOT be captured because focus category is disabled
      await page.focus('#focus-target');
      await page.waitForTimeout(300);

      await pipelineManager.stopRecording();

      // Focus events should NOT be captured when focus category is disabled
      const focusEvents = capturedEntries.filter((e) => getActionType(e) === ActionType.FOCUS);

      // With default config (focus.enabled = false), no focus events should be captured
      expect(focusEvents.length).toBe(0);
    });

    it('[CRITICAL] should capture click, scroll, and input events in a full session', async () => {
      await page.goto(server.getUrl('/interactive-page'));
      await waitForScriptReady(page, 5000);

      await pipelineManager.startRecording({
        sessionId: 'e2e-test-session',
        recordingId: 'full-session',
        onEntry: (entry) => capturedEntries.push(entry),
      });

      // 1. Click
      await page.click('#btn-1');
      await page.waitForTimeout(300);

      // 2. Scroll
      await simulateRealScroll(page, 200);
      await page.waitForTimeout(700);

      // 3. Input (type into field)
      await page.type('#text-input', 'test');
      await page.waitForTimeout(800); // Wait for input debounce

      await pipelineManager.stopRecording();

      // Validate all core event types were captured
      const clickEvents = capturedEntries.filter((e) => getActionType(e) === ActionType.CLICK);
      const scrollEvents = capturedEntries.filter((e) => getActionType(e) === ActionType.SCROLL);
      const inputEvents = capturedEntries.filter((e) => getActionType(e) === ActionType.INPUT);

      console.log('Full session captured:', {
        clicks: clickEvents.length,
        scrolls: scrollEvents.length,
        inputs: inputEvents.length,
        total: capturedEntries.length,
      });

      // All core event types must be captured
      expect(clickEvents.length).toBeGreaterThan(0);
      expect(scrollEvents.length).toBeGreaterThan(0);
      expect(inputEvents.length).toBeGreaterThan(0);

      // Note: Focus events are disabled by default (RECORDING_EVENT_CATEGORIES.focus.enabled = false)
      // This is intentional - focus events can be noisy and input events work without them
    });

    it('should maintain correct event ordering', async () => {
      await page.goto(server.getUrl('/interactive-page'));
      await waitForScriptReady(page, 5000);

      await pipelineManager.startRecording({
        sessionId: 'e2e-test-session',
        recordingId: 'ordering-test',
        onEntry: (entry) => capturedEntries.push(entry),
      });

      // Perform actions in known order
      await page.click('#btn-1');
      await page.waitForTimeout(200);
      await page.click('#btn-2');
      await page.waitForTimeout(200);
      await page.fill('#text-input', 'abc');
      await page.waitForTimeout(300);

      await pipelineManager.stopRecording();

      // Verify sequence numbers are ascending
      const sequenceNums = capturedEntries.map((e) => e.sequenceNum).filter((n): n is number => n !== undefined);

      for (let i = 1; i < sequenceNums.length; i++) {
        expect(sequenceNums[i]).toBeGreaterThan(sequenceNums[i - 1]);
      }

      // Verify timestamps are ascending (or equal for rapid events)
      const timestamps = capturedEntries
        .map((e) => e.timestamp)
        .filter((t): t is NonNullable<typeof t> => t !== undefined)
        .map((t) => Number(t.seconds) * 1000 + Number(t.nanos) / 1000000);

      for (let i = 1; i < timestamps.length; i++) {
        expect(timestamps[i]).toBeGreaterThanOrEqual(timestamps[i - 1]);
      }
    });

    it('[CRITICAL] should capture click before navigation event when click triggers nav', async () => {
      await page.goto(server.getUrl('/interactive-page'));
      await waitForScriptReady(page, 5000);

      await pipelineManager.startRecording({
        sessionId: 'e2e-test-session',
        recordingId: 'nav-ordering-test',
        onEntry: (entry) => capturedEntries.push(entry),
      });

      // Click the navigation link
      await page.click('#nav-link');
      await page.waitForURL('**/page-2', { timeout: 5000 });
      await page.waitForTimeout(500);

      await pipelineManager.stopRecording();

      // Find click and navigate events
      const clickEvents = capturedEntries.filter((e) => getActionType(e) === ActionType.CLICK);
      const navEvents = capturedEntries.filter((e) => getActionType(e) === ActionType.NAVIGATE);

      console.log('Nav ordering test:', {
        clicks: clickEvents.length,
        navs: navEvents.length,
        entries: capturedEntries.map((e) => ({
          type: getActionType(e),
          seq: e.sequenceNum,
        })),
      });

      // CRITICAL: Click MUST be captured
      expect(clickEvents.length).toBeGreaterThan(0);

      // Navigation should also be captured
      expect(navEvents.length).toBeGreaterThan(0);

      // Click should come before navigation in sequence
      const lastClick = clickEvents[clickEvents.length - 1];
      const firstNav = navEvents.find((n) => n.url?.includes('/page-2'));

      if (lastClick && firstNav && lastClick.sequenceNum !== undefined && firstNav.sequenceNum !== undefined) {
        expect(lastClick.sequenceNum).toBeLessThan(firstNav.sequenceNum);
      }
    });

    it('should continue capturing events after navigation', async () => {
      await page.goto(server.getUrl('/interactive-page'));
      await waitForScriptReady(page, 5000);

      await pipelineManager.startRecording({
        sessionId: 'e2e-test-session',
        recordingId: 'post-nav-test',
        onEntry: (entry) => capturedEntries.push(entry),
      });

      // Click before navigation
      await page.click('#btn-1');
      await page.waitForTimeout(200);

      // Navigate
      await page.click('#nav-link');
      await page.waitForURL('**/page-2', { timeout: 5000 });
      await waitForScriptReady(page, 5000);
      await page.waitForTimeout(300);

      const entriesBeforePostNavClick = capturedEntries.length;

      // Click on new page
      await page.click('#page-2-btn');
      await page.waitForTimeout(500);

      await pipelineManager.stopRecording();

      // Should have captured events both before and after navigation
      const postNavClicks = capturedEntries
        .slice(entriesBeforePostNavClick)
        .filter((e) => getActionType(e) === ActionType.CLICK);

      console.log('Post-navigation capture:', {
        totalEntries: capturedEntries.length,
        entriesBeforePostNavClick,
        postNavClicks: postNavClicks.length,
      });

      // CRITICAL: Events after navigation must be captured
      expect(postNavClicks.length).toBeGreaterThan(0);
    });
  });

  describe('timeline entry structure validation', () => {
    let context: BrowserContext;
    let page: Page;
    let initializer: RecordingContextInitializer;
    let pipelineManager: RecordingPipelineManager;
    let capturedEntries: TimelineEntry[];

    beforeEach(async () => {
      context = await browser.newContext();
      initializer = createRecordingContextInitializer({});
      await initializer.initialize(context);
      page = await context.newPage();

      pipelineManager = createRecordingPipelineManager(page, context, initializer, {
        sessionId: 'structure-test-session',
      });
      await pipelineManager.initialize();

      capturedEntries = [];
    });

    afterEach(async () => {
      if (pipelineManager.isRecording()) {
        await pipelineManager.stopRecording();
      }
      await context.close();
    });

    it('should have valid TimelineEntry structure for click events', async () => {
      await page.goto(server.getUrl('/interactive-page'));
      await waitForScriptReady(page, 5000);

      await pipelineManager.startRecording({
        sessionId: 'structure-test-session',
        recordingId: 'structure-click',
        onEntry: (entry) => capturedEntries.push(entry),
      });

      await page.click('#btn-1');
      await page.waitForTimeout(500);

      await pipelineManager.stopRecording();

      const clickEntry = capturedEntries.find((e) => getActionType(e) === ActionType.CLICK);

      expect(clickEntry).toBeDefined();
      if (clickEntry) {
        // Core TimelineEntry fields
        expect(clickEntry.id).toBeDefined();
        expect(clickEntry.sequenceNum).toBeDefined();
        expect(clickEntry.timestamp).toBeDefined();

        // Action structure
        expect(clickEntry.action).toBeDefined();
        expect(clickEntry.action?.type).toBe(ActionType.CLICK);

        // URL is stored in telemetry (per proto schema)
        expect(clickEntry.telemetry).toBeDefined();
        expect(clickEntry.telemetry?.url).toContain('/interactive-page');

        // SessionId is stored in context.origin (per proto schema)
        expect(clickEntry.context).toBeDefined();
        expect(clickEntry.context?.origin?.case).toBe('sessionId');
        expect(clickEntry.context?.origin?.value).toBe('structure-test-session');

        console.log('Click entry structure:', {
          id: clickEntry.id,
          url: clickEntry.telemetry?.url,
          sessionId: clickEntry.context?.origin?.value,
        });
      }
    });

    it('should have valid TimelineEntry structure for scroll events', async () => {
      await page.goto(server.getUrl('/interactive-page'));
      await waitForScriptReady(page, 5000);

      await pipelineManager.startRecording({
        sessionId: 'structure-test-session',
        recordingId: 'structure-scroll',
        onEntry: (entry) => capturedEntries.push(entry),
      });

      await simulateRealScroll(page, 200);
      await page.waitForTimeout(700);

      await pipelineManager.stopRecording();

      const scrollEntry = capturedEntries.find((e) => getActionType(e) === ActionType.SCROLL);

      expect(scrollEntry).toBeDefined();
      if (scrollEntry) {
        expect(scrollEntry.id).toBeDefined();
        expect(scrollEntry.action?.type).toBe(ActionType.SCROLL);

        // Scroll params are stored in action.params (per proto schema)
        expect(scrollEntry.action?.params?.case).toBe('scroll');

        // Log scroll params for debugging
        const scrollParams = scrollEntry.action?.params?.value;
        console.log('Scroll entry params:', {
          case: scrollEntry.action?.params?.case,
          scrollX: (scrollParams as any)?.x,
          scrollY: (scrollParams as any)?.y,
          deltaY: (scrollParams as any)?.deltaY,
        });
      }
    });

    it('should include element metadata for element-based actions', async () => {
      await page.goto(server.getUrl('/interactive-page'));
      await waitForScriptReady(page, 5000);

      await pipelineManager.startRecording({
        sessionId: 'structure-test-session',
        recordingId: 'structure-meta',
        onEntry: (entry) => capturedEntries.push(entry),
      });

      await page.click('#btn-1');
      await page.waitForTimeout(500);

      await pipelineManager.stopRecording();

      const clickEntry = capturedEntries.find((e) => getActionType(e) === ActionType.CLICK);

      expect(clickEntry).toBeDefined();
      if (clickEntry) {
        // Element metadata is stored in action.metadata.elementSnapshot (per proto schema)
        expect(clickEntry.action?.metadata).toBeDefined();
        expect(clickEntry.action?.metadata?.elementSnapshot).toBeDefined();
        // tagName may be lowercase or uppercase depending on browser script behavior
        expect(clickEntry.action?.metadata?.elementSnapshot?.tagName?.toLowerCase()).toBe('button');

        // Selectors are stored in action.metadata.selectorCandidates (per proto schema)
        expect(clickEntry.action?.metadata?.selectorCandidates).toBeDefined();
        expect(clickEntry.action?.metadata?.selectorCandidates?.length).toBeGreaterThan(0);

        console.log('Element metadata:', {
          tagName: clickEntry.action?.metadata?.elementSnapshot?.tagName,
          id: clickEntry.action?.metadata?.elementSnapshot?.id,
          selectorCount: clickEntry.action?.metadata?.selectorCandidates?.length,
        });
      }
    });
  });
});
