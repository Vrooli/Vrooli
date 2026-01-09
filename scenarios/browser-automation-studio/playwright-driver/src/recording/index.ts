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
 * │ FOLDER STRUCTURE:                                                       │
 * │   capture/           - Browser-side script injection                    │
 * │     └─ init-script-generator.ts - Generates recording script            │
 * │     └─ browser-scripts/         - Scripts injected into pages           │
 * │   orchestration/     - Pipeline state and control                       │
 * │     └─ state-machine.ts         - Core state machine                    │
 * │     └─ pipeline-manager.ts      - Main orchestrator                     │
 * │     └─ decisions.ts             - Named decision functions              │
 * │   io/                - Data flow and context setup                      │
 * │     └─ context-initializer.ts   - Context-level setup coordinator       │
 * │     └─ html-injector.ts         - HTML injection route                  │
 * │     └─ event-route.ts           - Page event interception               │
 * │     └─ buffer.ts                - In-memory entry storage               │
 * │   validation/        - Selectors, replay, verification                  │
 * │     └─ selector-service.ts      - Selector validation                   │
 * │     └─ selector-config.ts       - Selector scoring config               │
 * │     └─ replay-service.ts        - Replay preview service                │
 * │     └─ verification.ts          - Script injection verification         │
 * │   testing/           - Diagnostics and self-testing                     │
 * │     └─ self-test.ts             - Automated pipeline testing            │
 * │     └─ diagnostics.ts           - Comprehensive diagnostics             │
 * │                                                                         │
 * │ ROOT FILES (shared utilities):                                          │
 * │   types.ts           - Proto type re-exports and recording types        │
 * │   action-executor.ts - Proto-native action execution                    │
 * │   action-types.ts    - Action confidence calculation                    │
 * │   handler-adapter.ts - Bridge replay to handlers                        │
 * │                                                                         │
 * │ ADDING A NEW ACTION TYPE (e.g., 'drag'):                                │
 * │   1. packages/proto/schemas/.../action.proto - Add to ActionType enum   │
 * │   2. ../proto/action-type-utils.ts - Add string ↔ enum mappings         │
 * │   3. ../handlers/*.ts - Implement handler (preferred)                   │
 * │      OR action-executor.ts - Add executor (if handler not suitable)     │
 * │                                                                         │
 * │ MODIFYING SELECTOR GENERATION:                                          │
 * │   1. validation/selector-config.ts     - Configuration (scores)         │
 * │   2. capture/browser-scripts/*.js      - Browser-side logic             │
 * │                                                                         │
 * └─────────────────────────────────────────────────────────────────────────┘
 *
 * ARCHITECTURE OVERVIEW (PROTO-FIRST + MAIN CONTEXT):
 *
 * Context Setup (once per context)        Recording Sessions (per session)
 * ┌─────────────────────────────────┐     ┌─────────────────────────────────┐
 * │  RecordingContextInitializer    │     │  RecordingPipelineManager       │
 * │  ├─ io/html-injector.ts         │     │  ├─ orchestration/state-machine │
 * │  │   └─ Inject script into HTML │     │  ├─ startRecording()/stop()     │
 * │  └─ io/event-route.ts           │     │  ├─ handleRawEvent()            │
 * │      └─ Page event interception │     │  │   └─ rawBrowserEventTo...()  │
 * └─────────────────────────────────┘     │  └─ verifyPipeline()            │
 *                                         └──────────────┬──────────────────┘
 * Browser Page (MAIN context)                            │
 * ┌─────────────────────────────┐     ┌─────────────────────────────────────┐
 * │ capture/recording-script.js │     │  io/buffer.ts (TimelineEntry store) │
 * │ ├─ Listens for activation   │     └─────────────────────────────────────┘
 * │ ├─ Captures DOM events      │────▶ fetch POST to event route
 * │ ├─ Wraps History API        │
 * │ └─ Generates selectors      │
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
} from './orchestration/state-machine';

export {
  createRecordingStateMachine,
  recordingReducer,
  createInitialState,
  isValidTransition,
  RECOVERABLE_ERRORS,
} from './orchestration/state-machine';

// =============================================================================
// PIPELINE MANAGER (Orchestration Layer)
// =============================================================================

export {
  RecordingPipelineManager,
  createRecordingPipelineManager,
} from './orchestration/pipeline-manager';

export type {
  StartRecordingOptions as PipelineStartOptions,
  StopRecordingResult,
  VerifyPipelineOptions,
  PipelineManagerOptions,
} from './orchestration/pipeline-manager';

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

export * from './validation/selector-config';
export * from './validation/selector-service';

// =============================================================================
// CONTEXT INITIALIZATION (Recording Infrastructure)
// =============================================================================

// Context-level setup for recording (binding + init script)
// The coordinator imports from specialized modules:
//   - io/html-injector.ts - HTML injection into document responses
//   - io/event-route.ts - Page-level event route setup
export {
  RecordingContextInitializer,
  createRecordingContextInitializer,
} from './io/context-initializer';

export type {
  RecordingEventHandler,
  RecordingContextOptions,
  InjectionStats,
  RouteHandlerStats,
  SanityCheckResult,
} from './io/context-initializer';

// Specialized modules (for advanced usage)
export {
  setupHtmlInjectionRoute,
  injectScriptIntoHtml,
  createInjectionStats,
} from './io/html-injector';

export {
  createEventRouteManager,
  createRouteHandlerStats,
} from './io/event-route';

export type { EventRouteManager } from './io/event-route';

// Init script generation for context.addInitScript()
export {
  generateRecordingInitScript,
  generateActivationScript,
  generateDeactivationScript,
  RECORDING_CONTROL_MESSAGE_TYPE,
  DEFAULT_RECORDING_BINDING_NAME,
  clearScriptCache,
} from './capture/init-script-generator';

// =============================================================================
// REPLAY PREVIEW SERVICE
// =============================================================================

export {
  ReplayPreviewService,
  createReplayPreviewService,
} from './validation/replay-service';

export type {
  ReplayPreviewRequest,
  ReplayPreviewResponse,
} from './validation/replay-service';

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
} from './io/buffer';

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
} from './testing/diagnostics';

export type {
  RecordingDiagnosticResult,
  DiagnosticIssue,
  DiagnosticOptions,
  EventFlowTestResult,
  DiagnosticCheck,
} from './testing/diagnostics';

// =============================================================================
// VERIFICATION
// =============================================================================

export {
  verifyScriptInjection,
  assertScriptInjected,
  waitForScriptReady,
} from './validation/verification';

export type {
  InjectionVerification,
} from './validation/verification';

// =============================================================================
// SELF-TEST (Automated Pipeline Testing)
// =============================================================================

export {
  runRecordingPipelineTest,
  runExternalUrlInjectionTest,
  TEST_PAGE_HTML,
  DEFAULT_TEST_URL,
} from './testing/self-test';

export type {
  PipelineTestResult,
  PipelineFailurePoint,
  PipelineStepResult,
  PipelineTestDiagnostics,
  ExternalUrlTestResult,
} from './testing/self-test';
