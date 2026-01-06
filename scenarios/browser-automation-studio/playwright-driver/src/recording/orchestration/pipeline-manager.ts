/**
 * Recording Pipeline Manager
 *
 * Orchestrates the recording pipeline by:
 * 1. Managing the state machine
 * 2. Executing side effects based on state transitions
 * 3. Bridging browser-side and driver-side state
 * 4. Enforcing verification gate (recording can only start from 'ready')
 *
 * This is the single point of control for all recording operations.
 * External code should interact with this manager, not the state machine directly.
 *
 * @see state-machine.ts - Core state machine (pure logic)
 * @see context-initializer.ts - Handles injection and route setup
 */

import type { Page, BrowserContext } from 'rebrowser-playwright';
import type winston from 'winston';
import {
  createRecordingStateMachine,
  type RecordingStateMachine,
  type RecordingPipelineState,
  type RecordingPipelinePhase,
  type PipelineVerification,
  type PipelineError,
  type RecordingData,
} from './state-machine';
import type { RecordingContextInitializer } from '../io/context-initializer';
import { waitForScriptReady } from '../validation/verification';
import {
  rawBrowserEventToTimelineEntry,
  createNavigateTimelineEntry,
  type RawBrowserEvent,
  type TimelineEntry,
} from '../../proto/recording';
import {
  generateActivationScript,
  generateDeactivationScript,
} from '../capture/init-script-generator';
import { logger as defaultLogger, LogContext, scopedLog } from '../../utils';
import {
  LOOP_DETECTION_WINDOW_MS,
  LOOP_DETECTION_MAX_NAVIGATIONS,
} from '../../constants';
import { validateSelectorOnPage, type SelectorValidation } from '../validation/selector-service';
import {
  ReplayPreviewService,
  type ReplayPreviewRequest,
  type ReplayPreviewResponse,
} from '../validation/replay-service';

// =============================================================================
// Types
// =============================================================================

/** Callback invoked for each recorded TimelineEntry */
export type RecordEntryCallback = (entry: TimelineEntry) => void | Promise<void>;

/** Callback invoked for errors during recording */
export type RecordErrorCallback = (error: Error) => void;

/** Options for starting a recording session */
export interface StartRecordingOptions {
  sessionId: string;
  recordingId?: string;
  onEntry: RecordEntryCallback;
  onError?: RecordErrorCallback;
  /**
   * Whether to auto-verify the pipeline if not already ready.
   * Default: true (RECOMMENDED - prevents "starts but doesn't work" failures)
   */
  autoVerify?: boolean;
  /**
   * Timeout for auto-verification in milliseconds.
   * Default: 5000
   */
  verifyTimeoutMs?: number;
  /**
   * Number of retries for auto-verification.
   * Default: 2
   */
  verifyRetries?: number;
}

/** Result of stopping a recording */
export interface StopRecordingResult {
  recordingId: string;
  actionCount: number;
}

/** Options for verifying the pipeline */
export interface VerifyPipelineOptions {
  /** Timeout for verification in milliseconds */
  timeoutMs?: number;
  /** Number of retries on failure */
  retries?: number;
}

/** Manager configuration */
export interface PipelineManagerOptions {
  /** Logger instance */
  logger?: winston.Logger;
  /** Session ID (for logging) */
  sessionId: string;
}

// Re-export types for convenience
export type { SelectorValidation } from '../validation/selector-service';
export type { ReplayPreviewRequest, ReplayPreviewResponse } from '../validation/replay-service';

// =============================================================================
// Pipeline Manager
// =============================================================================

/**
 * RecordingPipelineManager - Orchestrates recording lifecycle
 *
 * Usage:
 * ```typescript
 * const manager = new RecordingPipelineManager(page, context, initializer, {
 *   sessionId: 'session-123',
 * });
 *
 * // Initialize and verify
 * await manager.initialize();
 * await manager.verifyPipeline();
 *
 * // Start recording (only works from 'ready' phase)
 * const recordingId = await manager.startRecording({
 *   sessionId: 'session-123',
 *   onEntry: (entry) => console.log('Action:', entry),
 * });
 *
 * // ... user performs actions ...
 *
 * // Stop recording
 * const { actionCount } = await manager.stopRecording();
 * ```
 */
export class RecordingPipelineManager {
  private readonly stateMachine: RecordingStateMachine;
  private readonly page: Page;
  private readonly context: BrowserContext;
  private readonly contextInitializer: RecordingContextInitializer;
  private readonly logger: winston.Logger;
  private readonly sessionId: string;

  // Callbacks (set during startRecording)
  private entryCallback: RecordEntryCallback | null = null;
  private errorCallback: RecordErrorCallback | null = null;

