import { chromium } from 'rebrowser-playwright';
import { applySafeAreaInsets } from '../../src/session/context-builder';
import { SessionManager } from '../../src/session/manager';
import { createTestConfig } from '../helpers';
import { randomUUID } from 'node:crypto';

// Run with REBROWSER_PATCHES_RUNTIME_FIX_MODE=0 for normal Playwright evaluation.
// This is Chromium CSS emulation evidence; it does not certify Safari/PWA chrome.
describe('mobile safe-area emulation across navigation', () => {
  it('retains actual env() insets through initial navigation and reload', async () => {
    const browser = await chromium.launch({ headless: true, args: ['--no-sandbox'] });
    try {
      const page = await browser.newPage({
        viewport: { width: 390, height: 844 },
        isMobile: true,
        hasTouch: true,
      });
      await page.route('http://safe-area.test/**', (route) =>
        route.fulfill({
          contentType: 'text/html',
          body: '<meta name="viewport" content="width=device-width,viewport-fit=cover"><div id="probe" style="padding-top:env(safe-area-inset-top);padding-bottom:env(safe-area-inset-bottom)"></div>',
        })
      );
      await applySafeAreaInsets(page, { width: 390, height: 844 });
      const read = () =>
        page.evaluate(() => {
          const style = getComputedStyle(document.getElementById('probe')!);
          return { top: style.paddingTop, bottom: style.paddingBottom };
        });
      await page.goto('http://safe-area.test/');
      expect(await read()).toEqual({ top: '59px', bottom: '34px' });
      await page.reload();
      expect(await read()).toEqual({ top: '59px', bottom: '34px' });
    } finally {
      await browser.close();
    }
  }, 30000);
  it('retains insets in the production session creation path', async () => {
    const manager = new SessionManager(createTestConfig());
    try {
      const { sessionId, actualViewport } = await manager.startSession({
        execution_id: randomUUID(),
        workflow_id: 'safe-area-navigation',
        viewport: { width: 390, height: 844 },
        reuse_mode: 'fresh',
        required_capabilities: {},
      });
      const session = manager.getSession(sessionId);
      await session.pipelineReadyPromise;
      const { page } = session;
      await page.route('http://safe-area.test/**', (route) =>
        route.fulfill({
          contentType: 'text/html',
          body: '<meta name="viewport" content="width=device-width,viewport-fit=cover"><div id="probe" style="padding-bottom:env(safe-area-inset-bottom)"></div>',
        })
      );
      await page.goto('http://safe-area.test/');
      expect(actualViewport).toMatchObject({ width: 390, height: 844 });
      expect(
        await page.locator('#probe').evaluate((el) => getComputedStyle(el).paddingBottom)
      ).toBe('34px');
    } finally {
      await manager.shutdown();
    }
  }, 30000);
});
