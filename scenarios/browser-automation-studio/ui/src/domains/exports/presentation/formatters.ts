/**
 * Export Presentation Formatters
 *
 * Pure formatting utilities for export-related UI display.
 * These functions are side-effect-free and easily testable.
 */

/**
 * Formats byte size into human-readable string.
 */
export const formatFileSize = (bytes?: number): string => {
  if (!bytes) return 'Unknown size';
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
};

/**
 * Formats duration in milliseconds to compact string (e.g., "1:30" or "45s").
 */
export const formatDuration = (ms?: number): string => {
  if (!ms) return '';
  const seconds = Math.floor(ms / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`;
};

/**
 * Formats duration in milliseconds to compact string for execution context.
 */
export const formatExecutionDuration = (ms: number): string => {
  if (ms < 1000) return '< 1s';
  if (ms < 60000) return `${Math.round(ms / 1000)}s`;
  const minutes = Math.floor(ms / 60000);
  const seconds = Math.round((ms % 60000) / 1000);
  return seconds > 0 ? `${minutes}m ${seconds}s` : `${minutes}m`;
};

/**
 * Formats a count with a noun, handling pluralization.
 */
export const formatCapturedLabel = (count: number, noun: string): string => {
  const rounded = Math.round(count);
  const suffix = rounded === 1 ? '' : 's';
  return `${rounded} ${noun}${suffix}`;
};

/**
 * Formats seconds to human-readable duration string.
 */
export const formatSeconds = (value: number): string => {
  if (value < 60) {
    return `${Math.round(value)}s`;
  }
  const minutes = Math.floor(value / 60);
  const seconds = Math.round(value % 60);
  return seconds > 0 ? `${minutes}m ${seconds}s` : `${minutes}m`;
};
