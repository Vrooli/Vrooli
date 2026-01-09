/**
 * Callback Streaming
 *
 * Handles streaming of recorded TimelineEntry events to callback URLs.
 * Uses circuit breaker pattern to prevent cascading failures when the
 * callback endpoint is unavailable.
 *
 * Graceful Degradation:
 * - Actions are always buffered locally, even when callbacks fail
 * - Circuit breaker opens after MAX_CALLBACK_FAILURES consecutive failures
 * - Circuit breaker resets after CIRCUIT_RESET_MS
 */

import { timelineEntryToJson, type TimelineEntry, ActionType } from '../../proto/recording';
import { createCircuitBreaker, type CircuitBreaker } from '../../infra';
import { logger, metrics, scopedLog, LogContext } from '../../utils';

// =============================================================================
// Circuit Breaker for Callback Streaming
// =============================================================================

/**
 * Circuit breaker for callback streaming.
 * Prevents cascading failures when callback endpoint is unavailable.
 */
export const callbackCircuitBreaker: CircuitBreaker<string> = createCircuitBreaker({
  maxFailures: 5,
  resetTimeoutMs: 30_000, // 30 seconds
  name: 'callback',
});

// Callback streaming timeout (5 seconds - callbacks should be fast)
const CALLBACK_TIMEOUT_MS = 5_000;

// =============================================================================
// Streaming Functions
// =============================================================================

/**
 * Stream a TimelineEntry to callback URL with circuit breaker protection.
 *
 * This handles the full lifecycle of streaming an entry:
 * 1. Check if circuit is open (skip if so, unless entering half-open)
 * 2. Stream the entry to the callback URL
 * 3. Record success/failure with the circuit breaker
 * 4. Log appropriate messages and metrics
 *
 * @param sessionId - Session ID for circuit breaker and logging
 * @param callbackUrl - URL to stream the entry to
 * @param entry - The TimelineEntry to stream
 */
export async function streamEntryWithCircuitBreaker(
  sessionId: string,
  callbackUrl: string,
  entry: TimelineEntry
): Promise<void> {
  const actionTypeName = ActionType[entry.action?.type ?? ActionType.UNSPECIFIED] ?? 'UNKNOWN';

  // Check if we should attempt half-open (atomically claims the attempt)
  const attemptHalfOpen = callbackCircuitBreaker.tryEnterHalfOpen(sessionId);

  // Skip if circuit is open and we're not the half-open attempt
  if (callbackCircuitBreaker.isOpen(sessionId) && !attemptHalfOpen) {
    const stats = callbackCircuitBreaker.getStats(sessionId);
    logger.debug(scopedLog(LogContext.RECORDING, 'skipping callback (circuit open)'), {
      sessionId,
      actionType: actionTypeName,
      sequenceNum: entry.sequenceNum,
      halfOpenInProgress: stats?.halfOpenInProgress,
    });
    return;
  }

  try {
    await streamEntryToCallback(callbackUrl, entry);
    callbackCircuitBreaker.recordSuccess(sessionId);
  } catch (err) {
    const circuitOpened = callbackCircuitBreaker.recordFailure(sessionId);

    // Classify failure reason for metrics
    const errorMessage = err instanceof Error ? err.message : String(err);
    let reason = 'unknown';
    if (errorMessage.includes('ECONNREFUSED') || errorMessage.includes('ENOTFOUND')) {
      reason = 'network';
    } else if (errorMessage.includes('timeout') || errorMessage.includes('ETIMEDOUT')) {
      reason = 'timeout';
    } else if (errorMessage.includes('returned')) {
      reason = 'http_error';
    }

    metrics.recordingCallbackFailures.inc({ reason });

    const stats = callbackCircuitBreaker.getStats(sessionId);
    logger.warn(scopedLog(LogContext.RECORDING, 'callback streaming failed'), {
      sessionId,
      callbackUrl,
      error: errorMessage,
      failureReason: reason,
      consecutiveFailures: stats?.consecutiveFailures,
      circuitOpen: circuitOpened,
      hint: circuitOpened
        ? 'Circuit breaker opened - entries still buffered locally'
        : 'Callback endpoint may be unreachable or returning errors',
    });
  }
}

/**
 * Stream a TimelineEntry to a callback URL with timeout.
 *
 * Hardened: Uses AbortController to enforce timeout and prevent
 * indefinite hangs if the callback endpoint is slow or unresponsive.
 *
 * Wire format: Serializes proto TimelineEntry to JSON (snake_case)
 * for interoperability with the Go API.
 */
async function streamEntryToCallback(callbackUrl: string, entry: TimelineEntry): Promise<void> {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), CALLBACK_TIMEOUT_MS);

  try {
    // Serialize proto to JSON wire format (snake_case)
    const wireFormat = timelineEntryToJson(entry);

    const response = await fetch(callbackUrl, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(wireFormat),
      signal: controller.signal,
    });

    if (!response.ok) {
      throw new Error(`Callback returned ${response.status}: ${response.statusText}`);
    }

    const actionTypeName = ActionType[entry.action?.type ?? ActionType.UNSPECIFIED] ?? 'UNKNOWN';
    logger.debug('recording: entry streamed', {
      status: response.status,
      actionType: actionTypeName,
      sequenceNum: entry.sequenceNum,
    });
  } finally {
    clearTimeout(timeoutId);
  }
}
