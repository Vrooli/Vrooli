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

import type { Page, Frame, BrowserContext } from 'rebrowser-playwright';
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
import { initRecordingBuffer, bufferTimelineEntry, getBufferedTimelineEntry, deliverTimelineEntry, flushRecordingDeliveries, assertRecordingAcknowledged } from '../io/buffer';
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
import { safeInvoke } from '../../instrumentation';
import { logger as defaultLogger, LogContext, scopedLog, metrics } from '../../utils';
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
  /** False for pull mode: the consumer must explicitly acknowledge stored entries. */
  acknowledgeOnDelivery?: boolean;
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
  private acknowledgeOnDelivery = true;
  private startPromise: Promise<string> | null = null;
  private stopPromise: Promise<StopRecordingResult> | null = null;
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

  private newPageHandler: ((page: Page) => void) | null = null;
  private loopDetectionInterval: ReturnType<typeof setInterval> | null = null;
  private pageHandlers = new Map<Page, { navigated: (frame: Frame) => void; closed: () => void }>();
  private pendingActivations = new Set<Promise<void>>();

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
    return this.startPromise !== null || (this.entryCallback !== null && !!this.stateMachine.getState().recording);
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
        const verification = this.stateMachine.getVerification();
        if (!verification) {
          throw new Error('Pipeline is ready but verification data is missing');
        }
        return verification;
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

    if (!lastVerification) {
      return {
        scriptLoaded: false,
        scriptReady: false,
        inMainContext: false,
        handlersCount: 0,
        eventRouteActive: false,
        verifiedAt: new Date().toISOString(),
        version: null,
      };
    }
    return lastVerification;
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
  startRecording(options: StartRecordingOptions): Promise<string> {
    if (this.startPromise || this.stopPromise || this.isRecording()) return Promise.reject(new Error('Recording transition is already owned; stop it before starting another'));
    this.startPromise = this.performStartRecording(options).finally(() => { this.startPromise = null; });
    return this.startPromise;
  }

  private async performStartRecording(options: StartRecordingOptions): Promise<string> {
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

    initRecordingBuffer(this.sessionId);
    this.acknowledgeOnDelivery = options.acknowledgeOnDelivery ?? true;
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
      // Starting owns observations until activation and its initial delivery finish.
      this.contextInitializer.setEventHandler((rawEvent: RawBrowserEvent) => this.handleRawEvent(rawEvent));

      // Setup page-level event route
      await this.contextInitializer.setupPageEventRoute(this.page, { force: true });

      // Capture initial navigation
      await this.captureInitialNavigation();

      // Own every current and future document, including dynamic child frames.
      this.newPageHandler = (page) => {
        this.watchPage(page, generation, recordingId);
        for (const frame of page.frames()) this.queueFrameActivation(frame, generation, recordingId);
      };
      this.context.on('page', this.newPageHandler);
      for (const page of this.context.pages()) this.watchPage(page, generation, recordingId);
      await Promise.all(this.context.pages().map(async (page) => {
        await this.contextInitializer.setupPageEventRoute(page, { force: true });
        await this.activateRecordingOnPage(page, recordingId);
      }));

      // Start loop detection
      this.startLoopDetection(generation);

      this.stateMachine.dispatch({ type: 'RECORDING_STARTED', startedAt: new Date().toISOString() });

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
  stopRecording(): Promise<StopRecordingResult> {
    if (this.stopPromise) return this.stopPromise;
    this.stopPromise = (async () => {
      // Join an admitted start before deactivation so it cannot reactivate a
      // frame after the terminal flush. A failed start still owns its entries.
      await this.startPromise?.catch(() => undefined);
      const state = this.stateMachine.getState();
      if (!state.recording || !this.entryCallback) throw new Error(`Cannot stop recording from phase '${state.phase}'`);
      const recordingId = state.recording.recordingId;
      this.stateMachine.dispatch({ type: 'STOP_RECORDING' });
      if (this.newPageHandler) { this.context.off('page', this.newPageHandler); this.newPageHandler = null; }
      this.cleanupPageHandlers();
      await Promise.all(this.pendingActivations);
      this.stopLoopDetection();
      // Keep the consumer installed while current frames flush input and wait
      // for their final accepted observations. Failed stops retain this owner.
      await this.deactivateRecordingOnAllPages();
      await flushRecordingDeliveries(this.sessionId);
      if (this.acknowledgeOnDelivery) assertRecordingAcknowledged(this.sessionId);
      const actionCount = this.stateMachine.getState().recording?.actionCount ?? 0;
      this.contextInitializer.clearEventHandler();
      this.entryCallback = null;
      this.errorCallback = null;
      this.stateMachine.dispatch({ type: 'RECORDING_STOPPED', actionCount });
      return { recordingId, actionCount };
    })().finally(() => { this.stopPromise = null; });
    return this.stopPromise;
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
  async handleRawEvent(raw: RawBrowserEvent): Promise<void> {
    try {
      const state = this.stateMachine.getState();
      if ((state.phase !== 'starting' && state.phase !== 'capturing' && state.phase !== 'stopping') || !state.recording || !this.entryCallback) {
        throw new Error('Recording is not accepting observations');
      }
      if (!raw.id || raw.recordingId !== state.recording.recordingId) throw new Error('Recording observation identity or owner is invalid');
      const previous = getBufferedTimelineEntry(this.sessionId, raw.id);
      const entry = rawBrowserEventToTimelineEntry(raw, {
        sessionId: this.sessionId, sequenceNum: previous?.sequenceNum ?? this.sequenceNum++,
      });
      const inserted = bufferTimelineEntry(this.sessionId, entry);
      if (inserted) {
        if (raw.actionType === 'navigate' && raw.payload?.targetUrl) {
          this.stateMachine.dispatch({ type: 'NAVIGATION', url: raw.payload.targetUrl as string });
        }
        this.stateMachine.dispatch({ type: 'ACTION_CAPTURED', actionType: raw.actionType });
        metrics.recordingActionsTotal.inc();
      }
      const callback = this.entryCallback;
      await deliverTimelineEntry(this.sessionId, entry, async () => { await callback(entry); }, this.acknowledgeOnDelivery);
    } catch (error) {
      await safeInvoke(this.handleError.bind(this), error instanceof Error ? error : new Error(String(error)));
      throw error;
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
    if (this.isRecording()) throw new Error('Stop recording before resetting its pipeline');
    assertRecordingAcknowledged(this.sessionId);
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
    await Promise.all(page.frames().map((frame) => frame.waitForLoadState('domcontentloaded', { timeout: 5000 })));
    await page.evaluate(generateActivationScript(recordingId));
  }

  private async deactivateRecordingOnAllPages(): Promise<void> {
    await Promise.all(this.context.pages().map((page) => page.evaluate(generateDeactivationScript())));
  }

  // ===========================================================================
  // Private: Navigation Handling
  // ===========================================================================

  private watchPage(page: Page, generation: number, recordingId: string): void {
    if (this.pageHandlers.has(page)) return;
    const navigated = (frame: Frame): void => { this.queueFrameActivation(frame, generation, recordingId); };
    const closed = (): void => { this.unwatchPage(page); };
    this.pageHandlers.set(page, { navigated, closed });
    page.on('framenavigated', navigated);
    page.on('close', closed);
  }

  private queueFrameActivation(frame: Frame, generation: number, recordingId: string): void {
    const isCurrent = (): boolean => {
      const state = this.stateMachine.getState();
      return (state.phase === 'starting' || state.phase === 'capturing') && state.recording?.generation === generation;
    };
    if (!isCurrent()) return;
    const operation = (async () => {
      await frame.waitForLoadState('domcontentloaded', { timeout: 5000 });
      if (!isCurrent() || frame.isDetached()) return;
      await this.contextInitializer.setupPageEventRoute(frame.page(), { force: true });
      if (!isCurrent() || frame.isDetached()) return;
      await frame.page().evaluate(generateActivationScript(recordingId));
      if (isCurrent() && frame === this.page.mainFrame()) {
        this.stateMachine.dispatch({ type: 'NAVIGATION', url: frame.url() });
      }
    })().catch(async (error: unknown) => {
      if (!frame.isDetached()) await safeInvoke(this.handleError.bind(this), error instanceof Error ? error : new Error(String(error)));
    }).finally(() => { this.pendingActivations.delete(operation); });
    this.pendingActivations.add(operation);
  }

  private unwatchPage(page: Page): void {
    const handlers = this.pageHandlers.get(page);
    if (!handlers) return;
    page.off('framenavigated', handlers.navigated);
    page.off('close', handlers.closed);
    this.pageHandlers.delete(page);
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
    const url = this.page.url();
    if (!url || url === 'about:blank' || !this.entryCallback) return;
    const entry = createNavigateTimelineEntry(url, { sessionId: this.sessionId, sequenceNum: this.sequenceNum++ });
    bufferTimelineEntry(this.sessionId, entry);
    this.stateMachine.dispatch({ type: 'ACTION_CAPTURED', actionType: 'navigate' });
    this.stateMachine.dispatch({ type: 'NAVIGATION', url });
    const callback = this.entryCallback;
    await deliverTimelineEntry(this.sessionId, entry, async () => { await callback(entry); }, this.acknowledgeOnDelivery);
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

  private cleanupPageHandlers(): void {
    for (const page of this.pageHandlers.keys()) this.unwatchPage(page);
  }

  /**
   * Cleanup resources.
   */
  private cleanup(): void {
    if (this.newPageHandler) {
      this.context.off('page', this.newPageHandler);
      this.newPageHandler = null;
    }

    // Remove all page and frame listeners
    this.cleanupPageHandlers();

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
