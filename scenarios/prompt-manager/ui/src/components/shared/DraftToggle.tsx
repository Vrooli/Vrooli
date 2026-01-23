/**
 * DraftToggle - Clickable badge for toggling draft status.
 *
 * When draft: amber "Draft" badge
 * When published: subtle or hidden
 * Click to toggle
 */

import { cn } from '@/lib/utils'

interface DraftToggleProps {
  isDraft: boolean
  onChange: (isDraft: boolean) => void
  disabled?: boolean
  className?: string
  showWhenPublished?: boolean
}

/**
 * Draft toggle badge component.
 */
export function DraftToggle({
  isDraft,
  onChange,
  disabled,
  className,
  showWhenPublished = false,
}: DraftToggleProps) {
  // Hide when published unless explicitly shown
  if (!isDraft && !showWhenPublished) {
    return null
  }

  return (
    <button
      type="button"
      onClick={() => !disabled && onChange(!isDraft)}
      disabled={disabled}
      title={isDraft ? 'Click to publish' : 'Click to mark as draft'}
      className={cn(
        'px-2 py-0.5 rounded text-xs font-medium transition-colors',
        'focus:outline-none focus:ring-2 focus:ring-offset-1',
        isDraft
          ? 'bg-amber-500/20 text-amber-300 hover:bg-amber-500/30 focus:ring-amber-500/50'
          : 'bg-emerald-500/20 text-emerald-300 hover:bg-emerald-500/30 focus:ring-emerald-500/50',
        disabled && 'opacity-50 cursor-not-allowed',
        className
      )}
    >
      {isDraft ? 'Draft' : 'Published'}
    </button>
  )
}
