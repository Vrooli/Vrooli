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

    // Start frame streaming if callback URL provided
    if (request.frame_callback_url) {
      startFrameStreaming(sessionId, sessionManager, {
        callbackUrl: request.frame_callback_url,
        quality: request.frame_quality,
        fps: request.frame_fps,
      });
    }

    logger.info(scopedLog(LogContext.RECORDING, 'started'), {
      sessionId,
      recordingId,
      phase: 'recording',
      url: session.page?.url?.() || '(unknown)',
      frameStreaming: !!request.frame_callback_url,
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

    // Stop frame streaming if active
    stopFrameStreaming(sessionId);

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

// =============================================================================
// Frame Streaming (WebSocket Binary)
// =============================================================================

/** Minimal WebSocket interface - subset of ws module for our needs */
interface FrameWebSocket {
  readyState: number;
  send(data: Buffer): void;
  close(): void;
  on(event: 'open', listener: () => void): void;
  on(event: 'close', listener: () => void): void;
  on(event: 'error', listener: (err: Error) => void): void;
}

// Use dynamic require - ws is definitely installed as a dependency
// eslint-disable-next-line @typescript-eslint/no-require-imports, @typescript-eslint/no-var-requires
const WS: new (url: string) => FrameWebSocket = require('ws');

/**
 * Frame streaming state for a session.
 * Uses WebSocket for efficient binary frame delivery.
 */
interface FrameStreamState {
  /** Whether streaming is active */
  isStreaming: boolean;
  /** AbortController to stop the streaming loop */
  abortController: AbortController;
  /** Last frame buffer for quick byte-level comparison */
  lastFrameBuffer: Buffer | null;
  /** Frame quality (0-100) */
  quality: number;
  /** Target FPS (user-requested) */
  targetFps: number;
  /** Current adaptive FPS (may be lower than target if client is slow) */
  currentFps: number;
  /** Timestamp of last frame sent (for adaptive FPS calculation) */
  lastFrameSentAt: number;
  /** Rolling average of frame processing times (ms) */
  avgFrameTime: number;
  /** Number of frames sent (for averaging) */
  frameCount: number;
  /** WebSocket connection to API */
  ws: FrameWebSocket | null;
  /** WebSocket URL for reconnection */
  wsUrl: string;
  /** Consecutive failure count for circuit breaker */
  consecutiveFailures: number;
  /** Whether WebSocket is connected and ready */
  wsReady: boolean;
  /** Screenshot scale: 'css' for 1x, 'device' for devicePixelRatio */
  scale: 'css' | 'device';
}

/** Per-session frame streaming state */
const frameStreamStates = new Map<string, FrameStreamState>();

/** Max consecutive frame failures before pausing */
const MAX_FRAME_FAILURES = 3;

/** WebSocket reconnection delay */
const WS_RECONNECT_DELAY_MS = 1000;

/** Minimum FPS floor for adaptive rate control */
const MIN_ADAPTIVE_FPS = 2;
/** Alpha for exponential moving average of frame times */
const FRAME_TIME_EMA_ALPHA = 0.3;
/** Threshold: if frame time exceeds target interval by this factor, reduce FPS */
const SLOWDOWN_THRESHOLD = 1.2;
/** Threshold: if frame time is below target interval by this factor, increase FPS */
const SPEEDUP_THRESHOLD = 0.7;
/** How often to log adaptive FPS changes (every N frames) */
const ADAPTIVE_LOG_INTERVAL = 30;

/**
 * Build WebSocket URL from HTTP callback URL.
 * Converts http://host:port/api/v1/recordings/live/{sessionId}/frame
 * to ws://host:port/ws/recording/{sessionId}/frames
 */
function buildWebSocketUrl(callbackUrl: string, sessionId: string): string {
  try {
    const url = new URL(callbackUrl);
    const protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${protocol}//${url.host}/ws/recording/${sessionId}/frames`;
  } catch {
    // Fallback: assume localhost API
    return `ws://127.0.0.1:8080/ws/recording/${sessionId}/frames`;
  }
}

/**
 * Start frame streaming for a recording session.
 * Uses WebSocket for efficient binary frame delivery.
 * Identical frames (same buffer content) are skipped to save bandwidth.
 */
export function startFrameStreaming(
  sessionId: string,
  sessionManager: SessionManager,
  options: {
    callbackUrl: string;
    quality?: number;
    fps?: number;
    /** Screenshot scale: 'css' for 1x (default), 'device' for devicePixelRatio */
    scale?: 'css' | 'device';
  }
): void {
  // Stop any existing stream for this session
  stopFrameStreaming(sessionId);

  const quality = options.quality ?? 65;
  const fps = Math.min(Math.max(options.fps ?? 6, 1), 30); // Clamp 1-30 FPS
  const scale = options.scale ?? 'css';
  const intervalMs = Math.floor(1000 / fps);
  const wsUrl = buildWebSocketUrl(options.callbackUrl, sessionId);

  const state: FrameStreamState = {
    isStreaming: true,
    abortController: new AbortController(),
    lastFrameBuffer: null,
    quality,
    targetFps: fps,
    currentFps: fps,
    lastFrameSentAt: 0,
    avgFrameTime: 0,
    frameCount: 0,
    ws: null,
    wsUrl,
    consecutiveFailures: 0,
    wsReady: false,
    scale,
  };

  frameStreamStates.set(sessionId, state);

  logger.info(scopedLog(LogContext.RECORDING, 'frame streaming started (WebSocket binary, adaptive FPS)'), {
    sessionId,
    targetFps: fps,
    quality,
    scale,
    initialIntervalMs: intervalMs,
    wsUrl,
    adaptiveFpsEnabled: true,
    minFps: MIN_ADAPTIVE_FPS,
  });

  // Connect WebSocket and start streaming loop
  connectFrameWebSocket(sessionId, state);
  void runFrameStreamLoop(sessionId, sessionManager, state, intervalMs);
}

/**
 * Connect WebSocket for frame streaming.
 */
function connectFrameWebSocket(sessionId: string, state: FrameStreamState): void {
  if (!state.isStreaming) return;

  try {
    const ws = new WS(state.wsUrl);
    state.ws = ws;

    ws.on('open', () => {
      state.wsReady = true;
      state.consecutiveFailures = 0;
      logger.info(scopedLog(LogContext.RECORDING, 'frame WebSocket connected'), { sessionId });
    });

    ws.on('close', () => {
      state.wsReady = false;
      logger.debug(scopedLog(LogContext.RECORDING, 'frame WebSocket closed'), { sessionId });

      // Reconnect if still streaming
      if (state.isStreaming) {
        setTimeout(() => connectFrameWebSocket(sessionId, state), WS_RECONNECT_DELAY_MS);
      }
    });

    ws.on('error', (err: Error) => {
      state.wsReady = false;
      logger.warn(scopedLog(LogContext.RECORDING, 'frame WebSocket error'), {
        sessionId,
        error: err.message,
      });
    });
  } catch (err) {
    logger.warn(scopedLog(LogContext.RECORDING, 'failed to create frame WebSocket'), {
      sessionId,
      error: err instanceof Error ? err.message : String(err),
    });

    // Retry connection
    if (state.isStreaming) {
      setTimeout(() => connectFrameWebSocket(sessionId, state), WS_RECONNECT_DELAY_MS);
    }
  }
}

/**
 * Stop frame streaming for a session.
 */
export function stopFrameStreaming(sessionId: string): void {
  const state = frameStreamStates.get(sessionId);
  if (state) {
    state.isStreaming = false;
    state.abortController.abort();

    // Close WebSocket connection
    if (state.ws) {
      try {
        state.ws.close();
      } catch {
        // Ignore close errors
      }
      state.ws = null;
    }

    frameStreamStates.delete(sessionId);

    logger.info(scopedLog(LogContext.RECORDING, 'frame streaming stopped'), {
      sessionId,
    });
  }
}

/**
 * The main frame streaming loop.
 * Captures frames and sends via WebSocket as binary data.
 * Uses adaptive FPS to match client processing capability.
 */
async function runFrameStreamLoop(
  sessionId: string,
  sessionManager: SessionManager,
  state: FrameStreamState,
  _initialIntervalMs: number
): Promise<void> {
  while (state.isStreaming && !state.abortController.signal.aborted) {
    const loopStart = performance.now();

    // Calculate current interval from adaptive FPS
    const currentIntervalMs = Math.floor(1000 / state.currentFps);

    try {
      // Check if session still exists
      let session;
      try {
        session = sessionManager.getSession(sessionId);
      } catch {
        // Session closed, stop streaming
        stopFrameStreaming(sessionId);
        return;
      }

      // Skip if WebSocket not connected (readyState 1 = OPEN)
      if (!state.wsReady || !state.ws || state.ws.readyState !== 1) {
        await sleepUntilNextFrame(loopStart, currentIntervalMs, state.abortController.signal);
        continue;
      }

      // Skip if too many consecutive failures (circuit breaker)
      if (state.consecutiveFailures >= MAX_FRAME_FAILURES) {
        // Wait longer before retrying
        await sleep(currentIntervalMs * 5, state.abortController.signal);
        state.consecutiveFailures = 0; // Reset and retry
        continue;
      }

      // Capture frame as raw buffer (no base64 encoding!)
      const buffer = await captureFrameBuffer(session.page, state.quality, state.scale);

      if (!buffer) {
        // Page not ready or error, skip this frame
        await sleepUntilNextFrame(loopStart, currentIntervalMs, state.abortController.signal);
        continue;
      }

      // Skip if frame content unchanged (fast buffer comparison)
      if (isFrameUnchanged(buffer, state)) {
        await sleepUntilNextFrame(loopStart, currentIntervalMs, state.abortController.signal);
        continue;
      }

      // Update last frame state
      state.lastFrameBuffer = buffer;

      // Send raw binary frame over WebSocket (no JSON, no base64!)
      state.ws.send(buffer);
      state.consecutiveFailures = 0;

      // Track frame timing for adaptive FPS
      const now = performance.now();
      if (state.lastFrameSentAt > 0) {
        const actualFrameTime = now - state.lastFrameSentAt;
        state.frameCount++;

        // Update exponential moving average of frame time
        if (state.avgFrameTime === 0) {
          state.avgFrameTime = actualFrameTime;
        } else {
          state.avgFrameTime = FRAME_TIME_EMA_ALPHA * actualFrameTime +
            (1 - FRAME_TIME_EMA_ALPHA) * state.avgFrameTime;
        }

        // Adaptive FPS adjustment every few frames to avoid oscillation
        if (state.frameCount % 5 === 0) {
          const targetInterval = 1000 / state.targetFps;
          const avgFrameTime = state.avgFrameTime;

          if (avgFrameTime > targetInterval * SLOWDOWN_THRESHOLD) {
            // Frames taking too long - reduce FPS
            const newFps = Math.max(MIN_ADAPTIVE_FPS, state.currentFps - 1);
            if (newFps !== state.currentFps) {
              state.currentFps = newFps;
              if (state.frameCount % ADAPTIVE_LOG_INTERVAL === 0) {
                logger.debug(scopedLog(LogContext.RECORDING, 'adaptive FPS reduced'), {
                  sessionId,
                  currentFps: state.currentFps,
                  targetFps: state.targetFps,
                  avgFrameTimeMs: Math.round(avgFrameTime),
                });
              }
            }
          } else if (avgFrameTime < targetInterval * SPEEDUP_THRESHOLD && state.currentFps < state.targetFps) {
            // Frames processing fast - increase FPS toward target
            const newFps = Math.min(state.targetFps, state.currentFps + 1);
            if (newFps !== state.currentFps) {
              state.currentFps = newFps;
              if (state.frameCount % ADAPTIVE_LOG_INTERVAL === 0) {
                logger.debug(scopedLog(LogContext.RECORDING, 'adaptive FPS increased'), {
                  sessionId,
                  currentFps: state.currentFps,
                  targetFps: state.targetFps,
                  avgFrameTimeMs: Math.round(avgFrameTime),
                });
              }
            }
          }
        }
      }
      state.lastFrameSentAt = now;

    } catch (err) {
      if (state.abortController.signal.aborted) {
        return; // Normal shutdown
      }

      state.consecutiveFailures++;
      const message = err instanceof Error ? err.message : String(err);

      logger.warn(scopedLog(LogContext.RECORDING, 'frame streaming error'), {
        sessionId,
        error: message,
        consecutiveFailures: state.consecutiveFailures,
      });
    }

    await sleepUntilNextFrame(loopStart, currentIntervalMs, state.abortController.signal);
  }
}

/**
 * Check if frame is unchanged from last frame.
 * Uses fast buffer length check before expensive byte comparison.
 */
function isFrameUnchanged(buffer: Buffer, state: FrameStreamState): boolean {
  if (!state.lastFrameBuffer) {
    return false;
  }

  // Fast path: different length means different content
  if (buffer.length !== state.lastFrameBuffer.length) {
    return false;
  }

  // Same length: do byte comparison (still faster than MD5 for most cases)
  return buffer.equals(state.lastFrameBuffer);
}

/**
 * Screenshot timeout - generous to handle complex pages.
 * The adaptive FPS system will reduce framerate if captures are slow.
 */
const SCREENSHOT_TIMEOUT_MS = 400;

/**
 * Capture a frame as raw JPEG buffer.
 * Only captures the visible viewport (not full page) to reduce bandwidth and improve performance.
 * No base64 encoding - send raw bytes over WebSocket.
 * Includes timeout to prevent blocking the frame loop if page is slow to render.
 *
 * @param scale - 'css' captures at 1x logical pixels (smaller, faster),
 *                'device' captures at devicePixelRatio (sharper on HiDPI, but 4x larger on 2x displays)
 */
async function captureFrameBuffer(
  page: import('playwright').Page,
  quality: number,
  scale: 'css' | 'device' = 'css'
): Promise<Buffer | null> {
  try {
    // Get viewport dimensions for clipping - only capture what's visible
    const viewport = page.viewportSize();
    if (!viewport) {
      // Fallback if viewport not available (shouldn't happen normally)
      return await page.screenshot({
        type: 'jpeg',
        quality,
        timeout: SCREENSHOT_TIMEOUT_MS,
        scale,
      });
    }

    // Capture only the viewport area, not the full page
    // This significantly reduces bandwidth for pages with scrollable content
    // Using 'css' scale by default captures at 1x resolution (not devicePixelRatio)
    // which is ~4x smaller on HiDPI displays while still looking good for preview
    return await page.screenshot({
      type: 'jpeg',
      quality,
      timeout: SCREENSHOT_TIMEOUT_MS,
      scale,
      clip: {
        x: 0,
        y: 0,
        width: viewport.width,
        height: viewport.height,
      },
    });
  } catch {
    // Timeout or other error - skip this frame
    return null;
  }
}

/**
 * Sleep until the next frame capture time.
 */
async function sleepUntilNextFrame(
  loopStart: number,
  intervalMs: number,
  signal: AbortSignal
): Promise<void> {
  const elapsed = performance.now() - loopStart;
  const sleepTime = Math.max(0, intervalMs - elapsed);
  if (sleepTime > 0) {
    await sleep(sleepTime, signal);
  }
}

/**
 * Sleep with abort signal support.
 */
function sleep(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    const timeoutId = setTimeout(resolve, ms);
    if (signal) {
      signal.addEventListener('abort', () => {
        clearTimeout(timeoutId);
        reject(new Error('Aborted'));
      }, { once: true });
    }
  });
}
