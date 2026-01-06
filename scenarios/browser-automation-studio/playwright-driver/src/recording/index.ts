/**
 * Recording Module - Record Mode for Browser Automation
 *
 * PROTO-FIRST ARCHITECTURE:
 * The canonical recording format is proto TimelineEntry.
 * All types flow from proto definitions in packages/proto/.
 *
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ FILE GUIDE - What to read based on what you're trying to do:           │
 * ├─────────────────────────────────────────────────────────────────────────┤
 * │                                                                         │
 * │ UNDERSTANDING THE SYSTEM:                                               │
 * │   state-machine.ts        - Core state machine (single source of truth) │
 * │   pipeline-manager.ts     - Main orchestrator (start/stop, state)       │
 * │   context-initializer.ts  - Context-level setup (binding + init script) │
 * │   init-script-generator.ts- Generates init script for addInitScript()   │
 * │   action-executor.ts      - Proto-native action execution               │
 * │   replay-service.ts       - Replay preview for testing recorded entries │
 * │   selector-service.ts     - Selector validation on pages                │
 * │   ../proto/recording.ts   - Proto conversion (RawBrowserEvent→Timeline) │
 * │                                                                         │
 * │ ADDING A NEW ACTION TYPE (e.g., 'drag'):                                │
 * │   1. packages/proto/schemas/.../action.proto - Add to ActionType enum   │
 * │   2. ../proto/action-type-utils.ts - Add string ↔ enum mappings         │
 * │   3. ../handlers/*.ts - Implement handler (preferred)                   │
 * │      OR action-executor.ts - Add executor (if handler not suitable)     │
 * │                                                                         │
 * │ MODIFYING SELECTOR GENERATION:                                          │
 * │   1. selector-config.ts      - Configuration (scores, patterns)         │
 * │   2. browser-scripts/recording-script.js - Browser-side selector logic  │
 * │   3. selectors.ts            - Documentation only, not executable       │
 * │                                                                         │
 * │ MODIFYING ACTION BUFFERING:                                             │
 * │   buffer.ts        - In-memory TimelineEntry storage                    │
 * │                                                                         │
 * └─────────────────────────────────────────────────────────────────────────┘
 *
 * ARCHITECTURE OVERVIEW (PROTO-FIRST + MAIN CONTEXT):
 *
 * Context Setup (once per context)    Recording Sessions (per session)
 * ┌───────────────────────────────┐   ┌─────────────────────────────────────┐
 * │  RecordingContextInitializer  │   │  RecordingPipelineManager           │
 * │  ├─ context.exposeBinding()   │   │  ├─ state-machine.ts (state)        │
 * │  └─ context.addInitScript()   │   │  ├─ startRecording() / stopRecording│
 * └───────────────────────────────┘   │  ├─ handleRawEvent()                │
 *                                      │  │   └─ rawBrowserEventToTimeline() │
 * Browser Page (MAIN context)         │  └─ verifyPipeline()                │
 * ┌─────────────────────────────┐     └───────────────┬─────────────────────┘
 * │ recording-script.js         │                     │
 * │ ├─ Listens for activation   │     ┌─────────────────────────────────────┐
 * │ ├─ Captures DOM events      │────▶│  buffer.ts (TimelineEntry storage)  │
 * │ ├─ Wraps History API        │     └─────────────────────────────────────┘
 * │ └─ Sends via exposeBinding  │
 * └─────────────────────────────┘
 */

// =============================================================================
// PROTO TYPES (Single Source of Truth)
// =============================================================================

// Timeline types from proto
export type {
  TimelineEntry,
  RawBrowserEvent,
  RawSelectorSet,
  RawSelectorCandidate,
  RawElementMeta,
  ConversionContext,
} from './types';

export {
  rawBrowserEventToTimelineEntry,
  timelineEntryToJson,
  createNavigateTimelineEntry,
  TimelineEntrySchema,
  ActionType,
} from './types';

// Domain types from proto
export type {
  BoundingBox,
  Point,
  SelectorCandidate,
  ElementMeta,
} from './types';
export { SelectorType } from './types';

// =============================================================================
// RECORDING-SPECIFIC TYPES
// =============================================================================

export type { SelectorGeneratorOptions } from './types';
export { DEFAULT_SELECTOR_OPTIONS } from './types';

// =============================================================================
// STATE MACHINE (Single Source of Truth)
// =============================================================================

export type {
  RecordingPipelineState,
  RecordingPipelinePhase,
  PipelineVerification,
  PipelineError,
  PipelineErrorCode,
  RecordingData,
  LoopDetectionState,
  RecordingTransition,
  StateListener,
  RecordingStateMachine,
} from './state-machine';

