/**
 * useLinkManagement - Hook for managing link state in TipTap editor.
 *
 * Handles:
 * - Link dialog open/close state
 * - Current link data
 * - Link creation, update, and removal
 * - URL normalization (adding https:// if missing)
 */

import { useState, useCallback, useRef } from 'react'
import type { Editor } from '@tiptap/react'

export interface LinkData {
  /** The link URL */
  url: string
  /** The link text (optional, uses selection if not provided) */
  text?: string
}

export interface UseLinkManagementOptions {
  /** The TipTap editor instance */
  editor: Editor | null
}

export interface UseLinkManagementResult {
  /** Whether the link dialog is currently open */
  isDialogOpen: boolean
  /** The current link URL being edited */
  linkUrl: string
  /** Ref for the link input element */
  linkInputRef: React.RefObject<HTMLInputElement | null>
  /** Open the link dialog, optionally with existing URL */
  openDialog: () => void
  /** Close the link dialog */
  closeDialog: () => void
  /** Update the link URL value */
  setLinkUrl: (url: string) => void
  /** Save the current link to the editor */
  saveLink: () => void
  /** Remove the link from the current selection */
  removeLink: () => void
  /** Check if the current selection has a link */
  hasLink: boolean
}

/**
 * Normalize a URL by adding https:// if no protocol is specified.
 *
 * @param url - The URL to normalize
 * @returns The normalized URL with protocol
 */
function normalizeUrl(url: string): string {
  const trimmed = url.trim()
  if (!trimmed) return ''
  if (/^https?:\/\//i.test(trimmed)) return trimmed
  return `https://${trimmed}`
}

/**
 * Hook for managing link state in the TipTap editor.
 *
 * @param options - Configuration options
 * @returns Link management utilities
 */
export function useLinkManagement({
  editor,
}: UseLinkManagementOptions): UseLinkManagementResult {
  const [isDialogOpen, setIsDialogOpen] = useState(false)
  const [linkUrl, setLinkUrl] = useState('')
  const linkInputRef = useRef<HTMLInputElement>(null)

  /**
   * Open the link dialog.
   * Pre-populates with existing link URL if present.
   */
  const openDialog = useCallback(() => {
    if (!editor) return

    // Get existing link URL if present
    const existingUrl = editor.getAttributes('link').href as string | undefined
    setLinkUrl(existingUrl ?? '')
    setIsDialogOpen(true)

    // Focus the input after state update
    setTimeout(() => linkInputRef.current?.focus(), 0)
  }, [editor])

  /**
   * Close the link dialog.
   */
  const closeDialog = useCallback(() => {
    setIsDialogOpen(false)
    setLinkUrl('')
  }, [])

  /**
   * Save the current link to the editor.
   */
  const saveLink = useCallback(() => {
    if (!editor) return

    const normalizedUrl = normalizeUrl(linkUrl)
    if (normalizedUrl) {
      editor
        .chain()
        .focus()
        .extendMarkRange('link')
        .setLink({ href: normalizedUrl })
        .run()
    }

    closeDialog()
  }, [editor, linkUrl, closeDialog])

  /**
   * Remove the link from the current selection.
   */
  const removeLink = useCallback(() => {
    editor?.chain().focus().unsetLink().run()
  }, [editor])

  /**
   * Check if the current selection has a link.
   */
  const hasLink = editor?.isActive('link') ?? false

  return {
    isDialogOpen,
    linkUrl,
    linkInputRef,
    openDialog,
    closeDialog,
    setLinkUrl,
    saveLink,
    removeLink,
    hasLink,
  }
}
