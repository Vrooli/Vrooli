/**
 * Idempotency Cache
 *
 * Provides request deduplication for safe retry behavior.
 * Caches responses by idempotency key so retried requests return
 * the same result without re-execution.
 *
 * Features:
 * - TTL-based expiration (default 5 minutes)
 * - Max size enforcement with LRU eviction
 * - Session-scoped cache clearing
 * - Automatic cleanup on interval
 *
 * @module infra/idempotency-cache
 */

import { logger, scopedLog, LogContext } from '../utils';

// =============================================================================
// Types
// =============================================================================

/**
 * Idempotency cache configuration.
 */
export interface IdempotencyCacheConfig {
  /** Time-to-live for cached entries in milliseconds */
  ttlMs: number;
  /** Maximum number of entries in the cache */
  maxSize: number;
  /** Cleanup interval in milliseconds */
  cleanupIntervalMs: number;
  /** Optional name for logging */
  name?: string;
}

/**
 * A cached result entry.
 */
export interface CachedEntry<T = unknown> {
  /** The cached response value */
  response: T;
  /** Timestamp when the entry was cached */
  timestamp: number;
  /** Session ID that owns this entry */
  sessionId: string;
  /** Key identifying the instruction/operation */
  instructionKey: string;
}

/**
 * Cache statistics.
 */
export interface CacheStats {
  /** Current number of entries */
  size: number;
  /** Number of cache hits */
  hits: number;
  /** Number of cache misses */
  misses: number;
  /** Number of entries expired */
  expired: number;
  /** Number of entries evicted for size */
  evicted: number;
}

/**
 * Idempotency cache interface.
 */
export interface IdempotencyCache<T = unknown> {
  /** Look up a cached entry by key. Returns null if not found or expired. */
  lookup(key: string, sessionId: string): T | null;
  /** Store a response in the cache */
  store(key: string, sessionId: string, instructionKey: string, response: T): void;
  /** Clear all entries for a specific session */
  clearSession(sessionId: string): number;
  /** Get cache statistics */
  getStats(): CacheStats;
  /** Manually trigger cleanup of expired entries */
  cleanup(): number;
  /** Shut down the cache (stops cleanup timer) */
  shutdown(): void;
}

// =============================================================================
// Default Configuration
// =============================================================================

export const DEFAULT_IDEMPOTENCY_CACHE_CONFIG: IdempotencyCacheConfig = {
  ttlMs: 300_000, // 5 minutes
  maxSize: 10_000,
  cleanupIntervalMs: 60_000, // 1 minute
  name: 'idempotency-cache',
};

// =============================================================================
// Helpers
// =============================================================================

/**
 * Check if a timestamp is within the TTL window.
 */
function isRecent(timestamp: number, ttlMs: number): boolean {
  return Date.now() - timestamp < ttlMs;
}

// =============================================================================
// Implementation
// =============================================================================

/**
 * Create a new idempotency cache instance.
 *
 * The cache allows clients to safely retry requests - if the same
 * idempotency key is provided, the cached result is returned instead
 * of re-executing the operation.
 *
 * @example
 * ```typescript
 * const cache = createIdempotencyCache<DriverOutcome>();
 *
 * // Check for cached result
 * const cached = cache.lookup(idempotencyKey, sessionId);
 * if (cached) {
 *   return cached;
 * }
 *
 * // Execute operation
 * const result = await executeInstruction();
 *
 * // Cache result
 * cache.store(idempotencyKey, sessionId, instructionKey, result);
 * return result;
 * ```
 */
