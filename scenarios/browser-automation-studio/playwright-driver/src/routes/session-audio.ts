import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import { parseJsonBody, sendJson, sendError } from '../middleware';
import { InvalidInstructionError, SessionNotFoundError } from '../utils';

/** Restart a session-owned deterministic host-audio source at a turn boundary. */
export async function handleSessionAudioRestart(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
): Promise<void> {
  try {
    const body = await parseJsonBody(req, {});
    const executionID = typeof body.execution_id === 'string' ? body.execution_id : '';
    const leaseID = typeof body.lease_id === 'string' ? body.lease_id : '';
    if (!executionID || !leaseID) {
      throw new InvalidInstructionError('execution_id and lease_id are required to restart session audio');
    }
    if (!(await sessionManager.restartAudioPlayback(sessionId, executionID, leaseID))) {
      throw new SessionNotFoundError(sessionId);
    }
    sendJson(res, 200, { success: true });
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/audio/restart`);
  }
}

/** Stop a session-owned deterministic host-audio source at a turn boundary. */
export async function handleSessionAudioStop(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
): Promise<void> {
  try {
    const body = await parseJsonBody(req, {});
    const executionID = typeof body.execution_id === 'string' ? body.execution_id : '';
    const leaseID = typeof body.lease_id === 'string' ? body.lease_id : '';
    if (!executionID || !leaseID) {
      throw new InvalidInstructionError('execution_id and lease_id are required to stop session audio');
    }
    if (!(await sessionManager.stopAudioPlayback(sessionId, executionID, leaseID))) {
      throw new SessionNotFoundError(sessionId);
    }
    sendJson(res, 200, { success: true });
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/audio/stop`);
  }
}
