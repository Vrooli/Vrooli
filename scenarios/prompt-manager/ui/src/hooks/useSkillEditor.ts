/**
 * useSkillEditor - Multi-item editing state management.
 *
 * Handles:
 * - Current skill form state
 * - Pending changes for multiple skills
 * - Dirty tracking across all edited skills
 * - Save/discard operations
 */

import { useState, useCallback, useMemo, useEffect } from 'react'
import type { Skill, UpdateSkillRequest } from '@/types'
import type { SkillFormState, PendingChange, ValidationResult } from '@/types/editor'
import {
  skillToFormState,
  formStateToUpdateRequest,
  isDirty,
  validateFormState,
  createEmptyFormState,
} from '@/services/editorService'
import * as skillService from '@/services/skillService'

interface UseSkillEditorProps {
  skills: Skill[]
  selectedItemId: string | null
  onSave: (updates: Map<string, UpdateSkillRequest>) => Promise<Map<string, Skill | Error>>
  onDelete: (id: string) => Promise<void>
}

interface UseSkillEditorReturn {
  // Current editor state
  currentSkill: Skill | null
  formState: SkillFormState

  // Form operations
  updateField: <K extends keyof SkillFormState>(field: K, value: SkillFormState[K]) => void
  setModes: (modes: string[]) => void
  resetForm: () => void

  // Validation
  validation: ValidationResult
  isValid: boolean

  // Dirty tracking
  isDirty: boolean
  dirtyItemIds: Set<string>
  dirtyCount: number

  // Pending changes (for other skills)
  pendingChanges: Map<string, PendingChange>
  storeCurrentChanges: () => void

  // Save operations
  saveCurrentSkill: () => Promise<void>
  saveAllChanges: () => Promise<void>
  discardCurrentChanges: () => void
  discardAllChanges: () => void

  // Delete
  deleteCurrentSkill: () => Promise<void>

  // Loading states
  isSaving: boolean
  isDeleting: boolean
}

/**
 * Hook for managing skill editing with multi-item support.
 */
