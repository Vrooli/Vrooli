/**
 * FilterPopover — Multi-section filter popover with apply/cancel pattern.
 *
 * Sections: Storage, Tags, Usage, Rating, Status.
 * Changes are buffered internally and committed on Apply.
 */

import { useState, useEffect, useRef } from 'react'
import { Search, Database, HardDrive, FileEdit, Star } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Popover } from '@/components/shared/Popover'
import { FilterPopoverSection } from './FilterPopoverSection'
import type { FilterState, UsagePreset, StatusFilter } from '@/types/filterSort'
import { DEFAULT_FILTER_STATE } from '@/types/filterSort'
import { countActiveFilters } from '@/services/filterSortService'

// ---------------------------------------------------------------------------
// Storage folder metadata
// ---------------------------------------------------------------------------

const FOLDER_INFO: Record<string, { label: string; icon: typeof Database; color: string }> = {
  core: { label: 'Core', icon: Database, color: 'text-blue-400' },
  local: { label: 'Local', icon: HardDrive, color: 'text-green-400' },
  drafts: { label: 'Drafts', icon: FileEdit, color: 'text-amber-400' },
}

// ---------------------------------------------------------------------------
// Usage presets
// ---------------------------------------------------------------------------

const USAGE_PRESETS: { value: UsagePreset; label: string }[] = [
  { value: 'usedThisWeek', label: 'Used this week' },
  { value: 'neverUsed', label: 'Never used' },
  { value: 'top10', label: 'Top 10 most used' },
]

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface FilterPopoverProps {
  isOpen: boolean
  onClose: () => void
  onApply: (state: FilterState) => void
  filterState: FilterState
  availableTags: string[]
  availableFolders: string[]
  anchorRef?: React.RefObject<HTMLElement | null>
}

