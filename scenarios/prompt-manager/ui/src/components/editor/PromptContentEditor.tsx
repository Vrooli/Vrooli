/**
 * PromptContentEditor - Monaco editor wrapper for prompt content.
 *
 * Features:
 * - Syntax highlighting for markdown
 * - Word wrap
 * - Line numbers
 * - Read-only mode support
 */

import { useCallback, useRef } from 'react'
import Editor, { type OnMount, type OnChange } from '@monaco-editor/react'
import { cn } from '@/lib/utils'

interface PromptContentEditorProps {
  value: string
  onChange: (value: string) => void
  disabled?: boolean
  error?: string
  className?: string
}

/**
 * Monaco-based content editor for prompts.
 */
export function PromptContentEditor({
  value,
  onChange,
  disabled = false,
  error,
  className,
}: PromptContentEditorProps) {
  const editorRef = useRef<Parameters<OnMount>[0] | null>(null)

  // Handle editor mount
  const handleEditorMount: OnMount = useCallback((editor) => {
    editorRef.current = editor
    // Focus editor on mount
    editor.focus()
  }, [])

  // Handle content change
  const handleChange: OnChange = useCallback(
    (newValue) => {
      if (!disabled && newValue !== undefined) {
        onChange(newValue)
      }
    },
    [onChange, disabled]
  )

  return (
    <div className={cn('flex flex-col', className)}>
      <label className="block text-sm font-medium text-slate-300 mb-1">
        Content <span className="text-red-400">*</span>
      </label>
      <div
        className={cn(
          'flex-1 rounded-lg overflow-hidden border',
          error ? 'border-red-500' : 'border-white/10',
          disabled && 'opacity-50'
        )}
      >
        <Editor
          height="100%"
          defaultLanguage="markdown"
          value={value}
          onChange={handleChange}
          onMount={handleEditorMount}
          theme="vs-dark"
          options={{
            readOnly: disabled,
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
      {error && <p className="mt-1 text-xs text-red-400">{error}</p>}
    </div>
  )
}
