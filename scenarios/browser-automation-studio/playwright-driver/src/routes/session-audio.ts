import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import { parseJsonBody, sendJson, sendError } from '../middleware';
import { InvalidInstructionError, SessionNotFoundError } from '../utils';
import type { SessionState } from '../types/session';
import { authorizeUnreceiptedOperation } from '../session/permission-policy';

function audioPermissionDenied(session: SessionState, kind: 'audio_playback_restart' | 'audio_playback_stop' | 'audio_observe'): string | undefined {
  const profile = session.spec?.permission_profile;
  if (!profile) return undefined;
  if (!profile.audio_mode) return 'browser audio operation requires an Agent Manager audio capability';
  if (kind.startsWith('audio_playback_') && profile.audio_mode !== 'pipewire_qualification') {
    return 'host audio playback controls require the PipeWire qualification capability';
  }
  let currentURL: string;
  try { currentURL = session.page.url(); } catch {
    return 'current page URL unavailable';
  }
  const decision = authorizeUnreceiptedOperation(
    profile,
    kind === 'audio_observe' ? 'observe' : kind,
    currentURL,
    undefined,
    Date.now(),
    session.createdAt.getTime(),
    session.instructionCount,
  );
  return decision.allowed ? undefined : decision.reason;
}

function sendAudioPermissionDenied(res: ServerResponse, message: string): void {
  sendJson(res, 403, { error: { code: 'BROWSER_PERMISSION_DENIED', message, kind: 'auth', retryable: false } });
}

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
    const session = sessionManager.getSessionForLease(sessionId, executionID, leaseID);
    const denied = audioPermissionDenied(session, 'audio_playback_restart');
    if (denied) {
      sendAudioPermissionDenied(res, denied);
      return;
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
    const session = sessionManager.getSessionForLease(sessionId, executionID, leaseID);
    const denied = audioPermissionDenied(session, 'audio_playback_stop');
    if (denied) {
      sendAudioPermissionDenied(res, denied);
      return;
    }
    if (!(await sessionManager.stopAudioPlayback(sessionId, executionID, leaseID))) {
      throw new SessionNotFoundError(sessionId);
    }
    sendJson(res, 200, { success: true });
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/audio/stop`);
  }
}

/** Read one of the fixed, metadata-focused Audio Tools page diagnostics. */
export async function handleSessionAudioDiagnostics(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
): Promise<void> {
  try {
    const body = await parseJsonBody(req, {});
    const executionID = typeof body.execution_id === 'string' ? body.execution_id : '';
    const leaseID = typeof body.lease_id === 'string' ? body.lease_id : '';
    const surface = typeof body.surface === 'string' ? body.surface : '';
    if (!executionID || !leaseID || (surface !== 'audio-tools' && surface !== 'swarm-manager')) {
      throw new InvalidInstructionError('owner lease and supported audio surface are required');
    }
    const session = sessionManager.getSessionForLease(sessionId, executionID, leaseID);
    const denied = audioPermissionDenied(session, 'audio_observe');
    if (denied) {
      sendAudioPermissionDenied(res, denied);
      return;
    }
    const result = await sessionManager.readAudioDiagnostics(
      sessionId,
      executionID,
      leaseID,
      surface,
    );
    if (result === null) throw new SessionNotFoundError(sessionId);
    sendJson(res, 200, { success: true, diagnostics: result });
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/audio/diagnostics`);
  }
}
