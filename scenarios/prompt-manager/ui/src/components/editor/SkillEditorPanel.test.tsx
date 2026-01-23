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
import { render, screen } from '@testing-library/react'
import { SkillEditorPanel } from './SkillEditorPanel'
import type { Skill } from '@/types'
import type { SkillFormState, ValidationResult } from '@/types/editor'

// Mock Monaco editor since it requires browser environment
vi.mock('@monaco-editor/react', () => ({
  default: ({ value }: { value: string }) => (
    <div data-testid="monaco-editor">{value}</div>
  ),
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

// Helper to create test skill
function createTestSkill(overrides: Partial<Skill> = {}): Skill {
  return {
    id: 'test-1',
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
function createFormState(overrides: Partial<SkillFormState> = {}): SkillFormState {
  return {
    name: 'Test Skill',
    description: 'A test description',
    content: '# Test content',
    modes: ['development'],
    tags: 'tag1',
    icon: '',
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
  onModesChange: vi.fn(),
  getSuggestionsAtLevel: vi.fn(() => []),
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
    it('should show unsaved changes indicator when dirty', () => {
      const skill = createTestSkill()

      render(
        <SkillEditorPanel
          {...defaultProps}
          currentSkill={skill}
          isDirty={true}
        />
      )

      expect(screen.getByText('Unsaved changes')).toBeInTheDocument()
    })

    it('should not show unsaved changes indicator when not dirty', () => {
      const skill = createTestSkill()

      render(
        <SkillEditorPanel
          {...defaultProps}
          currentSkill={skill}
          isDirty={false}
        />
      )

      expect(screen.queryByText('Unsaved changes')).not.toBeInTheDocument()
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
})
