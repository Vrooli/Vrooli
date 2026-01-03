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
  private loopDetectionRouteHandler: ((route: import('rebrowser-playwright').Route, request: import('rebrowser-playwright').Request) => Promise<void>) | null = null;
  private pendingInjectionTimeouts: Set<ReturnType<typeof setTimeout>> = new Set();
  private lastUrl: string | null = null;
  private recordingGeneration = 0;
  /** Navigation history for redirect loop detection */
  private navigationHistory: Array<{ url: string; timestamp: number }> = [];
  /** Domain currently in a detected redirect loop - requests to this domain will be blocked */
  private blockedLoopDomain: string | null = null;

  /** Replay service for executing timeline entries (delegated) */
  private readonly replayService: ReplayPreviewService;

  constructor(page: Page, sessionId: string) {
    this.page = page;
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
    this.blockedLoopDomain = null;

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

      // Setup route handler for redirect loop detection AND blocking
      // Using context.route() allows us to ABORT requests when a loop is detected
      // Register on CONTEXT level to catch requests from ALL pages (including new tabs)
      this.loopDetectionRouteHandler = this.createLoopDetectionRouteHandler(currentGeneration);
      const context = this.page.context();
      await context.route('**/*', this.loopDetectionRouteHandler);
      console.log('[LoopDetection] Route handler registered on context for redirect loop detection');

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
   * Create a route handler for redirect loop detection AND blocking.
   * Using context.route() allows us to ABORT requests when a loop is detected,
   * actually stopping the redirect loop instead of just detecting it.
   */
  private createLoopDetectionRouteHandler(
    generation: number
  ): (route: import('rebrowser-playwright').Route, request: import('rebrowser-playwright').Request) => Promise<void> {
    return async (route, request): Promise<void> => {
      try {
        const url = request.url();

        // Only process navigation requests for loop detection
        if (!request.isNavigationRequest()) {
          await route.continue();
          return;
        }

        // Only handle main frame navigations
        const frame = request.frame();
        if (!frame || frame.parentFrame() !== null) {
          await route.continue();
          return;
        }

        // If we've already detected a loop on this domain, BLOCK the request
        if (this.blockedLoopDomain) {
          try {
            const hostname = new URL(url).hostname;
            if (hostname === this.blockedLoopDomain) {
              console.log('[LoopDetection] BLOCKING request to looping domain:', hostname);
              await route.abort('blockedbyclient');
              return;
            }
          } catch {
            // Invalid URL, continue
          }
        }

        // Skip loop detection if not recording
        if (!this.state.isRecording || this.recordingGeneration !== generation) {
          await route.continue();
          return;
        }

        if (!url || url === 'about:blank') {
          await route.continue();
          return;
        }

        console.log('[LoopDetection] Checking navigation:', url, 'History size:', this.navigationHistory.length);

        // Check for redirect loop
        if (this.checkForRedirectLoop(url)) {
          let hostname = 'unknown';
          try {
            hostname = new URL(url).hostname;
          } catch {
            // Invalid URL
          }

          // Set the blocked domain to prevent further requests
          this.blockedLoopDomain = hostname;

          console.error('[LoopDetection] REDIRECT LOOP DETECTED - BLOCKING:', hostname);
          this.handleError(
            new Error(
              `Redirect loop detected on ${hostname}. ` +
                `This may be caused by ad blocking - try adding the domain to the whitelist.`
            )
          );

          // ABORT this request to break the loop
          await route.abort('blockedbyclient');
          return;
        }

        // Allow the request to continue
        await route.continue();
      } catch (err) {
        console.error('[LoopDetection] Error in route handler:', err);
        // On error, try to continue the request
        try {
          await route.continue();
        } catch {
          // Route may have already been handled
        }
      }
    };
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
          console.warn('[RecordModeController] Redirect loop detected (early)', {
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
      if (this.loopDetectionRouteHandler) {
        const context = this.page.context();
        await context.unroute('**/*', this.loopDetectionRouteHandler);
        this.loopDetectionRouteHandler = null;
      }
      this.blockedLoopDomain = null;

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
      console.error('[RecordModeController] Failed to capture initial navigation:', error);
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
      console.error('[RecordModeController] Error:', error.message);
    }
  }
}

/**
 * Create a new RecordModeController for a page.
 */
export function createRecordModeController(page: Page, sessionId: string): RecordModeController {
  return new RecordModeController(page, sessionId);
}
