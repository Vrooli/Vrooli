/**
 * usePromptEditor - Hook for managing prompt editing with the centralized store.
 *
 * This hook bridges the Zustand editorStore with React components.
 * It provides the same API surface as the legacy useSkillEditor for easy migration,
 * plus new capabilities like undo/redo.
 *
 * Key improvements over useSkillEditor:
 * - Normalized form state (tags as string[], not comma-separated)
 * - Per-prompt undo/redo history
 * - localStorage persistence (survives page refresh)
 * - Multi-prompt editing (switch prompts without losing changes)
 */

import { useState, useCallback, useMemo, useEffect, useRef } from 'react'
import type { Skill, UpdateSkillRequest } from '@/types'
import type { NormalizedFormState, ValidationResult } from '@/types/editorStore'
import {
  useEditorStore,
  createEmptyNormalizedState,
  isFormStateEqual,
  validateNormalizedState,
} from '@/stores/editorStore'
import { normalizedStateToUpdateRequest } from '@/services/editorService'
import * as skillService from '@/services/skillService'

/** Result returned from save operations */
export interface SaveResult {
  success: boolean
  savedCount: number
  failedCount: number
  errors: Array<{ id: string; name: string; message: string }>
  /** If the skill was renamed, this contains the new ID */
  newId?: string
}

interface UsePromptEditorProps {
  skills: Skill[]
  selectedItemId: string | null
  onSave: (updates: Map<string, UpdateSkillRequest>) => Promise<Map<string, Skill | Error>>
  onDelete: (id: string) => Promise<void>
}

interface UsePromptEditorReturn {
  // Current editor state
  currentSkill: Skill | null
  formState: NormalizedFormState
  originalContent: string | null

  // Form operations
  updateField: <K extends keyof NormalizedFormState>(field: K, value: NormalizedFormState[K]) => void
  setModes: (modes: string[]) => void
  setTags: (tags: string[]) => void
  resetForm: () => void

  // Validation
  validation: ValidationResult
  isValid: boolean

  // Dirty tracking
  isDirty: boolean
  dirtyItemIds: Set<string>
  dirtyCount: number

  // Undo/Redo
  undo: () => void
  redo: () => void
  canUndo: boolean
  canRedo: boolean

  // Pending changes (for other skills)
  storeCurrentChanges: () => void

  // Save operations
  saveCurrentSkill: () => Promise<SaveResult>
  saveAllChanges: () => Promise<SaveResult>
  discardCurrentChanges: () => void
  discardAllChanges: () => void

  // Delete
  deleteCurrentSkill: () => Promise<void>

  // Loading states
  isSaving: boolean
  isDeleting: boolean
  isLoadingContent: boolean
}

/**
 * Hook for managing prompt editing with centralized state.
 */
