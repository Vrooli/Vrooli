/**
 * useTipTapContent - Hook for managing content synchronization in TipTap editor.
 *
 * Handles:
 * - Converting markdown to HTML for TipTap rendering
 * - Converting HTML back to markdown for storage
 * - Preventing infinite loops from onChange callbacks
 * - Error handling for conversion failures
 */

import { useEffect, useRef, useCallback } from 'react'
import type { Editor } from '@tiptap/react'
import { isHtml, markdownToHtml, htmlToMarkdown } from '@/services/content'

export interface UseTipTapContentOptions {
  /** The external markdown value */
  value: string
  /** Callback when content changes */
  onChange: (value: string) => void
  /** The TipTap editor instance */
  editor: Editor | null
}

export interface UseTipTapContentResult {
  /** Whether the editor is currently loading/syncing content */
  isLoading: boolean
  /** Any error that occurred during conversion */
  error: string | null
  /** Force sync content from external value to editor */
  syncContent: () => void
  /** Get the initial HTML content for the editor */
  getInitialContent: () => string
  /** Handle editor update event */
  handleEditorUpdate: (updatedEditor: Editor) => void
}

/**
 * Hook for managing content synchronization between markdown and TipTap HTML.
 *
 * @param options - Configuration options
 * @returns Content synchronization utilities
 */
export function useTipTapContent({
  value,
  onChange,
  editor,
}: UseTipTapContentOptions): UseTipTapContentResult {
  // Track the last markdown value we output to avoid infinite loops
  const lastOutputRef = useRef<string>(value)
  const errorRef = useRef<string | null>(null)
  const isLoadingRef = useRef(false)

  /**
   * Get the initial HTML content for the editor.
   */
  const getInitialContent = useCallback((): string => {
    if (!value) return ''
    return isHtml(value) ? value : markdownToHtml(value)
  }, [value])

  /**
   * Handle editor content update.
   * Converts HTML to markdown and calls onChange.
   */
  const handleEditorUpdate = useCallback(
    (updatedEditor: Editor) => {
      try {
        const html = updatedEditor.getHTML()
        const markdown = htmlToMarkdown(html)
        lastOutputRef.current = markdown
        errorRef.current = null
        onChange(markdown)
      } catch (error) {
        errorRef.current =
          error instanceof Error ? error.message : 'Failed to convert content'
        console.error('TipTap content update error:', error)
      }
    },
    [onChange]
  )

  /**
   * Force sync content from external value to editor.
   */
  const syncContent = useCallback(() => {
    if (!editor) return

    try {
      isLoadingRef.current = true
      lastOutputRef.current = value
      const htmlContent = isHtml(value) ? value : markdownToHtml(value)
      editor.commands.setContent(htmlContent)
      errorRef.current = null
    } catch (error) {
      errorRef.current =
        error instanceof Error ? error.message : 'Failed to sync content'
      console.error('TipTap sync error:', error)
    } finally {
      isLoadingRef.current = false
    }
  }, [editor, value])

  // Update content when value changes externally
  useEffect(() => {
    if (!editor) return

    // Skip if the incoming value matches what we last output
    // This prevents infinite loops from our own onChange calls
    if (value === lastOutputRef.current) return

    // Update the ref and sync content
    lastOutputRef.current = value

    try {
      const htmlContent = isHtml(value) ? value : markdownToHtml(value)
      editor.commands.setContent(htmlContent)
      errorRef.current = null
    } catch (error) {
      errorRef.current =
        error instanceof Error ? error.message : 'Failed to update content'
      console.error('TipTap content update error:', error)
    }
  }, [editor, value])

  return {
    isLoading: isLoadingRef.current,
    error: errorRef.current,
    syncContent,
    getInitialContent,
    handleEditorUpdate,
  }
}
