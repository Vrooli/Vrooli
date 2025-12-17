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

import type { Page } from 'playwright';
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
  private pendingInjectionTimeouts: Set<ReturnType<typeof setTimeout>> = new Set();
  private lastUrl: string | null = null;
  private recordingGeneration = 0;

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

      // Remove navigation handler
      if (this.navigationHandler) {
        this.page.off('load', this.navigationHandler);
        this.navigationHandler = null;
      }

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
   */
  private async injectRecordingScript(): Promise<void> {
    await this.page.evaluate(getRecordingScript());
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
