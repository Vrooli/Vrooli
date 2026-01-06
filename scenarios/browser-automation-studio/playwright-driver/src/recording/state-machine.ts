/**
 * Recording Pipeline State Machine
 *
 * Single source of truth for all recording-related state.
 * Replaces scattered state across controller, initializer, and session.
 *
 * ARCHITECTURE:
 * - Phases represent discrete states in the recording lifecycle
 * - Transitions are explicit and validated
 * - State is immutable (reducer pattern)
 * - Subscribers notified synchronously on state changes
 *
 * @see pipeline-manager.ts - Orchestrates the state machine with side effects
 */

// =============================================================================
// Phase Definitions
// =============================================================================

/**
 * Recording pipeline phases.
 *
 * State machine diagram:
 * ```
 * uninitialized ──► initializing ──► verifying ──► ready ──► starting ──► recording
 *       ▲                │              │           ▲          │            │
 *       │                ▼              ▼           │          ▼            │
 *       └────────────── error ◄────────────────────┴─────────────────────────┘
 *                         │                                     │
 *                         └─────── (RECOVER) ───────────────────┘
 *                                   (RESET) ───────────────────► uninitialized
 * ```
 */
export type RecordingPipelinePhase =
  | 'uninitialized' // Context not yet set up
  | 'initializing' // Script injection in progress
  | 'verifying' // Checking script loaded correctly
  | 'ready' // Pipeline verified, can start recording
  | 'starting' // Recording being activated
  | 'recording' // Actively recording
  | 'stopping' // Recording being stopped
  | 'error'; // Pipeline error state

// =============================================================================
// Error Codes
// =============================================================================

/**
 * Error codes for pipeline failures.
 * Used to categorize errors and determine recoverability.
 */
export type PipelineErrorCode =
  | 'INJECTION_FAILED' // Script injection failed
  | 'SCRIPT_NOT_LOADED' // Script didn't load on page
  | 'SCRIPT_NOT_READY' // Script loaded but not initialized
  | 'WRONG_CONTEXT' // Script running in ISOLATED instead of MAIN
  | 'NO_HANDLERS' // Script ready but no event handlers registered
  | 'EVENT_ROUTE_FAILED' // Failed to set up event route
  | 'ACTIVATION_FAILED' // Failed to activate recording on page
  | 'REDIRECT_LOOP' // Detected redirect loop
  | 'PAGE_CLOSED' // Page was closed during operation
  | 'VERIFICATION_TIMEOUT' // Verification timed out
  | 'UNKNOWN'; // Unknown error

/**
 * Errors that can be recovered from by retrying.
 */
export const RECOVERABLE_ERRORS: PipelineErrorCode[] = [
  'SCRIPT_NOT_READY',
  'EVENT_ROUTE_FAILED',
  'REDIRECT_LOOP',
  'VERIFICATION_TIMEOUT',
];

// =============================================================================
// State Interfaces
// =============================================================================

/**
 * Pipeline verification result.
 * Captures the state of the recording infrastructure.
 */
export interface PipelineVerification {
  /** Script marker __vrooli_recording_script_loaded is true */
  scriptLoaded: boolean;
  /** Script marker __vrooli_recording_ready is true */
  scriptReady: boolean;
  /** Script is running in MAIN context (not ISOLATED) */
  inMainContext: boolean;
  /** Number of event handlers registered */
  handlersCount: number;
  /** Event route is set up and functional */
  eventRouteActive: boolean;
  /** Timestamp when verification completed */
  verifiedAt: string;
  /** Script version if available */
  version: string | null;
}

/**
 * Error state data.
 */
export interface PipelineError {
  /** Error classification */
  code: PipelineErrorCode;
  /** Human-readable error message */
  message: string;
  /** Whether this error can be recovered from */
  recoverable: boolean;
  /** Timestamp when error occurred */
  occurredAt: string;
  /** Phase when error occurred */
  previousPhase: RecordingPipelinePhase;
}

/**
 * Recording session data (present during 'recording' phase).
 */
export interface RecordingData {
  /** Unique recording identifier */
  recordingId: string;
  /** Session this recording belongs to */
  sessionId: string;
  /** ISO timestamp when recording started */
  startedAt: string;
  /** Generation counter for stale operation detection */
  generation: number;
  /** Number of actions captured */
  actionCount: number;
  /** Last URL seen (for navigation tracking) */
  lastUrl: string | null;
}

