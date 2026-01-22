/**
 * Tests for PromptTreeSidebar component.
 *
 * Tests cover:
 * - Tree rendering with categories and items
 * - Selection handling
 * - Search functionality
 * - Expand/collapse controls
 * - Collapsed sidebar state
 * - Dirty indicators
 * - New prompt button
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { PromptTreeSidebar } from './PromptTreeSidebar'
import type { TreeNode } from '@/types/editor'
import type { Prompt } from '@/types'

// Helper to create a test prompt
function createTestPrompt(overrides: Partial<Prompt> = {}): Prompt {
  return {
    id: 'test-1',
    name: 'Test Prompt',
    description: 'A test description',
    content: '# Test content',
    modes: ['development'],
    tags: [],
    icon: 'file',
    draft: false,
    folder: 'internal',
    createdAt: '2025-01-01T00:00:00Z',
    updatedAt: '2025-01-01T00:00:00Z',
    usageCount: 0,
    ...overrides,
  }
}

// Helper to create tree nodes
function createCategoryNode(id: string, label: string, children: TreeNode[] = [], depth = 0): TreeNode {
  return {
    id,
    label,
    isCategory: true,
    children,
    depth,
  }
}

function createItemNode(id: string, label: string, itemId: string, depth = 0): TreeNode {
  return {
    id,
    label,
    isCategory: false,
    children: [],
    itemId,
    depth,
  }
}

describe('PromptTreeSidebar', () => {
  const defaultProps = {
    treeNodes: [] as TreeNode[],
    prompts: [] as Prompt[],
    selectedItemId: null,
    onSelectItem: vi.fn(),
    dirtyItemIds: new Set<string>(),
    expandedNodes: new Set<string>(),
    onToggleNode: vi.fn(),
    searchQuery: '',
    onSearchChange: vi.fn(),
    isCollapsed: false,
    onToggleCollapse: vi.fn(),
    onExpandAll: vi.fn(),
    onCollapseAll: vi.fn(),
    onCreateNew: vi.fn(),
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('expanded state', () => {
    it('should render the sidebar with header', () => {
      render(<PromptTreeSidebar {...defaultProps} />)

      expect(screen.getByText('Prompts')).toBeInTheDocument()
    })

    it('should render search input', () => {
      render(<PromptTreeSidebar {...defaultProps} />)

      expect(screen.getByPlaceholderText('Search prompts...')).toBeInTheDocument()
    })

    it('should render expand/collapse buttons', () => {
      render(<PromptTreeSidebar {...defaultProps} />)

      expect(screen.getByTitle('Expand all')).toBeInTheDocument()
      expect(screen.getByTitle('Collapse all')).toBeInTheDocument()
    })

    it('should render new prompt button', () => {
      render(<PromptTreeSidebar {...defaultProps} />)

      expect(screen.getByRole('button', { name: /new prompt/i })).toBeInTheDocument()
    })

    it('should call onCreateNew when new prompt button is clicked', () => {
      const onCreateNew = vi.fn()
      render(<PromptTreeSidebar {...defaultProps} onCreateNew={onCreateNew} />)

      fireEvent.click(screen.getByRole('button', { name: /new prompt/i }))

      expect(onCreateNew).toHaveBeenCalledTimes(1)
    })

    it('should render empty message when no prompts', () => {
      render(<PromptTreeSidebar {...defaultProps} />)

      expect(screen.getByText('No prompts yet')).toBeInTheDocument()
    })

    it('should render search empty message when search has no results', () => {
      render(<PromptTreeSidebar {...defaultProps} searchQuery="nonexistent" />)

      expect(screen.getByText('No prompts match your search')).toBeInTheDocument()
    })
  })

  describe('tree rendering', () => {
    it('should render tree nodes', () => {
      const prompt1 = createTestPrompt({ id: 'p1', name: 'Prompt One' })
      const treeNodes: TreeNode[] = [
        createItemNode('item-p1', 'Prompt One', 'p1'),
      ]

      render(
        <PromptTreeSidebar
          {...defaultProps}
          treeNodes={treeNodes}
          prompts={[prompt1]}
        />
      )

      expect(screen.getByText('Prompt One')).toBeInTheDocument()
    })

    it('should render category nodes', () => {
      const prompt1 = createTestPrompt({ id: 'p1', name: 'Prompt One', modes: ['development'] })
      const treeNodes: TreeNode[] = [
        createCategoryNode('development', 'development', [
          createItemNode('item-p1', 'Prompt One', 'p1', 1),
        ]),
      ]

      render(
        <PromptTreeSidebar
          {...defaultProps}
          treeNodes={treeNodes}
          prompts={[prompt1]}
          expandedNodes={new Set(['development'])}
        />
      )

      expect(screen.getByText('development')).toBeInTheDocument()
      expect(screen.getByText('Prompt One')).toBeInTheDocument()
    })

    it('should not show children of collapsed category', () => {
      const prompt1 = createTestPrompt({ id: 'p1', name: 'Prompt One' })
      const treeNodes: TreeNode[] = [
        createCategoryNode('development', 'development', [
          createItemNode('item-p1', 'Prompt One', 'p1', 1),
        ]),
      ]

      render(
        <PromptTreeSidebar
          {...defaultProps}
          treeNodes={treeNodes}
          prompts={[prompt1]}
          expandedNodes={new Set()} // Category not expanded
        />
      )

      expect(screen.getByText('development')).toBeInTheDocument()
      expect(screen.queryByText('Prompt One')).not.toBeInTheDocument()
    })
  })

  describe('selection', () => {
    it('should call onSelectItem when item is clicked', () => {
      const onSelectItem = vi.fn()
      const prompt1 = createTestPrompt({ id: 'p1', name: 'Prompt One' })
      const treeNodes: TreeNode[] = [
        createItemNode('item-p1', 'Prompt One', 'p1'),
      ]

      render(
        <PromptTreeSidebar
          {...defaultProps}
          treeNodes={treeNodes}
          prompts={[prompt1]}
          onSelectItem={onSelectItem}
        />
      )

      fireEvent.click(screen.getByText('Prompt One'))

      expect(onSelectItem).toHaveBeenCalledWith('p1')
    })

    it('should call onToggleNode when category is clicked', () => {
      const onToggleNode = vi.fn()
      const treeNodes: TreeNode[] = [
        createCategoryNode('development', 'development'),
      ]

      render(
        <PromptTreeSidebar
          {...defaultProps}
          treeNodes={treeNodes}
          onToggleNode={onToggleNode}
        />
      )

      fireEvent.click(screen.getByText('development'))

      expect(onToggleNode).toHaveBeenCalledWith('development')
    })
  })

  describe('search', () => {
    it('should call onSearchChange when search input changes', () => {
      const onSearchChange = vi.fn()
      render(<PromptTreeSidebar {...defaultProps} onSearchChange={onSearchChange} />)

      const input = screen.getByPlaceholderText('Search prompts...')
      fireEvent.change(input, { target: { value: 'test query' } })

      expect(onSearchChange).toHaveBeenCalledWith('test query')
    })

    it('should display current search query', () => {
      render(<PromptTreeSidebar {...defaultProps} searchQuery="current query" />)

      const input = screen.getByPlaceholderText('Search prompts...')
      expect((input as HTMLInputElement).value).toBe('current query')
    })
  })

  describe('expand/collapse controls', () => {
    it('should call onExpandAll when expand button is clicked', () => {
      const onExpandAll = vi.fn()
      render(<PromptTreeSidebar {...defaultProps} onExpandAll={onExpandAll} />)

      fireEvent.click(screen.getByTitle('Expand all'))

      expect(onExpandAll).toHaveBeenCalledTimes(1)
    })

    it('should call onCollapseAll when collapse button is clicked', () => {
      const onCollapseAll = vi.fn()
      render(<PromptTreeSidebar {...defaultProps} onCollapseAll={onCollapseAll} />)

      fireEvent.click(screen.getByTitle('Collapse all'))

      expect(onCollapseAll).toHaveBeenCalledTimes(1)
    })

    it('should call onToggleCollapse when collapse sidebar button is clicked', () => {
      const onToggleCollapse = vi.fn()
      render(<PromptTreeSidebar {...defaultProps} onToggleCollapse={onToggleCollapse} />)

      fireEvent.click(screen.getByTitle('Collapse sidebar'))

      expect(onToggleCollapse).toHaveBeenCalledTimes(1)
    })
  })

  describe('collapsed state', () => {
    it('should render narrow sidebar when collapsed', () => {
      render(<PromptTreeSidebar {...defaultProps} isCollapsed={true} />)

      // Should show expand button
      expect(screen.getByTitle('Expand sidebar')).toBeInTheDocument()
      // Should not show full header
      expect(screen.queryByText('Prompts')).not.toBeInTheDocument()
    })

    it('should call onToggleCollapse when expand button clicked in collapsed state', () => {
      const onToggleCollapse = vi.fn()
      render(
        <PromptTreeSidebar
          {...defaultProps}
          isCollapsed={true}
          onToggleCollapse={onToggleCollapse}
        />
      )

      fireEvent.click(screen.getByTitle('Expand sidebar'))

      expect(onToggleCollapse).toHaveBeenCalledTimes(1)
    })

    it('should show new prompt button in collapsed state', () => {
      const onCreateNew = vi.fn()
      render(
        <PromptTreeSidebar
          {...defaultProps}
          isCollapsed={true}
          onCreateNew={onCreateNew}
        />
      )

      fireEvent.click(screen.getByTitle('New prompt'))

      expect(onCreateNew).toHaveBeenCalledTimes(1)
    })

    it('should show dirty count in collapsed state', () => {
      render(
        <PromptTreeSidebar
          {...defaultProps}
          isCollapsed={true}
          dirtyItemIds={new Set(['p1', 'p2'])}
        />
      )

      expect(screen.getByText('2')).toBeInTheDocument()
      expect(screen.getByTitle('2 unsaved changes')).toBeInTheDocument()
    })
  })

  describe('dirty indicators', () => {
    it('should show dirty count in expanded sidebar header', () => {
      render(
        <PromptTreeSidebar
          {...defaultProps}
          dirtyItemIds={new Set(['p1', 'p2', 'p3'])}
        />
      )

      expect(screen.getByText('3 unsaved')).toBeInTheDocument()
    })

    it('should not show dirty count when no dirty items', () => {
      render(<PromptTreeSidebar {...defaultProps} />)

      expect(screen.queryByText(/unsaved/)).not.toBeInTheDocument()
    })
  })

  describe('custom icon rendering', () => {
    it('should render custom item icons when provided', () => {
      const prompt1 = createTestPrompt({ id: 'p1', name: 'Prompt One' })
      const treeNodes: TreeNode[] = [
        createItemNode('item-p1', 'Prompt One', 'p1'),
      ]

      const renderItemIcon = vi.fn().mockReturnValue(
        <span data-testid="custom-icon">Icon</span>
      )

      render(
        <PromptTreeSidebar
          {...defaultProps}
          treeNodes={treeNodes}
          prompts={[prompt1]}
          renderItemIcon={renderItemIcon}
        />
      )

      expect(screen.getByTestId('custom-icon')).toBeInTheDocument()
      expect(renderItemIcon).toHaveBeenCalledWith(prompt1)
    })
  })

  describe('search input ref', () => {
    it('should forward ref to search input', () => {
      const ref = { current: null } as React.RefObject<HTMLInputElement>
      render(<PromptTreeSidebar {...defaultProps} searchInputRef={ref} />)

      expect(ref.current).toBeInstanceOf(HTMLInputElement)
    })
  })

  describe('accessibility', () => {
    it('should have buttons with type="button"', () => {
      const treeNodes: TreeNode[] = [
        createCategoryNode('development', 'development'),
      ]

      render(<PromptTreeSidebar {...defaultProps} treeNodes={treeNodes} />)

      const buttons = screen.getAllByRole('button')
      buttons.forEach((button) => {
        expect(button).toHaveAttribute('type', 'button')
      })
    })
  })
})
