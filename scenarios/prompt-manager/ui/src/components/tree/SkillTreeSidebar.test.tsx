/**
 * Tests for SkillTreeSidebar component.
 *
 * Tests cover:
 * - Tree rendering with categories and items
 * - Selection handling
 * - Search functionality
 * - Expand/collapse controls
 * - Collapsed sidebar state
 * - Dirty indicators
 * - New skill button
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { SkillTreeSidebar } from './SkillTreeSidebar'
import type { TreeNode } from '@/types/editor'
import type { Skill } from '@/types'

// Helper to create a test skill
function createTestSkill(overrides: Partial<Skill> = {}): Skill {
  return {
    id: 'test-1',
    name: 'Test Skill',
    description: 'A test description',
    content: '# Test content',
    modes: ['development'],
    tags: [],
    icon: 'file',
    draft: false,
    folder: 'local',
    file: 'test-skill.md',
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

describe('SkillTreeSidebar', () => {
  const defaultProps = {
    treeNodes: [] as TreeNode[],
    skills: [] as Skill[],
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
    // Tag filter props
    selectedTags: [] as string[],
    onSelectedTagsChange: vi.fn(),
    availableTags: [] as string[],
    // Folder filter props
    selectedFolders: [] as string[],
    onSelectedFoldersChange: vi.fn(),
    availableFolders: ['core', 'local', 'drafts'] as string[],
    // Skill selection mode props
    skillSelectionMode: false,
    skillSelectedIds: new Set<string>(),
    currentMember: null,
    onSkillSelectionSave: vi.fn(),
    onSkillSelectionCancel: vi.fn(),
    getSkillSelectionState: vi.fn().mockReturnValue('none' as const),
    onSkillCheckboxChange: vi.fn(),
    // Context menu callbacks
    onDeleteFolder: vi.fn(),
    onCopySkill: vi.fn(),
    onMoveToFolder: vi.fn(),
    onChangeStorage: vi.fn(),
    onCreateNewFolder: vi.fn(),
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('expanded state', () => {
    it('should render the sidebar with header', () => {
      render(<SkillTreeSidebar {...defaultProps} />)

      expect(screen.getByText('Skills')).toBeInTheDocument()
    })

    it('should render search input', () => {
      render(<SkillTreeSidebar {...defaultProps} />)

      expect(screen.getByPlaceholderText('Search skills... (Ctrl+K)')).toBeInTheDocument()
    })

    it('should render expand/collapse buttons', () => {
      render(<SkillTreeSidebar {...defaultProps} />)

      expect(screen.getByTitle('Expand all')).toBeInTheDocument()
      expect(screen.getByTitle('Collapse all')).toBeInTheDocument()
    })

    it('should render new skill button', () => {
      render(<SkillTreeSidebar {...defaultProps} />)

      expect(screen.getByRole('button', { name: /new skill/i })).toBeInTheDocument()
    })

    it('should call onCreateNew when new skill button is clicked', () => {
      const onCreateNew = vi.fn()
      render(<SkillTreeSidebar {...defaultProps} onCreateNew={onCreateNew} />)

      fireEvent.click(screen.getByRole('button', { name: /new skill/i }))

      expect(onCreateNew).toHaveBeenCalledTimes(1)
    })

    it('should render empty message when no skills', () => {
      render(<SkillTreeSidebar {...defaultProps} />)

      expect(screen.getByText('No skills yet')).toBeInTheDocument()
    })

    it('should render search empty message when search has no results', () => {
      render(<SkillTreeSidebar {...defaultProps} searchQuery="nonexistent" />)

      expect(screen.getByText('No skills match your filters')).toBeInTheDocument()
    })
  })

  describe('tree rendering', () => {
    it('should render tree nodes', () => {
      const skill1 = createTestSkill({ id: 'p1', name: 'Skill One' })
      const treeNodes: TreeNode[] = [
        createItemNode('item-p1', 'Skill One', 'p1'),
      ]

      render(
        <SkillTreeSidebar
          {...defaultProps}
          treeNodes={treeNodes}
          skills={[skill1]}
        />
      )

      expect(screen.getByText('Skill One')).toBeInTheDocument()
    })

    it('should render category nodes', () => {
      const skill1 = createTestSkill({ id: 'p1', name: 'Skill One', modes: ['development'] })
      const treeNodes: TreeNode[] = [
        createCategoryNode('development', 'development', [
          createItemNode('item-p1', 'Skill One', 'p1', 1),
        ]),
      ]

      render(
        <SkillTreeSidebar
          {...defaultProps}
          treeNodes={treeNodes}
          skills={[skill1]}
          expandedNodes={new Set(['development'])}
        />
      )

      expect(screen.getByText('development')).toBeInTheDocument()
      expect(screen.getByText('Skill One')).toBeInTheDocument()
    })

    it('should not show children of collapsed category', () => {
      const skill1 = createTestSkill({ id: 'p1', name: 'Skill One' })
      const treeNodes: TreeNode[] = [
        createCategoryNode('development', 'development', [
          createItemNode('item-p1', 'Skill One', 'p1', 1),
        ]),
      ]

      render(
        <SkillTreeSidebar
          {...defaultProps}
          treeNodes={treeNodes}
          skills={[skill1]}
          expandedNodes={new Set()} // Category not expanded
        />
      )

      expect(screen.getByText('development')).toBeInTheDocument()
      expect(screen.queryByText('Skill One')).not.toBeInTheDocument()
    })
  })

  describe('selection', () => {
    it('should call onSelectItem when item is clicked', () => {
      const onSelectItem = vi.fn()
      const skill1 = createTestSkill({ id: 'p1', name: 'Skill One' })
      const treeNodes: TreeNode[] = [
        createItemNode('item-p1', 'Skill One', 'p1'),
      ]

      render(
        <SkillTreeSidebar
          {...defaultProps}
          treeNodes={treeNodes}
          skills={[skill1]}
          onSelectItem={onSelectItem}
        />
      )

      fireEvent.click(screen.getByText('Skill One'))

      expect(onSelectItem).toHaveBeenCalledWith('p1')
    })

    it('should call onToggleNode when category is clicked', () => {
      const onToggleNode = vi.fn()
      const treeNodes: TreeNode[] = [
        createCategoryNode('development', 'development'),
      ]

      render(
        <SkillTreeSidebar
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
      render(<SkillTreeSidebar {...defaultProps} onSearchChange={onSearchChange} />)

      const input = screen.getByPlaceholderText('Search skills... (Ctrl+K)')
      fireEvent.change(input, { target: { value: 'test query' } })

      expect(onSearchChange).toHaveBeenCalledWith('test query')
    })

    it('should display current search query', () => {
      render(<SkillTreeSidebar {...defaultProps} searchQuery="current query" />)

      const input = screen.getByPlaceholderText('Search skills... (Ctrl+K)')
      expect((input as HTMLInputElement).value).toBe('current query')
    })
  })

  describe('expand/collapse controls', () => {
    it('should call onExpandAll when expand button is clicked', () => {
      const onExpandAll = vi.fn()
      render(<SkillTreeSidebar {...defaultProps} onExpandAll={onExpandAll} />)

      fireEvent.click(screen.getByTitle('Expand all'))

      expect(onExpandAll).toHaveBeenCalledTimes(1)
    })

    it('should call onCollapseAll when collapse button is clicked', () => {
      const onCollapseAll = vi.fn()
      render(<SkillTreeSidebar {...defaultProps} onCollapseAll={onCollapseAll} />)

      fireEvent.click(screen.getByTitle('Collapse all'))

      expect(onCollapseAll).toHaveBeenCalledTimes(1)
    })

    it('should call onToggleCollapse when collapse sidebar button is clicked', () => {
      const onToggleCollapse = vi.fn()
      render(<SkillTreeSidebar {...defaultProps} onToggleCollapse={onToggleCollapse} />)

      fireEvent.click(screen.getByTitle('Collapse sidebar'))

      expect(onToggleCollapse).toHaveBeenCalledTimes(1)
    })
  })

  describe('collapsed state', () => {
    it('should render narrow sidebar when collapsed', () => {
      render(<SkillTreeSidebar {...defaultProps} isCollapsed={true} />)

      // Should show expand button
      expect(screen.getByTitle('Expand sidebar')).toBeInTheDocument()
      // Should not show full header
      expect(screen.queryByText('Skills')).not.toBeInTheDocument()
    })

    it('should call onToggleCollapse when expand button clicked in collapsed state', () => {
      const onToggleCollapse = vi.fn()
      render(
        <SkillTreeSidebar
          {...defaultProps}
          isCollapsed={true}
          onToggleCollapse={onToggleCollapse}
        />
      )

      fireEvent.click(screen.getByTitle('Expand sidebar'))

      expect(onToggleCollapse).toHaveBeenCalledTimes(1)
    })

    it('should show new skill button in collapsed state', () => {
      const onCreateNew = vi.fn()
      render(
        <SkillTreeSidebar
          {...defaultProps}
          isCollapsed={true}
          onCreateNew={onCreateNew}
        />
      )

      fireEvent.click(screen.getByTitle('New skill (Ctrl+N)'))

      expect(onCreateNew).toHaveBeenCalledTimes(1)
    })

    it('should show dirty count in collapsed state', () => {
      render(
        <SkillTreeSidebar
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
        <SkillTreeSidebar
          {...defaultProps}
          dirtyItemIds={new Set(['p1', 'p2', 'p3'])}
        />
      )

      expect(screen.getByText('3 unsaved')).toBeInTheDocument()
    })

    it('should not show dirty count when no dirty items', () => {
      render(<SkillTreeSidebar {...defaultProps} />)

      expect(screen.queryByText(/unsaved/)).not.toBeInTheDocument()
    })
  })

  describe('custom icon rendering', () => {
    it('should render custom item icons when provided', () => {
      const skill1 = createTestSkill({ id: 'p1', name: 'Skill One' })
      const treeNodes: TreeNode[] = [
        createItemNode('item-p1', 'Skill One', 'p1'),
      ]

      const renderItemIcon = vi.fn().mockReturnValue(
        <span data-testid="custom-icon">Icon</span>
      )

      render(
        <SkillTreeSidebar
          {...defaultProps}
          treeNodes={treeNodes}
          skills={[skill1]}
          renderItemIcon={renderItemIcon}
        />
      )

      expect(screen.getByTestId('custom-icon')).toBeInTheDocument()
      expect(renderItemIcon).toHaveBeenCalledWith(skill1)
    })
  })

  describe('search input ref', () => {
    it('should forward ref to search input', () => {
      const ref = { current: null } as React.RefObject<HTMLInputElement>
      render(<SkillTreeSidebar {...defaultProps} searchInputRef={ref} />)

      expect(ref.current).toBeInstanceOf(HTMLInputElement)
    })
  })

  describe('accessibility', () => {
    it('should have buttons with type="button"', () => {
      const treeNodes: TreeNode[] = [
        createCategoryNode('development', 'development'),
      ]

      render(<SkillTreeSidebar {...defaultProps} treeNodes={treeNodes} />)

      const buttons = screen.getAllByRole('button')
      buttons.forEach((button) => {
        expect(button).toHaveAttribute('type', 'button')
      })
    })
  })
})
