/**
 * Tests for SkillContentEditor component.
 *
 * Tests cover:
 * - Editor type toggle (code vs WYSIWYG)
 * - localStorage persistence of editor preference
 * - Content change handling
 * - Error display
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { SkillContentEditor } from './SkillContentEditor'

// Mock Monaco Editor
vi.mock('@monaco-editor/react', () => ({
  default: ({ value, onChange, options }: {
    value: string
    onChange: (value: string | undefined) => void
    onMount?: (editor: unknown) => void
    options?: { readOnly?: boolean }
  }) => (
    <div data-testid="monaco-editor" data-readonly={options?.readOnly ?? false}>
      <textarea
        data-testid="monaco-textarea"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        readOnly={options?.readOnly ?? false}
      />
    </div>
  ),
}))

// Mock TipTap Editor
vi.mock('./TipTapEditor', () => ({
  TipTapEditor: ({ value, onChange, disabled, placeholder }: {
    value: string
    onChange: (value: string) => void
    disabled?: boolean
    placeholder?: string
  }) => (
    <div data-testid="tiptap-editor" data-disabled={disabled}>
      <textarea
        data-testid="tiptap-textarea"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
        placeholder={placeholder}
      />
    </div>
  ),
}))

describe('SkillContentEditor', () => {
  const defaultProps = {
    value: 'Initial content',
    onChange: vi.fn(),
  }

  beforeEach(() => {
    vi.clearAllMocks()
    // Reset localStorage mock
    vi.mocked(localStorage.getItem).mockReturnValue(null)
  })

  describe('editor rendering', () => {
    it('should render code editor by default', () => {
      render(<SkillContentEditor {...defaultProps} />)

      expect(screen.getByTestId('monaco-editor')).toBeInTheDocument()
      expect(screen.queryByTestId('tiptap-editor')).not.toBeInTheDocument()
    })

    it('should render WYSIWYG editor when stored preference is wysiwyg', () => {
      vi.mocked(localStorage.getItem).mockReturnValue('wysiwyg')

      render(<SkillContentEditor {...defaultProps} />)

      expect(screen.getByTestId('tiptap-editor')).toBeInTheDocument()
      expect(screen.queryByTestId('monaco-editor')).not.toBeInTheDocument()
    })

    it('should render label with required indicator', () => {
      render(<SkillContentEditor {...defaultProps} />)

      expect(screen.getByText('Content')).toBeInTheDocument()
      expect(screen.getByText('*')).toBeInTheDocument()
    })
  })

  describe('editor toggle', () => {
    it('should render toggle buttons', () => {
      render(<SkillContentEditor {...defaultProps} />)

      expect(screen.getByTitle('Code Editor (Monaco)')).toBeInTheDocument()
      expect(screen.getByTitle('Rich Text Editor (WYSIWYG)')).toBeInTheDocument()
    })

    it('should switch to WYSIWYG when rich text button is clicked', () => {
      render(<SkillContentEditor {...defaultProps} />)

      expect(screen.getByTestId('monaco-editor')).toBeInTheDocument()

      fireEvent.click(screen.getByTitle('Rich Text Editor (WYSIWYG)'))

      expect(screen.getByTestId('tiptap-editor')).toBeInTheDocument()
      expect(screen.queryByTestId('monaco-editor')).not.toBeInTheDocument()
    })

    it('should switch to code editor when code button is clicked', () => {
      vi.mocked(localStorage.getItem).mockReturnValue('wysiwyg')

      render(<SkillContentEditor {...defaultProps} />)

      expect(screen.getByTestId('tiptap-editor')).toBeInTheDocument()

      fireEvent.click(screen.getByTitle('Code Editor (Monaco)'))

      expect(screen.getByTestId('monaco-editor')).toBeInTheDocument()
      expect(screen.queryByTestId('tiptap-editor')).not.toBeInTheDocument()
    })

    it('should persist editor type to localStorage', () => {
      render(<SkillContentEditor {...defaultProps} />)

      fireEvent.click(screen.getByTitle('Rich Text Editor (WYSIWYG)'))

      expect(localStorage.setItem).toHaveBeenCalledWith('pm.editorType', 'wysiwyg')
    })
  })

  describe('content handling', () => {
    it('should display current value in code editor', () => {
      render(<SkillContentEditor {...defaultProps} value="Test content" />)

      const textarea = screen.getByTestId('monaco-textarea')
      expect((textarea as HTMLTextAreaElement).value).toBe('Test content')
    })

    it('should display current value in WYSIWYG editor', () => {
      vi.mocked(localStorage.getItem).mockReturnValue('wysiwyg')

      render(<SkillContentEditor {...defaultProps} value="Test content" />)

      const textarea = screen.getByTestId('tiptap-textarea')
      expect((textarea as HTMLTextAreaElement).value).toBe('Test content')
    })

    it('should call onChange when code editor content changes', () => {
      const onChange = vi.fn()
      render(<SkillContentEditor {...defaultProps} onChange={onChange} />)

      const textarea = screen.getByTestId('monaco-textarea')
      fireEvent.change(textarea, { target: { value: 'New content' } })

      expect(onChange).toHaveBeenCalledWith('New content')
    })

    it('should call onChange when WYSIWYG editor content changes', () => {
      vi.mocked(localStorage.getItem).mockReturnValue('wysiwyg')
      const onChange = vi.fn()

      render(<SkillContentEditor {...defaultProps} onChange={onChange} />)

      const textarea = screen.getByTestId('tiptap-textarea')
      fireEvent.change(textarea, { target: { value: 'New content' } })

      expect(onChange).toHaveBeenCalledWith('New content')
    })

  })

  describe('error display', () => {
    it('should display error message when error prop is provided', () => {
      render(<SkillContentEditor {...defaultProps} error="Content is required" />)

      expect(screen.getByText('Content is required')).toBeInTheDocument()
    })

    it('should not display error when no error prop', () => {
      render(<SkillContentEditor {...defaultProps} />)

      const errorElement = screen.queryByText(/is required/)
      expect(errorElement).not.toBeInTheDocument()
    })
  })

  describe('className prop', () => {
    it('should apply custom className', () => {
      const { container } = render(
        <SkillContentEditor {...defaultProps} className="custom-class" />
      )

      const wrapper = container.firstChild as HTMLElement
      expect(wrapper).toHaveClass('custom-class')
    })
  })
})
