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

      // Re-inject on navigation
      this.navigationHandler = (): void => {
        if (this.state.isRecording) {
          // Small delay to ensure page is ready
          setTimeout(() => {
            this.injectRecordingScript().catch((err: unknown) => {
              const message = err instanceof Error ? err.message : String(err);
              this.handleError(new Error(`Failed to re-inject on navigation: ${message}`));
            });
          }, 100);
        }
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
   * Replay recorded actions for preview/testing.
   *
   * Executes actions one by one and returns detailed results for each.
   * This allows users to test their recording before generating a workflow.
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
      switch (action.actionType) {
        case 'click':
          return await this.executeClick(action, timeout, baseResult);

        case 'type':
          return await this.executeType(action, timeout, baseResult);

        case 'navigate':
          return await this.executeNavigate(action, timeout, baseResult);

        case 'scroll':
          return await this.executeScroll(action, timeout, baseResult);

        case 'select':
          return await this.executeSelect(action, timeout, baseResult);

        case 'keypress':
          return await this.executeKeypress(action, timeout, baseResult);

        case 'focus':
          return await this.executeFocus(action, timeout, baseResult);

        case 'hover':
          return await this.executeHover(action, timeout, baseResult);

        case 'blur':
          // Blur doesn't need special handling, just skip
          return { ...baseResult, success: true };

        default:
          return {
            ...baseResult,
            error: {
              message: `Unsupported action type: ${action.actionType}`,
              code: 'UNKNOWN',
            },
          };
      }
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

  /**
   * Execute a click action.
   */
  private async executeClick(
    action: RecordedAction,
    timeout: number,
    baseResult: ActionReplayResult
  ): Promise<ActionReplayResult> {
    if (!action.selector?.primary) {
      return {
        ...baseResult,
        error: { message: 'Click action missing selector', code: 'UNKNOWN' },
      };
    }

    const selector = action.selector.primary;
    const validation = await this.validateSelector(selector);

    if (!validation.valid) {
      return {
        ...baseResult,
        error: {
          message: validation.matchCount === 0
            ? `Element not found: ${selector}`
            : `Multiple elements found (${validation.matchCount}): ${selector}`,
          code: validation.matchCount === 0 ? 'SELECTOR_NOT_FOUND' : 'SELECTOR_AMBIGUOUS',
          matchCount: validation.matchCount,
          selector,
        },
      };
    }

    await this.page.click(selector, { timeout });
    return { ...baseResult, success: true };
  }

  /**
   * Execute a type action.
   */
  private async executeType(
    action: RecordedAction,
    timeout: number,
    baseResult: ActionReplayResult
  ): Promise<ActionReplayResult> {
    if (!action.selector?.primary) {
      return {
        ...baseResult,
        error: { message: 'Type action missing selector', code: 'UNKNOWN' },
      };
    }

    const selector = action.selector.primary;
    const text = action.payload?.text || '';

    const validation = await this.validateSelector(selector);
    if (!validation.valid) {
      return {
        ...baseResult,
        error: {
          message: validation.matchCount === 0
            ? `Element not found: ${selector}`
            : `Multiple elements found (${validation.matchCount}): ${selector}`,
          code: validation.matchCount === 0 ? 'SELECTOR_NOT_FOUND' : 'SELECTOR_AMBIGUOUS',
          matchCount: validation.matchCount,
          selector,
        },
      };
    }

    await this.page.fill(selector, text, { timeout });
    return { ...baseResult, success: true };
  }

  /**
   * Execute a navigate action.
   */
  private async executeNavigate(
    action: RecordedAction,
    timeout: number,
    baseResult: ActionReplayResult
  ): Promise<ActionReplayResult> {
    const url = action.payload?.targetUrl || action.url;

    if (!url) {
      return {
        ...baseResult,
        error: { message: 'Navigate action missing URL', code: 'UNKNOWN' },
      };
    }

    try {
      await this.page.goto(url, { timeout, waitUntil: 'networkidle' });
      return { ...baseResult, success: true };
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      return {
        ...baseResult,
        error: {
          message: `Navigation failed: ${message}`,
          code: 'NAVIGATION_FAILED',
        },
      };
    }
  }

  /**
   * Execute a scroll action.
   */
  private async executeScroll(
    action: RecordedAction,
    _timeout: number,
    baseResult: ActionReplayResult
  ): Promise<ActionReplayResult> {
    const scrollY = action.payload?.scrollY ?? 0;
    const scrollX = action.payload?.scrollX ?? 0;

    if (action.selector?.primary) {
      // Scroll within element
      const selector = action.selector.primary;
      const validation = await this.validateSelector(selector);

      if (!validation.valid) {
        return {
          ...baseResult,
          error: {
            message: validation.matchCount === 0
              ? `Element not found: ${selector}`
              : `Multiple elements found (${validation.matchCount}): ${selector}`,
            code: validation.matchCount === 0 ? 'SELECTOR_NOT_FOUND' : 'SELECTOR_AMBIGUOUS',
            matchCount: validation.matchCount,
            selector,
          },
        };
      }

      // Scroll within element - uses browser context
      await this.page.evaluate(`
        (function() {
          const element = document.querySelector(${JSON.stringify(selector)});
          if (element) {
            element.scrollTo(${scrollX}, ${scrollY});
          }
        })()
      `);
    } else {
      // Scroll window - uses browser context
      await this.page.evaluate(`window.scrollTo(${scrollX}, ${scrollY})`);
    }

    return { ...baseResult, success: true };
  }

  /**
   * Execute a select action.
   */
  private async executeSelect(
    action: RecordedAction,
    timeout: number,
    baseResult: ActionReplayResult
  ): Promise<ActionReplayResult> {
    if (!action.selector?.primary) {
      return {
        ...baseResult,
        error: { message: 'Select action missing selector', code: 'UNKNOWN' },
      };
    }

    const selector = action.selector.primary;
    const value = action.payload?.value || '';

    const validation = await this.validateSelector(selector);
    if (!validation.valid) {
      return {
        ...baseResult,
        error: {
          message: validation.matchCount === 0
            ? `Element not found: ${selector}`
            : `Multiple elements found (${validation.matchCount}): ${selector}`,
          code: validation.matchCount === 0 ? 'SELECTOR_NOT_FOUND' : 'SELECTOR_AMBIGUOUS',
          matchCount: validation.matchCount,
          selector,
        },
      };
    }

    await this.page.selectOption(selector, value, { timeout });
    return { ...baseResult, success: true };
  }

  /**
   * Execute a keypress action.
   */
  private async executeKeypress(
    action: RecordedAction,
    _timeout: number,
    baseResult: ActionReplayResult
  ): Promise<ActionReplayResult> {
    const key = action.payload?.key;

    if (!key) {
      return {
        ...baseResult,
        error: { message: 'Keypress action missing key', code: 'UNKNOWN' },
      };
    }

    await this.page.keyboard.press(key);
    return { ...baseResult, success: true };
  }

  /**
   * Execute a focus action.
   */
  private async executeFocus(
    action: RecordedAction,
    timeout: number,
    baseResult: ActionReplayResult
  ): Promise<ActionReplayResult> {
    if (!action.selector?.primary) {
      return {
        ...baseResult,
        error: { message: 'Focus action missing selector', code: 'UNKNOWN' },
      };
    }

    const selector = action.selector.primary;
    const validation = await this.validateSelector(selector);

    if (!validation.valid) {
      return {
        ...baseResult,
        error: {
          message: validation.matchCount === 0
            ? `Element not found: ${selector}`
            : `Multiple elements found (${validation.matchCount}): ${selector}`,
          code: validation.matchCount === 0 ? 'SELECTOR_NOT_FOUND' : 'SELECTOR_AMBIGUOUS',
          matchCount: validation.matchCount,
          selector,
        },
      };
    }

    await this.page.focus(selector, { timeout });
    return { ...baseResult, success: true };
  }

  /**
   * Execute a hover action.
   */
  private async executeHover(
    action: RecordedAction,
    timeout: number,
    baseResult: ActionReplayResult
  ): Promise<ActionReplayResult> {
    if (!action.selector?.primary) {
      return {
        ...baseResult,
        error: { message: 'Hover action missing selector', code: 'UNKNOWN' },
      };
    }

    const selector = action.selector.primary;
    const validation = await this.validateSelector(selector);

    if (!validation.valid) {
      return {
        ...baseResult,
        error: {
          message: validation.matchCount === 0
            ? `Element not found: ${selector}`
            : `Multiple elements found (${validation.matchCount}): ${selector}`,
          code: validation.matchCount === 0 ? 'SELECTOR_NOT_FOUND' : 'SELECTOR_AMBIGUOUS',
          matchCount: validation.matchCount,
          selector,
        },
      };
    }

    await this.page.hover(selector, { timeout });
    return { ...baseResult, success: true };
  }

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
   */
  private normalizeEvent(raw: RawBrowserEvent): RecordedAction {
    const action: RecordedAction = {
      id: uuidv4(),
      sessionId: this.state.sessionId,
      sequenceNum: this.sequenceNum++,
      timestamp: new Date(raw.timestamp).toISOString(),
      actionType: this.normalizeActionType(raw.actionType),
      confidence: this.calculateActionConfidence(raw),
      selector: raw.selector,
      elementMeta: raw.elementMeta,
      boundingBox: raw.boundingBox,
      cursorPos: raw.cursorPos || undefined,
      url: raw.url,
      frameId: raw.frameId || undefined,
      payload: raw.payload as RecordedAction['payload'],
    };

    return action;
  }

  /**
   * Normalize action type string to valid ActionType.
   */
  private normalizeActionType(type: string): RecordedAction['actionType'] {
    const validTypes = ['click', 'type', 'scroll', 'navigate', 'select', 'hover', 'focus', 'blur', 'keypress'];
    const normalized = type.toLowerCase();

    if (validTypes.includes(normalized)) {
      return normalized as RecordedAction['actionType'];
    }

    // Map common variations
    if (normalized === 'input') return 'type';
    if (normalized === 'change') return 'select';
    if (normalized === 'keydown' || normalized === 'keyup') return 'keypress';

    // Default to click for unknown types
    return 'click';
  }

  /**
   * Calculate confidence score for an action based on selector quality.
   */
  private calculateActionConfidence(raw: RawBrowserEvent): number {
    // Actions like scroll/navigate don't rely on selectors; skip noisy warnings.
    const actionType = this.normalizeActionType(raw.actionType);
    if (actionType === 'scroll' || actionType === 'navigate') {
      return 1;
    }

    if (!raw.selector || !raw.selector.candidates || raw.selector.candidates.length === 0) {
      return 0.5;
    }

    // Use the confidence of the primary selector
    const primaryCandidate = raw.selector.candidates.find(
      (c) => c.value === raw.selector.primary
    );

    if (!primaryCandidate) {
      return 0.5;
    }

    // If we found a stable signal (data-testid/id/aria/data-*), bump to a safe floor
    // to avoid flashing "unstable" warnings on otherwise solid selectors.
    const strongTypes = ['data-testid', 'id', 'aria', 'data-attr'];
    if (strongTypes.includes(primaryCandidate.type) && primaryCandidate.confidence < 0.85) {
      return Math.max(primaryCandidate.confidence, 0.85);
    }

    return primaryCandidate.confidence ?? 0.5;
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
 *
 * @param page - Playwright Page instance
 * @param sessionId - Session ID
 * @returns RecordModeController instance
 */
export function createRecordModeController(page: Page, sessionId: string): RecordModeController {
  return new RecordModeController(page, sessionId);
}
