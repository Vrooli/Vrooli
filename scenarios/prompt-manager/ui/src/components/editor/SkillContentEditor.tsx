/**
 * SkillContentEditor - Dual-mode editor for skill content.
 *
 * Features:
 * - Toggle between Monaco (code) and TipTap (WYSIWYG) editors
 * - Syntax highlighting for markdown in Monaco
 * - Rich text formatting in TipTap
 * - Editor preference persisted to localStorage
 * - Toggle is embedded in each editor's header for maximum space efficiency
 */

import { useCallback, useRef, useState, useEffect } from 'react'
import Editor, { type OnMount, type OnChange } from '@monaco-editor/react'
import { Code, Type } from 'lucide-react'
import { cn } from '@/lib/utils'
import { TipTapEditor } from './TipTapEditor'

export type EditorType = 'code' | 'wysiwyg'

const STORAGE_KEY = 'pm.editorType'

interface EditorToggleProps {
  editorType: EditorType
  onEditorTypeChange: (type: EditorType) => void
  className?: string
}

/**
 * Shared toggle component for switching between Code and Rich Text modes.
 */
export function EditorToggle({ editorType, onEditorTypeChange, className }: EditorToggleProps) {
  return (
    <div className={cn('flex items-center gap-1 bg-muted rounded-lg p-0.5', className)}>
      <button
        type="button"
        onClick={() => onEditorTypeChange('code')}
        className={cn(
          'flex items-center gap-1.5 px-2 py-1 text-xs rounded-md transition-colors',
          editorType === 'code'
            ? 'bg-primary text-primary-foreground'
            : 'text-muted-foreground hover:text-foreground'
        )}
        title="Code Editor (Monaco)"
      >
        <Code className="h-3.5 w-3.5" />
        Code
      </button>
      <button
        type="button"
        onClick={() => onEditorTypeChange('wysiwyg')}
        className={cn(
          'flex items-center gap-1.5 px-2 py-1 text-xs rounded-md transition-colors',
          editorType === 'wysiwyg'
            ? 'bg-primary text-primary-foreground'
            : 'text-muted-foreground hover:text-foreground'
        )}
        title="Rich Text Editor (WYSIWYG)"
      >
        <Type className="h-3.5 w-3.5" />
        Rich Text
      </button>
    </div>
  )
}

interface SkillContentEditorProps {
  value: string
  onChange: (value: string) => void
  error?: string
  className?: string
}

/**
 * Dual-mode content editor for skills.
 */
export function SkillContentEditor({
  value,
  onChange,
  error,
  className,
}: SkillContentEditorProps) {
  const editorRef = useRef<Parameters<OnMount>[0] | null>(null)

  // Initialize editor type from localStorage
  const [editorType, setEditorType] = useState<EditorType>(() => {
    if (typeof window === 'undefined') return 'code'
    const stored = localStorage.getItem(STORAGE_KEY)
    return stored === 'wysiwyg' ? 'wysiwyg' : 'code'
  })

  // Persist editor type preference
  useEffect(() => {
    if (typeof window !== 'undefined') {
      localStorage.setItem(STORAGE_KEY, editorType)
    }
  }, [editorType])

  // Handle editor mount
  const handleEditorMount: OnMount = useCallback((editor) => {
    editorRef.current = editor
    // Focus editor on mount
    editor.focus()
  }, [])

  // Handle content change from Monaco
  const handleMonacoChange: OnChange = useCallback(
    (newValue) => {
      if (newValue !== undefined) {
        onChange(newValue)
      }
    },
    [onChange]
  )

  // Handle content change from TipTap
  const handleTipTapChange = useCallback(
    (newValue: string) => {
      onChange(newValue)
    },
    [onChange]
  )

  return (
    <div className={cn('flex flex-col', className)}>
      {/* Editor container - takes full height */}
      <div
        className={cn(
          'flex-1 flex flex-col overflow-hidden',
          error && 'ring-1 ring-red-500'
        )}
      >
        {editorType === 'code' ? (
          <div className="flex flex-col h-full">
            {/* Monaco header bar with toggle */}
            <div className="flex-shrink-0 flex items-center justify-end px-3 py-1.5 bg-[#1e1e1e] border-b border-[#3c3c3c]">
              <EditorToggle
                editorType={editorType}
                onEditorTypeChange={setEditorType}
              />
            </div>
            <div className="flex-1">
              <Editor
                height="100%"
                defaultLanguage="markdown"
                value={value}
                onChange={handleMonacoChange}
                onMount={handleEditorMount}
                theme="vs-dark"
                options={{
                  minimap: { enabled: false },
                  wordWrap: 'on',
                  lineNumbers: 'on',
                  fontSize: 13,
                  fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
                  tabSize: 2,
                  scrollBeyondLastLine: false,
                  padding: { top: 12, bottom: 12 },
                  renderLineHighlight: 'line',
                  cursorBlinking: 'smooth',
                  smoothScrolling: true,
                  scrollbar: {
                    vertical: 'auto',
                    horizontal: 'auto',
                    verticalScrollbarSize: 8,
                    horizontalScrollbarSize: 8,
                  },
                  overviewRulerBorder: false,
                  hideCursorInOverviewRuler: true,
                  folding: true,
                  foldingStrategy: 'indentation',
                  automaticLayout: true,
                }}
              />
            </div>
          </div>
        ) : (
          <TipTapEditor
            value={value}
            onChange={handleTipTapChange}
            placeholder="Start writing your skill content..."
            editorType={editorType}
            onEditorTypeChange={setEditorType}
            className="h-full"
          />
        )}
      </div>
      {error && <p className="mt-1 text-xs text-red-400">{error}</p>}
    </div>
  )
}
