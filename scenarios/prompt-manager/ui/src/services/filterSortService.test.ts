/**
 * Tests for filterSortService.ts
 *
 * Covers:
 * - Individual filter functions (storage, tags, usage, rating, status)
 * - Composed filter pipeline (applyFilters)
 * - Sorting (all fields, both directions, null handling)
 * - Active filter chips (generation, removal)
 * - Utility helpers (countActiveFilters, isFilterEmpty)
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import type { Skill } from '@/types'
import type { FilterState } from '@/types/filterSort'
import { DEFAULT_FILTER_STATE } from '@/types/filterSort'
import {
  filterByStorage,
  filterByTags,
  filterByUsagePreset,
  filterByRating,
  filterByStatus,
  applyFilters,
  sortSkills,
  getActiveChips,
  removeChip,
  countActiveFilters,
  isFilterEmpty,
} from './filterSortService'

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

function skill(overrides: Partial<Skill> = {}): Skill {
  return {
    id: 'sk-1',
    name: 'Test Skill',
    description: 'desc',
    content: '# content',
    modes: [],
    tags: [],
    draft: false,
    folder: 'local',
    file: 'test.md',
    createdAt: '2025-01-01T00:00:00Z',
    updatedAt: '2025-01-01T00:00:00Z',
    usageCount: 0,
    ...overrides,
  }
}

// ---------------------------------------------------------------------------
// filterByStorage
// ---------------------------------------------------------------------------

describe('filterByStorage', () => {
  const skills = [
    skill({ id: '1', folder: 'core' }),
    skill({ id: '2', folder: 'local' }),
    skill({ id: '3', folder: 'drafts' }),
  ]

  it('returns all skills when folders is empty', () => {
    expect(filterByStorage(skills, [])).toEqual(skills)
  })

  it('returns skills matching a single folder', () => {
    const result = filterByStorage(skills, ['core'])
    expect(result.map((s) => s.id)).toEqual(['1'])
  })

  it('returns skills matching multiple folders (OR)', () => {
    const result = filterByStorage(skills, ['core', 'drafts'])
    expect(result.map((s) => s.id)).toEqual(['1', '3'])
  })

  it('returns empty for non-matching folder', () => {
    expect(filterByStorage(skills, ['nonexistent'])).toEqual([])
  })

  it('handles empty skills array', () => {
    expect(filterByStorage([], ['core'])).toEqual([])
  })
})

// ---------------------------------------------------------------------------
// filterByTags
// ---------------------------------------------------------------------------

describe('filterByTags', () => {
  const skills = [
    skill({ id: '1', tags: ['automation', 'writing'] }),
    skill({ id: '2', tags: ['coding'] }),
    skill({ id: '3', tags: ['writing', 'coding'] }),
    skill({ id: '4', tags: [] }),
  ]

  it('returns all skills when tags is empty', () => {
    expect(filterByTags(skills, [])).toEqual(skills)
  })

  it('matches skills with at least one tag (OR)', () => {
    const result = filterByTags(skills, ['writing'])
    expect(result.map((s) => s.id)).toEqual(['1', '3'])
  })

  it('matches skills with any of multiple tags', () => {
    const result = filterByTags(skills, ['automation', 'coding'])
    expect(result.map((s) => s.id)).toEqual(['1', '2', '3'])
  })

  it('excludes skills with no tags', () => {
    const result = filterByTags(skills, ['anything'])
    expect(result).toEqual([])
  })
})

// ---------------------------------------------------------------------------
// filterByUsagePreset
// ---------------------------------------------------------------------------

describe('filterByUsagePreset', () => {
  const now = new Date('2025-06-15T12:00:00Z').getTime()

  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(now)
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  const skills = [
    skill({ id: '1', usageCount: 0, lastUsed: null }),
    skill({ id: '2', usageCount: 5, lastUsed: '2025-06-14T00:00:00Z' }), // yesterday
    skill({ id: '3', usageCount: 100, lastUsed: '2025-06-01T00:00:00Z' }), // 14 days ago
    skill({ id: '4', usageCount: 50, lastUsed: '2025-06-10T00:00:00Z' }), // 5 days ago
    skill({ id: '5', usageCount: 0, lastUsed: null }),
  ]

  it('neverUsed: returns skills with usageCount === 0', () => {
    const result = filterByUsagePreset(skills, 'neverUsed')
    expect(result.map((s) => s.id)).toEqual(['1', '5'])
  })

  it('usedThisWeek: returns skills used within 7 days', () => {
    const result = filterByUsagePreset(skills, 'usedThisWeek')
    expect(result.map((s) => s.id)).toEqual(['2', '4'])
  })

  it('usedThisWeek: excludes skills with no lastUsed', () => {
    const result = filterByUsagePreset([skill({ id: '1', lastUsed: null })], 'usedThisWeek')
    expect(result).toEqual([])
  })

  it('top10: returns top 10 by usageCount descending', () => {
    const result = filterByUsagePreset(skills, 'top10')
    expect(result.map((s) => s.id)).toEqual(['3', '4', '2', '1', '5'])
  })

  it('top10: caps at 10 results', () => {
    const many = Array.from({ length: 15 }, (_, i) =>
      skill({ id: `s${i}`, usageCount: i })
    )
    const result = filterByUsagePreset(many, 'top10')
    expect(result).toHaveLength(10)
    expect(result[0]!.usageCount).toBe(14)
  })

  it('handles empty skills array', () => {
    expect(filterByUsagePreset([], 'neverUsed')).toEqual([])
  })
})

// ---------------------------------------------------------------------------
// filterByRating
// ---------------------------------------------------------------------------

describe('filterByRating', () => {
  const skills = [
    skill({ id: '1', effectivenessRating: 5 }),
    skill({ id: '2', effectivenessRating: 3 }),
    skill({ id: '3', effectivenessRating: 1 }),
    skill({ id: '4', effectivenessRating: null }),
    skill({ id: '5', effectivenessRating: undefined }),
  ]

  it('returns skills with rating >= threshold', () => {
    const result = filterByRating(skills, 3)
    expect(result.map((s) => s.id)).toEqual(['1', '2'])
  })

  it('excludes null/undefined ratings', () => {
    const result = filterByRating(skills, 1)
    expect(result.map((s) => s.id)).toEqual(['1', '2', '3'])
  })

  it('returns empty when no skills meet threshold', () => {
    expect(filterByRating(skills, 6)).toEqual([])
  })

  it('handles empty skills array', () => {
    expect(filterByRating([], 1)).toEqual([])
  })
})

// ---------------------------------------------------------------------------
// filterByStatus
// ---------------------------------------------------------------------------

describe('filterByStatus', () => {
  const skills = [
    skill({ id: '1', draft: true }),
    skill({ id: '2', draft: false }),
    skill({ id: '3', draft: true }),
  ]

  it('all: returns all skills', () => {
    expect(filterByStatus(skills, 'all')).toEqual(skills)
  })

  it('draft: returns only drafts', () => {
    const result = filterByStatus(skills, 'draft')
    expect(result.map((s) => s.id)).toEqual(['1', '3'])
  })

  it('published: returns only non-drafts', () => {
    const result = filterByStatus(skills, 'published')
    expect(result.map((s) => s.id)).toEqual(['2'])
  })
})

// ---------------------------------------------------------------------------
// applyFilters
// ---------------------------------------------------------------------------

describe('applyFilters', () => {
  const skills = [
    skill({ id: '1', folder: 'core', tags: ['automation'], draft: false, usageCount: 10, effectivenessRating: 4 }),
    skill({ id: '2', folder: 'local', tags: ['writing'], draft: true, usageCount: 0, effectivenessRating: null }),
    skill({ id: '3', folder: 'core', tags: ['automation', 'writing'], draft: false, usageCount: 5, effectivenessRating: 2 }),
  ]

  it('returns all skills with default filter state', () => {
    expect(applyFilters(skills, DEFAULT_FILTER_STATE)).toEqual(skills)
  })

  it('combines storage + tags (AND between sections)', () => {
    const state: FilterState = {
      ...DEFAULT_FILTER_STATE,
      storage: ['core'],
      tags: ['writing'],
    }
    const result = applyFilters(skills, state)
    // Only skill 3 is in core AND has writing tag
    expect(result.map((s) => s.id)).toEqual(['3'])
  })

  it('combines multiple filter dimensions', () => {
    const state: FilterState = {
      storage: ['core'],
      tags: [],
      usagePreset: null,
      minRating: 3,
      status: 'published',
    }
    const result = applyFilters(skills, state)
    // core + rating >= 3 + published → only skill 1
    expect(result.map((s) => s.id)).toEqual(['1'])
  })

  it('returns empty when filters eliminate all skills', () => {
    const state: FilterState = {
      ...DEFAULT_FILTER_STATE,
      storage: ['drafts'],
    }
    expect(applyFilters(skills, state)).toEqual([])
  })
})

// ---------------------------------------------------------------------------
// sortSkills
// ---------------------------------------------------------------------------

describe('sortSkills', () => {
  const skills = [
    skill({ id: '1', name: 'Charlie', usageCount: 5, updatedAt: '2025-03-01T00:00:00Z', lastUsed: '2025-03-01T00:00:00Z', effectivenessRating: 3 }),
    skill({ id: '2', name: 'Alpha', usageCount: 100, updatedAt: '2025-01-01T00:00:00Z', lastUsed: null, effectivenessRating: 5 }),
    skill({ id: '3', name: 'Bravo', usageCount: 50, updatedAt: '2025-02-01T00:00:00Z', lastUsed: '2025-02-15T00:00:00Z', effectivenessRating: null }),
  ]

  it('alphabetical ascending', () => {
    const result = sortSkills(skills, { field: 'alphabetical', direction: 'asc' })
    expect(result.map((s) => s.name)).toEqual(['Alpha', 'Bravo', 'Charlie'])
  })

  it('alphabetical descending', () => {
    const result = sortSkills(skills, { field: 'alphabetical', direction: 'desc' })
    expect(result.map((s) => s.name)).toEqual(['Charlie', 'Bravo', 'Alpha'])
  })

  it('mostUsed descending', () => {
    const result = sortSkills(skills, { field: 'mostUsed', direction: 'desc' })
    expect(result.map((s) => s.usageCount)).toEqual([100, 50, 5])
  })

  it('mostUsed ascending', () => {
    const result = sortSkills(skills, { field: 'mostUsed', direction: 'asc' })
    expect(result.map((s) => s.usageCount)).toEqual([5, 50, 100])
  })

  it('recentlyUsed descending — null sorts to end', () => {
    const result = sortSkills(skills, { field: 'recentlyUsed', direction: 'desc' })
    expect(result.map((s) => s.id)).toEqual(['1', '3', '2'])
  })

  it('recentlyUsed ascending — null sorts to end', () => {
    const result = sortSkills(skills, { field: 'recentlyUsed', direction: 'asc' })
    expect(result.map((s) => s.id)).toEqual(['3', '1', '2'])
  })

  it('recentlyUpdated descending', () => {
    const result = sortSkills(skills, { field: 'recentlyUpdated', direction: 'desc' })
    expect(result.map((s) => s.id)).toEqual(['1', '3', '2'])
  })

  it('rating descending — null sorts to end', () => {
    const result = sortSkills(skills, { field: 'rating', direction: 'desc' })
    expect(result.map((s) => s.id)).toEqual(['2', '1', '3'])
  })

  it('rating ascending — null sorts to end', () => {
    const result = sortSkills(skills, { field: 'rating', direction: 'asc' })
    expect(result.map((s) => s.id)).toEqual(['1', '2', '3'])
  })

  it('does not mutate the input array', () => {
    const original = [...skills]
    sortSkills(skills, { field: 'alphabetical', direction: 'asc' })
    expect(skills).toEqual(original)
  })

  it('handles empty array', () => {
    expect(sortSkills([], { field: 'alphabetical', direction: 'asc' })).toEqual([])
  })

  it('handles all-null lastUsed', () => {
    const all = [
      skill({ id: '1', lastUsed: null }),
      skill({ id: '2', lastUsed: null }),
    ]
    const result = sortSkills(all, { field: 'recentlyUsed', direction: 'desc' })
    expect(result).toHaveLength(2)
  })
})

// ---------------------------------------------------------------------------
// getActiveChips
// ---------------------------------------------------------------------------

describe('getActiveChips', () => {
  it('returns empty for default state', () => {
    expect(getActiveChips(DEFAULT_FILTER_STATE)).toEqual([])
  })

  it('generates storage chips', () => {
    const state: FilterState = { ...DEFAULT_FILTER_STATE, storage: ['core', 'local'] }
    const chips = getActiveChips(state)
    expect(chips).toEqual([
      { id: 'storage:core', category: 'storage', label: 'Core' },
      { id: 'storage:local', category: 'storage', label: 'Local' },
    ])
  })

  it('generates tag chips', () => {
    const state: FilterState = { ...DEFAULT_FILTER_STATE, tags: ['automation'] }
    const chips = getActiveChips(state)
    expect(chips).toEqual([
      { id: 'tag:automation', category: 'tag', label: 'automation' },
    ])
  })

  it('generates usage preset chip', () => {
    const state: FilterState = { ...DEFAULT_FILTER_STATE, usagePreset: 'neverUsed' }
    const chips = getActiveChips(state)
    expect(chips).toEqual([
      { id: 'usage:neverUsed', category: 'usage', label: 'Never used' },
    ])
  })

  it('generates rating chip', () => {
    const state: FilterState = { ...DEFAULT_FILTER_STATE, minRating: 3 }
    const chips = getActiveChips(state)
    expect(chips).toEqual([
      { id: 'rating:3', category: 'rating', label: '3+ stars' },
    ])
  })

  it('generates status chip', () => {
    const state: FilterState = { ...DEFAULT_FILTER_STATE, status: 'draft' }
    const chips = getActiveChips(state)
    expect(chips).toEqual([
      { id: 'status:draft', category: 'status', label: 'Draft' },
    ])
  })

  it('generates chips for all active filters', () => {
    const state: FilterState = {
      storage: ['core'],
      tags: ['a', 'b'],
      usagePreset: 'top10',
      minRating: 4,
      status: 'published',
    }
    expect(getActiveChips(state)).toHaveLength(6)
  })
})

// ---------------------------------------------------------------------------
// removeChip
// ---------------------------------------------------------------------------

describe('removeChip', () => {
  it('removes a storage chip', () => {
    const state: FilterState = { ...DEFAULT_FILTER_STATE, storage: ['core', 'local'] }
    const result = removeChip(state, 'storage:core')
    expect(result.storage).toEqual(['local'])
  })

  it('removes a tag chip', () => {
    const state: FilterState = { ...DEFAULT_FILTER_STATE, tags: ['a', 'b'] }
    const result = removeChip(state, 'tag:a')
    expect(result.tags).toEqual(['b'])
  })

  it('removes usage preset chip', () => {
    const state: FilterState = { ...DEFAULT_FILTER_STATE, usagePreset: 'neverUsed' }
    const result = removeChip(state, 'usage:neverUsed')
    expect(result.usagePreset).toBeNull()
  })

  it('removes rating chip', () => {
    const state: FilterState = { ...DEFAULT_FILTER_STATE, minRating: 3 }
    const result = removeChip(state, 'rating:3')
    expect(result.minRating).toBeNull()
  })

  it('removes status chip', () => {
    const state: FilterState = { ...DEFAULT_FILTER_STATE, status: 'draft' }
    const result = removeChip(state, 'status:draft')
    expect(result.status).toBe('all')
  })

  it('returns unchanged state for unknown chip', () => {
    const result = removeChip(DEFAULT_FILTER_STATE, 'unknown:foo')
    expect(result).toEqual(DEFAULT_FILTER_STATE)
  })

  it('does not mutate the input state', () => {
    const state: FilterState = { ...DEFAULT_FILTER_STATE, tags: ['a', 'b'] }
    const original = { ...state, tags: [...state.tags] }
    removeChip(state, 'tag:a')
    expect(state).toEqual(original)
  })
})

// ---------------------------------------------------------------------------
// countActiveFilters
// ---------------------------------------------------------------------------

describe('countActiveFilters', () => {
  it('returns 0 for default state', () => {
    expect(countActiveFilters(DEFAULT_FILTER_STATE)).toBe(0)
  })

  it('counts storage entries individually', () => {
    expect(countActiveFilters({ ...DEFAULT_FILTER_STATE, storage: ['core', 'local'] })).toBe(2)
  })

  it('counts all dimensions', () => {
    const state: FilterState = {
      storage: ['core'],
      tags: ['a', 'b'],
      usagePreset: 'top10',
      minRating: 3,
      status: 'draft',
    }
    expect(countActiveFilters(state)).toBe(6)
  })
})

// ---------------------------------------------------------------------------
// isFilterEmpty
// ---------------------------------------------------------------------------

describe('isFilterEmpty', () => {
  it('returns true for default state', () => {
    expect(isFilterEmpty(DEFAULT_FILTER_STATE)).toBe(true)
  })

  it('returns false when any filter is active', () => {
    expect(isFilterEmpty({ ...DEFAULT_FILTER_STATE, tags: ['x'] })).toBe(false)
    expect(isFilterEmpty({ ...DEFAULT_FILTER_STATE, minRating: 1 })).toBe(false)
    expect(isFilterEmpty({ ...DEFAULT_FILTER_STATE, status: 'draft' })).toBe(false)
  })
})
