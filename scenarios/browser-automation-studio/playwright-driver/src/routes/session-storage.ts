import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import { sendError, sendJson } from '../middleware';
import { requireSessionLease } from './session-lease';

/**
 * GET /session/:id/storage-state
 * Returns the current Playwright storage state for the session.
 */
export async function handleSessionStorageState(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager
): Promise<void> {
  const owner = requireSessionLease(req, res, sessionId, sessionManager);
  if (!owner) return;
  try {
    const storageState = await sessionManager.getStorageStateForLease(
      sessionId,
      owner.executionId,
      owner.leaseId
    );
    sendJson(res, 200, {
      session_id: sessionId,
      storage_state: storageState,
    });
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/storage-state`);
  }
}