/**
 * Loop detection state.
 */
export interface LoopDetectionState {
  /** Recent navigation history for loop detection */
  navigationHistory: Array<{ url: string; timestamp: number }>;
  /** Currently attempting to break a loop */
  isBreakingLoop: boolean;
  /** Last URL checked by loop detector */
  lastCheckedUrl: string | null;
}

/**
 * Full recording pipeline state.
 */
export interface RecordingPipelineState {
  /** Current phase */
  phase: RecordingPipelinePhase;

  /** Initialization metadata (present after INITIALIZE) */
  initialization?: {
    startedAt: number;
    contextId: string;
    injectionAttempts: number;
  };

  /** Verification result (present after successful verification) */
  verification?: PipelineVerification;

  /** Recording data (present during 'starting', 'recording', 'stopping') */
  recording?: RecordingData;

  /** Error data (present during 'error' phase) */
  error?: PipelineError;

  /** Loop detection state (present after initialization) */
  loopDetection?: LoopDetectionState;

  /** Cumulative generation counter (survives stop/start cycles) */
  totalGenerations: number;
}

// =============================================================================
// Transitions
// =============================================================================

/**
 * All possible state transitions.
 */
export type RecordingTransition =
  | { type: 'INITIALIZE'; contextId: string }
  | { type: 'INJECTION_COMPLETE'; success: boolean; error?: string }
  | { type: 'VERIFICATION_COMPLETE'; verification: PipelineVerification }
  | { type: 'START_RECORDING'; sessionId: string; recordingId: string }
  | { type: 'RECORDING_STARTED'; startedAt: string }
  | { type: 'STOP_RECORDING' }
  | { type: 'RECORDING_STOPPED'; actionCount: number }
  | { type: 'ACTION_CAPTURED'; actionType: string }
  | { type: 'NAVIGATION'; url: string }
  | { type: 'REDIRECT_LOOP_DETECTED'; url: string }
  | { type: 'LOOP_BROKEN' }
  | { type: 'ERROR'; code: PipelineErrorCode; message: string }
  | { type: 'RECOVER' }
  | { type: 'RESET' };

// =============================================================================
// Valid Transitions
// =============================================================================

/**
 * Map of valid transitions from each phase.
 */
const VALID_TRANSITIONS: Record<RecordingPipelinePhase, RecordingPipelinePhase[]> = {
  uninitialized: ['initializing'],
  initializing: ['verifying', 'error'],
  verifying: ['ready', 'error'],
  ready: ['starting', 'error', 'uninitialized'],
  starting: ['recording', 'error'],
  recording: ['stopping', 'error'],
  stopping: ['ready', 'error'],
  error: ['ready', 'uninitialized'], // RECOVER or RESET
};

/**
 * Check if a transition from one phase to another is valid.
 */
export function isValidTransition(from: RecordingPipelinePhase, to: RecordingPipelinePhase): boolean {
  return VALID_TRANSITIONS[from].includes(to);
}

// =============================================================================
// Initial State
// =============================================================================

/**
 * Create initial state.
 */
export function createInitialState(): RecordingPipelineState {
  return {
    phase: 'uninitialized',
    totalGenerations: 0,
  };
}

// =============================================================================
// Reducer
// =============================================================================

/**
 * Build verification error message from verification result.
 */
function buildVerificationErrorMessage(v: PipelineVerification): string {
  const issues: string[] = [];
  if (!v.scriptLoaded) issues.push('script not loaded');
  if (!v.scriptReady) issues.push('script not ready');
  if (!v.inMainContext) issues.push('script in ISOLATED context (needs MAIN)');
  if (v.handlersCount === 0) issues.push('no event handlers registered');
  if (!v.eventRouteActive) issues.push('event route not active');
  return `Pipeline verification failed: ${issues.join(', ')}`;
}

/**
 * Determine error code from verification result.
 */
function getVerificationErrorCode(v: PipelineVerification): PipelineErrorCode {
  if (!v.scriptLoaded) return 'SCRIPT_NOT_LOADED';
  if (!v.scriptReady) return 'SCRIPT_NOT_READY';
  if (!v.inMainContext) return 'WRONG_CONTEXT';
  if (v.handlersCount === 0) return 'NO_HANDLERS';
  if (!v.eventRouteActive) return 'EVENT_ROUTE_FAILED';
  return 'UNKNOWN';
}

