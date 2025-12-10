/**
 * Recording Lifecycle Handlers
 *
 * Handles start/stop/status operations for record mode.
 * This is the core recording lifecycle management.
 *
 * Graceful Degradation:
 * - Callback streaming uses circuit breaker pattern to prevent cascading failures
 * - Actions are always buffered locally, even when callbacks fail
 * - Circuit breaker opens after MAX_CALLBACK_FAILURES consecutive failures
 * - Circuit breaker resets after CIRCUIT_RESET_MS
 */

import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../../session';
import type { Config } from '../../config';
import { parseJsonBody, sendJson, sendError } from '../../middleware';
import { logger, metrics, scopedLog, LogContext } from '../../utils';
import { createRecordModeController } from '../../recording/controller';
import {
  initRecordingBuffer,
  bufferRecordedAction,
  getRecordedActions,
  getRecordedActionCount,
  clearRecordedActions,
} from '../../recording/buffer';
import type { RecordedAction } from '../../recording/types';
import type {
  StartRecordingRequest,
  StartRecordingResponse,
  StopRecordingResponse,
  RecordingStatusResponse,
} from './types';

/**
 * Circuit breaker state for callback streaming.
 * Prevents cascading failures when callback endpoint is unavailable.
 *
 * Temporal hardening:
 * - halfOpenInProgress flag prevents multiple concurrent half-open attempts
 * - State transitions are atomic (single mutation per check)
 */
interface CallbackCircuitBreaker {
  consecutiveFailures: number;
  isOpen: boolean;
  lastFailureTime: number | null;
  totalFailures: number;
  totalSuccesses: number;
  /** Prevents multiple concurrent half-open attempts */
  halfOpenInProgress: boolean;
}

// Circuit breaker configuration
const MAX_CALLBACK_FAILURES = 5;
const CIRCUIT_RESET_MS = 30_000; // 30 seconds

// Per-session circuit breaker state
const callbackCircuitBreakers = new Map<string, CallbackCircuitBreaker>();

/**
 * Get or create circuit breaker for a session.
 */
function getCircuitBreaker(sessionId: string): CallbackCircuitBreaker {
  let breaker = callbackCircuitBreakers.get(sessionId);
  if (!breaker) {
    breaker = {
      consecutiveFailures: 0,
      isOpen: false,
      lastFailureTime: null,
      totalFailures: 0,
      totalSuccesses: 0,
      halfOpenInProgress: false,
    };
    callbackCircuitBreakers.set(sessionId, breaker);
  }
  return breaker;
}

/**
 * Record a callback success - resets consecutive failure count.
 *
 * Temporal hardening: Also clears halfOpenInProgress flag since the
 * half-open test succeeded and we're fully closed now.
 */
function recordCallbackSuccess(sessionId: string): void {
  const breaker = getCircuitBreaker(sessionId);
  breaker.consecutiveFailures = 0;
  breaker.isOpen = false;
  breaker.halfOpenInProgress = false;
  breaker.totalSuccesses++;
}

/**
 * Record a callback failure - may trip the circuit breaker.
 * Returns true if circuit is now open (should skip future callbacks).
 *
 * Temporal hardening: Also handles half-open failure by re-opening circuit.
 */
function recordCallbackFailure(sessionId: string): boolean {
  const breaker = getCircuitBreaker(sessionId);
  breaker.consecutiveFailures++;
  breaker.totalFailures++;
  breaker.lastFailureTime = Date.now();

  // If we were in half-open state and failed, re-open the circuit
  if (breaker.halfOpenInProgress) {
    breaker.halfOpenInProgress = false;
    breaker.isOpen = true;
    logger.info(scopedLog(LogContext.RECORDING, 'circuit breaker half-open failed, re-opening'), {
      sessionId,
      totalFailures: breaker.totalFailures,
    });
    return true;
  }

  if (breaker.consecutiveFailures >= MAX_CALLBACK_FAILURES && !breaker.isOpen) {
    breaker.isOpen = true;
    logger.warn(scopedLog(LogContext.RECORDING, 'circuit breaker opened'), {
      sessionId,
      consecutiveFailures: breaker.consecutiveFailures,
      totalFailures: breaker.totalFailures,
      hint: `Callback streaming disabled after ${MAX_CALLBACK_FAILURES} consecutive failures. Will retry in ${CIRCUIT_RESET_MS / 1000}s.`,
    });
    // Track circuit breaker state change in metrics
    metrics.circuitBreakerStateChanges.inc({ session_id: sessionId, state: 'opened' });
  }

  return breaker.isOpen;
}

