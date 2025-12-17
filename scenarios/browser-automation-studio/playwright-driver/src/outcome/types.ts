/**
 * Unified Outcome Types
 *
 * Shared types for execution results across the codebase.
 * These types provide a common vocabulary for "did this work?" across:
 * - Handler execution (HandlerResult)
 * - Recording replay (ActionReplayResult)
 * - StepOutcome building
 *
 * TYPE HIERARCHY
 * ==============
 *
 * ┌─────────────────────────────────────────────────────────────────────────────┐
 * │                         BaseExecutionResult                                  │
 * │  (success: boolean, error?: ExecutionError, durationMs?: number)            │
 * └─────────────────────────────────────────────────────────────────────────────┘
 *          │                          │                          │
 *          ▼                          ▼                          ▼
 * ┌─────────────────┐      ┌─────────────────────┐      ┌──────────────────────┐
 * │  HandlerResult  │      │  ActionReplayResult │      │ HandlerAdapterResult │
 * │  (outcome-      │      │  (action-executor)  │      │ (handler-adapter)    │
 * │   builder.ts)   │      │                     │      │                      │
 * │                 │      │ + entryId           │      │ (minimal: just       │
 * │ + screenshot    │      │ + sequenceNum       │      │  success/error/      │
 * │ + domSnapshot   │      │ + actionType        │      │  duration)           │
 * │ + consoleLogs   │      │ + screenshotOnError │      │                      │
 * │ + networkEvents │      │                     │      │                      │
 * │ + extracted_data│      │ Uses SelectorError  │      │                      │
 * │ + focus         │      │ for error field     │      │                      │
 * │ + assertion     │      │                     │      │                      │
 * └─────────────────┘      └─────────────────────┘      └──────────────────────┘
 *          │
 *          ▼
 * ┌─────────────────┐
 * │   StepOutcome   │
 * │  (proto type)   │
 * │                 │
 * │ Built by        │
 * │ buildStepOutcome│
 * │ from Handler-   │
 * │ Result          │
 * └─────────────────┘
 *
 * ERROR TYPE HIERARCHY
 * ====================
 *
 * ┌─────────────────────────────────────────────────────────────────────────────┐
 * │                          ExecutionError                                      │
 * │  (message, code, kind?, retryable?)                                         │
 * └─────────────────────────────────────────────────────────────────────────────┘
 *          │
 *          ▼
 * ┌─────────────────────────────────────────────────────────────────────────────┐
 * │                          SelectorError                                       │
 * │  extends ExecutionError + matchCount?, selector?                            │
 * └─────────────────────────────────────────────────────────────────────────────┘
 *
 * WHY MULTIPLE RESULT TYPES?
 * ==========================
 *
 * We use composition rather than forcing a single result type because:
 *
 * 1. Different contexts need different additional fields:
 *    - Handlers need telemetry (screenshots, DOM snapshots, console logs)
 *    - Replay needs entry metadata (entryId, sequenceNum, actionType)
 *    - Adapters need just the basics (minimal overhead)
 *
 * 2. Forcing one type adds unnecessary complexity:
 *    - ActionReplayResult doesn't need screenshot/domSnapshot fields
 *    - HandlerAdapterResult doesn't need entryId fields
 *
 * 3. Each variant serves a specific purpose:
 *    - HandlerResult → StepOutcome (wire format for Go API)
 *    - ActionReplayResult → ReplayPreviewResponse (UI feedback)
 *    - HandlerAdapterResult → ActionReplayResult (bridge layer)
 *
 * DESIGN PHILOSOPHY:
 * Rather than forcing all contexts to use a single result type (which would
 * add unnecessary complexity), we define shared building blocks:
 * - ExecutionError: Common error structure
 * - ExecutionMetrics: Common timing/metrics structure
 *
 * Each context (handlers, replay, telemetry) then composes these into
 * their domain-specific result types.
 *
 * CHANGE AXIS: Error Codes
 * When adding a new error condition:
 * 1. Add to the appropriate error code union (HandlerErrorCode or ActionErrorCode)
 * 2. Map to FailureKind in the toFailureKind function
 * 3. Update RETRYABLE_ERRORS if the new error is retryable
 */

