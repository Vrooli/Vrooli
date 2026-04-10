/**
 * Pipeline E2E Tests
 *
 * Tests the full recording pipeline in a CI-friendly way:
 * - Self-contained (no external server needed)
 * - Tests all event types (click, input, scroll, focus, navigation)
 * - Validates TimelineEntry structure
 * - Mirrors the diagnostic pipeline test logic from self-test.ts
 *
 * These tests ensure the recording pipeline works correctly and can be
 * run as part of `pnpm test` in CI environments.
 */

import { chromium, Browser, BrowserContext, Page } from 'rebrowser-playwright';
import * as http from 'http';
import {
  createRecordingContextInitializer,
  createRecordingPipelineManager,
  RecordingContextInitializer,
  RecordingPipelineManager,
  TimelineEntry,
  waitForScriptReady,
} from '../../src/recording';
import { ActionType } from '../../src/proto/recording';

// Increase timeout for comprehensive E2E testing
jest.setTimeout(120000);

/**
 * Test page HTML with all interactive elements needed for pipeline testing.
 * Includes scrollable content, buttons, inputs, and navigation links.
 */
const TEST_PAGE_HTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Pipeline Test Page</title>
  <style>
    body { min-height: 200vh; padding: 20px; font-family: sans-serif; }
    .section { margin: 20px 0; padding: 20px; background: #f5f5f5; border-radius: 8px; }
    button, input { padding: 10px 20px; margin: 5px; }
    .spacer { height: 500px; }
  </style>
</head>
<body>
  <h1>Pipeline E2E Test Page</h1>

  <div class="section">
    <h2>Click Test</h2>
    <button id="test-btn" data-testid="test-button">Click Me</button>
  </div>

  <div class="section">
    <h2>Input Test</h2>
    <input type="text" id="test-input" data-testid="test-input" placeholder="Type here" />
  </div>

  <div class="section">
    <h2>Navigation Test</h2>
    <a id="test-link" href="/page-2">Navigate to Page 2</a>
  </div>

  <div class="spacer"></div>
  <div class="section">
    <p>Bottom of page - scroll content</p>
  </div>
</body>
</html>`;

const PAGE_2_HTML = `<!DOCTYPE html>
<html>
<head><title>Page 2</title></head>
<body style="min-height: 200vh; padding: 20px;">
  <h1>Page 2</h1>
  <button id="page-2-btn">Button on Page 2</button>
  <a id="back-link" href="/">Back to Main</a>
</body>
</html>`;

/**
 * Test server providing pages with interactive elements.
 */
class PipelineTestServer {
  private server: http.Server | null = null;
  private port = 0;

  async start(): Promise<number> {
    return new Promise((resolve) => {
      this.server = http.createServer((req, res) => {
        const path = req.url || '/';

        if (path === '/' || path === '/index.html') {
          res.writeHead(200, { 'Content-Type': 'text/html' });
          res.end(TEST_PAGE_HTML);
          return;
        }

        if (path === '/page-2') {
          res.writeHead(200, { 'Content-Type': 'text/html' });
          res.end(PAGE_2_HTML);
          return;
        }

        res.writeHead(404);
        res.end('Not Found');
      });

      this.server.listen(0, () => {
        const server = this.server;
        if (!server) {
          throw new Error('Test server failed to initialize');
        }
        const address = server.address();
        if (!address || typeof address === 'string') {
          throw new Error('Unexpected server address for test server');
        }
        this.port = address.port;
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

/**
 * Helper to get action type from TimelineEntry.
 */
function getActionType(entry: TimelineEntry): ActionType | undefined {
  return entry.action?.type;
}

function getTelemetryUrl(entry: TimelineEntry): string | undefined {
  const telemetry = entry.telemetry as { url?: unknown } | undefined;
  if (!telemetry) {
    return undefined;
  }
  return typeof telemetry.url === 'string' ? telemetry.url : undefined;
}

function getEntryUrl(entry: TimelineEntry): string | undefined {
  const rawUrl = (entry as { url?: unknown }).url;
  return typeof rawUrl === 'string' ? rawUrl : undefined;
}

describe('Pipeline E2E Tests', () => {
  let browser: Browser;
  let server: PipelineTestServer;

  beforeAll(async () => {
    browser = await chromium.launch({ headless: true });
    server = new PipelineTestServer();
    await server.start();
  });

  afterAll(async () => {
    await browser.close();
    await server.stop();
  });

  describe('complete pipeline validation', () => {
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
        sessionId: 'pipeline-e2e-test',
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

    it('[CRITICAL] should capture all core event types in single session', async () => {
      await page.goto(server.getUrl('/'));
      await waitForScriptReady(page, 5000);

      await pipelineManager.startRecording({
        sessionId: 'pipeline-e2e-test',
        recordingId: 'full-pipeline',
        onEntry: (entry) => {
          capturedEntries.push(entry);
        },
      });

      // 1. Click
      await page.click('#test-btn');
      await page.waitForTimeout(300);

      // 2. Input
      await page.type('#test-input', 'test');
      await page.waitForTimeout(500);

      // 3. Scroll (page has min-height: 200vh)
      await simulateRealScroll(page, 200);
      await page.waitForTimeout(700);

      await pipelineManager.stopRecording();

      // Validate all event types captured
      const clicks = capturedEntries.filter(e => getActionType(e) === ActionType.CLICK);
      const inputs = capturedEntries.filter(e => getActionType(e) === ActionType.INPUT);
      const scrolls = capturedEntries.filter(e => getActionType(e) === ActionType.SCROLL);

      expect(clicks.length).toBeGreaterThan(0);
      expect(inputs.length).toBeGreaterThan(0);
      expect(scrolls.length).toBeGreaterThan(0);

      // Validate TimelineEntry structure
      const entry = capturedEntries[0];
      if (!entry) {
        throw new Error('Expected at least one captured entry');
      }
      expect(entry.id).toBeDefined();
      expect(entry.sequenceNum).toBeDefined();
      expect(entry.timestamp).toBeDefined();
      expect(entry.action).toBeDefined();
    });

    it('[CRITICAL] should have clean state between test runs', async () => {
      // Reset stats to ensure clean state
      initializer.resetStats();

      const statsBefore = initializer.getInjectionStats();
      expect(statsBefore.attempted).toBe(0);
      expect(statsBefore.successful).toBe(0);

      await page.goto(server.getUrl('/'));
      await waitForScriptReady(page, 5000);

      const statsAfter = initializer.getInjectionStats();

      // Should have exactly 1 injection attempt for this test
      expect(statsAfter.attempted).toBe(1);
      expect(statsAfter.successful).toBe(1);
    });

    it('[CRITICAL] should capture navigation events', async () => {
      await page.goto(server.getUrl('/'));
      await waitForScriptReady(page, 5000);

      await pipelineManager.startRecording({
        sessionId: 'pipeline-e2e-test',
        recordingId: 'nav-test',
        onEntry: (entry) => {
          capturedEntries.push(entry);
        },
      });

      // Click navigation link
      await page.click('#test-link');
      await page.waitForURL('**/page-2', { timeout: 5000 });
      await page.waitForTimeout(500);

      await pipelineManager.stopRecording();

      // Validate navigation event was captured
      const navEvents = capturedEntries.filter(e => getActionType(e) === ActionType.NAVIGATE);
      const clickEvents = capturedEntries.filter(e => getActionType(e) === ActionType.CLICK);

      // Click that triggered navigation should be captured
      expect(clickEvents.length).toBeGreaterThan(0);

      // Navigation event should be captured
      expect(navEvents.length).toBeGreaterThan(0);

      // Navigation should contain the target URL (check both telemetry.url and top-level url)
      const navToPage2 = navEvents.find((entry) => {
        const telemetryUrl = getTelemetryUrl(entry);
        const entryUrl = getEntryUrl(entry);
        return telemetryUrl?.includes('/page-2') || entryUrl?.includes('/page-2');
      });
      expect(navToPage2).toBeDefined();
    });

    it('[CRITICAL] should continue capturing events after navigation', async () => {
      await page.goto(server.getUrl('/'));
      await waitForScriptReady(page, 5000);

      await pipelineManager.startRecording({
        sessionId: 'pipeline-e2e-test',
        recordingId: 'post-nav-test',
        onEntry: (entry) => {
          capturedEntries.push(entry);
        },
      });

      // Click before navigation
      await page.click('#test-btn');
      await page.waitForTimeout(200);

      // Navigate
      await page.click('#test-link');
      await page.waitForURL('**/page-2', { timeout: 5000 });
      await waitForScriptReady(page, 5000);
      await page.waitForTimeout(300);

      const entriesAfterNav = capturedEntries.length;

      // Click on new page
      await page.click('#page-2-btn');
      await page.waitForTimeout(500);

      await pipelineManager.stopRecording();

      // Should have captured events after navigation
      const postNavEntries = capturedEntries.slice(entriesAfterNav);
      const postNavClicks = postNavEntries.filter(e => getActionType(e) === ActionType.CLICK);

      // CRITICAL: Events after navigation must be captured
      expect(postNavClicks.length).toBeGreaterThan(0);
    });

    it('should maintain correct event sequence ordering', async () => {
      await page.goto(server.getUrl('/'));
      await waitForScriptReady(page, 5000);

      await pipelineManager.startRecording({
        sessionId: 'pipeline-e2e-test',
        recordingId: 'sequence-test',
        onEntry: (entry) => {
          capturedEntries.push(entry);
        },
      });

      // Perform multiple actions
      await page.click('#test-btn');
      await page.waitForTimeout(200);
      await page.type('#test-input', 'abc');
      await page.waitForTimeout(500);
      await simulateRealScroll(page, 100);
      await page.waitForTimeout(700);

      await pipelineManager.stopRecording();

      // Verify sequence numbers are ascending
      const sequenceNums = capturedEntries
        .map(e => e.sequenceNum)
        .filter((n): n is number => n !== undefined);

      for (let i = 1; i < sequenceNums.length; i++) {
        const current = sequenceNums[i];
        const previous = sequenceNums[i - 1];
        if (current === undefined || previous === undefined) {
          throw new Error('Missing sequence number when validating ordering');
        }
        expect(current).toBeGreaterThan(previous);
      }

      // Verify timestamps are ascending (or equal for rapid events)
      const timestamps = capturedEntries
        .map(e => e.timestamp)
        .filter((t): t is NonNullable<typeof t> => t !== undefined)
        .map(t => Number(t.seconds) * 1000 + Number(t.nanos) / 1000000);

      for (let i = 1; i < timestamps.length; i++) {
        const current = timestamps[i];
        const previous = timestamps[i - 1];
        if (current === undefined || previous === undefined) {
          throw new Error('Missing timestamp when validating ordering');
        }
        expect(current).toBeGreaterThanOrEqual(previous);
      }
    });
  });

  describe('state management', () => {
    let context: BrowserContext;
    let page: Page;
    let initializer: RecordingContextInitializer;
    let pipelineManager: RecordingPipelineManager;

    beforeEach(async () => {
      context = await browser.newContext();
      initializer = createRecordingContextInitializer({});
      await initializer.initialize(context);
      page = await context.newPage();

      pipelineManager = createRecordingPipelineManager(page, context, initializer, {
        sessionId: 'state-test',
      });
      await pipelineManager.initialize();
    });

    afterEach(async () => {
      if (pipelineManager.isRecording()) {
        await pipelineManager.stopRecording();
      }
      await context.close();
    });

    it('should reset injection stats correctly', async () => {
      // First navigation to trigger injection
      await page.goto(server.getUrl('/'));
      await waitForScriptReady(page, 5000);

      const statsAfterFirst = initializer.getInjectionStats();
      expect(statsAfterFirst.attempted).toBeGreaterThan(0);
      expect(statsAfterFirst.successful).toBeGreaterThan(0);

      // Reset stats
      initializer.resetStats();

      const statsAfterReset = initializer.getInjectionStats();
      expect(statsAfterReset.attempted).toBe(0);
      expect(statsAfterReset.successful).toBe(0);
      expect(statsAfterReset.failed).toBe(0);
      expect(statsAfterReset.avgInjectionTimeMs).toBe(0);
      expect(statsAfterReset.lastInjectionAt).toBeNull();
    });

    it('should reset route handler stats correctly', async () => {
      await page.goto(server.getUrl('/'));
      await waitForScriptReady(page, 5000);

      // Start recording and generate some events
      const capturedEntries: TimelineEntry[] = [];
      await pipelineManager.startRecording({
        sessionId: 'state-test',
        recordingId: 'route-stats-test',
        onEntry: (entry) => {
          capturedEntries.push(entry);
        },
      });

      await page.click('#test-btn');
      await page.waitForTimeout(300);

      await pipelineManager.stopRecording();

      const routeStatsAfterEvents = initializer.getRouteHandlerStats();
      expect(routeStatsAfterEvents.eventsReceived).toBeGreaterThan(0);

      // Reset stats
      initializer.resetStats();

      const routeStatsAfterReset = initializer.getRouteHandlerStats();
      expect(routeStatsAfterReset.eventsReceived).toBe(0);
      expect(routeStatsAfterReset.eventsProcessed).toBe(0);
      expect(routeStatsAfterReset.eventsDroppedNoHandler).toBe(0);
      expect(routeStatsAfterReset.eventsWithErrors).toBe(0);
    });
  });
});
