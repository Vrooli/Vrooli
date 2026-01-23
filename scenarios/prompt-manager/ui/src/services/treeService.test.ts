/**
 * Tests for treeService.ts
 *
 * Tests cover:
 * - Building tree structures from prompts
 * - Counting dirty items in subtrees
 * - Finding paths to items
 * - Filtering tree by search query
 * - Getting modes at specific levels
 */

import { describe, it, expect } from 'vitest'
import {
  buildTree,
  countDirtyInSubtree,
  getPathsToItem,
  filterTree,
  getModesAtLevel,
} from './treeService'
import type { Prompt } from '@/types'
import type { TreeNode } from '@/types/editor'

// Helper to create a minimal prompt for testing
function createTestPrompt(overrides: Partial<Prompt> = {}): Prompt {
  return {
    id: 'test-1',
    name: 'Test Prompt',
    description: 'A test description',
    content: '# Test content',
    modes: [],
    tags: [],
    draft: false,
    folder: 'local',
    createdAt: '2025-01-01T00:00:00Z',
    updatedAt: '2025-01-01T00:00:00Z',
    usageCount: 0,
    ...overrides,
  }
}

describe('buildTree', () => {
  it('should create empty tree for empty prompts array', () => {
    const tree = buildTree([])
    expect(tree).toEqual([])
  })

  it('should put prompts without modes in "Other" category', () => {
    const prompts = [
      createTestPrompt({ id: '1', name: 'Prompt 1', modes: [] }),
      createTestPrompt({ id: '2', name: 'Prompt 2', modes: [] }),
    ]

    const tree = buildTree(prompts)

    expect(tree).toHaveLength(1)
    expect(tree[0]?.id).toBe('__other__')
    expect(tree[0]?.label).toBe('Other')
    expect(tree[0]?.isCategory).toBe(true)
    expect(tree[0]?.children).toHaveLength(2)
  })

  it('should create category nodes for single-level modes', () => {
    const prompts = [
      createTestPrompt({ id: '1', name: 'Prompt 1', modes: ['development'] }),
      createTestPrompt({ id: '2', name: 'Prompt 2', modes: ['development'] }),
      createTestPrompt({ id: '3', name: 'Prompt 3', modes: ['testing'] }),
    ]

    const tree = buildTree(prompts)

    // Should have two categories: development and testing
    expect(tree).toHaveLength(2)

    const devCategory = tree.find(n => n.label === 'development')
    expect(devCategory).toBeDefined()
    expect(devCategory?.isCategory).toBe(true)
    expect(devCategory?.children).toHaveLength(2)

    const testCategory = tree.find(n => n.label === 'testing')
    expect(testCategory).toBeDefined()
    expect(testCategory?.children).toHaveLength(1)
  })

  it('should create nested category nodes for multi-level modes', () => {
    const prompts = [
      createTestPrompt({ id: '1', name: 'Prompt 1', modes: ['development', 'react'] }),
      createTestPrompt({ id: '2', name: 'Prompt 2', modes: ['development', 'react'] }),
      createTestPrompt({ id: '3', name: 'Prompt 3', modes: ['development', 'vue'] }),
    ]

    const tree = buildTree(prompts)

    expect(tree).toHaveLength(1)
    const devCategory = tree[0]
    expect(devCategory?.label).toBe('development')
    expect(devCategory?.children).toHaveLength(2)

    const reactCategory = devCategory?.children.find(n => n.label === 'react')
    expect(reactCategory?.isCategory).toBe(true)
    expect(reactCategory?.children).toHaveLength(2)

    const vueCategory = devCategory?.children.find(n => n.label === 'vue')
    expect(vueCategory?.children).toHaveLength(1)
  })

  it('should create leaf nodes with correct itemId', () => {
    const prompts = [
      createTestPrompt({ id: 'prompt-123', name: 'My Prompt', modes: ['dev'] }),
    ]

    const tree = buildTree(prompts)
    const leaf = tree[0]?.children[0]

    expect(leaf?.id).toBe('item-prompt-123')
    expect(leaf?.itemId).toBe('prompt-123')
    expect(leaf?.label).toBe('My Prompt')
    expect(leaf?.isCategory).toBe(false)
  })

  it('should sort categories before items, then alphabetically', () => {
    const prompts = [
      createTestPrompt({ id: '1', name: 'Zebra', modes: ['dev'] }),
      createTestPrompt({ id: '2', name: 'Alpha', modes: ['dev'] }),
      createTestPrompt({ id: '3', name: 'Beta', modes: ['dev', 'sub'] }),
    ]

    const tree = buildTree(prompts)
    const devChildren = tree[0]?.children ?? []

    // The 'sub' category should come first (categories before items)
    expect(devChildren[0]?.isCategory).toBe(true)
    expect(devChildren[0]?.label).toBe('sub')

    // Then items sorted alphabetically
    expect(devChildren[1]?.label).toBe('Alpha')
    expect(devChildren[2]?.label).toBe('Zebra')
  })

  it('should set correct depth for nodes', () => {
    const prompts = [
      createTestPrompt({ id: '1', name: 'Prompt', modes: ['level1', 'level2', 'level3'] }),
    ]

    const tree = buildTree(prompts)

    expect(tree[0]?.depth).toBe(0)
    expect(tree[0]?.children[0]?.depth).toBe(1)
    expect(tree[0]?.children[0]?.children[0]?.depth).toBe(2)
    expect(tree[0]?.children[0]?.children[0]?.children[0]?.depth).toBe(3)
  })
})

