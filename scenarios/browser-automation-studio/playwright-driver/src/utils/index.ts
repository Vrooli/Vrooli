/**
 * Utils Module
 *
 * STABILITY: STABLE CORE
 *
 * Cross-cutting infrastructure concerns:
 * - Logger: Structured logging with scoped contexts
 * - Metrics: Prometheus metric definitions
 * - Metrics Server: Standalone metrics HTTP endpoint
 * - Errors: Typed error hierarchy (SessionNotFound, Timeout, etc.)
 * - Validation: Input validation and sanitization utilities
 *
 * These utilities are used throughout the codebase but don't
 * contain business logic specific to browser automation.
 *
 * CHANGE AXIS: Error Types
 * Primary extension point: errors.ts
 *
 * When adding a new error type:
 * 1. Define error class extending PlaywrightDriverError in errors.ts
 * 2. Assign unique error code (SCREAMING_SNAKE_CASE)
 * 3. Set appropriate FailureKind (engine/orchestration/infra/timeout/user/cancelled)
 * 4. Document retryability semantics
 *
 * Error codes are STABLE contracts - don't rename existing codes.
 */

export * from './logger';
export * from './metrics';
export * from './metrics-server';
export * from './errors';
export * from './validation';
