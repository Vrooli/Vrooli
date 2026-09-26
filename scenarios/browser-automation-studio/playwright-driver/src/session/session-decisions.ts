/**
 * Session Decisions
 *
 * Named decision functions for session management.
 * Makes the "why" behind session lifecycle decisions explicit and testable.
 *
 * DECISION CATEGORIES:
 * 1. Session Lookup - Finding existing sessions for reuse
 * 2. Session Reuse - Deciding whether/how to reuse a session
 *
 * CHANGE AXIS: Session Lifecycle
 * When modifying session lifecycle behavior:
 * 1. Add or modify decision functions here
 * 2. Keep state-machine.ts for phase transitions
 * 3. Keep manager.ts as the orchestrator
 */

import { isDeepStrictEqual } from 'node:util';
import type { SessionSpec, SessionState } from '../types';

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
export function matchesByExecutionId(session: SessionState, executionId: string): boolean {
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
export function matchesByLabels(session: SessionState, labels?: Record<string, string>): boolean {
  if (!labels || !session.spec.labels) {
    return false;
  }

  return Object.entries(labels).every(([key, value]) => session.spec.labels?.[key] === value);
}

/**
 * A released browser context can cross execution owners only when its
 * identity-bearing profile revision and context inputs still match. Labels
 * choose a pool; they do not prove that browser state belongs to the caller.
 */
export function matchesReusableContext(session: SessionState, requested: SessionSpec): boolean {
  const retained = session.spec;
  return (
    retained.session_profile_version === requested.session_profile_version &&
    isDeepStrictEqual(retained.viewport, requested.viewport) &&
    isDeepStrictEqual(retained.storage_state, requested.storage_state) &&
    isDeepStrictEqual(retained.browser_profile, requested.browser_profile) &&
    isDeepStrictEqual(retained.user_agent, requested.user_agent) &&
    isDeepStrictEqual(retained.locale, requested.locale) &&
    isDeepStrictEqual(retained.timezone, requested.timezone) &&
    isDeepStrictEqual(retained.geolocation, requested.geolocation) &&
    isDeepStrictEqual(retained.permissions, requested.permissions) &&
    isDeepStrictEqual(retained.service_worker_control, requested.service_worker_control) &&
    isDeepStrictEqual(retained.fake_media, requested.fake_media) &&
    isDeepStrictEqual(retained.app_target, requested.app_target) &&
    isDeepStrictEqual(retained.validation_context, requested.validation_context)
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
 * Check if a session is safe to hand to a DIFFERENT execution via label pooling.
 *
 * DECISION: Only idle sessions are poolable
 * Label-based reuse rebinds the session to a new execution (spec overwrite,
 * phase forced to 'ready'). Doing that to a session that is initializing,
 * executing, recording, resetting, or closing hijacks it out from under its
 * current owner: the owner's in-flight instruction gets its navigation
 * aborted (net::ERR_ABORTED) and subsequent instructions race into
 * SESSION_BUSY. Idempotent retries of the SAME execution are handled by the
 * execution_id match, which observes the current lease without phase recovery.
 *
 * @param session - Session to check
 * @returns true if the session may be pooled across executions
 */
export function isSafeForLabelReuse(session: SessionState): boolean {
  return (
    session.phase === 'ready' &&
    !session.instructionInFlight &&
    session.leaseReleasedAt !== undefined
  );
}

/**
 * Find a reusable session by labels and context identity.
 * Used when reuse_mode is 'reuse' or 'clean'.
 * Sessions that are busy with another execution are skipped (see
 * isSafeForLabelReuse); if every matching session is busy, the caller
 * creates a fresh session instead.
 *
 * @param sessions - All active sessions
 * @param requested - The requested labels and context-defining session options
 * @returns The first idle matching session or null
 */
export function findByLabels(
  sessions: Iterable<SessionState>,
  requested: SessionSpec
): SessionState | null {
  if (!requested.labels) {
    return null;
  }

  for (const session of sessions) {
    if (
      matchesByLabels(session, requested.labels) &&
      matchesReusableContext(session, requested) &&
      isSafeForLabelReuse(session)
    ) {
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
export function shouldAttemptReuse(reuseMode: SessionSpec['reuse_mode']): boolean {
  return reuseMode !== 'fresh';
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
  // An instruction may legitimately occupy the session longer than the idle
  // threshold (for example, a long audio-feed wait).  Cleanup must never reap
  // a session while its owner is executing; the in-flight Playwright action is
  // the activity signal even when no HTTP request reaches the driver during
  // that wait.
  if (session.instructionInFlight || session.phase !== 'ready') {
    return true;
  }
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
