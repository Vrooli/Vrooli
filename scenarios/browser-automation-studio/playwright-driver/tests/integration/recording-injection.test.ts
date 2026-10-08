import { generateActivationScript } from '../../src/recording/capture/init-script-generator';
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

import { chromium, Browser, BrowserContext, Page } from 'rebrowser-playwright';
import * as http from 'http';
import {
  createRecordingContextInitializer,
  RecordingContextInitializer,
  verifyScriptInjection,
  assertScriptInjected,
  waitForScriptReady,
  generateDeactivationScript,
} from '../../src/recording';
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
      // Use a fresh browser to avoid init-script leakage across contexts
      const isolatedBrowser = await chromium.launch({ headless: true });
      const rawContext = await isolatedBrowser.newContext();
      const rawPage = await rawContext.newPage();
      try {
        await rawPage.goto('about:blank');
        // Should throw because script is not loaded
        await expect(assertScriptInjected(rawPage)).rejects.toThrow(/not loaded/i);
      } finally {
        await rawContext.close();
        await isolatedBrowser.close();
      }
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
      await page.evaluate(generateActivationScript('injection-fixture'));

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
      await page.evaluate(generateActivationScript('injection-fixture'));

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

    it('[REQ:BAS-RH-J02] preserves final snapshots across replacement, deletion, paste, composition, and clear', async () => {
      server.setPage(
        '/test-j02-input-snapshots',
        `<html><head></head><body>
          <input type="text" id="test-input" value="initial" />
          <script>
            window.fixtureInputLog = [];
            document.querySelector('#test-input').addEventListener('input', event => {
              window.fixtureInputLog.push({ value: event.target.value, inputType: event.inputType });
            });
          </script>
        </body></html>`
      );
      await page.goto(server.getUrl('/test-j02-input-snapshots'));
      await waitForScriptReady(page, 5000);
      await context.grantPermissions(['clipboard-read', 'clipboard-write'], { origin: new URL(page.url()).origin });
      await page.evaluate(generateActivationScript('j02-input-snapshot-fixture'));

      const pauseForSnapshot = () => page.waitForTimeout(700);
      await page.fill('#test-input', 'typed before a pause');
      await pauseForSnapshot();

      await page.locator('#test-input').click();
      await page.keyboard.press('Control+A');
      await page.keyboard.type('replacement');
      await pauseForSnapshot();

      await page.keyboard.press('Home');
      await page.keyboard.press('Shift+ArrowRight');
      await page.keyboard.press('Shift+ArrowRight');
      await page.keyboard.press('Backspace');
      await pauseForSnapshot();

      await page.evaluate(async () => navigator.clipboard.writeText('pasted value'));
      await page.keyboard.press('Control+A');
      await page.keyboard.press('Control+V');
      await pauseForSnapshot();

      // Chromium automation cannot supply a native OS IME. Exercise the browser
      // composition/input commit path explicitly and keep that limitation clear.
      await page.evaluate(() => {
        const input = document.querySelector<HTMLInputElement>('#test-input');
        if (!input) throw new Error('fixture input is missing');
        input.focus();
        input.value = '東京';
        input.dispatchEvent(new CompositionEvent('compositionstart', { bubbles: true, data: '' }));
        input.dispatchEvent(new CompositionEvent('compositionupdate', { bubbles: true, data: '東京' }));
        input.dispatchEvent(new InputEvent('input', {
          bubbles: true, data: '東京', inputType: 'insertCompositionText', isComposing: true,
        }));
        input.dispatchEvent(new CompositionEvent('compositionend', { bubbles: true, data: '東京' }));
        input.dispatchEvent(new InputEvent('input', {
          bubbles: true, data: '東京', inputType: 'insertFromComposition', isComposing: false,
        }));
      });
      await pauseForSnapshot();

      await page.fill('#test-input', '');
      await page.evaluate(generateDeactivationScript());
      await page.waitForFunction(() => {
        const telemetry = (window as Window & {
          __vrooli_recording_telemetry?: { eventsCaptured: number; eventsSent: number; eventsSendSuccess: number };
        }).__vrooli_recording_telemetry;
        return Boolean(telemetry && telemetry.eventsCaptured >= 6 && telemetry.eventsSent === telemetry.eventsSendSuccess);
      });

      const fixtureValues = await page.evaluate(() => (window as Window & {
        fixtureInputLog?: Array<{ value: string; inputType: string }>;
      }).fixtureInputLog?.map(entry => entry.value) ?? []);
      const snapshots = capturedEvents
        .filter((event) => event.actionType === 'type')
        .map((event) => event.payload?.text);
      for (const expected of ['typed before a pause', 'replacement', 'placement', 'pasted value', '東京', '']) {
        expect(fixtureValues).toContain(expected);
        expect(snapshots).toContain(expected);
      }
      expect(snapshots.at(-1)).toBe('');
    });

    it('records the final empty input value when recording stops', async () => {
      server.setPage(
        '/test-empty-input-stop',
        '<html><head></head><body><input type="text" id="test-input" value="initial" /></body></html>'
      );
      await page.goto(server.getUrl('/test-empty-input-stop'));
      await waitForScriptReady(page, 5000);
      await page.evaluate(generateActivationScript('empty-input-stop-fixture'));

      await page.fill('#test-input', 'temporary');
      await page.fill('#test-input', '');
      await page.evaluate(generateDeactivationScript());

      const typeEvents = capturedEvents.filter((event) => event.actionType === 'type');
      expect(typeEvents).toHaveLength(1);
      expect(typeEvents[0]?.payload?.text).toBe('');
    });

    it('flushes a debounced input edit before acknowledging recording stop', async () => {
      server.setPage(
        '/test-pending-input-stop',
        '<html><head></head><body><input type="text" id="test-input" /></body></html>'
      );
      await page.goto(server.getUrl('/test-pending-input-stop'));
      await waitForScriptReady(page, 5000);
      await page.evaluate(generateActivationScript('pending-input-stop-fixture'));

      await page.fill('#test-input', 'captured before debounce');
      await page.evaluate(generateDeactivationScript());

      const typeEvents = capturedEvents.filter((event) => event.actionType === 'type');
      expect(typeEvents).toHaveLength(1);
      expect(typeEvents[0]?.payload?.text).toBe('captured before debounce');
    });

    it('never sends a password value in passive recording events', async () => {
      const secret = 'BAS_SYNTHETIC_PASSWORD_SENTINEL_9f52';
      const attributeSecret = 'BAS_SYNTHETIC_PASSWORD_ATTRIBUTE_SENTINEL_31ac';
      const otpSecret = 'BAS_SYNTHETIC_OTP_SENTINEL_d821';
      server.setPage(
        '/test-password-redaction',
        `<html><head></head><body>
          <label for="password">Password</label>
          <input type="password" id="password" placeholder="Enter password" value="${attributeSecret}" />
          <label for="otp">One time code</label>
          <input type="text" id="otp" autocomplete="one-time-code" />
          <button id="outside">Continue</button>
        </body></html>`
      );
      await page.goto(server.getUrl('/test-password-redaction'));
      await waitForScriptReady(page, 5000);
      await page.evaluate(generateActivationScript('password-redaction-fixture'));

      await page.click('#password');
      await page.fill('#password', secret);
      await page.fill('#otp', otpSecret);
      await page.click('#outside');
      await page.evaluate(() => window.dispatchEvent(new Event('beforeunload')));

      await page.waitForFunction(() => {
        const telemetry = (
          window as Window & {
            __vrooli_recording_telemetry?: {
              eventsCaptured: number;
              eventsSent: number;
              eventsSendSuccess: number;
            };
          }
        ).__vrooli_recording_telemetry;
        return Boolean(
          telemetry &&
          telemetry.eventsCaptured >= 2 &&
          telemetry.eventsSent === telemetry.eventsSendSuccess
        );
      });

      expect(capturedEvents.some((event) => event.actionType === 'click')).toBe(true);
      expect(JSON.stringify(capturedEvents)).not.toContain(secret);
      expect(JSON.stringify(capturedEvents)).not.toContain(attributeSecret);
      expect(JSON.stringify(capturedEvents)).not.toContain(otpSecret);
      const pendingEvents = await page.evaluate(
        () => sessionStorage.getItem('__vrooli_pending_events__') || ''
      );
      expect(pendingEvents).not.toContain(secret);
      expect(pendingEvents).not.toContain(attributeSecret);
      expect(pendingEvents).not.toContain(otpSecret);
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
      await page.evaluate(generateActivationScript('injection-fixture'));

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
      if (!navEvent) {
        throw new Error('Expected a navigation event');
      }
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
      await page.evaluate(generateActivationScript('injection-fixture'));

      // Navigate to another page (triggers re-injection)
      await page.goto(server.getUrl('/test-multi-2'));
      await waitForScriptReady(page, 5000);
      await page.evaluate(generateActivationScript('injection-fixture'));

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

      expect(stats.attempted).toBeGreaterThan(0);
      expect(stats.successful).toBeGreaterThan(0);
      expect(stats.lastInjectionAt).not.toBeNull();
    });

    it('should accumulate stats across multiple pages', async () => {
      server.setPage('/test-stats-nav-1', '<html><head></head><body>Page 1</body></html>');
      server.setPage('/test-stats-nav-2', '<html><head></head><body>Page 2</body></html>');

      initializer.resetStats();

      const firstPage = await context.newPage();
      await firstPage.goto(server.getUrl('/test-stats-nav-1'));
      await waitForScriptReady(firstPage, 5000);

      const secondPage = await context.newPage();
      await secondPage.goto(server.getUrl('/test-stats-nav-2'));
      await waitForScriptReady(secondPage, 5000);

      const stats = initializer.getInjectionStats();

      expect(stats.successful).toBeGreaterThanOrEqual(2);
    });
  });
});
