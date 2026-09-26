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
 * - Preview toggle renders split view on desktop, preview-only on mobile
 */

import { useCallback, useRef, useState, useEffect, useMemo, type ReactNode } from 'react'
import Editor, { DiffEditor, useMonaco, type OnMount, type OnChange } from '@monaco-editor/react'
import { AlertTriangle, Code, Diff, Eye, Files, PanelLeftOpen, Redo2, RotateCcw, Save, Type, Undo2, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import { MarkdownRenderer } from '@/components/markdown'
import { useResizableSplitPanel } from '@/hooks/useResizableSplitPanel'
import { useIsMobile } from '@/hooks/useMediaQuery'
import { useResolvedTheme } from '@/hooks/use-theme'
import { selectors } from '@/constants/selectors'
import { useEditorPreferencesStore } from '@/stores/editorPreferencesStore'
import { TipTapEditor } from './TipTapEditor'
import {
  validateMarkdown,
  validateRoundTrip,
  type MarkdownIssue,
  type RoundTripResult,
} from '@/services/content/validation'
import type { ContentSearchMatch } from '@/lib/schemas'

export type EditorType = 'code' | 'wysiwyg'
export type ViewMode = 'edit' | 'preview'

const STORAGE_KEY = 'pm.editorType'
const VIEW_MODE_STORAGE_KEY = 'pm.editorViewMode'

interface EditorToggleProps {
  editorType: EditorType
  onEditorTypeChange: (type: EditorType) => void
  variant?: 'light' | 'dark'
  className?: string
}

/**
 * Shared toggle component for switching between Code and Rich Text modes.
 */
export function EditorToggle({
  editorType,
  onEditorTypeChange,
  variant = 'light',
  className,
}: EditorToggleProps) {
  const isCode = editorType === 'code'
  const nextType: EditorType = isCode ? 'wysiwyg' : 'code'
  const baseClasses = cn(
    'flex items-center justify-center h-8 w-8 rounded-md transition-colors',
    variant === 'dark'
      ? 'text-muted-foreground hover:text-foreground hover:bg-muted'
      : 'text-muted-foreground hover:text-foreground hover:bg-muted'
  )

  return (
    <button
      type="button"
      onClick={() => onEditorTypeChange(nextType)}
      className={cn(baseClasses, className)}
      title={isCode ? 'Switch to Rich Text' : 'Switch to Code'}
      aria-label={isCode ? 'Switch to Rich Text' : 'Switch to Code'}
    >
      {isCode ? <Code className="h-4 w-4" /> : <Type className="h-4 w-4" />}
    </button>
  )
}

interface ViewToggleProps {
  viewMode: ViewMode
  onViewModeChange: (mode: ViewMode) => void
  variant?: 'light' | 'dark'
  className?: string
}

/**
 * Shared toggle component for switching between Editor and Preview views.
 */
export function ViewToggle({
  viewMode,
  onViewModeChange,
  variant = 'light',
  className,
}: ViewToggleProps) {
  const isPreview = viewMode === 'preview'
  const baseClasses = cn(
    'flex items-center justify-center h-8 w-8 rounded-md transition-colors',
    variant === 'dark'
      ? 'text-muted-foreground hover:text-foreground hover:bg-muted'
      : 'text-muted-foreground hover:text-foreground hover:bg-muted',
    isPreview &&
      (variant === 'dark' ? 'bg-muted text-foreground' : 'bg-primary/20 text-primary')
  )

  return (
    <button
      type="button"
      onClick={() => onViewModeChange(isPreview ? 'edit' : 'preview')}
      className={cn(baseClasses, className)}
      title={isPreview ? 'Show editor' : 'Show preview'}
      aria-pressed={isPreview}
      aria-label={isPreview ? 'Show editor' : 'Show preview'}
    >
      <Eye className="h-4 w-4" />
    </button>
  )
}

export interface EditorActionState {
  isDirty?: boolean
  dirtyCount?: number
  onUndo?: () => void
  onRedo?: () => void
  canUndo?: boolean
  canRedo?: boolean
  onSave?: () => void
  onSaveAll?: () => void
  onDiscard?: () => void
  onToggleDiff?: () => void
  isDiffMode?: boolean
  isSaving?: boolean
  isValid?: boolean
}

interface EditorActionButtonsProps extends EditorActionState {
  variant?: 'light' | 'dark'
  className?: string
}

export function EditorActionButtons({
  isDirty = false,
  dirtyCount = 0,
  onUndo,
  onRedo,
  canUndo = false,
  canRedo = false,
  onSave,
  onSaveAll,
  onDiscard,
  onToggleDiff,
  isDiffMode = false,
  isSaving = false,
  isValid = true,
  variant = 'light',
  className,
}: EditorActionButtonsProps) {
  const hasActions = Boolean(onUndo || onRedo || onSave || onSaveAll || onDiscard || onToggleDiff)
  if (!hasActions) return null

  const canSaveBtn = Boolean(onSave) && isDirty && !isSaving && isValid
  const canSaveAll = Boolean(onSaveAll) && dirtyCount > 1 && !isSaving
  const canShowDiff = Boolean(onToggleDiff) && isDirty
  const canToggleDiff = canShowDiff && !isSaving
  const canDiscard = Boolean(onDiscard) && isDirty && !isSaving

  const baseClass =
    variant === 'dark'
      ? 'text-foreground hover:text-foreground hover:bg-muted'
      : 'text-muted-foreground hover:text-foreground hover:bg-muted'
  const disabledClass =
    variant === 'dark'
      ? 'text-muted-foreground/50 cursor-not-allowed'
      : 'text-muted-foreground/50 cursor-not-allowed'
  const actionButtonClass = (enabled: boolean) =>
    cn('h-8 w-8 flex items-center justify-center rounded-md transition-colors',
      enabled ? baseClass : disabledClass
    )

  return (
    <div className={cn('flex items-center gap-2', className)}>
      {onUndo && (
        <button
          type="button"
          onClick={onUndo}
          disabled={!canUndo || isSaving}
          className={actionButtonClass(canUndo && !isSaving)}
          title="Undo (Ctrl+Z)"
          aria-label="Undo"
        >
          <Undo2 className="h-4 w-4" />
        </button>
      )}
      {onRedo && (
        <button
          type="button"
          onClick={onRedo}
          disabled={!canRedo || isSaving}
          className={actionButtonClass(canRedo && !isSaving)}
          title="Redo (Ctrl+Shift+Z)"
          aria-label="Redo"
        >
          <Redo2 className="h-4 w-4" />
        </button>
      )}
      {(onUndo || onRedo) && (
        <div
          className={cn('w-px h-6',
            variant === 'dark' ? 'bg-border' : 'bg-border'
          )}
        />
      )}
      {onSave && (
        <button
          type="button"
          onClick={onSave}
          disabled={!canSaveBtn}
          className={cn(
            'relative h-8 w-8 flex items-center justify-center rounded-md transition-colors',
            canSaveBtn
              ? 'bg-primary text-primary-foreground hover:bg-primary/90'
              : variant === 'dark'
                ? 'bg-muted/50 text-muted-foreground/50 cursor-not-allowed'
                : 'bg-muted text-muted-foreground cursor-not-allowed'
          )}
          title={isDirty ? 'Save changes (Ctrl+S)' : 'No changes to save'}
          aria-label={isDirty ? 'Save (unsaved changes)' : 'Save'}
          data-testid={selectors.editor.saveButton}
        >
          <Save className="h-4 w-4" />
          {isDirty && (
            <span
              aria-hidden="true"
              data-testid={selectors.editor.unsavedIndicator}
              className="absolute -top-0.5 -right-0.5 h-2 w-2 rounded-full bg-amber-400 ring-2 ring-background"
            />
          )}
        </button>
      )}
      {canShowDiff && (
        <button
          type="button"
          onClick={onToggleDiff}
          disabled={!canToggleDiff}
          className={cn(
            'h-8 w-8 flex items-center justify-center rounded-md transition-colors',
            canToggleDiff
              ? isDiffMode
                ? variant === 'dark'
                  ? 'bg-muted text-foreground'
                  : 'bg-primary/20 text-primary'
                : baseClass
              : disabledClass
          )}
          title={isDiffMode ? 'Hide diff' : 'Show diff'}
          aria-label="Toggle diff"
          aria-pressed={isDiffMode}
        >
          <Diff className="h-4 w-4" />
        </button>
      )}
      {isDiffMode && onDiscard && (
        <button
          type="button"
          onClick={onDiscard}
          disabled={!canDiscard}
          className={cn(
            'h-8 w-8 flex items-center justify-center rounded-md transition-colors',
            canDiscard
              ? variant === 'dark'
                ? 'text-rose-300 hover:text-white hover:bg-rose-500/20'
                : 'text-rose-600 hover:text-rose-700 hover:bg-rose-50'
              : disabledClass
          )}
          title="Discard changes"
          aria-label="Discard changes"
          data-testid={selectors.editor.discardButton}
        >
          <RotateCcw className="h-4 w-4" />
        </button>
      )}
      {onSaveAll && dirtyCount > 1 && (
        <button
          type="button"
          onClick={onSaveAll}
          disabled={!canSaveAll}
          className={cn(
            'h-8 w-8 flex items-center justify-center rounded-md transition-colors',
            canSaveAll
              ? 'bg-emerald-600 hover:bg-emerald-500 text-white'
              : variant === 'dark'
                ? 'bg-muted/50 text-muted-foreground/50 cursor-not-allowed'
                : 'bg-muted text-muted-foreground cursor-not-allowed'
          )}
          title={`Save all ${dirtyCount} pending changes (Ctrl+Shift+S)`}
          aria-label="Save all"
          data-testid={selectors.editor.saveAllButton}
        >
          <Files className="h-4 w-4" />
        </button>
      )}
    </div>
  )
}

interface SkillContentEditorProps extends EditorActionState {
  value: string
  originalValue?: string | null
  onChange: (value: string) => void
  error?: string
  /** Search matches to highlight in the editor */
  searchMatches?: ContentSearchMatch[]
  /** Line number to scroll to in the editor */
  scrollToLine?: number | null
  /** Called after the editor has scrolled to the requested line */
  onScrollToLineHandled?: () => void
  className?: string
  headerLeft?: ReactNode
  headerRight?: ReactNode
}

/**
 * Dual-mode content editor for skills.
 */
export function SkillContentEditor({
  value,
  originalValue,
  onChange,
  error,
  searchMatches = [],
  scrollToLine,
  onScrollToLineHandled,
  isDirty = false,
  dirtyCount = 0,
  onUndo,
  onRedo,
  canUndo = false,
  canRedo = false,
  onSave,
  onSaveAll,
  onDiscard,
  isSaving = false,
  isValid = true,
  className,
  headerLeft,
  headerRight,
}: SkillContentEditorProps) {
  const editorRef = useRef<Parameters<OnMount>[0] | null>(null)
  const searchDecorationsRef = useRef<ReturnType<Parameters<OnMount>[0]['createDecorationsCollection']> | null>(null)
  const [editorReady, setEditorReady] = useState(false)
  const monaco = useMonaco()
  const isMobile = useIsMobile()
  const resolvedTheme = useResolvedTheme()
  const showCodeLineNumbers = useEditorPreferencesStore((state) => state.showCodeLineNumbers)
  const monacoLineNumbers: 'on' | 'off' = showCodeLineNumbers ? 'on' : 'off'
  const {
    width: splitWidth,
    isResizing: isSplitResizing,
    isCollapsed: isSplitCollapsed,
    containerRef: splitContainerRef,
    handleResizeStart: handleSplitResizeStart,
    expand: expandSplit,
  } = useResizableSplitPanel()

  // Initialize editor type from localStorage
  const [editorType, setEditorType] = useState<EditorType>(() => {
    if (typeof window === 'undefined') return 'code'
    const stored = localStorage.getItem(STORAGE_KEY)
    return stored === 'wysiwyg' ? 'wysiwyg' : 'code'
  })
  const [viewMode, setViewMode] = useState<ViewMode>(() => {
    if (typeof window === 'undefined') return 'edit'
    const stored = localStorage.getItem(VIEW_MODE_STORAGE_KEY)
    return stored === 'preview' ? 'preview' : 'edit'
  })
  const [isDiffMode, setIsDiffMode] = useState(false)

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

  // Persist view mode preference
  useEffect(() => {
    if (typeof window !== 'undefined') {
      localStorage.setItem(VIEW_MODE_STORAGE_KEY, viewMode)
    }
  }, [viewMode])

  useEffect(() => {
    if (!isDirty) {
      setIsDiffMode(false)
    }
  }, [isDirty])

  useEffect(() => {
    setIsDiffMode(false)
  }, [originalValue])

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

  // Apply search match decorations to Monaco editor
  // Also depends on `value` to re-apply after content changes (e.g., switching skills)
  useEffect(() => {
    if (!monaco || !editorRef.current || !editorReady) return
    // Only apply when in code mode (not wysiwyg)
    if (editorType !== 'code') return

    const editor = editorRef.current

    // Use requestAnimationFrame to ensure Monaco has processed any pending updates
    const frameId = requestAnimationFrame(() => {
      const model = editor.getModel()
      if (!model || model.isDisposed()) return

      // Build decorations for search matches
      const decorations: Parameters<typeof editor.createDecorationsCollection>[0] = []

      for (const match of searchMatches) {
        // Each matchRange is a character position within the line
        for (const range of match.matchRanges) {
          decorations.push({
            range: new monaco.Range(
              match.lineNumber,
              range.start + 1, // Monaco uses 1-based columns
              match.lineNumber,
              range.end + 1
            ),
            options: {
              inlineClassName: 'search-match-highlight',
              overviewRuler: {
                color: '#3b82f6',
                position: monaco.editor.OverviewRulerLane.Center,
              },
            },
          })
        }
      }

      // Apply decorations using createDecorationsCollection
      if (searchDecorationsRef.current) {
        searchDecorationsRef.current.set(decorations)
      } else {
        searchDecorationsRef.current = editor.createDecorationsCollection(decorations)
      }
    })

    // Cleanup decorations when component unmounts or matches change
    return () => {
      cancelAnimationFrame(frameId)
      const model = editor.getModel()
      // Clear decorations if editor still exists
      if (model && !model.isDisposed()) {
        searchDecorationsRef.current?.clear()
        searchDecorationsRef.current = null
      }
    }
  }, [monaco, searchMatches, editorType, editorReady, value])

  // Scroll to a specific line when requested (e.g. clicking a content search result)
  // Depends on `value` so it re-fires when content loads (e.g. async fetch for a new skill).
  // Before scrolling, checks that the document has enough lines - if not, the effect
  // skips and will retry when `value` updates with the loaded content.
  useEffect(() => {
    if (scrollToLine == null || !editorRef.current || !editorReady) return
    if (editorType !== 'code') return

    const editor = editorRef.current
    // Use requestAnimationFrame to ensure the editor has rendered after skill switch
    const frameId = requestAnimationFrame(() => {
      const model = editor.getModel()
      if (!model || model.isDisposed()) return

      // Skip if the document doesn't have enough lines yet (content still loading)
      if (model.getLineCount() < scrollToLine) return

      editor.revealLineInCenter(scrollToLine)
      editor.setPosition({ lineNumber: scrollToLine, column: 1 })
      onScrollToLineHandled?.()
    })

    return () => cancelAnimationFrame(frameId)
  }, [scrollToLine, editorReady, editorType, onScrollToLineHandled, value])

  // Handle mode switching with validation warning and round-trip blocking
  const handleEditorTypeChange = useCallback(
    (newType: EditorType) => {
      // When switching to Rich mode, check for issues
      if (newType === 'wysiwyg') {
        // Block switch if round-trip validation fails (content would be corrupted)
        if (roundTripResult && !roundTripResult.isStable) {
          setShowRoundTripWarning(true)
          setViewMode('preview')
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
    setViewMode('edit')
  }, [])

  // Handle editor mount
  const handleEditorMount: OnMount = useCallback((editor) => {
    editorRef.current = editor
    setEditorReady(true)
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

  const handleViewModeChange = useCallback((mode: ViewMode) => {
    setViewMode(mode)
    // Auto-expand editor when leaving preview mode
    if (mode === 'edit' && isSplitCollapsed) {
      expandSplit()
    }
  }, [isSplitCollapsed, expandSplit])

  const handleToggleDiff = useCallback(() => {
    setIsDiffMode((prev) => !prev)
  }, [])

  const previewPane = (
    <div className="flex-1 overflow-y-auto bg-card" data-testid="markdown-preview">
      <div className="p-4">
        <MarkdownRenderer content={value} />
      </div>
    </div>
  )

  const headerVariant: 'light' | 'dark' = resolvedTheme === 'dark' ? 'dark' : 'light'
  const editorActionState: EditorActionState = {
    isDirty,
    dirtyCount,
    onUndo,
    onRedo,
    canUndo,
    canRedo,
    onSave,
    onSaveAll,
    onDiscard,
    onToggleDiff: originalValue !== null && originalValue !== undefined ? handleToggleDiff : undefined,
    isDiffMode,
    isSaving,
    isValid,
  }

  const renderHeader = (variant: 'light' | 'dark') => (
    <div
      className={cn(
        'flex-shrink-0 flex items-center gap-2 px-3 py-1.5 border-b',
        'bg-card/95 border-border'
      )}
    >
      <EditorActionButtons {...editorActionState} variant={variant} />
      {headerLeft && (
        <div
          className={cn(
            'flex items-center gap-2',
            variant === 'dark' ? 'text-foreground' : 'text-muted-foreground'
          )}
        >
          {headerLeft}
        </div>
      )}
      <div className="ml-auto flex items-center gap-2">
        {headerRight && (
          <div
            className={cn(
              'flex items-center gap-2',
              variant === 'dark' ? 'text-foreground' : 'text-muted-foreground'
            )}
          >
            {headerRight}
          </div>
        )}
        {isSplitCollapsed && viewMode === 'preview' && !isMobile && (
          <button
            type="button"
            onClick={expandSplit}
            className={cn(
              'flex items-center justify-center h-8 w-8 rounded-md transition-colors',
              variant === 'dark'
                ? 'text-muted-foreground hover:text-foreground hover:bg-muted'
                : 'text-muted-foreground hover:text-foreground hover:bg-muted'
            )}
            title="Expand editor"
            aria-label="Expand editor"
          >
            <PanelLeftOpen className="h-4 w-4" />
          </button>
        )}
        <EditorToggle
          editorType={editorType}
          onEditorTypeChange={handleEditorTypeChange}
          variant={variant}
        />
        {!isDiffMode && (
          <ViewToggle
            viewMode={viewMode}
            onViewModeChange={handleViewModeChange}
            variant={variant}
          />
        )}
      </div>
    </div>
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
        {isDiffMode && originalValue !== null && originalValue !== undefined ? (
          <div className="flex flex-col h-full">
            {renderHeader(headerVariant)}
            <div className="flex-1">
              <DiffEditor
                height="100%"
                language="markdown"
                original={originalValue}
                modified={value}
                theme={resolvedTheme === 'dark' ? 'vs-dark' : 'vs-light'}
                options={{
                  readOnly: true,
                  originalEditable: false,
                  renderSideBySide: !isMobile,
                  minimap: { enabled: false },
                  wordWrap: 'on',
                  lineNumbers: monacoLineNumbers,
                  fontSize: 13,
                  fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
                  scrollBeyondLastLine: false,
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
        ) : viewMode === 'preview' && isMobile ? (
          <div className="flex flex-col h-full">
            {renderHeader(headerVariant)}
            {previewPane}
          </div>
        ) : viewMode === 'preview' && isSplitCollapsed ? (
          <div ref={splitContainerRef} className="flex flex-col h-full">
            {renderHeader(headerVariant)}
            {previewPane}
          </div>
        ) : viewMode === 'preview' ? (
          <div
            ref={splitContainerRef}
            className={cn('flex h-full', isSplitResizing && 'select-none')}
          >
            <div className="flex-shrink-0 flex flex-col h-full" style={{ width: splitWidth }}>
              {editorType === 'code' ? (
                <div className="flex flex-col h-full">
                  {renderHeader(headerVariant)}
                  <div className="flex-1">
                    <Editor
                      height="100%"
                      defaultLanguage="markdown"
                      value={value}
                      onChange={handleMonacoChange}
                      onMount={handleEditorMount}
                      theme={resolvedTheme === 'dark' ? 'vs-dark' : 'vs-light'}
                      loading={<div className="flex-1 flex items-center justify-center h-full bg-card"><div className="w-6 h-6 border-2 border-slate-600 border-t-slate-300 rounded-full animate-spin" /></div>}
                      options={{
                        minimap: { enabled: false },
                        wordWrap: 'on',
                        lineNumbers: monacoLineNumbers,
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
                  viewMode={viewMode}
                  onViewModeChange={handleViewModeChange}
                  {...editorActionState}
                  className="h-full"
                />
              )}
            </div>
            <div
              role="separator"
              aria-orientation="vertical"
              aria-label="Resize editor preview split"
              tabIndex={0}
              onMouseDown={handleSplitResizeStart}
              className="relative flex-shrink-0 w-3 cursor-col-resize group"
            >
              <div className="absolute right-1 top-0 h-full w-0.5 bg-border group-hover:bg-primary/50 transition-colors" />
            </div>
            <div className="flex-1 min-w-0 flex flex-col">{previewPane}</div>
          </div>
        ) : editorType === 'code' ? (
          <div className="flex flex-col h-full">
            {renderHeader(headerVariant)}
            <div className="flex-1">
              <Editor
                height="100%"
                defaultLanguage="markdown"
                value={value}
                onChange={handleMonacoChange}
                onMount={handleEditorMount}
                theme={resolvedTheme === 'dark' ? 'vs-dark' : 'vs-light'}
                loading={<div className="flex-1 flex items-center justify-center h-full bg-card"><div className="w-6 h-6 border-2 border-slate-600 border-t-slate-300 rounded-full animate-spin" /></div>}
                options={{
                  minimap: { enabled: false },
                  wordWrap: 'on',
                  lineNumbers: monacoLineNumbers,
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
            viewMode={isDiffMode ? undefined : viewMode}
            onViewModeChange={isDiffMode ? undefined : handleViewModeChange}
            {...editorActionState}
            className="h-full"
          />
        )}
      </div>
      {error && <p className="mt-1 text-xs text-red-400">{error}</p>}
    </div>
  )
}
