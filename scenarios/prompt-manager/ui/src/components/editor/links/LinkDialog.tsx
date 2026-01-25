/**
 * LinkDialog - Inline dialog for editing links in the editor.
 *
 * Features:
 * - Input field for URL entry
 * - Keyboard shortcuts (Enter to save, Escape to cancel)
 * - Auto-focus on open
 */

import { Link as LinkIcon } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface LinkDialogProps {
  /** The current link URL */
  linkUrl: string
  /** Callback when URL changes */
  onLinkUrlChange: (url: string) => void
  /** Callback when link is saved */
  onSave: () => void
  /** Callback when dialog is closed */
  onClose: () => void
  /** Ref for the input element */
  inputRef: React.RefObject<HTMLInputElement>
  /** Additional CSS classes */
  className?: string
}

/**
 * Inline link dialog component.
 */
export function LinkDialog({
  linkUrl,
  onLinkUrlChange,
  onSave,
  onClose,
  inputRef,
  className,
}: LinkDialogProps) {
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      onSave()
    } else if (e.key === 'Escape') {
      onClose()
    }
  }

  return (
    <div
      className={cn(
        'flex-shrink-0 flex items-center gap-2 px-2 py-2 border-b border-border bg-muted/50',
        className
      )}
    >
      <LinkIcon className="h-4 w-4 text-muted-foreground flex-shrink-0" />
      <input
        ref={inputRef}
        type="url"
        value={linkUrl}
        onChange={(e) => onLinkUrlChange(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder="Enter URL (e.g., https://example.com)"
        className={cn(
          'flex-1 px-2 py-1 text-sm',
          'bg-muted border border-border rounded',
          'text-foreground placeholder:text-muted-foreground',
          'focus:outline-none focus:ring-2 focus:ring-primary'
        )}
      />
      <button
        type="button"
        onClick={onSave}
        className="px-3 py-1 text-sm bg-primary hover:bg-primary/90 text-primary-foreground rounded transition-colors"
      >
        Add
      </button>
      <button
        type="button"
        onClick={onClose}
        className="px-3 py-1 text-sm bg-muted hover:bg-muted/80 text-foreground rounded transition-colors"
      >
        Cancel
      </button>
    </div>
  )
}
