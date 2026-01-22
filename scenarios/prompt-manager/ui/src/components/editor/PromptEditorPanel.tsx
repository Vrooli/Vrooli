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
 */

import { FileText } from 'lucide-react'
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
      <div className={cn('p-4 h-full', className)}>
        <div
          className={cn(
            'flex flex-col items-center justify-center h-full',
            'bg-slate-900/50 rounded-lg border border-white/10'
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
      </div>
    )
  }

  return (
    <div className={cn('p-4 h-full', className)}>
      <div
        className="flex flex-col h-full bg-slate-900/50 rounded-lg border border-white/10"
      >
      {/* Header with status indicator */}
      <div className="flex-shrink-0 px-4 py-3 border-b border-white/10">
        <div className="flex items-center justify-between mb-2">
          <div className="flex items-center gap-2">
            {isDirty && (
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
          isValid={validation.valid}
        />
      </div>

      {/* Content area - responsive layout */}
      <div className="flex-1 overflow-y-auto p-4">
        {/* On large screens (>=1280px): side-by-side layout */}
        {/* On smaller screens: stacked layout */}
        <div className="h-full flex flex-col xl:flex-row xl:gap-4">
          {/* Metadata form - 1/3 width on large screens */}
          <div className="flex-shrink-0 xl:w-1/3 xl:max-w-md xl:overflow-y-auto">
            <PromptMetadataForm
              formState={formState}
              onFieldChange={onFieldChange}
              onModesChange={onModesChange}
              getSuggestionsAtLevel={getSuggestionsAtLevel}
              validation={validation}
            />
          </div>

          {/* Content editor - 2/3 width on large screens, takes remaining height */}
          <div className="flex-1 mt-4 xl:mt-0 min-h-[300px] xl:min-h-0">
            <PromptContentEditor
              value={formState.content}
              onChange={(v) => onFieldChange('content', v)}
              error={validation.errors.content}
              className="h-full"
            />
          </div>
        </div>
      </div>
      </div>
    </div>
  )
}
