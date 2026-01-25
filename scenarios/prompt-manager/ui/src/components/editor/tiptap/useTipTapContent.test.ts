/**
 * Tests for useTipTapContent hook.
 *
 * These tests use real TipTap editors to test the actual content conversion flow,
 * ensuring that content survives the full round-trip:
 * Markdown → markdownToHtml → TipTap → htmlToMarkdown → Markdown
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { Editor } from '@tiptap/react'
import { useTipTapContent } from './useTipTapContent'
import { createTestEditor, destroyTestEditor } from '@/test/tiptap-test-utils'

describe('useTipTapContent', () => {
  describe('with real TipTap editor', () => {
    let editor: Editor
    let onChange: ReturnType<typeof vi.fn>

    beforeEach(() => {
      editor = createTestEditor()
      onChange = vi.fn()
    })

    afterEach(() => {
      destroyTestEditor(editor)
    })

    describe('getInitialContent', () => {
      it('should return empty string for empty value', () => {
        const { result } = renderHook(() =>
          useTipTapContent({
            value: '',
            onChange,
            editor,
          })
        )

        expect(result.current.getInitialContent()).toBe('')
      })

      it('should return HTML as-is if value is HTML', () => {
        const { result } = renderHook(() =>
          useTipTapContent({
            value: '<p>already html</p>',
            onChange,
            editor,
          })
        )

        expect(result.current.getInitialContent()).toBe('<p>already html</p>')
      })

      it('should convert markdown to HTML with bold formatting', () => {
        const { result } = renderHook(() =>
          useTipTapContent({
            value: '**bold text**',
            onChange,
            editor,
          })
        )

        const html = result.current.getInitialContent()
        expect(html).toContain('<strong>bold text</strong>')
      })

      it('should convert markdown to HTML with headings', () => {
        const { result } = renderHook(() =>
          useTipTapContent({
            value: '## Heading 2',
            onChange,
            editor,
          })
        )

        const html = result.current.getInitialContent()
        expect(html).toContain('<h2')
        expect(html).toContain('Heading 2')
      })

      it('should convert markdown to HTML with code blocks', () => {
        const { result } = renderHook(() =>
          useTipTapContent({
            value: '```typescript\nconst x = 1;\n```',
            onChange,
            editor,
          })
        )

        const html = result.current.getInitialContent()
        expect(html).toContain('<pre>')
        expect(html).toContain('<code')
        expect(html).toContain('language-typescript')
        expect(html).toContain('const x = 1;')
      })
    })

    describe('handleEditorUpdate', () => {
      it('should convert TipTap HTML to markdown and call onChange', () => {
        const { result } = renderHook(() =>
          useTipTapContent({
            value: '',
            onChange,
            editor,
          })
        )

        // Set some content in the editor
        editor.commands.setContent('<p><strong>bold content</strong></p>')

        act(() => {
          result.current.handleEditorUpdate(editor)
        })

        expect(onChange).toHaveBeenCalled()
        const markdownOutput = onChange.mock.calls[0]?.[0] as string
        expect(markdownOutput).toContain('**bold content**')
      })

      it('should preserve code block language through update', () => {
        const { result } = renderHook(() =>
          useTipTapContent({
            value: '',
            onChange,
            editor,
          })
        )

        // Set a code block with language
        editor.commands.setContent(
          '<pre class="language-typescript"><code class="language-typescript">const x = 1;</code></pre>'
        )

        act(() => {
          result.current.handleEditorUpdate(editor)
        })

        expect(onChange).toHaveBeenCalled()
        const markdownOutput = onChange.mock.calls[0]?.[0] as string
        expect(markdownOutput).toContain('```typescript')
        expect(markdownOutput).toContain('const x = 1;')
      })

      it('should preserve links through update', () => {
        const { result } = renderHook(() =>
          useTipTapContent({
            value: '',
            onChange,
            editor,
          })
        )

        editor.commands.setContent(
          '<p><a href="https://example.com">Link text</a></p>'
        )

        act(() => {
          result.current.handleEditorUpdate(editor)
        })

        expect(onChange).toHaveBeenCalled()
        const markdownOutput = onChange.mock.calls[0]?.[0] as string
        expect(markdownOutput).toContain('[Link text](https://example.com)')
      })
    })

    describe('syncContent', () => {
      it('should sync markdown content to editor as HTML', () => {
        const { result } = renderHook(() =>
          useTipTapContent({
            value: '**bold text**',
            onChange,
            editor,
          })
        )

        act(() => {
          result.current.syncContent()
        })

        const editorHtml = editor.getHTML()
        expect(editorHtml).toContain('<strong>bold text</strong>')
      })

      it('should handle complex markdown content', () => {
        const { result } = renderHook(() =>
          useTipTapContent({
            value: '## Heading\n\n**bold** and *italic*\n\n```ts\ncode\n```',
            onChange,
            editor,
          })
        )

        act(() => {
          result.current.syncContent()
        })

        const editorHtml = editor.getHTML()
        expect(editorHtml).toContain('<h2')
        expect(editorHtml).toContain('<strong>bold</strong>')
        expect(editorHtml).toContain('<em>italic</em>')
        expect(editorHtml).toContain('<pre')
      })
    })

    describe('round-trip content preservation', () => {
      it('should preserve bold formatting through full cycle', () => {
        const originalMarkdown = '**bold text**'

        const { result } = renderHook(() =>
          useTipTapContent({
            value: originalMarkdown,
            onChange,
            editor,
          })
        )

        // Sync the original markdown to editor
        act(() => {
          result.current.syncContent()
        })

        // Get the update from editor
        act(() => {
          result.current.handleEditorUpdate(editor)
        })

        expect(onChange).toHaveBeenCalled()
        const outputMarkdown = onChange.mock.calls[0]?.[0] as string
        expect(outputMarkdown).toContain('**bold text**')
        expect(outputMarkdown).not.toContain('\\*')
      })

      it('should preserve code block language through full cycle', () => {
        const originalMarkdown = '```typescript\nconst x: number = 1;\n```'

        const { result } = renderHook(() =>
          useTipTapContent({
            value: originalMarkdown,
            onChange,
            editor,
          })
        )

        act(() => {
          result.current.syncContent()
        })

        act(() => {
          result.current.handleEditorUpdate(editor)
        })

        expect(onChange).toHaveBeenCalled()
        const outputMarkdown = onChange.mock.calls[0]?.[0] as string
        expect(outputMarkdown).toContain('```typescript')
        expect(outputMarkdown).toContain('const x: number = 1;')
      })

      it('should preserve headings through full cycle', () => {
        const originalMarkdown = '## My Heading'

        const { result } = renderHook(() =>
          useTipTapContent({
            value: originalMarkdown,
            onChange,
            editor,
          })
        )

        act(() => {
          result.current.syncContent()
        })

        act(() => {
          result.current.handleEditorUpdate(editor)
        })

        expect(onChange).toHaveBeenCalled()
        const outputMarkdown = onChange.mock.calls[0]?.[0] as string
        expect(outputMarkdown.trim()).toBe('## My Heading')
        expect(outputMarkdown).not.toContain('\\#')
      })
    })

    describe('initial state', () => {
      it('should have correct initial state', () => {
        const { result } = renderHook(() =>
          useTipTapContent({
            value: 'initial',
            onChange,
            editor,
          })
        )

        expect(result.current.isLoading).toBe(false)
        expect(result.current.error).toBe(null)
      })
    })
  })

  describe('edge cases and error handling', () => {
    let onChange: ReturnType<typeof vi.fn>

    beforeEach(() => {
      onChange = vi.fn()
    })

    it('should do nothing if editor is null', () => {
      const { result } = renderHook(() =>
        useTipTapContent({
          value: 'test value',
          onChange,
          editor: null,
        })
      )

      // Should not throw when syncing with null editor
      act(() => {
        result.current.syncContent()
      })

      // onChange should not be called
      expect(onChange).not.toHaveBeenCalled()
    })

    it('should handle errors in handleEditorUpdate gracefully', () => {
      const consoleErrorSpy = vi
        .spyOn(console, 'error')
        .mockImplementation(() => {})

      // Create a mock editor that throws
      const brokenEditor = {
        getHTML: () => {
          throw new Error('Editor error')
        },
      } as unknown as Editor

      const { result } = renderHook(() =>
        useTipTapContent({
          value: '',
          onChange,
          editor: brokenEditor,
        })
      )

      act(() => {
        result.current.handleEditorUpdate(brokenEditor)
      })

      expect(consoleErrorSpy).toHaveBeenCalled()
      consoleErrorSpy.mockRestore()
    })
  })
})