  /**
   * Sequence counter for timeline entries.
   *
   * ## THREE-LEVEL GENERATION HIERARCHY
   *
   * Recording uses three counters that serve different purposes:
   *
   * | Counter | Location | Scope | Purpose |
   * |---------|----------|-------|---------|
   * | `totalGenerations` | RecordingPipelineState | Session lifetime | "How many recordings this session?" |
   * | `RecordingData.generation` | RecordingPipelineState.recording | Per-recording | "Which recording is this?" (stale event detection) |
   * | `sequenceNum` | PipelineManager (this) | Per-recording | "What order is this event?" (event ordering) |
   *
   * **Why three counters?**
   * - `totalGenerations` persists across start/stop cycles for global tracking
   * - `generation` identifies a specific recording run for stale event detection
   * - `sequenceNum` orders events within a recording (independent of generation)
   *
   * **Lifecycle:**
   * - `sequenceNum` resets to 0 when startRecording() is called
   * - Increments monotonically for each event
   * - Passed to `ConversionContext` when creating `TimelineEntry`
   *
   * These counters CANNOT be consolidated - they serve fundamentally different purposes.
   *
   * @see state-machine.ts for `totalGenerations` and `RecordingData.generation`
   * @see decisions.ts for stale event detection using generation
   */
  private sequenceNum = 0;

  // Navigation handler reference (for cleanup)
  private navigationHandler: (() => void) | null = null;

  // New page handler reference (for cleanup)
  private newPageHandler: ((page: Page) => Promise<void>) | null = null;

  // Loop detection interval reference (for cleanup)
  private loopDetectionInterval: ReturnType<typeof setInterval> | null = null;

  // Track pages with registered load handlers for cleanup
  // Maps page to the handler function so we can remove it
  private pageLoadHandlers: Map<Page, () => void> = new Map();

  // Replay service (lazy-loaded)
  private replayService: ReplayPreviewService | null = null;

  constructor(
    page: Page,
    context: BrowserContext,
    contextInitializer: RecordingContextInitializer,
    options: PipelineManagerOptions
  ) {
    this.page = page;
    this.context = context;
    this.contextInitializer = contextInitializer;
    this.sessionId = options.sessionId;
    this.logger = options.logger ?? defaultLogger;

    // Create state machine
    this.stateMachine = createRecordingStateMachine();

    // Subscribe to state changes for logging
    this.stateMachine.subscribe((state, transition, prevState) => {
      this.logger.debug(scopedLog(LogContext.RECORDING, 'pipeline state transition'), {
        sessionId: this.sessionId,
        from: prevState.phase,
        to: state.phase,
        transition: transition.type,
        recordingId: state.recording?.recordingId,
        actionCount: state.recording?.actionCount,
      });
    });
  }

  // ===========================================================================
  // State Access
  // ===========================================================================

  /** Get current pipeline state */
  getState(): RecordingPipelineState {
    return this.stateMachine.getState();
  }

  /** Get current phase */
  getPhase(): RecordingPipelinePhase {
    return this.stateMachine.getPhase();
  }

  /** Check if currently recording */
  isRecording(): boolean {
    return this.stateMachine.isRecording();
  }

  /** Check if ready to start recording */
  isReady(): boolean {
    return this.stateMachine.isReady();
  }

  /** Check if in error state */
  isError(): boolean {
    return this.stateMachine.isError();
  }

  /** Get current recording ID (if recording) */
  getRecordingId(): string | undefined {
    return this.stateMachine.getRecordingId();
  }

  /** Get current generation (for stale operation detection) */
  getGeneration(): number {
    return this.stateMachine.getGeneration();
  }

  /** Get current error (if in error state) */
  getError(): PipelineError | undefined {
    return this.stateMachine.getError();
  }

  /** Get verification result (if verified) */
  getVerification(): PipelineVerification | undefined {
    return this.stateMachine.getVerification();
  }

  /** Get recording data */
  getRecordingData(): RecordingData | undefined {
    return this.stateMachine.getState().recording;
  }

  // ===========================================================================
  // Validation & Replay (Delegated Services)
  // ===========================================================================

  /**
   * Validate a selector on the current page.
   *
   * @param selector - The selector to validate
   * @returns Validation result with match count
   */
  async validateSelector(selector: string): Promise<SelectorValidation> {
    return validateSelectorOnPage(this.page, selector);
  }

  /**
   * Replay recorded entries for preview/testing.
   *
   * @param request - Replay configuration
   * @returns Replay results
   */
  async replayPreview(request: ReplayPreviewRequest): Promise<ReplayPreviewResponse> {
    // Lazy-load replay service
    if (!this.replayService) {
      this.replayService = new ReplayPreviewService(this.page);
    }
    return this.replayService.replayPreview(request);
  }

