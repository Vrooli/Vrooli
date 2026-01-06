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
import {
  initRecordingBuffer,
  bufferTimelineEntry,
  getTimelineEntries,
  getTimelineEntryCount,
  clearTimelineEntries,
} from '../../recording/buffer';
import { timelineEntryToJson, type TimelineEntry, ActionType } from '../../proto/recording';
import { runRecordingPipelineTest, runExternalUrlInjectionTest, TEST_PAGE_URL } from '../../recording/self-test';
import type { Page } from 'rebrowser-playwright';
import type {
  StartRecordingRequest,
  StartRecordingResponse,
  StopRecordingResponse,
  RecordingStatusResponse,
  StreamSettingsRequest,
  StreamSettingsResponse,
  DriverPageEvent,
} from './types';
import { createCircuitBreaker, type CircuitBreaker } from '../../infra';
import {
  startFrameStreaming,
  stopFrameStreaming,
  updateFrameStreamSettings,
  getFrameStreamSettings,
} from '../../frame-streaming';
import { emitHistoryCallback, captureThumbnail } from './recording-interaction';

// =============================================================================
// Circuit Breaker for Callback Streaming
// =============================================================================

/**
 * Circuit breaker for callback streaming.
 * Prevents cascading failures when callback endpoint is unavailable.
 */
const callbackCircuitBreaker: CircuitBreaker<string> = createCircuitBreaker({
  maxFailures: 5,
  resetTimeoutMs: 30_000, // 30 seconds
  name: 'callback',
});

/**
 * Circuit breaker for page event streaming.
 * Separate from action callback to allow independent failure handling.
 */
const pageEventCircuitBreaker: CircuitBreaker<string> = createCircuitBreaker({
  maxFailures: 5,
  resetTimeoutMs: 30_000, // 30 seconds
  name: 'pageEvent',
});

/** Timeout for page event callbacks (5 seconds) */
const PAGE_EVENT_TIMEOUT_MS = 5_000;

/**
 * Send a page event to the callback URL with circuit breaker protection.
 */
async function sendPageEvent(
  sessionId: string,
  callbackUrl: string,
  event: DriverPageEvent
): Promise<void> {
  // Check if we should attempt half-open (atomically claims the attempt)
  const attemptHalfOpen = pageEventCircuitBreaker.tryEnterHalfOpen(sessionId);

  // Skip if circuit is open and we're not the half-open attempt
  if (pageEventCircuitBreaker.isOpen(sessionId) && !attemptHalfOpen) {
    logger.debug(scopedLog(LogContext.RECORDING, 'skipping page event (circuit open)'), {
      sessionId,
      eventType: event.eventType,
      driverPageId: event.driverPageId,
    });
    return;
  }

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), PAGE_EVENT_TIMEOUT_MS);

  try {
    const response = await fetch(callbackUrl, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(event),
      signal: controller.signal,
    });

    if (!response.ok) {
      throw new Error(`Page event callback returned ${response.status}: ${response.statusText}`);
    }

    pageEventCircuitBreaker.recordSuccess(sessionId);

    logger.debug(scopedLog(LogContext.RECORDING, 'page event sent'), {
      sessionId,
      eventType: event.eventType,
      driverPageId: event.driverPageId,
      url: event.url,
    });
  } catch (err) {
    const circuitOpened = pageEventCircuitBreaker.recordFailure(sessionId);
    const errorMessage = err instanceof Error ? err.message : String(err);

    logger.warn(scopedLog(LogContext.RECORDING, 'page event send failed'), {
      sessionId,
      callbackUrl,
      eventType: event.eventType,
      error: errorMessage,
      circuitOpen: circuitOpened,
    });
  } finally {
    clearTimeout(timeoutId);
  }
}

/**
 * Set up page lifecycle listeners for a recording session.
 * Tracks new tabs, navigation events, and page closes.
 */
