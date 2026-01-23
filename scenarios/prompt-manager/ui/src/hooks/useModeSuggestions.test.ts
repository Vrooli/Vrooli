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
import type { Skill } from '@/types'

// Helper to create a test skill
function createTestSkill(overrides: Partial<Skill> = {}): Skill {
  return {
    id: 'test-1',
    name: 'Test Skill',
    description: 'A test description',
    content: '# Test content',
    modes: [],
    tags: [],
    icon: 'file',
    targetToolId: 'tool-123',
    draft: false,
    folder: 'local',
    file: 'test-skill.md',
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
    it('should return empty array when no skills have modes', () => {
      const skills = [
        createTestSkill({ id: '1', modes: [] }),
        createTestSkill({ id: '2', modes: [] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ skills }))

      expect(result.current.topLevelModes).toEqual([])
    })

    it('should return unique top-level modes', () => {
      const skills = [
        createTestSkill({ id: '1', modes: ['development'] }),
        createTestSkill({ id: '2', modes: ['testing'] }),
        createTestSkill({ id: '3', modes: ['development', 'backend'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ skills }))

      expect(result.current.topLevelModes).toEqual(['development', 'testing'])
    })

    it('should sort modes alphabetically', () => {
      const skills = [
        createTestSkill({ id: '1', modes: ['zebra'] }),
        createTestSkill({ id: '2', modes: ['alpha'] }),
        createTestSkill({ id: '3', modes: ['middle'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ skills }))

      expect(result.current.topLevelModes).toEqual(['alpha', 'middle', 'zebra'])
    })
  })

  describe('getSuggestionsAtLevel', () => {
    it('should return top-level modes for level 0', () => {
      const skills = [
        createTestSkill({ id: '1', modes: ['development'] }),
        createTestSkill({ id: '2', modes: ['testing'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ skills }))

      const suggestions = result.current.getSuggestionsAtLevel(0, [])
      expect(suggestions).toEqual(['development', 'testing'])
    })

    it('should return second-level modes for level 1 with parent path', () => {
      const skills = [
        createTestSkill({ id: '1', modes: ['development', 'frontend'] }),
        createTestSkill({ id: '2', modes: ['development', 'backend'] }),
        createTestSkill({ id: '3', modes: ['testing', 'unit'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ skills }))

      const suggestions = result.current.getSuggestionsAtLevel(1, ['development'])
      expect(suggestions).toEqual(['backend', 'frontend'])
    })

    it('should return empty array for unmatched parent path', () => {
      const skills = [
        createTestSkill({ id: '1', modes: ['development', 'frontend'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ skills }))

      const suggestions = result.current.getSuggestionsAtLevel(1, ['testing'])
      expect(suggestions).toEqual([])
    })

    it('should return third-level modes for level 2', () => {
      const skills = [
        createTestSkill({ id: '1', modes: ['development', 'frontend', 'react'] }),
        createTestSkill({ id: '2', modes: ['development', 'frontend', 'vue'] }),
        createTestSkill({ id: '3', modes: ['development', 'backend', 'node'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ skills }))

      const suggestions = result.current.getSuggestionsAtLevel(2, ['development', 'frontend'])
      expect(suggestions).toEqual(['react', 'vue'])
    })

    it('should return empty array when no modes at level', () => {
      const skills = [
        createTestSkill({ id: '1', modes: ['development'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ skills }))

      const suggestions = result.current.getSuggestionsAtLevel(1, ['development'])
      expect(suggestions).toEqual([])
    })
  })

  describe('isNewPath', () => {
    it('should return false for empty path', () => {
      const skills = [
        createTestSkill({ id: '1', modes: ['development'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ skills }))

      expect(result.current.isNewPath([])).toBe(false)
    })

    it('should return false for existing path', () => {
      const skills = [
        createTestSkill({ id: '1', modes: ['development'] }),
        createTestSkill({ id: '2', modes: ['development', 'frontend'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ skills }))

      expect(result.current.isNewPath(['development'])).toBe(false)
      expect(result.current.isNewPath(['development', 'frontend'])).toBe(false)
    })

    it('should return true for new single-level path', () => {
      const skills = [
        createTestSkill({ id: '1', modes: ['development'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ skills }))

      expect(result.current.isNewPath(['testing'])).toBe(true)
    })

    it('should return true for new multi-level path', () => {
      const skills = [
        createTestSkill({ id: '1', modes: ['development', 'frontend'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ skills }))

      expect(result.current.isNewPath(['development', 'backend'])).toBe(true)
      expect(result.current.isNewPath(['testing', 'unit'])).toBe(true)
    })

    it('should return true when path length differs from all skills', () => {
      const skills = [
        createTestSkill({ id: '1', modes: ['development'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ skills }))

      // Different length from existing ['development']
      expect(result.current.isNewPath(['development', 'frontend'])).toBe(true)
    })

    it('should return true for subset of existing longer path', () => {
      const skills = [
        createTestSkill({ id: '1', modes: ['development', 'frontend', 'react'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ skills }))

      // ['development'] would be a subset, but length differs
      expect(result.current.isNewPath(['development'])).toBe(true)
      expect(result.current.isNewPath(['development', 'frontend'])).toBe(true)
    })
  })

  describe('memoization', () => {
    it('should return same topLevelModes reference when skills unchanged', () => {
      const skills = [
        createTestSkill({ id: '1', modes: ['development'] }),
      ]

      const { result, rerender } = renderHook(
        ({ skills }: { skills: Skill[] }) => useModeSuggestions({ skills }),
        { initialProps: { skills } }
      )

      const firstResult = result.current.topLevelModes

      rerender({ skills })

      expect(result.current.topLevelModes).toBe(firstResult)
    })

    it('should return new topLevelModes reference when skills change', () => {
      const skills1 = [
        createTestSkill({ id: '1', modes: ['development'] }),
      ]
      const skills2 = [
        createTestSkill({ id: '1', modes: ['development'] }),
        createTestSkill({ id: '2', modes: ['testing'] }),
      ]

      const { result, rerender } = renderHook(
        ({ skills }: { skills: Skill[] }) => useModeSuggestions({ skills }),
        { initialProps: { skills: skills1 } }
      )

      const firstResult = result.current.topLevelModes

      rerender({ skills: skills2 })

      expect(result.current.topLevelModes).not.toBe(firstResult)
      expect(result.current.topLevelModes).toEqual(['development', 'testing'])
    })

    it('should return stable getSuggestionsAtLevel function when skills unchanged', () => {
      const skills = [
        createTestSkill({ id: '1', modes: ['development'] }),
      ]

      const { result, rerender } = renderHook(
        ({ skills }: { skills: Skill[] }) => useModeSuggestions({ skills }),
        { initialProps: { skills } }
      )

      const firstFn = result.current.getSuggestionsAtLevel

      rerender({ skills })

      expect(result.current.getSuggestionsAtLevel).toBe(firstFn)
    })

    it('should return stable isNewPath function when skills unchanged', () => {
      const skills = [
        createTestSkill({ id: '1', modes: ['development'] }),
      ]

      const { result, rerender } = renderHook(
        ({ skills }: { skills: Skill[] }) => useModeSuggestions({ skills }),
        { initialProps: { skills } }
      )

      const firstFn = result.current.isNewPath

      rerender({ skills })

      expect(result.current.isNewPath).toBe(firstFn)
    })
  })

  describe('edge cases', () => {
    it('should handle skills with deeply nested modes', () => {
      const skills = [
        createTestSkill({
          id: '1',
          modes: ['level1', 'level2', 'level3', 'level4', 'level5'],
        }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ skills }))

      expect(result.current.getSuggestionsAtLevel(4, ['level1', 'level2', 'level3', 'level4']))
        .toEqual(['level5'])
    })

    it('should handle empty skills array', () => {
      const { result } = renderHook(() => useModeSuggestions({ skills: [] }))

      expect(result.current.topLevelModes).toEqual([])
      expect(result.current.getSuggestionsAtLevel(0, [])).toEqual([])
      expect(result.current.isNewPath(['anything'])).toBe(true)
    })

    it('should handle skills with single mode', () => {
      const skills = [
        createTestSkill({ id: '1', modes: ['single'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ skills }))

      expect(result.current.topLevelModes).toEqual(['single'])
      expect(result.current.getSuggestionsAtLevel(0, [])).toEqual(['single'])
      expect(result.current.getSuggestionsAtLevel(1, ['single'])).toEqual([])
      expect(result.current.isNewPath(['single'])).toBe(false)
    })

    it('should handle mixed depth mode paths', () => {
      const skills = [
        createTestSkill({ id: '1', modes: ['a'] }),
        createTestSkill({ id: '2', modes: ['a', 'b'] }),
        createTestSkill({ id: '3', modes: ['a', 'b', 'c'] }),
      ]

      const { result } = renderHook(() => useModeSuggestions({ skills }))

      // All share 'a' at level 0
      expect(result.current.getSuggestionsAtLevel(0, [])).toEqual(['a'])

      // Level 1 under 'a'
      expect(result.current.getSuggestionsAtLevel(1, ['a'])).toEqual(['b'])

      // Level 2 under 'a/b'
      expect(result.current.getSuggestionsAtLevel(2, ['a', 'b'])).toEqual(['c'])
    })
  })
})
