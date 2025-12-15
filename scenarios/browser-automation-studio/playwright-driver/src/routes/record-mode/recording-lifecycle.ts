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
import WebSocket from 'ws';
import { parseJsonBody, sendJson, sendError } from '../../middleware';
import { logger, metrics, scopedLog, LogContext } from '../../utils';
import { createRecordModeController } from '../../recording/controller';
import {
  initRecordingBuffer,
  bufferTimelineEntry,
  getTimelineEntries,
  getTimelineEntryCount,
  clearTimelineEntries,
} from '../../recording/buffer';
import { timelineEntryToJson, type TimelineEntry, ActionType } from '../../proto/recording';
import type {
  StartRecordingRequest,
  StartRecordingResponse,
  StopRecordingResponse,
  RecordingStatusResponse,
  StreamSettingsRequest,
  StreamSettingsResponse,
} from './types';
import { PerfCollector } from '../../performance';
import { loadConfig } from '../../config';
import {
  createFpsController,
  processFrame as processFpsFrame,
  handleTimeout as handleFpsTimeout,
  getIntervalMs,
  getCurrentFps,
  FpsControllerState,
  FpsControllerConfig,
} from '../../fps';
import {
  startScreencastStreaming,
  type ScreencastHandle,
  type ScreencastState,
} from './screencast-streaming';

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

    // Start recording with callback to buffer entries
    const recordingId = await session.recordingController.startRecording({
      sessionId,
      onEntry: async (entry: TimelineEntry) => {
        // Buffer the entry (proto TimelineEntry)
        bufferTimelineEntry(sessionId, entry);

        // Track recording activity in metrics
        metrics.recordingActionsTotal.inc();

        // Extract action type name for logging
        const actionTypeName = ActionType[entry.action?.type ?? ActionType.UNSPECIFIED] ?? 'UNKNOWN';

        logger.debug(scopedLog(LogContext.RECORDING, 'entry captured'), {
          sessionId,
          recordingId,
          actionType: actionTypeName,
          sequenceNum: entry.sequenceNum,
          confidence: entry.action?.metadata?.confidence,
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
              actionType: actionTypeName,
              sequenceNum: entry.sequenceNum,
              halfOpenInProgress: breaker.halfOpenInProgress,
            });
            return;
          }

          try {
            await streamEntryToCallback(request.callback_url, entry);
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
                ? 'Circuit breaker opened - entries still buffered locally'
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
    await stopFrameStreaming(sessionId);

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
    const bufferedCount = getTimelineEntryCount(sessionId);

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
 * Returns all buffered actions for the session as TimelineEntry format.
 * Optionally clears the buffer after retrieval.
 *
 * Wire format: Returns proto TimelineEntry JSON format for interoperability.
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

    // Get buffered TimelineEntry objects (already proto format)
    const entries = getTimelineEntries(sessionId);

    // Check for clear query param (parse URL manually since we don't have query parser)
    const url = new URL(req.url || '', `http://localhost`);
    const shouldClear = url.searchParams.get('clear') === 'true';

    if (shouldClear) {
      clearTimelineEntries(sessionId);
    }

    // Convert to JSON wire format (snake_case)
    const entriesJson = entries.map(timelineEntryToJson);

    sendJson(res, 200, {
      session_id: sessionId,
      entries: entriesJson,
      count: entriesJson.length,
    });
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/actions`);
  }
}

// Callback streaming timeout (5 seconds - callbacks should be fast)
const CALLBACK_TIMEOUT_MS = 5_000;

/**
 * Stream a TimelineEntry to a callback URL with timeout.
 *
 * Hardened: Uses AbortController to enforce timeout and prevent
 * indefinite hangs if the callback endpoint is slow or unresponsive.
 *
 * Wire format: Serializes proto TimelineEntry to JSON (snake_case)
 * for interoperability with the Go API.
 */
async function streamEntryToCallback(callbackUrl: string, entry: TimelineEntry): Promise<void> {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), CALLBACK_TIMEOUT_MS);

  try {
    // Serialize proto to JSON wire format (snake_case)
    const wireFormat = timelineEntryToJson(entry);

    const response = await fetch(callbackUrl, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(wireFormat),
      signal: controller.signal,
    });

    if (!response.ok) {
      throw new Error(`Callback returned ${response.status}: ${response.statusText}`);
    }

    const actionTypeName = ActionType[entry.action?.type ?? ActionType.UNSPECIFIED] ?? 'UNKNOWN';
    logger.debug('recording: entry streamed', {
      status: response.status,
      actionType: actionTypeName,
      sequenceNum: entry.sequenceNum,
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

const WS: new (url: string) => FrameWebSocket = WebSocket as unknown as new (url: string) => FrameWebSocket;

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
  /** Number of frames sent (for logging/metrics) */
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
  /** Performance collector - always created for runtime perf mode toggling */
  perfCollector: PerfCollector;
  /** Whether to include timing headers in frames (toggled via UI) */
  includePerfHeaders: boolean;
  /** FPS controller state (handles adaptive FPS logic) - only used for polling mode */
  fpsState: FpsControllerState;
  /** FPS controller config - only used for polling mode */
  fpsConfig: FpsControllerConfig;
  /** Whether using CDP screencast (vs legacy polling) */
  useScreencast: boolean;
  /** Handle to stop screencast and cleanup - only set when useScreencast is true */
  screencastHandle?: ScreencastHandle;
}

/** Per-session frame streaming state */
const frameStreamStates = new Map<string, FrameStreamState>();

/** Max consecutive frame failures before pausing */
const MAX_FRAME_FAILURES = 3;

/** WebSocket reconnection delay */
const WS_RECONNECT_DELAY_MS = 1000;

/** How often to log FPS changes (every N frames) */
const FPS_LOG_INTERVAL = 30;

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
 *
 * Uses CDP screencast by default for push-based frame delivery (30-60 FPS).
 * Falls back to polling-based screenshot capture if screencast fails.
 *
 * WebSocket is used for efficient binary frame delivery to the API.
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
  // Stop any existing stream for this session (fire and forget - don't block)
  void stopFrameStreaming(sessionId);

  const quality = options.quality ?? 65;
  const fps = Math.min(Math.max(options.fps ?? 30, 1), 60); // Clamp 1-60 FPS (higher default for screencast)
  const scale = options.scale ?? 'css';
  const wsUrl = buildWebSocketUrl(options.callbackUrl, sessionId);

  // Load config for feature flags and performance settings
  const config = loadConfig();

  // Always create performance collector (it's cheap - just a ring buffer).
  const perfCollector = PerfCollector.fromConfig(sessionId, config, fps);

  // Initialize FPS controller (only used for polling fallback mode)
  const { state: fpsState, config: fpsConfig } = createFpsController(fps, {
    minFps: 2,
    maxFps: Math.min(60, fps * 2),
    targetUtilization: 0.7,
    smoothing: 0.25,
    adjustmentInterval: 3,
  });

  const state: FrameStreamState = {
    isStreaming: true,
    abortController: new AbortController(),
    lastFrameBuffer: null,
    quality,
    targetFps: fps,
    frameCount: 0,
    ws: null,
    wsUrl,
    consecutiveFailures: 0,
    wsReady: false,
    scale,
    perfCollector,
    includePerfHeaders: config.performance.enabled && config.performance.includeTimingHeaders,
    fpsState,
    fpsConfig,
    useScreencast: config.frameStreaming.useScreencast, // Feature flag from config
  };

  frameStreamStates.set(sessionId, state);

  // Connect WebSocket first (needed for both modes)
  connectFrameWebSocket(sessionId, state);

  // Start streaming based on mode
  if (state.useScreencast) {
    void startScreencastMode(sessionId, sessionManager, state, config);
  } else {
    logger.info(scopedLog(LogContext.RECORDING, 'frame streaming started (polling mode)'), {
      sessionId,
      targetFps: fps,
      quality,
      scale,
      wsUrl,
    });
    void runFrameStreamLoop(sessionId, sessionManager, state);
  }
}

/**
 * Start CDP screencast streaming mode.
 * Falls back to polling mode if screencast fails (and fallback is enabled).
 */
async function startScreencastMode(
  sessionId: string,
  sessionManager: SessionManager,
  state: FrameStreamState,
  config: ReturnType<typeof loadConfig>
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);
    const viewport = session.page.viewportSize() ?? { width: 1280, height: 720 };

    // Wait briefly for WebSocket to connect
    await new Promise((resolve) => setTimeout(resolve, 100));

    // Create a state proxy that keeps screencast in sync with our state
    const screencastState: ScreencastState = {
      get ws() {
        return state.ws;
      },
      get wsReady() {
        return state.wsReady;
      },
      get isStreaming() {
        return state.isStreaming;
      },
      frameCount: state.frameCount,
    };

    // Define a setter to update frame count in our state
    Object.defineProperty(screencastState, 'frameCount', {
      get: () => state.frameCount,
      set: (value: number) => {
        state.frameCount = value;
      },
    });

    state.screencastHandle = await startScreencastStreaming(
      session.page,
      sessionId,
      screencastState,
      {
        format: 'jpeg',
        quality: state.quality,
        maxWidth: viewport.width,
        maxHeight: viewport.height,
        everyNthFrame: 1, // Every frame - Chrome handles change detection
      }
    );

    logger.info(scopedLog(LogContext.RECORDING, 'frame streaming started (CDP screencast)'), {
      sessionId,
      quality: state.quality,
      viewport,
      wsUrl: state.wsUrl,
      mode: 'screencast',
    });
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);

    // Screencast failed - check if we should fall back to polling
    if (config.frameStreaming.fallbackToPolling) {
      logger.warn(scopedLog(LogContext.RECORDING, 'CDP screencast failed, falling back to polling'), {
        sessionId,
        error: message,
        fallback: true,
      });

      state.useScreencast = false;
      void runFrameStreamLoop(sessionId, sessionManager, state);
    } else {
      logger.error(scopedLog(LogContext.RECORDING, 'CDP screencast failed (no fallback)'), {
        sessionId,
        error: message,
        fallback: false,
        hint: 'Set FRAME_STREAMING_FALLBACK=true to enable polling fallback',
      });

      // Clean up state since we're not streaming
      state.isStreaming = false;
      frameStreamStates.delete(sessionId);
    }
  }
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
 * Update frame streaming settings for an active session.
 * Only updates quality and fps - scale changes require session restart.
 *
 * @returns true if settings were updated, false if no active stream
 */
export function updateFrameStreamSettings(
  sessionId: string,
  options: {
    quality?: number;
    fps?: number;
    perfMode?: boolean;
  }
): boolean {
  const state = frameStreamStates.get(sessionId);
  if (!state || !state.isStreaming) {
    return false;
  }

  let changed = false;

  if (options.quality !== undefined) {
    const newQuality = Math.min(Math.max(options.quality, 1), 100);
    if (newQuality !== state.quality) {
      state.quality = newQuality;
      changed = true;
    }
  }

  if (options.fps !== undefined) {
    const newFps = Math.min(Math.max(options.fps, 1), 60);
    if (newFps !== state.targetFps) {
      state.targetFps = newFps;
      // Update FPS config with new max (allow up to 2x target, capped at 60)
      state.fpsConfig = {
        ...state.fpsConfig,
        maxFps: Math.min(60, newFps * 2),
      };
      changed = true;
    }
  }

  if (options.perfMode !== undefined && options.perfMode !== state.includePerfHeaders) {
    state.includePerfHeaders = options.perfMode;
    changed = true;
  }

  if (changed) {
    logger.info(scopedLog(LogContext.RECORDING, 'frame stream settings updated'), {
      sessionId,
      quality: state.quality,
      targetFps: state.targetFps,
      currentFps: getCurrentFps(state.fpsState),
      scale: state.scale,
      perfMode: state.includePerfHeaders,
    });
  }

  return changed;
}

/**
 * Get current frame streaming settings for a session.
 * Returns null if no active stream.
 */
export function getFrameStreamSettings(sessionId: string): {
  quality: number;
  fps: number;
  scale: 'css' | 'device';
  currentFps: number;
  isStreaming: boolean;
  perfMode: boolean;
} | null {
  const state = frameStreamStates.get(sessionId);
  if (!state) {
    return null;
  }

  return {
    quality: state.quality,
    fps: state.targetFps,
    scale: state.scale,
    currentFps: getCurrentFps(state.fpsState),
    isStreaming: state.isStreaming,
    perfMode: state.includePerfHeaders,
  };
}

/**
 * Update stream settings endpoint
 *
 * POST /session/:id/record/stream-settings
 *
 * Updates stream settings for an active session.
 * Quality and FPS can be updated immediately. Scale changes require session restart.
 */
export async function handleStreamSettings(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  config: Config
): Promise<void> {
  try {
    // Verify session exists
    sessionManager.getSession(sessionId);

    const body = await parseJsonBody(req, config);
    const request = body as unknown as StreamSettingsRequest;

    // Get current settings (may be null if no stream active)
    const currentSettings = getFrameStreamSettings(sessionId);

    // Handle case where no stream is active
    if (!currentSettings) {
      logger.info(scopedLog(LogContext.RECORDING, 'stream settings update - no active stream'), {
        sessionId,
        requestedQuality: request.quality,
        requestedFps: request.fps,
      });

      // Return a response indicating no active stream
      const response: StreamSettingsResponse = {
        session_id: sessionId,
        quality: request.quality ?? 55,
        fps: request.fps ?? 6,
        current_fps: 0,
        scale: request.scale ?? 'css',
        is_streaming: false,
        updated: false,
        perf_mode: request.perfMode ?? false,
      };

      sendJson(res, 200, response);
      return;
    }

    // Check if scale change was requested (we can't change it mid-session)
    let scaleWarning: string | undefined;
    if (request.scale !== undefined && request.scale !== currentSettings.scale) {
      scaleWarning = `Scale cannot be changed mid-session. Current: ${currentSettings.scale}, Requested: ${request.scale}. Restart session to change scale.`;
      logger.info(scopedLog(LogContext.RECORDING, 'stream settings - scale change rejected'), {
        sessionId,
        currentScale: currentSettings.scale,
        requestedScale: request.scale,
      });
    }

    // Apply the settings update (quality, fps, and perfMode)
    const updated = updateFrameStreamSettings(sessionId, {
      quality: request.quality,
      fps: request.fps,
      perfMode: request.perfMode,
    });

    // Get the updated settings
    const newSettings = getFrameStreamSettings(sessionId);

    const response: StreamSettingsResponse = {
      session_id: sessionId,
      quality: newSettings?.quality ?? currentSettings.quality,
      fps: newSettings?.fps ?? currentSettings.fps,
      current_fps: newSettings?.currentFps ?? currentSettings.currentFps,
      scale: newSettings?.scale ?? currentSettings.scale,
      is_streaming: newSettings?.isStreaming ?? currentSettings.isStreaming,
      updated,
      scale_warning: scaleWarning,
      perf_mode: newSettings?.perfMode ?? currentSettings.perfMode,
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/stream-settings`);
  }
}

