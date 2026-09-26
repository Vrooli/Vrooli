/**
 * Infrastructure Module
 *
 * Cross-cutting infrastructure concerns that can be used across the codebase.
 * These are low-level utilities that don't depend on domain-specific logic.
 *
 * @module infra
 */

export {
  createCircuitBreaker,
  DEFAULT_CIRCUIT_BREAKER_CONFIG,
  type CircuitBreaker,
  type CircuitBreakerConfig,
  type CircuitState,
  type CircuitStats,
} from './circuit-breaker';

export {
  registerSessionCleanup,
  cleanupSession,
  getRegistrationCount,
  type SessionCleanupFn,
} from './session-cleanup-registry';

export {
  createOperationTracker,
  uploadTracker,
  tabTracker,
  type OperationTracker,
  type OperationTrackerConfig,
} from './operation-tracker';

export {
  createInFlightGuard,
  createSetGuard,
  createWeakSetGuard,
  type InFlightGuard,
  type InFlightGuardConfig,
  type InFlightStats,
  type SetGuard,
  type WeakSetGuard,
} from './in-flight-guard';
