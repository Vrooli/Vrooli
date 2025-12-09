import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import { sendError, sendJson } from '../middleware';

/**
 * GET /session/:id/storage-state
 * Returns the current Playwright storage state for the session.
 */
export async function handleSessionStorageState(
  _req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager
): Promise<void> {
  try {
    const storageState = await sessionManager.getStorageState(sessionId);
    sendJson(res, 200, {
      session_id: sessionId,
      storage_state: storageState,
    });
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/storage-state`);
  }
}
