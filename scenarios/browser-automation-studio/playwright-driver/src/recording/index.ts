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
 * │   controller.ts          - Main orchestrator (start/stop, replay)       │
 * │   action-executor.ts     - Proto-native action execution                │
 * │   ../proto/recording.ts  - Proto conversion (RawBrowserEvent→Timeline)  │
 * │                                                                         │
 * │ ADDING A NEW ACTION TYPE (e.g., 'drag'):                                │
 * │   1. packages/proto/schemas/.../action.proto - Add to ActionType enum   │
 * │   2. ../proto/action-type-utils.ts - Add string ↔ enum mappings         │
 * │   3. ../handlers/*.ts - Implement handler (preferred)                   │
 * │      OR action-executor.ts - Add executor (if handler not suitable)     │
 * │                                                                         │
 * │ MODIFYING SELECTOR GENERATION:                                          │
 * │   1. selector-config.ts - Configuration (scores, patterns) - EDIT THIS  │
 * │   2. injector.ts        - Runtime code injected into browser - EDIT THIS│
 * │   3. selectors.ts       - Documentation only, not executable            │
 * │                                                                         │
 * │ MODIFYING ACTION BUFFERING:                                             │
 * │   buffer.ts        - In-memory TimelineEntry storage                    │
 * │                                                                         │
 * └─────────────────────────────────────────────────────────────────────────┘
 *
 * ARCHITECTURE OVERVIEW (PROTO-FIRST):
 *
 * Browser Page                    Node.js Driver
 * ┌─────────────┐                ┌─────────────────────────────────────┐
 * │  injector   │  ─────────────▶│  RecordModeController               │
 * │  (JS code)  │  page.expose   │  ├─ handleRawEvent()                │
 * │             │  Function()    │  │   └─ rawBrowserEventToTimeline() │
 * └─────────────┘                │  ├─ replayPreview()                 │
 *       │                        │  │   └─ executeTimelineEntry()      │
 *       │ Captures events        │  └─ validateSelector                │
 *       ▼                        └───────────────┬─────────────────────┘
 * ┌─────────────┐                                │
 * │ DOM Events  │                ┌─────────────────────────────────────┐
 * │ click, type │                │  buffer.ts (TimelineEntry storage)  │
 * └─────────────┘                └─────────────────────────────────────┘
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

export type {
  RecordingState,
  SelectorGeneratorOptions,
} from './types';

export { DEFAULT_SELECTOR_OPTIONS } from './types';

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

// NOTE: selectors.reference.ts is NOT exported - it's documentation-only reference code.
// The actual runtime implementation is in injector.ts (stringified JavaScript).
// See selectors.reference.ts file header for details on why this separation exists.

// =============================================================================
// BROWSER SCRIPT INJECTION
// =============================================================================

export * from './injector';

// =============================================================================
// CONTROLLER
// =============================================================================

export {
  RecordModeController,
  createRecordModeController,
} from './controller';

export type {
  RecordEntryCallback,
  StartRecordingOptions,
  ReplayPreviewRequest,
  ReplayPreviewResponse,
} from './controller';

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
