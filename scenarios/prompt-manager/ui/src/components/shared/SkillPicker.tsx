/**
 * SkillPicker — Reusable multi-select skill picker with search, filter, sort, and view modes.
 *
 * Composes existing sidebar components (SkillListView, SkillCardView, FilterSortToolbar)
 * in a "permanently combine mode" configuration for skill selection use cases.
 *
 * Used by: TopicEditorPanel (skill assignment)
 */

import { useState, useMemo, useCallback } from 'react'
import { Search } from 'lucide-react'
import { cn } from '@/lib/utils'
import { SkillListView } from '@/components/sidebar/SkillListView'
import { SkillCardView } from '@/components/sidebar/SkillCardView'
import { FilterSortToolbar } from '@/components/sidebar/FilterSortToolbar'
import { applyFilters, sortSkills } from '@/services/filterSortService'
import type { Skill } from '@/types'
import type { ViewMode, DetailMode, FilterState, SortConfig } from '@/types/filterSort'
import { DEFAULT_FILTER_STATE, DEFAULT_SORT_CONFIG, DEFAULT_DETAIL_MODE } from '@/types/filterSort'

interface SkillPickerProps {
  /** All available skills to pick from */
  skills: Skill[]
  /** Currently selected skill IDs */
  selectedIds: Set<string>
  /** Toggle a skill's selection on/off */
  onToggle: (skillId: string) => void
  /** Available tags for the tag filter */
  availableTags?: string[]
  className?: string
}

const EMPTY_SET = new Set<string>()

export function SkillPicker({
  skills,
  selectedIds,
  onToggle,
  availableTags = [],
  className,
}: SkillPickerProps) {
  const [searchQuery, setSearchQuery] = useState('')
  const [viewMode, setViewMode] = useState<ViewMode>('list')
  const [detailMode, setDetailMode] = useState<DetailMode>(DEFAULT_DETAIL_MODE)
  const [filterState, setFilterState] = useState<FilterState>(DEFAULT_FILTER_STATE)
  const [sortConfig, setSortConfig] = useState<SortConfig>(DEFAULT_SORT_CONFIG)

  // No tree view in picker — fall back to list
  const handleViewModeChange = useCallback((mode: ViewMode) => {
    setViewMode(mode === 'tree' ? 'list' : mode)
  }, [])

  // Filter and sort skills
  const filteredSkills = useMemo(() => {
    let result = skills

    // Text search
    if (searchQuery) {
      const q = searchQuery.toLowerCase()
      result = result.filter((s) =>
        s.name.toLowerCase().includes(q) ||
        (s.description && s.description.toLowerCase().includes(q)),
      )
    }

    // Apply structured filters (tags, storage, etc.)
    result = applyFilters(result, filterState)

    // Sort
    result = sortSkills(result, sortConfig)

    return result
  }, [skills, searchQuery, filterState, sortConfig])

  // Noop handler for required SkillListView/SkillCardView props (unused in combine mode)
  const noop = useCallback(() => {}, [])

  return (
    <div className={cn('flex flex-col border border-border rounded-md overflow-hidden', className)}>
      {/* Search input */}
      <div className="flex items-center gap-2 px-2 py-1.5 border-b border-border">
        <Search className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          placeholder="Search skills..."
          className="flex-1 text-sm bg-transparent outline-none placeholder:text-muted-foreground"
        />
      </div>

      {/* Filter/Sort/View toolbar */}
      <div className="border-b border-border px-1 py-0.5">
        <FilterSortToolbar
          filterState={filterState}
          onFilterStateChange={setFilterState}
          sortConfig={sortConfig}
          onSortConfigChange={setSortConfig}
          viewMode={viewMode}
          onViewModeChange={handleViewModeChange}
          detailMode={detailMode}
          onDetailModeChange={setDetailMode}
          availableTags={availableTags}
          availableFolders={[]}
        />
      </div>

      {/* Skill list/card view (always in combine mode) */}
      <div className="flex-1 max-h-64 overflow-y-auto">
        {viewMode === 'card' ? (
          <SkillCardView
            skills={filteredSkills}
            selectedItemId={null}
            onSelectItem={noop}
            dirtyItemIds={EMPTY_SET}
            detailMode={detailMode}
            combineMode
            combineSelectedIds={selectedIds}
            onCombineToggleSkill={onToggle}
          />
        ) : (
          <SkillListView
            skills={filteredSkills}
            selectedItemId={null}
            onSelectItem={noop}
            dirtyItemIds={EMPTY_SET}
            detailMode={detailMode}
            combineMode
            combineSelectedIds={selectedIds}
            onCombineToggleSkill={onToggle}
          />
        )}
      </div>

      {/* Footer: selection count */}
      <div className="px-2 py-1 border-t border-border text-xs text-muted-foreground">
        {selectedIds.size} of {skills.length} selected
      </div>
    </div>
  )
}
