/**
 * Proto Types - Central Export Hub
 *
 * This module provides a clean interface to proto-generated types for the
 * browser-automation-studio domain. All proto types used by the driver
 * should be imported from here.
 *
 * DESIGN:
 *   - Re-exports proto types from @vrooli/proto-types
 *   - Provides utility functions for JSON serialization/deserialization
 *   - Maintains compatibility with existing snake_case wire format
 *
 * WHY THIS EXISTS:
 *   Proto types are the single source of truth for the API contract between
 *   the Go engine and this TypeScript driver. Using proto types ensures:
 *   1. Type safety across language boundaries
 *   2. Automatic validation of incoming data
 *   3. Consistent serialization format
 *   4. No manual type definitions to maintain
 */

// =============================================================================
// RE-EXPORTS FROM PROTO PACKAGE
// =============================================================================

// Driver contracts (engine <-> driver)
export {
  type StepOutcome,
  type CompiledInstruction,
  type ExecutionPlan,
  type StepFailure,
  type DriverScreenshot,
  type DOMSnapshot,
  type DriverConsoleLogEntry,
  type DriverNetworkEvent,
  type CursorPosition,
  type AssertionOutcome,
  type ConditionOutcome,
  type PlanStep,
  type PlanEdge,
  type PlanGraph,
  StepOutcomeSchema,
  CompiledInstructionSchema,
  ExecutionPlanSchema,
  StepFailureSchema,
  DriverScreenshotSchema,
  DOMSnapshotSchema,
  DriverConsoleLogEntrySchema,
  DriverNetworkEventSchema,
  CursorPositionSchema,
  AssertionOutcomeSchema,
  ConditionOutcomeSchema,
  PlanStepSchema,
  PlanEdgeSchema,
  PlanGraphSchema,
  FailureKind,
  FailureSource,
} from '@vrooli/proto-types/browser-automation-studio/v1/execution/driver_pb';

// Timeline (unified recording/execution format)
export {
  type TimelineEntry,
  type TimelineEntryAggregates,
  type TimelineArtifact,
  type ElementFocus,
  type TimelineLog,
  type TimelineStreamMessage,
  type TimelineStatusUpdate,
  type TimelineHeartbeat,
  TimelineEntrySchema,
  TimelineEntryAggregatesSchema,
  TimelineArtifactSchema,
  ElementFocusSchema,
  TimelineLogSchema,
  TimelineStreamMessageSchema,
  TimelineStatusUpdateSchema,
  TimelineHeartbeatSchema,
  TimelineMessageType,
} from '@vrooli/proto-types/browser-automation-studio/v1/timeline/entry_pb';

// Actions (action definitions and params)
export {
  type ActionDefinition,
  type ActionMetadata,
  type NavigateParams,
  type ClickParams,
  type InputParams,
  type WaitParams,
  type AssertParams,
  type ScrollParams,
  type SelectParams,
  type EvaluateParams,
  type KeyboardParams,
  type HoverParams,
  type ScreenshotParams,
  type FocusParams,
  type BlurParams,
  type SubflowParams,
  // New action params
  type ExtractParams,
  type UploadFileParams,
  type DownloadParams,
  type FrameSwitchParams,
  type TabSwitchParams,
  type CookieStorageParams,
  type CookieOptions,
  type ShortcutParams,
  type DragDropParams,
  type GestureParams,
  type NetworkMockParams,
  type RotateParams,
  // Schemas
  ActionDefinitionSchema,
  ActionMetadataSchema,
  NavigateParamsSchema,
  ClickParamsSchema,
  InputParamsSchema,
  WaitParamsSchema,
  AssertParamsSchema,
  ScrollParamsSchema,
  SelectParamsSchema,
  EvaluateParamsSchema,
  KeyboardParamsSchema,
  HoverParamsSchema,
  ScreenshotParamsSchema,
  FocusParamsSchema,
  BlurParamsSchema,
  SubflowParamsSchema,
  // New action param schemas
  ExtractParamsSchema,
  UploadFileParamsSchema,
  DownloadParamsSchema,
  FrameSwitchParamsSchema,
  TabSwitchParamsSchema,
  CookieStorageParamsSchema,
  CookieOptionsSchema,
  ShortcutParamsSchema,
  DragDropParamsSchema,
  GestureParamsSchema,
  NetworkMockParamsSchema,
  RotateParamsSchema,
  // Enums
  ActionType,
  MouseButton,
  NavigateWaitEvent,
  NavigateDestinationType,
  WaitState,
  ScrollBehavior,
  KeyAction,
  KeyboardModifier,
  // New enums
  ExtractType,
  FrameSwitchAction,
  TabSwitchAction,
  CookieOperation,
  StorageType,
  GestureType,
  SwipeDirection,
  NetworkMockOperation,
  DeviceOrientation,
  CookieSameSite,
} from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';

// Base types (shared enums and primitives)
export {
  type BoundingBox,
  type Point,
  type NodePosition,
  BoundingBoxSchema,
  PointSchema,
  NodePositionSchema,
} from '@vrooli/proto-types/browser-automation-studio/v1/base/geometry_pb';

export {
  type RetryStatus,
  type RetryAttempt,
  type AssertionResult,
  type EventContext,
  RetryStatusSchema,
  RetryAttemptSchema,
  AssertionResultSchema,
  EventContextSchema,
  ExecutionStatus,
  StepStatus,
  LogLevel,
  TriggerType,
  ArtifactType,
  SelectorType,
  NetworkEventType,
  RecordingSource,
  AssertionMode,
  HighlightColor,
} from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';

