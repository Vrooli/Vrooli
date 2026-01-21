/**
 * Retry Service
 *
 * Pure domain service for exponential backoff retry logic.
 * Extracted from hooks/useRecordingSession.ts for better testability.
 *
 * ## Problem
 *
 * Session creation can fail due to:
 * - Network issues
 * - Browser launch failures
 * - Resource contention
 *
 * Naive retries would hammer the server; no retries would frustrate users.
 *
 * ## Solution
 *
 * Exponential backoff with:
 * - Configurable max retries (default: 3)
 * - Exponential delay: 1s, 2s, 4s, 8s... capped at max
 * - State tracking for UI feedback (cooldown countdown, max exceeded)
 *
 * ## Design Decisions
 *
 * - **Pure functions**: No React state, easily testable
 * - **Immutable state**: Returns new state objects, never mutates
 * - **UI-friendly**: State includes fields for countdown display
 */

/**
 * Configuration for retry behavior.
 */
export interface RetryConfig {
  /** Maximum number of automatic retries before requiring manual intervention */
  maxRetries: number;
  /** Base delay in milliseconds (doubles with each attempt) */
  baseDelayMs: number;
  /** Maximum delay cap in milliseconds */
  maxDelayMs: number;
}

/**
 * State tracking for retry operations.
 */
export interface RetryState {
  /** Number of retry attempts made */
  attempts: number;
  /** Whether we're in a cooldown period before next retry */
  inCooldown: boolean;
  /** When the next retry is allowed (timestamp for cooldown display) */
  nextRetryAt: number | null;
  /** Whether max retries exceeded (requires manual retry) */
  maxRetriesExceeded: boolean;
}

/**
 * Default retry configuration.
 * - 3 max retries: Enough to handle transient issues, not enough to mask persistent problems
 * - 1s base delay: Quick first retry for network hiccups
 * - 30s max delay: Long enough to let resources recover, short enough for user patience
 */
export const DEFAULT_RETRY_CONFIG: RetryConfig = {
  maxRetries: 3,
  baseDelayMs: 1000, // 1 second
  maxDelayMs: 30000, // 30 seconds
};

/**
 * Create the initial retry state.
 */
export function createInitialRetryState(): RetryState {
  return {
    attempts: 0,
    inCooldown: false,
    nextRetryAt: null,
    maxRetriesExceeded: false,
  };
}

/**
 * Calculate the delay for the next retry attempt.
 *
 * Uses exponential backoff: delay = baseDelay * 2^(attempt-1)
 * Capped at maxDelay to prevent excessive waits.
 *
 * @param attempt - The retry attempt number (1-indexed)
 * @param config - Retry configuration
 * @returns Delay in milliseconds before the next retry
 *
 * @example
 * // With default config (1s base, 30s max):
 * calculateRetryDelay(1, DEFAULT_RETRY_CONFIG) // 1000ms  (1s)
 * calculateRetryDelay(2, DEFAULT_RETRY_CONFIG) // 2000ms  (2s)
 * calculateRetryDelay(3, DEFAULT_RETRY_CONFIG) // 4000ms  (4s)
 * calculateRetryDelay(4, DEFAULT_RETRY_CONFIG) // 8000ms  (8s)
 * calculateRetryDelay(5, DEFAULT_RETRY_CONFIG) // 16000ms (16s)
 * calculateRetryDelay(6, DEFAULT_RETRY_CONFIG) // 30000ms (capped at max)
 */
export function calculateRetryDelay(attempt: number, config: RetryConfig): number {
  if (attempt <= 0) return 0;

  // Exponential backoff: baseDelay * 2^(attempt-1)
  const exponentialDelay = config.baseDelayMs * Math.pow(2, attempt - 1);

  // Cap at max delay
  return Math.min(exponentialDelay, config.maxDelayMs);
}

/**
 * Compute the next retry state after a failure.
 *
 * @param currentAttempts - Current number of attempts (before this failure)
 * @param config - Retry configuration
 * @returns New retry state with updated attempts and cooldown info
 */
export function getNextRetryState(
  currentAttempts: number,
  config: RetryConfig
): RetryState {
  const newAttempts = currentAttempts + 1;

  // Check if we've exceeded max retries
  if (newAttempts >= config.maxRetries) {
    return {
      attempts: newAttempts,
      inCooldown: false,
      nextRetryAt: null,
      maxRetriesExceeded: true,
    };
  }

  // Calculate cooldown for next retry
  const delay = calculateRetryDelay(newAttempts, config);
  const nextRetryAt = Date.now() + delay;

  return {
    attempts: newAttempts,
    inCooldown: true,
    nextRetryAt,
    maxRetriesExceeded: false,
  };
}

/**
 * Check if a retry can be attempted given current state.
 *
 * @param state - Current retry state
 * @returns true if retry is allowed (not in cooldown and max not exceeded)
 */
export function canRetry(state: RetryState): boolean {
  return !state.inCooldown && !state.maxRetriesExceeded;
}

/**
 * Get the remaining cooldown time in milliseconds.
 *
 * @param state - Current retry state
 * @returns Remaining cooldown in ms, or 0 if not in cooldown
 */
export function getRemainingCooldown(state: RetryState): number {
  if (!state.inCooldown || state.nextRetryAt === null) {
    return 0;
  }

  const remaining = state.nextRetryAt - Date.now();
  return Math.max(0, remaining);
}

/**
 * Create a success state (reset all retry tracking).
 */
export function createSuccessState(): RetryState {
  return createInitialRetryState();
}

/**
 * Create state for manual retry (resets attempt count but preserves that user initiated).
 */
export function createManualRetryState(): RetryState {
  return createInitialRetryState();
}
