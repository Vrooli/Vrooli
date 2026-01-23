/**
 * TagFilterPopover - Popover for selecting tag filters.
 *
 * Multi-select list of all available tags with quick search.
 */

import { useState, useRef, useEffect } from 'react'
import { Search, Check } from 'lucide-react'
import { cn } from '@/lib/utils'

interface TagFilterPopoverProps {
  /** All available tags */
  availableTags: string[]
  /** Currently selected tags */
  selectedTags: string[]
  /** Whether the popover is open */
  isOpen: boolean
  /** Callback when popover should close */
  onClose: () => void
  /** Callback when tags are applied */
  onApply: (tags: string[]) => void
  /** Anchor element for positioning (optional) */
  anchorRef?: React.RefObject<HTMLElement>
  className?: string
}

/**
 * Tag filter popover with multi-select and search.
 */
export function TagFilterPopover({
  availableTags,
  selectedTags,
  isOpen,
  onClose,
  onApply,
  className,
}: TagFilterPopoverProps) {
  // Local state for pending selections
  const [pendingTags, setPendingTags] = useState<Set<string>>(new Set(selectedTags))
  const [searchQuery, setSearchQuery] = useState('')
  const popoverRef = useRef<HTMLDivElement>(null)
  const searchInputRef = useRef<HTMLInputElement>(null)

  // Reset pending tags when popover opens
  useEffect(() => {
    if (isOpen) {
      setPendingTags(new Set(selectedTags))
      setSearchQuery('')
      // Focus search input
      setTimeout(() => searchInputRef.current?.focus(), 50)
    }
  }, [isOpen, selectedTags])

  // Handle click outside to close
  useEffect(() => {
    if (!isOpen) return

    const handleClickOutside = (event: MouseEvent) => {
      if (popoverRef.current && !popoverRef.current.contains(event.target as Node)) {
        onClose()
      }
    }

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose()
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    document.addEventListener('keydown', handleEscape)
    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
      document.removeEventListener('keydown', handleEscape)
    }
  }, [isOpen, onClose])

  // Filter tags by search query
  const filteredTags = searchQuery
    ? availableTags.filter((tag) =>
        tag.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : availableTags

  // Toggle a tag in the pending selection
  const toggleTag = (tag: string) => {
    setPendingTags((prev) => {
      const next = new Set(prev)
      if (next.has(tag)) {
        next.delete(tag)
      } else {
        next.add(tag)
      }
      return next
    })
  }

  // Apply selections and close
  const handleApply = () => {
    onApply(Array.from(pendingTags))
    onClose()
  }

  // Clear all selections
  const handleClearAll = () => {
    setPendingTags(new Set())
  }

  if (!isOpen) return null

  return (
    <div
      ref={popoverRef}
      className={cn(
        'absolute z-50 mt-1 w-56',
        'bg-popover border border-border rounded-lg shadow-lg',
        'animate-in fade-in-0 zoom-in-95 duration-100',
        className
      )}
    >
      {/* Search */}
      <div className="p-2 border-b border-border">
        <div className="relative">
          <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
          <input
            ref={searchInputRef}
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search tags..."
            className={cn(
              'w-full pl-7 pr-2 py-1.5 text-xs',
              'bg-muted border border-border rounded',
              'text-foreground placeholder:text-muted-foreground',
              'focus:outline-none focus:ring-1 focus:ring-primary'
            )}
          />
        </div>
      </div>

      {/* Tag list */}
      <div className="max-h-48 overflow-y-auto p-1">
        {filteredTags.length === 0 ? (
          <div className="px-2 py-4 text-center text-xs text-muted-foreground">
            {searchQuery ? 'No tags match your search' : 'No tags available'}
          </div>
        ) : (
          filteredTags.map((tag) => {
            const isSelected = pendingTags.has(tag)
            return (
              <button
                key={tag}
                type="button"
                onClick={() => toggleTag(tag)}
                className={cn(
                  'w-full flex items-center gap-2 px-2 py-1.5 text-xs text-left',
                  'rounded hover:bg-muted transition-colors',
                  isSelected && 'bg-primary/10'
                )}
              >
                <div
                  className={cn(
                    'w-4 h-4 rounded border flex items-center justify-center flex-shrink-0 transition-colors',
                    isSelected
                      ? 'bg-primary border-primary'
                      : 'border-muted-foreground/30'
                  )}
                >
                  {isSelected && <Check className="h-3 w-3 text-primary-foreground" />}
                </div>
                <span className="truncate">{tag}</span>
              </button>
            )
          })
        )}
      </div>

      {/* Actions */}
      <div className="flex items-center justify-between p-2 border-t border-border">
        <button
          type="button"
          onClick={handleClearAll}
          disabled={pendingTags.size === 0}
          className={cn(
            'text-[10px] text-muted-foreground hover:text-foreground transition-colors',
            pendingTags.size === 0 && 'opacity-50 cursor-not-allowed'
          )}
        >
          Clear All
        </button>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={onClose}
            className="px-2 py-1 text-[10px] text-muted-foreground hover:text-foreground transition-colors"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={handleApply}
            className={cn(
              'px-3 py-1 text-[10px] rounded',
              'bg-primary text-primary-foreground',
              'hover:bg-primary/90 transition-colors'
            )}
          >
            Apply
          </button>
        </div>
      </div>
    </div>
  )
}
