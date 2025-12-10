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
 * - StepOutcome: Canonical outcome with nested objects
 * - DriverOutcome: Flattened format for Go parsing (screenshot_base64, etc.)
 *
 * CHANGE AXIS: New Outcome Fields
 *
 * When adding a new field to step outcomes:
 * 1. Add field to StepOutcome in `types/contracts.ts`
 * 2. Add to BuildOutcomeParams if data comes from execution
 * 3. Update buildStepOutcome() to populate the field
 * 4. If field needs flattening, update toDriverOutcome()
 * 5. Coordinate with Go API team for contracts.StepOutcome
 *
 * Adding fields is safe (Go uses omitempty).
 * Removing/renaming fields is a BREAKING CHANGE requiring API migration.
 */

export * from './outcome-builder';
