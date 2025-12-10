/**
 * Recording Controller
 *
 * Orchestrates Record Mode functionality:
 * - Manages recording session state
 * - Injects JavaScript event listeners into pages
 * - Receives events via page.exposeFunction()
 * - Normalizes raw events to RecordedAction
 * - Streams actions to callback for persistence
 */

import type { Page } from 'playwright';
import { v4 as uuidv4 } from 'uuid';
import { getRecordingScript, getCleanupScript } from './injector';
import type {
  RecordedAction,
  RawBrowserEvent,
  RecordingState,
  SelectorValidation,
  ActionReplayResult,
  ReplayPreviewRequest,
  ReplayPreviewResponse,
} from './types';
import {
  normalizeActionType,
  toRecordedActionKind,
  buildTypedActionPayload,
  calculateActionConfidence,
} from './action-types';
import {
  getActionExecutor,
  type ActionExecutorContext,
} from './action-executor';

/**
 * Callback type for streaming recorded actions.
 */
export type RecordActionCallback = (action: RecordedAction) => void | Promise<void>;

/**
 * Options for starting a recording session.
 */
export interface StartRecordingOptions {
  /** Session ID this recording is attached to */
  sessionId: string;
  /** Optional recording ID (generated if not provided) */
  recordingId?: string;
  /** Callback invoked for each recorded action */
  onAction: RecordActionCallback;
  /** Optional error callback */
  onError?: (error: Error) => void;
}

