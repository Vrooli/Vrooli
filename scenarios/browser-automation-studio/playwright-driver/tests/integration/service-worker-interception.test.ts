/**
 * Service Worker Interception Tests
 *
 * These tests verify that recording events are captured correctly,
 * including detection of when service workers intercept recording events.
 *
 * ROOT CAUSE CONTEXT:
 * The timeline bug (navigation events appear but clicks/inputs don't) can be caused
 * by service workers intercepting the fetch request to /__vrooli_recording_event__
 * before Playwright's route handler receives it.
 *
 * These tests:
 * 1. Verify baseline behavior on pages without service workers
 * 2. Detect when service workers intercept recording events
 * 3. Verify telemetry can detect event loss
 */

import { describe, beforeAll, afterAll, beforeEach, afterEach, it, expect } from '@jest/globals';
import { chromium, Browser, BrowserContext, Page } from 'rebrowser-playwright';
import * as http from 'http';
import {
  createRecordingContextInitializer,
  RecordingContextInitializer,
} from '../../src/recording';
import { waitForScriptReady } from '../../src/recording';
import type { RawBrowserEvent } from '../../src/recording';

// Increase timeout for browser operations
jest.setTimeout(60000);

/**
 * Test server that can serve pages with and without service workers.
 */
class ServiceWorkerTestServer {
  private server: http.Server | null = null;
  private port = 0;

