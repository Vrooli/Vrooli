/**
 * Operation Tracker
 *
 * Provides in-flight operation deduplication for handlers.
 * This extracts the common pattern of tracking concurrent operations
 * from individual handlers into a reusable infrastructure component.
 *
 * ARCHITECTURAL SEAM:
 * - Handlers use OperationTracker for concurrent operation deduplication
 * - Session cleanup registry clears tracker state on session close/reset
 * - Testing can create isolated tracker instances
 *
 * Features:
 * - In-flight tracking: Prevents concurrent operations with same key
 * - Result caching: Optional caching of successful results for retry
 * - Session isolation: Operations are scoped to sessions
 * - Testability: Factory function creates isolated instances
 *
 * @module infra/operation-tracker
 */

import { logger, scopedLog, LogContext } from '../utils';
import { registerSessionCleanup } from './session-cleanup-registry';

// =============================================================================
// Types
// =============================================================================

/**
 * Configuration for operation tracker.
 */
export interface OperationTrackerConfig {
  /** Unique name for this tracker (for logging) */
  name: string;
  /** Time-to-live for cached results in milliseconds */
  resultCacheTtlMs?: number;
  /** Whether to cache successful results */
  cacheResults?: boolean;
  /** Maximum cached entries per session */
  maxCachedPerSession?: number;
}

/**
 * A cached result entry.
 */
interface CachedResult<T> {
  result: T;
  timestamp: number;
}

/**
 * Operation tracker interface.
 */
export interface OperationTracker<T> {
  /**
   * Check if an operation with this key is already in-flight.
   * If so, return the pending promise. Otherwise, return null.
   */
  getInFlight(sessionId: string, operationKey: string): Promise<T> | null;

  /**
   * Track an in-flight operation.
   * Returns a cleanup function to call when the operation completes.
   */
  trackInFlight(sessionId: string, operationKey: string, promise: Promise<T>): () => void;

  /**
   * Get a cached result if available and not expired.
   */
  getCached(sessionId: string, operationKey: string): T | null;

  /**
   * Cache a successful result.
   */
  cacheResult(sessionId: string, operationKey: string, result: T): void;

  /**
   * Clear all state for a session.
   */
  clearSession(sessionId: string): void;

  /**
   * Get the tracker name (for logging/debugging).
   */
  getName(): string;
}

// =============================================================================
// Default Configuration
// =============================================================================

const DEFAULT_CONFIG: Required<Omit<OperationTrackerConfig, 'name'>> = {
  resultCacheTtlMs: 300_000, // 5 minutes
  cacheResults: false,
  maxCachedPerSession: 100,
};

// =============================================================================
// Implementation
// =============================================================================

/**
 * Create an operation tracker instance.
 *
 * @example
 * ```typescript
 * const downloadTracker = createOperationTracker<HandlerResult>({
 *   name: 'download',
 *   cacheResults: true,
 *   resultCacheTtlMs: 300_000,
 * });
 *
 * // In handler:
 * const inFlight = downloadTracker.getInFlight(sessionId, downloadKey);
 * if (inFlight) {
 *   return inFlight; // Return pending operation
 * }
 *
 * const cached = downloadTracker.getCached(sessionId, downloadKey);
 * if (cached) {
 *   return cached; // Return cached result
 * }
 *
 * const promise = executeDownload();
 * const cleanup = downloadTracker.trackInFlight(sessionId, downloadKey, promise);
 *
 * try {
 *   const result = await promise;
 *   downloadTracker.cacheResult(sessionId, downloadKey, result);
 *   return result;
 * } finally {
 *   cleanup();
 * }
 * ```
 */