/**
 * RecordModeController manages the recording lifecycle for a browser session.
 *
 * Usage:
 * ```typescript
 * const controller = new RecordModeController(page);
 *
 * await controller.startRecording({
 *   sessionId: 'session-123',
 *   onAction: (action) => {
 *     console.log('Recorded action:', action);
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
  private actionCallback: RecordActionCallback | null = null;
  private errorCallback: ((error: Error) => void) | null = null;
  private sequenceNum = 0;
  private exposedFunctionName = '__recordAction';
  private isExposed = false;
  private navigationHandler: (() => void) | null = null;
  /** Track pending injection timeouts to cancel on stop */
  private pendingInjectionTimeouts: Set<ReturnType<typeof setTimeout>> = new Set();
  /**
   * Monotonically increasing recording generation counter.
   * Used to detect stale async operations that started during a previous recording session.
   * Each startRecording() increments this; async operations check their captured generation
   * against current to detect if they should abort.
   */
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

    const recordingId = options.recordingId || uuidv4();

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

    this.actionCallback = options.onAction;
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

      // Inject recording script
      await this.injectRecordingScript();

      // Re-inject on navigation with robust error handling
      // Capture generation at handler creation time to detect staleness
      this.navigationHandler = (): void => {
        if (!this.state.isRecording) return;
        // Check if this handler is from a stale recording session
        if (this.recordingGeneration !== currentGeneration) return;

        // Use multiple injection attempts with backoff
        // This handles cases where page isn't fully ready after load event
        const attemptInjection = async (attempt: number, maxAttempts: number): Promise<void> => {
          // Check both recording state AND generation before each attempt
          // Generation check catches the case where recording was stopped and restarted
          if (!this.state.isRecording || this.recordingGeneration !== currentGeneration) {
            return; // Stale operation, abort
          }

          try {
            await this.injectRecordingScript();
          } catch (err: unknown) {
            const message = err instanceof Error ? err.message : String(err);

            // Ignore errors about page being closed/navigated away
            if (
              message.includes('closed') ||
              message.includes('navigating') ||
              message.includes('detached')
            ) {
              return; // Page is gone, nothing to do
            }

            if (attempt < maxAttempts) {
              // Exponential backoff: 100ms, 200ms, 400ms
              const delay = 100 * Math.pow(2, attempt);
              const timeoutId = setTimeout(() => {
                this.pendingInjectionTimeouts.delete(timeoutId);
                // Check generation again before retry (stop might have been called during delay)
                if (this.recordingGeneration !== currentGeneration) return;
                attemptInjection(attempt + 1, maxAttempts).catch(() => {
                  // Final fallback - just log, don't crash
                });
              }, delay);
              this.pendingInjectionTimeouts.add(timeoutId);
            } else {
              this.handleError(
                new Error(`Failed to re-inject recording script after ${maxAttempts} attempts: ${message}`)
              );
            }
          }
        };

        // Small initial delay to ensure page is ready, then attempt injection
        const initialTimeoutId = setTimeout(() => {
          this.pendingInjectionTimeouts.delete(initialTimeoutId);
          // Check generation before starting injection attempts
          if (this.recordingGeneration !== currentGeneration) return;
          attemptInjection(0, 3).catch(() => {
            // Swallow - errors already handled in attemptInjection
          });
        }, 100);
        this.pendingInjectionTimeouts.add(initialTimeoutId);
      };
      this.page.on('load', this.navigationHandler);

      return recordingId;
    } catch (error) {
      // Reset state on failure
      this.state.isRecording = false;
      throw error;
    }
  }

  /**
   * Stop recording and cleanup.
   *
   * @returns Summary of the recording session
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
      // Cancel any pending injection timeouts to prevent work after stop
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
      this.actionCallback = null;
      this.errorCallback = null;
    }

    return result;
  }

  /**
   * Track in-flight replay operations to prevent duplicate concurrent replays.
   * Key: stable hash of action IDs being replayed.
   */
  private pendingReplays: Map<string, Promise<ReplayPreviewResponse>> = new Map();

  /**
   * Replay recorded actions for preview/testing.
   *
   * Executes actions one by one and returns detailed results for each.
   * This allows users to test their recording before generating a workflow.
   *
   * Idempotency guarantees:
   * - Concurrent replay requests with the same actions will await the first
   * - Actions are identified by their stable ID, preventing duplicate execution
   * - If the same replay is requested while in-flight, returns the same result
   *
   * @param request - Replay request with actions and options
   * @returns Detailed results for each action
   */
  async replayPreview(request: ReplayPreviewRequest): Promise<ReplayPreviewResponse> {
    const {
      actions,
      limit,
      stopOnFailure = true,
      actionTimeout = 10000,
    } = request;

    // Determine how many actions to replay
    const actionsToReplay = limit ? actions.slice(0, limit) : actions;

    // Generate a stable key for this replay operation
    // Uses action IDs and limit to create a unique identifier
    const replayKey = this.generateReplayKey(actionsToReplay);

    // Idempotency: Check for in-flight replay with same actions
    const pendingReplay = this.pendingReplays.get(replayKey);
    if (pendingReplay) {
      // Return the same result as the in-flight replay
      return pendingReplay;
    }

    // Create the replay promise and track it
    const replayPromise = this.executeReplay(actionsToReplay, stopOnFailure, actionTimeout);
    this.pendingReplays.set(replayKey, replayPromise);

    try {
      return await replayPromise;
    } finally {
      // Clean up tracking after completion
      this.pendingReplays.delete(replayKey);
    }
  }

  /**
   * Generate a stable key for replay idempotency tracking.
   * Uses action IDs to create a unique identifier for the replay set.
   */
  private generateReplayKey(actions: RecordedAction[]): string {
    // Use a simple hash of action IDs and sequence numbers
    const actionIds = actions.map((a) => `${a.id}:${a.sequenceNum}`).join('|');
    return actionIds;
  }

  /**
   * Internal replay execution logic.
   * Separated from replayPreview to enable idempotency tracking.
   */
  private async executeReplay(
    actionsToReplay: RecordedAction[],
    stopOnFailure: boolean,
    actionTimeout: number
  ): Promise<ReplayPreviewResponse> {
    const results: ActionReplayResult[] = [];
    let stoppedEarly = false;
    const startTime = Date.now();

    for (const action of actionsToReplay) {
      const actionStart = Date.now();
      const result = await this.executeAction(action, actionTimeout);
      result.durationMs = Date.now() - actionStart;

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
   * Execute a single recorded action.
   *
   * CHANGE AXIS: Replay Execution for Action Types
   *
   * Action execution is now handled by the action-executor registry.
   * When adding a new action type:
   * 1. Add to ACTION_TYPES in action-types.ts (normalization, kind mapping)
   * 2. Register executor in action-executor.ts using registerActionExecutor()
   * 3. No changes needed here!
   *
   * @param action - The action to execute
   * @param timeout - Timeout in ms
   * @returns Result of the action execution
   */
  private async executeAction(action: RecordedAction, timeout: number): Promise<ActionReplayResult> {
    const baseResult: ActionReplayResult = {
      actionId: action.id,
      sequenceNum: action.sequenceNum,
      actionType: action.actionType,
      success: false,
      durationMs: 0,
    };

    try {
      // Use the action executor registry
      const executor = getActionExecutor(action.actionType);

      if (!executor) {
        return {
          ...baseResult,
          error: {
            message: `Unsupported action type: ${action.actionType}`,
            code: 'UNKNOWN',
          },
        };
      }

      // Create executor context
      const context: ActionExecutorContext = {
        page: this.page,
        timeout,
        validateSelector: (selector: string) => this.validateSelector(selector),
      };

      return await executor(action, context, baseResult);
    } catch (error) {
      const errorResult = this.handleActionError(error, action, baseResult);
      // Capture screenshot on error
      try {
        const screenshot = await this.page.screenshot({ type: 'png' });
        errorResult.screenshotOnError = screenshot.toString('base64');
      } catch {
        // Ignore screenshot errors
      }
      return errorResult;
    }
  }

  // NOTE: Individual execute* methods have been moved to action-executor.ts
  // This reduces shotgun surgery when adding new action types.
  // See action-executor.ts for the action executor registry pattern.

  /**
   * Handle action execution errors.
   */
  private handleActionError(
    error: unknown,
    action: RecordedAction,
    baseResult: ActionReplayResult
  ): ActionReplayResult {
    const message = error instanceof Error ? error.message : String(error);

    // Categorize the error
    type ErrorCode = 'SELECTOR_NOT_FOUND' | 'SELECTOR_AMBIGUOUS' | 'ELEMENT_NOT_VISIBLE' | 'ELEMENT_NOT_ENABLED' | 'TIMEOUT' | 'NAVIGATION_FAILED' | 'UNKNOWN';
    let code: ErrorCode = 'UNKNOWN';

    if (message.includes('waiting for selector') || message.includes('Timeout')) {
      code = 'TIMEOUT';
    } else if (message.includes('not visible')) {
      code = 'ELEMENT_NOT_VISIBLE';
    } else if (message.includes('disabled')) {
      code = 'ELEMENT_NOT_ENABLED';
    }

    return {
      ...baseResult,
      error: {
        message,
        code,
        selector: action.selector?.primary,
      },
    };
  }

  /**
   * Validate a selector on the current page.
   *
   * @param selector - CSS selector or XPath to validate
   * @returns Validation result
   */
  async validateSelector(selector: string): Promise<SelectorValidation> {
    try {
      // Determine if XPath or CSS
      const isXPath = selector.startsWith('/') || selector.startsWith('(');

      if (isXPath) {
        // Use string-based evaluate to avoid TypeScript issues with browser globals
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
        // CSS selector
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
   * Handle a raw event from the page and convert to RecordedAction.
   */
  private handleRawEvent(raw: RawBrowserEvent): void {
    if (!this.state.isRecording || !this.actionCallback || !this.state.recordingId) {
      return;
    }

    try {
      const action = this.normalizeEvent(raw);
      this.state.actionCount++;

      // Invoke callback (may be async)
      const result = this.actionCallback(action);
      if (result instanceof Promise) {
        result.catch((err: unknown) => {
          const message = err instanceof Error ? err.message : String(err);
          this.handleError(new Error(`Action callback failed: ${message}`));
        });
      }
    } catch (error) {
      this.handleError(error instanceof Error ? error : new Error(String(error)));
    }
  }

  /**
   * Normalize a raw browser event to a RecordedAction.
   *
   * Hardened assumptions:
   * - raw.timestamp may be invalid (NaN, negative, or future date)
   * - raw.selector may be missing or have empty primary
   * - raw.url may be empty or malformed
   */
  private normalizeEvent(raw: RawBrowserEvent): RecordedAction {
    // Use centralized action type normalization (see action-types.ts)
    const actionType = normalizeActionType(raw.actionType);
    const actionKind = toRecordedActionKind(actionType);

    // Hardened: Validate and sanitize timestamp
    let timestamp: string;
    try {
      const date = new Date(raw.timestamp);
      // Check for Invalid Date (NaN check)
      if (Number.isNaN(date.getTime())) {
        timestamp = new Date().toISOString();
      } else {
        timestamp = date.toISOString();
      }
    } catch {
      // Fallback to current time if Date construction fails
      timestamp = new Date().toISOString();
    }

    // Hardened: Ensure selector has valid primary if selector object exists
    let selector = raw.selector;
    if (selector && (!selector.primary || selector.primary.trim() === '')) {
      // If we have candidates but no primary, use the first candidate
      if (selector.candidates && selector.candidates.length > 0) {
        selector = {
          ...selector,
          primary: selector.candidates[0].value,
        };
      }
      // Otherwise selector.primary will be empty, which handlers should check
    }

    const action: RecordedAction = {
      id: uuidv4(),
      sessionId: this.state.sessionId,
      sequenceNum: this.sequenceNum++,
      timestamp,
      actionType,
      actionKind,
      // Use centralized confidence calculation (see action-types.ts)
      confidence: calculateActionConfidence(actionType, selector),
      selector,
      elementMeta: raw.elementMeta,
      boundingBox: raw.boundingBox,
      cursorPos: raw.cursorPos || undefined,
      url: raw.url || '',
      frameId: raw.frameId || undefined,
      payload: raw.payload as RecordedAction['payload'],
      // Use centralized typed payload builder (see action-types.ts)
      typedAction: buildTypedActionPayload(actionKind, raw),
    };

    return action;
  }

  // NOTE: Action type normalization, kind mapping, typed payload building, and confidence
  // calculation have been centralized in action-types.ts to reduce shotgun surgery
  // when adding new action types. See the imports at the top of this file.

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
 *
 * @param page - Playwright Page instance
 * @param sessionId - Session ID
 * @returns RecordModeController instance
 */
export function createRecordModeController(page: Page, sessionId: string): RecordModeController {
  return new RecordModeController(page, sessionId);
}
