/**
 * In-Flight Guard
 *
 * Prevents duplicate concurrent operations by returning the same promise
 * for operations with the same key that are already in-flight.
 *
 * This is different from IdempotencyCache:
 * - IdempotencyCache: Stores completed results for later lookup
 * - InFlightGuard: Returns same promise for concurrent in-flight operations
 *
 * Use cases:
 * - Session creation deduplication (same execution_id)
 * - Concurrent replay request deduplication
 * - Page setup lock (prevent duplicate route registration)
 * - Buffer entry deduplication (same entry ID)
 *
 * @module infra/in-flight-guard
 */

import { logger, scopedLog, LogContext } from '../utils';

// =============================================================================
// Types
// =============================================================================

/**
 * Configuration for an in-flight guard.
 */
export interface InFlightGuardConfig {
  /** Name for logging purposes */
  name: string;
  /** Log context for scoped logging (use values from LogContext enum) */
  logContext?: (typeof LogContext)[keyof typeof LogContext];
  /** Whether to log when returning in-flight promise (default: true) */
  logOnDedup?: boolean;
}

/**
 * Statistics about guard usage.
 */
export interface InFlightStats {
  /** Number of operations currently in-flight */
  inFlight: number;
  /** Total operations executed (not deduplicated) */
  executed: number;
  /** Total operations deduplicated (returned existing promise) */
  deduplicated: number;
}

/**
 * In-flight guard interface.
 */
export interface InFlightGuard<K, V> {
  /**
   * Execute an operation with deduplication.
   *
   * If an operation with the same key is already in-flight, returns
   * the existing promise instead of starting a new operation.
   *
   * @param key - Unique key for the operation
   * @param fn - Function to execute if not already in-flight
   * @returns Promise resolving to the operation result
   */
  execute(key: K, fn: () => Promise<V>): Promise<V>;

  /**
   * Check if an operation is currently in-flight.
   *
   * @param key - Key to check
   * @returns true if operation is in-flight
   */
  isInFlight(key: K): boolean;

  /**
   * Get the number of operations currently in-flight.
   */
  getInFlightCount(): number;

  /**
   * Get usage statistics.
   */
  getStats(): InFlightStats;

  /**
   * Clear all tracking (for shutdown).
   * Note: This does NOT cancel in-flight operations.
   */
  clear(): void;
}

// =============================================================================
// Implementation
// =============================================================================

/**
 * Create an in-flight guard for deduplicating concurrent operations.
 *
 * @example
 * ```typescript
 * const sessionGuard = createInFlightGuard<string, SessionResult>({
 *   name: 'session-creation',
 * });
 *
 * // Concurrent calls with same execution_id get the same promise
 * const result = await sessionGuard.execute(executionId, async () => {
 *   return await createSession(spec);
 * });
 * ```
 */
export function createInFlightGuard<K, V>(
  config: InFlightGuardConfig
): InFlightGuard<K, V> {
  const { name, logContext = LogContext.SESSION, logOnDedup = true } = config;
  const inFlight = new Map<K, Promise<V>>();

  // Statistics
  let executed = 0;
  let deduplicated = 0;

  return {
    async execute(key: K, fn: () => Promise<V>): Promise<V> {
      // Check if already in-flight
      const existing = inFlight.get(key);
      if (existing) {
        deduplicated++;
        if (logOnDedup) {
          logger.debug(scopedLog(logContext, `${name}: returning in-flight promise`), {
            key: typeof key === 'string' ? key : String(key),
            inFlightCount: inFlight.size,
          });
        }
        return existing;
      }

      // Start new operation
      executed++;
      const promise = fn();
      inFlight.set(key, promise);

      try {
        return await promise;
      } finally {
        // Always clean up, whether success or failure
        inFlight.delete(key);
      }
    },

    isInFlight(key: K): boolean {
      return inFlight.has(key);
    },

    getInFlightCount(): number {
      return inFlight.size;
    },

    getStats(): InFlightStats {
      return {
        inFlight: inFlight.size,
        executed,
        deduplicated,
      };
    },

    clear(): void {
      inFlight.clear();
    },
  };
}

// =============================================================================
// Set-Based Guard (for simple presence tracking)
// =============================================================================

/**
 * Simple set-based guard for tracking items without associated promises.
 *
 * Use cases:
 * - Tracking seen entry IDs (deduplication without promises)
 * - Tracking pages with event routes set up
 *
 * @example
 * ```typescript
 * const seenEntries = createSetGuard<string>({ name: 'seen-entries' });
 *
 * if (seenEntries.has(entryId)) {
 *   return false; // Already seen
 * }
 * seenEntries.add(entryId);
 * return true;
 * ```
 */
export interface SetGuard<K> {
  /** Check if key exists */
  has(key: K): boolean;
  /** Add key to set */
  add(key: K): void;
  /** Remove key from set */
  delete(key: K): boolean;
  /** Get size of set */
  size(): number;
  /** Clear all keys */
  clear(): void;
}

/**
 * Create a set-based guard for simple presence tracking.
 */
export function createSetGuard<K>(_config: { name: string }): SetGuard<K> {
  const set = new Set<K>();

  return {
    has: (key: K) => set.has(key),
    add: (key: K) => { set.add(key); },
    delete: (key: K) => set.delete(key),
    size: () => set.size,
    clear: () => set.clear(),
  };
}

// =============================================================================
// WeakSet-Based Guard (for object keys that can be garbage collected)
// =============================================================================

/**
 * WeakSet-based guard for tracking objects without preventing garbage collection.
 *
 * Use cases:
 * - Tracking Page objects with event routes set up
 * - Any case where keys are objects that may be garbage collected
 *
 * Note: WeakSet doesn't support iteration or size(), so this is more limited.
 */
export interface WeakSetGuard<K extends WeakKey> {
  /** Check if key exists */
  has(key: K): boolean;
  /** Add key to set */
  add(key: K): void;
  /** Remove key from set */
  delete(key: K): boolean;
}

/**
 * Create a WeakSet-based guard for object tracking.
 */
export function createWeakSetGuard<K extends WeakKey>(): WeakSetGuard<K> {
  const set = new WeakSet<K>();

  return {
    has: (key: K) => set.has(key),
    add: (key: K) => { set.add(key); },
    delete: (key: K) => set.delete(key),
  };
}
