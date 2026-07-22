import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import { parseJsonBody, sendJson, sendError } from '../middleware';
import { logger, scopedLog, LogContext } from '../utils';
import { InvalidInstructionError } from '../utils';
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
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager
): Promise<void> {
  try {
    const body = await parseJsonBody(req, {});
    const executionID = typeof body.execution_id === 'string' ? body.execution_id : '';
    const leaseID = typeof body.lease_id === 'string' ? body.lease_id : '';
    if (!executionID || !leaseID) {
      throw new InvalidInstructionError('execution_id and lease_id are required to close a session');
    }
    logger.info(scopedLog(LogContext.SESSION, 'closing session'), { sessionId });

    const result = await sessionManager.closeSessionForLease(sessionId, executionID, leaseID);

    // Clean up caches associated with this session
    clearSessionIdempotencyCache(sessionId);
    clearSessionDownloadCache(sessionId);

    logger.info(scopedLog(LogContext.SESSION, 'session closed'), { sessionId });

    const response: { success: boolean; video_paths?: string[]; trace_path?: string; har_path?: string } = {
      success: true,
    };
    if (result.videoPaths.length > 0) {
      response.video_paths = result.videoPaths;
    }
    if (result.tracePath) {
      response.trace_path = result.tracePath;
    }
    if (result.harPath) {
      response.har_path = result.harPath;
    }
    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/close`);
  }
}
