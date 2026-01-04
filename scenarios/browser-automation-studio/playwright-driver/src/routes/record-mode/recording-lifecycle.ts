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
  bufferTimelineEntry,
  getTimelineEntries,
  getTimelineEntryCount,
  clearTimelineEntries,
} from '../../recording/buffer';
import { timelineEntryToJson, type TimelineEntry, ActionType } from '../../proto/recording';
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
  pageCallbackUrl: string
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
      if (!session.recordingInitializer) {
        throw new Error('Recording initializer not set on session - context may not have been initialized properly');
      }
      session.recordingController = createRecordModeController(
        session.page,
        session.context,
        session.recordingInitializer,
        sessionId
      );
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
          await streamEntryWithCircuitBreaker(sessionId, request.callback_url, entry);
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

    // Set up page lifecycle listeners if page callback URL provided
    if (request.page_callback_url) {
      // Set up listeners for new pages
      session.pageLifecycleCleanup = setupPageLifecycleListeners(
        sessionId,
        session,
        request.page_callback_url
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
        });
      }
    }

    logger.info(scopedLog(LogContext.RECORDING, 'started'), {
      sessionId,
      recordingId,
      phase: 'recording',
      url: session.page?.url?.() || '(unknown)',
      frameStreaming: !!request.frame_callback_url,
      pageTracking: !!request.page_callback_url,
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