export {
  createRecordingStateMachine,
  recordingReducer,
  createInitialState,
  isValidTransition,
  RECOVERABLE_ERRORS,
} from './state-machine';

// =============================================================================
// PIPELINE MANAGER (Orchestration Layer)
// =============================================================================

export {
  RecordingPipelineManager,
  createRecordingPipelineManager,
} from './pipeline-manager';

export type {
  StartRecordingOptions as PipelineStartOptions,
  StopRecordingResult,
  VerifyPipelineOptions,
  PipelineManagerOptions,
} from './pipeline-manager';

// =============================================================================
// ACTION EXECUTOR (Proto-Native)
// =============================================================================

export {
  executeTimelineEntry,
  registerTimelineExecutor,
  getTimelineExecutor,
  hasTimelineExecutor,
  getRegisteredActionTypes,
} from './action-executor';

export type {
  TimelineExecutor,
  ExecutorContext,
  ActionReplayResult,
  SelectorValidation,
  ActionErrorCode,
} from './action-executor';

// =============================================================================
// ACTION TYPES UTILITIES
// =============================================================================
// NOTE: Action type utilities have been consolidated in proto/action-type-utils.ts
// Import from '../proto/action-type-utils' for: ACTION_TYPE_MAP, SELECTOR_OPTIONAL_ACTIONS,
// normalizeToProtoActionType, actionTypeToString, stringToActionType, etc.

// Recording-specific function (not in proto module)
export { calculateActionConfidence } from './action-types';

// =============================================================================
// SELECTOR CONFIGURATION & SERVICE
// =============================================================================

export * from './selector-config';
export * from './selector-service';

// =============================================================================
// CONTEXT INITIALIZATION (Recording Infrastructure)
// =============================================================================

// Context-level setup for recording (binding + init script)
export {
  RecordingContextInitializer,
  createRecordingContextInitializer,
} from './context-initializer';

export type {
  RecordingEventHandler,
  RecordingContextOptions,
} from './context-initializer';

// Init script generation for context.addInitScript()
export {
  generateRecordingInitScript,
  generateActivationScript,
  generateDeactivationScript,
  RECORDING_CONTROL_MESSAGE_TYPE,
  DEFAULT_RECORDING_BINDING_NAME,
  clearScriptCache,
} from './init-script-generator';

// =============================================================================
// REPLAY PREVIEW SERVICE
// =============================================================================

export {
  ReplayPreviewService,
  createReplayPreviewService,
} from './replay-service';

export type {
  ReplayPreviewRequest,
  ReplayPreviewResponse,
} from './replay-service';

// =============================================================================
// BUFFER (TimelineEntry Storage)
// =============================================================================

export {
  initRecordingBuffer,
  bufferTimelineEntry,
  getTimelineEntries,
  getTimelineEntryCount,
  clearTimelineEntries,
  removeRecordingBuffer,
  isEntryBuffered,
  getBufferStats,
} from './buffer';

// =============================================================================
// HANDLER ADAPTER (Bridge replay to handlers)
// =============================================================================

export {
  executeViaHandler,
  timelineEntryToHandlerInstruction,
  createReplayHandlerContext,
  hasHandlerForActionType,
  getHandlerSupportedActionTypes,
} from './handler-adapter';

export type {
  ReplayContext,
  HandlerAdapterResult,
} from './handler-adapter';

// =============================================================================
// DIAGNOSTICS
// =============================================================================

export {
  runRecordingDiagnostics,
  isRecordingReady,
  waitForRecordingReady,
  formatDiagnosticReport,
  RecordingDiagnosticLevel,
  DiagnosticSeverity,
  DIAGNOSTIC_CODES,
} from './diagnostics';

export type {
  RecordingDiagnosticResult,
  DiagnosticIssue,
  DiagnosticOptions,
  EventFlowTestResult,
} from './diagnostics';

// =============================================================================
// VERIFICATION
// =============================================================================

export {
  verifyScriptInjection,
  assertScriptInjected,
  waitForScriptReady,
} from './verification';

export type {
  InjectionVerification,
} from './verification';

// =============================================================================
// SELF-TEST (Automated Pipeline Testing)
// =============================================================================

export {
  runRecordingPipelineTest,
  runExternalUrlInjectionTest,
  TEST_PAGE_HTML,
  DEFAULT_TEST_URL,
} from './self-test';

export type {
  PipelineTestResult,
  PipelineFailurePoint,
  PipelineStepResult,
  PipelineTestDiagnostics,
  ExternalUrlTestResult,
} from './self-test';
