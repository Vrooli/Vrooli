/**
 * EditorToolbar - Action buttons for the prompt editor.
 *
 * Includes:
 * - Save button
 * - Save All button (when multiple dirty)
 * - Discard button
 * - Delete button
 */

import { Save, Trash2, RotateCcw, Zap } from 'lucide-react'
import { cn } from '@/lib/utils'

interface EditorToolbarProps {
  // Dirty state
  isDirty: boolean
  dirtyCount: number

  // Actions
  onSave: () => void
  onSaveAll: () => void
  onDiscard: () => void
  onDelete: () => void

  // Loading/disabled states
  isSaving: boolean
  isDeleting: boolean
  isReadonly: boolean
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
  onSave,
  onSaveAll,
  onDiscard,
  onDelete,
  isSaving,
  isDeleting,
  isReadonly,
  isValid,
  onTest,
  className,
}: EditorToolbarProps) {
  const canSave = isDirty && !isReadonly && !isSaving && isValid
  const canSaveAll = dirtyCount > 0 && !isSaving
  const canDiscard = isDirty && !isSaving
  const canDelete = !isReadonly && !isDeleting

  return (
    <div className={cn('flex items-center gap-2', className)}>
      {/* Save current */}
      <button
        type="button"
        onClick={onSave}
        disabled={!canSave}
        className={cn(
          'flex items-center gap-2 px-3 py-1.5 text-sm font-medium rounded-lg transition-colors',
          canSave
            ? 'bg-indigo-600 hover:bg-indigo-500 text-white'
            : 'bg-slate-800 text-slate-500 cursor-not-allowed'
        )}
        title={isReadonly ? 'Cannot edit read-only prompt' : isDirty ? 'Save changes' : 'No changes to save'}
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
              ? 'bg-emerald-600 hover:bg-emerald-500 text-white'
              : 'bg-slate-800 text-slate-500 cursor-not-allowed'
          )}
          title={`Save all ${dirtyCount} pending changes`}
        >
          <Save className="h-4 w-4" />
          Save All ({dirtyCount})
        </button>
      )}

      {/* Discard */}
      <button
        type="button"
        onClick={onDiscard}
        disabled={!canDiscard}
        className={cn(
          'flex items-center gap-2 px-3 py-1.5 text-sm font-medium rounded-lg transition-colors',
          canDiscard
            ? 'bg-slate-700 hover:bg-slate-600 text-white'
            : 'bg-slate-800 text-slate-500 cursor-not-allowed'
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
          disabled={isReadonly || isSaving}
          className={cn(
            'flex items-center gap-2 px-3 py-1.5 text-sm font-medium rounded-lg transition-colors',
            !isReadonly && !isSaving
              ? 'bg-amber-600 hover:bg-amber-500 text-white'
              : 'bg-slate-800 text-slate-500 cursor-not-allowed'
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
            ? 'bg-red-600/20 hover:bg-red-600 text-red-400 hover:text-white border border-red-600/50 hover:border-red-600'
            : 'bg-slate-800 text-slate-500 cursor-not-allowed'
        )}
        title={isReadonly ? 'Cannot delete read-only prompt' : 'Delete prompt'}
      >
        <Trash2 className="h-4 w-4" />
        {isDeleting ? 'Deleting...' : 'Delete'}
      </button>
    </div>
  )
}
