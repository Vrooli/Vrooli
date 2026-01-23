/**
 * SkillEditorPanel - Container component for the editor sections.
 *
 * Brings together:
 * - EditorToolbar
 * - SkillMetadataForm
 * - SkillContentEditor
 * - WorldCanvas (when no skill selected)
 *
 * Also handles:
 * - Empty state with 3D world visualization
 */

import { X, FileText } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useSelectionStore } from '@/stores/selectionStore'
import type { SkillFormState, ValidationResult } from '@/types/editor'
import type { Skill } from '@/types'
import type { CombineFormat } from '@/types/world'
import { EditorToolbar } from './EditorToolbar'
import { SkillMetadataForm } from './SkillMetadataForm'
import { SkillContentEditor } from './SkillContentEditor'
import { WorldCanvas } from '@/components/world'

interface SkillEditorPanelProps {
  // Current state
  currentSkill: Skill | null
  formState: SkillFormState
  validation: ValidationResult

  // All skills for skill tree
  allSkills?: Skill[]

  // Dirty tracking
  isDirty: boolean
  dirtyCount: number

  // Form operations
  onFieldChange: <K extends keyof SkillFormState>(field: K, value: SkillFormState[K]) => void
  onModesChange: (modes: string[]) => void
  getSuggestionsAtLevel: (level: number, parentPath: string[]) => string[]

  // Actions
  onSave: () => void
  onSaveAll: () => void
  onDiscard: () => void
  onDelete: () => void
  onSelectSkill?: (skillId: string) => void
  onCombineSkills?: (combined: string, format: CombineFormat) => void

  // Loading states
  isSaving: boolean
  isDeleting: boolean

  className?: string
}

/**
 * Main editor panel component.
 */
export function SkillEditorPanel({
  currentSkill,
  formState,
  validation,
  allSkills = [],
  isDirty,
  dirtyCount,
  onFieldChange,
  onModesChange,
  getSuggestionsAtLevel,
  onSave,
  onSaveAll,
  onDiscard,
  onDelete,
  onSelectSkill,
  onCombineSkills,
  isSaving,
  isDeleting,
  className,
}: SkillEditorPanelProps) {
  // Access the selection store for closing the editor
  const setSelectedSkillId = useSelectionStore((state) => state.setSelectedSkillId)

  // Handle close - return to skill tree view
  const handleClose = () => {
    setSelectedSkillId(null)
  }

  // Show 3D world when no skill selected
  if (!currentSkill) {
    return (
      <div className={cn('h-full', className)}>
        <WorldCanvas
          skills={allSkills}
          onSelectSkill={onSelectSkill}
          onCombineSkills={onCombineSkills}
        />
      </div>
    )
  }

  return (
    <div className={cn('h-full', className)}>
      <div
        className="flex flex-col h-full bg-card/50"
      >
      {/* Header with skill name and navigation */}
      <div className="flex-shrink-0 px-4 py-3 border-b border-border">
        {/* Top row: Close button, skill name, and status */}
        <div className="flex items-center gap-3 mb-3">
          {/* Close button */}
          <button
            type="button"
            onClick={handleClose}
            className="p-1.5 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
            aria-label="Close editor and return to world"
            title="Close (Esc)"
          >
            <X className="h-5 w-5" />
          </button>

          {/* Skill icon and name */}
          <div className="flex items-center gap-2 flex-1 min-w-0">
            <FileText className="h-5 w-5 text-primary flex-shrink-0" />
            <h2 className="text-lg font-semibold text-foreground truncate">
              {formState.name || 'Untitled Skill'}
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
            <SkillMetadataForm
              formState={formState}
              onFieldChange={onFieldChange}
              onModesChange={onModesChange}
              getSuggestionsAtLevel={getSuggestionsAtLevel}
              validation={validation}
            />
          </div>

          {/* Content editor - 2/3 width on large screens, takes remaining height */}
          <div className="flex-1 mt-4 xl:mt-0 min-h-[300px] xl:min-h-0">
            <SkillContentEditor
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
