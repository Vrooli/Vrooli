import type { SessionManager } from '../../session/manager';
import { isOperational } from '../../session/state-machine';
import { InvalidInstructionError, SessionNotFoundError } from '../../utils/errors';
import { authorizeUnreceiptedOperation } from '../../session/permission-policy';
import { sendError, sendJson } from '../../middleware';
import type { ServerResponse } from 'http';

// Recording operations retain one execution lease across admission and awaited work.
export function recordingOwner(body: Record<string, unknown>, sessionId: string, manager: SessionManager) {
  const { execution_id: executionId, lease_id: leaseId } = body;
  if (typeof executionId !== 'string' || !executionId.trim() || typeof leaseId !== 'string' || !leaseId.trim()) {
    throw new InvalidInstructionError('execution_id and lease_id are required for recording operations');
  }
  const session = manager.getSessionForLease(sessionId, executionId, leaseId);
  return () => {
    const current = manager.getSessionForLease(sessionId, executionId, leaseId);
    if (current !== session || !isOperational(current.phase)) throw new SessionNotFoundError(sessionId);
    return current;
  };
}

/**
 * Independent driver-side admission for lease-owned recording operations.
 * Approval tier mapping is shared with the instruction owner; these routes
 * do not carry a receipt verifier and fail closed when the tier requires one.
 */
export function recordingPermissionDenied(
  session: ReturnType<SessionManager['getSession']>,
  kind: string,
  targetURL?: string,
): string | undefined {
  const profile = session.spec?.permission_profile;
  if (!profile) return undefined;
  let currentURL: string;
  try {
    currentURL = session.page.url();
  } catch {
    return 'current page URL unavailable';
  }
  const now = Date.now();
  const current = authorizeUnreceiptedOperation(profile, kind, currentURL, undefined, now, session.createdAt.getTime(), session.instructionCount);
  if (!current.allowed) return current.reason;
  if (targetURL) {
    const target = kind === 'navigate'
      ? authorizeUnreceiptedOperation(profile, 'navigate', currentURL, targetURL, now, session.createdAt.getTime(), session.instructionCount)
      : authorizeUnreceiptedOperation(profile, kind === 'history_read' || kind === 'observe' || kind === 'screenshot' ? kind : 'navigate', targetURL, undefined, now, session.createdAt.getTime(), session.instructionCount);
    if (!target.allowed) return target.reason;
  }
  return undefined;
}

export function sendRecordingError(res: ServerResponse, error: Error, route: string): void {
  const prefix = 'BROWSER_PERMISSION_DENIED: ';
  if (error.message.startsWith(prefix)) {
    sendJson(res, 403, { error: { code: 'BROWSER_PERMISSION_DENIED', message: error.message.slice(prefix.length), kind: 'auth', retryable: false } });
    return;
  }
  sendError(res, error, route);
}