// Domain types (selectors, telemetry)
export {
  type SelectorCandidate,
  type ElementMeta,
  type HighlightRegion,
  type MaskRegion,
  SelectorCandidateSchema,
  ElementMetaSchema,
  HighlightRegionSchema,
  MaskRegionSchema,
} from '@vrooli/proto-types/browser-automation-studio/v1/domain/selectors_pb';

export {
  type ActionTelemetry,
  type TimelineScreenshot,
  type ConsoleLogEntry,
  type NetworkEvent,
  ActionTelemetrySchema,
  TimelineScreenshotSchema,
  ConsoleLogEntrySchema,
  NetworkEventSchema,
} from '@vrooli/proto-types/browser-automation-studio/v1/domain/telemetry_pb';

// Common types (JsonValue for dynamic data)
export {
  type JsonValue,
  type JsonObject,
  type JsonList,
  JsonValueSchema,
  JsonObjectSchema,
  JsonListSchema,
} from '@vrooli/proto-types/common/v1/types_pb';

// =============================================================================
// UTILITY EXPORTS
// =============================================================================

export * from './utils';
export * from './recording';

// =============================================================================
// ACTION TYPE UTILITIES (Single Source of Truth)
// =============================================================================
// Consolidated from previously duplicated implementations in:
// - proto/index.ts, recording/action-types.ts, recording/handler-adapter.ts

export {
  // Core conversion functions
  actionTypeToString,
  actionTypeToDisplayString,
  stringToActionType,
  normalizeToProtoActionType,
  // Utility functions
  isValidActionType,
  isSelectorOptional,
  getSupportedActionTypes,
  getRegisteredTypeStrings,
  // Data exports
  ACTION_TYPE_MAP,
  SELECTOR_OPTIONAL_ACTIONS,
} from './action-type-utils';

// NOTE: compat.ts was removed - it contained unused legacy type definitions.
// Use proto types directly from the exports above.

// =============================================================================
// HANDLER-FRIENDLY TYPES
// =============================================================================

import type { CompiledInstruction } from '@vrooli/proto-types/browser-automation-studio/v1/execution/driver_pb';
import type { ActionDefinition } from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import { ActionType } from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import { jsonValueMapToPlain } from './utils';
import { actionTypeToString } from './action-type-utils';

/**
 * HandlerInstruction is a handler-friendly wrapper around proto CompiledInstruction.
 *
 * The `action` field contains the typed ActionDefinition with strongly-typed params.
 * Handlers use `requireTypedParams()` to extract and validate params from action.
 *
 * @example
 * ```typescript
 * const typedParams = instruction.action ? getClickParams(instruction.action) : undefined;
 * const params = this.requireTypedParams(typedParams, 'click', instruction.nodeId);
 * ```
 */
export interface HandlerInstruction {
  /** Zero-based index in execution order */
  index: number;
  /** Node ID from the workflow definition (UUID) */
  nodeId: string;
  /**
   * @deprecated Legacy field - no longer populated by Go API.
   * Use `getActionType(instruction)` to get the action type string.
   */
  type: string;
  /**
   * @deprecated Legacy field - no longer populated by Go API.
   * Use typed param extractors like `getClickParams(instruction.action)` instead.
   * This field is always an empty object.
   */
  params: Record<string, unknown>;
  /** Optional preload HTML */
  preloadHtml?: string;
  /** Optional context data */
  context?: Record<string, unknown>;
  /** Optional metadata */
  metadata?: Record<string, string>;
  /**
   * Typed action definition from proto.
   * Contains the ActionType enum and strongly-typed params (navigate, click, etc.)
   * Always populated by the Go API - handlers should use requireTypedParams() to extract.
   */
  action?: ActionDefinition;
}

/**
 * Convert a proto CompiledInstruction to a HandlerInstruction.
 *
 * Preserves the typed action field which is the canonical representation.
 * Legacy type/params fields are set to empty values - they are no longer
 * populated by the Go API and should not be used.
 *
 * @param proto - Proto CompiledInstruction from API
 * @returns HandlerInstruction with typed action field
 */
export function toHandlerInstruction(proto: CompiledInstruction): HandlerInstruction {
  return {
    index: proto.index,
    nodeId: proto.nodeId,
    // Legacy fields - no longer populated, kept for interface compatibility
    type: '',
    params: {},
    preloadHtml: proto.preloadHtml,
    context: jsonValueMapToPlain(proto.context),
    metadata: proto.metadata ? { ...proto.metadata } : undefined,
    // Typed action - the canonical representation
    action: proto.action,
  };
}

/**
 * Get the action type string from a HandlerInstruction.
 * Prefers the typed action when present, falls back to legacy type string.
 */
export function getActionType(instruction: HandlerInstruction): string {
  if (instruction.action?.type !== undefined && instruction.action.type !== ActionType.UNSPECIFIED) {
    return actionTypeToString(instruction.action.type);
  }
  return instruction.type;
}

// NOTE: actionTypeToString is now imported from ./action-type-utils (single source of truth)

// =============================================================================
// PARAM EXTRACTORS (re-exported from ./params.ts)
// =============================================================================
// All param extractors have been moved to ./params.ts to reduce file size
// and improve maintainability. They are re-exported here for backward compatibility.

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
  getExtractParams,
  getUploadFileParams,
  getDownloadParams,
  getFrameSwitchParams,
  getTabSwitchParams,
  getCookieStorageParams,
  getShortcutParams,
  getDragDropParams,
  getGestureParams,
  getNetworkMockParams,
  getRotateParams,
} from './params';
