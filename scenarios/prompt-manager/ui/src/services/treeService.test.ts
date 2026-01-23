/**
 * Tests for treeService.ts
 *
 * Tests cover:
 * - Building tree structures from skills
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
import type { Skill } from '@/types'
import type { TreeNode } from '@/types/editor'

// Helper to create a minimal skill for testing
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

describe('buildTree', () => {
  it('should create empty tree for empty skills array', () => {
    const tree = buildTree([])
    expect(tree).toEqual([])
  })

  it('should put skills without modes in "Other" category', () => {
    const skills = [
      createTestSkill({ id: '1', name: 'Skill 1', modes: [] }),
      createTestSkill({ id: '2', name: 'Skill 2', modes: [] }),
    ]

    const tree = buildTree(skills)

    expect(tree).toHaveLength(1)
    expect(tree[0]?.id).toBe('__other__')
    expect(tree[0]?.label).toBe('Other')
    expect(tree[0]?.isCategory).toBe(true)
    expect(tree[0]?.children).toHaveLength(2)
  })

  it('should create category nodes for single-level modes', () => {
    const skills = [
      createTestSkill({ id: '1', name: 'Skill 1', modes: ['development'] }),
      createTestSkill({ id: '2', name: 'Skill 2', modes: ['development'] }),
      createTestSkill({ id: '3', name: 'Skill 3', modes: ['testing'] }),
    ]

    const tree = buildTree(skills)

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
    const skills = [
      createTestSkill({ id: '1', name: 'Skill 1', modes: ['development', 'react'] }),
      createTestSkill({ id: '2', name: 'Skill 2', modes: ['development', 'react'] }),
      createTestSkill({ id: '3', name: 'Skill 3', modes: ['development', 'vue'] }),
    ]

    const tree = buildTree(skills)

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
    const skills = [
      createTestSkill({ id: 'skill-123', name: 'My Skill', modes: ['dev'] }),
    ]

    const tree = buildTree(skills)
    const leaf = tree[0]?.children[0]

    expect(leaf?.id).toBe('item-skill-123')
    expect(leaf?.itemId).toBe('skill-123')
    expect(leaf?.label).toBe('My Skill')
    expect(leaf?.isCategory).toBe(false)
  })

  it('should sort categories before items, then alphabetically', () => {
    const skills = [
      createTestSkill({ id: '1', name: 'Zebra', modes: ['dev'] }),
      createTestSkill({ id: '2', name: 'Alpha', modes: ['dev'] }),
      createTestSkill({ id: '3', name: 'Beta', modes: ['dev', 'sub'] }),
    ]

    const tree = buildTree(skills)
    const devChildren = tree[0]?.children ?? []

    // The 'sub' category should come first (categories before items)
    expect(devChildren[0]?.isCategory).toBe(true)
    expect(devChildren[0]?.label).toBe('sub')

    // Then items sorted alphabetically
    expect(devChildren[1]?.label).toBe('Alpha')
    expect(devChildren[2]?.label).toBe('Zebra')
  })

  it('should set correct depth for nodes', () => {
    const skills = [
      createTestSkill({ id: '1', name: 'Skill', modes: ['level1', 'level2', 'level3'] }),
    ]

    const tree = buildTree(skills)

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
  it('should return empty array for skill without modes', () => {
    const skills = [createTestSkill({ id: '1', modes: [] })]
    expect(getPathsToItem(skills, '1')).toEqual([])
  })

  it('should return empty array for non-existent skill', () => {
    const skills = [createTestSkill({ id: '1', modes: ['dev'] })]
    expect(getPathsToItem(skills, 'non-existent')).toEqual([])
  })

  it('should return single path for single-level mode', () => {
    const skills = [createTestSkill({ id: '1', modes: ['development'] })]
    expect(getPathsToItem(skills, '1')).toEqual(['development'])
  })

  it('should return all path segments for multi-level modes', () => {
    const skills = [createTestSkill({ id: '1', modes: ['development', 'react', 'hooks'] })]

    expect(getPathsToItem(skills, '1')).toEqual([
      'development',
      'development/react',
      'development/react/hooks',
    ])
  })
})

describe('filterTree', () => {
  it('should return original tree for empty query', () => {
    const skills = [createTestSkill({ id: '1', name: 'Test', modes: ['dev'] })]
    const tree = buildTree(skills)

    expect(filterTree(tree, '', skills)).toEqual(tree)
    expect(filterTree(tree, '   ', skills)).toEqual(tree)
  })

  it('should filter by skill name', () => {
    const skills = [
      createTestSkill({ id: '1', name: 'Alpha Skill', modes: ['dev'] }),
      createTestSkill({ id: '2', name: 'Beta Skill', modes: ['dev'] }),
    ]
    const tree = buildTree(skills)

    const filtered = filterTree(tree, 'Alpha', skills)

    expect(filtered).toHaveLength(1)
    expect(filtered[0]?.children).toHaveLength(1)
    expect(filtered[0]?.children[0]?.itemId).toBe('1')
  })

  it('should filter by skill description', () => {
    const skills = [
      createTestSkill({ id: '1', name: 'Skill 1', description: 'Contains search term', modes: ['dev'] }),
      createTestSkill({ id: '2', name: 'Skill 2', description: 'No match here', modes: ['dev'] }),
    ]
    const tree = buildTree(skills)

    const filtered = filterTree(tree, 'search term', skills)

    expect(filtered[0]?.children).toHaveLength(1)
    expect(filtered[0]?.children[0]?.itemId).toBe('1')
  })

  it('should filter by skill content', () => {
    const skills = [
      createTestSkill({ id: '1', name: 'Skill 1', content: 'findme in content', modes: ['dev'] }),
      createTestSkill({ id: '2', name: 'Skill 2', content: 'nothing here', modes: ['dev'] }),
    ]
    const tree = buildTree(skills)

    const filtered = filterTree(tree, 'findme', skills)

    expect(filtered[0]?.children).toHaveLength(1)
    expect(filtered[0]?.children[0]?.itemId).toBe('1')
  })

  it('should filter by tags', () => {
    const skills = [
      createTestSkill({ id: '1', name: 'Skill 1', tags: ['important', 'urgent'], modes: ['dev'] }),
      createTestSkill({ id: '2', name: 'Skill 2', tags: ['low-priority'], modes: ['dev'] }),
    ]
    const tree = buildTree(skills)

    const filtered = filterTree(tree, 'urgent', skills)

    expect(filtered[0]?.children).toHaveLength(1)
    expect(filtered[0]?.children[0]?.itemId).toBe('1')
  })

  it('should filter by modes', () => {
    const skills = [
      createTestSkill({ id: '1', name: 'Skill 1', modes: ['development', 'react'] }),
      createTestSkill({ id: '2', name: 'Skill 2', modes: ['development', 'vue'] }),
    ]
    const tree = buildTree(skills)

    const filtered = filterTree(tree, 'react', skills)

    // Development category should still exist, with only react subcategory
    expect(filtered).toHaveLength(1)
    const devCategory = filtered[0]
    expect(devCategory?.children).toHaveLength(1)
    expect(devCategory?.children[0]?.label).toBe('react')
  })

  it('should be case-insensitive', () => {
    const skills = [
      createTestSkill({ id: '1', name: 'UPPERCASE', modes: ['dev'] }),
    ]
    const tree = buildTree(skills)

    const filtered = filterTree(tree, 'uppercase', skills)
    expect(filtered[0]?.children).toHaveLength(1)
  })

  it('should keep parent categories when children match', () => {
    const skills = [
      createTestSkill({ id: '1', name: 'Match', modes: ['category1', 'subcategory'] }),
    ]
    const tree = buildTree(skills)

    const filtered = filterTree(tree, 'Match', skills)

    expect(filtered).toHaveLength(1)
    expect(filtered[0]?.label).toBe('category1')
    expect(filtered[0]?.children[0]?.label).toBe('subcategory')
  })
})

describe('getModesAtLevel', () => {
  it('should return unique modes at level 0', () => {
    const skills = [
      createTestSkill({ id: '1', modes: ['development'] }),
      createTestSkill({ id: '2', modes: ['development'] }),
      createTestSkill({ id: '3', modes: ['testing'] }),
      createTestSkill({ id: '4', modes: ['production'] }),
    ]

    const modes = getModesAtLevel(skills, 0, [])

    expect(modes).toHaveLength(3)
    expect(modes).toContain('development')
    expect(modes).toContain('testing')
    expect(modes).toContain('production')
  })

  it('should return modes at level 1 filtered by parent path', () => {
    const skills = [
      createTestSkill({ id: '1', modes: ['development', 'react'] }),
      createTestSkill({ id: '2', modes: ['development', 'vue'] }),
      createTestSkill({ id: '3', modes: ['testing', 'unit'] }),
    ]

    const modes = getModesAtLevel(skills, 1, ['development'])

    expect(modes).toHaveLength(2)
    expect(modes).toContain('react')
    expect(modes).toContain('vue')
    expect(modes).not.toContain('unit')
  })

  it('should return empty array when no skills match parent path', () => {
    const skills = [
      createTestSkill({ id: '1', modes: ['development', 'react'] }),
    ]

    const modes = getModesAtLevel(skills, 1, ['testing'])

    expect(modes).toEqual([])
  })

  it('should return sorted modes', () => {
    const skills = [
      createTestSkill({ id: '1', modes: ['zebra'] }),
      createTestSkill({ id: '2', modes: ['alpha'] }),
      createTestSkill({ id: '3', modes: ['beta'] }),
    ]

    const modes = getModesAtLevel(skills, 0, [])

    expect(modes).toEqual(['alpha', 'beta', 'zebra'])
  })

  it('should handle skills with null/undefined modes', () => {
    // Test with empty modes array (simulating undefined after nullish coalescing)
    const skills = [
      createTestSkill({ id: '1', modes: [] }),
      createTestSkill({ id: '2', modes: ['development'] }),
    ]

    const modes = getModesAtLevel(skills, 0, [])

    expect(modes).toEqual(['development'])
  })
})
