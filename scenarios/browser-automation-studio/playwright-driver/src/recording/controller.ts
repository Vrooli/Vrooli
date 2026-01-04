/**
 * Recording Controller - Orchestrates Record Mode
 *
 * PROTO-FIRST ARCHITECTURE:
 * This controller converts RawBrowserEvent directly to TimelineEntry (proto),
 * eliminating intermediate types. The conversion happens at the earliest
 * possible point after receiving events from the browser.
 *
 * ┌────────────────────────────────────────────────────────────────────────┐
 * │ WHAT THIS CLASS DOES:                                                  │
 * │                                                                        │
 * │ 1. RECORDING LIFECYCLE (startRecording, stopRecording)                 │
 * │    - Activates/deactivates the pre-injected recording script           │
 * │    - Receives raw events via context.exposeBinding()                   │
 * │    - Uses message-based activation (no script re-injection needed)     │
 * │                                                                        │
 * │ 2. EVENT CONVERSION (handleRawEvent)                                   │
 * │    - Converts RawBrowserEvent → TimelineEntry (proto)                  │
 * │    - Calls onEntry callback for each converted entry                   │
 * │                                                                        │
 * │ 3. REPLAY PREVIEW (replayPreview) - Delegated to ReplayPreviewService  │
 * │    - Executes TimelineEntry actions to test them before saving         │
 * │    - Uses proto-native executeTimelineEntry for direct execution       │
 * │                                                                        │
 * │ 4. SELECTOR VALIDATION (validateSelector)                              │
 * │    - Checks if a selector matches exactly one element on the page      │
 * └────────────────────────────────────────────────────────────────────────┘
 *
 * KEY CONCEPT - Recording Generation:
 * A counter that increments each time recording starts. Used to detect and
 * ignore stale async operations from previous recording sessions. Without this,
 * callbacks from a stopped recording could affect a newly started one.
 *
 * DESIGN: Replay functionality is delegated to ReplayPreviewService (P1.2)
 * The controller maintains backward-compatible methods that delegate to the service.
 *
 * @see replay-service.ts - Extracted replay execution logic
 */

import type { Page, BrowserContext } from 'rebrowser-playwright';
import type winston from 'winston';
import {
  generateActivationScript,
  generateDeactivationScript,
} from './init-script-generator';
import type { RecordingContextInitializer } from './context-initializer';
import {
  rawBrowserEventToTimelineEntry,
  createNavigateTimelineEntry,
  type RawBrowserEvent,
  type TimelineEntry,
} from '../proto/recording';
import type { ActionReplayResult } from './action-executor';
import { validateSelectorOnPage, type SelectorValidation } from './selector-service';
import {
  ReplayPreviewService,
  type ReplayPreviewRequest,
  type ReplayPreviewResponse,
} from './replay-service';
import type { RecordingState } from './types';
import {
  LOOP_DETECTION_WINDOW_MS,
  LOOP_DETECTION_MAX_NAVIGATIONS,
  LOOP_DETECTION_HISTORY_SIZE,
} from '../constants';
import { logger as defaultLogger, LogContext, scopedLog } from '../utils';

// Re-export for backwards compatibility
export type { RecordingState } from './types';

// =============================================================================
// Types
// =============================================================================

/** Callback invoked for each recorded TimelineEntry */
export type RecordEntryCallback = (entry: TimelineEntry) => void | Promise<void>;

/** Options for starting a recording session */
export interface StartRecordingOptions {
  sessionId: string;
  recordingId?: string;
  onEntry: RecordEntryCallback;
  onError?: (error: Error) => void;
}

// Re-export types from replay-service for backward compatibility
export type { ReplayPreviewRequest, ReplayPreviewResponse } from './replay-service';

// Re-export types from action-executor for convenience
export type { ActionReplayResult, SelectorValidation };

