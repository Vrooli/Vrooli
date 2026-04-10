/**
 * SkillContentEditor Mode Switch Integration Tests
 *
 * These tests use REAL TipTap (not mocked) to verify content preservation
 * during mode switches. This is critical for catching content corruption bugs.
 *
 * Only Monaco is mocked since it doesn't affect content conversion.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import { SkillContentEditor } from './SkillContentEditor'

// Mock useResolvedTheme (used by SkillContentEditor)
vi.mock('@/hooks/use-theme', () => ({
  useResolvedTheme: vi.fn(() => 'dark'),
}))

// Only mock Monaco, NOT TipTap - we want to test the real conversion flow
vi.mock('@monaco-editor/react', () => ({
  default: ({ value }: { value: string }) => (
    <div data-testid="monaco-editor">
      <pre data-testid="monaco-content">{value}</pre>
    </div>
  ),
  useMonaco: () => ({
    MarkerSeverity: { Warning: 4, Error: 8 },
    editor: { setModelMarkers: vi.fn() },
  }),
}))

describe('SkillContentEditor Mode Switch Integration', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // Start with code editor
    vi.mocked(localStorage.getItem).mockReturnValue('code')
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('paragraph structure preservation', () => {
    it('should preserve paragraph structure through mode switch', async () => {
      const markdown = 'Line 1\n\nLine 2\n\nLine 3'
      const onChange = vi.fn()

      render(<SkillContentEditor value={markdown} onChange={onChange} />)

      // Verify we start in code mode
      expect(screen.getByTestId('monaco-editor')).toBeInTheDocument()

      // Switch to WYSIWYG
      act(() => {
        fireEvent.click(screen.getByTitle('Switch to Rich Text'))
      })

      // Wait for TipTap to initialize
      await waitFor(
        () => {
          const editor = document.querySelector('.ProseMirror')
          expect(editor).toBeInTheDocument()
        },
        { timeout: 3000 }
      )

      // Allow TipTap to process the content and trigger onChange
      // TipTap triggers an update when content is loaded
      await waitFor(
        () => {
          // onChange should have been called with the converted content
          expect(onChange).toHaveBeenCalled()
        },
        { timeout: 3000 }
      )

      // Get the last call to onChange
      const lastCall = onChange.mock.calls[onChange.mock.calls.length - 1]
      if (lastCall) {
        const resultMarkdown = lastCall[0]

        // Verify content is preserved
        expect(resultMarkdown).toContain('Line 1')
        expect(resultMarkdown).toContain('Line 2')
        expect(resultMarkdown).toContain('Line 3')

        // Verify paragraphs are separated by blank lines (not collapsed)
        const doubleNewlineCount = (resultMarkdown.match(/\n\n/g) || []).length
        expect(doubleNewlineCount).toBeGreaterThanOrEqual(2)
      }
    })

    it('should preserve complex document structure through mode switch', async () => {
      const markdown = `# Heading

This is paragraph one.

This is paragraph two.

## Code Example

\`\`\`typescript
const x = 1;
\`\`\`

- List item 1
- List item 2`

      const onChange = vi.fn()

      render(<SkillContentEditor value={markdown} onChange={onChange} />)

      // Switch to WYSIWYG
      act(() => {
        fireEvent.click(screen.getByTitle('Switch to Rich Text'))
      })

      // Wait for TipTap to initialize
      await waitFor(
        () => {
          const editor = document.querySelector('.ProseMirror')
          expect(editor).toBeInTheDocument()
        },
        { timeout: 3000 }
      )

      // Allow TipTap to process the content
      await waitFor(
        () => {
          expect(onChange).toHaveBeenCalled()
        },
        { timeout: 3000 }
      )

      // Get the last call
      const lastCall = onChange.mock.calls[onChange.mock.calls.length - 1]
      if (lastCall) {
        const resultMarkdown = lastCall[0]

        // Verify all content is preserved
        expect(resultMarkdown).toContain('# Heading')
        expect(resultMarkdown).toContain('This is paragraph one.')
        expect(resultMarkdown).toContain('This is paragraph two.')
        expect(resultMarkdown).toContain('## Code Example')
        expect(resultMarkdown).toContain('```typescript')
        expect(resultMarkdown).toContain('List item 1')
        expect(resultMarkdown).toContain('List item 2')

        // Verify no escaping corruption
        expect(resultMarkdown).not.toContain('\\*')
        expect(resultMarkdown).not.toContain('\\#')
        expect(resultMarkdown).not.toContain('\\`')
      }
    })
  })

  describe('formatting preservation', () => {
    it('should preserve formatting through mode switch', async () => {
      const markdown = '**Bold** and *italic* and `code`'
      const onChange = vi.fn()

      render(<SkillContentEditor value={markdown} onChange={onChange} />)

      // Switch to WYSIWYG
      act(() => {
        fireEvent.click(screen.getByTitle('Switch to Rich Text'))
      })

      await waitFor(
        () => {
          expect(onChange).toHaveBeenCalled()
        },
        { timeout: 3000 }
      )

      const lastCall = onChange.mock.calls[onChange.mock.calls.length - 1]
      if (lastCall) {
        const resultMarkdown = lastCall[0]

        expect(resultMarkdown).toContain('**Bold**')
        expect(resultMarkdown).toContain('*italic*')
        expect(resultMarkdown).toContain('`code`')

        // No escape character corruption
        expect(resultMarkdown).not.toContain('\\*')
        expect(resultMarkdown).not.toContain('\\`')
      }
    })
  })

  describe('code block preservation', () => {
    it('should preserve code blocks with language through mode switch', async () => {
      const markdown = '```typescript\nconst x: number = 1;\nconsole.log(x);\n```'
      const onChange = vi.fn()

      render(<SkillContentEditor value={markdown} onChange={onChange} />)

      // Switch to WYSIWYG
      act(() => {
        fireEvent.click(screen.getByTitle('Switch to Rich Text'))
      })

      await waitFor(
        () => {
          expect(onChange).toHaveBeenCalled()
        },
        { timeout: 3000 }
      )

      const lastCall = onChange.mock.calls[onChange.mock.calls.length - 1]
      if (lastCall) {
        const resultMarkdown = lastCall[0]

        expect(resultMarkdown).toContain('```typescript')
        expect(resultMarkdown).toContain('const x: number = 1;')
        expect(resultMarkdown).toContain('console.log(x);')
      }
    })
  })

  describe('idempotency through multiple mode switches', () => {
    it('should stabilize content after first mode switch', async () => {
      const markdown = '## Heading\n\n**Bold text** and *italic text*'
      const onChange = vi.fn()

      const { rerender } = render(
        <SkillContentEditor value={markdown} onChange={onChange} />
      )

      // First switch: Code → WYSIWYG
      act(() => {
        fireEvent.click(screen.getByTitle('Switch to Rich Text'))
      })

      await waitFor(
        () => {
          expect(onChange).toHaveBeenCalled()
        },
        { timeout: 3000 }
      )

      const firstSwitchResult =
        onChange.mock.calls[onChange.mock.calls.length - 1]?.[0]

      // Switch back: WYSIWYG → Code
      act(() => {
        fireEvent.click(screen.getByTitle('Switch to Code'))
      })

      // Update the value to what we got from the first switch
      if (firstSwitchResult) {
        onChange.mockClear()

        rerender(
          <SkillContentEditor value={firstSwitchResult} onChange={onChange} />
        )

        // Switch to WYSIWYG again
        act(() => {
          fireEvent.click(screen.getByTitle('Switch to Rich Text'))
        })

        await waitFor(
          () => {
            expect(onChange).toHaveBeenCalled()
          },
          { timeout: 3000 }
        )

        const secondSwitchResult =
          onChange.mock.calls[onChange.mock.calls.length - 1]?.[0]

        // Content should be stable (no further changes after first normalization)
        expect(secondSwitchResult).toBe(firstSwitchResult)
      }
    })

    it('should not accumulate escape characters through multiple switches', async () => {
      const markdown = '**bold** and *italic* and [link](url)'
      const onChange = vi.fn()

      const { rerender } = render(
        <SkillContentEditor value={markdown} onChange={onChange} />
      )

      let currentValue = markdown

      // Perform 3 round trips
      for (let i = 0; i < 3; i++) {
        onChange.mockClear()

        // Switch to WYSIWYG
        act(() => {
          fireEvent.click(screen.getByTitle('Switch to Rich Text'))
        })

        await waitFor(
          () => {
            expect(onChange).toHaveBeenCalled()
          },
          { timeout: 3000 }
        )

        const result = onChange.mock.calls[onChange.mock.calls.length - 1]?.[0]
        if (result) {
          currentValue = result
        }

        // Switch back to Code
        act(() => {
          fireEvent.click(screen.getByTitle('Switch to Code'))
        })

        // Update with the new value
        rerender(
          <SkillContentEditor value={currentValue} onChange={onChange} />
        )
      }

      // Final content should not have accumulated escape characters
      expect(currentValue).not.toContain('\\*')
      expect(currentValue).not.toContain('\\[')
      expect(currentValue).not.toContain('\\#')

      // Original formatting should be preserved
      expect(currentValue).toContain('**bold**')
      expect(currentValue).toContain('*italic*')
      expect(currentValue).toContain('[link](url)')
    })
  })
})