export function FilterPopover({
  isOpen,
  onClose,
  onApply,
  filterState,
  availableTags,
  availableFolders,
  anchorRef,
}: FilterPopoverProps) {
  // Pending state (buffered until Apply)
  const [pending, setPending] = useState<FilterState>(filterState)
  const [tagSearch, setTagSearch] = useState('')
  const searchRef = useRef<HTMLInputElement>(null)

  // Reset pending state when popover opens
  useEffect(() => {
    if (isOpen) {
      setPending(filterState)
      setTagSearch('')
      // Auto-focus tag search after animation
      const timer = setTimeout(() => searchRef.current?.focus(), 100)
      return () => clearTimeout(timer)
    }
  }, [isOpen, filterState])

  const handleApply = () => {
    onApply(pending)
    onClose()
  }

  const handleClearAll = () => {
    setPending(DEFAULT_FILTER_STATE)
  }

  // Anchor position
  const rect = anchorRef?.current?.getBoundingClientRect()
  const x = rect?.left
  const y = rect ? rect.bottom + 4 : undefined

  const filteredTags = tagSearch
    ? availableTags.filter((t) => t.toLowerCase().includes(tagSearch.toLowerCase()))
    : availableTags

  const pendingCount = countActiveFilters(pending)

  return (
    <Popover
      isOpen={isOpen}
      onClose={onClose}
      x={x}
      y={y}
      className="w-60"
      testId="filter-popover"
    >
      <div className="max-h-[400px] overflow-y-auto">
        {/* Storage */}
        {availableFolders.length > 0 && (
          <FilterPopoverSection
            title="Storage"
            count={pending.storage.length}
          >
            <div className="flex flex-col gap-1">
              {availableFolders.map((folder) => {
                const info = FOLDER_INFO[folder]
                const isChecked = pending.storage.includes(folder)
                return (
                  <label
                    key={folder}
                    className="flex items-center gap-2 px-1 py-1 rounded cursor-pointer hover:bg-muted transition-colors"
                  >
                    <input
                      type="checkbox"
                      checked={isChecked}
                      onChange={() => {
                        setPending((prev) => ({
                          ...prev,
                          storage: isChecked
                            ? prev.storage.filter((f) => f !== folder)
                            : [...prev.storage, folder],
                        }))
                      }}
                      className="rounded border-border"
                      data-testid={`filter-storage-${folder}`}
                    />
                    {info && <info.icon className={cn('h-3.5 w-3.5', info.color)} />}
                    <span className="text-xs text-foreground">{info?.label ?? folder}</span>
                  </label>
                )
              })}
            </div>
          </FilterPopoverSection>
        )}

        {/* Tags */}
        {availableTags.length > 0 && (
          <FilterPopoverSection
            title="Tags"
            count={pending.tags.length}
          >
            <div className="flex flex-col gap-1">
              {availableTags.length > 5 && (
                <div className="relative mb-1">
                  <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-3 w-3 text-muted-foreground" />
                  <input
                    ref={searchRef}
                    type="text"
                    value={tagSearch}
                    onChange={(e) => setTagSearch(e.target.value)}
                    placeholder="Search tags..."
                    className="w-full pl-7 pr-2 py-1 text-base md:text-xs bg-muted border border-border rounded text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary"
                    data-testid="filter-tag-search"
                  />
                </div>
              )}
              <div className="max-h-[120px] overflow-y-auto flex flex-col gap-0.5">
                {filteredTags.map((tag) => {
                  const isChecked = pending.tags.includes(tag)
                  return (
                    <label
                      key={tag}
                      className="flex items-center gap-2 px-1 py-0.5 rounded cursor-pointer hover:bg-muted transition-colors"
                    >
                      <input
                        type="checkbox"
                        checked={isChecked}
                        onChange={() => {
                          setPending((prev) => ({
                            ...prev,
                            tags: isChecked
                              ? prev.tags.filter((t) => t !== tag)
                              : [...prev.tags, tag],
                          }))
                        }}
                        className="rounded border-border"
                        data-testid={`filter-tag-${tag}`}
                      />
                      <span className="text-xs text-foreground truncate">{tag}</span>
                    </label>
                  )
                })}
                {filteredTags.length === 0 && (
                  <span className="text-[10px] text-muted-foreground px-1">No tags match</span>
                )}
              </div>
            </div>
          </FilterPopoverSection>
        )}

        {/* Usage */}
        <FilterPopoverSection
          title="Usage"
          count={pending.usagePreset != null ? 1 : 0}
          defaultOpen={false}
        >
          <div className="flex flex-col gap-0.5">
            {USAGE_PRESETS.map(({ value, label }) => (
              <button
                key={value}
                type="button"
                onClick={() => {
                  setPending((prev) => ({
                    ...prev,
                    usagePreset: prev.usagePreset === value ? null : value,
                  }))
                }}
                className={cn(
                  'text-left px-2 py-1 text-xs rounded transition-colors',
                  pending.usagePreset === value
                    ? 'bg-primary/10 text-primary'
                    : 'text-foreground hover:bg-muted'
                )}
                data-testid={`filter-usage-${value}`}
              >
                {label}
              </button>
            ))}
          </div>
        </FilterPopoverSection>

        {/* Rating */}
        <FilterPopoverSection
          title="Rating"
          count={pending.minRating != null ? 1 : 0}
          defaultOpen={false}
        >
          <div className="flex items-center gap-0.5">
            {[1, 2, 3, 4, 5].map((n) => (
              <button
                key={n}
                type="button"
                onClick={() => {
                  setPending((prev) => ({
                    ...prev,
                    minRating: prev.minRating === n ? null : n,
                  }))
                }}
                className="p-0.5 transition-colors"
                aria-label={`${n}+ stars`}
                data-testid={`filter-rating-${n}`}
              >
                <Star
                  className={cn(
                    'h-4 w-4',
                    pending.minRating != null && n <= pending.minRating
                      ? 'fill-amber-400 text-amber-400'
                      : 'text-muted-foreground hover:text-amber-400'
                  )}
                />
              </button>
            ))}
            {pending.minRating != null && (
              <span className="text-[10px] text-muted-foreground ml-1">
                {pending.minRating}+
              </span>
            )}
          </div>
        </FilterPopoverSection>

        {/* Status */}
        <FilterPopoverSection
          title="Status"
          count={pending.status !== 'all' ? 1 : 0}
          defaultOpen={false}
        >
          <div className="flex items-center gap-1">
            {(['all', 'published', 'draft'] as StatusFilter[]).map((value) => (
              <button
                key={value}
                type="button"
                onClick={() => setPending((prev) => ({ ...prev, status: value }))}
                className={cn(
                  'px-2 py-1 text-xs rounded transition-colors',
                  pending.status === value
                    ? 'bg-primary/10 text-primary'
                    : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                )}
                data-testid={`filter-status-${value}`}
              >
                {value === 'all' ? 'Both' : value.charAt(0).toUpperCase() + value.slice(1)}
              </button>
            ))}
          </div>
        </FilterPopoverSection>
      </div>

      {/* Footer */}
      <div className="flex items-center justify-between px-3 py-2 border-t border-border">
        <button
          type="button"
          onClick={handleClearAll}
          className="text-[10px] text-muted-foreground hover:text-foreground transition-colors"
          disabled={pendingCount === 0}
          data-testid="filter-clear-all"
        >
          Clear all
        </button>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={onClose}
            className="px-2 py-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
            data-testid="filter-cancel"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={handleApply}
            className="px-3 py-1 text-xs bg-primary text-primary-foreground rounded hover:bg-primary/90 transition-colors"
            data-testid="filter-apply"
          >
            Apply
          </button>
        </div>
      </div>
    </Popover>
  )
}
