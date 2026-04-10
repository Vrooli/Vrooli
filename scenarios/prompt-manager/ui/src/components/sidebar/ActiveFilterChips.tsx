/**
 * ActiveFilterChips — Removable chips showing active filters below the toolbar.
 */

import { X } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { FilterState } from '@/types/filterSort'
import { getActiveChips, removeChip, isFilterEmpty } from '@/services/filterSortService'
import { DEFAULT_FILTER_STATE } from '@/types/filterSort'

interface ActiveFilterChipsProps {
  filterState: FilterState
  onFilterStateChange: (state: FilterState) => void
  className?: string
}

export function ActiveFilterChips({
  filterState,
  onFilterStateChange,
  className,
}: ActiveFilterChipsProps) {
  const chips = getActiveChips(filterState)

  if (chips.length === 0) return null

  const handleRemove = (chipId: string) => {
    onFilterStateChange(removeChip(filterState, chipId))
  }

  const handleClearAll = () => {
    onFilterStateChange(DEFAULT_FILTER_STATE)
  }

  return (
    <div className={cn('flex items-center gap-1 flex-wrap', className)}>
      {chips.map((chip) => (
        <span
          key={chip.id}
          className="inline-flex items-center gap-1 px-2 py-0.5 text-[10px] font-medium bg-primary/20 text-primary rounded-full"
        >
          {chip.label}
          <button
            type="button"
            onClick={() => handleRemove(chip.id)}
            className="p-0.5 rounded-full hover:bg-primary/30 transition-colors"
            aria-label={`Remove ${chip.label} filter`}
            data-testid={`remove-chip-${chip.id}`}
          >
            <X className="h-2.5 w-2.5" />
          </button>
        </span>
      ))}
      {chips.length >= 2 && !isFilterEmpty(filterState) && (
        <button
          type="button"
          onClick={handleClearAll}
          className="text-[10px] text-muted-foreground hover:text-foreground transition-colors ml-1"
          data-testid="clear-all-filters"
        >
          Clear all
        </button>
      )}
    </div>
  )
}