/**
 * RecordModeController manages the recording lifecycle for a browser session.
 *
 * Usage:
 * ```typescript
 * const controller = new RecordModeController(page);
 *
 * await controller.startRecording({
 *   sessionId: 'session-123',
 *   onEntry: (entry) => {
 *     console.log('Recorded entry:', entry);
 *   },
 * });
 *
 * // User performs actions in browser...
 *
 * await controller.stopRecording();
 * ```
 */
export class RecordModeController {
  private page: Page;
  private context: BrowserContext;
  private recordingInitializer: RecordingContextInitializer;
  private state: RecordingState;
  private entryCallback: RecordEntryCallback | null = null;
  private errorCallback: ((error: Error) => void) | null = null;
  private sequenceNum = 0;
  private navigationHandler: (() => void) | null = null;
  private newPageHandler: ((page: Page) => Promise<void>) | null = null;
  private loopDetectionInterval: ReturnType<typeof setInterval> | null = null;
  private lastUrl: string | null = null;
  private recordingGeneration = 0;
  /** Navigation history for redirect loop detection (tracks URL changes with timestamps) */
  private navigationHistory: Array<{ url: string; timestamp: number }> = [];
  /** Flag to prevent multiple loop break attempts */
  private isBreakingLoop = false;
  /** Last URL seen by loop detector (for change detection) */
  private loopDetectorLastUrl: string | null = null;

  /** Replay service for executing timeline entries (delegated) */
  private readonly replayService: ReplayPreviewService;

  /** Logger instance for structured logging */
  private readonly logger: winston.Logger;

  constructor(
    page: Page,
    context: BrowserContext,
    recordingInitializer: RecordingContextInitializer,
    sessionId: string,
    logger?: winston.Logger
  ) {
    this.page = page;
    this.context = context;
    this.recordingInitializer = recordingInitializer;
    this.logger = logger ?? defaultLogger;
    this.state = {
      isRecording: false,
      sessionId,
      actionCount: 0,
    };
    this.replayService = new ReplayPreviewService(page);
  }

  /**
   * Get current recording state.
   */
  getState(): RecordingState {
    return { ...this.state };
  }

  /**
   * Check if recording is currently active.
   */
  isRecording(): boolean {
    return this.state.isRecording;
  }

  /**
   * Start recording user actions.
   *
   * @param options - Recording configuration
   * @returns The recording ID
   */
  async startRecording(options: StartRecordingOptions): Promise<string> {
    if (this.state.isRecording) {
      throw new Error('Recording already in progress');
    }

    if (!options.onEntry) {
      throw new Error('onEntry callback is required');
    }

    // Use crypto.randomUUID() for UUID generation (Node 19+)
    const recordingId = options.recordingId || crypto.randomUUID();

    // Increment generation to invalidate any stale async operations from previous recordings
    this.recordingGeneration++;
    const currentGeneration = this.recordingGeneration;

    // Update state
    this.state = {
      isRecording: true,
      recordingId,
      sessionId: options.sessionId,
      actionCount: 0,
      startedAt: new Date().toISOString(),
    };

    this.entryCallback = options.onEntry;
    this.errorCallback = options.onError || null;
    this.sequenceNum = 0;
    this.navigationHistory = [];
    this.isBreakingLoop = false;
    this.loopDetectorLastUrl = null;

    try {
      // Set up event handler on the context initializer
      // Events from all pages will flow through this handler
      this.recordingInitializer.setEventHandler((rawEvent: RawBrowserEvent) => {
        this.handleRawEvent(rawEvent);
      });
      this.logger.debug(scopedLog(LogContext.RECORDING, 'event handler registered with context initializer'));

      // Capture initial navigation action with current URL
      await this.captureInitialNavigation();

      // Send activation message to the current page
      // The recording script is already injected via context.addInitScript()
      // and is dormant until activated
      await this.activateRecordingOnPage(this.page, recordingId);

      // Setup navigation handler to send activation after page loads
      this.navigationHandler = this.createNavigationHandler(currentGeneration, recordingId);
      this.page.on('load', this.navigationHandler);

      // Setup handler for new pages (tabs) in this context
      this.newPageHandler = async (page: Page) => {
        if (!this.state.isRecording || this.recordingGeneration !== currentGeneration) {
          return;
        }
        this.logger.debug(scopedLog(LogContext.RECORDING, 'new page detected, activating recording'), {
          url: page.url()?.slice(0, 50),
        });
        // Wait for page to be ready, then activate recording
        page.once('load', () => {
          if (this.state.isRecording && this.recordingGeneration === currentGeneration) {
            this.activateRecordingOnPage(page, recordingId).catch((err) => {
              this.logger.warn(scopedLog(LogContext.RECORDING, 'failed to activate recording on new page'), {
                error: err instanceof Error ? err.message : String(err),
              });
            });
          }
        });
      };
      this.context.on('page', this.newPageHandler);

      // Setup polling-based redirect loop detection
      // NOTE: We intentionally do NOT use context.route() here because route interception
      // conflicts with service workers, causing redirect loops on sites like Google.
      // Polling is more reliable than framenavigated events for detecting rapid redirects.
      this.startLoopDetection(currentGeneration);
      this.logger.debug(scopedLog(LogContext.RECORDING, 'loop detection started (polling-based)'));

      return recordingId;
    } catch (error) {
      this.state.isRecording = false;
      this.recordingInitializer.clearEventHandler();
      throw error;
    }
  }

