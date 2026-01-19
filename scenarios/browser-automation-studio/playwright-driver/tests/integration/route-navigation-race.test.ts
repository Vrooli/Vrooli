/**
 * Route Loss During Navigation Tests
 *
 * These tests verify that recording events are not lost when navigation
 * occurs immediately after a user action (click, input).
 *
 * ROOT CAUSE CONTEXT:
 * The timeline bug (navigation events appear but clicks/inputs don't) can be caused
 * by rebrowser-playwright's page.route() handlers not persisting across navigation.
 * When a click triggers navigation, the event's fetch may complete after the route
 * is cleared, causing the event to be lost.
 *
 * These tests:
 * 1. Verify click events are captured even when they trigger navigation
 * 2. Verify event capture continues to work after navigation
 * 3. Test rapid navigation sequences
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
import type { RawBrowserEvent } from '../../src/recording';
import { ActionType } from '../../src/proto/recording';

// Helper to get action type from TimelineEntry (protobuf structure)
// NOTE: The property is `action.type`, not `definition.type`
function getActionType(entry: TimelineEntry): ActionType | undefined {
  return entry.action?.type;
}

// Increase timeout for browser operations with navigation
jest.setTimeout(60000);

/**
 * Test server for navigation scenarios.
 */
class NavigationTestServer {
  private server: http.Server | null = null;
  private port = 0;

