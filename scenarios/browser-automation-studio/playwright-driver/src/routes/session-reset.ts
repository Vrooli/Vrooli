import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import { parseJsonBody, sendJson, sendError } from '../middleware';
import { InvalidInstructionError } from '../utils';

/** POST /session/:id/reset requires the current, unreleased execution lease. */
export async function handleSessionReset(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager
): Promise<void> {
  try {
    const body = await parseJsonBody(req, {});
    const executionId = typeof body.execution_id === 'string' ? body.execution_id : '';
    const leaseId = typeof body.lease_id === 'string' ? body.lease_id : '';
    if (!executionId.trim() || !leaseId.trim()) {
      throw new InvalidInstructionError('execution_id and lease_id are required to reset a session');
    }
    // Parsing yields: validate ownership before joining work or changing activity.
    sessionManager.getSessionForLease(sessionId, executionId, leaseId);
    await sessionManager.resetSession(sessionId);
    sendJson(res, 200, { success: true, phase: sessionManager.peekSession(sessionId).phase });
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/reset`);
  }
}
