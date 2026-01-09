/**
 * Session State Machine
 *
 * Explicit state machine for session lifecycle phase transitions.
 * This provides a single source of truth for valid state transitions,
 * replacing ad-hoc phase changes scattered across the codebase.
 *
 * ## STATE DIAGRAM
 *
 * ```
 *                          ┌─────────────────────────────────────┐
 *                          │                                     │
 *    startSession()        │       NORMAL OPERATION              │
 *         │                │                                     │
 *         ▼                │   ┌────────────────────────────┐    │
 *  ┌──────────────┐        │   │                            │    │
 *  │ initializing │────────┼──▶│           ready            │◀───┼────┐
 *  └──────────────┘        │   │                            │    │    │
 *                          │   └─────────┬──────────────────┘    │    │
 *                          │             │                       │    │
 *                          │   runInstruction()    startRecording()   │
 *                          │             │                       │    │
 *                          │             ▼                       │    │
 *                          │   ┌────────────────┐               │    │
 *                          │   │   executing    │───────────────┼────┘
 *                          │   │                │  instruction  │
 *                          │   │ (one at a time │    done       │
 *                          │   │  per session)  │               │
 *                          │   └────────────────┘               │
 *                          │                                     │
 *                          │   ┌────────────────┐               │
 *              ┌───────────┼──▶│   recording    │───────────────┼────┐
 *              │           │   │                │  stopRecording()   │
 *  startRecording()        │   │ (captures user │               │    │
 *              │           │   │    actions)    │               │    │
 *              │           │   └────────────────┘               │    │
 *              │           │                                     │    │
 *              │           └─────────────────────────────────────┘    │
 *              │                                                      │
 *              └──────────────────────────────────────────────────────┘
 *
 *    resetSession()               closeSession()
 *         │                            │
 *         ▼                            ▼
 *  ┌──────────────┐            ┌──────────────┐
 *  │  resetting   │            │   closing    │
 *  │              │            │              │
 *  │ (clears      │            │ (frees       │
 *  │  state)      │────────▶   │  resources)  │
 *  └──────────────┘  done      └──────────────┘
 *                       │
 *                       ▼
 *                     ready
 * ```
 *
 * ## USAGE
 *
 * ```typescript
 * import { canTransition, transition, SessionPhase } from './state-machine';
 *
 * // Check if transition is valid
 * if (canTransition(currentPhase, 'executing')) {
 *   // Proceed with transition
 * }
 *
 * // Transition with validation (throws on invalid)
 * const newPhase = transition(currentPhase, 'executing', sessionId);
 * ```
 *
 * @module session/state-machine
 */

import type { SessionPhase } from '../types';
import { logger, scopedLog, LogContext } from '../utils';

// =============================================================================
// Transition Rules
// =============================================================================

/**
 * Valid state transitions for session phases.
 *
 * Each key is a source phase, and the value is an array of valid target phases.
 * This is the SINGLE SOURCE OF TRUTH for allowed transitions.
 *
 * DESIGN NOTES:
 * - 'closing' is terminal - no transitions out
 * - 'any' phase can transition to 'resetting' or 'closing'
 * - 'executing' can return to 'recording' if recording was active
 */
const VALID_TRANSITIONS: Record<SessionPhase, readonly SessionPhase[]> = {
  // Initial state - can only become ready or close
  initializing: ['ready', 'closing'],

  // Ready state - can start executing, recording, reset, or close
  ready: ['executing', 'recording', 'resetting', 'closing'],

  // Executing state - returns to ready, or recording if it was active
  executing: ['ready', 'recording', 'resetting', 'closing'],

  // Recording state - can return to ready, start executing, reset, or close
  recording: ['ready', 'executing', 'resetting', 'closing'],

  // Resetting state - returns to ready or can close
  resetting: ['ready', 'closing'],

  // Closing is terminal - no valid transitions out
  closing: [],
};

/**
 * Phases that allow immediate termination (closing).
 * All phases except 'closing' itself can transition to 'closing'.
 */
const CLOSEABLE_PHASES: readonly SessionPhase[] = [
  'initializing',
  'ready',
  'executing',
  'recording',
  'resetting',
];

