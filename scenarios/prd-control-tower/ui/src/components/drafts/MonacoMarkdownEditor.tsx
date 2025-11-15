import { useRef, useEffect } from 'react'
import Editor, { Monaco } from '@monaco-editor/react'
import { type editor } from 'monaco-editor'
import { cn } from '../../lib/utils'

interface MonacoMarkdownEditorProps {
  value: string
  onChange: (value: string) => void
  className?: string
  height?: string | number
  readOnly?: boolean
  /**
   * Reference to the textarea (for AI assistant text selection tracking)
   * This maintains backward compatibility with useAIAssistant hook
   */
  textareaRef?: React.RefObject<HTMLTextAreaElement>
  /**
   * Callback to open AI assistant dialog (triggered by Cmd+K / Ctrl+K)
   */
  onOpenAI?: () => void
}

/**
 * Monaco-based markdown editor with syntax highlighting, IntelliSense, and rich editing features.
 *
 * Features:
 * - Markdown syntax highlighting
 * - Line numbers and minimap
 * - Find/replace (Cmd+F / Ctrl+F)
 * - Multi-cursor editing (Cmd+Click / Ctrl+Click)
 * - Bracket matching and auto-closing
 * - Customized for PRD editing (word wrap, soft tabs)
 */
export function MonacoMarkdownEditor({
  value,
  onChange,
  className,
  height = 480,
  readOnly = false,
  textareaRef,
  onOpenAI,
}: MonacoMarkdownEditorProps) {
  const editorRef = useRef<editor.IStandaloneCodeEditor | null>(null)
  const monacoRef = useRef<Monaco | null>(null)

  // Handle editor mount
  const handleEditorDidMount = (editor: editor.IStandaloneCodeEditor, monaco: Monaco) => {
    editorRef.current = editor
    monacoRef.current = monaco

    // Focus editor on mount
    editor.focus()

    // Add custom keyboard shortcuts
    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
      // Prevent default save behavior - parent component handles saving
      // This prevents browser save dialog from opening
    })

    // Add Cmd+K / Ctrl+K shortcut for AI assistant
    if (onOpenAI) {
      editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyK, () => {
        onOpenAI()
      })
    }
  }

  // Handle content changes
  const handleChange = (newValue: string | undefined) => {
    if (newValue !== undefined && newValue !== value) {
      onChange(newValue)
    }
  }

  // Re-register keyboard shortcuts when onOpenAI changes
  useEffect(() => {
    if (!editorRef.current || !monacoRef.current || !onOpenAI) return

    const editor = editorRef.current
    const monaco = monacoRef.current

    // addCommand returns a disposable ID (string), not a disposable object
    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyK, () => {
      onOpenAI()
    })

    // No cleanup needed for keyboard commands
  }, [onOpenAI])

  // Sync selection with textareaRef for AI assistant compatibility
  useEffect(() => {
    if (!textareaRef || !editorRef.current) return

    const editor = editorRef.current
    const model = editor.getModel()
    if (!model) return

    const updateTextareaSelection = () => {
      const selection = editor.getSelection()
      if (!selection || !textareaRef.current) return

      // Convert Monaco position to textarea offset
      const startOffset = model.getOffsetAt(selection.getStartPosition())
      const endOffset = model.getOffsetAt(selection.getEndPosition())

      // Update textarea selection for AI assistant tracking
      textareaRef.current.selectionStart = startOffset
      textareaRef.current.selectionEnd = endOffset
    }

    const disposable = editor.onDidChangeCursorSelection(updateTextareaSelection)
    return () => disposable.dispose()
  }, [textareaRef])

  // Programmatically set value when textareaRef.value changes (for AI insertion)
  useEffect(() => {
    if (!textareaRef?.current) return

    const textarea = textareaRef.current

    // Watch for value changes from AI assistant
    const observer = new MutationObserver(() => {
      if (textarea.value !== value) {
        onChange(textarea.value)
      }
    })

    observer.observe(textarea, { attributes: true, attributeFilter: ['value'] })
    return () => observer.disconnect()
  }, [textareaRef, value, onChange])

  return (
    <div className={cn('rounded-md border overflow-hidden bg-white', className)}>
      <Editor
        height={height}
        language="markdown"
        value={value}
        onChange={handleChange}
        onMount={handleEditorDidMount}
        theme="vs-light"
        options={{
          readOnly,
          fontSize: 13,
          lineNumbers: 'on',
          wordWrap: 'on',
          wrappingStrategy: 'advanced',
          minimap: { enabled: true, maxColumn: 80 },
          scrollBeyondLastLine: false,
          smoothScrolling: true,
          cursorBlinking: 'smooth',
          cursorSmoothCaretAnimation: 'on',
          renderLineHighlight: 'all',
          renderWhitespace: 'selection',
          bracketPairColorization: { enabled: true },
          guides: {
            bracketPairs: true,
            indentation: true,
          },
          suggest: {
            showWords: true,
            showSnippets: true,
          },
          quickSuggestions: {
            other: true,
            comments: false,
            strings: false,
          },
          tabSize: 2,
          insertSpaces: true,
          detectIndentation: true,
          trimAutoWhitespace: true,
          formatOnPaste: true,
          formatOnType: true,
          autoClosingBrackets: 'always',
          autoClosingQuotes: 'always',
          autoIndent: 'full',
          folding: true,
          foldingStrategy: 'indentation',
          showFoldingControls: 'always',
          matchBrackets: 'always',
          occurrencesHighlight: 'singleFile',
          selectionHighlight: true,
          hover: {
            enabled: true,
            delay: 300,
          },
          links: true,
          colorDecorators: true,
          contextmenu: true,
          mouseWheelZoom: true,
          multiCursorModifier: 'ctrlCmd',
          snippetSuggestions: 'inline',
          wordBasedSuggestions: 'matchingDocuments',
          padding: { top: 12, bottom: 12 },
          scrollbar: {
            vertical: 'auto',
            horizontal: 'auto',
            verticalScrollbarSize: 12,
            horizontalScrollbarSize: 12,
          },
        }}
      />
      {/* Hidden textarea for AI assistant compatibility */}
      {textareaRef && (
        <textarea
          ref={textareaRef}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          style={{ position: 'absolute', left: '-9999px', opacity: 0, pointerEvents: 'none' }}
          aria-hidden="true"
          tabIndex={-1}
        />
      )}
    </div>
  )
}