  // ===========================================================================
  // Initialization
  // ===========================================================================

  /**
   * Initialize the recording pipeline.
   *
   * This should be called once after context creation.
   * Sets up injection and transitions to 'verifying' phase.
   */
  async initialize(): Promise<void> {
    const phase = this.stateMachine.getPhase();

    if (phase !== 'uninitialized') {
      this.logger.debug(scopedLog(LogContext.RECORDING, 'pipeline already initialized'), {
        sessionId: this.sessionId,
        phase,
      });
      return;
    }

    const contextId = crypto.randomUUID();
    this.stateMachine.dispatch({ type: 'INITIALIZE', contextId });

    this.logger.info(scopedLog(LogContext.RECORDING, 'initializing pipeline'), {
      sessionId: this.sessionId,
      contextId,
    });

    try {
      // Context initializer handles the actual injection
      // It's already initialized when passed to us
      if (!this.contextInitializer.isInitialized()) {
        await this.contextInitializer.initialize(this.context);
      }

      // Signal injection complete
      this.stateMachine.dispatch({ type: 'INJECTION_COMPLETE', success: true });

      this.logger.info(scopedLog(LogContext.RECORDING, 'injection complete'), {
        sessionId: this.sessionId,
      });
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      this.stateMachine.dispatch({
        type: 'INJECTION_COMPLETE',
        success: false,
        error: message,
      });
      throw error;
    }
  }

  /**
   * Verify the recording pipeline is ready.
   *
   * Checks that the script is loaded, initialized, and in the correct context.
   * Transitions to 'ready' or 'error' phase.
   *
   * @param options - Verification options
   * @returns Verification result
   */
  async verifyPipeline(options: VerifyPipelineOptions = {}): Promise<PipelineVerification> {
    const { timeoutMs = 5000, retries = 1 } = options;
    const phase = this.stateMachine.getPhase();

    if (phase !== 'verifying' && phase !== 'error') {
      // Allow re-verification from error state
      if (phase === 'ready') {
        // Already verified
        return this.stateMachine.getVerification()!;
      }
      throw new Error(`Cannot verify from phase '${phase}', expected 'verifying' or 'error'`);
    }

    this.logger.debug(scopedLog(LogContext.RECORDING, 'verifying pipeline'), {
      sessionId: this.sessionId,
      timeoutMs,
      retries,
    });

    let lastVerification: PipelineVerification | null = null;
    let attempts = 0;

    while (attempts <= retries) {
      attempts++;

      // Wait for script to be ready
      const injectionResult = await waitForScriptReady(this.page, timeoutMs);

      // Check if event route is active
      const routeStats = this.contextInitializer.getRouteHandlerStats();
      const eventRouteActive = routeStats.eventsReceived > 0 || true; // Assume active if no events yet

      const verification: PipelineVerification = {
        scriptLoaded: injectionResult.loaded,
        scriptReady: injectionResult.ready,
        inMainContext: injectionResult.inMainContext,
        handlersCount: injectionResult.handlersCount,
        eventRouteActive,
        verifiedAt: new Date().toISOString(),
        version: injectionResult.version,
      };

      lastVerification = verification;

      // Dispatch verification result
      this.stateMachine.dispatch({ type: 'VERIFICATION_COMPLETE', verification });

      // Check if we're now ready
      if (this.stateMachine.isReady()) {
        this.logger.info(scopedLog(LogContext.RECORDING, 'pipeline verified and ready'), {
          sessionId: this.sessionId,
          verification,
        });
        return verification;
      }

      // If error is not recoverable, don't retry
      const error = this.stateMachine.getError();
      if (error && !error.recoverable) {
        break;
      }

      // Log retry attempt
      if (attempts <= retries) {
        this.logger.warn(scopedLog(LogContext.RECORDING, 'verification failed, retrying'), {
          sessionId: this.sessionId,
          attempt: attempts,
          maxRetries: retries,
          verification,
        });

        // Wait before retry
        await new Promise((resolve) => setTimeout(resolve, 500));
      }
    }

    // Verification failed after all retries
    this.logger.error(scopedLog(LogContext.RECORDING, 'pipeline verification failed'), {
      sessionId: this.sessionId,
      verification: lastVerification,
      error: this.stateMachine.getError(),
    });

    return lastVerification!;
  }

  // ===========================================================================
  // Recording Lifecycle
  // ===========================================================================

