import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import { parseJsonBody, sendJson, sendError } from '../middleware';
import { InvalidInstructionError, SessionNotFoundError } from '../utils';

/**
 * Release execution lease endpoint.
 *
 * POST /session/:id/release
 *
 * Releasing keeps the browser resource alive but makes its label eligible for
 * a later execution. The exact owner and lease capability are required so a
 * stale executor cannot release a newer owner's session.
 */
export async function handleSessionRelease(
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
      throw new InvalidInstructionError('execution_id and lease_id are required to release a session');
    }
    if (!sessionManager.releaseExecutionLease(sessionId, executionID, leaseID)) {
      throw new SessionNotFoundError(sessionId);
    }
    sendJson(res, 200, { success: true });
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/release`);
  }
}