/**
 * Attempt to transition circuit breaker to half-open state.
 *
 * Temporal hardening: Uses atomic check-and-set pattern to prevent
 * multiple concurrent callbacks from all entering half-open state.
 * Only the first caller to enter half-open will return true.
 *
 * @returns true if this caller should attempt the half-open test
 */
function tryEnterHalfOpen(sessionId: string): boolean {
  const breaker = getCircuitBreaker(sessionId);

  // Not open, no need for half-open
  if (!breaker.isOpen) {
    return false;
  }

  // Already attempting half-open
  if (breaker.halfOpenInProgress) {
    return false;
  }

  // Check if enough time has passed
  if (!breaker.lastFailureTime) {
    return false;
  }

  const elapsed = Date.now() - breaker.lastFailureTime;
  if (elapsed < CIRCUIT_RESET_MS) {
    return false;
  }

  // Atomically claim the half-open attempt
  breaker.halfOpenInProgress = true;
  logger.info(scopedLog(LogContext.RECORDING, 'circuit breaker entering half-open state'), {
    sessionId,
    lastFailureAge: elapsed,
  });

  return true;
}

/**
 * Clean up circuit breaker state when recording stops.
 */
function cleanupCircuitBreaker(sessionId: string): void {
  callbackCircuitBreakers.delete(sessionId);
}

/**
 * Start recording endpoint
 *
 * POST /session/:id/record/start
 *
 * Starts recording user actions in the browser session.
 * Actions are captured and can be retrieved via the /actions endpoint
 * or streamed to a callback URL.
 *
 * Session phase transitions: ready -> recording
 *
 * Idempotency behavior:
 * - If already recording with the same recording_id, returns success (no-op)
 * - If already recording with a different recording_id, returns 409 Conflict
 * - This allows safe retries of recording start requests
 */
