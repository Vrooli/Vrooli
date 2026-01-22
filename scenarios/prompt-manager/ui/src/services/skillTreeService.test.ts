/**
 * Tests for skillTreeService.ts
 *
 * Tests cover:
 * - Building skill tree from prompts
 * - Node positioning and layout
 * - Selection state updates
 * - Camera position calculations
 */

import { describe, it, expect } from 'vitest'
import {
  buildSkillTree,
  updateSelection,
  findNodeByPromptId,
  getSelectedPrompts,
  calculateCameraPosition,
} from './skillTreeService'
import type { Prompt } from '@/types'

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
    folder: 'internal',
    createdAt: '2025-01-01T00:00:00Z',
    updatedAt: '2025-01-01T00:00:00Z',
    usageCount: 0,
    ...overrides,
  }
}

describe('buildSkillTree', () => {
  it('should create empty tree for empty prompts array', () => {
    const tree = buildSkillTree([])

    expect(tree.nodes).toEqual([])
    expect(tree.connections).toEqual([])
    expect(tree.roots).toEqual([])
    expect(tree.maxDepth).toBe(0)
  })

  it('should create nodes for each prompt (no modes = root level)', () => {
    const prompts = [
      createTestPrompt({ id: '1', name: 'Prompt 1' }),
      createTestPrompt({ id: '2', name: 'Prompt 2' }),
      createTestPrompt({ id: '3', name: 'Prompt 3' }),
    ]

    const tree = buildSkillTree(prompts)

    // Prompts without modes go directly to root as leaf nodes
    expect(tree.nodes).toHaveLength(3)
    expect(tree.nodes.map(n => n.promptId)).toContain('1')
    expect(tree.nodes.map(n => n.promptId)).toContain('2')
    expect(tree.nodes.map(n => n.promptId)).toContain('3')

    // All should be at root level
    expect(tree.roots).toHaveLength(3)
  })

  it('should create hierarchical tree from mode paths', () => {
    const prompts = [
      createTestPrompt({ id: '1', name: 'Code Review', modes: ['coding/review'] }),
      createTestPrompt({ id: '2', name: 'Code Debug', modes: ['coding/debug'] }),
      createTestPrompt({ id: '3', name: 'Write Blog', modes: ['writing/blog'] }),
    ]

    const tree = buildSkillTree(prompts)

    // Should create mode nodes + prompt nodes:
    // - mode-coding, mode-coding/review, mode-coding/debug (3 mode nodes under coding)
    // - mode-writing, mode-writing/blog (2 mode nodes under writing)
    // - 3 prompt leaf nodes
    // Total: 8 nodes (5 mode nodes + 3 prompt nodes)
    expect(tree.nodes).toHaveLength(8)

    // Root level should have top-level mode nodes
    expect(tree.roots).toHaveLength(2)
    expect(tree.roots).toContain('mode-coding')
    expect(tree.roots).toContain('mode-writing')

    // Prompt nodes should exist
    expect(tree.nodes.some(n => n.promptId === '1')).toBe(true)
    expect(tree.nodes.some(n => n.promptId === '2')).toBe(true)
    expect(tree.nodes.some(n => n.promptId === '3')).toBe(true)

    // Mode nodes should be marked as such
    const modeNodes = tree.nodes.filter(n => n.isModeNode === true)
    expect(modeNodes.length).toBe(5)
  })

  it('should assign positions to nodes', () => {
    const prompts = [
      createTestPrompt({ id: '1', name: 'Prompt 1', modes: ['dev'] }),
    ]

    const tree = buildSkillTree(prompts)

    // With mode 'dev', we get: mode-dev (mode node) + node-1 (prompt node)
    expect(tree.nodes).toHaveLength(2)

    // Check that all nodes have valid positions
    for (const node of tree.nodes) {
      expect(node.position).toBeDefined()
      expect(node.position).toHaveLength(3)
      expect(typeof node.position[0]).toBe('number')
      expect(typeof node.position[1]).toBe('number')
      expect(typeof node.position[2]).toBe('number')
    }
  })

  it('should assign colors based on mode categories', () => {
    const prompts = [
      createTestPrompt({ id: '1', modes: ['coding'] }),
      createTestPrompt({ id: '2', modes: ['writing'] }),
      createTestPrompt({ id: '3', modes: ['analysis'] }),
      createTestPrompt({ id: '4', modes: [] }),
    ]

    const tree = buildSkillTree(prompts)

    // With modes, we get: 3 mode nodes (coding, writing, analysis) + 4 prompt nodes
    // The prompt without modes (id: '4') goes directly to root
    expect(tree.nodes.length).toBeGreaterThanOrEqual(4)

    // Each node should have a color
    tree.nodes.forEach((node) => {
      expect(node.color).toBeDefined()
      expect(node.color).toMatch(/^#[0-9a-f]{6}$/i)
    })
  })

  it('should calculate node size based on usage', () => {
    const prompts = [
      createTestPrompt({ id: '1', usageCount: 0 }),
      createTestPrompt({ id: '2', usageCount: 50 }),
    ]

    const tree = buildSkillTree(prompts)
    const node0 = tree.nodes.find(n => n.promptId === '1')
    const node1 = tree.nodes.find(n => n.promptId === '2')

    // Higher usage should result in larger size
    expect(node0).toBeDefined()
    expect(node1).toBeDefined()
    if (node0 && node1) {
      expect(node1.size).toBeGreaterThan(node0.size)
    }
  })

  it('should create connections between parent and child nodes', () => {
    const prompts = [
      createTestPrompt({ id: '1', modes: ['coding'] }),
      createTestPrompt({ id: '2', modes: ['coding'] }),
      createTestPrompt({ id: '3', modes: ['coding'] }),
    ]

    const tree = buildSkillTree(prompts)

    // With mode 'coding', we get: 1 mode node + 3 prompt nodes
    // Each prompt node connects to its parent mode node
    expect(tree.nodes).toHaveLength(4)

    // Should have connections from mode node to each prompt
    expect(tree.connections.length).toBe(3)
    for (const conn of tree.connections) {
      expect(conn.source).toHaveLength(3)
      expect(conn.target).toHaveLength(3)
    }
  })

  it('should store original prompt reference in prompt nodes', () => {
    const prompts = [createTestPrompt({ id: '1', name: 'Test Prompt' })]

    const tree = buildSkillTree(prompts)

    // Prompt without modes goes directly to root
    const promptNode = tree.nodes.find(n => n.promptId === '1')

    expect(promptNode).toBeDefined()
    expect(promptNode?.prompt).toBeDefined()
    expect(promptNode?.prompt.id).toBe('1')
    expect(promptNode?.prompt.name).toBe('Test Prompt')
    expect(promptNode?.isModeNode).toBeFalsy()
  })
})

describe('updateSelection', () => {
  it('should mark specified nodes as selected', () => {
    const prompts = [
      createTestPrompt({ id: '1' }),
      createTestPrompt({ id: '2' }),
      createTestPrompt({ id: '3' }),
    ]

    const tree = buildSkillTree(prompts)
    const updated = updateSelection(tree, ['1', '3'])

    const node1 = updated.nodes.find((n) => n.promptId === '1')
    const node2 = updated.nodes.find((n) => n.promptId === '2')
    const node3 = updated.nodes.find((n) => n.promptId === '3')

    expect(node1?.isSelected).toBe(true)
    expect(node2?.isSelected).toBe(false)
    expect(node3?.isSelected).toBe(true)
  })

  it('should clear selection when empty array passed', () => {
    const prompts = [createTestPrompt({ id: '1' })]

    const tree = buildSkillTree(prompts)
    const selected = updateSelection(tree, ['1'])
    const cleared = updateSelection(selected, [])
    const node = cleared.nodes[0]

    expect(node).toBeDefined()
    expect(node?.isSelected).toBe(false)
  })

  it('should not mutate original tree', () => {
    const prompts = [createTestPrompt({ id: '1' })]

    const tree = buildSkillTree(prompts)
    const node = tree.nodes[0]
    expect(node).toBeDefined()
    const original = node?.isSelected
    updateSelection(tree, ['1'])

    expect(tree.nodes[0]?.isSelected).toBe(original)
  })
})

describe('findNodeByPromptId', () => {
  it('should find node by prompt ID', () => {
    const prompts = [
      createTestPrompt({ id: 'prompt-123', name: 'Target Prompt' }),
      createTestPrompt({ id: 'prompt-456', name: 'Other Prompt' }),
    ]

    const tree = buildSkillTree(prompts)
    const node = findNodeByPromptId(tree, 'prompt-123')

    expect(node).toBeDefined()
    expect(node?.promptId).toBe('prompt-123')
    expect(node?.name).toBe('Target Prompt')
  })

  it('should return undefined for non-existent ID', () => {
    const prompts = [createTestPrompt({ id: '1' })]

    const tree = buildSkillTree(prompts)
    const node = findNodeByPromptId(tree, 'non-existent')

    expect(node).toBeUndefined()
  })
})

describe('getSelectedPrompts', () => {
  it('should return prompts for selected nodes', () => {
    const prompts = [
      createTestPrompt({ id: '1', name: 'First' }),
      createTestPrompt({ id: '2', name: 'Second' }),
      createTestPrompt({ id: '3', name: 'Third' }),
    ]

    const tree = buildSkillTree(prompts)
    const selected = updateSelection(tree, ['1', '3'])
    const result = getSelectedPrompts(selected)

    expect(result).toHaveLength(2)
    expect(result.map((p) => p.id)).toContain('1')
    expect(result.map((p) => p.id)).toContain('3')
    expect(result.map((p) => p.id)).not.toContain('2')
  })

  it('should return empty array when nothing selected', () => {
    const prompts = [createTestPrompt({ id: '1' })]

    const tree = buildSkillTree(prompts)
    const result = getSelectedPrompts(tree)

    expect(result).toEqual([])
  })
})

describe('calculateCameraPosition', () => {
  it('should return default position for empty tree', () => {
    const tree = buildSkillTree([])
    const position = calculateCameraPosition(tree)

    expect(position).toHaveLength(3)
    expect(position[1]).toBeGreaterThan(0) // Y should be elevated
    expect(position[2]).toBeGreaterThan(0) // Z should be positive (looking at scene)
  })

  it('should position camera to view all nodes', () => {
    const prompts = [
      createTestPrompt({ id: '1', modes: ['left'] }),
      createTestPrompt({ id: '2', modes: ['right'] }),
      createTestPrompt({ id: '3', modes: ['center'] }),
    ]

    const tree = buildSkillTree(prompts)
    const position = calculateCameraPosition(tree)

    // Camera should be far enough to see all nodes
    expect(position[2]).toBeGreaterThan(5)
  })

  it('should adjust for large trees', () => {
    const prompts = Array.from({ length: 20 }, (_, i) =>
      createTestPrompt({ id: `${i}`, modes: [`mode${i % 5}`] })
    )

    const smallTree = buildSkillTree(prompts.slice(0, 3))
    const largeTree = buildSkillTree(prompts)

    const smallPos = calculateCameraPosition(smallTree)
    const largePos = calculateCameraPosition(largeTree)

    // Larger tree should have camera further back
    expect(largePos[2]).toBeGreaterThan(smallPos[2])
  })
})
