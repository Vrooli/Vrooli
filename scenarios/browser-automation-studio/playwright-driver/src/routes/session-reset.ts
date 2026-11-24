import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import { parseJsonBody, sendJson, sendError } from '../middleware';
import { InvalidInstructionError } from '../utils';
import { authorizeUnreceiptedOperation } from '../session/permission-policy';

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
    const session = sessionManager.getSessionForLease(sessionId, executionId, leaseId);
    const profile = session.spec?.permission_profile;
    if (profile) {
      if (profile.profile_mode === 'user_attached') {
        sendJson(res, 403, { error: { code: 'BROWSER_PERMISSION_DENIED', message: 'resetting an attached user browser is denied', kind: 'auth', retryable: false } });
        return;
      }
      let currentURL: string;
      try { currentURL = session.page.url(); } catch {
        sendJson(res, 403, { error: { code: 'BROWSER_PERMISSION_DENIED', message: 'current page URL unavailable', kind: 'auth', retryable: false } });
        return;
      }
      const decision = authorizeUnreceiptedOperation(profile, 'reset', currentURL, undefined, Date.now(), session.createdAt.getTime(), session.instructionCount);
      if (!decision.allowed) {
        sendJson(res, 403, { error: { code: 'BROWSER_PERMISSION_DENIED', message: decision.reason, kind: 'auth', retryable: false } });
        return;
      }
    }
    await sessionManager.resetSession(sessionId);
    sendJson(res, 200, { success: true, phase: sessionManager.peekSession(sessionId).phase });
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/reset`);
  }
}
