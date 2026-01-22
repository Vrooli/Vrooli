/**
 * usePromptEditor - Multi-item editing state management.
 *
 * Handles:
 * - Current prompt form state
 * - Pending changes for multiple prompts
 * - Dirty tracking across all edited prompts
 * - Save/discard operations
 */

import { useState, useCallback, useMemo, useEffect } from 'react'
import type { Prompt, UpdatePromptRequest } from '@/types'
import type { PromptFormState, PendingChange, ValidationResult } from '@/types/editor'
import {
  promptToFormState,
  formStateToUpdateRequest,
  isDirty,
  validateFormState,
  createEmptyFormState,
} from '@/services/editorService'
import { isEditable } from '@/services/promptService'

interface UsePromptEditorProps {
  prompts: Prompt[]
  selectedItemId: string | null
  onSave: (updates: Map<string, UpdatePromptRequest>) => Promise<Map<string, Prompt | Error>>
  onDelete: (id: string) => Promise<void>
}

interface UsePromptEditorReturn {
  // Current editor state
  currentPrompt: Prompt | null
  formState: PromptFormState
  isReadonly: boolean

  // Form operations
  updateField: <K extends keyof PromptFormState>(field: K, value: PromptFormState[K]) => void
  setModes: (modes: string[]) => void
  resetForm: () => void

  // Validation
  validation: ValidationResult
  isValid: boolean

  // Dirty tracking
  isDirty: boolean
  dirtyItemIds: Set<string>
  dirtyCount: number

  // Pending changes (for other prompts)
  pendingChanges: Map<string, PendingChange>
  storeCurrentChanges: () => void

  // Save operations
  saveCurrentPrompt: () => Promise<void>
  saveAllChanges: () => Promise<void>
  discardCurrentChanges: () => void
  discardAllChanges: () => void

  // Delete
  deleteCurrentPrompt: () => Promise<void>

  // Loading states
  isSaving: boolean
  isDeleting: boolean
}

/**
 * Hook for managing prompt editing with multi-item support.
 */