describe('countDirtyInSubtree', () => {
  it('should return 0 for empty dirty set', () => {
    const node: TreeNode = {
      id: 'item-1',
      label: 'Test',
      isCategory: false,
      children: [],
      itemId: '1',
      depth: 0,
    }

    expect(countDirtyInSubtree(node, new Set())).toBe(0)
  })

  it('should return 1 for dirty leaf node', () => {
    const node: TreeNode = {
      id: 'item-1',
      label: 'Test',
      isCategory: false,
      children: [],
      itemId: '1',
      depth: 0,
    }

    expect(countDirtyInSubtree(node, new Set(['1']))).toBe(1)
  })

  it('should count dirty items in category subtree', () => {
    const node: TreeNode = {
      id: 'category',
      label: 'Category',
      isCategory: true,
      depth: 0,
      children: [
        { id: 'item-1', label: 'Item 1', isCategory: false, children: [], itemId: '1', depth: 1 },
        { id: 'item-2', label: 'Item 2', isCategory: false, children: [], itemId: '2', depth: 1 },
        { id: 'item-3', label: 'Item 3', isCategory: false, children: [], itemId: '3', depth: 1 },
      ],
    }

    expect(countDirtyInSubtree(node, new Set(['1', '3']))).toBe(2)
  })

  it('should recursively count in nested categories', () => {
    const node: TreeNode = {
      id: 'root',
      label: 'Root',
      isCategory: true,
      depth: 0,
      children: [
        {
          id: 'sub',
          label: 'Sub',
          isCategory: true,
          depth: 1,
          children: [
            { id: 'item-1', label: 'Item 1', isCategory: false, children: [], itemId: '1', depth: 2 },
          ],
        },
        { id: 'item-2', label: 'Item 2', isCategory: false, children: [], itemId: '2', depth: 1 },
      ],
    }

    expect(countDirtyInSubtree(node, new Set(['1', '2']))).toBe(2)
  })
})

describe('getPathsToItem', () => {
  it('should return empty array for prompt without modes', () => {
    const prompts = [createTestPrompt({ id: '1', modes: [] })]
    expect(getPathsToItem(prompts, '1')).toEqual([])
  })

  it('should return empty array for non-existent prompt', () => {
    const prompts = [createTestPrompt({ id: '1', modes: ['dev'] })]
    expect(getPathsToItem(prompts, 'non-existent')).toEqual([])
  })

  it('should return single path for single-level mode', () => {
    const prompts = [createTestPrompt({ id: '1', modes: ['development'] })]
    expect(getPathsToItem(prompts, '1')).toEqual(['development'])
  })

  it('should return all path segments for multi-level modes', () => {
    const prompts = [createTestPrompt({ id: '1', modes: ['development', 'react', 'hooks'] })]

    expect(getPathsToItem(prompts, '1')).toEqual([
      'development',
      'development/react',
      'development/react/hooks',
    ])
  })
})

