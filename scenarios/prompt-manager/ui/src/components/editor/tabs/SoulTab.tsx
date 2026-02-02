/**
 * SoulTab - Agent SOUL.md editor.
 *
 * Features:
 * - Full markdown editor for SOUL.md content (code/WYSIWYG modes, preview, diff)
 * - Uses centralized form state for dirty tracking
 */

import { useCallback } from 'react'
import type { NormalizedAgentFormState } from '@/stores/agentEditorStore'
import { SkillContentEditor } from '../SkillContentEditor'

interface SoulTabProps {
  /** Form state from the editor store */
  formState: NormalizedAgentFormState
  /** Original state for diff comparison */
  originalState: NormalizedAgentFormState | null
  /** Update a single field */
  updateField: <K extends keyof NormalizedAgentFormState>(field: K, value: NormalizedAgentFormState[K]) => void
  /** Update multiple fields at once */
  updateFields: (updates: Partial<NormalizedAgentFormState>) => void
  /** Whether the form has unsaved changes */
  isDirty: boolean
  /** Count of dirty entities */
  dirtyCount: number
  /** Undo last change */
  onUndo: () => void
  /** Redo last undone change */
  onRedo: () => void
  /** Whether undo is available */
  canUndo: boolean
  /** Whether redo is available */
  canRedo: boolean
  /** Save current changes */
  onSave: () => void
  /** Discard current changes */
  onDiscard: () => void
  /** Whether saving is in progress */
  isSaving: boolean
  /** Whether the form is valid */
  isValid: boolean
}

/**
 * SOUL.md editor tab component.
 */
export function SoulTab({
  formState,
  originalState,
  updateField,
  updateFields: _updateFields,
  isDirty,
  dirtyCount,
  onUndo,
  onRedo,
  canUndo,
  canRedo,
  onSave,
  onDiscard,
  isSaving,
  isValid,
}: SoulTabProps) {
  // TODO: Use updateFields for batch updates if needed
  void _updateFields

  const content = formState.soul
  const originalContent = originalState?.soul ?? ''

  const handleContentChange = useCallback((newContent: string) => {
    updateField('soul', newContent)
  }, [updateField])

  return (
    <div className="flex flex-col h-full">
      <div className="flex-1 min-h-0">
        <SkillContentEditor
          value={content}
          originalValue={originalContent}
          onChange={handleContentChange}
          isDirty={isDirty}
          dirtyCount={dirtyCount}
          onUndo={onUndo}
          onRedo={onRedo}
          canUndo={canUndo}
          canRedo={canRedo}
          onSave={onSave}
          onDiscard={onDiscard}
          isSaving={isSaving}
          isValid={isValid}
          className="h-full"
        />
      </div>
    </div>
  )
}
