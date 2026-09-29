import type { Route } from 'rebrowser-playwright';
import type { SessionState } from '../types';
import { cleanupSession } from '../infra';
import { assertRecordingAcknowledged } from '../recording';
import { stopFrameStreaming } from '../frame-streaming';
import { resetPageInputState, settlePageInput } from './live-input';
import { clearFrameCache } from './frame-cache';
import { resetKeyboardState } from '../handlers/keyboard';

/** Clear managed context state; SessionManager owns admission and phase changes. */
export async function resetSessionState(session: SessionState): Promise<void> {
  if (session.externalTarget) {
    throw new Error('Cannot reset an externally controlled target');
  }
  // Background AI navigation owns the same page as reset. Stop and join that
  // owner before any reset-side listener, input, or navigation effect starts.
  const aiNavigationCleanup = session.aiNavigationCleanup;
  if (aiNavigationCleanup) await aiNavigationCleanup();
  if (session.pipelineManager?.isRecording()) await session.pipelineManager.stopRecording();
  // Session admission starts pipeline verification in the background. Reset
  // must join that owner after any admitted recording flush and before
  // navigating or clearing browser state.
  await session.pipelineReadyPromise?.catch(() => undefined);
  assertRecordingAcknowledged(session.id);
  session.pageLifecycleCleanup?.();
  session.pageLifecycleCleanup = undefined;
  await stopFrameStreaming(session.id);
  clearFrameCache(session.id);
  await Promise.all([...new Set([...session.pages, session.page])].map(async (page) => {
    await settlePageInput(page);
    await resetPageInputState(page);
    await resetKeyboardState(page);
  }));
  // Preserve the session's active page when resetting. `pages[0]` is the
  // original tab, not necessarily the tab owning the current workflow state;
  // resetting that choice silently discards the user's active page and keeps
  // a stale tab instead.
  const page = session.page ?? session.pages[0];
  for (const other of session.context.pages()) {
    if (other !== page) await other.close();
  }
  await page.goto('about:blank');
  if (session.storageOrigins.size) {
    const cdp = await session.context.newCDPSession(page);
    try {
      for (const origin of session.storageOrigins) {
        await cdp.send('Storage.clearDataForOrigin', { origin, storageTypes: 'all' });
      }
    } finally {
      await cdp.detach();
    }
    // Session storage belongs to the retained tab. Empty intercepted documents
    // expose each origin's namespace without making application requests.
    const emptyDocument = (route: Route): Promise<void> => route.fulfill({
      status: 200, contentType: 'text/html', body: '<!doctype html><title>Reset</title>',
    });
    await page.route('**/*', emptyDocument);
    try {
      for (const origin of session.storageOrigins) {
        await page.goto(origin);
        await page.evaluate(() => { localStorage.clear(); sessionStorage.clear(); });
      }
      await page.goto('about:blank');
    } finally {
      await page.unroute('**/*', emptyDocument);
    }
  }
  await session.context.clearCookies();
  await session.context.clearPermissions();
  await page.unroute('**/*');
  await cleanupSession(session.id);
  session.storageOrigins.clear();
  session.activeMocks.clear();
  session.frameStack = [];
  session.page = page;
  session.pages = [page];
  session.currentPageIndex = 0;
  for (const [id, tracked] of session.pageIdMap) {
    if (tracked !== page) {
      session.pageIdMap.delete(id);
      session.pageToIdMap.delete(tracked);
    }
  }
  // A reset must not authorize repeating an old instruction.
  session.instructionReceipts?.clear();
  session.lastUsedAt = new Date();
}
