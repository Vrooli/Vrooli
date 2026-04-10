/**
 * Shared time formatting utilities.
 *
 * Extracted from TeamDashboardTab for reuse across dashboard and schedule components.
 */

/** Format a future Date as a relative time string (e.g. "In 2 hours"). */
export function formatRelativeTime(date: Date): string {
  const diffMs = date.getTime() - Date.now()
  if (Number.isNaN(diffMs)) return 'Unknown'
  if (diffMs < 0) return 'Overdue'
  if (diffMs < 60000) return 'In less than a minute'
  if (diffMs < 3600000) return `In ${Math.round(diffMs / 60000)} minutes`
  if (diffMs < 86400000) return `In ${Math.round(diffMs / 3600000)} hours`
  return `In ${Math.round(diffMs / 86400000)} days`
}

/** Format a past Date as a relative time string (e.g. "3 mins ago"). */
export function formatRelativePastTime(date: Date): string {
  const diffMs = Date.now() - date.getTime()
  if (Number.isNaN(diffMs) || diffMs < 0) return 'Just now'
  if (diffMs < 60000) return 'Just now'
  if (diffMs < 3600000) {
    const mins = Math.round(diffMs / 60000)
    return `${mins} min${mins !== 1 ? 's' : ''} ago`
  }
  if (diffMs < 86400000) {
    const hrs = Math.round(diffMs / 3600000)
    return `${hrs} hour${hrs !== 1 ? 's' : ''} ago`
  }
  const days = Math.round(diffMs / 86400000)
  return `${days} day${days !== 1 ? 's' : ''} ago`
}

/** Format an ISO date string to locale string. */
export function formatDate(dateString?: string): string {
  if (!dateString) return 'Unknown'
  return new Date(dateString).toLocaleString()
}

/** Format a Date into a date group label (e.g. "Today", "Yesterday", "Mar 24"). */
export function formatDateGroup(date: Date): string {
  if (Number.isNaN(date.getTime())) return 'Unknown'
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const target = new Date(date.getFullYear(), date.getMonth(), date.getDate())
  const diffDays = Math.round((today.getTime() - target.getTime()) / 86_400_000)
  if (diffDays === 0) return 'Today'
  if (diffDays === 1) return 'Yesterday'
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}

/** Format a duration in milliseconds to a compact string (e.g. "2m 14s"). */
export function formatDuration(ms: number): string {
  if (ms < 0 || Number.isNaN(ms)) return '—'
  const totalSeconds = Math.round(ms / 1000)
  if (totalSeconds < 60) return `${totalSeconds}s`
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  if (minutes < 60) return `${minutes}m ${seconds}s`
  const hours = Math.floor(minutes / 60)
  const remainingMinutes = minutes % 60
  return `${hours}h ${remainingMinutes}m`
}
