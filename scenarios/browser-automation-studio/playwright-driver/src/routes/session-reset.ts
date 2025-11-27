import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import { sendJson, sendError } from '../middleware';
import { logger } from '../utils';

/**
 * Reset session endpoint
 *
 * POST /session/:id/reset
 */
export async function handleSessionReset(
  _req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager
): Promise<void> {
  try {
    logger.info('Resetting session', { sessionId });

    await sessionManager.resetSession(sessionId);

    sendJson(res, 200, { success: true });
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/reset`);
  }
}
