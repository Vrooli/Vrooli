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
 */
interface CallbackCircuitBreaker {
  consecutiveFailures: number;
  isOpen: boolean;
  lastFailureTime: number | null;
  totalFailures: number;
  totalSuccesses: number;
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
    };
    callbackCircuitBreakers.set(sessionId, breaker);
  }
  return breaker;
}

/**
 * Record a callback success - resets consecutive failure count.
 */
function recordCallbackSuccess(sessionId: string): void {
  const breaker = getCircuitBreaker(sessionId);
  breaker.consecutiveFailures = 0;
  breaker.isOpen = false;
  breaker.totalSuccesses++;
}

/**
 * Record a callback failure - may trip the circuit breaker.
 * Returns true if circuit is now open (should skip future callbacks).
 */
function recordCallbackFailure(sessionId: string): boolean {
  const breaker = getCircuitBreaker(sessionId);
  breaker.consecutiveFailures++;
  breaker.totalFailures++;
  breaker.lastFailureTime = Date.now();

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
 * Check if circuit breaker should be reset (half-open state).
 */
function shouldResetCircuitBreaker(sessionId: string): boolean {
  const breaker = getCircuitBreaker(sessionId);
  if (!breaker.isOpen || !breaker.lastFailureTime) {
    return false;
  }

  const elapsed = Date.now() - breaker.lastFailureTime;
  return elapsed >= CIRCUIT_RESET_MS;
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

    // Check if already recording
    if (session.recordingController?.isRecording()) {
      logger.warn(scopedLog(LogContext.RECORDING, 'already in progress'), {
        sessionId,
        recordingId: session.recordingId,
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

          // Check if circuit breaker should be reset (half-open state)
          if (breaker.isOpen && shouldResetCircuitBreaker(sessionId)) {
            logger.info(scopedLog(LogContext.RECORDING, 'circuit breaker half-open, retrying'), {
              sessionId,
              lastFailureAge: Date.now() - (breaker.lastFailureTime || 0),
            });
            breaker.isOpen = false; // Try again
          }

          // Skip if circuit is open
          if (breaker.isOpen) {
            logger.debug(scopedLog(LogContext.RECORDING, 'skipping callback (circuit open)'), {
              sessionId,
              actionType: action.actionType,
              sequenceNum: action.sequenceNum,
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
 */
export async function handleRecordStop(
  _req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);

    if (!session.recordingController?.isRecording()) {
      logger.warn(scopedLog(LogContext.RECORDING, 'stop called with no recording'), {
        sessionId,
        phase: session.phase,
        hint: 'No recording is currently active for this session',
      });
      sendJson(res, 404, {
        error: 'NO_RECORDING',
        message: 'No recording in progress for this session',
        hint: 'Call /record/start to begin recording before stopping',
      });
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

/**
 * Stream an action to a callback URL
 */
async function streamActionToCallback(callbackUrl: string, action: RecordedAction): Promise<void> {
  const response = await fetch(callbackUrl, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(action),
  });

  if (!response.ok) {
    throw new Error(`Callback returned ${response.status}: ${response.statusText}`);
  }

  logger.debug('recording: action streamed', {
    status: response.status,
    actionType: action.actionType,
    sequenceNum: action.sequenceNum,
  });
}
