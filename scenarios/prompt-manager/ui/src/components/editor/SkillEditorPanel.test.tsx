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
import { render, screen, fireEvent } from '@testing-library/react'
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

// Mock WorldCanvas - it shows when no skill is selected
vi.mock('@/components/world', () => ({
  WorldCanvas: ({ skills }: { skills: unknown[] }) => (
    <div data-testid="world-canvas">
      {skills.length === 0 ? 'No Skills Yet' : `${skills.length} skills in world`}
    </div>
  ),
}))

// Mock ViewOverlay and graph/world content components used in empty state
vi.mock('../shared/ViewOverlay', () => ({
  ViewOverlay: () => <div data-testid="view-overlay" />,
}))
vi.mock('@/components/world/WorldSettingsContent', () => ({
  WorldSettingsContent: () => <div />,
}))
vi.mock('@/components/world/WorldHelpContent', () => ({
  WorldHelpContent: () => <div />,
}))
vi.mock('@/components/graph/GraphView', () => ({
  GraphView: () => <div data-testid="graph-view" />,
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
  describe('empty state (world)', () => {
    it('should show world canvas when no skill is selected', () => {
      render(<SkillEditorPanel {...defaultProps} currentSkill={null} />)

      // Now shows world instead of empty message
      expect(screen.getByTestId('world-canvas')).toBeInTheDocument()
    })

    it('should show empty tree message when no skills available', () => {
      render(<SkillEditorPanel {...defaultProps} currentSkill={null} allSkills={[]} />)

      expect(screen.getByText('No Skills Yet')).toBeInTheDocument()
    })

    it('should show skill count when skills are available', () => {
      const skills = [createTestSkill({ id: '1' }), createTestSkill({ id: '2' })]

      render(
        <SkillEditorPanel
          {...defaultProps}
          currentSkill={null}
          allSkills={skills}
        />
      )

      expect(screen.getByText('2 skills in world')).toBeInTheDocument()
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
    it('renders the draft status chip on all viewport widths', () => {
      const skill = createTestSkill()
      render(<SkillEditorPanel {...defaultProps} currentSkill={skill} />)
      expect(screen.getByTestId('draft-status-chip')).toBeInTheDocument()
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
    it('exposes File path, Discard changes, and Delete skill items', () => {
      const skill = createTestSkill()
      render(
        <SkillEditorPanel
          {...defaultProps}
          currentSkill={skill}
          isDirty={true}
        />
      )
      fireEvent.click(screen.getByTestId('skill-more-menu'))
      expect(screen.getByTestId('skill-more-file-path')).toBeInTheDocument()
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
      fireEvent.click(screen.getByTestId('skill-more-menu'))
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
      fireEvent.click(screen.getByTestId('skill-more-menu'))
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
      fireEvent.click(screen.getByTestId('skill-more-menu'))
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
      fireEvent.click(screen.getByTestId('skill-more-menu'))
      expect(screen.getByText(/Deleting/)).toBeInTheDocument()
    })

    it('opens the file path popover when File path item is clicked', () => {
      const skill = createTestSkill()
      render(<SkillEditorPanel {...defaultProps} currentSkill={skill} />)
      fireEvent.click(screen.getByTestId('skill-more-menu'))
      fireEvent.click(screen.getByTestId('skill-more-file-path'))
      // The FilePathMenu popover contains a Filename label
      expect(screen.getByText('Filename')).toBeInTheDocument()
    })
  })
})
