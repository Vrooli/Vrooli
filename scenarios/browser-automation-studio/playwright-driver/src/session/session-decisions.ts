/**
 * Session Decisions
 *
 * Named decision functions for session management.
 * Makes the "why" behind session lifecycle decisions explicit and testable.
 *
 * DECISION CATEGORIES:
 * 1. Session Lookup - Finding existing sessions for reuse
 * 2. Session Reuse - Deciding whether/how to reuse a session
 * 3. Phase Recovery - Handling stuck session phases
 *
 * CHANGE AXIS: Session Lifecycle
 * When modifying session lifecycle behavior:
 * 1. Add or modify decision functions here
 * 2. Keep state-machine.ts for phase transitions
 * 3. Keep manager.ts as the orchestrator
 */

import type { SessionSpec, SessionState, SessionPhase } from '../types';

// =============================================================================
// Types
// =============================================================================

/**
 * Result of looking up a session for potential reuse.
 */
export interface SessionLookupResult {
  /** The session that was found, if any */
  session: SessionState | null;
  /** Reason the session was selected */
  reason: 'execution_id_match' | 'label_match' | 'none';
}

/**
 * Decision about how to handle an existing session.
 */
export interface ReuseDecision {
  /** Whether to reuse the existing session */
  shouldReuse: boolean;
  /** Whether to reset the session before reuse */
  shouldReset: boolean;
  /** Whether to recover the session phase */
  shouldRecoverPhase: boolean;
  /** Reason for the decision */
  reason: string;
}

// =============================================================================
// Session Lookup Decisions
// =============================================================================

/**
 * Check if a session matches by execution_id.
 *
 * DECISION: Execution ID match for idempotency
 * When the same execution_id is provided, we return the existing session.
 * This ensures that retried requests don't create duplicate sessions.
 *
 * @param session - Session to check
 * @param executionId - Execution ID to match
 * @returns true if the session matches the execution ID
 */
export function matchesByExecutionId(
  session: SessionState,
  executionId: string
): boolean {
  return session.spec.execution_id === executionId;
}

/**
 * Check if a session matches by labels.
 *
 * DECISION: Label matching for session pooling
 * Sessions can be reused across different executions if they have matching labels.
 * All specified labels must match for the session to be considered.
 *
 * @param session - Session to check
 * @param labels - Labels to match (all must match)
 * @returns true if all specified labels match
 */
export function matchesByLabels(
  session: SessionState,
  labels?: Record<string, string>
): boolean {
  if (!labels || !session.spec.labels) {
    return false;
  }

  return Object.entries(labels).every(
    ([key, value]) => session.spec.labels?.[key] === value
  );
}

/**
 * Find a session by execution ID.
 * Used for idempotent session creation.
 *
 * @param sessions - All active sessions
 * @param executionId - Execution ID to find
 * @returns The matching session or null
 */
export function findByExecutionId(
  sessions: Iterable<SessionState>,
  executionId: string
): SessionState | null {
  for (const session of sessions) {
    if (matchesByExecutionId(session, executionId)) {
      return session;
    }
  }
  return null;
}

/**
 * Find a reusable session by labels.
 * Used when reuse_mode is 'reuse' or 'clean'.
 *
 * @param sessions - All active sessions
 * @param labels - Labels to match
 * @returns The first matching session or null
 */
export function findByLabels(
  sessions: Iterable<SessionState>,
  labels?: Record<string, string>
): SessionState | null {
  if (!labels) {
    return null;
  }

  for (const session of sessions) {
    if (matchesByLabels(session, labels)) {
      return session;
    }
  }
  return null;
}

// =============================================================================
// Session Reuse Decisions
// =============================================================================

/**
 * Determine if we should attempt to reuse an existing session.
 *
 * DECISION: Reuse mode interpretation
 * - 'fresh': Always create a new session
 * - 'reuse': Reuse existing session if labels match
 * - 'clean': Reuse existing session but reset its state
 *
 * @param reuseMode - The requested reuse mode
 * @returns true if we should look for reusable sessions
 */
export function shouldAttemptReuse(
  reuseMode: SessionSpec['reuse_mode']
): boolean {
  return reuseMode !== 'fresh';
}

