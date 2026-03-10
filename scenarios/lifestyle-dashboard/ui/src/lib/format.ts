/**
 * Formatting utilities for the Lifestyle Dashboard.
 * Provides consistent date/time/size formatting across components.
 *
 * Architecture tier: core (pure utilities, no framework dependencies)
 *
 * @module lib/format
 */

/**
 * Format a timestamp as relative time (e.g., "5m ago", "2h ago")
 */
export function formatRelativeTime(timestamp: string): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return "just now";
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  return `${diffDays}d ago`;
}

/**
 * Format a date for display (e.g., "Mon, Mar 9")
 */
export function formatShortDate(timestamp: string): string {
  const date = new Date(timestamp);
  return date.toLocaleDateString("en-US", {
    weekday: "short",
    month: "short",
    day: "numeric",
  });
}

/**
 * Format a date with time (e.g., "Mar 9, 2026 at 3:45 PM")
 */
export function formatDateTime(timestamp: string): string {
  const date = new Date(timestamp);
  return date.toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

/**
 * Format a date string safely (e.g., "Mar 9, 2026").
 * Returns fallback for undefined/invalid dates.
 *
 * @param dateStr - ISO date string or undefined
 * @param fallback - Value to return for invalid/missing dates (default: "-")
 */
export function formatDate(dateStr: string | undefined, fallback = "-"): string {
  if (!dateStr) return fallback;
  try {
    return new Date(dateStr).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  } catch {
    return fallback;
  }
}

/**
 * Format bytes to human-readable size (e.g., "1.5 MB")
 *
 * @param bytes - Number of bytes
 * @returns Formatted string with appropriate unit (B, KB, MB, GB)
 */
export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"] as const;
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  const idx = Math.min(i, sizes.length - 1);
  const size = sizes[idx];
  return `${(bytes / Math.pow(k, idx)).toFixed(1)} ${size}`;
}
