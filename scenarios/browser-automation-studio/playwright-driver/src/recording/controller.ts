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
 * │    - Injects event listener script into browser pages                  │
 * │    - Receives raw events via page.exposeFunction()                     │
 * │    - Re-injects script after navigation (pages lose injected JS)       │
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

import type { Page } from 'rebrowser-playwright';
import type winston from 'winston';
import { getRecordingScript, getCleanupScript } from './injector';
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
  INJECTION_RETRY_MAX_ATTEMPTS,
  INJECTION_RETRY_BASE_DELAY_MS,
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
  private state: RecordingState;
  private entryCallback: RecordEntryCallback | null = null;
  private errorCallback: ((error: Error) => void) | null = null;
  private sequenceNum = 0;
  private exposedFunctionName = '__recordAction';
  private isExposed = false;
  private navigationHandler: (() => void) | null = null;
  private loopDetectionInterval: ReturnType<typeof setInterval> | null = null;
  private pendingInjectionTimeouts: Set<ReturnType<typeof setTimeout>> = new Set();
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

  constructor(page: Page, sessionId: string, logger?: winston.Logger) {
    this.page = page;
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
      // Expose callback function to page context if not already exposed
      if (!this.isExposed) {
        await this.page.exposeFunction(this.exposedFunctionName, (rawEvent: RawBrowserEvent) => {
          this.handleRawEvent(rawEvent);
        });
        this.isExposed = true;
      }

      // Capture initial navigation action with current URL
      await this.captureInitialNavigation();

      // Inject recording script
      await this.injectRecordingScript();

      // Setup navigation handler to re-inject script after page loads
      this.navigationHandler = this.createNavigationHandler(currentGeneration);
      this.page.on('load', this.navigationHandler);

      // Setup polling-based redirect loop detection
      // NOTE: We intentionally do NOT use context.route() here because route interception
      // conflicts with service workers, causing redirect loops on sites like Google.
      // Polling is more reliable than framenavigated events for detecting rapid redirects.
      this.startLoopDetection(currentGeneration);
      this.logger.debug(scopedLog(LogContext.RECORDING, 'loop detection started (polling-based)'));

      return recordingId;
    } catch (error) {
      this.state.isRecording = false;
      throw error;
    }
  }

  /**
   * Create a navigation handler that re-injects the recording script after page loads.
   */
  private createNavigationHandler(generation: number): () => void {
    return (): void => {
      if (!this.state.isRecording || this.recordingGeneration !== generation) {
        return;
      }

      // Capture the navigation event if URL changed
      const newUrl = this.page.url();
      if (newUrl && newUrl !== this.lastUrl) {
        this.captureNavigation(newUrl);
      }

      // Schedule injection with retry support
      this.scheduleInjectionWithRetry(generation);
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
   * Schedule script injection with exponential backoff retry.
   * Uses constants from ../constants.ts for retry configuration.
   */
  private scheduleInjectionWithRetry(
    generation: number,
    attempt = 0
  ): void {
    if (!this.state.isRecording || this.recordingGeneration !== generation) {
      return;
    }

    const delay = attempt === 0
      ? INJECTION_RETRY_BASE_DELAY_MS
      : INJECTION_RETRY_BASE_DELAY_MS * Math.pow(2, attempt);

    const timeoutId = setTimeout(async () => {
      this.pendingInjectionTimeouts.delete(timeoutId);

      if (!this.state.isRecording || this.recordingGeneration !== generation) {
        return;
      }

      try {
        await this.injectRecordingScript();
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : String(err);

        if (this.isPageGoneError(message)) {
          return;
        }

        if (attempt < INJECTION_RETRY_MAX_ATTEMPTS - 1) {
          this.scheduleInjectionWithRetry(generation, attempt + 1);
        } else {
          this.handleError(
            new Error(`Failed to re-inject recording script after ${INJECTION_RETRY_MAX_ATTEMPTS} attempts: ${message}`)
          );
        }
      }
    }, delay);

    this.pendingInjectionTimeouts.add(timeoutId);
  }

  /**
   * Check if an error message indicates the page is gone.
   */
  private isPageGoneError(message: string): boolean {
    return message.includes('closed') ||
           message.includes('navigating') ||
           message.includes('detached');
  }

  /**
   * Check if an error is a transient context destruction that can be retried.
   * This happens when the page navigates during script injection.
   */
  private isContextDestroyedError(message: string): boolean {
    return message.includes('Execution context was destroyed') ||
           message.includes('context was destroyed') ||
           message.includes('Cannot find context');
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

    try {
      // Cancel any pending injection timeouts
      for (const timeoutId of this.pendingInjectionTimeouts) {
        clearTimeout(timeoutId);
      }
      this.pendingInjectionTimeouts.clear();

      // Remove navigation handlers
      if (this.navigationHandler) {
        this.page.off('load', this.navigationHandler);
        this.navigationHandler = null;
      }

      // Stop loop detection
      this.stopLoopDetection();
      this.isBreakingLoop = false;
      this.loopDetectorLastUrl = null;

      // Inject cleanup script
      await this.page.evaluate(getCleanupScript()).catch(() => {
        // Ignore errors during cleanup (page may have navigated)
      });
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
   * Inject the recording script into the current page.
   * Includes retry logic to handle transient "Execution context was destroyed"
   * errors that occur when the page navigates during injection.
   */
  private async injectRecordingScript(): Promise<void> {
    let lastError: Error | undefined;

    for (let attempt = 0; attempt < INJECTION_RETRY_MAX_ATTEMPTS; attempt++) {
      try {
        // Wait for the page to be in a stable state before injecting
        try {
          await this.page.waitForLoadState('domcontentloaded', { timeout: 5000 });
        } catch {
          // Ignore timeout - page might be a blank page or slow loading
        }

        await this.page.evaluate(getRecordingScript());
        return; // Success - exit the retry loop
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : String(err);
        lastError = err instanceof Error ? err : new Error(message);

        // If page is gone (closed/detached), don't retry
        if (this.isPageGoneError(message)) {
          throw lastError;
        }

        // If context was destroyed (navigation), retry with backoff
        if (this.isContextDestroyedError(message) && attempt < INJECTION_RETRY_MAX_ATTEMPTS - 1) {
          const delay = INJECTION_RETRY_BASE_DELAY_MS * Math.pow(2, attempt);
          await new Promise(resolve => setTimeout(resolve, delay));
          continue;
        }

        // Unknown error or max retries reached
        throw lastError;
      }
    }

    // Should not reach here, but just in case
    throw lastError || new Error('Failed to inject recording script');
  }

  /**
   * Handle a raw event from the page and convert to TimelineEntry.
   */
  private handleRawEvent(raw: RawBrowserEvent): void {
    if (!this.state.isRecording || !this.entryCallback || !this.state.recordingId) {
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

      // Invoke callback
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
 */
export function createRecordModeController(
  page: Page,
  sessionId: string,
  logger?: winston.Logger
): RecordModeController {
  return new RecordModeController(page, sessionId, logger);
}
