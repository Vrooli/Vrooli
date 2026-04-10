// Formatting utilities for stats display

/**
 * Format a number with compact notation (1K, 1M, etc.)
 */
export function formatCompact(value: number): string {
  return new Intl.NumberFormat("en-US", {
    notation: "compact",
    compactDisplay: "short",
  }).format(value);
}

/**
 * Format milliseconds as human-readable duration
 */
export function formatDuration(ms: number): string {
  if (ms < 1000) {
    return `${Math.round(ms)}ms`;
  }
  const seconds = ms / 1000;
  if (seconds < 60) {
    return `${seconds.toFixed(1)}s`;
  }
  const minutes = seconds / 60;
  if (minutes < 60) {
    return `${minutes.toFixed(1)}m`;
  }
  const hours = minutes / 60;
  return `${hours.toFixed(1)}h`;
}

/**
 * Format a percentage (0-1 to 0-100%)
 */
export function formatPercent(value: number, decimals = 1): string {
  return `${(value * 100).toFixed(decimals)}%`;
}

/**
 * Format a number with locale-specific thousands separators
 */
export function formatNumber(value: number): string {
  return new Intl.NumberFormat("en-US").format(value);
}

/**
 * Format token count (compact for large numbers)
 */
export function formatTokens(count: number): string {
  if (count < 1000) {
    return String(count);
  }
  if (count < 1_000_000) {
    return `${(count / 1000).toFixed(1)}K`;
  }
  return `${(count / 1_000_000).toFixed(2)}M`;
}