function setupPageLifecycleListeners(
  sessionId: string,
  session: ReturnType<SessionManager['getSession']>,
  pageCallbackUrl: string,
  config: Config
): () => void {
  const context = session.context;

  // Handler for new pages (new tabs/popups)
  const onNewPage = async (newPage: Page): Promise<void> => {
    // Generate a unique ID for this page
    const pageId = crypto.randomUUID();

    // Add to session tracking
    session.pages.push(newPage);
    session.pageIdMap.set(pageId, newPage);
    session.pageToIdMap.set(newPage, pageId);

    // Find the opener page ID if any
    const openerPage = await newPage.opener();
    const openerPageId = openerPage ? session.pageToIdMap.get(openerPage) : undefined;

    // Wait for the page to be ready enough to get URL/title
    try {
      await newPage.waitForLoadState('domcontentloaded', { timeout: 5000 }).catch(() => {});
    } catch {
      // Ignore timeout - page might be blank
    }

    const url = newPage.url();
    const title = await newPage.title().catch(() => '');

    logger.info(scopedLog(LogContext.RECORDING, 'new page detected'), {
      sessionId,
      pageId,
      openerPageId,
      url,
      title,
      totalPages: session.pages.length,
    });

    // Send page_created event
    const event: DriverPageEvent = {
      sessionId,
      driverPageId: pageId,
      vrooliPageId: '',
      eventType: 'created',
      url,
      title,
      openerDriverPageId: openerPageId,
      timestamp: new Date().toISOString(),
    };

    await sendPageEvent(sessionId, pageCallbackUrl, event);

    // Set up navigation listener for this page
    newPage.on('framenavigated', async (frame) => {
      // Only track main frame navigations
      if (frame !== newPage.mainFrame()) return;

      const navUrl = newPage.url();
      const navTitle = await newPage.title().catch(() => '');

      logger.debug(scopedLog(LogContext.RECORDING, 'page navigated'), {
        sessionId,
        pageId,
        url: navUrl,
      });

      const navEvent: DriverPageEvent = {
        sessionId,
        driverPageId: pageId,
        vrooliPageId: '',
        eventType: 'navigated',
        url: navUrl,
        title: navTitle,
        timestamp: new Date().toISOString(),
      };

      await sendPageEvent(sessionId, pageCallbackUrl, navEvent);

      // Emit history callback for session profile history tracking
      const thumbnail = config.history.thumbnailEnabled
        ? await captureThumbnail(newPage, config.history.thumbnailQuality)
        : undefined;
      emitHistoryCallback(config, sessionId, navUrl, navTitle, 'navigate', thumbnail).catch(() => {
        // Error already logged in emitHistoryCallback
      });
    });

    // Set up close listener for this page
    newPage.on('close', async () => {
      logger.info(scopedLog(LogContext.RECORDING, 'page closed'), {
        sessionId,
        pageId,
      });

      const closeEvent: DriverPageEvent = {
        sessionId,
        driverPageId: pageId,
        vrooliPageId: '',
        eventType: 'closed',
        url: '',
        title: '',
        timestamp: new Date().toISOString(),
      };

      await sendPageEvent(sessionId, pageCallbackUrl, closeEvent);

      // Remove from tracking
      session.pageIdMap.delete(pageId);
      const pageIndex = session.pages.indexOf(newPage);
      if (pageIndex !== -1) {
        session.pages.splice(pageIndex, 1);
      }
    });
  };

  // Listen for new pages
  context.on('page', onNewPage);

  // Return cleanup function
  return () => {
    context.off('page', onNewPage);
    pageEventCircuitBreaker.cleanup(sessionId);
  };
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

    // Get pipeline manager (single source of truth for recording state)
    const pipelineManager = session.pipelineManager;
    if (!pipelineManager) {
      throw new Error('Pipeline manager not set on session - context may not have been initialized properly');
    }

    // Idempotency: Check if already recording
    if (pipelineManager.isRecording()) {
      const currentRecordingId = pipelineManager.getRecordingId();
      // If the same recording_id is provided, this is an idempotent retry - return success
      if (request.recording_id && currentRecordingId === request.recording_id) {
        const recordingData = pipelineManager.getRecordingData();
        logger.info(scopedLog(LogContext.RECORDING, 'idempotent start - already recording'), {
          sessionId,
          recordingId: currentRecordingId,
          hint: 'Treating as successful retry of previous request',
        });

        const response: StartRecordingResponse = {
          recording_id: currentRecordingId || request.recording_id,
          session_id: sessionId,
          started_at: recordingData?.startedAt || new Date().toISOString(),
        };

        sendJson(res, 200, response);
        return;
      }

      // Different recording_id or no recording_id provided - this is a conflict
      logger.warn(scopedLog(LogContext.RECORDING, 'already in progress'), {
        sessionId,
        recordingId: currentRecordingId,
        requestedRecordingId: request.recording_id,
        hint: 'Call /record/stop to end the current recording before starting a new one',
      });
      sendJson(res, 409, {
        error: 'RECORDING_IN_PROGRESS',
        message: 'Recording is already in progress for this session',
        recording_id: currentRecordingId,
        hint: 'Call /record/stop to end the current recording before starting a new one',
      });
      return;
    }

    // Initialize action buffer for this session
    initRecordingBuffer(sessionId);

    logger.info(scopedLog(LogContext.RECORDING, 'starting'), {
      sessionId,
      hasCallback: !!request.callback_url,
      currentUrl: session.page?.url?.() || '(unknown)',
    });

    // Start recording with callback to buffer entries (using pipelineManager directly)
    const recordingId = await pipelineManager.startRecording({
      sessionId,
      recordingId: request.recording_id,
      onEntry: async (entry: TimelineEntry) => {
        // Buffer the entry (proto TimelineEntry)
        bufferTimelineEntry(sessionId, entry);

        // Track recording activity in metrics
        metrics.recordingActionsTotal.inc();

        // Extract action type name for logging
        const actionTypeName = ActionType[entry.action?.type ?? ActionType.UNSPECIFIED] ?? 'UNKNOWN';

        logger.debug(scopedLog(LogContext.RECORDING, 'entry captured'), {
          sessionId,
          recordingId: pipelineManager.getRecordingId(),
          actionType: actionTypeName,
          sequenceNum: entry.sequenceNum,
          confidence: entry.action?.metadata?.confidence,
        });

        // If callback URL provided, stream to it (with circuit breaker protection)
        if (request.callback_url) {
          await streamEntryWithCircuitBreaker(sessionId, request.callback_url, entry);
        }
      },
      onError: (error: Error) => {
        logger.error(scopedLog(LogContext.RECORDING, 'capture error'), {
          sessionId,
          recordingId: pipelineManager.getRecordingId(),
          error: error.message,
          hint: 'Recording may have encountered a page issue or event listener failure',
        });
      },
    });

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

    // Set up page lifecycle listeners if page callback URL provided
    if (request.page_callback_url) {
      // Set up listeners for new pages
      session.pageLifecycleCleanup = setupPageLifecycleListeners(
        sessionId,
        session,
        request.page_callback_url,
        config
      );

      // Send initial page's page_created event
      const initialPageId = session.pageToIdMap.get(session.page);
      if (initialPageId) {
        const initialUrl = session.page.url();
        const initialTitle = await session.page.title().catch(() => '');

        const initialPageEvent: DriverPageEvent = {
          sessionId,
          driverPageId: initialPageId,
          vrooliPageId: '',
          eventType: 'initial',
          url: initialUrl,
          title: initialTitle,
          timestamp: new Date().toISOString(),
        };

        await sendPageEvent(sessionId, request.page_callback_url, initialPageEvent);

        logger.info(scopedLog(LogContext.RECORDING, 'initial page event sent'), {
          sessionId,
          pageId: initialPageId,
          url: initialUrl,
        });

        // Set up navigation listener for the initial page
        // (new pages get this in setupPageLifecycleListeners, but initial page needs it here)
        const pageCallbackUrl = request.page_callback_url;
        session.page.on('framenavigated', async (frame) => {
          // Only track main frame navigations
          if (frame !== session.page.mainFrame()) return;

          const navUrl = session.page.url();
          const navTitle = await session.page.title().catch(() => '');

          logger.debug(scopedLog(LogContext.RECORDING, 'initial page navigated'), {
            sessionId,
            pageId: initialPageId,
            url: navUrl,
          });

          const navEvent: DriverPageEvent = {
            sessionId,
            driverPageId: initialPageId,
            vrooliPageId: '',
            eventType: 'navigated',
            url: navUrl,
            title: navTitle,
            timestamp: new Date().toISOString(),
          };

          await sendPageEvent(sessionId, pageCallbackUrl, navEvent);

          // Emit history callback for session profile history tracking
          const thumbnail = config.history.thumbnailEnabled
            ? await captureThumbnail(session.page, config.history.thumbnailQuality)
            : undefined;
          emitHistoryCallback(config, sessionId, navUrl, navTitle, 'navigate', thumbnail).catch(() => {
            // Error already logged in emitHistoryCallback
          });
        });
      }
    }

    // Get verification info for response (helps diagnose "no events" issues)
    const verificationData = pipelineManager.getVerification();
    const warnings: string[] = [];

    // Generate warnings based on verification data
    if (verificationData) {
      if (!verificationData.inMainContext) {
        warnings.push('Script not in MAIN context - History API events will NOT be captured');
      }
      if (verificationData.handlersCount < 7) {
        warnings.push(`Low handler count (${verificationData.handlersCount}) - some events may not be captured`);
      }
    }

    logger.info(scopedLog(LogContext.RECORDING, 'started'), {
      sessionId,
      recordingId,
      phase: 'recording',
      url: session.page?.url?.() || '(unknown)',
      frameStreaming: !!request.frame_callback_url,
      pageTracking: !!request.page_callback_url,
      verification: verificationData,
      warnings: warnings.length > 0 ? warnings : undefined,
    });

    const response: StartRecordingResponse = {
      recording_id: recordingId,
      session_id: sessionId,
      started_at: new Date().toISOString(),
      verification: verificationData
        ? {
            script_loaded: verificationData.scriptLoaded,
            script_ready: verificationData.scriptReady,
            in_main_context: verificationData.inMainContext,
            handlers_count: verificationData.handlersCount,
            version: verificationData.version,
            warnings: warnings.length > 0 ? warnings : undefined,
          }
        : undefined,
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

    // Get pipeline manager (single source of truth for recording state)
    const pipelineManager = session.pipelineManager;
    if (!pipelineManager) {
      throw new Error('Pipeline manager not set on session');
    }

    // Idempotency: If not recording, treat as successful no-op
    // This handles retries where the first request succeeded but response was lost
    if (!pipelineManager.isRecording()) {
      // Get last known recording ID from pipeline state if available
      const lastRecordingId = pipelineManager.getRecordingId();
      logger.info(scopedLog(LogContext.RECORDING, 'idempotent stop - no active recording'), {
        sessionId,
        phase: session.phase,
        hint: 'No recording active, treating as successful stop (idempotent)',
      });

      // Return success with zero action count
      const response: StopRecordingResponse = {
        recording_id: lastRecordingId || 'unknown',
        session_id: sessionId,
        action_count: 0,
        stopped_at: new Date().toISOString(),
      };

      sendJson(res, 200, response);
      return;
    }

    const recordingId = pipelineManager.getRecordingId();
    const result = await pipelineManager.stopRecording();

    // Stop frame streaming if active
    await stopFrameStreaming(sessionId);

    // Clean up page lifecycle listeners if set up
    if (session.pageLifecycleCleanup) {
      session.pageLifecycleCleanup();
      session.pageLifecycleCleanup = undefined;
    }

    // Clean up circuit breaker state
    callbackCircuitBreaker.cleanup(sessionId);

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

    // Use pipelineManager as single source of truth
    const pipelineManager = session.pipelineManager;
    const pipelineState = pipelineManager?.getState();
    const bufferedCount = getTimelineEntryCount(sessionId);

    const response: RecordingStatusResponse = {
      session_id: sessionId,
      is_recording: pipelineState?.phase === 'recording',
      recording_id: pipelineState?.recording?.recordingId,
      action_count: bufferedCount || pipelineState?.recording?.actionCount || 0,
      started_at: pipelineState?.recording?.startedAt,
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
 * Stream a TimelineEntry to callback URL with circuit breaker protection.
 *
 * This handles the full lifecycle of streaming an entry:
 * 1. Check if circuit is open (skip if so, unless entering half-open)
 * 2. Stream the entry to the callback URL
 * 3. Record success/failure with the circuit breaker
 * 4. Log appropriate messages and metrics
 *
 * @param sessionId - Session ID for circuit breaker and logging
 * @param callbackUrl - URL to stream the entry to
 * @param entry - The TimelineEntry to stream
 */
async function streamEntryWithCircuitBreaker(
  sessionId: string,
  callbackUrl: string,
  entry: TimelineEntry
): Promise<void> {
  const actionTypeName = ActionType[entry.action?.type ?? ActionType.UNSPECIFIED] ?? 'UNKNOWN';

  // Check if we should attempt half-open (atomically claims the attempt)
  const attemptHalfOpen = callbackCircuitBreaker.tryEnterHalfOpen(sessionId);

  // Skip if circuit is open and we're not the half-open attempt
  if (callbackCircuitBreaker.isOpen(sessionId) && !attemptHalfOpen) {
    const stats = callbackCircuitBreaker.getStats(sessionId);
    logger.debug(scopedLog(LogContext.RECORDING, 'skipping callback (circuit open)'), {
      sessionId,
      actionType: actionTypeName,
      sequenceNum: entry.sequenceNum,
      halfOpenInProgress: stats?.halfOpenInProgress,
    });
    return;
  }

  try {
    await streamEntryToCallback(callbackUrl, entry);
    callbackCircuitBreaker.recordSuccess(sessionId);
  } catch (err) {
    const circuitOpened = callbackCircuitBreaker.recordFailure(sessionId);

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

    const stats = callbackCircuitBreaker.getStats(sessionId);
    logger.warn(scopedLog(LogContext.RECORDING, 'callback streaming failed'), {
      sessionId,
      callbackUrl,
      error: errorMessage,
      failureReason: reason,
      consecutiveFailures: stats?.consecutiveFailures,
      circuitOpen: circuitOpened,
      hint: circuitOpened
        ? 'Circuit breaker opened - entries still buffered locally'
        : 'Callback endpoint may be unreachable or returning errors',
    });
  }
}

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
      // Note: Defaults should match API config (BAS_RECORDING_DEFAULT_STREAM_*)
      const response: StreamSettingsResponse = {
        session_id: sessionId,
        quality: request.quality ?? 55,
        fps: request.fps ?? 30, // Match API config default (BAS_RECORDING_DEFAULT_STREAM_FPS)
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
 * Live recording debug endpoint
 *
 * GET /session/:id/record/debug
 *
 * Queries the live state of the recording session including:
 * - Server-side recording state
 * - Browser script state (injected, handlers registered, isActive)
 * - Route handler stats (events received, processed, dropped)
 * - Event handler status
 *
 * This is used for real-time debugging during active recording sessions.
 */
export async function handleRecordDebug(
  _req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);

    // Server-side state (using pipelineManager as single source of truth)
    const pipelineManager = session.pipelineManager;
    const pipelineState = pipelineManager?.getState();
    const isRecording = pipelineState?.phase === 'recording';
    const recordingId = pipelineState?.recording?.recordingId;
    const pipelinePhase = pipelineState?.phase;
    const pipelineVerification = pipelineState?.verification;
    const hasEventHandler = session.recordingInitializer?.hasEventHandler() ?? false;
    const routeHandlerStats = session.recordingInitializer?.getRouteHandlerStats();
    const injectionStats = session.recordingInitializer?.getInjectionStats();

    // Query browser script state via CDP
    let browserScriptState: {
      loaded: boolean;
      ready: boolean;
      isActive: boolean;
      inMainContext: boolean;
      handlersCount: number;
      version: string | null;
      eventsDetected: number;
      eventsCaptured: number;
      eventsSent: number;
      eventsSendFailed: number;
      lastError: string | null;
      serviceWorkerActive: boolean;
      serviceWorkerUrl: string | null;
    } | null = null;

    try {
      const client = await session.page.context().newCDPSession(session.page);
      try {
        const { result } = await client.send('Runtime.evaluate', {
          expression: `(function() {
            if (!window.__vrooli_recording_script_loaded) {
              return JSON.stringify({ loaded: false });
            }
            const telemetry = window.__vrooli_recording_telemetry || {};
            // Check for service worker
            let serviceWorkerActive = false;
            let serviceWorkerUrl = null;
            if (navigator.serviceWorker && navigator.serviceWorker.controller) {
              serviceWorkerActive = true;
              serviceWorkerUrl = navigator.serviceWorker.controller.scriptURL || null;
            }
            return JSON.stringify({
              loaded: true,
              ready: window.__vrooli_recording_ready === true,
              isActive: typeof window.__isRecordingActive === 'function' ? window.__isRecordingActive() : null,
              inMainContext: window.__vrooli_recording_script_context === 'MAIN',
              handlersCount: window.__vrooli_recording_handlers_count || 0,
              version: window.__vrooli_recording_script_version || null,
              eventsDetected: telemetry.eventsDetected || 0,
              eventsCaptured: telemetry.eventsCaptured || 0,
              eventsSent: telemetry.eventsSent || 0,
              eventsSendFailed: telemetry.eventsSendFailed || 0,
              lastError: telemetry.lastError || null,
              serviceWorkerActive: serviceWorkerActive,
              serviceWorkerUrl: serviceWorkerUrl
            });
          })()`,
          returnByValue: true,
        });

        if (result.type === 'string' && result.value) {
          browserScriptState = JSON.parse(result.value);
        }
      } finally {
        await client.detach().catch(() => {});
      }
    } catch (error) {
      logger.debug(scopedLog(LogContext.RECORDING, 'debug: failed to query browser script state'), {
        sessionId,
        error: error instanceof Error ? error.message : String(error),
      });
    }

    // Build response
    const response = {
      session_id: sessionId,
      timestamp: new Date().toISOString(),
      server: {
        is_recording: isRecording,
        recording_id: recordingId,
        has_event_handler: hasEventHandler,
        phase: session.phase,
        current_url: session.page?.url?.() || null,
      },
      pipeline: {
        phase: pipelinePhase,
        action_count: pipelineState?.recording?.actionCount ?? 0,
        generation: pipelineState?.recording?.generation ?? 0,
        verification: pipelineVerification ? {
          script_loaded: pipelineVerification.scriptLoaded,
          script_ready: pipelineVerification.scriptReady,
          in_main_context: pipelineVerification.inMainContext,
          handlers_count: pipelineVerification.handlersCount,
          event_route_active: pipelineVerification.eventRouteActive,
          verified_at: pipelineVerification.verifiedAt,
        } : null,
        error: pipelineState?.error ? {
          code: pipelineState.error.code,
          message: pipelineState.error.message,
          recoverable: pipelineState.error.recoverable,
        } : null,
      },
      route_handler: routeHandlerStats ? {
        events_received: routeHandlerStats.eventsReceived,
        events_processed: routeHandlerStats.eventsProcessed,
        events_dropped_no_handler: routeHandlerStats.eventsDroppedNoHandler,
        events_with_errors: routeHandlerStats.eventsWithErrors,
        last_event_at: routeHandlerStats.lastEventAt,
        last_event_type: routeHandlerStats.lastEventType,
      } : null,
      injection: injectionStats ? {
        attempted: injectionStats.attempted,
        successful: injectionStats.successful,
        failed: injectionStats.failed,
        skipped: injectionStats.skipped,
      } : null,
      browser_script: browserScriptState,
      diagnostics: {
        // Check for common issues
        script_not_loaded: browserScriptState && !browserScriptState.loaded,
        script_not_ready: browserScriptState?.loaded && !browserScriptState.ready,
        script_not_in_main: browserScriptState?.loaded && !browserScriptState.inMainContext,
        script_inactive: browserScriptState?.loaded && browserScriptState.isActive === false,
        no_handlers: browserScriptState?.loaded && browserScriptState.handlersCount === 0,
        no_event_handler: isRecording && !hasEventHandler,
        events_being_dropped: (routeHandlerStats?.eventsDroppedNoHandler ?? 0) > 0,
        // Service worker interception detection
        service_worker_blocking: browserScriptState?.serviceWorkerActive &&
          (browserScriptState.eventsSent > 0) &&
          (routeHandlerStats?.eventsReceived ?? 0) === 0,
      },
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/debug`);
  }
}

/**
 * Recording pipeline self-test endpoint
 *
 * POST /session/:id/record/pipeline-test
 *
 * Runs an automated end-to-end test of the recording pipeline.
 * This test:
 * 1. Navigates to a special test page served by the driver
 * 2. Simulates real user interactions using CDP (not page.click)
 * 3. Verifies events flow through the entire pipeline
 * 4. Reports detailed diagnostics at each step
 *
 * This eliminates the need for human-in-the-loop debugging - the system
 * tests itself and reports exactly where events are getting lost.
 *
 * Request body (optional):
 * {
 *   "timeout_ms": 30000,          // Test timeout (default: 30000)
 *   "return_to_original": true,   // Navigate back to original URL after test (default: true)
 * }
 *
 * Response:
 * {
 *   "success": boolean,           // Whether all tests passed
 *   "failure_point": string,      // Where it failed (if failed)
 *   "failure_message": string,    // Human-readable failure description
 *   "suggestions": string[],      // Suggestions for fixing the issue
 *   "steps": [...],               // Detailed results for each step
 *   "diagnostics": {...},         // Raw diagnostic data
 * }
 */
export async function handleRecordPipelineTest(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  config: Config
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);
    const body = await parseJsonBody(req, config).catch(() => ({}));
    const request = body as { test_url?: string; timeout_ms?: number; return_to_original?: boolean };

    const testUrl = request.test_url; // undefined = use default (example.com)
    const timeoutMs = request.timeout_ms ?? 30000;
    const returnToOriginal = request.return_to_original ?? true;

    // Store original URL to navigate back after test
    const originalUrl = session.page.url();

    logger.info(scopedLog(LogContext.RECORDING, 'starting pipeline self-test'), {
      sessionId,
      originalUrl: originalUrl.slice(0, 80),
      timeoutMs,
      testUrl: testUrl || 'default (example.com)',
    });

    // Ensure we have a recording initializer and pipeline manager
    if (!session.recordingInitializer) {
      sendJson(res, 500, {
        error: 'MISSING_INITIALIZER',
        message: 'Recording initializer not set on session',
        hint: 'The session may not have been properly initialized for recording',
      });
      return;
    }

    if (!session.pipelineManager) {
      sendJson(res, 500, {
        error: 'MISSING_PIPELINE_MANAGER',
        message: 'Pipeline manager not set on session',
        hint: 'The session may not have been properly initialized for recording',
      });
      return;
    }

    // Run the pipeline test using the REAL recording path
    // This tests pipelineManager.startRecording() → onEntry callback flow
    const result = await runRecordingPipelineTest(
      session.page,
      session.context,
      session.pipelineManager,
      session.recordingInitializer,
      {
        testUrl,
        timeoutMs,
        captureConsole: true,
      }
    );

    // Navigate back to original URL if requested and we were on a real page
    if (returnToOriginal && originalUrl && !originalUrl.startsWith('about:')) {
      try {
        await session.page.goto(originalUrl, { timeout: 10000, waitUntil: 'domcontentloaded' });
        logger.debug(scopedLog(LogContext.RECORDING, 'returned to original URL after test'), {
          sessionId,
          originalUrl: originalUrl.slice(0, 80),
        });
      } catch (navError) {
        logger.warn(scopedLog(LogContext.RECORDING, 'failed to return to original URL'), {
          sessionId,
          originalUrl: originalUrl.slice(0, 80),
          error: navError instanceof Error ? navError.message : String(navError),
        });
      }
    }

    // Log result summary
    if (result.success) {
      logger.info(scopedLog(LogContext.RECORDING, 'pipeline self-test PASSED'), {
        sessionId,
        durationMs: result.durationMs,
        stepsCompleted: result.steps.filter(s => s.passed).length,
        totalSteps: result.steps.length,
      });
    } else {
      logger.warn(scopedLog(LogContext.RECORDING, 'pipeline self-test FAILED'), {
        sessionId,
        durationMs: result.durationMs,
        failurePoint: result.failurePoint,
        failureMessage: result.failureMessage,
        suggestions: result.suggestions,
      });
    }

    // Build response
    const response = {
      success: result.success,
      timestamp: result.timestamp,
      duration_ms: result.durationMs,
      failure_point: result.failurePoint,
      failure_message: result.failureMessage,
      suggestions: result.suggestions,
      steps: result.steps.map(step => ({
        name: step.name,
        passed: step.passed,
        duration_ms: step.durationMs,
        error: step.error,
        details: step.details,
      })),
      diagnostics: {
        test_page_url: result.diagnostics.testPageUrl,
        test_page_injected: result.diagnostics.testPageInjected,
        script_status_before: result.diagnostics.scriptStatusBefore,
        script_status_after: result.diagnostics.scriptStatusAfter,
        telemetry_before: result.diagnostics.telemetryBefore,
        telemetry_after: result.diagnostics.telemetryAfter,
        route_stats_before: result.diagnostics.routeStatsBefore,
        route_stats_after: result.diagnostics.routeStatsAfter,
        events_captured: result.diagnostics.eventsCaptured,
        console_messages: result.diagnostics.consoleMessages.slice(0, 50), // Limit console messages
      },
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/pipeline-test`);
  }
}

/**
 * External URL injection test endpoint
 *
 * POST /session/:id/record/external-url-test
 *
 * Tests the ACTUAL injection path that real users experience when navigating
 * to external URLs. This is critical because the standard pipeline test uses
 * an internal test page that bypasses the real injection mechanism.
 *
 * This test:
 * 1. Navigates to a real external URL (default: https://example.com)
 * 2. Tests the full route.fetch() → modify HTML → route.fulfill() path
 * 3. Verifies script injection worked in the browser
 * 4. Reports specific failure points if injection fails
 *
 * Request body (optional):
 * {
 *   "test_url": "https://example.com",  // URL to test (default: example.com)
 *   "timeout_ms": 30000,                // Test timeout (default: 30000)
 *   "return_to_original": true,         // Navigate back after test (default: true)
 * }
 *
 * Response:
 * {
 *   "success": boolean,
 *   "tested_url": string,
 *   "failure_point": string,           // fetch|modify|fulfill|script_load|script_ready|context_wrong|network|timeout
 *   "failure_message": string,
 *   "suggestions": string[],
 *   "verification": {...},             // Script verification results
 *   "injection_stats": {...},          // Injection statistics
 * }
 */
export async function handleRecordExternalUrlTest(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  config: Config
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);
    const body = await parseJsonBody(req, config).catch(() => ({}));
    const request = body as { test_url?: string; timeout_ms?: number; return_to_original?: boolean };

    const testUrl = request.test_url ?? 'https://example.com';
    const timeoutMs = request.timeout_ms ?? 30000;
    const returnToOriginal = request.return_to_original ?? true;

    // Store original URL to navigate back after test
    const originalUrl = session.page.url();

    logger.info(scopedLog(LogContext.RECORDING, 'starting external URL injection test'), {
      sessionId,
      testUrl,
      originalUrl: originalUrl.slice(0, 80),
      timeoutMs,
    });

    // Ensure we have a recording initializer
    if (!session.recordingInitializer) {
      sendJson(res, 500, {
        error: 'MISSING_INITIALIZER',
        message: 'Recording initializer not set on session',
        hint: 'The session may not have been properly initialized for recording',
      });
      return;
    }

    // Run the external URL injection test
    const result = await runExternalUrlInjectionTest(
      session.page,
      session.recordingInitializer,
      {
        testUrl,
        timeoutMs,
      }
    );

    // Navigate back to original URL if requested and we were on a real page
    if (returnToOriginal && originalUrl && !originalUrl.startsWith('about:')) {
      try {
        await session.page.goto(originalUrl, { timeout: 10000, waitUntil: 'domcontentloaded' });
        logger.debug(scopedLog(LogContext.RECORDING, 'returned to original URL after external test'), {
          sessionId,
          originalUrl: originalUrl.slice(0, 80),
        });
      } catch (navError) {
        logger.warn(scopedLog(LogContext.RECORDING, 'failed to return to original URL'), {
          sessionId,
          originalUrl: originalUrl.slice(0, 80),
          error: navError instanceof Error ? navError.message : String(navError),
        });
      }
    }

    // Log result summary
    if (result.success) {
      logger.info(scopedLog(LogContext.RECORDING, 'external URL injection test PASSED'), {
        sessionId,
        testUrl,
        durationMs: result.durationMs,
        handlersCount: result.verification?.handlersCount,
      });
    } else {
      logger.warn(scopedLog(LogContext.RECORDING, 'external URL injection test FAILED'), {
        sessionId,
        testUrl,
        durationMs: result.durationMs,
        failurePoint: result.failurePoint,
        failureMessage: result.failureMessage,
        suggestions: result.suggestions,
      });
    }

    // Build response
    const response = {
      success: result.success,
      timestamp: result.timestamp,
      duration_ms: result.durationMs,
      tested_url: result.testedUrl,
      failure_point: result.failurePoint,
      failure_message: result.failureMessage,
      suggestions: result.suggestions,
      verification: result.verification,
      injection_stats: result.injectionStats,
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/external-url-test`);
  }
}