export async function handleRecordStart(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  config: Config
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);
    const body = await parseJsonBody(req, config);
    const request = body as unknown as StartRecordingRequest;

    // Idempotency: Check if already recording
    if (session.recordingController?.isRecording()) {
      // If the same recording_id is provided, this is an idempotent retry - return success
      if (request.recording_id && session.recordingId === request.recording_id) {
        logger.info(scopedLog(LogContext.RECORDING, 'idempotent start - already recording'), {
          sessionId,
          recordingId: session.recordingId,
          hint: 'Treating as successful retry of previous request',
        });

        const response: StartRecordingResponse = {
          recording_id: session.recordingId || request.recording_id,
          session_id: sessionId,
          started_at: session.recordingController.getState().startedAt || new Date().toISOString(),
        };

        sendJson(res, 200, response);
        return;
      }

      // Different recording_id or no recording_id provided - this is a conflict
      logger.warn(scopedLog(LogContext.RECORDING, 'already in progress'), {
        sessionId,
        recordingId: session.recordingId,
        requestedRecordingId: request.recording_id,
        hint: 'Call /record/stop to end the current recording before starting a new one',
      });
      sendJson(res, 409, {
        error: 'RECORDING_IN_PROGRESS',
        message: 'Recording is already in progress for this session',
        recording_id: session.recordingId,
        hint: 'Call /record/stop to end the current recording before starting a new one',
      });
      return;
    }

    // Create recording controller if not exists
    if (!session.recordingController) {
      session.recordingController = createRecordModeController(session.page, sessionId);
    }

    // Initialize action buffer for this session
    initRecordingBuffer(sessionId);

    logger.info(scopedLog(LogContext.RECORDING, 'starting'), {
      sessionId,
      hasCallback: !!request.callback_url,
      currentUrl: session.page?.url?.() || '(unknown)',
    });

    // Start recording with callback to buffer actions
    const recordingId = await session.recordingController.startRecording({
      sessionId,
      onAction: async (action: RecordedAction) => {
        // Buffer the action
        bufferRecordedAction(sessionId, action);

        // Track recording activity in metrics
        metrics.recordingActionsTotal.inc();

        logger.debug(scopedLog(LogContext.RECORDING, 'action captured'), {
          sessionId,
          recordingId,
          actionType: action.actionType,
          sequenceNum: action.sequenceNum,
          selector: action.selector?.primary,
          confidence: action.confidence,
        });

        // If callback URL provided, stream to it (with circuit breaker protection)
        if (request.callback_url) {
          const breaker = getCircuitBreaker(sessionId);

          // Check if we should attempt half-open (atomically claims the attempt)
          const attemptHalfOpen = tryEnterHalfOpen(sessionId);

          // Skip if circuit is open and we're not the half-open attempt
          if (breaker.isOpen && !attemptHalfOpen) {
            logger.debug(scopedLog(LogContext.RECORDING, 'skipping callback (circuit open)'), {
              sessionId,
              actionType: action.actionType,
              sequenceNum: action.sequenceNum,
              halfOpenInProgress: breaker.halfOpenInProgress,
            });
            return;
          }

          try {
            await streamActionToCallback(request.callback_url, action);
            recordCallbackSuccess(sessionId);
          } catch (err) {
            const circuitOpened = recordCallbackFailure(sessionId);

            // Classify failure reason for metrics
            const errorMessage = err instanceof Error ? err.message : String(err);
            let reason = 'unknown';
            if (errorMessage.includes('ECONNREFUSED') || errorMessage.includes('ENOTFOUND')) {
              reason = 'network';
            } else if (errorMessage.includes('timeout') || errorMessage.includes('ETIMEDOUT')) {
              reason = 'timeout';
            } else if (errorMessage.includes('returned')) {
              reason = 'http_error';
            }

            metrics.recordingCallbackFailures.inc({ reason });

            logger.warn(scopedLog(LogContext.RECORDING, 'callback streaming failed'), {
              sessionId,
              callbackUrl: request.callback_url,
              error: errorMessage,
              failureReason: reason,
              consecutiveFailures: breaker.consecutiveFailures,
              circuitOpen: circuitOpened,
              hint: circuitOpened
                ? 'Circuit breaker opened - actions still buffered locally'
                : 'Callback endpoint may be unreachable or returning errors',
            });
          }
        }
      },
      onError: (error: Error) => {
        logger.error(scopedLog(LogContext.RECORDING, 'capture error'), {
          sessionId,
          recordingId,
          error: error.message,
          hint: 'Recording may have encountered a page issue or event listener failure',
        });
      },
    });

    // Store recording ID on session
    session.recordingId = recordingId;

    // Update session phase
    sessionManager.setSessionPhase(sessionId, 'recording');

    // Update recording session metric
    metrics.recordingSessionsActive.inc();

    logger.info(scopedLog(LogContext.RECORDING, 'started'), {
      sessionId,
      recordingId,
      phase: 'recording',
      url: session.page?.url?.() || '(unknown)',
    });

    const response: StartRecordingResponse = {
      recording_id: recordingId,
      session_id: sessionId,
      started_at: new Date().toISOString(),
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/start`);
  }
}

/**
 * Stop recording endpoint
 *
 * POST /session/:id/record/stop
 *
 * Stops the current recording session and returns summary.
 *
 * Session phase transitions: recording -> ready
 *
 * Idempotency behavior:
 * - If recording is not active, returns success with action_count: 0
 * - This allows safe retries of recording stop requests
 * - Calling stop twice is safe and produces consistent results
 */
export async function handleRecordStop(
  _req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);

    // Idempotency: If not recording, treat as successful no-op
    // This handles retries where the first request succeeded but response was lost
    if (!session.recordingController?.isRecording()) {
      logger.info(scopedLog(LogContext.RECORDING, 'idempotent stop - no active recording'), {
        sessionId,
        phase: session.phase,
        hint: 'No recording active, treating as successful stop (idempotent)',
      });

      // Return success with zero action count
      // Use the last known recording ID if available
      const response: StopRecordingResponse = {
        recording_id: session.recordingId || 'unknown',
        session_id: sessionId,
        action_count: 0,
        stopped_at: new Date().toISOString(),
      };

      sendJson(res, 200, response);
      return;
    }

    const recordingId = session.recordingId;
    const result = await session.recordingController.stopRecording();

    // Clean up circuit breaker state
    cleanupCircuitBreaker(sessionId);

    // Update recording session metric
    metrics.recordingSessionsActive.dec();

    // Update session phase
    sessionManager.setSessionPhase(sessionId, 'ready');

    logger.info(scopedLog(LogContext.RECORDING, 'stopped'), {
      sessionId,
      recordingId: recordingId || result.recordingId,
      actionCount: result.actionCount,
      phase: 'ready',
      finalUrl: session.page?.url?.() || '(unknown)',
    });

    // Clear recording ID from session
    session.recordingId = undefined;

    const response: StopRecordingResponse = {
      recording_id: recordingId || result.recordingId,
      session_id: sessionId,
      action_count: result.actionCount,
      stopped_at: new Date().toISOString(),
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/stop`);
  }
}