export function createIdempotencyCache<T = unknown>(
  config: Partial<IdempotencyCacheConfig> = {}
): IdempotencyCache<T> {
  const cfg: IdempotencyCacheConfig = { ...DEFAULT_IDEMPOTENCY_CACHE_CONFIG, ...config };
  const cache = new Map<string, CachedEntry<T>>();

  // Statistics
  let hits = 0;
  let misses = 0;
  let expired = 0;
  let evicted = 0;

  /**
   * Enforce max cache size by evicting oldest entries.
   */
  function enforceMaxSize(): void {
    if (cache.size <= cfg.maxSize) return;

    const toEvict = cache.size - cfg.maxSize;
    let evictedCount = 0;

    // Map iteration order is insertion order, so we evict oldest first
    for (const key of cache.keys()) {
      if (evictedCount >= toEvict) break;
      cache.delete(key);
      evictedCount++;
    }

    evicted += evictedCount;

    if (evictedCount > 0) {
      logger.warn(scopedLog(LogContext.INSTRUCTION, `${cfg.name}: size limit reached`), {
        maxSize: cfg.maxSize,
        evictedForSize: evictedCount,
      });
    }
  }

  /**
   * Remove expired entries from the cache.
   */
  function cleanupExpired(): number {
    const now = Date.now();
    let expiredCount = 0;

    for (const [key, entry] of cache.entries()) {
      if (now - entry.timestamp > cfg.ttlMs) {
        cache.delete(key);
        expiredCount++;
      }
    }

    expired += expiredCount;

    if (expiredCount > 0) {
      logger.debug(scopedLog(LogContext.INSTRUCTION, `${cfg.name}: cleanup`), {
        expiredCount,
        remainingCount: cache.size,
      });
    }

    // Also enforce size limit after cleanup
    enforceMaxSize();

    return expiredCount;
  }

  // Start cleanup interval
  const cleanupTimer = setInterval(cleanupExpired, cfg.cleanupIntervalMs);
  cleanupTimer.unref(); // Don't keep process alive for cleanup

  return {
    lookup(key: string, sessionId: string): T | null {
      if (!key) {
        misses++;
        return null;
      }

      const entry = cache.get(key);

      if (!entry || !isRecent(entry.timestamp, cfg.ttlMs)) {
        misses++;
        return null;
      }

      // Security: Reject if key was used for a different session
      if (entry.sessionId !== sessionId) {
        logger.warn(scopedLog(LogContext.INSTRUCTION, `${cfg.name}: key reused for different session`), {
          sessionId,
          cachedSessionId: entry.sessionId,
          idempotencyKey: key,
        });
        misses++;
        return null;
      }

      hits++;
      logger.info(scopedLog(LogContext.INSTRUCTION, `${cfg.name}: returning cached result`), {
        sessionId,
        idempotencyKey: key,
        instructionKey: entry.instructionKey,
        cacheAgeMs: Date.now() - entry.timestamp,
      });

      return entry.response;
    },

    store(key: string, sessionId: string, instructionKey: string, response: T): void {
      if (!key) return;

      cache.set(key, {
        response,
        timestamp: Date.now(),
        sessionId,
        instructionKey,
      });

      logger.debug(scopedLog(LogContext.INSTRUCTION, `${cfg.name}: cached result`), {
        sessionId,
        idempotencyKey: key,
        instructionKey,
        cacheSize: cache.size,
      });

      // Enforce size limit
      enforceMaxSize();
    },

    clearSession(sessionId: string): number {
      let clearedCount = 0;

      for (const [key, entry] of cache.entries()) {
        if (entry.sessionId === sessionId) {
          cache.delete(key);
          clearedCount++;
        }
      }

      if (clearedCount > 0) {
        logger.debug(scopedLog(LogContext.SESSION, `${cfg.name}: cleared session entries`), {
          sessionId,
          clearedCount,
        });
      }

      return clearedCount;
    },

    getStats(): CacheStats {
      return {
        size: cache.size,
        hits,
        misses,
        expired,
        evicted,
      };
    },

    cleanup(): number {
      return cleanupExpired();
    },

    shutdown(): void {
      clearInterval(cleanupTimer);
      cache.clear();
    },
  };
}

// =============================================================================
// Singleton Instance
// =============================================================================

/**
 * Default idempotency cache instance for instruction execution.
 * Used by session-run.ts for request deduplication.
 */
let defaultCache: IdempotencyCache | null = null;

/**
 * Get the default idempotency cache instance.
 * Creates the instance on first call (lazy initialization).
 */
export function getIdempotencyCache(): IdempotencyCache {
  if (!defaultCache) {
    defaultCache = createIdempotencyCache();
  }
  return defaultCache;
}

/**
 * Shut down and clear the default cache.
 * Called during server shutdown.
 */
export function shutdownIdempotencyCache(): void {
  if (defaultCache) {
    defaultCache.shutdown();
    defaultCache = null;
  }
}
