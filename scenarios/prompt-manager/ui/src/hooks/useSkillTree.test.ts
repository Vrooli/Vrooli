/**
 * Tests for useSkillTree hook.
 *
 * Tests cover:
 * - Tree building from skills
 * - Selection state management
 * - Expansion/collapse operations
 * - Search filtering
 * - Auto-expand behavior
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { act } from 'react'
import { useSkillTree } from './useSkillTree'
import type { Skill } from '@/types'

// Helper to create test skills
function createTestSkill(overrides: Partial<Skill> = {}): Skill {
  return {
    id: 'test-1',
    name: 'Test Skill',
    description: 'A test description',
    content: '# Test content',
    modes: [],
    tags: [],
    draft: false,
    folder: 'local',
    file: 'test-skill.md',
    createdAt: '2025-01-01T00:00:00Z',
    updatedAt: '2025-01-01T00:00:00Z',
    usageCount: 0,
    ...overrides,
  }
}

describe('useSkillTree', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('initialization', () => {
    it('should initialize with empty tree for no skills', () => {
      const { result } = renderHook(() =>
        useSkillTree({ skills: [] })
      )

      expect(result.current.treeNodes).toEqual([])
      expect(result.current.filteredTreeNodes).toEqual([])
    })

    it('should build tree from skills', () => {
      const skills = [
        createTestSkill({ id: '1', name: 'Skill 1', modes: ['development'] }),
        createTestSkill({ id: '2', name: 'Skill 2', modes: ['development'] }),
      ]

      const { result } = renderHook(() =>
        useSkillTree({ skills })
      )

      expect(result.current.treeNodes).toHaveLength(1) // One category
      expect(result.current.treeNodes[0]?.label).toBe('development')
      expect(result.current.treeNodes[0]?.children).toHaveLength(2)
    })

    it('should initialize with null selection by default', () => {
      const { result } = renderHook(() =>
        useSkillTree({ skills: [] })
      )

      expect(result.current.selectedItemId).toBeNull()
    })

    it('should accept initial selection', () => {
      const skills = [createTestSkill({ id: '1' })]

      const { result } = renderHook(() =>
        useSkillTree({ skills, initialSelectedId: '1' })
      )

      expect(result.current.selectedItemId).toBe('1')
    })

    it('should initialize with empty expanded nodes', () => {
      const { result } = renderHook(() =>
        useSkillTree({ skills: [] })
      )

      expect(result.current.expandedNodes.size).toBe(0)
    })

    it('should initialize with empty search query', () => {
      const { result } = renderHook(() =>
        useSkillTree({ skills: [] })
      )

      expect(result.current.searchQuery).toBe('')
    })

    it('should initialize as not collapsed', () => {
      const { result } = renderHook(() =>
        useSkillTree({ skills: [] })
      )

      expect(result.current.isCollapsed).toBe(false)
    })
  })

  describe('selection', () => {
    it('should update selection when setSelectedItemId is called', () => {
      const skills = [
        createTestSkill({ id: '1', name: 'Skill 1' }),
        createTestSkill({ id: '2', name: 'Skill 2' }),
      ]

      const { result } = renderHook(() =>
        useSkillTree({ skills })
      )

      act(() => {
        result.current.setSelectedItemId('2')
      })

      expect(result.current.selectedItemId).toBe('2')
    })

    it('should allow clearing selection', () => {
      const skills = [createTestSkill({ id: '1' })]

      const { result } = renderHook(() =>
        useSkillTree({ skills, initialSelectedId: '1' })
      )

      act(() => {
        result.current.setSelectedItemId(null)
      })

      expect(result.current.selectedItemId).toBeNull()
    })
  })

  describe('expansion', () => {
    it('should toggle node expansion', () => {
      const skills = [
        createTestSkill({ id: '1', modes: ['development'] }),
      ]

      const { result } = renderHook(() =>
        useSkillTree({ skills })
      )

      // Initially not expanded
      expect(result.current.expandedNodes.has('development')).toBe(false)

      // Expand
      act(() => {
        result.current.toggleNode('development')
      })
      expect(result.current.expandedNodes.has('development')).toBe(true)

      // Collapse
      act(() => {
        result.current.toggleNode('development')
      })
      expect(result.current.expandedNodes.has('development')).toBe(false)
    })

    it('should expand all category nodes', () => {
      const skills = [
        createTestSkill({ id: '1', modes: ['development', 'react'] }),
        createTestSkill({ id: '2', modes: ['testing', 'unit'] }),
      ]

      const { result } = renderHook(() =>
        useSkillTree({ skills })
      )

      act(() => {
        result.current.expandAll()
      })

      expect(result.current.expandedNodes.has('development')).toBe(true)
      expect(result.current.expandedNodes.has('development/react')).toBe(true)
      expect(result.current.expandedNodes.has('testing')).toBe(true)
      expect(result.current.expandedNodes.has('testing/unit')).toBe(true)
    })

    it('should collapse all nodes', () => {
      const skills = [
        createTestSkill({ id: '1', modes: ['development'] }),
      ]

      const { result } = renderHook(() =>
        useSkillTree({ skills })
      )

      // First expand
      act(() => {
        result.current.expandAll()
      })
      expect(result.current.expandedNodes.size).toBeGreaterThan(0)

      // Then collapse all
      act(() => {
        result.current.collapseAll()
      })
      expect(result.current.expandedNodes.size).toBe(0)
    })

    it('should expand to specific item', () => {
      const skills = [
        createTestSkill({ id: '1', modes: ['level1', 'level2', 'level3'] }),
      ]

      const { result } = renderHook(() =>
        useSkillTree({ skills })
      )

      act(() => {
        result.current.expandToItem('1')
      })

      expect(result.current.expandedNodes.has('level1')).toBe(true)
      expect(result.current.expandedNodes.has('level1/level2')).toBe(true)
      expect(result.current.expandedNodes.has('level1/level2/level3')).toBe(true)
    })
  })

  describe('search', () => {
    it('should filter tree by search query', () => {
      const skills = [
        createTestSkill({ id: '1', name: 'Alpha Skill', modes: ['dev'] }),
        createTestSkill({ id: '2', name: 'Beta Skill', modes: ['dev'] }),
      ]

      const { result } = renderHook(() =>
        useSkillTree({ skills })
      )

      // Initially all skills visible
      expect(result.current.filteredTreeNodes[0]?.children).toHaveLength(2)

      // Filter
      act(() => {
        result.current.setSearchQuery('Alpha')
      })

      expect(result.current.filteredTreeNodes[0]?.children).toHaveLength(1)
      expect(result.current.filteredTreeNodes[0]?.children[0]?.itemId).toBe('1')
    })

    it('should return original tree for empty search query', () => {
      const skills = [
        createTestSkill({ id: '1', modes: ['dev'] }),
      ]

      const { result } = renderHook(() =>
        useSkillTree({ skills })
      )

      act(() => {
        result.current.setSearchQuery('   ')
      })

      expect(result.current.filteredTreeNodes).toEqual(result.current.treeNodes)
    })

    it('should auto-expand nodes when searching', () => {
      const skills = [
        createTestSkill({ id: '1', name: 'Match', modes: ['development', 'react'] }),
      ]

      const { result } = renderHook(() =>
        useSkillTree({ skills })
      )

      act(() => {
        result.current.setSearchQuery('Match')
      })

      // Should auto-expand to show search results
      expect(result.current.expandedNodes.has('development')).toBe(true)
      expect(result.current.expandedNodes.has('development/react')).toBe(true)
    })

    it('should preserve persisted expansion on mount with restored search query', () => {
      const skills = [
        createTestSkill({ id: '1', name: 'Match', modes: ['development', 'react'] }),
      ]

      const { result } = renderHook(() =>
        useSkillTree({
          skills,
          initialSearchQuery: 'Match',
          initialExpandedNodes: ['development'],
        })
      )

      expect(result.current.expandedNodes.has('development')).toBe(true)
      expect(result.current.expandedNodes.has('development/react')).toBe(false)
    })

    it('should use server-backed quick search matches by ID', () => {
      const skills = [
        createTestSkill({ id: '1', name: 'Alpha Skill', content: '' }),
        createTestSkill({ id: '2', name: 'Beta Skill', content: '' }),
      ]

      const { result } = renderHook(() =>
        useSkillTree({
          skills,
          initialSearchQuery: 'content-only',
          searchMatchedSkillIds: new Set(['2']),
        })
      )

      expect(result.current.filteredSortedSkills).toHaveLength(1)
      expect(result.current.filteredSortedSkills[0]?.id).toBe('2')
      expect(result.current.filteredTreeNodes[0]?.children[0]?.itemId).toBe('2')
    })

    it('should not match skill body content in quick search fallback', () => {
      const skills = [
        createTestSkill({ id: '1', name: 'Alpha Skill', description: 'Alpha description', content: 'body-only-term' }),
        createTestSkill({ id: '2', name: 'Beta Skill', description: 'Beta description', content: '' }),
      ]

      const { result } = renderHook(() =>
        useSkillTree({
          skills,
          initialSearchQuery: 'body-only-term',
        })
      )

      expect(result.current.filteredSortedSkills).toHaveLength(0)
      expect(result.current.filteredTreeNodes).toHaveLength(0)
    })

    it('should keep local name matches when server-backed search returns no IDs', () => {
      const skills = [
        createTestSkill({ id: '1', name: 'Tests Skill', description: 'Alpha description', content: '' }),
        createTestSkill({ id: '2', name: 'Beta Skill', description: 'Beta description', content: '' }),
      ]

      const { result } = renderHook(() =>
        useSkillTree({
          skills,
          initialSearchQuery: 'test',
          searchMatchedSkillIds: new Set(),
        })
      )

      expect(result.current.filteredSortedSkills).toHaveLength(1)
      expect(result.current.filteredSortedSkills[0]?.id).toBe('1')
      expect(result.current.filteredTreeNodes[0]?.children[0]?.itemId).toBe('1')
    })
  })

  describe('sidebar collapse', () => {
    it('should toggle sidebar collapse state', () => {
      const { result } = renderHook(() =>
        useSkillTree({ skills: [] })
      )

      expect(result.current.isCollapsed).toBe(false)

      act(() => {
        result.current.toggleCollapse()
      })
      expect(result.current.isCollapsed).toBe(true)

      act(() => {
        result.current.toggleCollapse()
      })
      expect(result.current.isCollapsed).toBe(false)
    })
  })

  describe('auto-expand on selection', () => {
    it('should auto-expand path to selected item', () => {
      const skills = [
        createTestSkill({ id: '1', modes: ['category1', 'subcategory'] }),
      ]

      const { result } = renderHook(() =>
        useSkillTree({ skills })
      )

      // Select the item
      act(() => {
        result.current.setSelectedItemId('1')
      })

      // Should auto-expand the path
      expect(result.current.expandedNodes.has('category1')).toBe(true)
      expect(result.current.expandedNodes.has('category1/subcategory')).toBe(true)
    })
  })

  describe('tree updates', () => {
    it('should rebuild tree when skills change', () => {
      const skills1 = [createTestSkill({ id: '1', modes: ['dev'] })]
      const skills2 = [
        createTestSkill({ id: '1', modes: ['dev'] }),
        createTestSkill({ id: '2', modes: ['test'] }),
      ]

      const { result, rerender } = renderHook(
        ({ skills }: { skills: Skill[] }) => useSkillTree({ skills }),
        { initialProps: { skills: skills1 } }
      )

      expect(result.current.treeNodes).toHaveLength(1)

      rerender({ skills: skills2 })

      expect(result.current.treeNodes).toHaveLength(2)
    })
  })
})