/**
 * Pure reducer function for state transitions.
 *
 * @param state - Current state
 * @param transition - Transition to apply
 * @returns New state (or same state if transition invalid)
 */
export function recordingReducer(
  state: RecordingPipelineState,
  transition: RecordingTransition
): RecordingPipelineState {
  switch (transition.type) {
    // -------------------------------------------------------------------------
    // Initialization Flow
    // -------------------------------------------------------------------------

    case 'INITIALIZE': {
      if (state.phase !== 'uninitialized') {
        return state; // Invalid transition
      }
      return {
        ...state,
        phase: 'initializing',
        initialization: {
          startedAt: Date.now(),
          contextId: transition.contextId,
          injectionAttempts: 0,
        },
        loopDetection: {
          navigationHistory: [],
          isBreakingLoop: false,
          lastCheckedUrl: null,
        },
      };
    }

    case 'INJECTION_COMPLETE': {
      if (state.phase !== 'initializing') {
        return state;
      }
      if (!transition.success) {
        return {
          ...state,
          phase: 'error',
          error: {
            code: 'INJECTION_FAILED',
            message: transition.error || 'Script injection failed',
            recoverable: true,
            occurredAt: new Date().toISOString(),
            previousPhase: 'initializing',
          },
        };
      }
      return {
        ...state,
        phase: 'verifying',
        initialization: state.initialization
          ? {
              ...state.initialization,
              injectionAttempts: state.initialization.injectionAttempts + 1,
            }
          : undefined,
      };
    }

    case 'VERIFICATION_COMPLETE': {
      if (state.phase !== 'verifying') {
        return state;
      }
      const { verification } = transition;

      // Check if verification passed
      const passed =
        verification.scriptLoaded &&
        verification.scriptReady &&
        verification.inMainContext &&
        verification.handlersCount > 0;

      if (!passed) {
        return {
          ...state,
          phase: 'error',
          verification,
          error: {
            code: getVerificationErrorCode(verification),
            message: buildVerificationErrorMessage(verification),
            recoverable: RECOVERABLE_ERRORS.includes(getVerificationErrorCode(verification)),
            occurredAt: new Date().toISOString(),
            previousPhase: 'verifying',
          },
        };
      }

      return {
        ...state,
        phase: 'ready',
        verification,
        error: undefined, // Clear any previous error
      };
    }

    // -------------------------------------------------------------------------
    // Recording Lifecycle
    // -------------------------------------------------------------------------

    case 'START_RECORDING': {
      if (state.phase !== 'ready') {
        return state; // Verification gate: can only start from 'ready'
      }
      return {
        ...state,
        phase: 'starting',
        recording: {
          recordingId: transition.recordingId,
          sessionId: transition.sessionId,
          startedAt: '',
          generation: state.totalGenerations + 1,
          actionCount: 0,
          lastUrl: null,
        },
        totalGenerations: state.totalGenerations + 1,
      };
    }

    case 'RECORDING_STARTED': {
      if (state.phase !== 'starting' || !state.recording) {
        return state;
      }
      return {
        ...state,
        phase: 'recording',
        recording: {
          ...state.recording,
          startedAt: transition.startedAt,
        },
      };
    }

    case 'STOP_RECORDING': {
      if (state.phase !== 'recording') {
        return state;
      }
      return {
        ...state,
        phase: 'stopping',
      };
    }

    case 'RECORDING_STOPPED': {
      if (state.phase !== 'stopping' || !state.recording) {
        return state;
      }
      // Preserve recording data temporarily for result retrieval
      const finalRecording = {
        ...state.recording,
        actionCount: transition.actionCount,
      };
      return {
        ...state,
        phase: 'ready',
        recording: finalRecording, // Keep for result retrieval, cleared on next start
      };
    }

    // -------------------------------------------------------------------------
    // Event Processing
    // -------------------------------------------------------------------------

    case 'ACTION_CAPTURED': {
      if (state.phase !== 'recording' || !state.recording) {
        return state;
      }
      return {
        ...state,
        recording: {
          ...state.recording,
          actionCount: state.recording.actionCount + 1,
        },
      };
    }

    case 'NAVIGATION': {
      if (!state.recording || !state.loopDetection) {
        return state;
      }
      const now = Date.now();
      const newHistory = [
        ...state.loopDetection.navigationHistory,
        { url: transition.url, timestamp: now },
      ].slice(-50); // Keep last 50 navigations

      return {
        ...state,
        recording: {
          ...state.recording,
          lastUrl: transition.url,
        },
        loopDetection: {
          ...state.loopDetection,
          navigationHistory: newHistory,
          lastCheckedUrl: transition.url,
        },
      };
    }

    // -------------------------------------------------------------------------
    // Loop Detection
    // -------------------------------------------------------------------------

    case 'REDIRECT_LOOP_DETECTED': {
      if (state.phase !== 'recording' || !state.loopDetection) {
        return state;
      }
      return {
        ...state,
        loopDetection: {
          ...state.loopDetection,
          isBreakingLoop: true,
        },
      };
    }

    case 'LOOP_BROKEN': {
      if (!state.loopDetection) {
        return state;
      }
      return {
        ...state,
        loopDetection: {
          navigationHistory: [],
          isBreakingLoop: false,
          lastCheckedUrl: null,
        },
      };
    }

    // -------------------------------------------------------------------------
    // Error Handling
    // -------------------------------------------------------------------------

    case 'ERROR': {
      const recoverable = RECOVERABLE_ERRORS.includes(transition.code);
      return {
        ...state,
        phase: 'error',
        error: {
          code: transition.code,
          message: transition.message,
          recoverable,
          occurredAt: new Date().toISOString(),
          previousPhase: state.phase,
        },
      };
    }

    case 'RECOVER': {
      if (state.phase !== 'error' || !state.error?.recoverable) {
        return state;
      }
      return {
        ...state,
        phase: 'ready',
        error: undefined,
      };
    }

    case 'RESET': {
      return createInitialState();
    }

    default:
      return state;
  }
}

