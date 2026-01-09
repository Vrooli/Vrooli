/**
 * Integration Tests for Recording Script Injection
 *
 * These tests verify the complete injection and event flow using a real browser.
 * They ensure that:
 * - Script injection works correctly
 * - Events are captured and transmitted
 * - History API navigation is captured (proves MAIN context)
 * - Verification functions work correctly
 *
 * NOTE: These tests use a local HTTP server because:
 * - Route interception only works for actual network requests
 * - data: URLs are not intercepted by Playwright's route patterns
 * - page.setContent() does not trigger route interception
 */

import { describe, beforeAll, afterAll, beforeEach, afterEach, it, expect } from '@jest/globals';
import { chromium, Browser, BrowserContext, Page } from 'rebrowser-playwright';
import * as http from 'http';
import {
  createRecordingContextInitializer,
  RecordingContextInitializer,
} from '../../src/recording/context-initializer';
import {
  verifyScriptInjection,
  assertScriptInjected,
  waitForScriptReady,
} from '../../src/recording/verification';
import type { RawBrowserEvent } from '../../src/recording/types';

// Increase timeout for browser operations
jest.setTimeout(30000);

/**
 * Simple HTTP server for testing that serves HTML pages
 */
class TestServer {
  private server: http.Server | null = null;
  private port = 0;
  private pages: Map<string, string> = new Map();

