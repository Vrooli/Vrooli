import { randomUUID } from 'node:crypto';
import { SessionManager } from '../../src/session/manager';
import { TabHandler } from '../../src/handlers/tab';
import type { HandlerContext } from '../../src/handlers/base';
import type { SessionSpec } from '../../src/types';
import { ResourceLimitError } from '../../src/utils/errors';
import { logger, metrics } from '../../src/utils';
import { createTestConfig, createTypedInstruction } from '../helpers';

describe('session capacity admission with Chromium', () => {
  it('replays equal selectors on the opener and popup pages through typed tab switches', async () => {
    const manager = new SessionManager(createTestConfig());
    const spec: SessionSpec = {
      execution_id: randomUUID(),
      workflow_id: 'workflow-popup-tab-replay',
      viewport: { width: 640, height: 480 },
      reuse_mode: 'fresh',
      required_capabilities: {},
    };

    try {
      const { sessionId } = await manager.startSession(spec);
      const session = manager.getSession(sessionId);
      session.page.setDefaultTimeout(3000);
      session.context.setDefaultTimeout(3000);
      const popupHTML =
        '<button id="same" onclick="document.body.dataset.clicked=\'popup\'">same</button>';
      await session.page.evaluate(() => {
        document.body.innerHTML =
          '<button id="same" onclick="document.body.dataset.clicked=\'main\'">same</button>';
      });
      const popupPromise = session.context.waitForEvent('page', { timeout: 3000 });
      await session.page.evaluate(() => window.open('about:blank'));
      const popup = await popupPromise;
      await popup.evaluate((html) => {
        document.body.innerHTML = html;
      }, popupHTML);
      expect(session.pages).toEqual([session.page, popup]);

      const context: HandlerContext = {
        page: session.page,
        browserContext: session.context,
        tabStack: session.pages,
        config: createTestConfig(),
        logger,
        metrics,
        sessionId,
        frameStack: session.frameStack,
      };
      const handler = new TabHandler();
      const switchToPopup = createTypedInstruction('tab-switch', { action: 'switch', index: 1 });
      expect((await handler.execute(switchToPopup, context)).success).toBe(true);
      await context.page.locator('#same').click();
      expect(await context.page.locator('body').getAttribute('data-clicked')).toBe('popup');

      const switchToOpener = createTypedInstruction('tab-switch', { action: 'switch', index: 0 });
      expect((await handler.execute(switchToOpener, context)).success).toBe(true);
      await context.page.locator('#same').click();
      expect(await context.page.locator('body').getAttribute('data-clicked')).toBe('main');
      await popup.close();
    } finally {
      await manager.shutdown();
    }
  }, 15000);

  it('tracks popup pages in the workflow tab stack when recording is off', async () => {
    const manager = new SessionManager(createTestConfig());
    const spec: SessionSpec = {
      execution_id: randomUUID(),
      workflow_id: 'workflow-popup-tab-stack',
      viewport: { width: 640, height: 480 },
      reuse_mode: 'fresh',
      required_capabilities: {},
    };

    try {
      const { sessionId } = await manager.startSession(spec);
      const session = manager.getSession(sessionId);
      const initialPage = session.page;
      const popupPromise = session.context.waitForEvent('page');
      await session.page.evaluate(() => window.open('about:blank'));
      const popup = await popupPromise;

      expect(session.pipelineManager.isRecording()).toBe(false);
      expect(session.pages).toContain(popup);
      expect(session.pages.filter((page) => page === popup)).toHaveLength(1);

      session.page = popup;
      session.currentPageIndex = session.pages.indexOf(popup);
      await popup.close();
      expect(session.pages).not.toContain(popup);
      expect(session.page).toBe(initialPage);
      expect(session.currentPageIndex).toBe(0);
    } finally {
      await manager.shutdown();
    }
  }, 30000);

  it('coalesces a same-owner retry and rejects a distinct start at the configured limit', async () => {
    const manager = new SessionManager(
      createTestConfig({
        session: { maxConcurrent: 1, idleTimeoutMs: 300000, poolSize: 5, cleanupIntervalMs: 60000 },
      })
    );
    const sameOwnerSpec: SessionSpec = {
      execution_id: randomUUID(),
      workflow_id: 'session-capacity-same-owner',
      viewport: { width: 640, height: 480 },
      reuse_mode: 'fresh',
      required_capabilities: {},
    };

    try {
      const [firstRetry, sameOwnerRetry] = await Promise.all([
        manager.startSession(sameOwnerSpec),
        manager.startSession(sameOwnerSpec),
      ]);
      expect(sameOwnerRetry.sessionId).toBe(firstRetry.sessionId);
      expect(sameOwnerRetry.leaseId).toBe(firstRetry.leaseId);
      expect(manager.getSessionCount()).toBe(1);
      await manager.closeSession(firstRetry.sessionId);

      const [first, second] = await Promise.allSettled([
        manager.startSession({ ...sameOwnerSpec, execution_id: randomUUID() }),
        manager.startSession({ ...sameOwnerSpec, execution_id: randomUUID() }),
      ]);
      const outcomes = [first, second];
      expect(outcomes.filter((outcome) => outcome.status === 'fulfilled')).toHaveLength(1);
      const rejected = outcomes.filter((outcome) => outcome.status === 'rejected');
      expect(rejected).toHaveLength(1);
      if (rejected[0]?.status === 'rejected') {
        expect(rejected[0].reason).toBeInstanceOf(ResourceLimitError);
      }
      expect(manager.getSessionCount()).toBe(1);
    } finally {
      await manager.shutdown();
    }
  }, 30000);
});