// =============================================================================
// State Machine Factory
// =============================================================================

/**
 * State change listener callback.
 */
export type StateListener = (
  state: RecordingPipelineState,
  transition: RecordingTransition,
  previousState: RecordingPipelineState
) => void;

/**
 * Recording state machine interface.
 */
export interface RecordingStateMachine {
  /** Get current state */
  getState(): RecordingPipelineState;

  /** Dispatch a transition, returns new state */
  dispatch(transition: RecordingTransition): RecordingPipelineState;

  /** Subscribe to state changes, returns unsubscribe function */
  subscribe(listener: StateListener): () => void;

  // Convenience methods
  /** Check if currently recording */
  isRecording(): boolean;
  /** Check if ready to start recording */
  isReady(): boolean;
  /** Check if in error state */
  isError(): boolean;
  /** Get current phase */
  getPhase(): RecordingPipelinePhase;
  /** Get current recording ID (if recording) */
  getRecordingId(): string | undefined;
  /** Get current generation (for stale operation detection) */
  getGeneration(): number;
  /** Get current error (if in error state) */
  getError(): PipelineError | undefined;
  /** Get verification result (if verified) */
  getVerification(): PipelineVerification | undefined;
}

/**
 * Create a new recording state machine.
 *
 * @returns State machine instance
 */
export function createRecordingStateMachine(): RecordingStateMachine {
  let state = createInitialState();
  const listeners = new Set<StateListener>();

  return {
    getState: () => state,

    dispatch: (transition: RecordingTransition): RecordingPipelineState => {
      const previousState = state;
      const newState = recordingReducer(state, transition);

      // Only notify if state actually changed
      if (newState !== previousState) {
        state = newState;
        // Notify listeners synchronously
        listeners.forEach((listener) => {
          try {
            listener(state, transition, previousState);
          } catch (err) {
            // Don't let listener errors break the state machine
            console.error('[StateMachine] Listener error:', err);
          }
        });
      }

      return state;
    },

    subscribe: (listener: StateListener): (() => void) => {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },

    // Convenience methods
    isRecording: () => state.phase === 'recording',
    isReady: () => state.phase === 'ready',
    isError: () => state.phase === 'error',
    getPhase: () => state.phase,
    getRecordingId: () => state.recording?.recordingId,
    getGeneration: () => state.recording?.generation ?? state.totalGenerations,
    getError: () => state.error,
    getVerification: () => state.verification,
  };
}