export function usePromptEditor({
  skills,
  selectedItemId,
  onSave,
  onDelete,
}: UsePromptEditorProps): UsePromptEditorReturn {
  // Local loading states
  const [isSaving, setIsSaving] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)
  const [isLoadingContent, setIsLoadingContent] = useState(false)

  // Track if we've fetched full skill data for the current selection
  const fetchedSkillRef = useRef<string | null>(null)

  // Get store actions - these are stable references
  const initializePrompt = useEditorStore((state) => state.initializePrompt)
  const storeUpdateField = useEditorStore((state) => state.updateField)
  const storeUndo = useEditorStore((state) => state.undo)
  const storeRedo = useEditorStore((state) => state.redo)
  const markAsSaved = useEditorStore((state) => state.markAsSaved)
  const discardChanges = useEditorStore((state) => state.discardChanges)
  const storeDiscardAll = useEditorStore((state) => state.discardAllChanges)
  const removePrompt = useEditorStore((state) => state.removePrompt)
  const hydrate = useEditorStore((state) => state.hydrate)
  const cleanupStalePrompts = useEditorStore((state) => state.cleanupStalePrompts)
  const isHydrated = useEditorStore((state) => state.isHydrated)

  // Get the prompt edit state for the current selection
  // This is a stable selector that won't cause infinite loops
  const promptEditState = useEditorStore((state) =>
    selectedItemId ? state.prompts.get(selectedItemId) : undefined
  )

  // Derive values from promptEditState - these are computed, not from selectors
  const formState = promptEditState?.current ?? null
  const originalState = promptEditState?.original ?? null

  const isDirty = useMemo(() => {
    if (!formState || !originalState) return false
    return !isFormStateEqual(originalState, formState)
  }, [formState, originalState])

  const canUndo = promptEditState ? promptEditState.undoStack.length > 0 : false
  const canRedo = promptEditState ? promptEditState.redoStack.length > 0 : false

  const validation = useMemo((): ValidationResult => {
    if (!formState) return { valid: true, errors: {} }
    return validateNormalizedState(formState)
  }, [formState])

  // Compute dirty count directly from the store (stable primitive)
  // This selector returns a number, which is always compared by value
  const dirtyCount = useEditorStore((state) => {
    let count = 0
    for (const [, promptState] of state.prompts) {
      if (!isFormStateEqual(promptState.original, promptState.current)) {
        count++
      }
    }
    return count
  })

  // Cache the previous dirty IDs to avoid creating new Set references unnecessarily
  const prevDirtyIdsRef = useRef<Set<string>>(new Set())

  // Compute dirty item IDs, using dirtyCount as a trigger
  const dirtyItemIds = useMemo(() => {
    // Early exit if no dirty prompts
    if (dirtyCount === 0) {
      if (prevDirtyIdsRef.current.size === 0) {
        return prevDirtyIdsRef.current
      }
      const emptySet = new Set<string>()
      prevDirtyIdsRef.current = emptySet
      return emptySet
    }

    // Get dirty IDs from store
    const newDirtyIds = useEditorStore.getState().getDirtyPromptIds()

    // Check if content is the same (avoid new reference if unchanged)
    const prevIds = prevDirtyIdsRef.current
    if (
      newDirtyIds.size === prevIds.size &&
      [...newDirtyIds].every((id) => prevIds.has(id))
    ) {
      return prevIds
    }
    prevDirtyIdsRef.current = newDirtyIds
    return newDirtyIds
  }, [dirtyCount])

  // Get the current skill object
  const currentSkill = useMemo(() => {
    if (!selectedItemId) return null
    return skills.find((p) => p.id === selectedItemId) ?? null
  }, [skills, selectedItemId])

  // Hydrate store from localStorage on mount
  useEffect(() => {
    if (!isHydrated) {
      hydrate()
    }
  }, [isHydrated, hydrate])

  // Clean up stale prompts when skills list changes
  // Guard: Don't cleanup until skills have actually loaded (prevents race condition
  // where cleanup runs with empty skills list and wipes all persisted edits)
  useEffect(() => {
    if (!isHydrated || skills.length === 0) return
    const existingIds = new Set(skills.map((s) => s.id))
    cleanupStalePrompts(existingIds)
  }, [skills, isHydrated, cleanupStalePrompts])

  // Initialize prompt when selection changes
  useEffect(() => {
    if (!selectedItemId || !isHydrated) return

    const listSkill = skills.find((s) => s.id === selectedItemId)
    if (!listSkill) return

    // If list skill has content, initialize with it
    if (listSkill.content && listSkill.content.trim()) {
      initializePrompt(selectedItemId, listSkill)
      fetchedSkillRef.current = selectedItemId
      setIsLoadingContent(false)
      return
    }

    // Check if we already fetched full data for this skill
    if (fetchedSkillRef.current === selectedItemId) {
      return
    }

    // Fetch full skill with content from API
    setIsLoadingContent(true)
    skillService
      .getSkill(selectedItemId)
      .then((fullSkill) => {
        if (fullSkill) {
          initializePrompt(selectedItemId, fullSkill)
          fetchedSkillRef.current = selectedItemId
        } else {
          // Fallback to list skill if fetch fails
          initializePrompt(selectedItemId, listSkill)
          fetchedSkillRef.current = selectedItemId
        }
      })
      .catch(() => {
        // Fallback to list skill on error
        initializePrompt(selectedItemId, listSkill)
        fetchedSkillRef.current = selectedItemId
      })
      .finally(() => {
        setIsLoadingContent(false)
      })
  }, [selectedItemId, skills, isHydrated, initializePrompt])

  // Reset fetch ref when selection changes
  useEffect(() => {
    if (!selectedItemId) {
      fetchedSkillRef.current = null
    }
  }, [selectedItemId])

  // Update a single form field
  const updateField = useCallback(
    <K extends keyof NormalizedFormState>(field: K, value: NormalizedFormState[K]) => {
      if (!selectedItemId) return
      storeUpdateField(selectedItemId, field, value)
    },
    [selectedItemId, storeUpdateField]
  )

  // Update modes array
  const setModes = useCallback(
    (modes: string[]) => {
      if (!selectedItemId) return
      storeUpdateField(selectedItemId, 'modes', modes)
    },
    [selectedItemId, storeUpdateField]
  )

  // Update tags array
  const setTags = useCallback(
    (tags: string[]) => {
      if (!selectedItemId) return
      storeUpdateField(selectedItemId, 'tags', tags)
    },
    [selectedItemId, storeUpdateField]
  )

  // Reset form to original skill state
  const resetForm = useCallback(() => {
    if (!selectedItemId) return
    discardChanges(selectedItemId)
  }, [selectedItemId, discardChanges])

  // Undo
  const undo = useCallback(() => {
    if (!selectedItemId) return
    storeUndo(selectedItemId)
  }, [selectedItemId, storeUndo])

  // Redo
  const redo = useCallback(() => {
    if (!selectedItemId) return
    storeRedo(selectedItemId)
  }, [selectedItemId, storeRedo])

  // Store current changes (no-op with new store - changes are automatically stored)
  const storeCurrentChanges = useCallback(() => {
    // With the new store, changes are automatically persisted
    // This function exists for API compatibility
  }, [])

  // Get movePromptState action
  const movePromptState = useEditorStore((state) => state.movePromptState)

  // Save current skill only
  const saveCurrentSkill = useCallback(async (): Promise<SaveResult> => {
    if (!currentSkill || !isDirty || !validation.valid || !formState) {
      return { success: true, savedCount: 0, failedCount: 0, errors: [], newId: undefined }
    }

    setIsSaving(true)
    try {
      const updates = new Map<string, UpdateSkillRequest>()
      updates.set(currentSkill.id, normalizedStateToUpdateRequest(formState, originalState?.file))
      const results = await onSave(updates)

      // If save succeeded, update the original in the store
      const result = results.get(currentSkill.id)
      if (result && !(result instanceof Error)) {
        // Check if ID changed (rename operation)
        if (result.id !== currentSkill.id) {
          // Migrate editor state to new ID
          movePromptState(currentSkill.id, result.id)
          markAsSaved(result.id, result)
          return { success: true, savedCount: 1, failedCount: 0, errors: [], newId: result.id }
        }
        markAsSaved(currentSkill.id, result)
        return { success: true, savedCount: 1, failedCount: 0, errors: [], newId: undefined }
      } else if (result instanceof Error) {
        return {
          success: false,
          savedCount: 0,
          failedCount: 1,
          errors: [{ id: currentSkill.id, name: currentSkill.name, message: result.message }],
          newId: undefined,
        }
      }
      return { success: true, savedCount: 1, failedCount: 0, errors: [], newId: undefined }
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unknown error'
      return {
        success: false,
        savedCount: 0,
        failedCount: 1,
        errors: [{ id: currentSkill.id, name: currentSkill.name, message }],
        newId: undefined,
      }
    } finally {
      setIsSaving(false)
    }
  }, [currentSkill, isDirty, validation.valid, formState, originalState, onSave, markAsSaved, movePromptState])

  // Save all pending changes
  const saveAllChanges = useCallback(async (): Promise<SaveResult> => {
    const updates = new Map<string, UpdateSkillRequest>()

    // Get all dirty prompts from the store
    const store = useEditorStore.getState()
    const dirtyIds = store.getDirtyPromptIds()

    // Build a map of id to name for error reporting
    const idToName = new Map<string, string>()
    for (const skill of skills) {
      idToName.set(skill.id, skill.name)
    }

    for (const id of dirtyIds) {
      const state = store.getFormState(id)
      const original = store.getOriginalState(id)
      const storeValidation = store.getValidation(id)
      if (state && storeValidation.valid) {
        updates.set(id, normalizedStateToUpdateRequest(state, original?.file))
      }
    }

    if (updates.size === 0) {
      return { success: true, savedCount: 0, failedCount: 0, errors: [], newId: undefined }
    }

    setIsSaving(true)
    try {
      const results = await onSave(updates)

      // Track successes and failures
      let savedCount = 0
      const errors: Array<{ id: string; name: string; message: string }> = []

      // Mark successfully saved items
      for (const [id, result] of results) {
        if (!(result instanceof Error)) {
          // Check if ID changed (rename operation)
          if (result.id !== id) {
            movePromptState(id, result.id)
            markAsSaved(result.id, result)
          } else {
            markAsSaved(id, result)
          }
          savedCount++
        } else {
          errors.push({
            id,
            name: idToName.get(id) || 'Unknown skill',
            message: result.message,
          })
        }
      }

      return {
        success: errors.length === 0,
        savedCount,
        failedCount: errors.length,
        errors,
        newId: undefined, // Multiple skills saved, no single new ID
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unknown error'
      return {
        success: false,
        savedCount: 0,
        failedCount: updates.size,
        errors: [{ id: '', name: 'All skills', message }],
        newId: undefined,
      }
    } finally {
      setIsSaving(false)
    }
  }, [skills, onSave, markAsSaved, movePromptState])

  // Discard current skill changes
  const discardCurrentChanges = useCallback(() => {
    if (!selectedItemId) return
    discardChanges(selectedItemId)
  }, [selectedItemId, discardChanges])

  // Discard all changes
  const discardAllChanges = useCallback(() => {
    storeDiscardAll()
  }, [storeDiscardAll])

  // Delete current skill
  const deleteCurrentSkill = useCallback(async () => {
    if (!currentSkill) return

    setIsDeleting(true)
    try {
      await onDelete(currentSkill.id)
      removePrompt(currentSkill.id)
    } finally {
      setIsDeleting(false)
    }
  }, [currentSkill, onDelete, removePrompt])

  // Return empty state if no form state available
  const effectiveFormState = formState ?? createEmptyNormalizedState()
  const originalContent = originalState?.content ?? null

  return {
    // Current editor state
    currentSkill,
    formState: effectiveFormState,
    originalContent,

    // Form operations
    updateField,
    setModes,
    setTags,
    resetForm,

    // Validation
    validation,
    isValid: validation.valid,

    // Dirty tracking
    isDirty,
    dirtyItemIds,
    dirtyCount,

    // Undo/Redo
    undo,
    redo,
    canUndo,
    canRedo,

    // Pending changes
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
    isLoadingContent,
  }
}
