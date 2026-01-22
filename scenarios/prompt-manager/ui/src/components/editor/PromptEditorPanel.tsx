/**
 * PromptEditorPanel - Container component for the editor sections.
 *
 * Brings together:
 * - EditorToolbar
 * - PromptMetadataForm
 * - PromptContentEditor
 * - SkillTreeCanvas (when no prompt selected)
 *
 * Also handles:
 * - Empty state with 3D skill tree visualization
 */

import { X, FileText } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useSelectionStore } from '@/stores/selectionStore'
import type { PromptFormState, ValidationResult } from '@/types/editor'
import type { Prompt } from '@/types'
import type { CombineFormat } from '@/types/skilltree'
import { EditorToolbar } from './EditorToolbar'
import { PromptMetadataForm } from './PromptMetadataForm'
import { PromptContentEditor } from './PromptContentEditor'
import { SkillTreeCanvas } from '@/components/skilltree'

interface PromptEditorPanelProps {
  // Current state
  currentPrompt: Prompt | null
  formState: PromptFormState
  validation: ValidationResult

  // All prompts for skill tree
  allPrompts?: Prompt[]

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
  onSelectPrompt?: (promptId: string) => void
  onCombinePrompts?: (combined: string, format: CombineFormat) => void

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
  allPrompts = [],
  isDirty,
  dirtyCount,
  onFieldChange,
  onModesChange,
  getSuggestionsAtLevel,
  onSave,
  onSaveAll,
  onDiscard,
  onDelete,
  onSelectPrompt,
  onCombinePrompts,
  isSaving,
  isDeleting,
  className,
}: PromptEditorPanelProps) {
  // Access the selection store for closing the editor
  const setSelectedPromptId = useSelectionStore((state) => state.setSelectedPromptId)

  // Handle close - return to skill tree view
  const handleClose = () => {
    setSelectedPromptId(null)
  }

  // Show 3D skill tree when no prompt selected
  if (!currentPrompt) {
    return (
      <div className={cn('h-full', className)}>
        <SkillTreeCanvas
          prompts={allPrompts}
          onSelectPrompt={onSelectPrompt}
          onCombinePrompts={onCombinePrompts}
        />
      </div>
    )
  }

  return (
    <div className={cn('h-full', className)}>
      <div
        className="flex flex-col h-full bg-slate-900/50"
      >
      {/* Header with prompt name and navigation */}
      <div className="flex-shrink-0 px-4 py-3 border-b border-white/10">
        {/* Top row: Close button, prompt name, and status */}
        <div className="flex items-center gap-3 mb-3">
          {/* Close button */}
          <button
            type="button"
            onClick={handleClose}
            className="p-1.5 rounded-lg hover:bg-white/10 text-slate-400 hover:text-white transition-colors"
            aria-label="Close editor and return to skill tree"
            title="Close (Esc)"
          >
            <X className="h-5 w-5" />
          </button>

          {/* Prompt icon and name */}
          <div className="flex items-center gap-2 flex-1 min-w-0">
            <FileText className="h-5 w-5 text-indigo-400 flex-shrink-0" />
            <h2 className="text-lg font-semibold text-white truncate">
              {formState.name || 'Untitled Prompt'}
            </h2>
          </div>

          {/* Status indicator */}
          {isDirty && (
            <div className="flex items-center gap-1.5 px-2.5 py-1 bg-amber-500/20 text-amber-300 rounded-md text-xs font-medium flex-shrink-0">
              Unsaved changes
            </div>
          )}
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