export function createOperationTracker<T>(config: OperationTrackerConfig): OperationTracker<T> {
  const cfg = { ...DEFAULT_CONFIG, ...config };

  // In-flight operations: sessionId -> operationKey -> Promise
  const inFlight = new Map<string, Map<string, Promise<T>>>();

  // Cached results: sessionId -> operationKey -> CachedResult
  const cached = new Map<string, Map<string, CachedResult<T>>>();

  function getSessionInFlight(sessionId: string): Map<string, Promise<T>> {
    let session = inFlight.get(sessionId);
    if (!session) {
      session = new Map();
      inFlight.set(sessionId, session);
    }
    return session;
  }

  function getSessionCached(sessionId: string): Map<string, CachedResult<T>> {
    let session = cached.get(sessionId);
    if (!session) {
      session = new Map();
      cached.set(sessionId, session);
    }
    return session;
  }

  const tracker: OperationTracker<T> = {
    getInFlight(sessionId: string, operationKey: string): Promise<T> | null {
      const session = inFlight.get(sessionId);
      if (!session) return null;

      const pending = session.get(operationKey);
      if (pending) {
        logger.debug(scopedLog(LogContext.INSTRUCTION, `${cfg.name}: awaiting in-flight operation`), {
          sessionId,
          operationKey,
        });
        return pending;
      }
      return null;
    },

    trackInFlight(sessionId: string, operationKey: string, promise: Promise<T>): () => void {
      const session = getSessionInFlight(sessionId);
      session.set(operationKey, promise);

      logger.debug(scopedLog(LogContext.INSTRUCTION, `${cfg.name}: tracking in-flight operation`), {
        sessionId,
        operationKey,
      });

      return () => {
        session.delete(operationKey);
        // Clean up empty session map
        if (session.size === 0) {
          inFlight.delete(sessionId);
        }
      };
    },

    getCached(sessionId: string, operationKey: string): T | null {
      if (!cfg.cacheResults) return null;

      const session = cached.get(sessionId);
      if (!session) return null;

      const entry = session.get(operationKey);
      if (!entry) return null;

      // Check TTL
      if (Date.now() - entry.timestamp > cfg.resultCacheTtlMs) {
        session.delete(operationKey);
        return null;
      }

      logger.debug(scopedLog(LogContext.INSTRUCTION, `${cfg.name}: returning cached result`), {
        sessionId,
        operationKey,
        cacheAgeMs: Date.now() - entry.timestamp,
      });

      return entry.result;
    },

    cacheResult(sessionId: string, operationKey: string, result: T): void {
      if (!cfg.cacheResults) return;

      const session = getSessionCached(sessionId);

      // Enforce max size with FIFO eviction
      if (session.size >= cfg.maxCachedPerSession) {
        const firstKey = session.keys().next().value;
        if (firstKey) {
          session.delete(firstKey);
        }
      }

      session.set(operationKey, {
        result,
        timestamp: Date.now(),
      });

      logger.debug(scopedLog(LogContext.INSTRUCTION, `${cfg.name}: cached result`), {
        sessionId,
        operationKey,
        cacheSize: session.size,
      });
    },

    clearSession(sessionId: string): void {
      const inFlightCount = inFlight.get(sessionId)?.size ?? 0;
      const cachedCount = cached.get(sessionId)?.size ?? 0;

      inFlight.delete(sessionId);
      cached.delete(sessionId);

      if (inFlightCount > 0 || cachedCount > 0) {
        logger.debug(scopedLog(LogContext.CLEANUP, `${cfg.name}: cleared session state`), {
          sessionId,
          inFlightCleared: inFlightCount,
          cachedCleared: cachedCount,
        });
      }
    },

    getName(): string {
      return cfg.name;
    },
  };

  // Register with session cleanup registry
  registerSessionCleanup(`operation-tracker:${cfg.name}`, (sessionId) => {
    tracker.clearSession(sessionId);
  });

  return tracker;
}

// =============================================================================
// Pre-built Trackers for Common Handlers
// =============================================================================

/**
 * Global download operation tracker.
 * Used by the download handler for in-flight deduplication and result caching.
 */
export const downloadTracker = createOperationTracker<unknown>({
  name: 'download',
  cacheResults: true,
  resultCacheTtlMs: 300_000, // 5 minutes
});

/**
 * Global upload operation tracker.
 * Used by the upload handler for in-flight deduplication.
 */
export const uploadTracker = createOperationTracker<unknown>({
  name: 'upload',
  cacheResults: false, // Uploads don't need result caching
});

/**
 * Global tab operation tracker.
 * Used by the tab handler for in-flight deduplication.
 */
export const tabTracker = createOperationTracker<unknown>({
  name: 'tab',
  cacheResults: false,
});
