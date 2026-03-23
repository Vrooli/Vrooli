/**
 * SortDropdown — Sort field selector with direction toggle.
 */

import { useState, useRef } from 'react'
import { ArrowUpDown, ArrowUp, ArrowDown } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Popover } from '@/components/shared/Popover'
import type { SortConfig, SortField, SortDirection } from '@/types/filterSort'

interface SortDropdownProps {
  sortConfig: SortConfig
  onSortConfigChange: (config: SortConfig) => void
}

const SORT_OPTIONS: { field: SortField; label: string; defaultDir: SortDirection }[] = [
  { field: 'alphabetical', label: 'A–Z', defaultDir: 'asc' },
  { field: 'mostUsed', label: 'Most used', defaultDir: 'desc' },
  { field: 'recentlyUsed', label: 'Recently used', defaultDir: 'desc' },
  { field: 'recentlyUpdated', label: 'Recently updated', defaultDir: 'desc' },
  { field: 'rating', label: 'Highest rated', defaultDir: 'desc' },
]

export function SortDropdown({ sortConfig, onSortConfigChange }: SortDropdownProps) {
  const [isOpen, setIsOpen] = useState(false)
  const buttonRef = useRef<HTMLButtonElement>(null)

  const current = SORT_OPTIONS.find((o) => o.field === sortConfig.field) ?? SORT_OPTIONS[0]!

  const handleFieldSelect = (field: SortField, defaultDir: SortDirection) => {
    onSortConfigChange({
      field,
      direction: field === sortConfig.field ? sortConfig.direction : defaultDir,
    })
    setIsOpen(false)
  }

  const toggleDirection = (e: React.MouseEvent) => {
    e.stopPropagation()
    onSortConfigChange({
      ...sortConfig,
      direction: sortConfig.direction === 'asc' ? 'desc' : 'asc',
    })
  }

  const rect = buttonRef.current?.getBoundingClientRect()
  const DirectionIcon = sortConfig.direction === 'asc' ? ArrowUp : ArrowDown

  return (
    <div className="relative">
      <button
        ref={buttonRef}
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className={cn(
          'flex items-center gap-1 px-1.5 py-1 text-[10px] rounded border transition-colors',
          isOpen
            ? 'bg-primary/10 text-primary border-primary/40'
            : 'text-muted-foreground border-border hover:text-foreground hover:bg-muted/50'
        )}
        aria-label={`Sort: ${current.label}`}
        aria-expanded={isOpen}
        data-testid="sort-dropdown-trigger"
      >
        <ArrowUpDown className="h-3 w-3 flex-shrink-0" />
        <span>{current.label}</span>
      </button>

      <Popover
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        x={rect?.left}
        y={rect ? rect.bottom + 4 : undefined}
        className="w-44"
        testId="sort-dropdown-menu"
      >
        <div className="py-1">
          {SORT_OPTIONS.map(({ field, label, defaultDir }) => (
            <button
              key={field}
              type="button"
              onClick={() => handleFieldSelect(field, defaultDir)}
              className={cn(
                'w-full text-left px-3 py-1.5 text-xs transition-colors',
                field === sortConfig.field
                  ? 'bg-primary/10 text-primary'
                  : 'text-foreground hover:bg-muted'
              )}
              data-testid={`sort-option-${field}`}
            >
              {label}
            </button>
          ))}
        </div>
        {/* Direction toggle at bottom */}
        <div className="border-t border-border px-3 py-1.5">
          <button
            type="button"
            onClick={toggleDirection}
            className="flex items-center gap-2 w-full text-xs text-muted-foreground hover:text-foreground transition-colors"
            data-testid="sort-direction-toggle"
          >
            <DirectionIcon className="h-3 w-3" />
            <span>{sortConfig.direction === 'asc' ? 'Ascending' : 'Descending'}</span>
          </button>
        </div>
      </Popover>
    </div>
  )
}
