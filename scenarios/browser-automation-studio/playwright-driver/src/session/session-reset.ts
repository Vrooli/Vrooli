import type { SessionState } from '../types';
import { cleanupSession } from '../infra';
import { logger, metrics, scopedLog, LogContext } from '../utils';

const message = (error: unknown): string => error instanceof Error ? error.message : String(error);

/** Reset mutable browser state while retaining the session and its lease. */
export async function resetSessionState(session: SessionState): Promise<void> {
  const { id: sessionId } = session;
  const previousPhase = session.phase;
  session.phase = 'resetting';
  logger.info(scopedLog(LogContext.SESSION, 'resetting'), { sessionId, previousPhase, instructionCount: session.instructionCount });
  await session.page.goto('about:blank');
  await session.context.clearCookies();
  await session.context.clearPermissions();
  await session.page.evaluate(() => { window.localStorage.clear(); window.sessionStorage.clear(); });
  session.frameStack = [];
  for (const page of session.pages.slice(1)) {
    await page.close().catch((error: unknown) => {
      logger.warn(scopedLog(LogContext.CLEANUP, 'page close failed'), { sessionId, error: message(error) });
      metrics.cleanupFailures.inc({ operation: 'page_close' });
    });
  }
  session.pages = [session.page];
  session.currentPageIndex = 0;
  session.activeMocks.clear();
  await session.page.unroute('**/*').catch((error: unknown) => logger.warn(scopedLog(LogContext.CLEANUP, 'unroute failed'), { sessionId, error: message(error) }));
  await cleanupSession(sessionId);
  session.executedInstructions?.clear();
  session.lastUsedAt = new Date();
  session.phase = 'ready';
  logger.info(scopedLog(LogContext.SESSION, 'reset complete'), { sessionId, phase: 'ready' });
}
