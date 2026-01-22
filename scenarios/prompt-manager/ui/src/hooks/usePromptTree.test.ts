/**
 * Tests for usePromptTree hook.
 *
 * Tests cover:
 * - Tree building from prompts
 * - Selection state management
 * - Expansion/collapse operations
 * - Search filtering
 * - Auto-expand behavior
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { act } from 'react'
import { usePromptTree } from './usePromptTree'
import type { Prompt } from '@/types'

// Helper to create test prompts
function createTestPrompt(overrides: Partial<Prompt> = {}): Prompt {
  return {
    id: 'test-1',
    name: 'Test Prompt',
    description: 'A test description',
    content: '# Test content',
    modes: [],
    tags: [],
    draft: false,
    folder: 'internal',
    createdAt: '2025-01-01T00:00:00Z',
    updatedAt: '2025-01-01T00:00:00Z',
    usageCount: 0,
    ...overrides,
  }
}

describe('usePromptTree', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('initialization', () => {
    it('should initialize with empty tree for no prompts', () => {
      const { result } = renderHook(() =>
        usePromptTree({ prompts: [] })
      )

      expect(result.current.treeNodes).toEqual([])
      expect(result.current.filteredTreeNodes).toEqual([])
    })

    it('should build tree from prompts', () => {
      const prompts = [
        createTestPrompt({ id: '1', name: 'Prompt 1', modes: ['development'] }),
        createTestPrompt({ id: '2', name: 'Prompt 2', modes: ['development'] }),
      ]

      const { result } = renderHook(() =>
        usePromptTree({ prompts })
      )

      expect(result.current.treeNodes).toHaveLength(1) // One category
      expect(result.current.treeNodes[0]?.label).toBe('development')
      expect(result.current.treeNodes[0]?.children).toHaveLength(2)
    })

    it('should initialize with null selection by default', () => {
      const { result } = renderHook(() =>
        usePromptTree({ prompts: [] })
      )

      expect(result.current.selectedItemId).toBeNull()
    })

    it('should accept initial selection', () => {
      const prompts = [createTestPrompt({ id: '1' })]

      const { result } = renderHook(() =>
        usePromptTree({ prompts, initialSelectedId: '1' })
      )

      expect(result.current.selectedItemId).toBe('1')
    })

    it('should initialize with empty expanded nodes', () => {
      const { result } = renderHook(() =>
        usePromptTree({ prompts: [] })
      )

      expect(result.current.expandedNodes.size).toBe(0)
    })

    it('should initialize with empty search query', () => {
      const { result } = renderHook(() =>
        usePromptTree({ prompts: [] })
      )

      expect(result.current.searchQuery).toBe('')
    })

    it('should initialize as not collapsed', () => {
      const { result } = renderHook(() =>
        usePromptTree({ prompts: [] })
      )

      expect(result.current.isCollapsed).toBe(false)
    })
  })

  describe('selection', () => {
    it('should update selection when setSelectedItemId is called', () => {
      const prompts = [
        createTestPrompt({ id: '1', name: 'Prompt 1' }),
        createTestPrompt({ id: '2', name: 'Prompt 2' }),
      ]

      const { result } = renderHook(() =>
        usePromptTree({ prompts })
      )

      act(() => {
        result.current.setSelectedItemId('2')
      })

      expect(result.current.selectedItemId).toBe('2')
    })

    it('should allow clearing selection', () => {
      const prompts = [createTestPrompt({ id: '1' })]

      const { result } = renderHook(() =>
        usePromptTree({ prompts, initialSelectedId: '1' })
      )

      act(() => {
        result.current.setSelectedItemId(null)
      })

      expect(result.current.selectedItemId).toBeNull()
    })
  })

  describe('expansion', () => {
    it('should toggle node expansion', () => {
      const prompts = [
        createTestPrompt({ id: '1', modes: ['development'] }),
      ]

      const { result } = renderHook(() =>
        usePromptTree({ prompts })
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
      const prompts = [
        createTestPrompt({ id: '1', modes: ['development', 'react'] }),
        createTestPrompt({ id: '2', modes: ['testing', 'unit'] }),
      ]

      const { result } = renderHook(() =>
        usePromptTree({ prompts })
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
      const prompts = [
        createTestPrompt({ id: '1', modes: ['development'] }),
      ]

      const { result } = renderHook(() =>
        usePromptTree({ prompts })
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
      const prompts = [
        createTestPrompt({ id: '1', modes: ['level1', 'level2', 'level3'] }),
      ]

      const { result } = renderHook(() =>
        usePromptTree({ prompts })
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
      const prompts = [
        createTestPrompt({ id: '1', name: 'Alpha Prompt', modes: ['dev'] }),
        createTestPrompt({ id: '2', name: 'Beta Prompt', modes: ['dev'] }),
      ]

      const { result } = renderHook(() =>
        usePromptTree({ prompts })
      )

      // Initially all prompts visible
      expect(result.current.filteredTreeNodes[0]?.children).toHaveLength(2)

      // Filter
      act(() => {
        result.current.setSearchQuery('Alpha')
      })

      expect(result.current.filteredTreeNodes[0]?.children).toHaveLength(1)
      expect(result.current.filteredTreeNodes[0]?.children[0]?.itemId).toBe('1')
    })

    it('should return original tree for empty search query', () => {
      const prompts = [
        createTestPrompt({ id: '1', modes: ['dev'] }),
      ]

      const { result } = renderHook(() =>
        usePromptTree({ prompts })
      )

      act(() => {
        result.current.setSearchQuery('   ')
      })

      expect(result.current.filteredTreeNodes).toEqual(result.current.treeNodes)
    })

    it('should auto-expand nodes when searching', () => {
      const prompts = [
        createTestPrompt({ id: '1', name: 'Match', modes: ['development', 'react'] }),
      ]

      const { result } = renderHook(() =>
        usePromptTree({ prompts })
      )

      act(() => {
        result.current.setSearchQuery('Match')
      })

      // Should auto-expand to show search results
      expect(result.current.expandedNodes.has('development')).toBe(true)
      expect(result.current.expandedNodes.has('development/react')).toBe(true)
    })
  })

  describe('sidebar collapse', () => {
    it('should toggle sidebar collapse state', () => {
      const { result } = renderHook(() =>
        usePromptTree({ prompts: [] })
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
      const prompts = [
        createTestPrompt({ id: '1', modes: ['category1', 'subcategory'] }),
      ]

      const { result } = renderHook(() =>
        usePromptTree({ prompts })
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
    it('should rebuild tree when prompts change', () => {
      const prompts1 = [createTestPrompt({ id: '1', modes: ['dev'] })]
      const prompts2 = [
        createTestPrompt({ id: '1', modes: ['dev'] }),
        createTestPrompt({ id: '2', modes: ['test'] }),
      ]

      const { result, rerender } = renderHook(
        ({ prompts }: { prompts: Prompt[] }) => usePromptTree({ prompts }),
        { initialProps: { prompts: prompts1 } }
      )

      expect(result.current.treeNodes).toHaveLength(1)

      rerender({ prompts: prompts2 })

      expect(result.current.treeNodes).toHaveLength(2)
    })
  })
})
