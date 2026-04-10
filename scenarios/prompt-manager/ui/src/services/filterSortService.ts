/**
 * Filter & Sort Service — Pure functions for filtering and sorting skills.
 *
 * Zero React dependencies. Every function is a pure transform:
 * skills in → skills out (or chips, counts, booleans).
 *
 * The sidebar composes these via applyFilters() + sortSkills().
 */

import type { Skill } from '@/types'
import type {
  FilterState,
  SortConfig,
  ActiveFilterChip,
  UsagePreset,
  StatusFilter,
} from '@/types/filterSort'
import { DEFAULT_FILTER_STATE } from '@/types/filterSort'

// ---------------------------------------------------------------------------
// Individual filters (each returns a filtered Skill[])
// ---------------------------------------------------------------------------

/** Keep skills in any of the specified storage folders. */
export function filterByStorage(skills: Skill[], folders: string[]): Skill[] {
  if (folders.length === 0) return skills
  const set = new Set(folders)
  return skills.filter((s) => set.has(s.folder))
}

/** Keep skills that have at least one of the specified tags (OR logic). */
export function filterByTags(skills: Skill[], tags: string[]): Skill[] {
  if (tags.length === 0) return skills
  const set = new Set(tags)
  return skills.filter((s) => s.tags.some((t) => set.has(t)))
}

/** Keep skills matching a usage preset. */
export function filterByUsagePreset(skills: Skill[], preset: UsagePreset): Skill[] {
  switch (preset) {
    case 'neverUsed':
      return skills.filter((s) => s.usageCount === 0)
    case 'usedThisWeek': {
      const oneWeekAgo = Date.now() - 7 * 24 * 60 * 60 * 1000
      return skills.filter((s) => {
        if (!s.lastUsed) return false
        return new Date(s.lastUsed).getTime() >= oneWeekAgo
      })
    }
    case 'top10': {
      return [...skills].sort((a, b) => b.usageCount - a.usageCount).slice(0, 10)
    }
  }
}

/** Keep skills with effectivenessRating >= minRating. Null ratings are excluded. */
export function filterByRating(skills: Skill[], minRating: number): Skill[] {
  return skills.filter((s) =>
    s.effectivenessRating != null && s.effectivenessRating >= minRating
  )
}

/** Keep skills matching a status filter. */
export function filterByStatus(skills: Skill[], status: StatusFilter): Skill[] {
  switch (status) {
    case 'all':
      return skills
    case 'draft':
      return skills.filter((s) => s.draft)
    case 'published':
      return skills.filter((s) => !s.draft)
  }
}

// ---------------------------------------------------------------------------
// Composed filter pipeline
// ---------------------------------------------------------------------------

/**
 * Apply all active filters (AND between sections).
 * Short-circuits when a section is at its default (no-op) value.
 */
export function applyFilters(skills: Skill[], state: FilterState): Skill[] {
  let result = skills

  if (state.storage.length > 0) {
    result = filterByStorage(result, state.storage)
  }
  if (state.tags.length > 0) {
    result = filterByTags(result, state.tags)
  }
  if (state.usagePreset != null) {
    result = filterByUsagePreset(result, state.usagePreset)
  }
  if (state.minRating != null) {
    result = filterByRating(result, state.minRating)
  }
  if (state.status !== 'all') {
    result = filterByStatus(result, state.status)
  }

  return result
}

// ---------------------------------------------------------------------------
// Sorting
// ---------------------------------------------------------------------------

/**
 * Sort a copy of the skills array by the given config.
 * Null values for nullable fields sort to the end regardless of direction.
 */
export function sortSkills(skills: Skill[], config: SortConfig): Skill[] {
  const sorted = [...skills]
  const dir = config.direction === 'asc' ? 1 : -1

  sorted.sort((a, b) => {
    switch (config.field) {
      case 'alphabetical':
        return dir * a.name.localeCompare(b.name)

      case 'mostUsed':
        return dir * (a.usageCount - b.usageCount)

      case 'recentlyUsed': {
        const aTime = a.lastUsed ? new Date(a.lastUsed).getTime() : -Infinity
        const bTime = b.lastUsed ? new Date(b.lastUsed).getTime() : -Infinity
        if (aTime === -Infinity && bTime === -Infinity) return 0
        if (aTime === -Infinity) return 1 // a has no date → sort to end
        if (bTime === -Infinity) return -1
        return dir * (aTime - bTime)
      }

      case 'recentlyUpdated': {
        const aTime = new Date(a.updatedAt).getTime()
        const bTime = new Date(b.updatedAt).getTime()
        return dir * (aTime - bTime)
      }

      case 'rating': {
        const aRating = a.effectivenessRating ?? -Infinity
        const bRating = b.effectivenessRating ?? -Infinity
        if (aRating === -Infinity && bRating === -Infinity) return 0
        if (aRating === -Infinity) return 1
        if (bRating === -Infinity) return -1
        return dir * (aRating - bRating)
      }
    }
  })

  return sorted
}

// ---------------------------------------------------------------------------
// Active filter chips
// ---------------------------------------------------------------------------

const USAGE_PRESET_LABELS: Record<UsagePreset, string> = {
  usedThisWeek: 'Used this week',
  neverUsed: 'Never used',
  top10: 'Top 10 most used',
}

/** Build the list of removable chips representing the current filter state. */
export function getActiveChips(state: FilterState): ActiveFilterChip[] {
  const chips: ActiveFilterChip[] = []

  for (const folder of state.storage) {
    chips.push({ id: `storage:${folder}`, category: 'storage', label: capitalize(folder) })
  }
  for (const tag of state.tags) {
    chips.push({ id: `tag:${tag}`, category: 'tag', label: tag })
  }
  if (state.usagePreset != null) {
    chips.push({
      id: `usage:${state.usagePreset}`,
      category: 'usage',
      label: USAGE_PRESET_LABELS[state.usagePreset],
    })
  }
  if (state.minRating != null) {
    chips.push({
      id: `rating:${state.minRating}`,
      category: 'rating',
      label: `${state.minRating}+ stars`,
    })
  }
  if (state.status !== 'all') {
    chips.push({
      id: `status:${state.status}`,
      category: 'status',
      label: capitalize(state.status),
    })
  }

  return chips
}

/** Remove a single chip by id, returning a new FilterState. */
export function removeChip(state: FilterState, chipId: string): FilterState {
  const [category, ...rest] = chipId.split(':')
  const value = rest.join(':') // handles values that contain colons

  switch (category) {
    case 'storage':
      return { ...state, storage: state.storage.filter((f) => f !== value) }
    case 'tag':
      return { ...state, tags: state.tags.filter((t) => t !== value) }
    case 'usage':
      return { ...state, usagePreset: null }
    case 'rating':
      return { ...state, minRating: null }
    case 'status':
      return { ...state, status: 'all' }
    default:
      return state
  }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Count total active filters for badge display. */
export function countActiveFilters(state: FilterState): number {
  let count = state.storage.length + state.tags.length
  if (state.usagePreset != null) count++
  if (state.minRating != null) count++
  if (state.status !== 'all') count++
  return count
}

/** True when the filter state matches all defaults. */
export function isFilterEmpty(state: FilterState): boolean {
  return (
    state.storage.length === 0 &&
    state.tags.length === 0 &&
    state.usagePreset == null &&
    state.minRating == null &&
    state.status === DEFAULT_FILTER_STATE.status
  )
}

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1)
}