import { FailureKind } from '@vrooli/proto-types/browser-automation-studio/v1/execution/driver_pb';

// =============================================================================
// ERROR CODES
// =============================================================================

/**
 * Handler-level error codes.
 * Used by instruction handlers to classify failures.
 */
export type HandlerErrorCode =
  | 'MISSING_PARAM'        // Required parameter not provided
  | 'INVALID_PARAM'        // Parameter value is invalid
  | 'SELECTOR_NOT_FOUND'   // Element not found
  | 'SELECTOR_AMBIGUOUS'   // Multiple elements match
  | 'ELEMENT_NOT_VISIBLE'  // Element exists but not visible
  | 'ELEMENT_NOT_ENABLED'  // Element is disabled
  | 'TIMEOUT'              // Operation timed out
  | 'NAVIGATION_FAILED'    // Page navigation failed
  | 'NETWORK_ERROR'        // Network request failed
  | 'ASSERTION_FAILED'     // Assertion check failed
  | 'SCRIPT_ERROR'         // JavaScript evaluation error
  | 'UNKNOWN';             // Catch-all for unexpected errors

/**
 * Action replay error codes.
 * Used by the recording replay system to classify failures.
 * Subset of HandlerErrorCode with replay-specific additions.
 */
export type ActionErrorCode =
  | 'SELECTOR_NOT_FOUND'
  | 'SELECTOR_AMBIGUOUS'
  | 'ELEMENT_NOT_VISIBLE'
  | 'ELEMENT_NOT_ENABLED'
  | 'TIMEOUT'
  | 'NAVIGATION_FAILED'
  | 'UNSUPPORTED_ACTION'
  | 'MISSING_PARAMS'
  | 'UNKNOWN';

/**
 * Map from error code to FailureKind for StepOutcome construction.
 */
export const ERROR_CODE_TO_FAILURE_KIND: Record<string, FailureKind> = {
  // User errors (bad input/selector)
  MISSING_PARAM: FailureKind.USER,
  INVALID_PARAM: FailureKind.USER,
  SELECTOR_NOT_FOUND: FailureKind.ENGINE,
  SELECTOR_AMBIGUOUS: FailureKind.ENGINE,
  ELEMENT_NOT_VISIBLE: FailureKind.ENGINE,
  ELEMENT_NOT_ENABLED: FailureKind.ENGINE,
  ASSERTION_FAILED: FailureKind.ENGINE,
  UNSUPPORTED_ACTION: FailureKind.ORCHESTRATION,
  MISSING_PARAMS: FailureKind.USER,

  // Infrastructure errors
  TIMEOUT: FailureKind.TIMEOUT,
  NAVIGATION_FAILED: FailureKind.INFRA,
  NETWORK_ERROR: FailureKind.INFRA,
  SCRIPT_ERROR: FailureKind.ENGINE,

  // Catch-all
  UNKNOWN: FailureKind.ENGINE,
};

/**
 * Error codes that indicate the operation may succeed on retry.
 */
export const RETRYABLE_ERROR_CODES: ReadonlySet<string> = new Set([
  'TIMEOUT',
  'NETWORK_ERROR',
  'ELEMENT_NOT_VISIBLE', // Element may become visible
  'NAVIGATION_FAILED',   // Network glitch
]);

// =============================================================================
// SHARED ERROR INTERFACE
// =============================================================================

/**
 * Base execution error interface.
 * Provides a common structure for error reporting across the codebase.
 *
 * This is the CANONICAL error type. All domain-specific error types should
 * either use this directly or extend it with additional context fields.
 *
 * @see SelectorError - Extended with selector context (matchCount, selector)
 * @see HandlerError - Alias for backward compatibility in handler code
 */
export interface ExecutionError {
  /** Human-readable error message */
  message: string;
  /** Structured error code for programmatic handling */
  code: string;
  /** Failure classification for retry/reporting logic */
  kind?: FailureKind | string;
  /** Whether the operation may succeed on retry */
  retryable?: boolean;
}

