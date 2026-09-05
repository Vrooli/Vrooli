/**
 * Tests for SkillEditorPanel component.
 *
 * Tests cover:
 * - Empty state rendering
 * - Skill display with form fields
 * - Read-only mode indicator
 * - Dirty state indicator
 * - Toolbar visibility
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@/test-utils/renderWithProviders'
import { SkillEditorPanel } from './SkillEditorPanel'
import type { Skill } from '@/types'
import type { NormalizedFormState, ValidationResult } from '@/types/editorStore'

// Mock useResolvedTheme (used by SkillContentEditor)
vi.mock('@/hooks/use-theme', () => ({
  useResolvedTheme: vi.fn(() => 'dark'),
}))

// Mock Monaco editor since it requires browser environment
vi.mock('@monaco-editor/react', () => ({
  default: ({ value }: { value: string }) => (
    <div data-testid="monaco-editor">{value}</div>
  ),
  useMonaco: () => ({
    MarkerSeverity: { Warning: 4, Error: 8 },
    editor: { setModelMarkers: vi.fn() },
  }),
}))

// Mock TipTapEditor
vi.mock('./TipTapEditor', () => ({
  TipTapEditor: ({ value }: { value: string }) => (
    <div data-testid="tiptap-editor">{value}</div>
  ),
}))

// Mock the world route component - it shows when no skill is selected
vi.mock('@/world', () => ({
  WorldView: () => <div data-testid="world-view" />,
}))

// Mock ViewOverlay and graph/world content components used in empty state
vi.mock('../shared/ViewOverlay', () => ({
  ViewOverlay: () => <div data-testid="view-overlay" />,
}))
vi.mock('@/components/graph/GraphView', () => ({
  GraphView: () => <div data-testid="graph-view" />,
}))
vi.mock('@/components/graph/OperatingMapFlow', () => ({
  OperatingMapFlow: () => <div data-testid="operating-map-flow" />,
}))
vi.mock('@/components/graph/GraphSettingsContent', () => ({
  GraphSettingsContent: () => <div />,
}))
vi.mock('@/components/graph/GraphHelpContent', () => ({
  GraphHelpContent: () => <div />,
}))
vi.mock('@/components/graph/GraphLegend', () => ({
  GraphLegend: () => <div />,
}))
vi.mock('@/components/graph/GraphQueryPanel', () => ({
  GraphQueryPanel: () => <div />,
}))

// Mock lineage child panels to avoid their hook/network dependencies.
vi.mock('./tabs/VersionHistoryTab', () => ({
  VersionHistoryTab: () => <div data-testid="mock-history">history</div>,
}))
vi.mock('./VariantPanel', () => ({
  VariantPanel: () => <div data-testid="mock-variants">variants</div>,
}))
vi.mock('./ExperimentPanel', () => ({
  ExperimentPanel: () => <div data-testid="mock-experiments">experiments</div>,
}))
vi.mock('./CrossReferencePanel', () => ({
  CrossReferencePanel: () => <div data-testid="mock-xref" />,
}))

// Helper to create test skill
function createTestSkill(overrides: Partial<Skill> = {}): Skill {
  return {
    id: 'test-1',
    file: 'test-1.md',
    name: 'Test Skill',
    description: 'A test description',
    content: '# Test content',
    modes: ['development'],
    tags: ['tag1'],
    draft: false,
    folder: 'local',
    createdAt: '2025-01-01T00:00:00Z',
    updatedAt: '2025-01-01T00:00:00Z',
    usageCount: 0,
    ...overrides,
  }
}

// Helper to create default form state
function createFormState(overrides: Partial<NormalizedFormState> = {}): NormalizedFormState {
  return {
    file: 'test-1.md',
    name: 'Test Skill',
    description: 'A test description',
    content: '# Test content',
    modes: ['development'],
    tags: ['tag1'],  // Now an array
    icon: '',
    defaultScope: '',
    draft: false,
    folder: 'local',
    ...overrides,
  }
}

// Helper to create validation result
function createValidation(
  valid = true,
  errors: Record<string, string> = {}
): ValidationResult {
  return { valid, errors }
}

const defaultProps = {
  currentSkill: null,
  formState: createFormState(),
  validation: createValidation(),
  isDirty: false,
  dirtyCount: 0,
  onFieldChange: vi.fn(),
  onSave: vi.fn(),
  onSaveAll: vi.fn(),
  onDiscard: vi.fn(),
  onDelete: vi.fn(),
  isSaving: false,
  isDeleting: false,
}

describe('SkillEditorPanel', () => {
  describe('graph projection selector', () => {
    it('keeps Relationships as the default and persists a Flow selection', () => {
      localStorage.removeItem('pm.graphProjection')
      render(<SkillEditorPanel {...defaultProps} currentSkill={null} homeView="graph" />)
      expect(screen.getByTestId('graph-view')).toBeInTheDocument()
      expect(screen.getByTestId('graph-projection-toggle')).toHaveClass('bottom-32', 'sm:top-3')
      fireEvent.click(screen.getByRole('button', { name: 'Flow' }))
      expect(screen.getByTestId('operating-map-flow')).toBeInTheDocument()
      expect(localStorage.getItem('pm.graphProjection')).toBe('flow')
    })
  })

  describe('empty state (world)', () => {
    it('should show the world surface when no skill is selected', () => {
      render(<SkillEditorPanel {...defaultProps} currentSkill={null} />)

      expect(screen.getByTestId('world-view')).toBeInTheDocument()
      expect(screen.queryByTestId('view-overlay')).not.toBeInTheDocument()
    })
  })

  describe('skill display', () => {
    it('should render editor when skill is selected', () => {
      const skill = createTestSkill()

      render(<SkillEditorPanel {...defaultProps} currentSkill={skill} />)

      // Should not show empty state
      expect(screen.queryByText('No Skill Selected')).not.toBeInTheDocument()
    })

    it('should display content in editor', () => {
      const skill = createTestSkill({ content: 'Test content here' })
      const formState = createFormState({ content: 'Test content here' })

      render(
        <SkillEditorPanel
          {...defaultProps}
          currentSkill={skill}
          formState={formState}
        />
      )

      // Content should be passed to editor
      expect(screen.getByTestId('monaco-editor')).toHaveTextContent('Test content here')
    })
  })

  describe('dirty state indicator', () => {
    it('shows an unsaved dot on the save button when dirty', () => {
      const skill = createTestSkill()

      render(
        <SkillEditorPanel
          {...defaultProps}
          currentSkill={skill}
          isDirty={true}
        />
      )

      expect(screen.getByTestId('skill-editor-unsaved-indicator')).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /Save \(unsaved changes\)/ })).toBeInTheDocument()
    })

    it('does not render the unsaved dot when not dirty', () => {
      const skill = createTestSkill()

      render(
        <SkillEditorPanel
          {...defaultProps}
          currentSkill={skill}
          isDirty={false}
        />
      )

      expect(screen.queryByTestId('skill-editor-unsaved-indicator')).not.toBeInTheDocument()
    })
  })

  describe('validation errors', () => {
    it('should pass validation errors to form', () => {
      const skill = createTestSkill()
      const validation = createValidation(false, { content: 'Content is required' })

      render(
        <SkillEditorPanel
          {...defaultProps}
          currentSkill={skill}
          validation={validation}
        />
      )

      // The error should be passed to the content editor
      expect(screen.getByText('Content is required')).toBeInTheDocument()
    })
  })

  describe('className prop', () => {
    it('should apply custom className', () => {
      const { container } = render(
        <SkillEditorPanel {...defaultProps} currentSkill={null} className="custom-class" />
      )

      expect(container.firstChild).toHaveClass('custom-class')
    })
  })

  describe('header cleanup', () => {
    it('hides the published status chip and shows draft status only when draft', () => {
      const skill = createTestSkill()
      render(<SkillEditorPanel {...defaultProps} currentSkill={skill} />)
      expect(screen.queryByTestId('draft-status-chip')).not.toBeInTheDocument()
    })

    it('renders the draft status chip for draft skills', () => {
      const skill = createTestSkill({ draft: true })
      render(
        <SkillEditorPanel
          {...defaultProps}
          currentSkill={skill}
          formState={createFormState({ draft: true })}
        />
      )
      expect(screen.getByTestId('draft-status-chip')).toHaveTextContent('Draft')
    })

    it('does not render the legacy row-1 Skill actions ellipsis', () => {
      const skill = createTestSkill()
      render(<SkillEditorPanel {...defaultProps} currentSkill={skill} />)
      expect(screen.queryByRole('button', { name: 'Skill actions' })).not.toBeInTheDocument()
    })

    it('does not render legacy per-panel toggle buttons', () => {
      const skill = createTestSkill()
      render(<SkillEditorPanel {...defaultProps} currentSkill={skill} />)
      expect(screen.queryByRole('button', { name: /Toggle version history/ })).not.toBeInTheDocument()
      expect(screen.queryByRole('button', { name: /Toggle variants/ })).not.toBeInTheDocument()
      expect(screen.queryByRole('button', { name: /Toggle experiments/ })).not.toBeInTheDocument()
    })
  })

  describe('lineage panel', () => {
    it('is closed by default', () => {
      const skill = createTestSkill()
      render(<SkillEditorPanel {...defaultProps} currentSkill={skill} />)
      const toggle = screen.getByTestId('lineage-toggle')
      expect(toggle).toHaveAttribute('aria-expanded', 'false')
      expect(screen.queryByTestId('lineage-tab-history')).not.toBeInTheDocument()
    })

    it('opens on click and shows three tabs', () => {
      const skill = createTestSkill()
      render(<SkillEditorPanel {...defaultProps} currentSkill={skill} />)
      fireEvent.click(screen.getByTestId('lineage-toggle'))
      expect(screen.getByTestId('lineage-toggle')).toHaveAttribute('aria-expanded', 'true')
      expect(screen.getByTestId('lineage-tab-history')).toBeInTheDocument()
      expect(screen.getByTestId('lineage-tab-variants')).toBeInTheDocument()
      expect(screen.getByTestId('lineage-tab-experiments')).toBeInTheDocument()
    })

    it('remembers last-selected tab across open/close within the session', () => {
      const skill = createTestSkill()
      render(<SkillEditorPanel {...defaultProps} currentSkill={skill} />)
      fireEvent.click(screen.getByTestId('lineage-toggle'))
      fireEvent.mouseDown(screen.getByTestId('lineage-tab-variants'))
      expect(screen.getByTestId('lineage-tab-variants')).toHaveAttribute('data-state', 'active')
      // close
      fireEvent.click(screen.getByTestId('lineage-toggle'))
      expect(screen.queryByTestId('lineage-tab-variants')).not.toBeInTheDocument()
      // reopen — variants should still be active
      fireEvent.click(screen.getByTestId('lineage-toggle'))
      expect(screen.getByTestId('lineage-tab-variants')).toHaveAttribute('data-state', 'active')
    })
  })

  describe('More overflow menu', () => {
    it('exposes chat, Discard changes, and Delete skill items', () => {
      const skill = createTestSkill()
      render(
        <SkillEditorPanel
          {...defaultProps}
          currentSkill={skill}
          isDirty={true}
        />
      )
      fireEvent.click(screen.getByTestId('skill-editor-actions-menu'))
      expect(screen.getByText('Start agent chat')).toBeInTheDocument()
      expect(screen.getByText('Discard changes')).toBeInTheDocument()
      expect(screen.getByText('Delete skill')).toBeInTheDocument()
    })

    it('invokes onDiscard when Discard changes is clicked and enabled', () => {
      const onDiscard = vi.fn()
      const skill = createTestSkill()
      render(
        <SkillEditorPanel
          {...defaultProps}
          currentSkill={skill}
          isDirty={true}
          onDiscard={onDiscard}
        />
      )
      fireEvent.click(screen.getByTestId('skill-editor-actions-menu'))
      fireEvent.click(screen.getByText('Discard changes'))
      expect(onDiscard).toHaveBeenCalled()
    })

    it('disables Discard changes when not dirty', () => {
      const onDiscard = vi.fn()
      const skill = createTestSkill()
      render(
        <SkillEditorPanel
          {...defaultProps}
          currentSkill={skill}
          isDirty={false}
          onDiscard={onDiscard}
        />
      )
      fireEvent.click(screen.getByTestId('skill-editor-actions-menu'))
      const item = screen.getByText('Discard changes').closest('button')
      expect(item).toBeDisabled()
    })

    it('invokes onDelete when Delete skill is clicked', () => {
      const onDelete = vi.fn()
      const skill = createTestSkill()
      render(
        <SkillEditorPanel
          {...defaultProps}
          currentSkill={skill}
          onDelete={onDelete}
        />
      )
      fireEvent.click(screen.getByTestId('skill-editor-actions-menu'))
      fireEvent.click(screen.getByText('Delete skill'))
      expect(onDelete).toHaveBeenCalled()
    })

    it('shows Deleting… while delete is in flight', () => {
      const skill = createTestSkill()
      render(
        <SkillEditorPanel
          {...defaultProps}
          currentSkill={skill}
          isDeleting={true}
        />
      )
      fireEvent.click(screen.getByTestId('skill-editor-actions-menu'))
      expect(screen.getByText(/Deleting/)).toBeInTheDocument()
    })

    it('opens the file path popover from the header button', () => {
      const skill = createTestSkill()
      render(<SkillEditorPanel {...defaultProps} currentSkill={skill} />)
      fireEvent.click(screen.getByRole('button', { name: 'Open file path menu for test-1.md' }))
      // The FilePathMenu popover contains a Filename label
      expect(screen.getByText('Filename')).toBeInTheDocument()
    })
  })
})