  /**
   * Send activation message to a page to start recording.
   * The recording init script is already present (via context.addInitScript)
   * and listens for this activation message.
   *
   * NOTE: The recording script starts with isActive=true by default, so
   * activation failures during navigation are acceptable - events are
   * still being captured.
   */
  private async activateRecordingOnPage(page: Page, recordingId: string): Promise<void> {
    try {
      // Wait for page to be in a stable state
      try {
        await page.waitForLoadState('domcontentloaded', { timeout: 5000 });
      } catch {
        // Ignore timeout - page might be a blank page or slow loading
      }

      this.logger.debug(scopedLog(LogContext.RECORDING, 'sending activation message'), {
        recordingId,
        url: page.url()?.slice(0, 50),
      });

      // Try to send activation message, but don't fail if it doesn't work
      // The recording script starts active by default, so this is optional
      try {
        await page.evaluate(generateActivationScript(recordingId, this.recordingInitializer.getBindingName()));
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        if (!this.isPageGoneError(message)) {
          this.logger.warn(scopedLog(LogContext.RECORDING, 'activation script failed (non-fatal)'), {
            recordingId,
            error: message,
          });
        }
        // Don't throw - script is already active by default
        return;
      }

      // Diagnostic: Check if the init script ran (optional, non-fatal)
      try {
        const scriptStatus = await page.evaluate(() => {
          return {
            loaded: !!(window as any).__vrooli_recording_script_loaded,
            loadTime: (window as any).__vrooli_recording_script_load_time,
            diagnosticSent: !!(window as any).__vrooli_recording_diagnostic_sent,
            diagnosticError: (window as any).__vrooli_recording_diagnostic_error,
            bindingStatus: (window as any).__vrooli_recording_binding_status,
            recordingActive: !!(window as any).__recordingActive,
          };
        });

        this.logger.info(scopedLog(LogContext.RECORDING, 'init script status after activation'), {
          recordingId,
          url: page.url()?.slice(0, 50),
          ...scriptStatus,
        });
      } catch {
        // Diagnostic failed, but that's okay - script may still be working
        this.logger.debug(scopedLog(LogContext.RECORDING, 'diagnostic check failed (non-fatal)'), {
          recordingId,
        });
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      // Ignore errors for closed/navigating pages
      if (!this.isPageGoneError(message)) {
        throw err;
      }
    }
  }

  /**
   * Send deactivation message to all pages in the context.
   * Called when stopping recording to ensure all pages stop capturing events.
   */
  private async deactivateRecordingOnAllPages(): Promise<void> {
    const pages = this.context.pages();
    this.logger.debug(scopedLog(LogContext.RECORDING, 'deactivating recording on all pages'), {
      pageCount: pages.length,
    });

    const deactivationPromises = pages.map(async (page) => {
      try {
        await page.evaluate(generateDeactivationScript());
      } catch (err) {
        // Ignore errors - page may be closed or navigating
        const message = err instanceof Error ? err.message : String(err);
        if (!this.isPageGoneError(message)) {
          this.logger.warn(scopedLog(LogContext.RECORDING, 'failed to deactivate recording on page'), {
            url: page.url()?.slice(0, 50),
            error: message,
          });
        }
      }
    });

    await Promise.all(deactivationPromises);
    this.logger.debug(scopedLog(LogContext.RECORDING, 'recording deactivated on all pages'));
  }

  /**
   * Create a navigation handler that re-activates recording after page loads.
   * The init script is already present via context.addInitScript(), so we just
   * need to send the activation message again.
   */
  private createNavigationHandler(generation: number, recordingId: string): () => void {
    return (): void => {
      if (!this.state.isRecording || this.recordingGeneration !== generation) {
        return;
      }

      // Capture the navigation event if URL changed
      const newUrl = this.page.url();
      if (newUrl && newUrl !== this.lastUrl) {
        this.captureNavigation(newUrl);
      }

      // Re-activate recording on the new page
      // The init script runs on every page load but starts dormant
      this.activateRecordingOnPage(this.page, recordingId).catch((err) => {
        this.logger.warn(scopedLog(LogContext.RECORDING, 'failed to re-activate recording after navigation'), {
          error: err instanceof Error ? err.message : String(err),
        });
      });
    };
  }

  /**
   * Start polling-based redirect loop detection.
   *
   * Polls the page URL every 200ms and tracks rapid URL changes to the same domain.
   * This approach is more reliable than event-based detection because:
   * - Doesn't rely on framenavigated events which may not fire reliably during rapid redirects
   * - Doesn't use route interception which conflicts with service workers
   * - Can detect loops even when events are delayed or batched
   */
  private startLoopDetection(generation: number): void {
    // Poll every 200ms - fast enough to catch redirect loops
    this.loopDetectionInterval = setInterval(() => {
      // Skip if not recording or generation mismatch
      if (!this.state.isRecording || this.recordingGeneration !== generation) {
        return;
      }

      // Skip if already breaking a loop
      if (this.isBreakingLoop) {
        return;
      }

      let currentUrl: string;
      try {
        currentUrl = this.page.url();
      } catch {
        // Page might be closed
        return;
      }

      // Skip special pages
      if (!currentUrl || currentUrl === 'about:blank' || currentUrl.startsWith('chrome:')) {
        return;
      }

      // Only process if URL changed since last check
      if (currentUrl === this.loopDetectorLastUrl) {
        return;
      }

      this.loopDetectorLastUrl = currentUrl;
      this.logger.debug(scopedLog(LogContext.RECORDING, 'URL change detected'), {
        url: currentUrl,
        historySize: this.navigationHistory.length,
      });

      // Check for redirect loop
      if (this.checkForRedirectLoop(currentUrl)) {
        let hostname = 'unknown';
        try {
          hostname = new URL(currentUrl).hostname;
        } catch {
          // Invalid URL
        }

        this.logger.error(scopedLog(LogContext.RECORDING, 'redirect loop detected - breaking loop'), {
          hostname,
          historySize: this.navigationHistory.length,
        });
        this.handleError(
          new Error(
            `Redirect loop detected on ${hostname}. ` +
              `Navigating away to break the loop.`
          )
        );

        // Set flag to prevent re-entry
        this.isBreakingLoop = true;

        // Break the loop by navigating to about:blank
        this.page.goto('about:blank', { timeout: 5000 })
          .then(() => {
            this.logger.info(scopedLog(LogContext.RECORDING, 'successfully broke redirect loop'));
          })
          .catch((err) => {
            this.logger.warn(scopedLog(LogContext.RECORDING, 'failed to navigate away from loop'), {
              error: err instanceof Error ? err.message : String(err),
            });
          })
          .finally(() => {
            // Reset after a delay to allow re-detection if user navigates back
            setTimeout(() => {
              this.isBreakingLoop = false;
              this.navigationHistory = [];
              this.loopDetectorLastUrl = null;
            }, 2000);
          });
      }
    }, 200);
  }

  /**
   * Stop the loop detection polling.
   */
  private stopLoopDetection(): void {
    if (this.loopDetectionInterval) {
      clearInterval(this.loopDetectionInterval);
      this.loopDetectionInterval = null;
    }
  }

  /**
   * Check for redirect loop without capturing the navigation.
   * Used by request handler for early detection.
   */
  private checkForRedirectLoop(newUrl: string): boolean {
    const now = Date.now();

    // Add to history
    this.navigationHistory.push({ url: newUrl, timestamp: now });

    // Trim old entries outside the detection window
    this.navigationHistory = this.navigationHistory.filter(
      (entry) => now - entry.timestamp < LOOP_DETECTION_WINDOW_MS
    );

    // Keep history bounded to prevent unbounded memory growth
    if (this.navigationHistory.length > LOOP_DETECTION_HISTORY_SIZE) {
      this.navigationHistory = this.navigationHistory.slice(-LOOP_DETECTION_HISTORY_SIZE);
    }

    // Detect loop: N+ navigations to same domain in window
    if (this.navigationHistory.length >= LOOP_DETECTION_MAX_NAVIGATIONS) {
      try {
        const newHostname = new URL(newUrl).hostname;
        const domains = this.navigationHistory.map((e) => {
          try {
            return new URL(e.url).hostname;
          } catch {
            return '';
          }
        });
        const sameDomainCount = domains.filter((d) => d === newHostname).length;

        if (sameDomainCount >= LOOP_DETECTION_MAX_NAVIGATIONS) {
          this.logger.warn(scopedLog(LogContext.RECORDING, 'redirect loop detected (early)'), {
            url: newUrl,
            hostname: newHostname,
            count: sameDomainCount,
            windowMs: LOOP_DETECTION_WINDOW_MS,
          });
          return true;
        }
      } catch {
        // Invalid URL, not a loop
      }
    }

    return false;
  }

  /**
   * Check if an error message indicates the page is gone or context destroyed.
   * These are expected during navigation and should be handled gracefully.
   */
  private isPageGoneError(message: string): boolean {
    return message.includes('closed') ||
           message.includes('navigating') ||
           message.includes('detached') ||
           message.includes('destroyed') ||
           message.includes('Target closed') ||
           message.includes('context was destroyed');
  }

  /**
   * Stop recording and cleanup.
   */
  async stopRecording(): Promise<{ recordingId: string; actionCount: number }> {
    if (!this.state.isRecording || !this.state.recordingId) {
      throw new Error('No recording in progress');
    }

    const result = {
      recordingId: this.state.recordingId,
      actionCount: this.state.actionCount,
    };

    this.logger.debug(scopedLog(LogContext.RECORDING, 'stopping recording'), {
      recordingId: result.recordingId,
      actionCount: result.actionCount,
    });

    try {
      // Remove navigation handler
      if (this.navigationHandler) {
        this.page.off('load', this.navigationHandler);
        this.navigationHandler = null;
      }

      // Remove new page handler
      if (this.newPageHandler) {
        this.context.off('page', this.newPageHandler);
        this.newPageHandler = null;
      }

      // Stop loop detection
      this.stopLoopDetection();
      this.isBreakingLoop = false;
      this.loopDetectorLastUrl = null;

      // Clear event handler from context initializer
      // This stops events from being processed even if they arrive
      this.recordingInitializer.clearEventHandler();
      this.logger.debug(scopedLog(LogContext.RECORDING, 'event handler cleared from context initializer'));

      // Send deactivation message to all pages in context
      // This tells the recording script to stop capturing events
      await this.deactivateRecordingOnAllPages();
    } finally {
      // Reset state
      this.state = {
        isRecording: false,
        sessionId: this.state.sessionId,
        actionCount: 0,
      };
      this.entryCallback = null;
      this.errorCallback = null;
      this.navigationHistory = [];
    }

    return result;
  }

  /**
   * Capture initial navigation action at the start of recording.
   */
  private async captureInitialNavigation(): Promise<void> {
    try {
      const url = this.page.url();
      this.lastUrl = url;

      if (!url || url === 'about:blank') {
        return;
      }

      const entry = createNavigateTimelineEntry(url, {
        sessionId: this.state.sessionId,
        sequenceNum: this.sequenceNum++,
      });

      this.state.actionCount++;

      if (this.entryCallback) {
        const result = this.entryCallback(entry);
        if (result instanceof Promise) {
          await result;
        }
      }
    } catch (error) {
      this.logger.error(scopedLog(LogContext.RECORDING, 'failed to capture initial navigation'), {
        error: error instanceof Error ? error.message : String(error),
      });
    }
  }

  /**
   * Capture a navigation action when URL changes during recording.
   * Note: Redirect loop detection is handled earlier in framenavigated handler.
   */
  private captureNavigation(url: string): void {
    if (!this.state.isRecording || !this.entryCallback || !this.state.recordingId) {
      return;
    }

    if (url === this.lastUrl || url === 'about:blank') {
      return;
    }

    this.lastUrl = url;

    try {
      const entry = createNavigateTimelineEntry(url, {
        sessionId: this.state.sessionId,
        sequenceNum: this.sequenceNum++,
      });

      this.state.actionCount++;

      const result = this.entryCallback(entry);
      if (result instanceof Promise) {
        result.catch((err: unknown) => {
          const message = err instanceof Error ? err.message : String(err);
          this.handleError(new Error(`Entry callback failed: ${message}`));
        });
      }
    } catch (error) {
      this.handleError(error instanceof Error ? error : new Error(String(error)));
    }
  }

  /**
   * Replay recorded entries for preview/testing.
   *
   * Delegates to ReplayPreviewService for execution.
   * This maintains backward compatibility while centralizing replay logic.
   *
   * @param request - Replay configuration
   * @returns Replay results
   */
  async replayPreview(request: ReplayPreviewRequest): Promise<ReplayPreviewResponse> {
    return this.replayService.replayPreview(request);
  }

  /**
   * Validate a selector on the current page.
   *
   * Delegates to the standalone validateSelectorOnPage function from selector-service.ts.
   * This eliminates code duplication and centralizes selector validation logic.
   */
  async validateSelector(selector: string): Promise<SelectorValidation> {
    return validateSelectorOnPage(this.page, selector);
  }

  /**
   * Handle a raw event from the page and convert to TimelineEntry.
   *
   * Event flow tracing is logged using LogContext.EVENT_FLOW for debugging.
   */
  private handleRawEvent(raw: RawBrowserEvent): void {
    // Generate event ID for tracing
    const eventId = `${raw.actionType}-${Date.now()}-${this.sequenceNum}`;

    // Log event received
    this.logger.debug(scopedLog(LogContext.EVENT_FLOW, 'event received'), {
      eventId,
      actionType: raw.actionType,
      url: raw.url?.slice(0, 50),
      hasSelector: !!raw.selector?.primary,
      timestamp: raw.timestamp,
    });

    if (!this.state.isRecording || !this.entryCallback || !this.state.recordingId) {
      this.logger.debug(scopedLog(LogContext.EVENT_FLOW, 'event dropped (not recording)'), {
        eventId,
        isRecording: this.state.isRecording,
        hasCallback: !!this.entryCallback,
        hasRecordingId: !!this.state.recordingId,
      });
      return;
    }

    try {
      // For navigate events from the injected script, update lastUrl
      if (raw.actionType === 'navigate' && raw.payload?.targetUrl) {
        this.lastUrl = raw.payload.targetUrl as string;
      }

      // Convert directly to TimelineEntry (proto)
      const entry = rawBrowserEventToTimelineEntry(raw, {
        sessionId: this.state.sessionId,
        sequenceNum: this.sequenceNum++,
      });

      this.state.actionCount++;

      // Log event converted
      this.logger.debug(scopedLog(LogContext.EVENT_FLOW, 'event converted'), {
        eventId,
        entryType: entry.action?.type,
        sequenceNum: entry.sequenceNum,
        actionCount: this.state.actionCount,
      });

      // Invoke callback
      const result = this.entryCallback(entry);
      if (result instanceof Promise) {
        result.catch((err: unknown) => {
          const message = err instanceof Error ? err.message : String(err);
          this.handleError(new Error(`Entry callback failed: ${message}`));
        });
      }
    } catch (error) {
      this.logger.error(scopedLog(LogContext.EVENT_FLOW, 'conversion failed'), {
        eventId,
        error: error instanceof Error ? error.message : String(error),
        rawEventType: raw.actionType,
      });
      this.handleError(error instanceof Error ? error : new Error(String(error)));
    }
  }

  /**
   * Reset recording state on a page.
   *
   * Cleans up existing listeners and re-activates recording.
   * Safe to call multiple times (idempotent).
   * Uses the browser-side cleanup function to properly reset state.
   *
   * @param page - The page to reset (defaults to main page)
   */
  async resetRecordingOnPage(page?: Page): Promise<void> {
    const targetPage = page || this.page;
    const url = targetPage.url()?.slice(0, 50);

    this.logger.debug(scopedLog(LogContext.RECORDING, 'resetting page recording'), { url });

    try {
      // Trigger cleanup in browser - this removes all event listeners
      // and resets state to allow fresh re-initialization
      await targetPage.evaluate(() => {
        if (typeof (window as any).__vrooli_recording_cleanup === 'function') {
          (window as any).__vrooli_recording_cleanup();
        }
      });
      this.logger.debug(scopedLog(LogContext.RECORDING, 'browser cleanup executed'), { url });
    } catch (e) {
      // Ignore - cleanup function may not exist or page may be navigating
      const error = e instanceof Error ? e.message : String(e);
      if (!this.isPageGoneError(error)) {
        this.logger.debug(scopedLog(LogContext.RECORDING, 'cleanup failed (may be expected)'), {
          url,
          error,
        });
      }
    }

    // Re-activate if currently recording
    if (this.state.isRecording && this.state.recordingId) {
      await this.activateRecordingOnPage(targetPage, this.state.recordingId);
      this.logger.debug(scopedLog(LogContext.RECORDING, 'recording re-activated after reset'), { url });
    }
  }

  /**
   * Handle errors during recording.
   */
  private handleError(error: Error): void {
    if (this.errorCallback) {
      this.errorCallback(error);
    } else {
      this.logger.error(scopedLog(LogContext.RECORDING, 'recording error'), { error: error.message });
    }
  }
}

/**
 * Create a new RecordModeController for a page.
 *
 * @param page - The page to record on
 * @param context - The browser context (for multi-tab support)
 * @param recordingInitializer - The context initializer for event handling
 * @param sessionId - The session ID
 * @param logger - Optional logger instance
 */
export function createRecordModeController(
  page: Page,
  context: BrowserContext,
  recordingInitializer: RecordingContextInitializer,
  sessionId: string,
  logger?: winston.Logger
): RecordModeController {
  return new RecordModeController(page, context, recordingInitializer, sessionId, logger);
}
