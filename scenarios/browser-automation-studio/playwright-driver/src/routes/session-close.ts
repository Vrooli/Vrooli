import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import { sendJson, sendError } from '../middleware';
import { logger, scopedLog, LogContext } from '../utils';
import { clearSessionIdempotencyCache } from './session-run';
import { clearSessionDownloadCache } from '../handlers/download';

/**
 * Close session endpoint
 *
 * POST /session/:id/close
 *
 * Cleanup behavior:
 * - Closes browser context and page
 * - Clears idempotency cache entries for this session
 * - Clears download cache entries for this session
 */
export async function handleSessionClose(
  _req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager
): Promise<void> {
  try {
    logger.info(scopedLog(LogContext.SESSION, 'closing session'), { sessionId });

    const result = await sessionManager.closeSession(sessionId);

    // Clean up caches associated with this session
    clearSessionIdempotencyCache(sessionId);
    clearSessionDownloadCache(sessionId);

    logger.info(scopedLog(LogContext.SESSION, 'session closed'), { sessionId });

    const response: { success: boolean; video_paths?: string[] } = { success: true };
    if (result.videoPaths.length > 0) {
      response.video_paths = result.videoPaths;
    }
    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/close`);
  }
}