/**
 * Determine if a session should be reset on reuse.
 *
 * DECISION: Clean mode behavior
 * When reuse_mode is 'clean', we reuse the browser context but
 * clear cookies, storage, and navigation state.
 *
 * @param reuseMode - The requested reuse mode
 * @returns true if the session should be reset before reuse
 */
export function shouldResetOnReuse(
  reuseMode: SessionSpec['reuse_mode']
): boolean {
  return reuseMode === 'clean';
}

/**
 * Determine if a session phase should be recovered.
 *
 * DECISION: Stuck phase recovery
 * If a session is found in 'executing' phase during a retry (same execution_id),
 * the previous execution likely crashed or timed out.
 * We recover by resetting the phase to 'ready'.
 *
 * Recovery is safe because:
 * 1. Same execution_id indicates a retry of the same operation
 * 2. The previous execution won't complete (crashed/timed out)
 * 3. Leaving in 'executing' would permanently block the session
 *
 * @param currentPhase - Current session phase
 * @param isRetry - Whether this is a retry (same execution_id)
 * @returns true if phase should be recovered to 'ready'
 */
export function shouldRecoverPhase(
  currentPhase: SessionPhase,
  isRetry: boolean
): boolean {
  return currentPhase === 'executing' && isRetry;
}

/**
 * Make a complete reuse decision for an existing session.
 *
 * @param session - The existing session
 * @param spec - The new session spec
 * @param matchReason - How the session was matched
 * @returns Decision about how to handle the session
 */
export function makeReuseDecision(
  session: SessionState,
  spec: SessionSpec,
  matchReason: 'execution_id_match' | 'label_match'
): ReuseDecision {
  const isRetry = matchReason === 'execution_id_match';
  const shouldReset = shouldResetOnReuse(spec.reuse_mode);
  const needsPhaseRecovery = shouldRecoverPhase(session.phase, isRetry);

  let reason: string;
  if (isRetry) {
    reason = needsPhaseRecovery
      ? 'Retry of previous execution - recovering from stuck executing phase'
      : 'Retry of previous execution - returning existing session';
  } else {
    reason = shouldReset
      ? 'Label match with clean mode - resetting session state'
      : 'Label match - reusing session with current state';
  }

  return {
    shouldReuse: true,
    shouldReset,
    shouldRecoverPhase: needsPhaseRecovery,
    reason,
  };
}

// =============================================================================
// Session State Decisions
// =============================================================================

/**
 * Determine if a session is actively being used.
 *
 * DECISION: Active session criteria
 * A session is active if it was used within the idle timeout period.
 * Used for metrics and cleanup decisions.
 *
 * @param session - Session to check
 * @param idleTimeoutMs - Idle timeout in milliseconds
 * @param now - Current timestamp (for testing)
 * @returns true if the session is active
 */
export function isSessionActive(
  session: SessionState,
  idleTimeoutMs: number,
  now: number = Date.now()
): boolean {
  const idleTimeMs = now - session.lastUsedAt.getTime();
  return idleTimeMs < idleTimeoutMs;
}

/**
 * Determine if a session should be cleaned up due to idleness.
 *
 * DECISION: Idle cleanup criteria
 * Sessions that haven't been used within the idle timeout are candidates for cleanup.
 * This prevents resource leaks from abandoned sessions.
 *
 * @param session - Session to check
 * @param idleTimeoutMs - Idle timeout in milliseconds
 * @param now - Current timestamp (for testing)
 * @returns true if the session should be cleaned up
 */
export function shouldCleanupSession(
  session: SessionState,
  idleTimeoutMs: number,
  now: number = Date.now()
): boolean {
  return !isSessionActive(session, idleTimeoutMs, now);
}

/**
 * Find all sessions that should be cleaned up.
 *
 * @param sessions - All active sessions
 * @param idleTimeoutMs - Idle timeout in milliseconds
 * @param now - Current timestamp (for testing)
 * @returns Array of session IDs to clean up
 */
export function findIdleSessions(
  sessions: Map<string, SessionState>,
  idleTimeoutMs: number,
  now: number = Date.now()
): string[] {
  const idleSessions: string[] = [];

  for (const [sessionId, session] of sessions.entries()) {
    if (shouldCleanupSession(session, idleTimeoutMs, now)) {
      idleSessions.push(sessionId);
    }
  }

  return idleSessions;
}