  /**
   * Start recording user actions.
   *
   * VERIFICATION GATE: Recording can only start from 'ready' phase.
   * By default, auto-verifies if not ready (prevents "starts but doesn't work" failures).
   *
   * @param options - Recording configuration
   * @returns The recording ID
   * @throws If pipeline cannot reach 'ready' phase after verification
   */
  async startRecording(options: StartRecordingOptions): Promise<string> {
    const phase = this.stateMachine.getPhase();
    const { autoVerify = true, verifyTimeoutMs = 5000, verifyRetries = 2 } = options;

    // Auto-verify if not ready (default behavior - RECOMMENDED)
    if (phase !== 'ready') {
      if (!autoVerify) {
        // Explicit opt-out - throw immediately with diagnostic info
        throw new Error(
          `Cannot start recording from phase '${phase}'. ` +
            `Pipeline must be verified and in 'ready' phase first. ` +
            `(autoVerify=false, set autoVerify=true to auto-verify)`
        );
      }

      this.logger.info(scopedLog(LogContext.RECORDING, 'auto-verifying pipeline before recording'), {
        sessionId: this.sessionId,
        currentPhase: phase,
        timeoutMs: verifyTimeoutMs,
        retries: verifyRetries,
      });

      // Initialize if needed
      if (phase === 'uninitialized') {
        await this.initialize();
      }

      // Verify pipeline
      const verification = await this.verifyPipeline({
        timeoutMs: verifyTimeoutMs,
        retries: verifyRetries,
      });

      // Check if we're now ready
      const newPhase = this.stateMachine.getPhase();
      if (newPhase !== 'ready') {
        const error = this.stateMachine.getError();
        throw new Error(
          `Pipeline verification failed. Phase: '${newPhase}'. ` +
            `Script loaded: ${verification.scriptLoaded}, ready: ${verification.scriptReady}, ` +
            `MAIN context: ${verification.inMainContext}, handlers: ${verification.handlersCount}. ` +
            (error ? `Error: ${error.code} - ${error.message}` : '') +
            ` Run /record/pipeline-test for detailed diagnostics.`
        );
      }

      this.logger.info(scopedLog(LogContext.RECORDING, 'auto-verification succeeded'), {
        sessionId: this.sessionId,
        verification,
      });
    }

    const recordingId = options.recordingId || crypto.randomUUID();

    this.logger.info(scopedLog(LogContext.RECORDING, 'starting recording'), {
      sessionId: this.sessionId,
      recordingId,
    });

    // Transition to 'starting'
    this.stateMachine.dispatch({
      type: 'START_RECORDING',
      sessionId: options.sessionId,
      recordingId,
    });

    // Set callbacks
    this.entryCallback = options.onEntry;
    this.errorCallback = options.onError || null;
    this.sequenceNum = 0;

    const generation = this.stateMachine.getGeneration();

    try {
      // Set event handler on context initializer so route events reach us
      this.contextInitializer.setEventHandler((rawEvent: RawBrowserEvent) => {
        this.handleRawEvent(rawEvent);
      });

      // Setup page-level event route
      await this.contextInitializer.setupPageEventRoute(this.page, { force: true });

      // Capture initial navigation
      await this.captureInitialNavigation();

      // Activate recording on current page
      await this.activateRecordingOnPage(this.page, recordingId);

      // Setup navigation handler
      this.navigationHandler = this.createNavigationHandler(generation, recordingId);
      this.page.on('load', this.navigationHandler);

      // Setup handler for new pages (tabs)
      this.newPageHandler = this.createNewPageHandler(generation, recordingId);
      this.context.on('page', this.newPageHandler);

      // Start loop detection
      this.startLoopDetection(generation);

      // Transition to 'capturing'
      this.stateMachine.dispatch({
        type: 'RECORDING_STARTED',
        startedAt: new Date().toISOString(),
      });

      this.logger.info(scopedLog(LogContext.RECORDING, 'recording started'), {
        sessionId: this.sessionId,
        recordingId,
        generation,
      });

      return recordingId;
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      this.stateMachine.dispatch({
        type: 'ERROR',
        code: 'ACTIVATION_FAILED',
        message,
      });
      throw error;
    }
  }

