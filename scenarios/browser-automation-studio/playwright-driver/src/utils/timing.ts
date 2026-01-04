/**
 * Timing Utilities
 *
 * Cross-cutting timing utilities for async delays.
 * These are general-purpose utilities not specific to any domain.
 */

/**
 * Sleep for specified milliseconds.
 * Returns immediately if duration is <= 0.
 *
 * @param ms - Duration in milliseconds
 * @returns Promise that resolves after the delay
 */
export function sleep(ms: number): Promise<void> {
  if (ms <= 0) return Promise.resolve();
  return new Promise((resolve) => setTimeout(resolve, ms));
}