/**
 * Get recording status endpoint
 *
 * GET /session/:id/record/status
 *
 * Returns current recording status for the session.
 */
export async function handleRecordStatus(
  _req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);

    const state = session.recordingController?.getState();
    const bufferedCount = getRecordedActionCount(sessionId);

    const response: RecordingStatusResponse = {
      session_id: sessionId,
      is_recording: state?.isRecording || false,
      recording_id: state?.recordingId,
      action_count: bufferedCount || state?.actionCount || 0,
      started_at: state?.startedAt,
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/status`);
  }
}

/**
 * Get recorded actions endpoint
 *
 * GET /session/:id/record/actions
 *
 * Returns all buffered actions for the session.
 * Optionally clears the buffer after retrieval.
 */
export async function handleRecordActions(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager
): Promise<void> {
  try {
    // Verify session exists
    sessionManager.getSession(sessionId);

    const actions = getRecordedActions(sessionId);

    // Check for clear query param (parse URL manually since we don't have query parser)
    const url = new URL(req.url || '', `http://localhost`);
    const shouldClear = url.searchParams.get('clear') === 'true';

    if (shouldClear) {
      clearRecordedActions(sessionId);
    }

    sendJson(res, 200, {
      session_id: sessionId,
      actions,
      count: actions.length,
    });
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/actions`);
  }
}

// Callback streaming timeout (5 seconds - callbacks should be fast)
const CALLBACK_TIMEOUT_MS = 5_000;

/**
 * Stream an action to a callback URL with timeout.
 *
 * Hardened: Uses AbortController to enforce timeout and prevent
 * indefinite hangs if the callback endpoint is slow or unresponsive.
 */
async function streamActionToCallback(callbackUrl: string, action: RecordedAction): Promise<void> {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), CALLBACK_TIMEOUT_MS);

  try {
    const response = await fetch(callbackUrl, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(action),
      signal: controller.signal,
    });

    if (!response.ok) {
      throw new Error(`Callback returned ${response.status}: ${response.statusText}`);
    }

    logger.debug('recording: action streamed', {
      status: response.status,
      actionType: action.actionType,
      sequenceNum: action.sequenceNum,
    });
  } finally {
    clearTimeout(timeoutId);
  }
}
