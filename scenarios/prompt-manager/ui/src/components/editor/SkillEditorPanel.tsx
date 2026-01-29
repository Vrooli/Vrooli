/**
 * SkillEditorPanel - Container component for the editor sections.
 *
 * Brings together:
 * - SkillContentEditor (full width)
 * - WorldCanvas (when no skill selected)
 *
 * Header contains:
 * - Icon selector
 * - Inline editable name
 * - Draft toggle
 * - Actions menu (discard/delete)
 * - Expandable description
 * - Tag chips editor
 * - Mode path display
 *
 * Also handles:
 * - Empty state with 3D world visualization
 */

import { MoreHorizontal, RotateCcw, Trash2, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useSelectionStore } from '@/stores/selectionStore'
import type { NormalizedFormState, ValidationResult } from '@/types/editorStore'
import type { Skill } from '@/types'
import type { DisplayFormat } from '@/types/world'
import { SkillContentEditor } from './SkillContentEditor'
import { FilePathMenu } from './FilePathMenu'
import { ToolbarDropdown, DropdownItem } from './ToolbarDropdown'
import { WorldCanvas } from '@/components/world'
import { IconSelector } from '../shared/IconSelector'
import { InlineEditableText } from '../shared/InlineEditableText'
import { DraftToggle } from '../shared/DraftToggle'
import { ExpandableDescription } from '../shared/ExpandableDescription'
import { TagChipsEditor } from '../shared/TagChipsEditor'
import { PanelErrorBoundary } from '../PanelErrorBoundary'

interface SkillEditorPanelProps {
  // Current state
  currentSkill: Skill | null
  formState: NormalizedFormState
  originalContent?: string | null
  validation: ValidationResult

  // All skills for skill tree
  allSkills?: Skill[]

  // Dirty tracking
  isDirty: boolean
  dirtyCount: number

  // Form operations
  onFieldChange: <K extends keyof NormalizedFormState>(field: K, value: NormalizedFormState[K]) => void

  // Available tags for autocomplete
  availableTags?: string[]

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
  onSelectSkill?: (skillId: string) => void
  onDisplaySkills?: (combined: string, format: DisplayFormat) => void

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
  originalContent,
  validation,
  allSkills = [],
  isDirty,
  dirtyCount,
  onFieldChange,
  availableTags = [],
  onUndo,
  onRedo,
  canUndo = false,
  canRedo = false,
  onSave,
  onSaveAll,
  onDiscard,
  onDelete,
  onSelectSkill,
  onDisplaySkills,
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
        <PanelErrorBoundary panelName="3D World" className="h-full">
          <WorldCanvas
            skills={allSkills}
            onSelectSkill={onSelectSkill}
            onDisplaySkills={onDisplaySkills}
          />
        </PanelErrorBoundary>
      </div>
    )
  }

  const canDiscard = isDirty && !isSaving
  const canDelete = !isDeleting

  return (
    <div className={cn('h-full', className)}>
      <div className="flex flex-col h-full bg-card/50">
        {/* Header with all metadata */}
        <div className="flex-shrink-0 px-4 py-3 border-b border-border space-y-2">
          {/* Row 1: Close, Icon, Name, Draft toggle, Unsaved indicator, Actions */}
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-2 flex-1 min-w-0">
              {/* Close button */}
              <button
                type="button"
                onClick={handleClose}
                className="p-1.5 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors flex-shrink-0"
                aria-label="Close editor and return to world"
                title="Close (Esc)"
              >
                <X className="h-5 w-5" />
              </button>

              {/* Icon selector */}
              <IconSelector
                value={formState.icon}
                onChange={(v) => onFieldChange('icon', v)}
                className="flex-shrink-0"
              />

              {/* Editable name */}
              <div className="flex-1 min-w-0">
                <InlineEditableText
                  value={formState.name}
                  onChange={(v) => onFieldChange('name', v)}
                  placeholder="Untitled Skill"
                  error={validation.errors.name}
                  as="h2"
                  className="text-foreground"
                />
              </div>
            </div>

            <div className="flex items-center gap-2 flex-shrink-0">
              {/* Draft toggle */}
              <DraftToggle
                isDraft={formState.draft}
                onChange={(v) => onFieldChange('draft', v)}
                className="flex-shrink-0"
              />

              {/* Unsaved indicator */}
              {isDirty && (
                <div className="flex items-center gap-1.5 px-2.5 py-1 bg-amber-500/20 text-amber-300 rounded-md text-xs font-medium flex-shrink-0">
                  Unsaved
                </div>
              )}

              {/* Actions menu */}
              <ToolbarDropdown
                icon={<MoreHorizontal className="h-4 w-4" />}
                label="Skill actions"
                showChevron={false}
                align="right"
                className="p-1.5 rounded-lg"
              >
                <DropdownItem
                  onClick={onDiscard}
                  disabled={!canDiscard}
                  icon={<RotateCcw className="h-4 w-4" />}
                  label="Discard changes"
                />
                <DropdownItem
                  onClick={onDelete}
                  disabled={!canDelete}
                  icon={<Trash2 className="h-4 w-4 text-destructive" />}
                  label={isDeleting ? 'Deleting...' : 'Delete skill'}
                />
              </ToolbarDropdown>
            </div>
          </div>

          {/* Row 2: Description */}
          <ExpandableDescription
            value={formState.description}
            onChange={(v) => onFieldChange('description', v)}
            placeholder="Click to add description..."
            error={validation.errors.description}
            className="text-muted-foreground"
          />

          {/* Row 3: Tags and File path menu */}
          <div className="flex items-center gap-4 flex-wrap">
            <TagChipsEditor
              value={formState.tags}
              onChange={(v) => onFieldChange('tags', v)}
              availableTags={availableTags}
              placeholder="Add tags..."
              className="flex-1 min-w-0"
            />

            {/* File path menu with filename, breadcrumb, copy actions, and storage toggle */}
            <FilePathMenu
              file={formState.file}
              modes={formState.modes}
              folder={formState.folder}
              onFileChange={(v) => onFieldChange('file', v)}
              onFolderChange={(v) => onFieldChange('folder', v)}
              className="flex-shrink-0"
            />
          </div>
        </div>

        {/* Content area - full width */}
        <div className="flex-1 overflow-hidden">
          <SkillContentEditor
            value={formState.content}
            originalValue={originalContent ?? undefined}
            onChange={(v) => onFieldChange('content', v)}
            error={validation.errors.content}
            isDirty={isDirty}
            dirtyCount={dirtyCount}
            onUndo={onUndo}
            onRedo={onRedo}
            canUndo={canUndo}
            canRedo={canRedo}
            onSave={onSave}
            onSaveAll={onSaveAll}
            onDiscard={onDiscard}
            isSaving={isSaving}
            isValid={validation.valid}
            className="h-full"
          />
        </div>
      </div>
    </div>
  )
}
