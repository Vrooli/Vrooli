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