export function usePromptEditor({
  prompts,
  selectedItemId,
  onSave,
  onDelete,
}: UsePromptEditorProps): UsePromptEditorReturn {
  // Pending changes for prompts other than the currently selected one
  const [pendingChanges, setPendingChanges] = useState<Map<string, PendingChange>>(new Map())

  // Current form state (for the selected prompt)
  const [formState, setFormState] = useState<PromptFormState>(createEmptyFormState())

  // Loading states
  const [isSaving, setIsSaving] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)

  // Get the current prompt object
  const currentPrompt = useMemo(() => {
    if (!selectedItemId) return null
    return prompts.find((p) => p.id === selectedItemId) ?? null
  }, [prompts, selectedItemId])

  // Check if current prompt is readonly
  const isReadonly = useMemo(() => {
    if (!currentPrompt) return true
    return !isEditable(currentPrompt)
  }, [currentPrompt])

  // Check if current form has changes
  const currentIsDirty = useMemo(() => {
    if (!currentPrompt) return false
    return isDirty(currentPrompt, formState)
  }, [currentPrompt, formState])

  // Validate current form
  const validation = useMemo(() => validateFormState(formState), [formState])

  // Compute all dirty item IDs (including pending changes)
  const dirtyItemIds = useMemo(() => {
    const ids = new Set<string>()

    // Add current prompt if dirty
    if (currentPrompt && currentIsDirty) {
      ids.add(currentPrompt.id)
    }

    // Add all pending changes that are dirty
    for (const [id, change] of pendingChanges) {
      if (change.isDirty) {
        ids.add(id)
      }
    }

    return ids
  }, [currentPrompt, currentIsDirty, pendingChanges])

  // Store current changes before switching prompts
  const storeCurrentChanges = useCallback(() => {
    if (!currentPrompt || !currentIsDirty) return

    setPendingChanges((prev) => {
      const next = new Map(prev)
      next.set(currentPrompt.id, {
        original: currentPrompt,
        current: { ...formState },
        isDirty: true,
      })
      return next
    })
  }, [currentPrompt, currentIsDirty, formState])

  // Load form state when selected prompt changes
  useEffect(() => {
    if (!selectedItemId) {
      setFormState(createEmptyFormState())
      return
    }

    // Check if we have pending changes for this prompt
    const pending = pendingChanges.get(selectedItemId)
    if (pending) {
      setFormState(pending.current)
      return
    }

    // Load from current prompt
    if (currentPrompt) {
      setFormState(promptToFormState(currentPrompt))
    }
  }, [selectedItemId, currentPrompt, pendingChanges])

  // Update a single form field
  const updateField = useCallback(
    <K extends keyof PromptFormState>(field: K, value: PromptFormState[K]) => {
      if (isReadonly) return
      setFormState((prev) => ({ ...prev, [field]: value }))
    },
    [isReadonly]
  )

  // Update modes array
  const setModes = useCallback(
    (modes: string[]) => {
      if (isReadonly) return
      setFormState((prev) => ({ ...prev, modes }))
    },
    [isReadonly]
  )

  // Reset form to original prompt state
  const resetForm = useCallback(() => {
    if (!currentPrompt) return
    setFormState(promptToFormState(currentPrompt))
  }, [currentPrompt])

  // Save current prompt only
  const saveCurrentPrompt = useCallback(async () => {
    if (!currentPrompt || !currentIsDirty || !validation.valid) return

    setIsSaving(true)
    try {
      const updates = new Map<string, UpdatePromptRequest>()
      updates.set(currentPrompt.id, formStateToUpdateRequest(formState))
      await onSave(updates)

      // Clear from pending changes if it was there
      setPendingChanges((prev) => {
        const next = new Map(prev)
        next.delete(currentPrompt.id)
        return next
      })
    } finally {
      setIsSaving(false)
    }
  }, [currentPrompt, currentIsDirty, validation, formState, onSave])

  // Save all pending changes
  const saveAllChanges = useCallback(async () => {
    const updates = new Map<string, UpdatePromptRequest>()

    // Add current prompt if dirty and valid
    if (currentPrompt && currentIsDirty && validation.valid) {
      updates.set(currentPrompt.id, formStateToUpdateRequest(formState))
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
  }, [currentPrompt, currentIsDirty, validation, formState, pendingChanges, onSave])

  // Discard current prompt changes
  const discardCurrentChanges = useCallback(() => {
    if (!currentPrompt) return

    // Reset form to original
    setFormState(promptToFormState(currentPrompt))

    // Remove from pending changes
    setPendingChanges((prev) => {
      const next = new Map(prev)
      next.delete(currentPrompt.id)
      return next
    })
  }, [currentPrompt])

  // Discard all changes
  const discardAllChanges = useCallback(() => {
    // Reset current form
    if (currentPrompt) {
      setFormState(promptToFormState(currentPrompt))
    }

    // Clear all pending changes
    setPendingChanges(new Map())
  }, [currentPrompt])

  // Delete current prompt
  const deleteCurrentPrompt = useCallback(async () => {
    if (!currentPrompt || isReadonly) return

    setIsDeleting(true)
    try {
      await onDelete(currentPrompt.id)

      // Clear from pending changes
      setPendingChanges((prev) => {
        const next = new Map(prev)
        next.delete(currentPrompt.id)
        return next
      })
    } finally {
      setIsDeleting(false)
    }
  }, [currentPrompt, isReadonly, onDelete])

  return {
    // Current editor state
    currentPrompt,
    formState,
    isReadonly,

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
    saveCurrentPrompt,
    saveAllChanges,
    discardCurrentChanges,
    discardAllChanges,

    // Delete
    deleteCurrentPrompt,

    // Loading states
    isSaving,
    isDeleting,
  }
}
