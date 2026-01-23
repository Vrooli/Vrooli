/**
 * EditorToolbar - Action buttons for the prompt editor.
 *
 * Includes:
 * - Undo/Redo buttons
 * - Save button
 * - Save All button (when multiple dirty)
 * - Discard button
 * - Delete button
 */

import { Save, Trash2, RotateCcw, Zap, Undo2, Redo2 } from 'lucide-react'
import { cn } from '@/lib/utils'

interface EditorToolbarProps {
  // Dirty state
  isDirty: boolean
  dirtyCount: number

  // Undo/Redo
  onUndo?: () => void
  onRedo?: () => void
  canUndo?: boolean
  canRedo?: boolean

  // Actions
  onSave: () => void
  onSaveAll: () => void
  onDiscard: () => void
  onDelete: () => void

  // Loading/disabled states
  isSaving: boolean
  isDeleting: boolean
  isValid: boolean

  // Optional test button
  onTest?: () => void
  className?: string
}

/**
 * Toolbar component with action buttons.
 */
export function EditorToolbar({
  isDirty,
  dirtyCount,
  onUndo,
  onRedo,
  canUndo = false,
  canRedo = false,
  onSave,
  onSaveAll,
  onDiscard,
  onDelete,
  isSaving,
  isDeleting,
  isValid,
  onTest,
  className,
}: EditorToolbarProps) {
  const canSaveBtn = isDirty && !isSaving && isValid
  const canSaveAll = dirtyCount > 0 && !isSaving
  const canDiscardBtn = isDirty && !isSaving
  const canDelete = !isDeleting

  return (
    <div className={cn('flex items-center gap-2', className)}>
      {/* Undo */}
      {onUndo && (
        <button
          type="button"
          onClick={onUndo}
          disabled={!canUndo || isSaving}
          className={cn(
            'flex items-center gap-1 px-2 py-1.5 text-sm font-medium rounded-lg transition-colors',
            canUndo && !isSaving
              ? 'bg-muted hover:bg-muted/80 text-foreground'
              : 'bg-muted/50 text-muted-foreground cursor-not-allowed'
          )}
          title="Undo (Ctrl+Z)"
        >
          <Undo2 className="h-4 w-4" />
        </button>
      )}

      {/* Redo */}
      {onRedo && (
        <button
          type="button"
          onClick={onRedo}
          disabled={!canRedo || isSaving}
          className={cn(
            'flex items-center gap-1 px-2 py-1.5 text-sm font-medium rounded-lg transition-colors',
            canRedo && !isSaving
              ? 'bg-muted hover:bg-muted/80 text-foreground'
              : 'bg-muted/50 text-muted-foreground cursor-not-allowed'
          )}
          title="Redo (Ctrl+Shift+Z)"
        >
          <Redo2 className="h-4 w-4" />
        </button>
      )}

      {/* Separator when undo/redo shown */}
      {(onUndo || onRedo) && <div className="w-px h-6 bg-border mx-1" />}

      {/* Save current */}
      <button
        type="button"
        onClick={onSave}
        disabled={!canSaveBtn}
        className={cn(
          'flex items-center gap-2 px-3 py-1.5 text-sm font-medium rounded-lg transition-colors',
          canSaveBtn
            ? 'bg-primary hover:bg-primary/90 text-primary-foreground'
            : 'bg-muted text-muted-foreground cursor-not-allowed'
        )}
        title={isDirty ? 'Save changes (Ctrl+S)' : 'No changes to save'}
      >
        <Save className="h-4 w-4" />
        {isSaving ? 'Saving...' : 'Save'}
      </button>

      {/* Save all (only show if multiple dirty) */}
      {dirtyCount > 1 && (
        <button
          type="button"
          onClick={onSaveAll}
          disabled={!canSaveAll}
          className={cn(
            'flex items-center gap-2 px-3 py-1.5 text-sm font-medium rounded-lg transition-colors',
            canSaveAll
              ? 'bg-emerald-600 hover:bg-emerald-500 text-white dark:bg-emerald-600 dark:hover:bg-emerald-500'
              : 'bg-muted text-muted-foreground cursor-not-allowed'
          )}
          title={`Save all ${dirtyCount} pending changes (Ctrl+Shift+S)`}
        >
          <Save className="h-4 w-4" />
          Save All ({dirtyCount})
        </button>
      )}

      {/* Discard */}
      <button
        type="button"
        onClick={onDiscard}
        disabled={!canDiscardBtn}
        className={cn(
          'flex items-center gap-2 px-3 py-1.5 text-sm font-medium rounded-lg transition-colors',
          canDiscardBtn
            ? 'bg-muted hover:bg-muted/80 text-foreground'
            : 'bg-muted text-muted-foreground cursor-not-allowed'
        )}
        title="Discard changes"
      >
        <RotateCcw className="h-4 w-4" />
        Discard
      </button>

      {/* Spacer */}
      <div className="flex-1" />

      {/* Test button (optional) */}
      {onTest && (
        <button
          type="button"
          onClick={onTest}
          disabled={isSaving}
          className={cn(
            'flex items-center gap-2 px-3 py-1.5 text-sm font-medium rounded-lg transition-colors',
            !isSaving
              ? 'bg-amber-600 hover:bg-amber-500 text-white dark:bg-amber-600 dark:hover:bg-amber-500'
              : 'bg-muted text-muted-foreground cursor-not-allowed'
          )}
          title="Test prompt with Ollama"
        >
          <Zap className="h-4 w-4" />
          Test
        </button>
      )}

      {/* Delete */}
      <button
        type="button"
        onClick={onDelete}
        disabled={!canDelete}
        className={cn(
          'flex items-center gap-2 px-3 py-1.5 text-sm font-medium rounded-lg transition-colors',
          canDelete
            ? 'bg-destructive/20 hover:bg-destructive text-destructive hover:text-destructive-foreground border border-destructive/50 hover:border-destructive'
            : 'bg-muted text-muted-foreground cursor-not-allowed'
        )}
        title="Delete prompt"
      >
        <Trash2 className="h-4 w-4" />
        {isDeleting ? 'Deleting...' : 'Delete'}
      </button>
    </div>
  )
}
