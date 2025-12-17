/**
 * Outcome Module
 *
 * STABILITY: STABLE CONTRACT BOUNDARY
 *
 * Transforms handler results into wire-format outcomes for the Go API.
 * This module owns the serialization boundary between internal types
 * and external contract types (StepOutcome, DriverOutcome).
 *
 * CRITICAL: Changes here affect the Go API contract.
 *
 * Wire Format:
 * - StepOutcome: Proto-defined canonical outcome (from @vrooli/proto-types)
 * - DriverOutcome: Flattened format for Go parsing (screenshot_base64, etc.)
 *
 * CHANGE AXIS: New Outcome Fields
 *
 * When adding a new field to step outcomes:
 * 1. Add field to StepOutcome in proto schema (driver.proto)
 * 2. Run proto generation (make generate in packages/proto)
 * 3. Add to BuildOutcomeParams if data comes from execution
 * 4. Update buildStepOutcome() to populate the field
 * 5. If field needs flattening, update toDriverOutcome()
 *
 * Adding fields is safe (proto uses optional by default).
 * Removing/renaming fields requires proto deprecation pattern.
 */

export {
  buildStepOutcome,
  toDriverOutcome,
  type BuildOutcomeParams,
  type HandlerResult,
  type DriverOutcome,
  type Screenshot,
  type DOMSnapshot,
  type ConsoleLogEntry,
  type NetworkEvent,
  type HandlerAssertionOutcome,
} from './outcome-builder';

// Unified outcome types (shared error codes, failure kinds, utilities)
export {
  type HandlerErrorCode,
  type ActionErrorCode,
  type ExecutionError,
  type ExecutionMetrics,
  type BaseExecutionResult,
  ERROR_CODE_TO_FAILURE_KIND,
  RETRYABLE_ERROR_CODES,
  createExecutionError,
  isRetryableError,
  getFailureKindFromCode,
  normalizeFailureKind,
} from './types';
