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
 * │ 3. REPLAY PREVIEW (replayPreview)                                      │
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
 */

import type { Page } from 'playwright';
import { getRecordingScript, getCleanupScript } from './injector';
import {
  rawBrowserEventToTimelineEntry,
  createNavigateTimelineEntry,
  type RawBrowserEvent,
  type TimelineEntry,
} from '../proto/recording';
import {
  executeTimelineEntry,
  type ExecutorContext,
  type ActionReplayResult,
  type SelectorValidation,
} from './action-executor';

// =============================================================================
// Types
// =============================================================================

/** Recording state */
export interface RecordingState {
  isRecording: boolean;
  recordingId?: string;
  sessionId: string;
  actionCount: number;
  startedAt?: string;
}

/** Callback invoked for each recorded TimelineEntry */
export type RecordEntryCallback = (entry: TimelineEntry) => void | Promise<void>;

/** Options for starting a recording session */
export interface StartRecordingOptions {
  sessionId: string;
  recordingId?: string;
  onEntry: RecordEntryCallback;
  onError?: (error: Error) => void;
}

/** Configuration for script injection retry logic */
interface InjectionRetryConfig {
  maxAttempts: number;
  baseDelayMs: number;
}

/** Request for replay preview */
export interface ReplayPreviewRequest {
  entries: TimelineEntry[];
  limit?: number;
  stopOnFailure?: boolean;
  actionTimeout?: number;
}

/** Response from replay preview */
export interface ReplayPreviewResponse {
  success: boolean;
  totalActions: number;
  passedActions: number;
  failedActions: number;
  results: ActionReplayResult[];
  totalDurationMs: number;
  stoppedEarly: boolean;
}

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

  constructor(page: Page, sessionId: string) {
    this.page = page;
    this.state = {
      isRecording: false,
      sessionId,
      actionCount: 0,
    };
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
      this.scheduleInjectionWithRetry(generation, { maxAttempts: 3, baseDelayMs: 100 });
    };
  }

  /**
   * Schedule script injection with exponential backoff retry.
   */
  private scheduleInjectionWithRetry(
    generation: number,
    config: InjectionRetryConfig,
    attempt = 0
  ): void {
    if (!this.state.isRecording || this.recordingGeneration !== generation) {
      return;
    }

    const delay = attempt === 0
      ? config.baseDelayMs
      : config.baseDelayMs * Math.pow(2, attempt);

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

        if (attempt < config.maxAttempts - 1) {
          this.scheduleInjectionWithRetry(generation, config, attempt + 1);
        } else {
          this.handleError(
            new Error(`Failed to re-inject recording script after ${config.maxAttempts} attempts: ${message}`)
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
   * Track in-flight replay operations for idempotency.
   */
  private pendingReplays: Map<string, Promise<ReplayPreviewResponse>> = new Map();

  /**
   * Replay recorded entries for preview/testing.
   */
  async replayPreview(request: ReplayPreviewRequest): Promise<ReplayPreviewResponse> {
    const {
      entries,
      limit,
      stopOnFailure = true,
      actionTimeout = 10000,
    } = request;

    const entriesToReplay = limit ? entries.slice(0, limit) : entries;
    const replayKey = this.generateReplayKey(entriesToReplay);

    const pendingReplay = this.pendingReplays.get(replayKey);
    if (pendingReplay) {
      return pendingReplay;
    }

    const replayPromise = this.executeReplay(entriesToReplay, stopOnFailure, actionTimeout);
    this.pendingReplays.set(replayKey, replayPromise);

    try {
      return await replayPromise;
    } finally {
      this.pendingReplays.delete(replayKey);
    }
  }

  /**
   * Generate a stable key for replay idempotency tracking.
   */
  private generateReplayKey(entries: TimelineEntry[]): string {
    return entries.map((e) => `${e.id}:${e.sequenceNum}`).join('|');
  }

  /**
   * Internal replay execution logic.
   */
  private async executeReplay(
    entriesToReplay: TimelineEntry[],
    stopOnFailure: boolean,
    actionTimeout: number
  ): Promise<ReplayPreviewResponse> {
    const results: ActionReplayResult[] = [];
    let stoppedEarly = false;
    const startTime = Date.now();

    // Create executor context once for all entries
    const context: ExecutorContext = {
      page: this.page,
      timeout: actionTimeout,
      validateSelector: (sel: string) => this.validateSelector(sel),
    };

    for (const entry of entriesToReplay) {
      // Execute using proto-native executor
      const result = await executeTimelineEntry(entry, context);

      // Capture screenshot on error
      if (!result.success) {
        try {
          const screenshot = await this.page.screenshot({ type: 'png' });
          result.screenshotOnError = screenshot.toString('base64');
        } catch {
          // Ignore screenshot errors
        }
      }

      results.push(result);

      if (!result.success && stopOnFailure) {
        stoppedEarly = true;
        break;
      }
    }

    const passedActions = results.filter((r) => r.success).length;
    const failedActions = results.filter((r) => !r.success).length;

    return {
      success: failedActions === 0,
      totalActions: results.length,
      passedActions,
      failedActions,
      results,
      totalDurationMs: Date.now() - startTime,
      stoppedEarly,
    };
  }

  /**
   * Validate a selector on the current page.
   */
  async validateSelector(selector: string): Promise<SelectorValidation> {
    try {
      const isXPath = selector.startsWith('/') || selector.startsWith('(');

      if (isXPath) {
        const count = await this.page.evaluate<number>(`
          (function() {
            try {
              const result = document.evaluate(
                ${JSON.stringify(selector)},
                document,
                null,
                XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
                null
              );
              return result.snapshotLength;
            } catch {
              return -1;
            }
          })()
        `);

        if (count === -1) {
          return { valid: false, matchCount: 0, selector, error: 'Invalid XPath expression' };
        }

        return {
          valid: count === 1,
          matchCount: count,
          selector,
        };
      } else {
        const count = await this.page.locator(selector).count();
        return {
          valid: count === 1,
          matchCount: count,
          selector,
        };
      }
    } catch (error) {
      return {
        valid: false,
        matchCount: 0,
        selector,
        error: error instanceof Error ? error.message : String(error),
      };
    }
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