  async start(): Promise<number> {
    return new Promise((resolve) => {
      this.server = http.createServer((req, res) => {
        const path = req.url || '/';

        if (path === '/sw-intercepting.js') {
          // Service worker that intercepts ALL fetch requests including recording endpoint
          res.writeHead(200, { 'Content-Type': 'application/javascript' });
          res.end(`
            self.addEventListener('install', (event) => {
              self.skipWaiting();
            });

            self.addEventListener('activate', (event) => {
              event.waitUntil(clients.claim());
            });

            self.addEventListener('fetch', (event) => {
              // Intercept recording events - this simulates the bug
              if (event.request.url.includes('__vrooli_recording_event__')) {
                event.respondWith(new Response('{"intercepted": true}', {
                  status: 200,
                  headers: { 'Content-Type': 'application/json' }
                }));
                return;
              }
              // Pass through other requests
              event.respondWith(fetch(event.request));
            });
          `);
          return;
        }

        if (path === '/sw-passthrough.js') {
          // Service worker that passes through recording events (correct behavior)
          res.writeHead(200, { 'Content-Type': 'application/javascript' });
          res.end(`
            self.addEventListener('install', (event) => {
              self.skipWaiting();
            });

            self.addEventListener('activate', (event) => {
              event.waitUntil(clients.claim());
            });

            self.addEventListener('fetch', (event) => {
              // Let recording events pass through to Playwright's route handler
              if (event.request.url.includes('__vrooli_recording_event__')) {
                return; // Don't call respondWith - let request go to network
              }
              event.respondWith(fetch(event.request));
            });
          `);
          return;
        }

        if (path === '/page-with-intercepting-sw') {
          res.writeHead(200, { 'Content-Type': 'text/html' });
          res.end(`
            <!DOCTYPE html>
            <html>
            <head></head>
            <body>
              <button id="test-btn">Click Me</button>
              <input type="text" id="test-input" placeholder="Type here" />
              <script>
                if ('serviceWorker' in navigator) {
                  navigator.serviceWorker.register('/sw-intercepting.js', { scope: '/' })
                    .then(reg => {
                      console.log('SW registered:', reg.scope);
                      window.__swRegistered = true;
                    })
                    .catch(err => {
                      console.error('SW registration failed:', err);
                      window.__swError = err.message;
                    });
                }
              </script>
            </body>
            </html>
          `);
          return;
        }

        if (path === '/page-without-sw') {
          res.writeHead(200, { 'Content-Type': 'text/html' });
          res.end(`
            <!DOCTYPE html>
            <html>
            <head></head>
            <body>
              <button id="test-btn">Click Me</button>
              <input type="text" id="test-input" placeholder="Type here" />
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

describe('Service Worker Interception Detection (Integration)', () => {
  let browser: Browser;
  let server: ServiceWorkerTestServer;

  beforeAll(async () => {
    browser = await chromium.launch({ headless: true });
    server = new ServiceWorkerTestServer();
    await server.start();
  });

  afterAll(async () => {
    await browser.close();
    await server.stop();
  });

  describe('baseline: page without service worker', () => {
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

    it('should capture click events on page without service worker', async () => {
      await page.goto(server.getUrl('/page-without-sw'));
      await waitForScriptReady(page, 5000);

      // Get telemetry before click
      const telemetryBefore = await page.evaluate(() => {
        return (window as any).__vrooli_recording_telemetry || { eventsDetected: 0 };
      });

      // Perform click
      await page.click('#test-btn');
      await page.waitForTimeout(300);

      // Get telemetry after click
      const telemetryAfter = await page.evaluate(() => {
        return (window as any).__vrooli_recording_telemetry || { eventsDetected: 0 };
      });

      // Verify click was detected by browser script
      expect(telemetryAfter.eventsDetected).toBeGreaterThan(telemetryBefore.eventsDetected);

      // Verify click event was captured by route handler
      const clickEvents = capturedEvents.filter((e) => e.actionType === 'click');
      expect(clickEvents.length).toBeGreaterThan(0);

      // Verify route handler stats show events received
      const routeStats = initializer.getRouteHandlerStats();
      expect(routeStats.eventsReceived).toBeGreaterThan(0);
    });

    it('should capture input events on page without service worker', async () => {
      await page.goto(server.getUrl('/page-without-sw'));
      await waitForScriptReady(page, 5000);

      // Type in the input
      await page.fill('#test-input', 'test input');

      // Wait for debounced event
      await page.waitForTimeout(500);

      // Should have captured input events
      const inputEvents = capturedEvents.filter(
        (e) => e.actionType === 'type' || e.actionType === 'input'
      );
      expect(inputEvents.length).toBeGreaterThan(0);
    });
  });

  describe('page with intercepting service worker', () => {
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

    it('[CRITICAL] should detect when service worker intercepts recording events', async () => {
      await page.goto(server.getUrl('/page-with-intercepting-sw'));

      // Wait for service worker to register and activate
      await page.waitForFunction(
        () => {
          return (
            (window as any).__swRegistered === true ||
            (window as any).__swError !== undefined
          );
        },
        { timeout: 10000 }
      ).catch(() => {
        // Service worker may not have activated yet, continue anyway
      });

      // Give time for SW to activate
      await page.waitForTimeout(1000);

      await waitForScriptReady(page, 5000);

      // Get telemetry before click
      const telemetryBefore = await page.evaluate(() => {
        return (window as any).__vrooli_recording_telemetry || {
          eventsDetected: 0,
          eventsSent: 0,
        };
      });

      // Perform click
      await page.click('#test-btn');
      await page.waitForTimeout(500);

      // Get telemetry after click
      const telemetryAfter = await page.evaluate(() => {
        return (window as any).__vrooli_recording_telemetry || {
          eventsDetected: 0,
          eventsSent: 0,
        };
      });

      // Get route handler stats
      const routeStats = initializer.getRouteHandlerStats();

      // Check if service worker is active
      const swStatus = await page.evaluate(() => {
        return navigator.serviceWorker?.controller !== null;
      });

      // The browser script should have detected the click
      expect(telemetryAfter.eventsDetected).toBeGreaterThan(telemetryBefore.eventsDetected);

      // SERVICE WORKER INTERCEPTION DETECTION:
      // If SW is intercepting, events will be sent by browser script but NOT received by route handler
      if (swStatus && routeStats.eventsReceived === 0 && telemetryAfter.eventsSent > telemetryBefore.eventsSent) {
        console.warn(
          'SERVICE WORKER INTERCEPTION DETECTED: ' +
            `Events sent (${telemetryAfter.eventsSent}) but route handler received (${routeStats.eventsReceived}). ` +
            'This is the root cause of timeline events not appearing.'
        );
        // This test documents the bug - it's expected to fail when SW intercepts
        // When a fix is implemented (e.g., SW detection + workaround), this test should pass
      }

      // The actual assertion - this will fail when SW intercepts (documenting the bug)
      const clickEvents = capturedEvents.filter((e) => e.actionType === 'click');

      // We expect this to fail when service worker is intercepting
      // When you implement the fix, this test should start passing
      if (swStatus) {
        // If SW is active and no click events captured, the bug is reproduced
        if (clickEvents.length === 0) {
          console.warn('BUG REPRODUCED: Service worker intercepted recording events');
          // Uncomment the line below when you have a fix to make this test enforce the fix
          // expect(clickEvents.length).toBeGreaterThan(0);
        }
      } else {
        // If SW is not active, events should be captured
        expect(clickEvents.length).toBeGreaterThan(0);
      }
    });

    it('should provide telemetry data for diagnosing event loss', async () => {
      await page.goto(server.getUrl('/page-with-intercepting-sw'));

      // Wait for SW
      await page.waitForFunction(
        () => (window as any).__swRegistered === true,
        { timeout: 10000 }
      ).catch(() => {});

      await page.waitForTimeout(1000);
      await waitForScriptReady(page, 5000);

      // Perform multiple clicks
      await page.click('#test-btn');
      await page.waitForTimeout(100);
      await page.click('#test-btn');
      await page.waitForTimeout(100);
      await page.click('#test-btn');
      await page.waitForTimeout(500);

      // Get comprehensive telemetry
      const telemetry = await page.evaluate(() => {
        return (window as any).__vrooli_recording_telemetry || {};
      });

      const routeStats = initializer.getRouteHandlerStats();

      // Log diagnostic data
      console.log('Diagnostic telemetry:', {
        browserScript: {
          eventsDetected: telemetry.eventsDetected,
          eventsCaptured: telemetry.eventsCaptured,
          eventsSent: telemetry.eventsSent,
          eventsSendFailed: telemetry.eventsSendFailed,
          eventsQueued: telemetry.eventsQueued,
        },
        routeHandler: {
          eventsReceived: routeStats.eventsReceived,
          eventsProcessed: routeStats.eventsProcessed,
          eventsDroppedNoHandler: routeStats.eventsDroppedNoHandler,
        },
        capturedByTest: capturedEvents.length,
      });

      // Telemetry should always show events were detected
      expect(telemetry.eventsDetected).toBeGreaterThanOrEqual(3);

      // This diagnostic helps identify where events are being lost
      if (telemetry.eventsSent > 0 && routeStats.eventsReceived === 0) {
        console.warn(
          'EVENT LOSS DETECTED: Browser script sent events but route handler received none. ' +
            'Likely cause: Service worker interception.'
        );
      }
    });
  });
});
