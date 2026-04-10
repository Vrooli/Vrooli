/**
 * Rate limiting protection for pipeline creation.
 * Prevents runaway pipeline creation from bugs (e.g., infinite effect loops).
 * Uses exponential backoff: if pipelines are created too quickly, the cooldown
 * doubles each time, up to a maximum. Resets after a quiet period.
 */

const RATE_LIMIT_WINDOW_MS = 5000; // Time window for counting pipeline creations
const RATE_LIMIT_MAX_CREATIONS = 3; // Max pipelines allowed in window before throttling
const RATE_LIMIT_INITIAL_COOLDOWN_MS = 1000; // Initial cooldown when rate limited
const RATE_LIMIT_MAX_COOLDOWN_MS = 30000; // Maximum cooldown (30 seconds)
const RATE_LIMIT_RESET_AFTER_MS = 60000; // Reset backoff after 1 minute of quiet

export interface RateLimiter {
  /**
   * Check if pipeline creation is rate limited.
   * Returns null if OK to proceed, or an error message if rate limited.
   */
  checkRateLimit: () => string | null;

  /**
   * Record a pipeline creation for rate limiting purposes.
   */
  recordPipelineCreation: () => void;
}

export function createRateLimiter(): RateLimiter {
  let pipelineCreationTimestamps: number[] = [];
  let currentCooldownMs = RATE_LIMIT_INITIAL_COOLDOWN_MS;
  let cooldownUntil = 0;
  let lastCreationTime = 0;

  const checkRateLimit = (): string | null => {
    const now = Date.now();

    // Reset backoff after quiet period
    if (lastCreationTime > 0 && now - lastCreationTime > RATE_LIMIT_RESET_AFTER_MS) {
      currentCooldownMs = RATE_LIMIT_INITIAL_COOLDOWN_MS;
      pipelineCreationTimestamps = [];
    }

    // Check if we're in a cooldown period
    if (now < cooldownUntil) {
      const remainingMs = cooldownUntil - now;
      console.warn(
        `[PipelineStore] Rate limited: too many pipeline creations. ` +
        `Cooldown: ${Math.ceil(remainingMs / 1000)}s remaining. ` +
        `This usually indicates a bug causing infinite pipeline creation.`
      );
      return `Rate limited: please wait ${Math.ceil(remainingMs / 1000)} seconds before creating another pipeline`;
    }

    // Clean old timestamps outside the window
    pipelineCreationTimestamps = pipelineCreationTimestamps.filter(
      (ts) => now - ts < RATE_LIMIT_WINDOW_MS
    );

    // Check if we've exceeded the rate limit
    if (pipelineCreationTimestamps.length >= RATE_LIMIT_MAX_CREATIONS) {
      // Trigger exponential backoff
      cooldownUntil = now + currentCooldownMs;
      console.error(
        `[PipelineStore] RATE LIMIT TRIGGERED: ${pipelineCreationTimestamps.length} pipelines ` +
        `created in ${RATE_LIMIT_WINDOW_MS / 1000}s. Enforcing ${currentCooldownMs / 1000}s cooldown. ` +
        `This is likely a bug - check for infinite effect loops in React components.`
      );

      // Double the cooldown for next time (exponential backoff)
      currentCooldownMs = Math.min(currentCooldownMs * 2, RATE_LIMIT_MAX_COOLDOWN_MS);

      return `Rate limited: ${RATE_LIMIT_MAX_CREATIONS} pipelines created in ${RATE_LIMIT_WINDOW_MS / 1000}s. Please wait.`;
    }

    return null; // OK to proceed
  };

  const recordPipelineCreation = () => {
    const now = Date.now();
    pipelineCreationTimestamps.push(now);
    lastCreationTime = now;
  };

  return { checkRateLimit, recordPipelineCreation };
}
