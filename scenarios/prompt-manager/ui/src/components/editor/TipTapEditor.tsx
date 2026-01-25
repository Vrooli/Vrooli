/**
 * TipTapEditor - WYSIWYG editor built on TipTap.
 *
 * Features:
 * - Rich text formatting (bold, italic, headings, lists, etc.)
 * - Code blocks with syntax highlighting
 * - Placeholder text
 * - Clean, dark-themed UI
 *
 * This component has been refactored to use extracted hooks and components
 * for better separation of concerns and testability.
 */

import { useEditor, EditorContent, type Editor } from '@tiptap/react'
import { useEffect, useRef, useState } from 'react'
import { cn } from '@/lib/utils'
import {
  setLinkHoverCallback,
  LinkPreviewTooltip,
  type LinkHoverEvent,
} from './linkpreview'
import { createTipTapExtensions, getEditorProseClasses, useTipTapContent } from './tiptap'
import { useLinkManagement, LinkDialog } from './links'
import { TipTapToolbar } from './toolbar'
import { type EditorType } from './SkillContentEditor'

export interface TipTapEditorProps {
  /** The markdown content value */
  value: string
  /** Callback when content changes */
  onChange: (value: string) => void
  /** Whether the editor is disabled */
  disabled?: boolean
  /** Placeholder text shown when editor is empty */
  placeholder?: string
  /** Current editor type (code/wysiwyg) */
  editorType?: EditorType
  /** Callback when editor type changes */
  onEditorTypeChange?: (type: EditorType) => void
  /** Additional CSS classes */
  className?: string
}

/**
 * TipTap WYSIWYG editor component.
 *
 * Converts markdown to HTML for editing and back to markdown for storage.
 * Uses the useTipTapContent hook for content synchronization with
 * better error handling and infinite loop prevention.
 */
export function TipTapEditor({
  value,
  onChange,
  disabled = false,
  placeholder = 'Start writing your prompt...',
  editorType,
  onEditorTypeChange,
  className,
}: TipTapEditorProps) {
  // Track if we've set initial content to avoid re-setting on every render
  const initializedRef = useRef(false)

  // Create the editor first (without content - will be set separately)
  const editor = useEditor({
    extensions: createTipTapExtensions({ placeholder }),
    content: '', // Initial content will be set by hook
    editable: !disabled,
    editorProps: {
      attributes: {
        class: getEditorProseClasses(),
      },
    },
  })

  // Use the content synchronization hook
  // This handles markdown <-> HTML conversion with error handling
  const { getInitialContent, handleEditorUpdate, error } = useTipTapContent({
    value,
    onChange,
    editor,
  })

  // Set initial content once when editor is first ready
  useEffect(() => {
    if (!editor || initializedRef.current) return

    const initialContent = getInitialContent()
    if (initialContent) {
      editor.commands.setContent(initialContent)
    }
    initializedRef.current = true
  }, [editor, getInitialContent])

  // Set up the onUpdate handler
  useEffect(() => {
    if (!editor) return

    const updateHandler = ({ editor: updatedEditor }: { editor: Editor }) => {
      handleEditorUpdate(updatedEditor)
    }
    editor.on('update', updateHandler)

    return () => {
      editor.off('update', updateHandler)
    }
  }, [editor, handleEditorUpdate])

  // Update editable state when disabled changes
  useEffect(() => {
    if (editor) {
      editor.setEditable(!disabled)
    }
  }, [editor, disabled])

  // Link management
  const linkManagement = useLinkManagement({ editor })

  // Link preview state
  const [linkPreview, setLinkPreview] = useState<{
    url: string
    position: { x: number; y: number }
  } | null>(null)

  // Set up link hover callback
  useEffect(() => {
    const handleLinkHover = (event: LinkHoverEvent) => {
      if (event.type === 'hover') {
        setLinkPreview({
          url: event.url,
          position: event.position,
        })
      } else {
        setLinkPreview(null)
      }
    }

    setLinkHoverCallback(handleLinkHover)

    return () => {
      setLinkHoverCallback(null)
    }
  }, [])

  // Loading state
  if (!editor) {
    return (
      <div className={cn('flex-1 bg-card rounded-lg border border-border', className)}>
        <div className="flex items-center justify-center h-full text-muted-foreground text-sm">
          Loading editor...
        </div>
      </div>
    )
  }

  return (
    <div className={cn('flex flex-col bg-card', className)}>
      {/* Toolbar */}
      {!disabled && (
        <TipTapToolbar
          editor={editor}
          onOpenLinkDialog={linkManagement.openDialog}
          onRemoveLink={linkManagement.removeLink}
          editorType={editorType}
          onEditorTypeChange={onEditorTypeChange}
        />
      )}

      {/* Link input dialog */}
      {linkManagement.isDialogOpen && (
        <LinkDialog
          linkUrl={linkManagement.linkUrl}
          onLinkUrlChange={linkManagement.setLinkUrl}
          onSave={linkManagement.saveLink}
          onClose={linkManagement.closeDialog}
          inputRef={linkManagement.linkInputRef}
        />
      )}

      {/* Error display */}
      {error && (
        <div className="px-4 py-2 text-sm text-destructive bg-destructive/10 border-t border-destructive/20">
          {error}
        </div>
      )}

      {/* Editor content */}
      <div className={cn('flex-1 overflow-y-auto', disabled && 'opacity-50')}>
        <EditorContent editor={editor} />
      </div>

      {/* Link preview tooltip */}
      {linkPreview && (
        <LinkPreviewTooltip
          url={linkPreview.url}
          position={linkPreview.position}
          onClose={() => setLinkPreview(null)}
        />
      )}
    </div>
  )
}
