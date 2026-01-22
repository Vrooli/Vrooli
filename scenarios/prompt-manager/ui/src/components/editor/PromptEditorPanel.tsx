/**
 * PromptEditorPanel - Container component for the editor sections.
 *
 * Brings together:
 * - EditorToolbar
 * - PromptMetadataForm
 * - PromptContentEditor
 *
 * Also handles:
 * - Empty state when no prompt selected
 * - Read-only indicator for core prompts
 */

import { Lock, FileText } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { PromptFormState, ValidationResult } from '@/types/editor'
import type { Prompt } from '@/types'
import { EditorToolbar } from './EditorToolbar'
import { PromptMetadataForm } from './PromptMetadataForm'
import { PromptContentEditor } from './PromptContentEditor'

interface PromptEditorPanelProps {
  // Current state
  currentPrompt: Prompt | null
  formState: PromptFormState
  validation: ValidationResult
  isReadonly: boolean

  // Dirty tracking
  isDirty: boolean
  dirtyCount: number

  // Form operations
  onFieldChange: <K extends keyof PromptFormState>(field: K, value: PromptFormState[K]) => void
  onModesChange: (modes: string[]) => void
  getSuggestionsAtLevel: (level: number, parentPath: string[]) => string[]

  // Actions
  onSave: () => void
  onSaveAll: () => void
  onDiscard: () => void
  onDelete: () => void

  // Loading states
  isSaving: boolean
  isDeleting: boolean

  className?: string
}

/**
 * Main editor panel component.
 */
export function PromptEditorPanel({
  currentPrompt,
  formState,
  validation,
  isReadonly,
  isDirty,
  dirtyCount,
  onFieldChange,
  onModesChange,
  getSuggestionsAtLevel,
  onSave,
  onSaveAll,
  onDiscard,
  onDelete,
  isSaving,
  isDeleting,
  className,
}: PromptEditorPanelProps) {
  // Empty state when no prompt selected
  if (!currentPrompt) {
    return (
      <div
        className={cn(
          'flex flex-col items-center justify-center h-full',
          'bg-slate-900/50 rounded-lg border border-white/10',
          className
        )}
      >
        <div className="text-center">
          <div className="w-16 h-16 mx-auto mb-4 bg-slate-800 rounded-2xl flex items-center justify-center">
            <FileText className="h-8 w-8 text-slate-600" />
          </div>
          <h3 className="text-lg font-medium text-slate-400 mb-2">No Prompt Selected</h3>
          <p className="text-sm text-slate-500 max-w-xs">
            Select a prompt from the tree to view and edit, or create a new one.
          </p>
        </div>
      </div>
    )
  }

  return (
    <div
      className={cn(
        'flex flex-col h-full bg-slate-900/50 rounded-lg border border-white/10',
        className
      )}
    >
      {/* Header with read-only indicator */}
      <div className="flex-shrink-0 px-4 py-3 border-b border-white/10">
        <div className="flex items-center justify-between mb-2">
          <div className="flex items-center gap-2">
            {isReadonly && (
              <div className="flex items-center gap-1.5 px-2 py-1 bg-amber-500/20 text-amber-300 rounded-md text-xs">
                <Lock className="h-3 w-3" />
                Read-only
              </div>
            )}
            {isDirty && !isReadonly && (
              <div className="flex items-center gap-1.5 px-2 py-1 bg-amber-500/20 text-amber-300 rounded-md text-xs">
                Unsaved changes
              </div>
            )}
          </div>
        </div>

        {/* Toolbar */}
        <EditorToolbar
          isDirty={isDirty}
          dirtyCount={dirtyCount}
          onSave={onSave}
          onSaveAll={onSaveAll}
          onDiscard={onDiscard}
          onDelete={onDelete}
          isSaving={isSaving}
          isDeleting={isDeleting}
          isReadonly={isReadonly}
          isValid={validation.valid}
        />
      </div>

      {/* Content area - scrollable */}
      <div className="flex-1 overflow-y-auto">
        <div className="p-4 space-y-6">
          {/* Metadata form */}
          <PromptMetadataForm
            formState={formState}
            onFieldChange={onFieldChange}
            onModesChange={onModesChange}
            getSuggestionsAtLevel={getSuggestionsAtLevel}
            validation={validation}
            disabled={isReadonly}
          />
        </div>
      </div>

      {/* Content editor - fixed height section */}
      <div className="flex-shrink-0 h-[40vh] min-h-[200px] px-4 pb-4">
        <PromptContentEditor
          value={formState.content}
          onChange={(v) => onFieldChange('content', v)}
          disabled={isReadonly}
          error={validation.errors.content}
          className="h-full"
        />
      </div>
    </div>
  )
}
