/**
 * Stats Formatting Utilities
 *
 * Pure formatting functions for stats display. No React, no app-level imports.
 * Companion to format-utils.ts, scoped to stats-specific formatting needs.
 */

/**
 * Formats a duration in hours to a human-readable string.
 *
 * @example
 * formatHours(0)     // "< 1 min"
 * formatHours(0.5)   // "30 min"
 * formatHours(2.5)   // "2.5 hrs"
 * formatHours(48)    // "48.0 hrs"
 */
export function formatHours(hours: number): string {
  if (hours <= 0) return "< 1 min";
  const minutes = hours * 60;
  if (minutes < 1) return "< 1 min";
  if (minutes < 60) return `${Math.round(minutes)} min`;
  return `${hours.toFixed(1)} hrs`;
}

/**
 * Formats a decimal rate (0.0–1.0) as a percentage string.
 *
 * @example
 * formatRate(0)       // "0%"
 * formatRate(0.852)   // "85.2%"
 * formatRate(1)       // "100%"
 */
export function formatRate(rate: number, decimals = 1): string {
  const pct = rate * 100;
  // Avoid showing "100.0%" or "0.0%" — use whole numbers at boundaries
  if (pct === 0 || pct === 100) return `${Math.round(pct)}%`;
  return `${pct.toFixed(decimals)}%`;
}

/**
 * Formats a signed integer with +/- prefix.
 *
 * @example
 * formatDelta(12)   // "+12"
 * formatDelta(-3)   // "-3"
 * formatDelta(0)    // "0"
 */
export function formatDelta(value: number): string {
  if (value > 0) return `+${value}`;
  return `${value}`;
}

/**
 * Computes a bar width percentage, clamped to 0–100.
 *
 * @example
 * toBarPercent(3, 10)   // 30
 * toBarPercent(15, 10)  // 100
 * toBarPercent(0, 0)    // 0
 */
export function toBarPercent(value: number, max: number): number {
  if (max <= 0) return 0;
  return Math.min(100, Math.max(0, (value / max) * 100));
}
