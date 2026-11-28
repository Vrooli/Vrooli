import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import type { SessionState } from '../types/session';
import { sendJson } from '../middleware';

export const SESSION_EXECUTION_HEADER = 'x-bas-execution-id';
export const SESSION_LEASE_HEADER = 'x-bas-session-lease-id';

/** Resolve a session only when the request carries its current broker lease. */
export interface SessionLease {
  session: SessionState;
  executionId: string;
  leaseId: string;
}

export function requireSessionLease(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager
): SessionLease | null {
  const executionId = req.headers[SESSION_EXECUTION_HEADER];
  const leaseId = req.headers[SESSION_LEASE_HEADER];
  if (typeof executionId !== 'string' || typeof leaseId !== 'string' || !executionId || !leaseId) {
    sendJson(res, 404, { error: 'session not found' });
    return null;
  }
  try {
    return { session: sessionManager.getSessionForLease(sessionId, executionId, leaseId), executionId, leaseId };
  } catch {
    // Wrong-owner and stale-lease requests are indistinguishable from an
    // absent session; do not reveal whether another execution owns the ID.
    sendJson(res, 404, { error: 'session not found' });
    return null;
  }
}
