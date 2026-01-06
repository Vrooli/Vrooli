/**
 * Recording Lifecycle Handlers
 *
 * Handles core start/stop/status/actions operations for record mode.
 * This is the core recording lifecycle management.
 *
 * Other recording functionality has been split into:
 * - callback-streaming.ts: Callback streaming with circuit breaker
 * - page-events.ts: Multi-tab page event handling
 * - recording-diagnostics-routes.ts: Debug and test endpoints
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
import type {
  StartRecordingRequest,
  StartRecordingResponse,
  StopRecordingResponse,
  RecordingStatusResponse,
  DriverPageEvent,
} from './types';
import {
  startFrameStreaming,
  stopFrameStreaming,
} from '../../frame-streaming';
import { streamEntryWithCircuitBreaker, callbackCircuitBreaker } from './callback-streaming';
import { setupPageLifecycleListeners, sendPageEvent, pageEventCircuitBreaker } from './page-events';
import { captureThumbnail, emitHistoryCallback } from './recording-pages';

// =============================================================================
// Recording Lifecycle Handlers
// =============================================================================

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
    pageEventCircuitBreaker.cleanup(sessionId);

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
