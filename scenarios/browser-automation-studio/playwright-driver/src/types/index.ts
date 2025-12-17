/**
 * Types Module
 *
 * Central export for type definitions and contracts.
 *
 * TYPE SOURCES:
 * - Proto types: Import from '../proto'
 *   - CompiledInstruction, StepOutcome: Wire format types
 *   - HandlerInstruction: Handler-friendly wrapper with typed action
 *   - ActionDefinition, *Params: Strongly-typed action parameters
 * - Handler types (HandlerResult, HandlerContext): Import from '../handlers/base'
 * - Telemetry types (Screenshot, DOMSnapshot, etc.): Import from '../outcome'
 * - Session types (SessionState, SessionPhase): Defined in './session'
 *
 * STABILITY GUIDE:
 *
 * STABLE CONTRACT (from proto):
 *   - Wire format between Playwright driver and Go API
 *   - Proto types are the single source of truth
 *
 * STABLE CORE (session.ts):
 *   - Session lifecycle types
 *   - Only additive changes expected
 */

// Re-export session types
export * from './session';

// Re-export proto types for convenience
// Note: HandlerInstruction is a handler-friendly wrapper with plain object params
// CompiledInstruction is the actual proto wire type with JsonValue params
export type {
  HandlerInstruction,
  CompiledInstruction,
  BoundingBox,
  Point,
  ConditionOutcome,
  StepFailure,
  HighlightRegion,
  MaskRegion,
  CursorPosition,
  ElementFocus,
  ExecutionPlan,
  PlanGraph,
  PlanStep,
  PlanEdge,
  ActionDefinition,
} from '../proto';

// Re-export the conversion function
export { toHandlerInstruction, getActionType } from '../proto';

// Re-export ALL typed param extractors for handlers
// This provides a convenient import path: import { getClickParams } from '../types'
export {
  // Core interaction params
  getClickParams,
  getInputParams,
  getHoverParams,
  getFocusParams,
  getBlurParams,
  // Navigation & frames
  getNavigateParams,
  getFrameSwitchParams,
  getTabSwitchParams,
  // Wait & assert
  getWaitParams,
  getAssertParams,
  // Data extraction
  getScrollParams,
  getSelectParams,
  getScreenshotParams,
  getEvaluateParams,
  getExtractParams,
  // Keyboard & shortcuts
  getKeyboardParams,
  getShortcutParams,
  // File operations
  getUploadFileParams,
  getDownloadParams,
  // Storage
  getCookieStorageParams,
  // Gestures & drag-drop
  getDragDropParams,
  getGestureParams,
  // Network & device
  getNetworkMockParams,
  getRotateParams,
} from '../proto';

// Re-export proto enums directly (no more string literal types)
export { FailureKind, FailureSource } from '../proto';

// Re-export constants from proto utils
export {
  STEP_OUTCOME_SCHEMA_VERSION,
  PAYLOAD_VERSION,
  EXECUTION_PLAN_SCHEMA_VERSION,
} from '../proto';

// Re-export telemetry types from outcome module
// Note: AssertionOutcome is the handler-friendly type (uses strings, not JsonValue)
export type {
  Screenshot,
  DOMSnapshot,
  ConsoleLogEntry,
  NetworkEvent,
  HandlerAssertionOutcome as AssertionOutcome,
} from '../outcome';