describe('filterTree', () => {
  it('should return original tree for empty query', () => {
    const prompts = [createTestPrompt({ id: '1', name: 'Test', modes: ['dev'] })]
    const tree = buildTree(prompts)

    expect(filterTree(tree, '', prompts)).toEqual(tree)
    expect(filterTree(tree, '   ', prompts)).toEqual(tree)
  })

  it('should filter by prompt name', () => {
    const prompts = [
      createTestPrompt({ id: '1', name: 'Alpha Prompt', modes: ['dev'] }),
      createTestPrompt({ id: '2', name: 'Beta Prompt', modes: ['dev'] }),
    ]
    const tree = buildTree(prompts)

    const filtered = filterTree(tree, 'Alpha', prompts)

    expect(filtered).toHaveLength(1)
    expect(filtered[0]?.children).toHaveLength(1)
    expect(filtered[0]?.children[0]?.itemId).toBe('1')
  })

  it('should filter by prompt description', () => {
    const prompts = [
      createTestPrompt({ id: '1', name: 'Prompt 1', description: 'Contains search term', modes: ['dev'] }),
      createTestPrompt({ id: '2', name: 'Prompt 2', description: 'No match here', modes: ['dev'] }),
    ]
    const tree = buildTree(prompts)

    const filtered = filterTree(tree, 'search term', prompts)

    expect(filtered[0]?.children).toHaveLength(1)
    expect(filtered[0]?.children[0]?.itemId).toBe('1')
  })

  it('should filter by prompt content', () => {
    const prompts = [
      createTestPrompt({ id: '1', name: 'Prompt 1', content: 'findme in content', modes: ['dev'] }),
      createTestPrompt({ id: '2', name: 'Prompt 2', content: 'nothing here', modes: ['dev'] }),
    ]
    const tree = buildTree(prompts)

    const filtered = filterTree(tree, 'findme', prompts)

    expect(filtered[0]?.children).toHaveLength(1)
    expect(filtered[0]?.children[0]?.itemId).toBe('1')
  })

  it('should filter by tags', () => {
    const prompts = [
      createTestPrompt({ id: '1', name: 'Prompt 1', tags: ['important', 'urgent'], modes: ['dev'] }),
      createTestPrompt({ id: '2', name: 'Prompt 2', tags: ['low-priority'], modes: ['dev'] }),
    ]
    const tree = buildTree(prompts)

    const filtered = filterTree(tree, 'urgent', prompts)

    expect(filtered[0]?.children).toHaveLength(1)
    expect(filtered[0]?.children[0]?.itemId).toBe('1')
  })

  it('should filter by modes', () => {
    const prompts = [
      createTestPrompt({ id: '1', name: 'Prompt 1', modes: ['development', 'react'] }),
      createTestPrompt({ id: '2', name: 'Prompt 2', modes: ['development', 'vue'] }),
    ]
    const tree = buildTree(prompts)

    const filtered = filterTree(tree, 'react', prompts)

    // Development category should still exist, with only react subcategory
    expect(filtered).toHaveLength(1)
    const devCategory = filtered[0]
    expect(devCategory?.children).toHaveLength(1)
    expect(devCategory?.children[0]?.label).toBe('react')
  })

  it('should be case-insensitive', () => {
    const prompts = [
      createTestPrompt({ id: '1', name: 'UPPERCASE', modes: ['dev'] }),
    ]
    const tree = buildTree(prompts)

    const filtered = filterTree(tree, 'uppercase', prompts)
    expect(filtered[0]?.children).toHaveLength(1)
  })

  it('should keep parent categories when children match', () => {
    const prompts = [
      createTestPrompt({ id: '1', name: 'Match', modes: ['category1', 'subcategory'] }),
    ]
    const tree = buildTree(prompts)

    const filtered = filterTree(tree, 'Match', prompts)

    expect(filtered).toHaveLength(1)
    expect(filtered[0]?.label).toBe('category1')
    expect(filtered[0]?.children[0]?.label).toBe('subcategory')
  })
})

describe('getModesAtLevel', () => {
  it('should return unique modes at level 0', () => {
    const prompts = [
      createTestPrompt({ id: '1', modes: ['development'] }),
      createTestPrompt({ id: '2', modes: ['development'] }),
      createTestPrompt({ id: '3', modes: ['testing'] }),
      createTestPrompt({ id: '4', modes: ['production'] }),
    ]

    const modes = getModesAtLevel(prompts, 0, [])

    expect(modes).toHaveLength(3)
    expect(modes).toContain('development')
    expect(modes).toContain('testing')
    expect(modes).toContain('production')
  })

  it('should return modes at level 1 filtered by parent path', () => {
    const prompts = [
      createTestPrompt({ id: '1', modes: ['development', 'react'] }),
      createTestPrompt({ id: '2', modes: ['development', 'vue'] }),
      createTestPrompt({ id: '3', modes: ['testing', 'unit'] }),
    ]

    const modes = getModesAtLevel(prompts, 1, ['development'])

    expect(modes).toHaveLength(2)
    expect(modes).toContain('react')
    expect(modes).toContain('vue')
    expect(modes).not.toContain('unit')
  })

  it('should return empty array when no prompts match parent path', () => {
    const prompts = [
      createTestPrompt({ id: '1', modes: ['development', 'react'] }),
    ]

    const modes = getModesAtLevel(prompts, 1, ['testing'])

    expect(modes).toEqual([])
  })

  it('should return sorted modes', () => {
    const prompts = [
      createTestPrompt({ id: '1', modes: ['zebra'] }),
      createTestPrompt({ id: '2', modes: ['alpha'] }),
      createTestPrompt({ id: '3', modes: ['beta'] }),
    ]

    const modes = getModesAtLevel(prompts, 0, [])

    expect(modes).toEqual(['alpha', 'beta', 'zebra'])
  })

  it('should handle prompts with null/undefined modes', () => {
    // Test with empty modes array (simulating undefined after nullish coalescing)
    const prompts = [
      createTestPrompt({ id: '1', modes: [] }),
      createTestPrompt({ id: '2', modes: ['development'] }),
    ]

    const modes = getModesAtLevel(prompts, 0, [])

    expect(modes).toEqual(['development'])
  })
})
