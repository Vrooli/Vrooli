import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import { sendJson, sendError } from '../middleware';
import { logger } from '../utils';

/**
 * Close session endpoint
 *
 * POST /session/:id/close
 */
export async function handleSessionClose(
  _req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager
): Promise<void> {
  try {
    logger.info('Closing session', { sessionId });

    await sessionManager.closeSession(sessionId);

    sendJson(res, 200, { success: true });
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/close`);
  }
}
