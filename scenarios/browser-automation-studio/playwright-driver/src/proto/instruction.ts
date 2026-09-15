/**
 * Handler Instruction Types
 *
 * This module defines the handler-friendly wrapper types around proto CompiledInstruction.
 * These types bridge the gap between the proto wire format and what handlers expect.
 *
 * DESIGN:
 *   - HandlerInstruction wraps proto CompiledInstruction for handler consumption
 *   - toHandlerInstruction() converts from proto to handler format
 *   - getActionType() extracts the action type string from instruction
 *
 * WHY THIS EXISTS:
 *   Handlers need a consistent interface that doesn't change with proto evolution.
 *   This wrapper provides stability while the underlying proto types can evolve.
 *
 * @module proto/instruction
 */

import type {
  CompiledInstruction,
  StepTelemetryDirective,
} from '@vrooli/proto-types/browser-automation-studio/v1/execution/driver_pb';
import type { ActionDefinition } from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import { ActionType } from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import { jsonValueMapToPlain } from './utils';
import { actionTypeToString } from './action-type-utils';

// =============================================================================
// HandlerInstruction Type
// =============================================================================

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
  action: ActionDefinition;
  /**
   * Per-step telemetry collection intent from the API.
   * Absent means "use driver defaults", which keeps older API builds working.
   */
  telemetry?: StepTelemetryDirective;
}

// =============================================================================
// Conversion Functions
// =============================================================================

/**
 * Convert a proto CompiledInstruction to a HandlerInstruction.
 *
 * Preserves the typed action field, the sole execution representation.
 *
 * @param proto - Proto CompiledInstruction from API
 * @returns HandlerInstruction with typed action field
 */
export function toHandlerInstruction(proto: CompiledInstruction): HandlerInstruction {
	if (!proto.action || proto.action.type === ActionType.UNSPECIFIED) {
		throw new Error(`Instruction ${proto.nodeId} is missing a typed action`);
	}
	return {
		index: proto.index,
		nodeId: proto.nodeId,
		preloadHtml: proto.preloadHtml,
    context: jsonValueMapToPlain(proto.context),
    metadata: proto.metadata ? { ...proto.metadata } : undefined,
    // Typed action - the canonical representation
    action: proto.action,
    telemetry: proto.telemetry,
  };
}

/**
 * Get the handler dispatch key from the typed action.
 */
export function getActionType(instruction: HandlerInstruction): string {
	if (!instruction.action || instruction.action.type === ActionType.UNSPECIFIED) {
		return 'unknown';
	}

	return actionTypeToString(instruction.action.type);
}
