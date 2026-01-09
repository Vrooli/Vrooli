/**
 * Session Cleanup Registry
 *
 * Provides a centralized registry for session cleanup operations.
 * This creates a seam between session management and handler-specific cleanup,
 * eliminating direct imports from session layer into handler modules.
 *
 * ARCHITECTURAL SEAM:
 * - Session manager calls `cleanupSession(sessionId)` without knowing handler details
 * - Handlers register their cleanup functions at module load time
 * - Cleanup order is defined by registration order (FIFO)
 *
 * @module infra/session-cleanup-registry
 */

import { logger, scopedLog, LogContext } from '../utils';

// =============================================================================
// Types
// =============================================================================

/**
 * A cleanup function that clears session-specific state.
 * Should be idempotent - safe to call multiple times.
 */
export type SessionCleanupFn = (sessionId: string) => void | Promise<void>;

/**
 * Registration entry for a cleanup function.
 */
interface CleanupRegistration {
  /** Unique name for logging and debugging */
  name: string;
  /** The cleanup function */
  cleanup: SessionCleanupFn;
}

// =============================================================================
// Registry State
// =============================================================================

/**
 * Registered cleanup functions.
 * Order matters - functions are called in registration order.
 */
const registrations: CleanupRegistration[] = [];

// =============================================================================
// Public API
// =============================================================================

/**
 * Register a cleanup function for session resources.
 *
 * Handlers should call this at module initialization to register their
 * session-specific cleanup operations.
 *
 * @param name - Unique identifier for this cleanup operation (for logging)
 * @param cleanup - Function to call when a session is closed or reset
 *
 * @example
 * ```typescript
 * // In handlers/download.ts
 * registerSessionCleanup('download-cache', (sessionId) => {
 *   downloadCache.delete(sessionId);
 * });
 * ```
 */
export function registerSessionCleanup(name: string, cleanup: SessionCleanupFn): void {
  // Check for duplicate registration (dev-time warning)
  const existing = registrations.find((r) => r.name === name);
  if (existing) {
    logger.warn(scopedLog(LogContext.CLEANUP, 'duplicate cleanup registration'), {
      name,
      hint: 'Cleanup function with this name already registered. Ignoring duplicate.',
    });
    return;
  }

  registrations.push({ name, cleanup });

  logger.debug(scopedLog(LogContext.CLEANUP, 'cleanup function registered'), {
    name,
    totalRegistrations: registrations.length,
  });
}

/**
 * Clean up all registered session resources.
 *
 * Called by session manager during session close or reset.
 * Executes all registered cleanup functions in order.
 *
 * @param sessionId - The session being cleaned up
 */
export async function cleanupSession(sessionId: string): Promise<void> {
  if (registrations.length === 0) {
    return;
  }

  logger.debug(scopedLog(LogContext.CLEANUP, 'running session cleanup'), {
    sessionId,
    cleanupCount: registrations.length,
  });

  for (const { name, cleanup } of registrations) {
    try {
      await cleanup(sessionId);
    } catch (error) {
      // Log but don't fail - continue with other cleanup operations
      logger.warn(scopedLog(LogContext.CLEANUP, 'cleanup function failed'), {
        sessionId,
        name,
        error: error instanceof Error ? error.message : String(error),
        hint: 'Cleanup failed but continuing with remaining cleanup operations',
      });
    }
  }
}

/**
 * Get count of registered cleanup functions.
 * Useful for testing and debugging.
 */
export function getRegistrationCount(): number {
  return registrations.length;
}

/**
 * Clear all registrations.
 * Only for testing - should not be called in production.
 *
 * @internal
 */
export function _clearRegistrations(): void {
  registrations.length = 0;
}
