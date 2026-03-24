/**
 * FilterSortToolbar — Unified horizontal bar composing Filter, Sort, View, and Combine controls.
 */

import { useState, useRef } from 'react'
import { Filter, Layers } from 'lucide-react'
import { cn } from '@/lib/utils'
import { ViewModeToggle } from './ViewModeToggle'
import { SortDropdown } from './SortDropdown'
import { FilterPopover } from './FilterPopover'
import type { FilterState, SortConfig, ViewMode, DetailMode } from '@/types/filterSort'
import { countActiveFilters } from '@/services/filterSortService'

interface FilterSortToolbarProps {
  filterState: FilterState
  onFilterStateChange: (state: FilterState) => void
  sortConfig: SortConfig
  onSortConfigChange: (config: SortConfig) => void
  viewMode: ViewMode
  onViewModeChange: (mode: ViewMode) => void
  detailMode: DetailMode
  onDetailModeChange: (mode: DetailMode) => void
  availableTags: string[]
  availableFolders: string[]
  /** Whether combine (multi-select) mode is active */
  combineMode?: boolean
  /** Toggle combine mode on/off. When undefined, the button is hidden. */
  onCombineModeToggle?: () => void
  className?: string
}

export function FilterSortToolbar({
  filterState,
  onFilterStateChange,
  sortConfig,
  onSortConfigChange,
  viewMode,
  onViewModeChange,
  detailMode,
  onDetailModeChange,
  availableTags,
  availableFolders,
  combineMode = false,
  onCombineModeToggle,
  className,
}: FilterSortToolbarProps) {
  const [isFilterOpen, setIsFilterOpen] = useState(false)
  const filterButtonRef = useRef<HTMLButtonElement>(null)

  const activeCount = countActiveFilters(filterState)

  return (
    <div className={cn('flex items-center justify-between gap-2', className)}>
      {/* Left: Filter + Sort + Combine */}
      <div className="flex items-center gap-1.5">
        {/* Filter button */}
        <button
          ref={filterButtonRef}
          type="button"
          onClick={() => setIsFilterOpen(!isFilterOpen)}
          className={cn(
            'flex items-center gap-1 px-1.5 py-1 text-[10px] rounded border transition-colors',
            isFilterOpen || activeCount > 0
              ? 'bg-primary/10 text-primary border-primary/40'
              : 'text-muted-foreground border-border hover:text-foreground hover:bg-muted/50'
          )}
          aria-label="Filter skills"
          aria-expanded={isFilterOpen}
          data-testid="filter-trigger"
        >
          <Filter className="h-3 w-3" />
          <span>Filter</span>
          {activeCount > 0 && (
            <span className="px-1 py-0.5 text-[9px] font-semibold bg-primary text-primary-foreground rounded-full leading-none min-w-[14px] text-center">
              {activeCount}
            </span>
          )}
        </button>

        <FilterPopover
          isOpen={isFilterOpen}
          onClose={() => setIsFilterOpen(false)}
          onApply={onFilterStateChange}
          filterState={filterState}
          availableTags={availableTags}
          availableFolders={availableFolders}
          anchorRef={filterButtonRef}
        />

        {/* Sort dropdown — hidden in tree view */}
        {viewMode !== 'tree' && (
          <SortDropdown
            sortConfig={sortConfig}
            onSortConfigChange={onSortConfigChange}
          />
        )}

        {/* Combine (multi-select) toggle */}
        {onCombineModeToggle && (
          <button
            type="button"
            onClick={onCombineModeToggle}
            className={cn(
              'flex items-center gap-1 px-1.5 py-1 text-[10px] rounded border transition-colors',
              combineMode
                ? 'bg-primary/10 text-primary border-primary/40'
                : 'text-muted-foreground border-border hover:text-foreground hover:bg-muted/50'
            )}
            title={combineMode ? 'Exit combine mode' : 'Combine skills'}
            data-testid="combine-mode-toggle"
          >
            <Layers className="h-3 w-3" />
            <span>Select</span>
          </button>
        )}
      </div>

      {/* Right: View mode toggle */}
      <ViewModeToggle
        viewMode={viewMode}
        onViewModeChange={onViewModeChange}
        detailMode={detailMode}
        onDetailModeChange={onDetailModeChange}
      />
    </div>
  )
}