export function useSkillEditor({
  skills,
  selectedItemId,
  onSave,
  onDelete,
}: UseSkillEditorProps): UseSkillEditorReturn {
  // Pending changes for skills other than the currently selected one
  const [pendingChanges, setPendingChanges] = useState<Map<string, PendingChange>>(new Map())

  // Current form state (for the selected skill)
  const [formState, setFormState] = useState<SkillFormState>(createEmptyFormState())

  // Loading states
  const [isSaving, setIsSaving] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)

  // Get the current skill object
  const currentSkill = useMemo(() => {
    if (!selectedItemId) return null
    return skills.find((p) => p.id === selectedItemId) ?? null
  }, [skills, selectedItemId])

  // Check if current form has changes
  const currentIsDirty = useMemo(() => {
    if (!currentSkill) return false
    return isDirty(currentSkill, formState)
  }, [currentSkill, formState])

  // Validate current form
  const validation = useMemo(() => validateFormState(formState), [formState])

  // Compute all dirty item IDs (including pending changes)
  const dirtyItemIds = useMemo(() => {
    const ids = new Set<string>()

    // Add current skill if dirty
    if (currentSkill && currentIsDirty) {
      ids.add(currentSkill.id)
    }

    // Add all pending changes that are dirty
    for (const [id, change] of pendingChanges) {
      if (change.isDirty) {
        ids.add(id)
      }
    }

    return ids
  }, [currentSkill, currentIsDirty, pendingChanges])

  // Store current changes before switching skills
  const storeCurrentChanges = useCallback(() => {
    if (!currentSkill || !currentIsDirty) return

    setPendingChanges((prev) => {
      const next = new Map(prev)
      next.set(currentSkill.id, {
        original: currentSkill,
        current: { ...formState },
        isDirty: true,
      })
      return next
    })
  }, [currentSkill, currentIsDirty, formState])

  // Effect 1: Clear form when deselected
  useEffect(() => {
    if (!selectedItemId) {
      setFormState(createEmptyFormState())
    }
  }, [selectedItemId])

  // Effect 2: Load form when selection or skills change
  // Fetches full skill with content if the list skill doesn't include content
  useEffect(() => {
    if (!selectedItemId) return

    // Check if we have pending changes for this skill
    const pending = pendingChanges.get(selectedItemId)
    if (pending) {
      setFormState(pending.current)
      return
    }

    // Find skill directly instead of relying on memo to avoid race condition
    const listSkill = skills.find((p) => p.id === selectedItemId)
    if (!listSkill) return

    // If list skill has content, use it directly
    if (listSkill.content && listSkill.content.trim()) {
      setFormState(skillToFormState(listSkill))
      return
    }

    // Fetch full skill with content from API
    skillService.getSkill(selectedItemId)
      .then((fullSkill) => {
        if (fullSkill) {
          setFormState(skillToFormState(fullSkill))
        } else {
          // Fallback to list skill if fetch fails
          setFormState(skillToFormState(listSkill))
        }
      })
      .catch(() => {
        // Fallback to list skill on error
        setFormState(skillToFormState(listSkill))
      })
    // eslint-disable-next-line react-hooks/exhaustive-deps -- pendingChanges intentionally excluded to prevent stale closure
  }, [selectedItemId, skills])

  // Update a single form field
  const updateField = useCallback(
    <K extends keyof SkillFormState>(field: K, value: SkillFormState[K]) => {
      setFormState((prev) => ({ ...prev, [field]: value }))
    },
    []
  )

  // Update modes array
  const setModes = useCallback(
    (modes: string[]) => {
      setFormState((prev) => ({ ...prev, modes }))
    },
    []
  )

  // Reset form to original skill state
  const resetForm = useCallback(() => {
    if (!currentSkill) return
    setFormState(skillToFormState(currentSkill))
  }, [currentSkill])

  // Save current skill only
  const saveCurrentSkill = useCallback(async () => {
    if (!currentSkill || !currentIsDirty || !validation.valid) return

    setIsSaving(true)
    try {
      const updates = new Map<string, UpdateSkillRequest>()
      updates.set(currentSkill.id, formStateToUpdateRequest(formState))
      await onSave(updates)

      // Clear from pending changes if it was there
      setPendingChanges((prev) => {
        const next = new Map(prev)
        next.delete(currentSkill.id)
        return next
      })
    } finally {
      setIsSaving(false)
    }
  }, [currentSkill, currentIsDirty, validation, formState, onSave])

  // Save all pending changes
  const saveAllChanges = useCallback(async () => {
    const updates = new Map<string, UpdateSkillRequest>()

    // Add current skill if dirty and valid
    if (currentSkill && currentIsDirty && validation.valid) {
      updates.set(currentSkill.id, formStateToUpdateRequest(formState))
    }

    // Add all pending changes that are dirty
    for (const [id, change] of pendingChanges) {
      if (change.isDirty) {
        const pendingValidation = validateFormState(change.current)
        if (pendingValidation.valid) {
          updates.set(id, formStateToUpdateRequest(change.current))
        }
      }
    }

    if (updates.size === 0) return

    setIsSaving(true)
    try {
      const results = await onSave(updates)

      // Clear successfully saved items from pending
      setPendingChanges((prev) => {
        const next = new Map(prev)
        for (const [id, result] of results) {
          if (!(result instanceof Error)) {
            next.delete(id)
          }
        }
        return next
      })
    } finally {
      setIsSaving(false)
    }
  }, [currentSkill, currentIsDirty, validation, formState, pendingChanges, onSave])

  // Discard current skill changes
  const discardCurrentChanges = useCallback(() => {
    if (!currentSkill) return

    // Reset form to original
    setFormState(skillToFormState(currentSkill))

    // Remove from pending changes
    setPendingChanges((prev) => {
      const next = new Map(prev)
      next.delete(currentSkill.id)
      return next
    })
  }, [currentSkill])

  // Discard all changes
  const discardAllChanges = useCallback(() => {
    // Reset current form
    if (currentSkill) {
      setFormState(skillToFormState(currentSkill))
    }

    // Clear all pending changes
    setPendingChanges(new Map())
  }, [currentSkill])

  // Delete current skill
  const deleteCurrentSkill = useCallback(async () => {
    if (!currentSkill) return

    setIsDeleting(true)
    try {
      await onDelete(currentSkill.id)

      // Clear from pending changes
      setPendingChanges((prev) => {
        const next = new Map(prev)
        next.delete(currentSkill.id)
        return next
      })
    } finally {
      setIsDeleting(false)
    }
  }, [currentSkill, onDelete])

  return {
    // Current editor state
    currentSkill,
    formState,

    // Form operations
    updateField,
    setModes,
    resetForm,

    // Validation
    validation,
    isValid: validation.valid,

    // Dirty tracking
    isDirty: currentIsDirty,
    dirtyItemIds,
    dirtyCount: dirtyItemIds.size,

    // Pending changes
    pendingChanges,
    storeCurrentChanges,

    // Save operations
    saveCurrentSkill,
    saveAllChanges,
    discardCurrentChanges,
    discardAllChanges,

    // Delete
    deleteCurrentSkill,

    // Loading states
    isSaving,
    isDeleting,
  }
}
