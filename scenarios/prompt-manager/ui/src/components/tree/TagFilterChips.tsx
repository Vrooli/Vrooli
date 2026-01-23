/**
 * TagFilterChips - Horizontal scrollable tag filter display.
 *
 * Shows active tag filters as removable chips below the search bar.
 * Includes an "Add Filter" button to open the filter popover.
 */

import { X, Filter, Plus } from 'lucide-react'
import { cn } from '@/lib/utils'

interface TagFilterChipsProps {
  /** Currently selected tags */
  selectedTags: string[]
  /** Callback when a tag is removed */
  onRemoveTag: (tag: string) => void
  /** Callback when "Add Filter" is clicked */
  onAddFilter: () => void
  /** Callback when all filters are cleared */
  onClearAll: () => void
  className?: string
}

/**
 * Displays active tag filters as removable chips.
 */
export function TagFilterChips({
  selectedTags,
  onRemoveTag,
  onAddFilter,
  onClearAll,
  className,
}: TagFilterChipsProps) {
  if (selectedTags.length === 0) {
    return (
      <div className={cn('flex items-center gap-1', className)}>
        <button
          type="button"
          onClick={onAddFilter}
          className={cn(
            'flex items-center gap-1 px-2 py-1 text-[10px]',
            'text-muted-foreground hover:text-foreground',
            'hover:bg-muted/50 rounded transition-colors'
          )}
          title="Filter by tags"
        >
          <Filter className="h-3 w-3" />
          Filter
        </button>
      </div>
    )
  }

  return (
    <div className={cn('flex items-center gap-1 overflow-x-auto', className)}>
      {/* Active tags */}
      {selectedTags.map((tag) => (
        <span
          key={tag}
          className={cn(
            'inline-flex items-center gap-1 px-2 py-0.5',
            'text-[10px] bg-primary/20 text-primary rounded-full',
            'whitespace-nowrap flex-shrink-0'
          )}
        >
          {tag}
          <button
            type="button"
            onClick={() => onRemoveTag(tag)}
            className="p-0.5 hover:bg-primary/30 rounded-full transition-colors"
            title={`Remove ${tag} filter`}
          >
            <X className="h-2.5 w-2.5" />
          </button>
        </span>
      ))}

      {/* Add more button */}
      <button
        type="button"
        onClick={onAddFilter}
        className={cn(
          'flex items-center gap-0.5 px-1.5 py-0.5 text-[10px]',
          'text-muted-foreground hover:text-foreground',
          'hover:bg-muted/50 rounded transition-colors',
          'whitespace-nowrap flex-shrink-0'
        )}
        title="Add tag filter"
      >
        <Plus className="h-3 w-3" />
      </button>

      {/* Clear all */}
      {selectedTags.length > 1 && (
        <button
          type="button"
          onClick={onClearAll}
          className={cn(
            'flex items-center gap-0.5 px-1.5 py-0.5 text-[10px]',
            'text-muted-foreground hover:text-destructive',
            'hover:bg-muted/50 rounded transition-colors',
            'whitespace-nowrap flex-shrink-0'
          )}
          title="Clear all filters"
        >
          Clear
        </button>
      )}
    </div>
  )
}
