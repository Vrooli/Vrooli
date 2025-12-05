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
    if (!raw.selector || !raw.selector.candidates || raw.selector.candidates.length === 0) {
      return 0.5;
    }

    // Use the confidence of the primary selector
    const primaryCandidate = raw.selector.candidates.find(
      (c) => c.value === raw.selector.primary
    );

    return primaryCandidate?.confidence ?? 0.5;
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
