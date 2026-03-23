/**
 * Filter, sort, and view mode types for the sidebar skill browser.
 *
 * All types here are UI-only — they drive the FilterSortToolbar and
 * its sub-components without touching the API layer.
 */

// ---------------------------------------------------------------------------
// Sort
// ---------------------------------------------------------------------------

export type SortField =
  | 'alphabetical'
  | 'mostUsed'
  | 'recentlyUsed'
  | 'recentlyUpdated'
  | 'rating'

export type SortDirection = 'asc' | 'desc'

export interface SortConfig {
  field: SortField
  direction: SortDirection
}

// ---------------------------------------------------------------------------
// Filter
// ---------------------------------------------------------------------------

export type UsagePreset = 'usedThisWeek' | 'neverUsed' | 'top10'

export type StatusFilter = 'all' | 'draft' | 'published'

export interface FilterState {
  /** Storage locations to include (OR logic). Empty = all. */
  storage: string[]
  /** Tags to match (OR logic — skill must have at least one). Empty = all. */
  tags: string[]
  /** Preset usage filter. null = no usage filter. */
  usagePreset: UsagePreset | null
  /** Minimum effectiveness rating threshold (1-5). null = no rating filter. */
  minRating: number | null
  /** Draft / published / both. */
  status: StatusFilter
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

export type ViewMode = 'tree' | 'list' | 'card'

// ---------------------------------------------------------------------------
// Active filter chip (for display below toolbar)
// ---------------------------------------------------------------------------

export interface ActiveFilterChip {
  /** Unique key used for removal. */
  id: string
  /** Filter category this chip belongs to. */
  category: 'storage' | 'tag' | 'usage' | 'rating' | 'status'
  /** Human-readable label. */
  label: string
}

// ---------------------------------------------------------------------------
// Defaults
// ---------------------------------------------------------------------------

export const DEFAULT_FILTER_STATE: FilterState = {
  storage: [],
  tags: [],
  usagePreset: null,
  minRating: null,
  status: 'all',
}

export const DEFAULT_SORT_CONFIG: SortConfig = {
  field: 'alphabetical',
  direction: 'asc',
}

export const DEFAULT_VIEW_MODE: ViewMode = 'tree'
