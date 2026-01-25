/**
 * SkillContentEditor - Dual-mode editor for skill content.
 *
 * Features:
 * - Toggle between Monaco (code) and TipTap (WYSIWYG) editors
 * - Syntax highlighting for markdown in Monaco
 * - Rich text formatting in TipTap
 * - Editor preference persisted to localStorage
 * - Toggle is embedded in each editor's header for maximum space efficiency
 * - Validates markdown for patterns that won't survive HTML round-trip
 * - Shows warnings in code mode and when switching to rich mode
 */

import { useCallback, useRef, useState, useEffect, useMemo } from 'react'
import Editor, { useMonaco, type OnMount, type OnChange } from '@monaco-editor/react'
import { AlertTriangle, Code, Type, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import { TipTapEditor } from './TipTapEditor'
import {
  validateMarkdown,
  validateRoundTrip,
  type MarkdownIssue,
  type RoundTripResult,
} from '@/services/content/validation'

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
  const monaco = useMonaco()

  // Initialize editor type from localStorage
  const [editorType, setEditorType] = useState<EditorType>(() => {
    if (typeof window === 'undefined') return 'code'
    const stored = localStorage.getItem(STORAGE_KEY)
    return stored === 'wysiwyg' ? 'wysiwyg' : 'code'
  })

  // State for markdown validation - issues persist across mode switches
  const [validationIssues, setValidationIssues] = useState<MarkdownIssue[]>([])
  const [showValidationWarning, setShowValidationWarning] = useState(false)

  // State for round-trip validation (catch-all protection)
  const [roundTripResult, setRoundTripResult] = useState<RoundTripResult | null>(null)
  const [showRoundTripWarning, setShowRoundTripWarning] = useState(false)

  // Persist editor type preference
  useEffect(() => {
    if (typeof window !== 'undefined') {
      localStorage.setItem(STORAGE_KEY, editorType)
    }
  }, [editorType])

  // Validate markdown content whenever value changes (regardless of mode)
  // Issues persist so we can warn when switching modes
  const validationResult = useMemo(() => validateMarkdown(value), [value])

  // Update validation issues state when validation result changes
  useEffect(() => {
    setValidationIssues(validationResult.issues)
  }, [validationResult.issues])

  // Validate round-trip when content changes (debounced for performance)
  useEffect(() => {
    const timer = setTimeout(() => {
      setRoundTripResult(validateRoundTrip(value))
    }, 500)
    return () => clearTimeout(timer)
  }, [value])

  // Set Monaco markers when validation issues change
  useEffect(() => {
    if (!monaco || !editorRef.current) return

    const model = editorRef.current.getModel()
    if (!model) return

    const markers = validationIssues.map((issue) => ({
      severity:
        issue.severity === 'error'
          ? monaco.MarkerSeverity.Error
          : monaco.MarkerSeverity.Warning,
      message:
        issue.message + (issue.suggestion ? `\n\nSuggestion: ${issue.suggestion}` : ''),
      startLineNumber: issue.line,
      startColumn: issue.column,
      endLineNumber: issue.endLine || issue.line,
      endColumn: issue.endColumn || issue.column + 1,
    }))

    monaco.editor.setModelMarkers(model, 'markdown-validator', markers)

    // Cleanup on unmount or when issues change
    // Note: model check is needed at cleanup time since model state may have changed
    return () => {
      // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- model may be disposed at cleanup time
      if (model && !model.isDisposed()) {
        monaco.editor.setModelMarkers(model, 'markdown-validator', [])
      }
    }
  }, [monaco, validationIssues])

  // Handle mode switching with validation warning and round-trip blocking
  const handleEditorTypeChange = useCallback(
    (newType: EditorType) => {
      // When switching to Rich mode, check for issues
      if (newType === 'wysiwyg') {
        // Block switch if round-trip validation fails (content would be corrupted)
        if (roundTripResult && !roundTripResult.isStable) {
          setShowRoundTripWarning(true)
          return // Block the switch
        }

        // Show warning for known validation issues (non-blocking)
        if (validationIssues.length > 0) {
          setShowValidationWarning(true)
        }
      }
      setEditorType(newType)
    },
    [validationIssues.length, roundTripResult]
  )

  // Force switch to Rich mode despite round-trip warning
  const handleForceRichMode = useCallback(() => {
    setShowRoundTripWarning(false)
    setEditorType('wysiwyg')
  }, [])

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
      {/* Round-trip warning banner - BLOCKS switching to rich mode when content would be corrupted */}
      {showRoundTripWarning && roundTripResult && !roundTripResult.isStable && (
        <div className="flex items-center gap-2 bg-red-900/30 border-l-4 border-red-500 px-3 py-2 text-sm text-red-200">
          <AlertTriangle className="h-4 w-4 flex-shrink-0" />
          <span className="flex-1">
            <strong>Rich mode blocked:</strong> Content would be corrupted during conversion.
            {roundTripResult.changeDescription && ` (${roundTripResult.changeDescription})`}
          </span>
          <button
            type="button"
            onClick={handleForceRichMode}
            className="px-2 py-0.5 text-xs bg-red-800/50 hover:bg-red-700/50 rounded transition-colors"
          >
            Show anyway
          </button>
          <button
            type="button"
            onClick={() => setShowRoundTripWarning(false)}
            className="p-0.5 hover:bg-red-800/50 rounded"
            aria-label="Dismiss warning"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      )}
      {/* Validation warning banner - shown when switching to rich mode with issues */}
      {showValidationWarning && validationIssues.length > 0 && (
        <div className="flex items-center gap-2 bg-yellow-900/30 border-l-4 border-yellow-500 px-3 py-2 text-sm text-yellow-200">
          <AlertTriangle className="h-4 w-4 flex-shrink-0" />
          <span className="flex-1">
            {validationIssues.length} markdown issue
            {validationIssues.length !== 1 ? 's' : ''} detected. Content may not display
            correctly in Rich mode.
          </span>
          <button
            type="button"
            onClick={() => setShowValidationWarning(false)}
            className="p-0.5 hover:bg-yellow-800/50 rounded"
            aria-label="Dismiss warning"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      )}
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
                onEditorTypeChange={handleEditorTypeChange}
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
            onEditorTypeChange={handleEditorTypeChange}
            className="h-full"
          />
        )}
      </div>
      {error && <p className="mt-1 text-xs text-red-400">{error}</p>}
    </div>
  )
}
