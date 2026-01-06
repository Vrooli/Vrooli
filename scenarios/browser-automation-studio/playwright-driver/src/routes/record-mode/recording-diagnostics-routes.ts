/**
 * Recording Diagnostics Routes
 *
 * Endpoints for debugging and testing the recording pipeline:
 * - Stream settings management
 * - Live debug endpoint for querying recording state
 * - Pipeline self-test for end-to-end validation
 * - External URL injection test
 */

import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../../session';
import type { Config } from '../../config';
import { parseJsonBody, sendJson, sendError } from '../../middleware';
import { logger, scopedLog, LogContext } from '../../utils';
import {
  updateFrameStreamSettings,
  getFrameStreamSettings,
} from '../../frame-streaming';
import { runRecordingPipelineTest, runExternalUrlInjectionTest } from '../../recording/self-test';
import type { StreamSettingsRequest, StreamSettingsResponse } from './types';

// =============================================================================
// Stream Settings Handler
// =============================================================================

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

// =============================================================================
// Debug Endpoint
// =============================================================================

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
    const isRecording = pipelineState?.phase === 'capturing';
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

// =============================================================================
// Pipeline Test Endpoints
// =============================================================================

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