// =============================================================================
// Transition Functions
// =============================================================================

/**
 * Check if a phase transition is valid.
 *
 * @param from - Current phase
 * @param to - Target phase
 * @returns true if the transition is allowed
 */
export function canTransition(from: SessionPhase, to: SessionPhase): boolean {
  const validTargets = VALID_TRANSITIONS[from];
  return validTargets.includes(to);
}

/**
 * Get all valid target phases from a given phase.
 *
 * @param from - Current phase
 * @returns Array of phases that can be transitioned to
 */
export function getValidTransitions(from: SessionPhase): readonly SessionPhase[] {
  return VALID_TRANSITIONS[from];
}

/**
 * Check if a phase can be closed (terminated).
 *
 * @param phase - Current phase
 * @returns true if the session can be closed from this phase
 */
export function canClose(phase: SessionPhase): boolean {
  return (CLOSEABLE_PHASES as readonly SessionPhase[]).includes(phase);
}

/**
 * Transition to a new phase with validation.
 *
 * This function validates the transition and logs it. It does NOT throw
 * on invalid transitions - instead it logs a warning and returns the
 * original phase. This "fail-safe" behavior prevents crashes from
 * unexpected state scenarios while still capturing the issue in logs.
 *
 * @param from - Current phase
 * @param to - Target phase
 * @param sessionId - Session ID for logging
 * @returns The new phase if valid, or the original phase if invalid
 */
export function transition(
  from: SessionPhase,
  to: SessionPhase,
  sessionId: string
): SessionPhase {
  if (canTransition(from, to)) {
    logger.debug(scopedLog(LogContext.SESSION, 'phase transition'), {
      sessionId,
      from,
      to,
    });
    return to;
  }

  // Invalid transition - log warning but don't crash
  logger.warn(scopedLog(LogContext.SESSION, 'invalid phase transition attempted'), {
    sessionId,
    from,
    to,
    validTargets: VALID_TRANSITIONS[from],
    hint: `Transition from '${from}' to '${to}' is not allowed. Check state machine rules.`,
  });

  return from;
}

/**
 * Transition to a new phase with strict validation.
 *
 * Unlike `transition()`, this function throws an error on invalid transitions.
 * Use this when an invalid transition indicates a programming error that
 * should be caught during development.
 *
 * @param from - Current phase
 * @param to - Target phase
 * @param sessionId - Session ID for logging
 * @returns The new phase
 * @throws Error if the transition is invalid
 */
export function transitionStrict(
  from: SessionPhase,
  to: SessionPhase,
  sessionId: string
): SessionPhase {
  if (!canTransition(from, to)) {
    const error = new Error(
      `Invalid session phase transition: ${from} → ${to} (session: ${sessionId}). ` +
      `Valid transitions from '${from}': [${VALID_TRANSITIONS[from].join(', ')}]`
    );
    logger.error(scopedLog(LogContext.SESSION, 'invalid phase transition'), {
      sessionId,
      from,
      to,
      validTargets: VALID_TRANSITIONS[from],
      error: error.message,
    });
    throw error;
  }

  logger.debug(scopedLog(LogContext.SESSION, 'phase transition'), {
    sessionId,
    from,
    to,
  });

  return to;
}

// =============================================================================
// Phase Predicates
// =============================================================================

/**
 * Check if a phase indicates the session is operational.
 * Operational phases are those where the session can accept work.
 */
export function isOperational(phase: SessionPhase): boolean {
  return phase === 'ready' || phase === 'executing' || phase === 'recording';
}

/**
 * Check if a phase indicates the session is busy.
 * Busy phases are those where the session is actively processing.
 */
export function isBusy(phase: SessionPhase): boolean {
  return phase === 'executing' || phase === 'resetting' || phase === 'initializing';
}

/**
 * Check if a phase is terminal (session is ending).
 */
export function isTerminal(phase: SessionPhase): boolean {
  return phase === 'closing';
}

/**
 * Check if a phase allows new instructions to be submitted.
 */
export function canAcceptInstructions(phase: SessionPhase): boolean {
  return phase === 'ready' || phase === 'recording';
}
