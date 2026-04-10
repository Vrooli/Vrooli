/**
 * Service layer - pure functions for business logic.
 *
 * Services contain:
 * - Pure functions with no side effects
 * - Data transformations
 * - Validation logic
 * - State computation
 *
 * Services do NOT:
 * - Make API calls directly (use controllers for orchestration)
 * - Manage React state
 * - Access the DOM
 */

export * from "./generator.service";
export * from "./pipeline.service";
export * from "./preflight.service";
export * from "./signing.service";
