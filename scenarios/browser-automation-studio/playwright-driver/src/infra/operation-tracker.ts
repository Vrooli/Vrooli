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
   * Clear all state for a session.
   */
  clearSession(sessionId: string): void;

  /**
   * Get the tracker name (for logging/debugging).
   */
  getName(): string;
}

export function createOperationTracker<T>(config: OperationTrackerConfig): OperationTracker<T> {
  // In-flight operations: sessionId -> operationKey -> Promise
  const inFlight = new Map<string, Map<string, Promise<T>>>();

  function getSessionInFlight(sessionId: string): Map<string, Promise<T>> {
    let session = inFlight.get(sessionId);
    if (!session) {
      session = new Map();
      inFlight.set(sessionId, session);
    }
    return session;
  }

  const tracker: OperationTracker<T> = {
    getInFlight(sessionId: string, operationKey: string): Promise<T> | null {
      const session = inFlight.get(sessionId);
      if (!session) return null;

      const pending = session.get(operationKey);
      if (pending) {
        logger.debug(scopedLog(LogContext.INSTRUCTION, `${config.name}: awaiting in-flight operation`), {
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

      logger.debug(scopedLog(LogContext.INSTRUCTION, `${config.name}: tracking in-flight operation`), {
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

    clearSession(sessionId: string): void {
      const inFlightCount = inFlight.get(sessionId)?.size ?? 0;

      inFlight.delete(sessionId);

      if (inFlightCount > 0) {
        logger.debug(scopedLog(LogContext.CLEANUP, `${config.name}: cleared session state`), {
          sessionId,
          inFlightCleared: inFlightCount,
        });
      }
    },

    getName(): string {
      return config.name;
    },
  };

  // Register with session cleanup registry
  registerSessionCleanup(`operation-tracker:${config.name}`, (sessionId) => {
    tracker.clearSession(sessionId);
  });

  return tracker;
}

// =============================================================================
// Pre-built Trackers for Common Handlers
// =============================================================================

/**
 * Global upload operation tracker.
 * Used by the upload handler for in-flight deduplication.
 */
export const uploadTracker = createOperationTracker<unknown>({
  name: 'upload',
});

/**
 * Global tab operation tracker.
 * Used by the tab handler for in-flight deduplication.
 */
export const tabTracker = createOperationTracker<unknown>({
  name: 'tab',
});
