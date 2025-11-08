/**
 * Shared formatting utilities for consistent display across the application
 */

/**
 * Format a date string for display
 */
export function formatDate(dateString: string): string {
  const date = new Date(dateString)
  return date.toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

/**
 * Format file size in bytes to human-readable format (KB)
 */
export function formatFileSize(bytes: number): string {
  return (bytes / 1024).toFixed(1)
}

/**
 * Safely decode a URL-encoded route segment
 */
export function decodeRouteSegment(value?: string): string {
  if (!value) {
    return ''
  }
  try {
    return decodeURIComponent(value)
  } catch (error) {
    // Fallback to original value if decoding fails (e.g., malformed URL encoding)
    return value
  }
}

/**
 * Calculate draft metrics for display
 */
export interface DraftMetrics {
  wordCount: number
  characterCount: number
  sizeKb: number
  estimatedReadMinutes: number
}

export function calculateDraftMetrics(content: string): DraftMetrics {
  const trimmedContent = content.trim()
  const wordCount = trimmedContent ? trimmedContent.split(/\s+/).filter(Boolean).length : 0
  const characterCount = content.length
  const sizeKb = characterCount / 1024
  const estimatedReadMinutes = wordCount === 0 ? 0 : Math.max(1, Math.round(wordCount / 200))

  return {
    wordCount,
    characterCount,
    sizeKb,
    estimatedReadMinutes,
  }
}
