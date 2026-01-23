/**
 * Tests for skillService.ts
 *
 * Tests cover:
 * - Caching behavior
 * - Cache invalidation on mutations
 * - Batch updates
 * - Search functionality
 * - Helper utilities
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
  getSkills,
  getSkill,
  createSkill,
  updateSkill,
  updateSkills,
  deleteSkill,
  searchSkills,
  getAllTags,
  invalidateCache,
} from './skillService'
import { api } from '@/lib/api'
import type { Skill, CreateSkillRequest, UpdateSkillRequest } from '@/types'

// Mock the API module
vi.mock('@/lib/api', () => ({
  api: {
    getSkills: vi.fn(),
    getSkill: vi.fn(),
    createSkill: vi.fn(),
    updateSkill: vi.fn(),
    deleteSkill: vi.fn(),
    searchSkills: vi.fn(),
  },
}))

// Helper to create a minimal skill for testing
function createTestSkill(overrides: Partial<Skill> = {}): Skill {
  return {
    id: 'test-1',
    name: 'Test Skill',
    description: 'A test description',
    content: '# Test content',
    modes: [],
    tags: ['tag1', 'tag2'],
    draft: false,
    folder: 'local',
    file: 'test-skill.md',
    createdAt: '2025-01-01T00:00:00Z',
    updatedAt: '2025-01-01T00:00:00Z',
    usageCount: 0,
    ...overrides,
  }
}

describe('skillService', () => {
  beforeEach(() => {
    // Clear all mocks
    vi.clearAllMocks()
    // Reset cache before each test
    invalidateCache()
    // Reset time mocks
    vi.useRealTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('getSkills', () => {
    it('should fetch skills from API on first call', async () => {
      const mockSkills = [createTestSkill({ id: '1' }), createTestSkill({ id: '2' })]
      vi.mocked(api.getSkills).mockResolvedValue(mockSkills)

      const result = await getSkills()

      expect(api.getSkills).toHaveBeenCalledTimes(1)
      expect(result).toEqual(mockSkills)
    })

    it('should return cached data on subsequent calls within TTL', async () => {
      const mockSkills = [createTestSkill({ id: '1' })]
      vi.mocked(api.getSkills).mockResolvedValue(mockSkills)

      // First call
      await getSkills()
      // Second call
      const result = await getSkills()

      expect(api.getSkills).toHaveBeenCalledTimes(1)
      expect(result).toEqual(mockSkills)
    })

    it('should refetch after cache TTL expires', async () => {
      vi.useFakeTimers()
      const mockSkills = [createTestSkill({ id: '1' })]
      vi.mocked(api.getSkills).mockResolvedValue(mockSkills)

      // First call
      await getSkills()

      // Advance time past cache TTL (5 seconds)
      vi.advanceTimersByTime(6000)

      // Second call should fetch fresh data
      await getSkills()

      expect(api.getSkills).toHaveBeenCalledTimes(2)
    })

    it('should bypass cache when forceRefresh is true', async () => {
      const mockSkills = [createTestSkill({ id: '1' })]
      vi.mocked(api.getSkills).mockResolvedValue(mockSkills)

      // First call
      await getSkills()
      // Force refresh
      await getSkills(true)

      expect(api.getSkills).toHaveBeenCalledTimes(2)
    })
  })

  describe('getSkill', () => {
    it('should return cached skill if available', async () => {
      const mockSkills = [
        createTestSkill({ id: '1', name: 'Skill 1' }),
        createTestSkill({ id: '2', name: 'Skill 2' }),
      ]
      vi.mocked(api.getSkills).mockResolvedValue(mockSkills)

      // Populate cache
      await getSkills()

      // Get single skill - should use cache
      const result = await getSkill('1')

      expect(api.getSkill).not.toHaveBeenCalled()
      expect(result?.name).toBe('Skill 1')
    })

    it('should fetch from API if not in cache', async () => {
      const mockSkill = createTestSkill({ id: '1', name: 'Fetched Skill' })
      vi.mocked(api.getSkill).mockResolvedValue(mockSkill)

      const result = await getSkill('1')

      expect(api.getSkill).toHaveBeenCalledWith('1')
      expect(result?.name).toBe('Fetched Skill')
    })

    it('should return undefined and log error on API failure', async () => {
      const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
      vi.mocked(api.getSkill).mockRejectedValue(new Error('API Error'))

      const result = await getSkill('non-existent')

      expect(result).toBeUndefined()
      expect(consoleErrorSpy).toHaveBeenCalled()
      consoleErrorSpy.mockRestore()
    })
  })

  describe('createSkill', () => {
    it('should create skill and invalidate cache', async () => {
      const request: CreateSkillRequest = {
        name: 'New Skill',
        description: 'Description',
        content: 'Content',
        folder: 'local',
      }
      const mockCreated = createTestSkill({ id: 'new-1', name: 'New Skill' })
      vi.mocked(api.createSkill).mockResolvedValue(mockCreated)
      vi.mocked(api.getSkills).mockResolvedValue([createTestSkill()])

      // Populate cache
      await getSkills()
      expect(api.getSkills).toHaveBeenCalledTimes(1)

      // Create skill
      const result = await createSkill(request)

      expect(api.createSkill).toHaveBeenCalledWith(request)
      expect(result.id).toBe('new-1')

      // Cache should be invalidated - next call should fetch
      await getSkills()
      expect(api.getSkills).toHaveBeenCalledTimes(2)
    })
  })

  describe('updateSkill', () => {
    it('should update skill and invalidate cache', async () => {
      const updates: UpdateSkillRequest = { name: 'Updated Name' }
      const mockUpdated = createTestSkill({ id: '1', name: 'Updated Name' })
      vi.mocked(api.updateSkill).mockResolvedValue(mockUpdated)
      vi.mocked(api.getSkills).mockResolvedValue([createTestSkill()])

      // Populate cache
      await getSkills()

      // Update skill
      const result = await updateSkill('1', updates)

      expect(api.updateSkill).toHaveBeenCalledWith('1', updates)
      expect(result.name).toBe('Updated Name')

      // Cache should be invalidated
      await getSkills()
      expect(api.getSkills).toHaveBeenCalledTimes(2)
    })
  })

  describe('updateSkills (batch)', () => {
    it('should update multiple skills in parallel', async () => {
      const updates = new Map<string, UpdateSkillRequest>([
        ['1', { name: 'Updated 1' }],
        ['2', { name: 'Updated 2' }],
      ])

      vi.mocked(api.updateSkill)
        .mockResolvedValueOnce(createTestSkill({ id: '1', name: 'Updated 1' }))
        .mockResolvedValueOnce(createTestSkill({ id: '2', name: 'Updated 2' }))

      const results = await updateSkills(updates)

      expect(api.updateSkill).toHaveBeenCalledTimes(2)
      expect(results.size).toBe(2)
      expect((results.get('1') as Skill).name).toBe('Updated 1')
      expect((results.get('2') as Skill).name).toBe('Updated 2')
    })

    it('should handle partial failures', async () => {
      const updates = new Map<string, UpdateSkillRequest>([
        ['1', { name: 'Updated 1' }],
        ['2', { name: 'Updated 2' }],
      ])

      vi.mocked(api.updateSkill)
        .mockResolvedValueOnce(createTestSkill({ id: '1', name: 'Updated 1' }))
        .mockRejectedValueOnce(new Error('Failed to update'))

      const results = await updateSkills(updates)

      expect(results.size).toBe(2)
      expect((results.get('1') as Skill).name).toBe('Updated 1')
      expect(results.get('2')).toBeInstanceOf(Error)
    })

    it('should invalidate cache after batch update', async () => {
      vi.mocked(api.getSkills).mockResolvedValue([createTestSkill()])
      vi.mocked(api.updateSkill).mockResolvedValue(createTestSkill())

      // Populate cache
      await getSkills()

      // Batch update
      await updateSkills(new Map([['1', { name: 'Updated' }]]))

      // Cache should be invalidated
      await getSkills()
      expect(api.getSkills).toHaveBeenCalledTimes(2)
    })
  })

  describe('deleteSkill', () => {
    it('should delete skill and invalidate cache', async () => {
      vi.mocked(api.deleteSkill).mockResolvedValue()
      vi.mocked(api.getSkills).mockResolvedValue([createTestSkill()])

      // Populate cache
      await getSkills()

      // Delete skill
      await deleteSkill('1')

      expect(api.deleteSkill).toHaveBeenCalledWith('1')

      // Cache should be invalidated
      await getSkills()
      expect(api.getSkills).toHaveBeenCalledTimes(2)
    })
  })

  describe('searchSkills', () => {
    it('should search in cached data when available', async () => {
      const mockSkills = [
        createTestSkill({ id: '1', name: 'Alpha Skill' }),
        createTestSkill({ id: '2', name: 'Beta Skill' }),
        createTestSkill({ id: '3', name: 'Gamma Skill' }),
      ]
      vi.mocked(api.getSkills).mockResolvedValue(mockSkills)

      // Populate cache
      await getSkills()

      // Search - should use cache
      const results = await searchSkills('Alpha')

      expect(api.searchSkills).not.toHaveBeenCalled()
      expect(results).toHaveLength(1)
      expect(results[0]?.name).toBe('Alpha Skill')
    })

    it('should search in description', async () => {
      const mockSkills = [
        createTestSkill({ id: '1', name: 'Skill', description: 'Search term here' }),
        createTestSkill({ id: '2', name: 'Another', description: 'Nothing matching' }),
      ]
      vi.mocked(api.getSkills).mockResolvedValue(mockSkills)
      await getSkills()

      const results = await searchSkills('search term')

      expect(results).toHaveLength(1)
      expect(results[0]?.id).toBe('1')
    })

    it('should search in content', async () => {
      const mockSkills = [
        createTestSkill({ id: '1', content: 'Contains the keyword' }),
        createTestSkill({ id: '2', content: 'No match' }),
      ]
      vi.mocked(api.getSkills).mockResolvedValue(mockSkills)
      await getSkills()

      const results = await searchSkills('keyword')

      expect(results).toHaveLength(1)
      expect(results[0]?.id).toBe('1')
    })

    it('should search in tags', async () => {
      const mockSkills = [
        createTestSkill({ id: '1', tags: ['important', 'urgent'] }),
        createTestSkill({ id: '2', tags: ['low-priority'] }),
      ]
      vi.mocked(api.getSkills).mockResolvedValue(mockSkills)
      await getSkills()

      const results = await searchSkills('urgent')

      expect(results).toHaveLength(1)
      expect(results[0]?.id).toBe('1')
    })

    it('should search in modes', async () => {
      const mockSkills = [
        createTestSkill({ id: '1', modes: ['development', 'react'] }),
        createTestSkill({ id: '2', modes: ['testing'] }),
      ]
      vi.mocked(api.getSkills).mockResolvedValue(mockSkills)
      await getSkills()

      const results = await searchSkills('react')

      expect(results).toHaveLength(1)
      expect(results[0]?.id).toBe('1')
    })

    it('should fall back to API when cache is empty', async () => {
      const mockSkills = [createTestSkill({ id: '1', name: 'Result' })]
      vi.mocked(api.searchSkills).mockResolvedValue(mockSkills)

      const results = await searchSkills('query')

      expect(api.searchSkills).toHaveBeenCalledWith('query')
      expect(results).toEqual(mockSkills)
    })

    it('should be case-insensitive', async () => {
      const mockSkills = [
        createTestSkill({ id: '1', name: 'UPPERCASE' }),
      ]
      vi.mocked(api.getSkills).mockResolvedValue(mockSkills)
      await getSkills()

      const results = await searchSkills('uppercase')

      expect(results).toHaveLength(1)
    })
  })

  describe('getAllTags', () => {
    it('should return unique sorted tags from all skills', async () => {
      const mockSkills = [
        createTestSkill({ id: '1', tags: ['zebra', 'alpha'] }),
        createTestSkill({ id: '2', tags: ['beta', 'alpha'] }),
        createTestSkill({ id: '3', tags: ['gamma'] }),
      ]
      vi.mocked(api.getSkills).mockResolvedValue(mockSkills)

      const tags = await getAllTags()

      expect(tags).toEqual(['alpha', 'beta', 'gamma', 'zebra'])
    })

    it('should return empty array when no tags exist', async () => {
      const mockSkills = [
        createTestSkill({ id: '1', tags: [] }),
        createTestSkill({ id: '2', tags: [] }),
      ]
      vi.mocked(api.getSkills).mockResolvedValue(mockSkills)

      const tags = await getAllTags()

      expect(tags).toEqual([])
    })
  })
})
