import type { SessionManager } from '../../session/manager';
import { isOperational } from '../../session/state-machine';
import { InvalidInstructionError, SessionNotFoundError } from '../../utils/errors';

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

