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

import type { CompiledInstruction } from '@vrooli/proto-types/browser-automation-studio/v1/execution/driver_pb';
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

// =============================================================================
// Conversion Functions
// =============================================================================

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