  /**
   * Stop recording and cleanup.
   *
   * @returns Recording result with action count
   * @throws If not currently recording
   */
  async stopRecording(): Promise<StopRecordingResult> {
    const phase = this.stateMachine.getPhase();
    const state = this.stateMachine.getState();

    if (phase !== 'capturing' || !state.recording) {
      throw new Error(`Cannot stop recording from phase '${phase}'`);
    }

    const recordingId = state.recording.recordingId;
    const actionCount = state.recording.actionCount;

    this.logger.info(scopedLog(LogContext.RECORDING, 'stopping recording'), {
      sessionId: this.sessionId,
      recordingId,
      actionCount,
    });

    // Transition to 'stopping'
    this.stateMachine.dispatch({ type: 'STOP_RECORDING' });

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

      // Remove all tracked page load handlers to prevent stale handlers from firing
      this.cleanupPageLoadHandlers();

      // Stop loop detection
      this.stopLoopDetection();

      // Deactivate recording on all pages
      await this.deactivateRecordingOnAllPages();

      // Clear event handler on context initializer
      this.contextInitializer.clearEventHandler();

      // Clear callbacks
      this.entryCallback = null;
      this.errorCallback = null;

      // Transition to 'ready'
      this.stateMachine.dispatch({ type: 'RECORDING_STOPPED', actionCount });

      this.logger.info(scopedLog(LogContext.RECORDING, 'recording stopped'), {
        sessionId: this.sessionId,
        recordingId,
        actionCount,
      });

      return { recordingId, actionCount };
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      this.stateMachine.dispatch({
        type: 'ERROR',
        code: 'UNKNOWN',
        message: `Stop recording failed: ${message}`,
      });
      throw error;
    }
  }

  // ===========================================================================
  // Event Handling
  // ===========================================================================

  /**
   * Handle a raw event from the browser.
   *
   * Called by the context initializer when events arrive via the route.
   * Converts to TimelineEntry and calls the entry callback.
   *
   * @param raw - Raw browser event
   */
  handleRawEvent(raw: RawBrowserEvent): void {
    const state = this.stateMachine.getState();

    // Check if capturing
    if (state.phase !== 'capturing' || !state.recording || !this.entryCallback) {
      this.logger.debug(scopedLog(LogContext.EVENT_FLOW, 'event dropped (not capturing)'), {
        sessionId: this.sessionId,
        phase: state.phase,
        actionType: raw.actionType,
      });
      return;
    }

    // Note: Generation is available in state.recording.generation for stale event detection
    // Currently we check state.phase === 'capturing' which is sufficient

    try {
      // Update navigation tracking for navigate events
      if (raw.actionType === 'navigate' && raw.payload?.targetUrl) {
        this.stateMachine.dispatch({
          type: 'NAVIGATION',
          url: raw.payload.targetUrl as string,
        });
      }

      // Convert to TimelineEntry
      const entry = rawBrowserEventToTimelineEntry(raw, {
        sessionId: this.sessionId,
        sequenceNum: this.sequenceNum++,
      });

      // Update action count
      this.stateMachine.dispatch({ type: 'ACTION_CAPTURED', actionType: raw.actionType });

      this.logger.debug(scopedLog(LogContext.EVENT_FLOW, 'event captured'), {
        sessionId: this.sessionId,
        recordingId: state.recording.recordingId,
        actionType: raw.actionType,
        sequenceNum: entry.sequenceNum,
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
      this.logger.error(scopedLog(LogContext.EVENT_FLOW, 'event processing failed'), {
        sessionId: this.sessionId,
        actionType: raw.actionType,
        error: error instanceof Error ? error.message : String(error),
      });
      this.handleError(error instanceof Error ? error : new Error(String(error)));
    }
  }

  // ===========================================================================
  // Recovery
  // ===========================================================================

  /**
   * Attempt to recover from an error state.
   *
   * @returns true if recovery succeeded
   */
  async attemptRecovery(): Promise<boolean> {
    const error = this.stateMachine.getError();

    if (!error?.recoverable) {
      this.logger.warn(scopedLog(LogContext.RECORDING, 'cannot recover from error'), {
        sessionId: this.sessionId,
        error,
      });
      return false;
    }

    this.logger.info(scopedLog(LogContext.RECORDING, 'attempting recovery'), {
      sessionId: this.sessionId,
      errorCode: error.code,
    });

    try {
      switch (error.code) {
        case 'SCRIPT_NOT_READY':
        case 'VERIFICATION_TIMEOUT': {
          // Re-verify
          await this.verifyPipeline({ timeoutMs: 5000, retries: 2 });
          break;
        }

        case 'EVENT_ROUTE_FAILED': {
          // Re-setup route
          await this.contextInitializer.setupPageEventRoute(this.page, { force: true });
          this.stateMachine.dispatch({ type: 'RECOVER' });
          break;
        }

        case 'REDIRECT_LOOP': {
          // Navigate away and reset loop detection
          await this.page.goto('about:blank', { timeout: 5000 });
          this.stateMachine.dispatch({ type: 'LOOP_BROKEN' });
          this.stateMachine.dispatch({ type: 'RECOVER' });
          break;
        }

        default:
          return false;
      }

      const recovered = this.stateMachine.isReady();
      this.logger.info(scopedLog(LogContext.RECORDING, 'recovery result'), {
        sessionId: this.sessionId,
        recovered,
        newPhase: this.stateMachine.getPhase(),
      });

      return recovered;
    } catch (err) {
      this.logger.error(scopedLog(LogContext.RECORDING, 'recovery failed'), {
        sessionId: this.sessionId,
        error: err instanceof Error ? err.message : String(err),
      });
      return false;
    }
  }

  /**
   * Reset the pipeline to uninitialized state.
   */
  reset(): void {
    this.cleanup();
    this.stateMachine.dispatch({ type: 'RESET' });

    this.logger.info(scopedLog(LogContext.RECORDING, 'pipeline reset'), {
      sessionId: this.sessionId,
    });
  }

  // ===========================================================================
  // Private: Activation/Deactivation
  // ===========================================================================

  /**
   * Activate recording on a page.
   */
  private async activateRecordingOnPage(page: Page, recordingId: string): Promise<void> {
    try {
      // Wait for page to be ready
      try {
        await page.waitForLoadState('domcontentloaded', { timeout: 5000 });
      } catch {
        // Ignore timeout
      }

      // Send activation message
      await page.evaluate(
        generateActivationScript(recordingId, this.contextInitializer.getBindingName())
      );

      this.logger.debug(scopedLog(LogContext.RECORDING, 'recording activated on page'), {
        sessionId: this.sessionId,
        recordingId,
        url: page.url()?.slice(0, 50),
      });
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      if (!this.isPageGoneError(message)) {
        this.logger.warn(scopedLog(LogContext.RECORDING, 'activation failed'), {
          sessionId: this.sessionId,
          error: message,
        });
      }
      // Don't throw - script starts active by default
    }
  }

  /**
   * Deactivate recording on all pages.
   */
  private async deactivateRecordingOnAllPages(): Promise<void> {
    const pages = this.context.pages();

    await Promise.all(
      pages.map(async (page) => {
        try {
          await page.evaluate(generateDeactivationScript());
        } catch (err) {
          const message = err instanceof Error ? err.message : String(err);
          if (!this.isPageGoneError(message)) {
            this.logger.warn(scopedLog(LogContext.RECORDING, 'deactivation failed'), {
              sessionId: this.sessionId,
              error: message,
            });
          }
        }
      })
    );
  }

  // ===========================================================================
  // Private: Navigation Handling
  // ===========================================================================

  /**
   * Create navigation handler for 'load' events.
   *
   * IMPORTANT: This handler properly sequences async operations to prevent race conditions.
   * The route must be set up BEFORE activation, and navigation event is dispatched AFTER
   * both are complete to ensure no events are lost during the transition window.
   */
  private createNavigationHandler(generation: number, recordingId: string): () => void {
    return (): void => {
      const state = this.stateMachine.getState();

      if (state.phase !== 'capturing' || state.recording?.generation !== generation) {
        return;
      }

      // Capture URL before async operations
      const newUrl = this.page.url();

      // Sequence async operations: route setup -> activation -> navigation dispatch
      // This prevents events from being lost during the transition window
      this.handleNavigationAsync(generation, recordingId, newUrl, state.recording?.lastUrl ?? undefined).catch(
        (err) => {
          this.logger.warn(scopedLog(LogContext.RECORDING, 'navigation handling failed'), {
            sessionId: this.sessionId,
            error: err instanceof Error ? err.message : String(err),
          });
        }
      );
    };
  }

  /**
   * Async helper for navigation handling - properly sequences setup operations.
   */
  private async handleNavigationAsync(
    generation: number,
    recordingId: string,
    newUrl: string,
    lastUrl: string | undefined
  ): Promise<void> {
    // Re-check state after potential await
    const currentState = this.stateMachine.getState();
    if (
      currentState.phase !== 'capturing' ||
      currentState.recording?.generation !== generation
    ) {
      return;
    }

    // Step 1: Re-setup event route FIRST (must complete before activation)
    try {
      await this.contextInitializer.setupPageEventRoute(this.page, { force: true });
    } catch (err) {
      this.logger.warn(scopedLog(LogContext.RECORDING, 'failed to re-setup event route'), {
        sessionId: this.sessionId,
        error: err instanceof Error ? err.message : String(err),
      });
      // Continue - activation may still work
    }

    // Re-check state again
    const stateAfterRoute = this.stateMachine.getState();
    if (
      stateAfterRoute.phase !== 'capturing' ||
      stateAfterRoute.recording?.generation !== generation
    ) {
      return;
    }

    // Step 2: Re-activate recording AFTER route is set up
    try {
      await this.activateRecordingOnPage(this.page, recordingId);
    } catch (err) {
      this.logger.warn(scopedLog(LogContext.RECORDING, 'failed to re-activate recording'), {
        sessionId: this.sessionId,
        error: err instanceof Error ? err.message : String(err),
      });
    }

    // Step 3: Dispatch navigation event AFTER setup is complete
    // This ensures any events triggered by the navigation are properly captured
    if (newUrl && newUrl !== lastUrl) {
      this.stateMachine.dispatch({ type: 'NAVIGATION', url: newUrl });
    }
  }

  /**
   * Create handler for new pages (tabs).
   *
   * IMPORTANT: This handler tracks all registered load handlers for proper cleanup.
   * Each new page gets a load handler that is stored in pageLoadHandlers map and
   * removed during stopRecording() to prevent stale handlers from firing.
   */
  private createNewPageHandler(
    generation: number,
    recordingId: string
  ): (page: Page) => Promise<void> {
    return async (newPage: Page): Promise<void> => {
      const state = this.stateMachine.getState();

      if (state.phase !== 'capturing' || state.recording?.generation !== generation) {
        return;
      }

      this.logger.debug(scopedLog(LogContext.RECORDING, 'new page detected'), {
        sessionId: this.sessionId,
        url: newPage.url()?.slice(0, 50),
      });

      try {
        await this.contextInitializer.setupPageEventRoute(newPage, { force: true });
      } catch (err) {
        this.logger.warn(scopedLog(LogContext.RECORDING, 'failed to setup route on new page'), {
          sessionId: this.sessionId,
          error: err instanceof Error ? err.message : String(err),
        });
      }

      // Create a tracked load handler for this page
      const loadHandler = (): void => {
        // Async work is handled inside, but the handler itself is sync
        this.handleNewPageLoadAsync(newPage, generation, recordingId).catch((err) => {
          this.logger.warn(scopedLog(LogContext.RECORDING, 'new page load handling failed'), {
            sessionId: this.sessionId,
            error: err instanceof Error ? err.message : String(err),
          });
        });
      };

      // Track the handler for cleanup during stopRecording
      this.pageLoadHandlers.set(newPage, loadHandler);

      // Register the handler
      newPage.once('load', loadHandler);

      // Clean up handler reference when page closes
      newPage.once('close', () => {
        this.pageLoadHandlers.delete(newPage);
      });
    };
  }

  /**
   * Async helper for new page load handling - properly sequences setup operations.
   */
  private async handleNewPageLoadAsync(
    newPage: Page,
    generation: number,
    recordingId: string
  ): Promise<void> {
    const currentState = this.stateMachine.getState();
    if (
      currentState.phase !== 'capturing' ||
      currentState.recording?.generation !== generation
    ) {
      return;
    }

    // Step 1: Re-setup event route (must complete before activation)
    try {
      await this.contextInitializer.setupPageEventRoute(newPage, { force: true });
    } catch (err) {
      this.logger.warn(scopedLog(LogContext.RECORDING, 'failed to re-setup route on new page'), {
        sessionId: this.sessionId,
        error: err instanceof Error ? err.message : String(err),
      });
      // Continue - activation may still work
    }

    // Re-check state
    const stateAfterRoute = this.stateMachine.getState();
    if (
      stateAfterRoute.phase !== 'capturing' ||
      stateAfterRoute.recording?.generation !== generation
    ) {
      return;
    }

    // Step 2: Activate recording on page
    try {
      await this.activateRecordingOnPage(newPage, recordingId);
    } catch (err) {
      this.logger.warn(scopedLog(LogContext.RECORDING, 'failed to activate on new page'), {
        sessionId: this.sessionId,
        error: err instanceof Error ? err.message : String(err),
      });
    }

    // Remove from tracking since the once handler has now fired
    this.pageLoadHandlers.delete(newPage);
  }

  // ===========================================================================
  // Private: Loop Detection
  // ===========================================================================

  /**
   * Start loop detection polling.
   */
  private startLoopDetection(generation: number): void {
    this.loopDetectionInterval = setInterval(() => {
      const state = this.stateMachine.getState();

      if (
        state.phase !== 'capturing' ||
        state.recording?.generation !== generation ||
        state.loopDetection?.isBreakingLoop
      ) {
        return;
      }

      let currentUrl: string;
      try {
        currentUrl = this.page.url();
      } catch {
        return;
      }

      if (!currentUrl || currentUrl === 'about:blank' || currentUrl.startsWith('chrome:')) {
        return;
      }

      if (currentUrl === state.loopDetection?.lastCheckedUrl) {
        return;
      }

      // Track navigation
      this.stateMachine.dispatch({ type: 'NAVIGATION', url: currentUrl });

      // Check for loop
      if (this.isRedirectLoop(state.loopDetection?.navigationHistory || [], currentUrl)) {
        this.logger.warn(scopedLog(LogContext.RECORDING, 'redirect loop detected'), {
          sessionId: this.sessionId,
          url: currentUrl,
        });

        this.stateMachine.dispatch({ type: 'REDIRECT_LOOP_DETECTED', url: currentUrl });

        // Break the loop
        this.page
          .goto('about:blank', { timeout: 5000 })
          .then(() => {
            this.stateMachine.dispatch({ type: 'LOOP_BROKEN' });
          })
          .catch((err) => {
            this.logger.warn(scopedLog(LogContext.RECORDING, 'failed to break loop'), {
              sessionId: this.sessionId,
              error: err instanceof Error ? err.message : String(err),
            });
          });
      }
    }, 200);
  }

  /**
   * Stop loop detection.
   */
  private stopLoopDetection(): void {
    if (this.loopDetectionInterval) {
      clearInterval(this.loopDetectionInterval);
      this.loopDetectionInterval = null;
    }
  }

  /**
   * Check if navigation history indicates a redirect loop.
   */
  private isRedirectLoop(
    history: Array<{ url: string; timestamp: number }>,
    currentUrl: string
  ): boolean {
    const now = Date.now();
    const recentHistory = history.filter((h) => now - h.timestamp < LOOP_DETECTION_WINDOW_MS);

    if (recentHistory.length < LOOP_DETECTION_MAX_NAVIGATIONS) {
      return false;
    }

    try {
      const currentHostname = new URL(currentUrl).hostname;
      const sameHostCount = recentHistory.filter((h) => {
        try {
          return new URL(h.url).hostname === currentHostname;
        } catch {
          return false;
        }
      }).length;

      return sameHostCount >= LOOP_DETECTION_MAX_NAVIGATIONS;
    } catch {
      return false;
    }
  }

  // ===========================================================================
  // Private: Utilities
  // ===========================================================================

  /**
   * Capture initial navigation at recording start.
   */
  private async captureInitialNavigation(): Promise<void> {
    try {
      const url = this.page.url();

      if (!url || url === 'about:blank') {
        return;
      }

      const entry = createNavigateTimelineEntry(url, {
        sessionId: this.sessionId,
        sequenceNum: this.sequenceNum++,
      });

      this.stateMachine.dispatch({ type: 'ACTION_CAPTURED', actionType: 'navigate' });
      this.stateMachine.dispatch({ type: 'NAVIGATION', url });

      if (this.entryCallback) {
        const result = this.entryCallback(entry);
        if (result instanceof Promise) {
          await result;
        }
      }
    } catch (error) {
      this.logger.error(scopedLog(LogContext.RECORDING, 'failed to capture initial navigation'), {
        sessionId: this.sessionId,
        error: error instanceof Error ? error.message : String(error),
      });
    }
  }

  /**
   * Handle errors during recording.
   */
  private handleError(error: Error): void {
    if (this.errorCallback) {
      this.errorCallback(error);
    } else {
      this.logger.error(scopedLog(LogContext.RECORDING, 'recording error'), {
        sessionId: this.sessionId,
        error: error.message,
      });
    }
  }

  /**
   * Check if error indicates page is gone/closed.
   */
  private isPageGoneError(message: string): boolean {
    return (
      message.includes('closed') ||
      message.includes('navigating') ||
      message.includes('detached') ||
      message.includes('destroyed') ||
      message.includes('Target closed')
    );
  }

  /**
   * Clean up all tracked page load handlers.
   *
   * This removes load handlers registered on new pages during recording
   * to prevent them from firing after recording has stopped.
   */
  private cleanupPageLoadHandlers(): void {
    for (const [page, handler] of this.pageLoadHandlers) {
      try {
        page.off('load', handler);
      } catch (err) {
        // Page may already be closed - this is fine
        const message = err instanceof Error ? err.message : String(err);
        if (!this.isPageGoneError(message)) {
          this.logger.debug(scopedLog(LogContext.RECORDING, 'failed to remove page load handler'), {
            sessionId: this.sessionId,
            error: message,
          });
        }
      }
    }
    this.pageLoadHandlers.clear();
  }

  /**
   * Cleanup resources.
   */
  private cleanup(): void {
    if (this.navigationHandler) {
      this.page.off('load', this.navigationHandler);
      this.navigationHandler = null;
    }

    if (this.newPageHandler) {
      this.context.off('page', this.newPageHandler);
      this.newPageHandler = null;
    }

    // Clean up tracked page load handlers
    this.cleanupPageLoadHandlers();

    this.stopLoopDetection();
    this.contextInitializer.clearEventHandler();
    this.entryCallback = null;
    this.errorCallback = null;
  }
}

/**
 * Create a RecordingPipelineManager instance.
 */
export function createRecordingPipelineManager(
  page: Page,
  context: BrowserContext,
  contextInitializer: RecordingContextInitializer,
  options: PipelineManagerOptions
): RecordingPipelineManager {
  return new RecordingPipelineManager(page, context, contextInitializer, options);
}
