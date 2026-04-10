/**
 * Formatting Utilities
 *
 * Core formatting utilities that are pure, generic, and reusable across the application.
 * These utilities are framework-agnostic and have no external dependencies.
 *
 * IMPORTANT: Keep this module pure - no React, no app-level imports, no side effects.
 */

/**
 * Extracts the file extension from a filename, lowercased.
 *
 * @param fileName - The file name to extract extension from
 * @returns The lowercase file extension without the dot, or empty string if none
 *
 * @example
 * getFileExtension("document.pdf")     // "pdf"
 * getFileExtension("script.test.ts")   // "ts"
 * getFileExtension("README")           // ""
 * getFileExtension(".gitignore")       // "gitignore"
 */
export function getFileExtension(fileName: string): string {
  if (!fileName) return "";
  const ext = fileName.split(".").pop();
  // Handle edge case where file starts with dot (e.g., .gitignore)
  // If there's only one part after split, and it equals the original minus leading dot, it's an extension
  if (ext === fileName || ext === undefined) return "";
  return ext.toLowerCase();
}

/**
 * Formats a file size in bytes to a human-readable string.
 *
 * @param bytes - File size in bytes
 * @returns Formatted string like "1.5 KB" or "0 B"
 *
 * @example
 * formatFileSize(0)        // "0 B"
 * formatFileSize(1024)     // "1 KB"
 * formatFileSize(1536)     // "1.5 KB"
 * formatFileSize(1048576)  // "1 MB"
 */
export function formatFileSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  const size = bytes / Math.pow(1024, i);
  return `${size.toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

/**
 * Capitalizes the first letter of a string.
 *
 * @param text - String to capitalize
 * @returns String with first letter capitalized
 *
 * @example
 * capitalize("running")   // "Running"
 * capitalize("stopped")   // "Stopped"
 * capitalize("")          // ""
 */
export function capitalize(text: string): string {
  if (!text) return text;
  return text.charAt(0).toUpperCase() + text.slice(1);
}

/**
 * Formats a snake_case or kebab-case string for display by replacing
 * separators with spaces and capitalizing the first letter.
 *
 * @param text - Text with underscores or hyphens
 * @returns Formatted text with spaces
 *
 * @example
 * formatDisplayText("in_progress")   // "In progress"
 * formatDisplayText("ready")         // "Ready"
 * formatDisplayText("high-priority") // "High priority"
 */
export function formatDisplayText(text: string): string {
  if (!text) return text;
  const formatted = text.replace(/[_-]/g, " ");
  return capitalize(formatted);
}

/**
 * Formats a date as a relative time string (e.g., "2 days ago", "just now").
 * Falls back to short date format for dates older than 30 days.
 *
 * Experience Architecture (Phase 29): Helps returning users quickly understand
 * recency of items without mental date math.
 *
 * @param date - Date string or Date object to format (null/undefined yields "Unknown")
 * @param now - Optional "now" reference (for testing)
 * @returns Relative time string or short date
 *
 * @example
 * formatRelativeTime(new Date())                    // "just now"
 * formatRelativeTime(new Date(Date.now() - 3600000)) // "1 hour ago"
 * formatRelativeTime(new Date(Date.now() - 86400000 * 2)) // "2 days ago"
 * formatRelativeTime("2024-01-01")                   // "Jan 1, 2024"
 */
export function formatRelativeTime(date: Date | string | null | undefined, now?: Date): string {
  if (!date) {
    return "Unknown";
  }
  const targetDate = typeof date === "string" ? new Date(date) : date;
  if (!(targetDate instanceof Date)) {
    return "Unknown";
  }
  const reference = now ?? new Date();

  // Handle invalid dates
  if (isNaN(targetDate.getTime())) {
    return "Unknown";
  }

  const diffMs = reference.getTime() - targetDate.getTime();
  const diffSeconds = Math.floor(diffMs / 1000);
  const diffMinutes = Math.floor(diffSeconds / 60);
  const diffHours = Math.floor(diffMinutes / 60);
  const diffDays = Math.floor(diffHours / 24);

  // Future dates (shouldn't happen in this app, but handle gracefully)
  if (diffMs < 0) {
    return targetDate.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" });
  }

  // Just now (within 60 seconds)
  if (diffSeconds < 60) {
    return "just now";
  }

  // Minutes ago
  if (diffMinutes < 60) {
    return diffMinutes === 1 ? "1 minute ago" : `${diffMinutes} minutes ago`;
  }

  // Hours ago
  if (diffHours < 24) {
    return diffHours === 1 ? "1 hour ago" : `${diffHours} hours ago`;
  }

  // Days ago (up to 30 days)
  if (diffDays <= 30) {
    return diffDays === 1 ? "1 day ago" : `${diffDays} days ago`;
  }

  // Older than 30 days: show short date
  return targetDate.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" });
}
