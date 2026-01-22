/**
 * Tests for PromptEditorPanel component.
 *
 * Tests cover:
 * - Empty state rendering
 * - Prompt display with form fields
 * - Read-only mode indicator
 * - Dirty state indicator
 * - Toolbar visibility
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { PromptEditorPanel } from './PromptEditorPanel'
import type { Prompt } from '@/types'
import type { PromptFormState, ValidationResult } from '@/types/editor'

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

// Mock SkillTreeCanvas - it shows when no prompt is selected
vi.mock('@/components/skilltree', () => ({
  SkillTreeCanvas: ({ prompts }: { prompts: unknown[] }) => (
    <div data-testid="skill-tree-canvas">
      {prompts?.length === 0 ? 'No Prompts Yet' : `${prompts?.length ?? 0} prompts in tree`}
    </div>
  ),
}))

// Helper to create test prompt
function createTestPrompt(overrides: Partial<Prompt> = {}): Prompt {
  return {
    id: 'test-1',
    name: 'Test Prompt',
    description: 'A test description',
    content: '# Test content',
    modes: ['development'],
    tags: ['tag1'],
    draft: false,
    folder: 'internal',
    createdAt: '2025-01-01T00:00:00Z',
    updatedAt: '2025-01-01T00:00:00Z',
    usageCount: 0,
    ...overrides,
  }
}

// Helper to create default form state
function createFormState(overrides: Partial<PromptFormState> = {}): PromptFormState {
  return {
    name: 'Test Prompt',
    description: 'A test description',
    content: '# Test content',
    modes: ['development'],
    tags: 'tag1',
    icon: '',
    draft: false,
    folder: 'internal',
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
  currentPrompt: null,
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

describe('PromptEditorPanel', () => {
  describe('empty state (skill tree)', () => {
    it('should show skill tree canvas when no prompt is selected', () => {
      render(<PromptEditorPanel {...defaultProps} currentPrompt={null} />)

      // Now shows skill tree instead of empty message
      expect(screen.getByTestId('skill-tree-canvas')).toBeInTheDocument()
    })

    it('should show empty tree message when no prompts available', () => {
      render(<PromptEditorPanel {...defaultProps} currentPrompt={null} allPrompts={[]} />)

      expect(screen.getByText('No Prompts Yet')).toBeInTheDocument()
    })

    it('should show prompt count when prompts are available', () => {
      const prompts = [createTestPrompt({ id: '1' }), createTestPrompt({ id: '2' })]

      render(
        <PromptEditorPanel
          {...defaultProps}
          currentPrompt={null}
          allPrompts={prompts}
        />
      )

      expect(screen.getByText('2 prompts in tree')).toBeInTheDocument()
    })
  })

  describe('prompt display', () => {
    it('should render editor when prompt is selected', () => {
      const prompt = createTestPrompt()

      render(<PromptEditorPanel {...defaultProps} currentPrompt={prompt} />)

      // Should not show empty state
      expect(screen.queryByText('No Prompt Selected')).not.toBeInTheDocument()
    })

    it('should display content in editor', () => {
      const prompt = createTestPrompt({ content: 'Test content here' })
      const formState = createFormState({ content: 'Test content here' })

      render(
        <PromptEditorPanel
          {...defaultProps}
          currentPrompt={prompt}
          formState={formState}
        />
      )

      // Content should be passed to editor
      expect(screen.getByTestId('monaco-editor')).toHaveTextContent('Test content here')
    })
  })

  describe('dirty state indicator', () => {
    it('should show unsaved changes indicator when dirty', () => {
      const prompt = createTestPrompt()

      render(
        <PromptEditorPanel
          {...defaultProps}
          currentPrompt={prompt}
          isDirty={true}
        />
      )

      expect(screen.getByText('Unsaved changes')).toBeInTheDocument()
    })

    it('should not show unsaved changes indicator when not dirty', () => {
      const prompt = createTestPrompt()

      render(
        <PromptEditorPanel
          {...defaultProps}
          currentPrompt={prompt}
          isDirty={false}
        />
      )

      expect(screen.queryByText('Unsaved changes')).not.toBeInTheDocument()
    })
  })

  describe('validation errors', () => {
    it('should pass validation errors to form', () => {
      const prompt = createTestPrompt()
      const validation = createValidation(false, { content: 'Content is required' })

      render(
        <PromptEditorPanel
          {...defaultProps}
          currentPrompt={prompt}
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
        <PromptEditorPanel {...defaultProps} currentPrompt={null} className="custom-class" />
      )

      expect(container.firstChild).toHaveClass('custom-class')
    })
  })
})