/**
 * Stop frame streaming for a session.
 * Handles cleanup for both screencast and polling modes.
 */
export async function stopFrameStreaming(sessionId: string): Promise<void> {
  const state = frameStreamStates.get(sessionId);
  if (!state) return;

  state.isStreaming = false;
  state.abortController.abort();

  // Stop CDP screencast if active
  if (state.screencastHandle) {
    try {
      await state.screencastHandle.stop();
    } catch (error) {
      logger.debug(scopedLog(LogContext.RECORDING, 'screencast stop error (may already be stopped)'), {
        sessionId,
        error: error instanceof Error ? error.message : String(error),
      });
    }
    state.screencastHandle = undefined;
  }

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
    totalFrames: state.frameCount,
    mode: state.useScreencast ? 'screencast' : 'polling',
  });
}

/**
 * The main frame streaming loop.
 * Captures frames and sends via WebSocket as binary data.
 *
 * Uses target-utilization based adaptive FPS via the fps controller:
 * - Measures actual screenshot capture time (the bottleneck)
 * - Adjusts FPS to keep capture at ~70% of frame budget
 * - Simple, single-mechanism feedback loop
 */
async function runFrameStreamLoop(
  sessionId: string,
  sessionManager: SessionManager,
  state: FrameStreamState
): Promise<void> {
  while (state.isStreaming && !state.abortController.signal.aborted) {
    const loopStart = performance.now();

    // Get current interval from FPS controller
    const currentIntervalMs = getIntervalMs(state.fpsState);

    try {
      // Check if session still exists
      let session;
      try {
        session = sessionManager.getSession(sessionId);
      } catch {
        // Session closed, stop streaming
        void stopFrameStreaming(sessionId);
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

      // === CAPTURE FRAME WITH TIMING ===
      const captureStart = performance.now();
      const buffer = await captureFrameBuffer(session.page, state.quality, state.scale);
      const captureTime = performance.now() - captureStart;

      if (!buffer) {
        // Timeout hit - use FPS controller's timeout handler
        const timeoutResult = handleFpsTimeout(state.fpsState, SCREENSHOT_TIMEOUT_MS, state.fpsConfig);
        state.fpsState = timeoutResult.state;

        // Record timeout in perf metrics
        metrics.frameSkipCount.inc({ session_id: sessionId, reason: 'timeout' });

        if (timeoutResult.adjusted && timeoutResult.diagnostics) {
          logger.debug(scopedLog(LogContext.RECORDING, 'FPS reduced (capture timeout)'), {
            sessionId,
            previousFps: timeoutResult.diagnostics.previousFps,
            currentFps: timeoutResult.newFps,
          });
        }

        await sleepUntilNextFrame(loopStart, currentIntervalMs, state.abortController.signal);
        continue;
      }

      // === UPDATE FPS CONTROLLER WITH CAPTURE TIME ===
      const fpsResult = processFpsFrame(state.fpsState, captureTime, state.fpsConfig);
      state.fpsState = fpsResult.state;
      state.frameCount++;

      // Log FPS changes periodically
      if (fpsResult.adjusted && fpsResult.diagnostics && state.frameCount % FPS_LOG_INTERVAL === 0) {
        const direction = fpsResult.diagnostics.reason === 'too_fast' ? 'increased' : 'reduced';
        logger.debug(scopedLog(LogContext.RECORDING, `FPS ${direction}`), {
          sessionId,
          previousFps: fpsResult.diagnostics.previousFps,
          currentFps: fpsResult.newFps,
          avgCaptureMs: fpsResult.diagnostics.avgCaptureMs,
          targetCaptureMs: fpsResult.diagnostics.targetCaptureMs,
          reason: fpsResult.diagnostics.reason,
        });
      }

      // === COMPARE WITH TIMING ===
      const compareStart = performance.now();
      const isUnchanged = isFrameUnchanged(buffer, state);
      const compareTime = performance.now() - compareStart;

      // Skip if frame content unchanged (fast buffer comparison)
      if (isUnchanged) {
        // Record skipped frame in perf collector
        state.perfCollector.recordSkipped(captureTime, compareTime);
        metrics.frameSkipCount.inc({ session_id: sessionId, reason: 'unchanged' });
        await sleepUntilNextFrame(loopStart, currentIntervalMs, state.abortController.signal);
        continue;
      }

      // Update last frame state
      state.lastFrameBuffer = buffer;

      // === SEND WITH TIMING ===
      const wsSendStart = performance.now();

      // Send frame with or without perf header
      if (state.includePerfHeaders) {
        // Build and prepend binary header with timing data
        const header = state.perfCollector.buildFrameHeader(
          captureTime,
          compareTime,
          0, // wsSendMs will be measured after send
          buffer.length
        );
        state.ws.send(Buffer.concat([header, buffer]));
      } else {
        // Send raw binary frame (no JSON, no base64!)
        state.ws.send(buffer);
      }

      const wsSendTime = performance.now() - wsSendStart;
      state.consecutiveFailures = 0;

      // === RECORD PERFORMANCE DATA ===
      state.perfCollector.recordFrame({
        captureMs: captureTime,
        compareMs: compareTime,
        wsSendMs: wsSendTime,
        frameBytes: buffer.length,
        skipped: false,
      });

      // Record to Prometheus metrics
      metrics.frameCaptureLatency.observe({ session_id: sessionId }, captureTime);
      metrics.frameE2ELatency.observe({ session_id: sessionId }, captureTime + compareTime + wsSendTime);

      // Log summary periodically
      if (state.perfCollector.shouldLogSummary()) {
        const stats = state.perfCollector.getAggregatedStats();
        logger.info(scopedLog(LogContext.RECORDING, 'frame perf summary'), {
          session_id: sessionId,
          frame_count: stats.frame_count,
          skipped_count: stats.skipped_count,
          capture_p50_ms: stats.capture_p50_ms,
          capture_p90_ms: stats.capture_p90_ms,
          e2e_p50_ms: stats.e2e_p50_ms,
          e2e_p90_ms: stats.e2e_p90_ms,
          actual_fps: getCurrentFps(state.fpsState),
          target_fps: state.targetFps,
          bottleneck: stats.primary_bottleneck,
        });
      }

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

    await sleepUntilNextFrame(loopStart, getIntervalMs(state.fpsState), state.abortController.signal);
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
 * Screenshot timeout - reduced to allow faster adaptation.
 * If a page consistently hits this timeout, adaptive FPS will reduce framerate.
 */
const SCREENSHOT_TIMEOUT_MS = 200;

/**
 * CDP session cache for frame capture.
 * Reusing the CDP session avoids the overhead of creating a new one per frame.
 * WeakMap ensures sessions are cleaned up when pages are garbage collected.
 */
const cdpSessionCache = new WeakMap<import('playwright').Page, import('playwright').CDPSession>();

/**
 * Get or create a CDP session for a page.
 * CDP sessions are cached to avoid per-frame creation overhead.
 */
async function getCDPSession(page: import('playwright').Page): Promise<import('playwright').CDPSession> {
  let session = cdpSessionCache.get(page);
  if (!session) {
    session = await page.context().newCDPSession(page);
    cdpSessionCache.set(page, session);
  }
  return session;
}

/**
 * Capture a frame as raw JPEG buffer using CDP directly.
 * Uses Page.captureScreenshot with optimizeForSpeed for faster encoding.
 *
 * Only captures the visible viewport (not full page) to reduce bandwidth and improve performance.
 * No base64 encoding overhead on our side - CDP returns base64 but we decode it once.
 *
 * Performance notes:
 * - optimizeForSpeed uses faster zlib settings (q1/RLE) for ~2x speedup
 * - CDP clip uses device pixels, so we scale viewport dimensions accordingly
 * - Using 'css' scale (1x) produces smaller images, 'device' uses devicePixelRatio
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
      // Fallback to Playwright's screenshot if viewport not available
      return await page.screenshot({
        type: 'jpeg',
        quality,
        timeout: SCREENSHOT_TIMEOUT_MS,
        scale,
      });
    }

    // Get or create CDP session (cached for performance)
    const cdp = await getCDPSession(page);

    // Use CDP Page.captureScreenshot directly with optimizeForSpeed
    // This provides ~2x speedup on encoding by using faster compression settings
    //
    // Key settings:
    // - captureBeyondViewport: false - only capture visible viewport (not full page)
    // - fromSurface: true - capture from compositor surface (default, faster)
    // - optimizeForSpeed: true - use faster encoding (zlib q1/RLE)
    //
    // Note: We don't use 'clip' because it captures a fixed page region, not the
    // current viewport. With captureBeyondViewport: false, CDP captures exactly
    // what's visible in the viewport, which is what we want for live streaming.
    const result = await Promise.race([
      cdp.send('Page.captureScreenshot', {
        format: 'jpeg',
        quality,
        optimizeForSpeed: true, // Key optimization: faster encoding
        captureBeyondViewport: false, // Only capture visible viewport
        fromSurface: true, // Capture from compositor (faster)
      }),
      // Timeout handling
      new Promise<null>((resolve) => setTimeout(() => resolve(null), SCREENSHOT_TIMEOUT_MS)),
    ]);

    if (!result) {
      return null; // Timeout
    }

    // CDP returns base64, decode to Buffer
    return Buffer.from(result.data, 'base64');
  } catch {
    // CDP error or other failure - skip this frame
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