  async start(): Promise<number> {
    return new Promise((resolve) => {
      this.server = http.createServer((req, res) => {
        const path = req.url || '/';
        const html = this.pages.get(path) || '<html><head></head><body>404</body></html>';
        res.writeHead(200, { 'Content-Type': 'text/html' });
        res.end(html);
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

  setPage(path: string, html: string): void {
    this.pages.set(path, html);
  }

  getUrl(path: string = '/'): string {
    return `http://localhost:${this.port}${path}`;
  }
}

describe('Recording Script Injection (Integration)', () => {
  let browser: Browser;
  let server: TestServer;

  beforeAll(async () => {
    browser = await chromium.launch({ headless: true });
    server = new TestServer();
    await server.start();
  });

  afterAll(async () => {
    await browser.close();
    await server.stop();
  });

  describe('script injection verification', () => {
    let context: BrowserContext;
    let page: Page;
    let initializer: RecordingContextInitializer;

    beforeEach(async () => {
      context = await browser.newContext();
      initializer = createRecordingContextInitializer({});
      await initializer.initialize(context);
      page = await context.newPage();
    });

    afterEach(async () => {
      await context.close();
    });

    it('should inject recording script into HTML pages', async () => {
      server.setPage('/test-basic', '<html><head></head><body>Test</body></html>');
      await page.goto(server.getUrl('/test-basic'));

      // Wait for script to be ready
      const verification = await waitForScriptReady(page, 5000);

      expect(verification.loaded).toBe(true);
      expect(verification.ready).toBe(true);
    });

    it('should set correct verification markers', async () => {
      server.setPage('/test-markers', '<html><head></head><body>Test</body></html>');
      await page.goto(server.getUrl('/test-markers'));

      const verification = await verifyScriptInjection(page);

      expect(verification.loaded).toBe(true);
      expect(verification.loadTime).toBeGreaterThan(0);
      expect(verification.version).toBeDefined();
      expect(verification.handlersCount).toBeGreaterThan(0);
    });

    it('should verify script runs in MAIN context', async () => {
      server.setPage('/test-main-context', '<html><head></head><body>Test</body></html>');
      await page.goto(server.getUrl('/test-main-context'));

      const verification = await verifyScriptInjection(page);

      // The script context marker proves it's in MAIN context
      expect(verification.inMainContext).toBe(true);
    });

    it('should assertScriptInjected throw when script not loaded', async () => {
      // Navigate to about:blank (no injection)
      await page.goto('about:blank');

      // Should throw because script is not loaded
      await expect(assertScriptInjected(page)).rejects.toThrow(/not loaded/i);
    });

    it('should handle pages without <head> tag', async () => {
      server.setPage('/test-no-head', '<body>No head tag</body>');
      await page.goto(server.getUrl('/test-no-head'));

      const verification = await waitForScriptReady(page, 5000);

      // Script should still be injected (via prepend or doctype fallback)
      expect(verification.loaded).toBe(true);
    });
  });

  describe('event capture flow', () => {
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

      // Set up event handler
      initializer.setEventHandler((event) => {
        capturedEvents.push(event);
      });
    });

    afterEach(async () => {
      initializer.clearEventHandler();
      await context.close();
    });

    it('should capture click events', async () => {
      server.setPage(
        '/test-click',
        `<html>
          <head></head>
          <body>
            <button id="test-btn">Click Me</button>
          </body>
        </html>`
      );
      await page.goto(server.getUrl('/test-click'));

      // Wait for script to be ready
      await waitForScriptReady(page, 5000);

      // Click the button
      await page.click('#test-btn');

      // Wait for event to propagate
      await page.waitForTimeout(200);

      // Should have captured a click event
      const clickEvents = capturedEvents.filter((e) => e.actionType === 'click');
      expect(clickEvents.length).toBeGreaterThan(0);
    });

    it('should capture input events', async () => {
      server.setPage(
        '/test-input',
        `<html>
          <head></head>
          <body>
            <input type="text" id="test-input" />
          </body>
        </html>`
      );
      await page.goto(server.getUrl('/test-input'));

      await waitForScriptReady(page, 5000);

      // Type in the input
      await page.fill('#test-input', 'test input');

      // Wait for debounced event
      await page.waitForTimeout(700);

      // Should have captured input events
      const inputEvents = capturedEvents.filter(
        (e) => e.actionType === 'type' || e.actionType === 'input'
      );
      expect(inputEvents.length).toBeGreaterThan(0);
    });

    it('should capture History API navigation (proves MAIN context)', async () => {
      server.setPage(
        '/test-history',
        `<html>
          <head></head>
          <body>
            <button id="nav-btn" onclick="history.pushState({}, '', '/new-path')">Navigate</button>
          </body>
        </html>`
      );
      await page.goto(server.getUrl('/test-history'));

      await waitForScriptReady(page, 5000);

      // Click button that triggers pushState
      await page.click('#nav-btn');

      // Wait for event to propagate
      await page.waitForTimeout(200);

      // Should have captured a navigation event
      // This is the critical test - if History API wrapping didn't work
      // (i.e., if script ran in ISOLATED context), this would fail
      const navEvents = capturedEvents.filter((e) => e.actionType === 'navigate');
      expect(navEvents.length).toBeGreaterThan(0);

      // Verify it captured the correct URL
      const navEvent = navEvents[navEvents.length - 1];
      expect(navEvent.payload?.targetUrl).toContain('/new-path');
    });
  });

  describe('idempotency', () => {
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

    it('should handle multiple page loads without duplicate handlers', async () => {
      server.setPage(
        '/test-multi-1',
        `<html>
          <head></head>
          <body>
            <button id="btn">Click</button>
          </body>
        </html>`
      );
      server.setPage(
        '/test-multi-2',
        `<html>
          <head></head>
          <body>
            <button id="btn2">Click 2</button>
          </body>
        </html>`
      );

      await page.goto(server.getUrl('/test-multi-1'));
      await waitForScriptReady(page, 5000);

      // Navigate to another page (triggers re-injection)
      await page.goto(server.getUrl('/test-multi-2'));
      await waitForScriptReady(page, 5000);

      // Click button
      await page.click('#btn2');
      await page.waitForTimeout(200);

      // Should only get one click event, not duplicates
      const clickEvents = capturedEvents.filter((e) => e.actionType === 'click');
      expect(clickEvents.length).toBe(1);
    });

    it('should properly clean up handlers on re-injection', async () => {
      server.setPage(
        '/test-cleanup-1',
        `<html>
          <head></head>
          <body>
            <button id="btn">Click</button>
          </body>
        </html>`
      );
      server.setPage(
        '/test-cleanup-2',
        `<html>
          <head></head>
          <body>
            <button id="btn2">Click 2</button>
          </body>
        </html>`
      );

      await page.goto(server.getUrl('/test-cleanup-1'));
      const verification1 = await waitForScriptReady(page, 5000);
      const handlersCount1 = verification1.handlersCount;

      // Navigate to another page (triggers re-injection)
      await page.goto(server.getUrl('/test-cleanup-2'));
      const verification2 = await waitForScriptReady(page, 5000);
      const handlersCount2 = verification2.handlersCount;

      // Handler count should be the same (not doubled)
      expect(handlersCount2).toBe(handlersCount1);
    });
  });

  describe('injection statistics', () => {
    let context: BrowserContext;
    let page: Page;
    let initializer: RecordingContextInitializer;

    beforeEach(async () => {
      context = await browser.newContext();
      initializer = createRecordingContextInitializer({ diagnosticsEnabled: true });
      await initializer.initialize(context);
      page = await context.newPage();
    });

    afterEach(async () => {
      await context.close();
    });

    it('should track successful injections', async () => {
      server.setPage('/test-stats-1', '<html><head></head><body>Test</body></html>');
      await page.goto(server.getUrl('/test-stats-1'));
      await waitForScriptReady(page, 5000);

      const stats = initializer.getInjectionStats();

      expect(stats.successful).toBeGreaterThan(0);
      expect(stats.methods.head).toBeGreaterThan(0);
    });

    it('should accumulate stats across multiple navigations', async () => {
      server.setPage('/test-stats-nav-1', '<html><head></head><body>Page 1</body></html>');
      server.setPage('/test-stats-nav-2', '<html><head></head><body>Page 2</body></html>');

      await page.goto(server.getUrl('/test-stats-nav-1'));
      await waitForScriptReady(page, 5000);

      await page.goto(server.getUrl('/test-stats-nav-2'));
      await waitForScriptReady(page, 5000);

      const stats = initializer.getInjectionStats();

      expect(stats.successful).toBeGreaterThanOrEqual(2);
    });
  });
});