  async start(): Promise<number> {
    return new Promise((resolve) => {
      this.server = http.createServer((req, res) => {
        const path = req.url || '/';

        if (path === '/page-a') {
          res.writeHead(200, { 'Content-Type': 'text/html' });
          res.end(`
            <!DOCTYPE html>
            <html>
            <head></head>
            <body>
              <h1>Page A</h1>
              <a id="nav-link" href="/page-b">Go to Page B (link)</a>
              <button id="nav-btn" onclick="window.location.href='/page-b'">Go to Page B (JS)</button>
              <button id="regular-btn">Regular Button</button>
            </body>
            </html>
          `);
          return;
        }

        if (path === '/page-b') {
          res.writeHead(200, { 'Content-Type': 'text/html' });
          res.end(`
            <!DOCTYPE html>
            <html>
            <head></head>
            <body>
              <h1>Page B</h1>
              <button id="btn-b">Button on Page B</button>
              <a id="back-link" href="/page-a">Back to Page A</a>
            </body>
            </html>
          `);
          return;
        }

        if (path === '/page-c') {
          res.writeHead(200, { 'Content-Type': 'text/html' });
          res.end(`
            <!DOCTYPE html>
            <html>
            <head></head>
            <body>
              <h1>Page C</h1>
              <button id="btn-c">Button on Page C</button>
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

describe('Route Loss During Navigation (Integration)', () => {
  let browser: Browser;
  let server: NavigationTestServer;

  beforeAll(async () => {
    browser = await chromium.launch({ headless: true });
    server = new NavigationTestServer();
    await server.start();
  });

  afterAll(async () => {
    await browser.close();
    await server.stop();
  });

  describe('click events that trigger navigation', () => {
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
        sessionId: 'test-session',
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

    it('[CRITICAL] should capture click event that triggers link navigation', async () => {
      await page.goto(server.getUrl('/page-a'));
      await waitForScriptReady(page, 5000);

      // Start recording
      await pipelineManager.startRecording({
        sessionId: 'test-session',
        recordingId: 'test-recording-1',
        onEntry: (entry) => {
          capturedEntries.push(entry);
        },
      });

      // Get telemetry before click
      const telemetryBefore = await page.evaluate(() => {
        return (window as any).__vrooli_recording_telemetry || { eventsDetected: 0 };
      });

      // Click the link that navigates to page B
      await page.click('#nav-link');

      // Wait for navigation to complete
      await page.waitForURL('**/page-b', { timeout: 5000 });
      await page.waitForTimeout(500);

      // Get telemetry after navigation
      const telemetryAfter = await page.evaluate(() => {
        return (window as any).__vrooli_recording_telemetry || { eventsDetected: 0 };
      });

      // Log captured entries for debugging
      console.log('Captured entries:', capturedEntries.map((e) => ({
        actionType: getActionType(e),
        sequenceNum: e.sequenceNum,
      })));

      // CRITICAL ASSERTION: The click event must be captured
      const clickEvents = capturedEntries.filter((e) => getActionType(e) === ActionType.CLICK);
      const navEvents = capturedEntries.filter((e) => getActionType(e) === ActionType.NAVIGATE);

      // Check if events were detected by browser script
      console.log('Browser telemetry:', {
        before: telemetryBefore.eventsDetected,
        after: telemetryAfter.eventsDetected,
      });

      // If no click events captured but navigation events appear, the bug is reproduced
      if (clickEvents.length === 0 && navEvents.length > 0) {
        console.warn(
          'BUG REPRODUCED: Navigation events captured but click event that triggered ' +
            'navigation was lost. This is likely due to route being cleared during navigation.'
        );
      }

      // The click that triggered navigation must be captured
      expect(clickEvents.length).toBeGreaterThan(0);

      // At minimum, we should have navigation events
      expect(navEvents.length).toBeGreaterThan(0);
    });

    it('[CRITICAL] should capture button click that triggers JS navigation', async () => {
      await page.goto(server.getUrl('/page-a'));
      await waitForScriptReady(page, 5000);

      await pipelineManager.startRecording({
        sessionId: 'test-session',
        recordingId: 'test-recording-2',
        onEntry: (entry) => {
          capturedEntries.push(entry);
        },
      });

      // Click button that triggers navigation via JavaScript
      await page.click('#nav-btn');

      // Wait for navigation
      await page.waitForURL('**/page-b', { timeout: 5000 });
      await page.waitForTimeout(500);

      const clickEvents = capturedEntries.filter((e) => getActionType(e) === ActionType.CLICK);
      const navEvents = capturedEntries.filter((e) => getActionType(e) === ActionType.NAVIGATE);

      console.log('JS Navigation test - captured:', {
        clicks: clickEvents.length,
        navigations: navEvents.length,
      });

      // The button click event should be captured even though it triggered navigation
      expect(clickEvents.length).toBeGreaterThan(0);

      // At minimum, we should have navigation events
      expect(navEvents.length).toBeGreaterThan(0);
    });

    it('should maintain event capture after navigation completes', async () => {
      await page.goto(server.getUrl('/page-a'));
      await waitForScriptReady(page, 5000);

      await pipelineManager.startRecording({
        sessionId: 'test-session',
        recordingId: 'test-recording-3',
        onEntry: (entry) => {
          capturedEntries.push(entry);
        },
      });

      // Navigate to page B
      await page.click('#nav-link');
      await page.waitForURL('**/page-b', { timeout: 5000 });

      // Wait for route re-registration and script injection
      await waitForScriptReady(page, 5000);
      await page.waitForTimeout(300);

      // Clear previous entries to test new page capture
      const entriesBeforePageBClick = capturedEntries.length;

      // Click button on page B
      await page.click('#btn-b');
      await page.waitForTimeout(500);

      // Should have captured click on page B
      const newEntries = capturedEntries.slice(entriesBeforePageBClick);
      const pageBClicks = newEntries.filter((e) => getActionType(e) === ActionType.CLICK);

      console.log('Post-navigation capture test:', {
        entriesBeforeClick: entriesBeforePageBClick,
        totalEntries: capturedEntries.length,
        newClicks: pageBClicks.length,
      });

      // Events on the new page should be captured
      expect(pageBClicks.length).toBeGreaterThan(0);
    });

    it('should verify route handler is active after navigation', async () => {
      await page.goto(server.getUrl('/page-a'));
      await waitForScriptReady(page, 5000);

      await pipelineManager.startRecording({
        sessionId: 'test-session',
        recordingId: 'test-recording-4',
        onEntry: () => {},
      });

      const statsBeforeNav = initializer.getRouteHandlerStats();

      // Navigate
      await page.goto(server.getUrl('/page-b'));
      await waitForScriptReady(page, 5000);

      // Perform a click to test route
      await page.click('#btn-b');
      await page.waitForTimeout(500);

      const statsAfterClick = initializer.getRouteHandlerStats();

      console.log('Route handler stats:', {
        beforeNav: statsBeforeNav,
        afterClick: statsAfterClick,
      });

      // Route should have received events after navigation
      expect(statsAfterClick.eventsReceived).toBeGreaterThan(0);
    });
  });

  describe('rapid navigation sequences', () => {
    let context: BrowserContext;
    let page: Page;
    let initializer: RecordingContextInitializer;
    let capturedEvents: RawBrowserEvent[];

    beforeEach(async () => {
      context = await browser.newContext();
      initializer = createRecordingContextInitializer({});
      await initializer.initialize(context);
      page = await context.newPage();
      capturedEvents = [];
      initializer.setEventHandler((event) => {
        capturedEvents.push(event);
      });
    });

    afterEach(async () => {
      initializer.clearEventHandler();
      await context.close();
    });

    it('should handle multiple rapid navigations without losing events', async () => {
      await page.goto(server.getUrl('/page-a'));
      await waitForScriptReady(page, 5000);

      // Rapid navigation sequence
      await page.goto(server.getUrl('/page-b'));
      await page.waitForTimeout(100);
      await page.goto(server.getUrl('/page-c'));
      await page.waitForTimeout(100);
      await page.goto(server.getUrl('/page-a'));
      await page.waitForTimeout(100);
      await page.goto(server.getUrl('/page-b'));

      await waitForScriptReady(page, 5000);

      // Clear events from navigations
      capturedEvents.length = 0;

      // Verify recording still works after rapid navigations
      await page.click('#btn-b');
      await page.waitForTimeout(500);

      const clickEvents = capturedEvents.filter((e) => e.actionType === 'click'); // RawBrowserEvent uses string

      console.log('Rapid navigation test - clicks captured:', clickEvents.length);

      // Should still be able to capture events after rapid navigations
      expect(clickEvents.length).toBeGreaterThan(0);
    });

    it('should handle back/forward navigation', async () => {
      await page.goto(server.getUrl('/page-a'));
      await waitForScriptReady(page, 5000);

      // Navigate forward
      await page.goto(server.getUrl('/page-b'));
      await page.waitForTimeout(200);

      // Go back
      await page.goBack();
      await page.waitForTimeout(200);

      // Go forward
      await page.goForward();
      await waitForScriptReady(page, 5000);

      // Clear events
      capturedEvents.length = 0;

      // Should still capture events
      await page.click('#btn-b');
      await page.waitForTimeout(500);

      const clickEvents = capturedEvents.filter((e) => e.actionType === 'click'); // RawBrowserEvent uses string

      console.log('Back/forward navigation test - clicks captured:', clickEvents.length);

      expect(clickEvents.length).toBeGreaterThan(0);
    });
  });

  describe('event ordering during navigation', () => {
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
        sessionId: 'test-session',
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

    it('should preserve correct event ordering (click before navigate)', async () => {
      await page.goto(server.getUrl('/page-a'));
      await waitForScriptReady(page, 5000);

      await pipelineManager.startRecording({
        sessionId: 'test-session',
        recordingId: 'test-ordering',
        onEntry: (entry) => {
          capturedEntries.push(entry);
        },
      });

      // Click a regular button first
      await page.click('#regular-btn');
      await page.waitForTimeout(200);

      // Then click the navigation link
      await page.click('#nav-link');
      await page.waitForURL('**/page-b', { timeout: 5000 });
      await page.waitForTimeout(500);

      const clicks = capturedEntries.filter((e) => getActionType(e) === ActionType.CLICK);
      const navs = capturedEntries.filter((e) => getActionType(e) === ActionType.NAVIGATE);

      console.log('Event ordering:', capturedEntries.map((e) => ({
        type: getActionType(e),
        seq: e.sequenceNum,
      })));

      // If we have both click and navigate events, verify ordering
      if (clicks.length >= 2 && navs.length > 0) {
        // The last click (nav link) should come before the navigation
        const lastClick = clicks[clicks.length - 1];
        const navToPageB = navs.find((n) => n.url?.includes('/page-b'));

        if (lastClick && navToPageB) {
          expect(lastClick.sequenceNum).toBeLessThan(navToPageB.sequenceNum);
        }
      }
    });
  });
});
