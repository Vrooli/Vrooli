import type { Route } from 'rebrowser-playwright';
import type { SessionState } from '../types';
import { cleanupSession } from '../infra';
import { assertRecordingAcknowledged } from '../recording';

/** Clear managed context state; SessionManager owns admission and phase changes. */
export async function resetSessionState(session: SessionState): Promise<void> {
  if (session.pipelineManager?.isRecording()) await session.pipelineManager.stopRecording();
  assertRecordingAcknowledged(session.id);
  const page = session.pages[0] ?? session.page;
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
