/**
 * SkillContentEditor - Dual-mode editor for skill content.
 *
 * Features:
 * - Toggle between Monaco (code) and TipTap (WYSIWYG) editors
 * - Syntax highlighting for markdown in Monaco
 * - Rich text formatting in TipTap
 * - Editor preference persisted to localStorage
 */

import { useCallback, useRef, useState, useEffect } from 'react'
import Editor, { type OnMount, type OnChange } from '@monaco-editor/react'
import { Code, Type } from 'lucide-react'
import { cn } from '@/lib/utils'
import { TipTapEditor } from './TipTapEditor'

type EditorType = 'code' | 'wysiwyg'

const STORAGE_KEY = 'pm.editorType'

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
      {/* Header with label and editor toggle */}
      <div className="flex items-center justify-between mb-1">
        <label className="block text-sm font-medium text-muted-foreground">
          Content <span className="text-red-400">*</span>
        </label>
        <div className="flex items-center gap-1 bg-muted rounded-lg p-0.5">
          <button
            type="button"
            onClick={() => setEditorType('code')}
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
            onClick={() => setEditorType('wysiwyg')}
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
      </div>

      {/* Editor container */}
      <div
        className={cn(
          'flex-1 rounded-lg overflow-hidden border',
          error ? 'border-red-500' : 'border-border'
        )}
      >
        {editorType === 'code' ? (
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
        ) : (
          <TipTapEditor
            value={value}
            onChange={handleTipTapChange}
            placeholder="Start writing your skill content..."
            className="h-full"
          />
        )}
      </div>
      {error && <p className="mt-1 text-xs text-red-400">{error}</p>}
    </div>
  )
}
