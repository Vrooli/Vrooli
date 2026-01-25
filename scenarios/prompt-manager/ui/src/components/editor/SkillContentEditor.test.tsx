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
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { SkillContentEditor } from './SkillContentEditor'

// Mock Monaco Editor with useMonaco hook
const mockSetModelMarkers = vi.fn()
const mockMonaco = {
  MarkerSeverity: {
    Warning: 4,
    Error: 8,
  },
  editor: {
    setModelMarkers: mockSetModelMarkers,
  },
}

vi.mock('@monaco-editor/react', () => ({
  default: ({ value, onChange, options, onMount }: {
    value: string
    onChange: (value: string | undefined) => void
    onMount?: (editor: unknown) => void
    options?: { readOnly?: boolean }
  }) => {
    // Simulate onMount to set editorRef
    if (onMount) {
      setTimeout(() => {
        onMount({
          focus: vi.fn(),
          getModel: () => ({
            isDisposed: () => false,
          }),
        })
      }, 0)
    }
    return (
      <div data-testid="monaco-editor" data-readonly={options?.readOnly ?? false}>
        <textarea
          data-testid="monaco-textarea"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          readOnly={options?.readOnly ?? false}
        />
      </div>
    )
  },
  useMonaco: () => mockMonaco,
}))

// Mock TipTap Editor - includes EditorToggle when props are provided
vi.mock('./TipTapEditor', () => ({
  TipTapEditor: ({ value, onChange, disabled, placeholder, editorType, onEditorTypeChange }: {
    value: string
    onChange: (value: string) => void
    disabled?: boolean
    placeholder?: string
    editorType?: string
    onEditorTypeChange?: (type: string) => void
  }) => (
    <div data-testid="tiptap-editor" data-disabled={disabled}>
      {/* Render EditorToggle when props are provided (matching real TipTapEditor behavior) */}
      {editorType && onEditorTypeChange && (
        <div className="editor-toggle">
          <button
            type="button"
            onClick={() => onEditorTypeChange('code')}
            title="Code Editor (Monaco)"
          >
            Code
          </button>
          <button
            type="button"
            onClick={() => onEditorTypeChange('wysiwyg')}
            title="Rich Text Editor (WYSIWYG)"
          >
            Rich Text
          </button>
        </div>
      )}
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
    mockSetModelMarkers.mockClear()
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

  describe('markdown validation', () => {
    it('should show warning banner when switching to Rich mode with escaped code fences', async () => {
      const markdown = '\\`\\`\\`bash\ncode\n\\`\\`\\`'
      render(<SkillContentEditor value={markdown} onChange={vi.fn()} />)

      // Should be in code mode initially
      expect(screen.getByTestId('monaco-editor')).toBeInTheDocument()

      // Switch to Rich mode
      fireEvent.click(screen.getByTitle('Rich Text Editor (WYSIWYG)'))

      // Should show warning banner
      await waitFor(() => {
        expect(screen.getByText(/markdown issue.*detected/i)).toBeInTheDocument()
        expect(screen.getByText(/may not display correctly/i)).toBeInTheDocument()
      })
    })

    it('should not show warning for valid markdown', () => {
      const markdown = '```bash\ncode\n```'
      render(<SkillContentEditor value={markdown} onChange={vi.fn()} />)

      // Switch to Rich mode
      fireEvent.click(screen.getByTitle('Rich Text Editor (WYSIWYG)'))

      // Should not show warning banner
      expect(screen.queryByText(/markdown issue.*detected/i)).not.toBeInTheDocument()
    })

    it('should allow dismissing the warning banner', async () => {
      const markdown = '\\`\\`\\`bash\ncode\n\\`\\`\\`'
      render(<SkillContentEditor value={markdown} onChange={vi.fn()} />)

      // Switch to Rich mode
      fireEvent.click(screen.getByTitle('Rich Text Editor (WYSIWYG)'))

      // Wait for warning to appear
      await waitFor(() => {
        expect(screen.getByText(/markdown issue.*detected/i)).toBeInTheDocument()
      })

      // Dismiss the warning
      fireEvent.click(screen.getByLabelText('Dismiss warning'))

      // Warning should be gone
      await waitFor(() => {
        expect(screen.queryByText(/markdown issue.*detected/i)).not.toBeInTheDocument()
      })
    })

    it('should detect multiple validation issues', () => {
      // Test with multiple escaped fences
      const markdown = '\\`\\`\\`bash\ncode1\n\\`\\`\\`\n\nMore text\n\n\\`\\`\\`python\ncode2\n\\`\\`\\`'
      render(<SkillContentEditor value={markdown} onChange={vi.fn()} />)

      // Switch to Rich mode - should show warning about multiple issues
      fireEvent.click(screen.getByTitle('Rich Text Editor (WYSIWYG)'))

      // Should show warning mentioning multiple issues
      expect(screen.getByText(/4 markdown issues detected/i)).toBeInTheDocument()
    })

    it('should detect extended code fences (4+ backticks)', () => {
      // Test with 4-backtick fence that will be corrupted
      const markdown = '````markdown\n```bash\ncode\n```\n````'
      render(<SkillContentEditor value={markdown} onChange={vi.fn()} />)

      // Switch to Rich mode - should show warning
      fireEvent.click(screen.getByTitle('Rich Text Editor (WYSIWYG)'))

      // Should show warning about the issue
      expect(screen.getByText(/1 markdown issue detected/i)).toBeInTheDocument()
    })
  })
})
