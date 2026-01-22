/**
 * Tests for useModeSuggestions hook.
 *
 * Tests cover:
 * - Getting mode suggestions at different levels
 * - Top-level modes extraction
 * - New path detection
 * - Memoization behavior
 */

import { describe, it, expect } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useModeSuggestions } from './useModeSuggestions'
import type { Prompt } from '@/types'

// Helper to create a test prompt
function createTestPrompt(overrides: Partial<Prompt> = {}): Prompt {
  return {
    id: 'test-1',
    name: 'Test Prompt',
    description: 'A test description',
    content: '# Test content',
    modes: [],
    tags: [],
    icon: 'file',
    targetToolId: 'tool-123',
    draft: false,
    folder: 'internal',
    createdAt: '2025-01-01T00:00:00Z',
    updatedAt: '2025-01-01T00:00:00Z',
    usageCount: 5,
    lastUsed: '2025-01-01T12:00:00Z',
    effectivenessRating: 4.5,
    ...overrides,
  }
}

describe('useModeSuggestions', () => {
  describe('topLevelModes', () => {
    it('should return empty array when no prompts have modes', () => {
      const prompts = [
        createTestPrompt({ id: '1', modes: [] }),
        createTestPrompt({ id: '2', modes: [] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ prompts }))

      expect(result.current.topLevelModes).toEqual([])
    })

    it('should return unique top-level modes', () => {
      const prompts = [
        createTestPrompt({ id: '1', modes: ['development'] }),
        createTestPrompt({ id: '2', modes: ['testing'] }),
        createTestPrompt({ id: '3', modes: ['development', 'backend'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ prompts }))

      expect(result.current.topLevelModes).toEqual(['development', 'testing'])
    })

    it('should sort modes alphabetically', () => {
      const prompts = [
        createTestPrompt({ id: '1', modes: ['zebra'] }),
        createTestPrompt({ id: '2', modes: ['alpha'] }),
        createTestPrompt({ id: '3', modes: ['middle'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ prompts }))

      expect(result.current.topLevelModes).toEqual(['alpha', 'middle', 'zebra'])
    })
  })

  describe('getSuggestionsAtLevel', () => {
    it('should return top-level modes for level 0', () => {
      const prompts = [
        createTestPrompt({ id: '1', modes: ['development'] }),
        createTestPrompt({ id: '2', modes: ['testing'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ prompts }))

      const suggestions = result.current.getSuggestionsAtLevel(0, [])
      expect(suggestions).toEqual(['development', 'testing'])
    })

    it('should return second-level modes for level 1 with parent path', () => {
      const prompts = [
        createTestPrompt({ id: '1', modes: ['development', 'frontend'] }),
        createTestPrompt({ id: '2', modes: ['development', 'backend'] }),
        createTestPrompt({ id: '3', modes: ['testing', 'unit'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ prompts }))

      const suggestions = result.current.getSuggestionsAtLevel(1, ['development'])
      expect(suggestions).toEqual(['backend', 'frontend'])
    })

    it('should return empty array for unmatched parent path', () => {
      const prompts = [
        createTestPrompt({ id: '1', modes: ['development', 'frontend'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ prompts }))

      const suggestions = result.current.getSuggestionsAtLevel(1, ['testing'])
      expect(suggestions).toEqual([])
    })

    it('should return third-level modes for level 2', () => {
      const prompts = [
        createTestPrompt({ id: '1', modes: ['development', 'frontend', 'react'] }),
        createTestPrompt({ id: '2', modes: ['development', 'frontend', 'vue'] }),
        createTestPrompt({ id: '3', modes: ['development', 'backend', 'node'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ prompts }))

      const suggestions = result.current.getSuggestionsAtLevel(2, ['development', 'frontend'])
      expect(suggestions).toEqual(['react', 'vue'])
    })

    it('should return empty array when no modes at level', () => {
      const prompts = [
        createTestPrompt({ id: '1', modes: ['development'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ prompts }))

      const suggestions = result.current.getSuggestionsAtLevel(1, ['development'])
      expect(suggestions).toEqual([])
    })
  })

  describe('isNewPath', () => {
    it('should return false for empty path', () => {
      const prompts = [
        createTestPrompt({ id: '1', modes: ['development'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ prompts }))

      expect(result.current.isNewPath([])).toBe(false)
    })

    it('should return false for existing path', () => {
      const prompts = [
        createTestPrompt({ id: '1', modes: ['development'] }),
        createTestPrompt({ id: '2', modes: ['development', 'frontend'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ prompts }))

      expect(result.current.isNewPath(['development'])).toBe(false)
      expect(result.current.isNewPath(['development', 'frontend'])).toBe(false)
    })

    it('should return true for new single-level path', () => {
      const prompts = [
        createTestPrompt({ id: '1', modes: ['development'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ prompts }))

      expect(result.current.isNewPath(['testing'])).toBe(true)
    })

    it('should return true for new multi-level path', () => {
      const prompts = [
        createTestPrompt({ id: '1', modes: ['development', 'frontend'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ prompts }))

      expect(result.current.isNewPath(['development', 'backend'])).toBe(true)
      expect(result.current.isNewPath(['testing', 'unit'])).toBe(true)
    })

    it('should return true when path length differs from all prompts', () => {
      const prompts = [
        createTestPrompt({ id: '1', modes: ['development'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ prompts }))

      // Different length from existing ['development']
      expect(result.current.isNewPath(['development', 'frontend'])).toBe(true)
    })

    it('should return true for subset of existing longer path', () => {
      const prompts = [
        createTestPrompt({ id: '1', modes: ['development', 'frontend', 'react'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ prompts }))

      // ['development'] would be a subset, but length differs
      expect(result.current.isNewPath(['development'])).toBe(true)
      expect(result.current.isNewPath(['development', 'frontend'])).toBe(true)
    })
  })

  describe('memoization', () => {
    it('should return same topLevelModes reference when prompts unchanged', () => {
      const prompts = [
        createTestPrompt({ id: '1', modes: ['development'] }),
      ]

      const { result, rerender } = renderHook(
        ({ prompts }: { prompts: Prompt[] }) => useModeSuggestions({ prompts }),
        { initialProps: { prompts } }
      )

      const firstResult = result.current.topLevelModes

      rerender({ prompts })

      expect(result.current.topLevelModes).toBe(firstResult)
    })

    it('should return new topLevelModes reference when prompts change', () => {
      const prompts1 = [
        createTestPrompt({ id: '1', modes: ['development'] }),
      ]
      const prompts2 = [
        createTestPrompt({ id: '1', modes: ['development'] }),
        createTestPrompt({ id: '2', modes: ['testing'] }),
      ]

      const { result, rerender } = renderHook(
        ({ prompts }: { prompts: Prompt[] }) => useModeSuggestions({ prompts }),
        { initialProps: { prompts: prompts1 } }
      )

      const firstResult = result.current.topLevelModes

      rerender({ prompts: prompts2 })

      expect(result.current.topLevelModes).not.toBe(firstResult)
      expect(result.current.topLevelModes).toEqual(['development', 'testing'])
    })

    it('should return stable getSuggestionsAtLevel function when prompts unchanged', () => {
      const prompts = [
        createTestPrompt({ id: '1', modes: ['development'] }),
      ]

      const { result, rerender } = renderHook(
        ({ prompts }: { prompts: Prompt[] }) => useModeSuggestions({ prompts }),
        { initialProps: { prompts } }
      )

      const firstFn = result.current.getSuggestionsAtLevel

      rerender({ prompts })

      expect(result.current.getSuggestionsAtLevel).toBe(firstFn)
    })

    it('should return stable isNewPath function when prompts unchanged', () => {
      const prompts = [
        createTestPrompt({ id: '1', modes: ['development'] }),
      ]

      const { result, rerender } = renderHook(
        ({ prompts }: { prompts: Prompt[] }) => useModeSuggestions({ prompts }),
        { initialProps: { prompts } }
      )

      const firstFn = result.current.isNewPath

      rerender({ prompts })

      expect(result.current.isNewPath).toBe(firstFn)
    })
  })

  describe('edge cases', () => {
    it('should handle prompts with deeply nested modes', () => {
      const prompts = [
        createTestPrompt({
          id: '1',
          modes: ['level1', 'level2', 'level3', 'level4', 'level5'],
        }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ prompts }))

      expect(result.current.getSuggestionsAtLevel(4, ['level1', 'level2', 'level3', 'level4']))
        .toEqual(['level5'])
    })

    it('should handle empty prompts array', () => {
      const { result } = renderHook(() => useModeSuggestions({ prompts: [] }))

      expect(result.current.topLevelModes).toEqual([])
      expect(result.current.getSuggestionsAtLevel(0, [])).toEqual([])
      expect(result.current.isNewPath(['anything'])).toBe(true)
    })

    it('should handle prompts with single mode', () => {
      const prompts = [
        createTestPrompt({ id: '1', modes: ['single'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ prompts }))

      expect(result.current.topLevelModes).toEqual(['single'])
      expect(result.current.getSuggestionsAtLevel(0, [])).toEqual(['single'])
      expect(result.current.getSuggestionsAtLevel(1, ['single'])).toEqual([])
      expect(result.current.isNewPath(['single'])).toBe(false)
    })

    it('should handle mixed depth mode paths', () => {
      const prompts = [
        createTestPrompt({ id: '1', modes: ['a'] }),
        createTestPrompt({ id: '2', modes: ['a', 'b'] }),
        createTestPrompt({ id: '3', modes: ['a', 'b', 'c'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ prompts }))

      // All share 'a' at level 0
      expect(result.current.getSuggestionsAtLevel(0, [])).toEqual(['a'])

      // Level 1 under 'a'
      expect(result.current.getSuggestionsAtLevel(1, ['a'])).toEqual(['b'])

      // Level 2 under 'a/b'
      expect(result.current.getSuggestionsAtLevel(2, ['a', 'b'])).toEqual(['c'])
    })
  })
})
