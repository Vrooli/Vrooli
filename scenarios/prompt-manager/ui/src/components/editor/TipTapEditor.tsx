/**
 * TipTapEditor - WYSIWYG editor built on TipTap.
 *
 * Features:
 * - Rich text formatting (bold, italic, headings, lists, etc.)
 * - Code blocks with syntax highlighting
 * - Placeholder text
 * - Clean, dark-themed UI
 */

import { useEditor, EditorContent, type Editor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'
import Highlight from '@tiptap/extension-highlight'
import Typography from '@tiptap/extension-typography'
import { CodeBlockExtension } from './codeblock'
import { InlineCodeExtension } from './inlinecode'
import {
  LinkPreviewExtension,
  setLinkHoverCallback,
  LinkPreviewTooltip,
  type LinkHoverEvent,
} from './linkpreview'
import { useEffect, useCallback, useRef, useState } from 'react'
import {
  Bold,
  Italic,
  Strikethrough,
  Code,
  Heading1,
  Heading2,
  Heading3,
  List,
  ListOrdered,
  Quote,
  Minus,
  Undo,
  Redo,
  FileCode,
  Highlighter,
  Link as LinkIcon,
  Unlink,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { isHtml, markdownToHtml, htmlToMarkdown } from '@/services/contentConverter'

interface TipTapEditorProps {
  value: string
  onChange: (value: string) => void
  disabled?: boolean
  placeholder?: string
  className?: string
}

interface ToolbarButtonProps {
  onClick: () => void
  isActive?: boolean
  disabled?: boolean
  title: string
  children: React.ReactNode
}

function ToolbarButton({ onClick, isActive, disabled, title, children }: ToolbarButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      title={title}
      className={cn(
        'p-1.5 rounded transition-colors',
        isActive
          ? 'bg-primary/30 text-primary'
          : 'text-muted-foreground hover:text-foreground hover:bg-muted',
        disabled && 'opacity-50 cursor-not-allowed'
      )}
    >
      {children}
    </button>
  )
}

function ToolbarDivider() {
  return <div className="w-px h-6 bg-border mx-1" />
}

export function TipTapEditor({
  value,
  onChange,
  disabled = false,
  placeholder = 'Start writing your prompt...',
  className,
}: TipTapEditorProps) {
  // Track the last markdown value we output to avoid infinite loops
  // when comparing incoming value with editor state
  const lastOutputRef = useRef<string>(value)

  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        heading: {
          levels: [1, 2, 3],
        },
        // Disable default codeBlock - we use our custom extension
        codeBlock: false,
        // Disable default code mark - we use our custom extension with copy button
        code: false,
      }),
      // Custom inline code with copy button on hover
      InlineCodeExtension.configure({
        HTMLAttributes: {
          class: 'bg-muted rounded px-1.5 py-0.5 font-mono text-sm text-primary',
        },
      }),
      // Custom code block with syntax highlighting and copy button
      CodeBlockExtension.configure({
        languages: [
          '',
          'typescript',
          'javascript',
          'python',
          'go',
          'json',
          'bash',
          'sql',
          'html',
          'css',
          'yaml',
          'rust',
          'java',
          'cpp',
          'ruby',
        ],
      }),
      Placeholder.configure({
        placeholder,
        emptyEditorClass: 'is-editor-empty',
      }),
      Highlight.configure({
        HTMLAttributes: {
          class: 'bg-yellow-500/30 rounded px-0.5',
        },
      }),
      Typography,
      LinkPreviewExtension.configure({
        openOnClick: false,
        HTMLAttributes: {
          class: 'text-primary underline hover:text-primary/80 cursor-pointer',
        },
      }),
    ],
    // Convert markdown to HTML for initial content since TipTap works with HTML
    content: isHtml(value) ? value : markdownToHtml(value),
    editable: !disabled,
    editorProps: {
      attributes: {
        class: cn(
          'prose dark:prose-invert prose-sm max-w-none',
          'focus:outline-none min-h-[200px] p-4',
          // Heading styles - distinct sizes for visual hierarchy
          'prose-h1:text-2xl prose-h1:font-bold prose-h1:text-foreground prose-h1:mt-6 prose-h1:mb-4',
          'prose-h2:text-xl prose-h2:font-bold prose-h2:text-foreground prose-h2:mt-5 prose-h2:mb-3',
          'prose-h3:text-lg prose-h3:font-semibold prose-h3:text-foreground prose-h3:mt-4 prose-h3:mb-2',
          'prose-p:text-muted-foreground prose-p:leading-relaxed',
          'prose-a:text-primary prose-a:no-underline hover:prose-a:underline',
          'prose-strong:text-foreground prose-em:text-foreground/90',
          'prose-code:text-primary prose-code:bg-muted',
          'prose-pre:bg-muted prose-pre:rounded-lg',
          'prose-blockquote:border-primary prose-blockquote:text-muted-foreground',
          'prose-ul:text-muted-foreground prose-ol:text-muted-foreground',
          'prose-li:text-muted-foreground'
        ),
      },
    },
    onUpdate: ({ editor: updatedEditor }: { editor: Editor }) => {
      // Get HTML content and convert to markdown for storage
      const html = updatedEditor.getHTML()
      // Convert HTML to markdown so content is stored in markdown format
      const markdown = htmlToMarkdown(html)
      // Track the output to avoid infinite loops in useEffect
      lastOutputRef.current = markdown
      onChange(markdown)
    },
  })

  // Update content when value changes externally
  // Only update if the incoming value is different from what we last output
  useEffect(() => {
    if (!editor) return

    // Skip if the incoming value matches what we last output
    // This prevents infinite loops from our own onChange calls
    if (value === lastOutputRef.current) return

    // Update the ref to track this new incoming value
    lastOutputRef.current = value

    // Convert markdown to HTML if needed, then set content
    const htmlContent = isHtml(value) ? value : markdownToHtml(value)
    editor.commands.setContent(htmlContent)
  }, [editor, value])

  // Update editable state
  useEffect(() => {
    if (editor) {
      editor.setEditable(!disabled)
    }
  }, [editor, disabled])

  const setHeading = useCallback(
    (level: 1 | 2 | 3) => {
      editor?.chain().focus().toggleHeading({ level }).run()
    },
    [editor]
  )

  // Link dialog state
  const [showLinkInput, setShowLinkInput] = useState(false)
  const [linkUrl, setLinkUrl] = useState('')
  const linkInputRef = useRef<HTMLInputElement>(null)

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

  const openLinkDialog = useCallback(() => {
    if (!editor) return
    // Pre-populate with existing link if present
    const existingUrl = editor.getAttributes('link').href as string | undefined
    setLinkUrl(existingUrl ?? '')
    setShowLinkInput(true)
    // Focus input after state update
    setTimeout(() => linkInputRef.current?.focus(), 0)
  }, [editor])

  const setLink = useCallback(() => {
    if (!editor) return
    if (linkUrl.trim()) {
      // Add https:// if no protocol specified
      const url = linkUrl.match(/^https?:\/\//) ? linkUrl : `https://${linkUrl}`
      editor.chain().focus().extendMarkRange('link').setLink({ href: url }).run()
    }
    setShowLinkInput(false)
    setLinkUrl('')
  }, [editor, linkUrl])

  const removeLink = useCallback(() => {
    editor?.chain().focus().unsetLink().run()
  }, [editor])

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
    <div className={cn('flex flex-col bg-card rounded-lg border border-border', className)}>
      {/* Toolbar */}
      {!disabled && (
        <div className="flex-shrink-0 flex flex-wrap items-center gap-0.5 px-2 py-1.5 border-b border-border">
          <ToolbarButton
            onClick={() => editor.chain().focus().undo().run()}
            disabled={!editor.can().undo()}
            title="Undo"
          >
            <Undo className="h-4 w-4" />
          </ToolbarButton>
          <ToolbarButton
            onClick={() => editor.chain().focus().redo().run()}
            disabled={!editor.can().redo()}
            title="Redo"
          >
            <Redo className="h-4 w-4" />
          </ToolbarButton>

          <ToolbarDivider />

          <ToolbarButton
            onClick={() => setHeading(1)}
            isActive={editor.isActive('heading', { level: 1 })}
            title="Heading 1"
          >
            <Heading1 className="h-4 w-4" />
          </ToolbarButton>
          <ToolbarButton
            onClick={() => setHeading(2)}
            isActive={editor.isActive('heading', { level: 2 })}
            title="Heading 2"
          >
            <Heading2 className="h-4 w-4" />
          </ToolbarButton>
          <ToolbarButton
            onClick={() => setHeading(3)}
            isActive={editor.isActive('heading', { level: 3 })}
            title="Heading 3"
          >
            <Heading3 className="h-4 w-4" />
          </ToolbarButton>

          <ToolbarDivider />

          <ToolbarButton
            onClick={() => editor.chain().focus().toggleBold().run()}
            isActive={editor.isActive('bold')}
            title="Bold"
          >
            <Bold className="h-4 w-4" />
          </ToolbarButton>
          <ToolbarButton
            onClick={() => editor.chain().focus().toggleItalic().run()}
            isActive={editor.isActive('italic')}
            title="Italic"
          >
            <Italic className="h-4 w-4" />
          </ToolbarButton>
          <ToolbarButton
            onClick={() => editor.chain().focus().toggleStrike().run()}
            isActive={editor.isActive('strike')}
            title="Strikethrough"
          >
            <Strikethrough className="h-4 w-4" />
          </ToolbarButton>
          <ToolbarButton
            onClick={() => editor.chain().focus().toggleHighlight().run()}
            isActive={editor.isActive('highlight')}
            title="Highlight"
          >
            <Highlighter className="h-4 w-4" />
          </ToolbarButton>

          <ToolbarDivider />

          <ToolbarButton
            onClick={() => editor.chain().focus().toggleCode().run()}
            isActive={editor.isActive('code')}
            title="Inline Code"
          >
            <Code className="h-4 w-4" />
          </ToolbarButton>
          <ToolbarButton
            onClick={() => editor.chain().focus().toggleCodeBlock().run()}
            isActive={editor.isActive('codeBlock')}
            title="Code Block"
          >
            <FileCode className="h-4 w-4" />
          </ToolbarButton>

          <ToolbarDivider />

          <ToolbarButton
            onClick={openLinkDialog}
            isActive={editor.isActive('link')}
            title="Add Link"
          >
            <LinkIcon className="h-4 w-4" />
          </ToolbarButton>
          <ToolbarButton
            onClick={removeLink}
            disabled={!editor.isActive('link')}
            title="Remove Link"
          >
            <Unlink className="h-4 w-4" />
          </ToolbarButton>

          <ToolbarDivider />

          <ToolbarButton
            onClick={() => editor.chain().focus().toggleBulletList().run()}
            isActive={editor.isActive('bulletList')}
            title="Bullet List"
          >
            <List className="h-4 w-4" />
          </ToolbarButton>
          <ToolbarButton
            onClick={() => editor.chain().focus().toggleOrderedList().run()}
            isActive={editor.isActive('orderedList')}
            title="Numbered List"
          >
            <ListOrdered className="h-4 w-4" />
          </ToolbarButton>
          <ToolbarButton
            onClick={() => editor.chain().focus().toggleBlockquote().run()}
            isActive={editor.isActive('blockquote')}
            title="Blockquote"
          >
            <Quote className="h-4 w-4" />
          </ToolbarButton>
          <ToolbarButton
            onClick={() => editor.chain().focus().setHorizontalRule().run()}
            title="Horizontal Rule"
          >
            <Minus className="h-4 w-4" />
          </ToolbarButton>
        </div>
      )}

      {/* Link input dialog */}
      {showLinkInput && (
        <div className="flex-shrink-0 flex items-center gap-2 px-2 py-2 border-b border-border bg-muted/50">
          <LinkIcon className="h-4 w-4 text-muted-foreground flex-shrink-0" />
          <input
            ref={linkInputRef}
            type="url"
            value={linkUrl}
            onChange={(e) => setLinkUrl(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault()
                setLink()
              } else if (e.key === 'Escape') {
                setShowLinkInput(false)
                setLinkUrl('')
              }
            }}
            placeholder="Enter URL (e.g., https://example.com)"
            className={cn(
              'flex-1 px-2 py-1 text-sm',
              'bg-muted border border-border rounded',
              'text-foreground placeholder:text-muted-foreground',
              'focus:outline-none focus:ring-2 focus:ring-primary'
            )}
          />
          <button
            type="button"
            onClick={setLink}
            className="px-3 py-1 text-sm bg-primary hover:bg-primary/90 text-primary-foreground rounded transition-colors"
          >
            Add
          </button>
          <button
            type="button"
            onClick={() => {
              setShowLinkInput(false)
              setLinkUrl('')
            }}
            className="px-3 py-1 text-sm bg-muted hover:bg-muted/80 text-foreground rounded transition-colors"
          >
            Cancel
          </button>
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
