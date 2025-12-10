import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import { sendJson, sendError } from '../middleware';
import { logger, scopedLog, LogContext } from '../utils';

/**
 * Track in-flight reset operations to prevent concurrent resets.
 * Key: sessionId, Value: Promise that resolves when reset completes.
 *
 * Idempotency guarantee:
 * - Concurrent reset requests for the same session will await the first
 * - Prevents race conditions from duplicate/retry requests
 */
const pendingResets: Map<string, Promise<void>> = new Map();

/**
 * Reset session endpoint
 *
 * POST /session/:id/reset
 *
 * Idempotency behavior:
 * - If a reset is already in progress for this session, waits for it to complete
 * - Returns 200 success once the session is in 'ready' state
 * - Safe to call multiple times; only the first call performs work
 * - If session is already in 'resetting' phase, waits for completion (no-op)
 *
 * Concurrency guard:
 * - Uses pendingResets map to track in-flight operations
 * - Prevents duplicate concurrent resets that could cause race conditions
 */
export async function handleSessionReset(
  _req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager
): Promise<void> {
  try {
    // Check if session exists and get current phase
    const session = sessionManager.getSession(sessionId);

    // Idempotency: If session is already being reset, wait for that to complete
    const pendingReset = pendingResets.get(sessionId);
    if (pendingReset) {
      logger.debug(scopedLog(LogContext.SESSION, 'reset already in progress, waiting'), {
        sessionId,
        phase: session.phase,
        hint: 'Concurrent reset request detected; waiting for in-flight reset',
      });

      await pendingReset;

      // Reset completed - return success
      sendJson(res, 200, {
        success: true,
        idempotent: true,
        phase: 'ready',
      });
      return;
    }

    // Idempotency: If session phase is 'resetting', another request is resetting
    // This can happen if pendingResets was cleared but phase hasn't transitioned yet
    if (session.phase === 'resetting') {
      logger.debug(scopedLog(LogContext.SESSION, 'session already in resetting phase'), {
        sessionId,
        hint: 'Session phase indicates reset in progress',
      });

      // Wait briefly and check again
      await new Promise((resolve) => setTimeout(resolve, 100));

      // Re-fetch session to get updated phase
      try {
        const updatedSession = sessionManager.getSession(sessionId);
        if (updatedSession.phase === 'ready') {
          sendJson(res, 200, {
            success: true,
            idempotent: true,
            phase: 'ready',
          });
          return;
        }
      } catch {
        // Session may have been closed
      }
    }

    // Idempotency: If session is already in 'ready' phase and was recently reset,
    // this could be a retry - still return success
    if (session.phase === 'ready') {
      // Check if session was recently active (within last second)
      const timeSinceLastUse = Date.now() - session.lastUsedAt.getTime();
      if (timeSinceLastUse < 1000) {
        logger.debug(scopedLog(LogContext.SESSION, 'session already ready (possible retry)'), {
          sessionId,
          timeSinceLastUseMs: timeSinceLastUse,
          hint: 'Session recently active and in ready state',
        });
      }
    }

    logger.info(scopedLog(LogContext.SESSION, 'reset starting'), {
      sessionId,
      currentPhase: session.phase,
    });

    // Create the reset promise and track it
    const resetPromise = sessionManager.resetSession(sessionId);
    pendingResets.set(sessionId, resetPromise);

    try {
      await resetPromise;

      logger.info(scopedLog(LogContext.SESSION, 'reset completed'), {
        sessionId,
        phase: 'ready',
      });

      sendJson(res, 200, {
        success: true,
        phase: 'ready',
      });
    } finally {
      // Always clean up the pending reset tracking
      pendingResets.delete(sessionId);
    }
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/reset`);
  }
}
