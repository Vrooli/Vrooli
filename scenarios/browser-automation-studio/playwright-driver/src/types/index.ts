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

// Re-export typed param extractors for handlers that want to use typed action
export {
  getClickParams,
  getInputParams,
  getNavigateParams,
  getWaitParams,
  getAssertParams,
  getHoverParams,
  getFocusParams,
  getBlurParams,
  getScrollParams,
  getSelectParams,
  getScreenshotParams,
  getEvaluateParams,
  getKeyboardParams,
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