/**
 * Error type with selector context.
 * Used when failures relate to element selection (not found, ambiguous, etc.)
 *
 * Use this type when errors need to communicate selector-specific details
 * like match count or the selector that failed.
 */
export interface SelectorError extends ExecutionError {
  /** Number of elements matching the selector (0 = not found, >1 = ambiguous) */
  matchCount?: number;
  /** The selector that was used */
  selector?: string;
}

/**
 * Handler error type alias.
 * For backward compatibility with handler code that expects HandlerError.
 * This is semantically identical to ExecutionError.
 *
 * @deprecated Prefer using ExecutionError directly in new code
 */
export type HandlerError = ExecutionError;

/**
 * Create an ExecutionError from basic inputs.
 * Automatically determines retryability from error code.
 */
export function createExecutionError(
  message: string,
  code: HandlerErrorCode | ActionErrorCode,
  kind?: FailureKind
): ExecutionError {
  return {
    message,
    code,
    kind: kind ?? ERROR_CODE_TO_FAILURE_KIND[code] ?? FailureKind.ENGINE,
    retryable: RETRYABLE_ERROR_CODES.has(code),
  };
}

// =============================================================================
// SHARED METRICS
// =============================================================================

/**
 * Execution timing metrics.
 * Common structure for tracking execution duration.
 */
export interface ExecutionMetrics {
  /** Execution duration in milliseconds */
  durationMs: number;
  /** Start time (ISO 8601) */
  startedAt?: string;
  /** End time (ISO 8601) */
  endedAt?: string;
}

// =============================================================================
// BASE EXECUTION RESULT
// =============================================================================

/**
 * Base execution result interface.
 *
 * This is the foundational "did this work?" type that all domain-specific
 * result types should compose from. It provides the minimal contract for
 * execution outcomes across the codebase:
 *
 * - HandlerResult (outcome/outcome-builder.ts) - extends with telemetry fields
 * - ActionReplayResult (recording/action-executor.ts) - extends with entry metadata
 * - HandlerAdapterResult (recording/handler-adapter.ts) - minimal adapter result
 *
 * DESIGN: We use composition rather than forcing a single type because:
 * 1. Different contexts need different additional fields
 * 2. Handler results need telemetry (screenshots, DOM, logs)
 * 3. Replay results need entry IDs and action types
 * 4. Forcing one type would add unnecessary complexity to simple cases
 *
 * @example
 * // Implementing a domain-specific result type
 * interface MyResult extends BaseExecutionResult {
 *   myDomainField: string;
 * }
 */
export interface BaseExecutionResult {
  /** Whether the execution succeeded */
  success: boolean;
  /** Execution duration in milliseconds (optional - not all contexts track timing) */
  durationMs?: number;
  /** Error details if execution failed */
  error?: ExecutionError;
}

// =============================================================================
// RESULT TYPE UTILITIES
// =============================================================================

/**
 * Check if an error code is retryable.
 */
export function isRetryableError(code: string): boolean {
  return RETRYABLE_ERROR_CODES.has(code);
}

/**
 * Get FailureKind from error code.
 */
export function getFailureKindFromCode(code: string): FailureKind {
  return ERROR_CODE_TO_FAILURE_KIND[code] ?? FailureKind.ENGINE;
}

/**
 * Convert legacy string failure kind to proto enum.
 * Supports both proto enum values and legacy string values for backward compatibility.
 */
export function normalizeFailureKind(kind: FailureKind | string | undefined): FailureKind {
  if (kind === undefined) return FailureKind.ENGINE;

  // If it's already a proto enum value (number), return it
  if (typeof kind === 'number') return kind;

  // Legacy string conversion
  const FAILURE_KIND_MAP: Record<string, FailureKind> = {
    engine: FailureKind.ENGINE,
    infra: FailureKind.INFRA,
    orchestration: FailureKind.ORCHESTRATION,
    user: FailureKind.USER,
    timeout: FailureKind.TIMEOUT,
    cancelled: FailureKind.CANCELLED,
  };

  return FAILURE_KIND_MAP[kind] ?? FailureKind.ENGINE;
}
