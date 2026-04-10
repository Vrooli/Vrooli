/**
 * useToolbarActions - Hook for editor toolbar actions.
 *
 * Centralizes all toolbar action handlers for the TipTap editor.
 */

import { useCallback } from 'react'
import type { Editor } from '@tiptap/react'

export interface ToolbarActions {
  // Headings
  toggleHeading1: () => void
  toggleHeading2: () => void
  toggleHeading3: () => void

  // Text formatting
  toggleBold: () => void
  toggleItalic: () => void
  toggleStrike: () => void
  toggleHighlight: () => void

  // Code
  toggleCode: () => void
  toggleCodeBlock: () => void

  // Lists
  toggleBulletList: () => void
  toggleOrderedList: () => void

  // Block elements
  toggleBlockquote: () => void
  insertHorizontalRule: () => void

  // State checks
  isHeading1Active: () => boolean
  isHeading2Active: () => boolean
  isHeading3Active: () => boolean
  isBoldActive: () => boolean
  isItalicActive: () => boolean
  isStrikeActive: () => boolean
  isHighlightActive: () => boolean
  isCodeActive: () => boolean
  isCodeBlockActive: () => boolean
  isBulletListActive: () => boolean
  isOrderedListActive: () => boolean
  isBlockquoteActive: () => boolean
  isLinkActive: () => boolean

  // Compound checks
  hasActiveHeading: () => boolean
  hasActiveTextFormat: () => boolean
}

/**
 * Hook providing toolbar action handlers for the TipTap editor.
 *
 * @param editor - The TipTap editor instance
 * @returns Object with action handlers and state checks
 */
export function useToolbarActions(editor: Editor | null): ToolbarActions {
  // Heading actions
  const toggleHeading1 = useCallback(() => {
    editor?.chain().focus().toggleHeading({ level: 1 }).run()
  }, [editor])

  const toggleHeading2 = useCallback(() => {
    editor?.chain().focus().toggleHeading({ level: 2 }).run()
  }, [editor])

  const toggleHeading3 = useCallback(() => {
    editor?.chain().focus().toggleHeading({ level: 3 }).run()
  }, [editor])

  // Text formatting actions
  const toggleBold = useCallback(() => {
    editor?.chain().focus().toggleBold().run()
  }, [editor])

  const toggleItalic = useCallback(() => {
    editor?.chain().focus().toggleItalic().run()
  }, [editor])

  const toggleStrike = useCallback(() => {
    editor?.chain().focus().toggleStrike().run()
  }, [editor])

  const toggleHighlight = useCallback(() => {
    editor?.chain().focus().toggleHighlight().run()
  }, [editor])

  // Code actions
  const toggleCode = useCallback(() => {
    editor?.chain().focus().toggleCode().run()
  }, [editor])

  const toggleCodeBlock = useCallback(() => {
    editor?.chain().focus().toggleCodeBlock().run()
  }, [editor])

  // List actions
  const toggleBulletList = useCallback(() => {
    editor?.chain().focus().toggleBulletList().run()
  }, [editor])

  const toggleOrderedList = useCallback(() => {
    editor?.chain().focus().toggleOrderedList().run()
  }, [editor])

  // Block element actions
  const toggleBlockquote = useCallback(() => {
    editor?.chain().focus().toggleBlockquote().run()
  }, [editor])

  const insertHorizontalRule = useCallback(() => {
    editor?.chain().focus().setHorizontalRule().run()
  }, [editor])

  // State checks
  const isHeading1Active = useCallback(() => {
    return editor?.isActive('heading', { level: 1 }) ?? false
  }, [editor])

  const isHeading2Active = useCallback(() => {
    return editor?.isActive('heading', { level: 2 }) ?? false
  }, [editor])

  const isHeading3Active = useCallback(() => {
    return editor?.isActive('heading', { level: 3 }) ?? false
  }, [editor])

  const isBoldActive = useCallback(() => {
    return editor?.isActive('bold') ?? false
  }, [editor])

  const isItalicActive = useCallback(() => {
    return editor?.isActive('italic') ?? false
  }, [editor])

  const isStrikeActive = useCallback(() => {
    return editor?.isActive('strike') ?? false
  }, [editor])

  const isHighlightActive = useCallback(() => {
    return editor?.isActive('highlight') ?? false
  }, [editor])

  const isCodeActive = useCallback(() => {
    return editor?.isActive('code') ?? false
  }, [editor])

  const isCodeBlockActive = useCallback(() => {
    return editor?.isActive('codeBlock') ?? false
  }, [editor])

  const isBulletListActive = useCallback(() => {
    return editor?.isActive('bulletList') ?? false
  }, [editor])

  const isOrderedListActive = useCallback(() => {
    return editor?.isActive('orderedList') ?? false
  }, [editor])

  const isBlockquoteActive = useCallback(() => {
    return editor?.isActive('blockquote') ?? false
  }, [editor])

  const isLinkActive = useCallback(() => {
    return editor?.isActive('link') ?? false
  }, [editor])

  // Compound checks
  const hasActiveHeading = useCallback(() => {
    return isHeading1Active() || isHeading2Active() || isHeading3Active()
  }, [isHeading1Active, isHeading2Active, isHeading3Active])

  const hasActiveTextFormat = useCallback(() => {
    return isBoldActive() || isItalicActive() || isStrikeActive() || isHighlightActive()
  }, [isBoldActive, isItalicActive, isStrikeActive, isHighlightActive])

  return {
    toggleHeading1,
    toggleHeading2,
    toggleHeading3,
    toggleBold,
    toggleItalic,
    toggleStrike,
    toggleHighlight,
    toggleCode,
    toggleCodeBlock,
    toggleBulletList,
    toggleOrderedList,
    toggleBlockquote,
    insertHorizontalRule,
    isHeading1Active,
    isHeading2Active,
    isHeading3Active,
    isBoldActive,
    isItalicActive,
    isStrikeActive,
    isHighlightActive,
    isCodeActive,
    isCodeBlockActive,
    isBulletListActive,
    isOrderedListActive,
    isBlockquoteActive,
    isLinkActive,
    hasActiveHeading,
    hasActiveTextFormat,
  }
}
