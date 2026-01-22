/**
 * Tests for promptService.ts
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
  getPrompts,
  getPrompt,
  createPrompt,
  updatePrompt,
  updatePrompts,
  deletePrompt,
  searchPrompts,
  getAllTags,
  invalidateCache,
} from './promptService'
import { api } from '@/lib/api'
import type { Prompt, CreatePromptRequest, UpdatePromptRequest } from '@/types'

// Mock the API module
vi.mock('@/lib/api', () => ({
  api: {
    getPrompts: vi.fn(),
    getPrompt: vi.fn(),
    createPrompt: vi.fn(),
    updatePrompt: vi.fn(),
    deletePrompt: vi.fn(),
    searchPrompts: vi.fn(),
  },
}))

// Helper to create a minimal prompt for testing
function createTestPrompt(overrides: Partial<Prompt> = {}): Prompt {
  return {
    id: 'test-1',
    name: 'Test Prompt',
    description: 'A test description',
    content: '# Test content',
    modes: [],
    tags: ['tag1', 'tag2'],
    draft: false,
    folder: 'internal',
    createdAt: '2025-01-01T00:00:00Z',
    updatedAt: '2025-01-01T00:00:00Z',
    usageCount: 0,
    ...overrides,
  }
}

describe('promptService', () => {
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

  describe('getPrompts', () => {
    it('should fetch prompts from API on first call', async () => {
      const mockPrompts = [createTestPrompt({ id: '1' }), createTestPrompt({ id: '2' })]
      vi.mocked(api.getPrompts).mockResolvedValue(mockPrompts)

      const result = await getPrompts()

      expect(api.getPrompts).toHaveBeenCalledTimes(1)
      expect(result).toEqual(mockPrompts)
    })

    it('should return cached data on subsequent calls within TTL', async () => {
      const mockPrompts = [createTestPrompt({ id: '1' })]
      vi.mocked(api.getPrompts).mockResolvedValue(mockPrompts)

      // First call
      await getPrompts()
      // Second call
      const result = await getPrompts()

      expect(api.getPrompts).toHaveBeenCalledTimes(1)
      expect(result).toEqual(mockPrompts)
    })

    it('should refetch after cache TTL expires', async () => {
      vi.useFakeTimers()
      const mockPrompts = [createTestPrompt({ id: '1' })]
      vi.mocked(api.getPrompts).mockResolvedValue(mockPrompts)

      // First call
      await getPrompts()

      // Advance time past cache TTL (5 seconds)
      vi.advanceTimersByTime(6000)

      // Second call should fetch fresh data
      await getPrompts()

      expect(api.getPrompts).toHaveBeenCalledTimes(2)
    })

    it('should bypass cache when forceRefresh is true', async () => {
      const mockPrompts = [createTestPrompt({ id: '1' })]
      vi.mocked(api.getPrompts).mockResolvedValue(mockPrompts)

      // First call
      await getPrompts()
      // Force refresh
      await getPrompts(true)

      expect(api.getPrompts).toHaveBeenCalledTimes(2)
    })
  })

  describe('getPrompt', () => {
    it('should return cached prompt if available', async () => {
      const mockPrompts = [
        createTestPrompt({ id: '1', name: 'Prompt 1' }),
        createTestPrompt({ id: '2', name: 'Prompt 2' }),
      ]
      vi.mocked(api.getPrompts).mockResolvedValue(mockPrompts)

      // Populate cache
      await getPrompts()

      // Get single prompt - should use cache
      const result = await getPrompt('1')

      expect(api.getPrompt).not.toHaveBeenCalled()
      expect(result?.name).toBe('Prompt 1')
    })

    it('should fetch from API if not in cache', async () => {
      const mockPrompt = createTestPrompt({ id: '1', name: 'Fetched Prompt' })
      vi.mocked(api.getPrompt).mockResolvedValue(mockPrompt)

      const result = await getPrompt('1')

      expect(api.getPrompt).toHaveBeenCalledWith('1')
      expect(result?.name).toBe('Fetched Prompt')
    })

    it('should return undefined and log error on API failure', async () => {
      const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
      vi.mocked(api.getPrompt).mockRejectedValue(new Error('API Error'))

      const result = await getPrompt('non-existent')

      expect(result).toBeUndefined()
      expect(consoleErrorSpy).toHaveBeenCalled()
      consoleErrorSpy.mockRestore()
    })
  })

  describe('createPrompt', () => {
    it('should create prompt and invalidate cache', async () => {
      const request: CreatePromptRequest = {
        name: 'New Prompt',
        description: 'Description',
        content: 'Content',
        folder: 'internal',
      }
      const mockCreated = createTestPrompt({ id: 'new-1', name: 'New Prompt' })
      vi.mocked(api.createPrompt).mockResolvedValue(mockCreated)
      vi.mocked(api.getPrompts).mockResolvedValue([createTestPrompt()])

      // Populate cache
      await getPrompts()
      expect(api.getPrompts).toHaveBeenCalledTimes(1)

      // Create prompt
      const result = await createPrompt(request)

      expect(api.createPrompt).toHaveBeenCalledWith(request)
      expect(result.id).toBe('new-1')

      // Cache should be invalidated - next call should fetch
      await getPrompts()
      expect(api.getPrompts).toHaveBeenCalledTimes(2)
    })
  })

  describe('updatePrompt', () => {
    it('should update prompt and invalidate cache', async () => {
      const updates: UpdatePromptRequest = { name: 'Updated Name' }
      const mockUpdated = createTestPrompt({ id: '1', name: 'Updated Name' })
      vi.mocked(api.updatePrompt).mockResolvedValue(mockUpdated)
      vi.mocked(api.getPrompts).mockResolvedValue([createTestPrompt()])

      // Populate cache
      await getPrompts()

      // Update prompt
      const result = await updatePrompt('1', updates)

      expect(api.updatePrompt).toHaveBeenCalledWith('1', updates)
      expect(result.name).toBe('Updated Name')

      // Cache should be invalidated
      await getPrompts()
      expect(api.getPrompts).toHaveBeenCalledTimes(2)
    })
  })

  describe('updatePrompts (batch)', () => {
    it('should update multiple prompts in parallel', async () => {
      const updates = new Map<string, UpdatePromptRequest>([
        ['1', { name: 'Updated 1' }],
        ['2', { name: 'Updated 2' }],
      ])

      vi.mocked(api.updatePrompt)
        .mockResolvedValueOnce(createTestPrompt({ id: '1', name: 'Updated 1' }))
        .mockResolvedValueOnce(createTestPrompt({ id: '2', name: 'Updated 2' }))

      const results = await updatePrompts(updates)

      expect(api.updatePrompt).toHaveBeenCalledTimes(2)
      expect(results.size).toBe(2)
      expect((results.get('1') as Prompt).name).toBe('Updated 1')
      expect((results.get('2') as Prompt).name).toBe('Updated 2')
    })

    it('should handle partial failures', async () => {
      const updates = new Map<string, UpdatePromptRequest>([
        ['1', { name: 'Updated 1' }],
        ['2', { name: 'Updated 2' }],
      ])

      vi.mocked(api.updatePrompt)
        .mockResolvedValueOnce(createTestPrompt({ id: '1', name: 'Updated 1' }))
        .mockRejectedValueOnce(new Error('Failed to update'))

      const results = await updatePrompts(updates)

      expect(results.size).toBe(2)
      expect((results.get('1') as Prompt).name).toBe('Updated 1')
      expect(results.get('2')).toBeInstanceOf(Error)
    })

    it('should invalidate cache after batch update', async () => {
      vi.mocked(api.getPrompts).mockResolvedValue([createTestPrompt()])
      vi.mocked(api.updatePrompt).mockResolvedValue(createTestPrompt())

      // Populate cache
      await getPrompts()

      // Batch update
      await updatePrompts(new Map([['1', { name: 'Updated' }]]))

      // Cache should be invalidated
      await getPrompts()
      expect(api.getPrompts).toHaveBeenCalledTimes(2)
    })
  })

  describe('deletePrompt', () => {
    it('should delete prompt and invalidate cache', async () => {
      vi.mocked(api.deletePrompt).mockResolvedValue()
      vi.mocked(api.getPrompts).mockResolvedValue([createTestPrompt()])

      // Populate cache
      await getPrompts()

      // Delete prompt
      await deletePrompt('1')

      expect(api.deletePrompt).toHaveBeenCalledWith('1')

      // Cache should be invalidated
      await getPrompts()
      expect(api.getPrompts).toHaveBeenCalledTimes(2)
    })
  })

  describe('searchPrompts', () => {
    it('should search in cached data when available', async () => {
      const mockPrompts = [
        createTestPrompt({ id: '1', name: 'Alpha Prompt' }),
        createTestPrompt({ id: '2', name: 'Beta Prompt' }),
        createTestPrompt({ id: '3', name: 'Gamma Prompt' }),
      ]
      vi.mocked(api.getPrompts).mockResolvedValue(mockPrompts)

      // Populate cache
      await getPrompts()

      // Search - should use cache
      const results = await searchPrompts('Alpha')

      expect(api.searchPrompts).not.toHaveBeenCalled()
      expect(results).toHaveLength(1)
      expect(results[0]?.name).toBe('Alpha Prompt')
    })

    it('should search in description', async () => {
      const mockPrompts = [
        createTestPrompt({ id: '1', name: 'Prompt', description: 'Search term here' }),
        createTestPrompt({ id: '2', name: 'Another', description: 'Nothing matching' }),
      ]
      vi.mocked(api.getPrompts).mockResolvedValue(mockPrompts)
      await getPrompts()

      const results = await searchPrompts('search term')

      expect(results).toHaveLength(1)
      expect(results[0]?.id).toBe('1')
    })

    it('should search in content', async () => {
      const mockPrompts = [
        createTestPrompt({ id: '1', content: 'Contains the keyword' }),
        createTestPrompt({ id: '2', content: 'No match' }),
      ]
      vi.mocked(api.getPrompts).mockResolvedValue(mockPrompts)
      await getPrompts()

      const results = await searchPrompts('keyword')

      expect(results).toHaveLength(1)
      expect(results[0]?.id).toBe('1')
    })

    it('should search in tags', async () => {
      const mockPrompts = [
        createTestPrompt({ id: '1', tags: ['important', 'urgent'] }),
        createTestPrompt({ id: '2', tags: ['low-priority'] }),
      ]
      vi.mocked(api.getPrompts).mockResolvedValue(mockPrompts)
      await getPrompts()

      const results = await searchPrompts('urgent')

      expect(results).toHaveLength(1)
      expect(results[0]?.id).toBe('1')
    })

    it('should search in modes', async () => {
      const mockPrompts = [
        createTestPrompt({ id: '1', modes: ['development', 'react'] }),
        createTestPrompt({ id: '2', modes: ['testing'] }),
      ]
      vi.mocked(api.getPrompts).mockResolvedValue(mockPrompts)
      await getPrompts()

      const results = await searchPrompts('react')

      expect(results).toHaveLength(1)
      expect(results[0]?.id).toBe('1')
    })

    it('should fall back to API when cache is empty', async () => {
      const mockPrompts = [createTestPrompt({ id: '1', name: 'Result' })]
      vi.mocked(api.searchPrompts).mockResolvedValue(mockPrompts)

      const results = await searchPrompts('query')

      expect(api.searchPrompts).toHaveBeenCalledWith('query')
      expect(results).toEqual(mockPrompts)
    })

    it('should be case-insensitive', async () => {
      const mockPrompts = [
        createTestPrompt({ id: '1', name: 'UPPERCASE' }),
      ]
      vi.mocked(api.getPrompts).mockResolvedValue(mockPrompts)
      await getPrompts()

      const results = await searchPrompts('uppercase')

      expect(results).toHaveLength(1)
    })
  })

  describe('getAllTags', () => {
    it('should return unique sorted tags from all prompts', async () => {
      const mockPrompts = [
        createTestPrompt({ id: '1', tags: ['zebra', 'alpha'] }),
        createTestPrompt({ id: '2', tags: ['beta', 'alpha'] }),
        createTestPrompt({ id: '3', tags: ['gamma'] }),
      ]
      vi.mocked(api.getPrompts).mockResolvedValue(mockPrompts)

      const tags = await getAllTags()

      expect(tags).toEqual(['alpha', 'beta', 'gamma', 'zebra'])
    })

    it('should return empty array when no tags exist', async () => {
      const mockPrompts = [
        createTestPrompt({ id: '1', tags: [] }),
        createTestPrompt({ id: '2', tags: [] }),
      ]
      vi.mocked(api.getPrompts).mockResolvedValue(mockPrompts)

      const tags = await getAllTags()

      expect(tags).toEqual([])
    })
  })
})
