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
  createIdempotencyCache,
  getIdempotencyCache,
  shutdownIdempotencyCache,
  DEFAULT_IDEMPOTENCY_CACHE_CONFIG,
  type IdempotencyCache,
  type IdempotencyCacheConfig,
  type CachedEntry,
  type CacheStats,
} from './idempotency-cache';
