/**
 * useAgentEditor - Hook for managing agent editing with the centralized store.
 *
 * This hook bridges the Zustand agentEditorStore with React components.
 * It provides the same API pattern as usePromptEditor for consistency.
 *
 * Key features:
 * - Normalized form state
 * - Per-agent undo/redo history
 * - localStorage persistence (survives page refresh)
 * - Multi-agent editing (switch agents without losing changes)
 */

import { useState, useCallback, useMemo, useEffect, useRef } from 'react'
import type { Agent, UpdateAgentRequest } from '@/types/agent'
import type { ValidationResult, SaveResult } from '@/types/entityEditorStore'
import {
  useAgentEditorStore,
  type NormalizedAgentFormState,
  createEmptyAgentState,
  isAgentFormStateEqual,
  validateAgentState,
} from '@/stores/agentEditorStore'

interface UseAgentEditorProps {
  agents: Agent[]
  selectedAgentId: string | null
  onSave: (id: string, updates: UpdateAgentRequest) => Promise<Agent>
  onDelete: (id: string) => Promise<void>
}

interface UseAgentEditorReturn {
  // Current editor state
  currentAgent: Agent | null
  formState: NormalizedAgentFormState
  originalState: NormalizedAgentFormState | null

  // Form operations
  updateField: <K extends keyof NormalizedAgentFormState>(field: K, value: NormalizedAgentFormState[K]) => void
  updateFields: (updates: Partial<NormalizedAgentFormState>) => void
  renameFileOrderPath: (fromPath: string, toPath: string, isDir: boolean) => void
  resetForm: () => void

  // Validation
  validation: ValidationResult
  isValid: boolean

  // Dirty tracking
  isDirty: boolean
  dirtyAgentIds: Set<string>
  dirtyCount: number

  // Undo/Redo
  undo: () => void
  redo: () => void
  canUndo: boolean
  canRedo: boolean

  // Save operations
  saveCurrentAgent: () => Promise<SaveResult>
  saveAllChanges: () => Promise<SaveResult>
  discardCurrentChanges: () => void
  discardAllChanges: () => void

  // Delete
  deleteCurrentAgent: () => Promise<void>

  // Loading states
  isSaving: boolean
  isDeleting: boolean
}

/**
 * Hook for managing agent editing with centralized state.
 */
export function useAgentEditor({
  agents,
  selectedAgentId,
  onSave,
  onDelete,
}: UseAgentEditorProps): UseAgentEditorReturn {
  // Local loading states
  const [isSaving, setIsSaving] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)

  // Track if we've fetched full agent data for the current selection
  const fetchedAgentRef = useRef<string | null>(null)

  // Get store actions
  const initializeAgent = useAgentEditorStore((state) => state.initializeAgent)
  const storeUpdateField = useAgentEditorStore((state) => state.updateField)
  const storeUpdateFields = useAgentEditorStore((state) => state.updateFields)
  const storeRenameFileOrderPath = useAgentEditorStore((state) => state.renameFileOrderPath)
  const storeUndo = useAgentEditorStore((state) => state.undo)
  const storeRedo = useAgentEditorStore((state) => state.redo)
  const markAsSaved = useAgentEditorStore((state) => state.markAsSaved)
  const discardChanges = useAgentEditorStore((state) => state.discardChanges)
  const storeDiscardAll = useAgentEditorStore((state) => state.discardAllChanges)
  const removeAgent = useAgentEditorStore((state) => state.removeAgent)
  const hydrate = useAgentEditorStore((state) => state.hydrate)
  const cleanupStaleAgents = useAgentEditorStore((state) => state.cleanupStaleAgents)
  const isHydrated = useAgentEditorStore((state) => state.isHydrated)

  // Get the agent edit state for the current selection
  const agentEditState = useAgentEditorStore((state) =>
    selectedAgentId ? state.agents.get(selectedAgentId) : undefined
  )

  // Derive values from agentEditState
  const formState = agentEditState?.current ?? null
  const originalState = agentEditState?.original ?? null

  const isDirty = useMemo(() => {
    if (!formState || !originalState) return false
    return !isAgentFormStateEqual(originalState, formState)
  }, [formState, originalState])

  const canUndo = agentEditState ? agentEditState.undoStack.length > 0 : false
  const canRedo = agentEditState ? agentEditState.redoStack.length > 0 : false

  const validation = useMemo((): ValidationResult => {
    if (!formState) return { valid: true, errors: {} }
    return validateAgentState(formState)
  }, [formState])

  // Compute dirty count directly from the store (stable primitive)
  // This selector returns a number, which is always compared by value
  const dirtyCount = useAgentEditorStore((state) => {
    let count = 0
    for (const [, agentState] of state.agents) {
      if (!isAgentFormStateEqual(agentState.original, agentState.current)) {
        count++
      }
    }
    return count
  })

  // Compute dirty agent IDs, using dirtyCount as a trigger
  // Cache the previous result to avoid creating new Set references unnecessarily
  const prevDirtyIdsRef = useRef<Set<string>>(new Set())

  const dirtyAgentIds = useMemo(() => {
    // Early exit if no dirty agents
    if (dirtyCount === 0) {
      if (prevDirtyIdsRef.current.size === 0) {
        return prevDirtyIdsRef.current
      }
      const emptySet = new Set<string>()
      prevDirtyIdsRef.current = emptySet
      return emptySet
    }

    // Get dirty IDs from store
    const newDirtyIds = useAgentEditorStore.getState().getDirtyAgentIds()

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

  // Get the current agent object
  const currentAgent = useMemo(() => {
    if (!selectedAgentId) return null
    return agents.find((a) => a.id === selectedAgentId) ?? null
  }, [agents, selectedAgentId])

  // Hydrate store from localStorage on mount
  useEffect(() => {
    if (!isHydrated) {
      hydrate()
    }
  }, [isHydrated, hydrate])

  // Clean up stale agents when agents list changes
  useEffect(() => {
    if (!isHydrated || agents.length === 0) return
    const existingIds = new Set(agents.map((a) => a.id))
    cleanupStaleAgents(existingIds)
  }, [agents, isHydrated, cleanupStaleAgents])

  // Initialize agent when selection changes
  useEffect(() => {
    if (!selectedAgentId || !isHydrated) return

    const listAgent = agents.find((a) => a.id === selectedAgentId)
    if (!listAgent) return

    // Initialize with the list agent data
    initializeAgent(selectedAgentId, listAgent)
    fetchedAgentRef.current = selectedAgentId
  }, [selectedAgentId, agents, isHydrated, initializeAgent])

  // Reset fetch ref when selection changes
  useEffect(() => {
    if (!selectedAgentId) {
      fetchedAgentRef.current = null
    }
  }, [selectedAgentId])

  // Update a single form field
  const updateField = useCallback(
    <K extends keyof NormalizedAgentFormState>(field: K, value: NormalizedAgentFormState[K]) => {
      if (!selectedAgentId) return
      storeUpdateField(selectedAgentId, field, value)
    },
    [selectedAgentId, storeUpdateField]
  )

  // Update multiple fields at once
  const updateFields = useCallback(
    (updates: Partial<NormalizedAgentFormState>) => {
      if (!selectedAgentId) return
      storeUpdateFields(selectedAgentId, updates)
    },
    [selectedAgentId, storeUpdateFields]
  )

  const renameFileOrderPath = useCallback(
    (fromPath: string, toPath: string, isDir: boolean) => {
      if (!selectedAgentId) return
      storeRenameFileOrderPath(selectedAgentId, fromPath, toPath, isDir)
    },
    [selectedAgentId, storeRenameFileOrderPath]
  )

  // Reset form to original state
  const resetForm = useCallback(() => {
    if (!selectedAgentId) return
    discardChanges(selectedAgentId)
  }, [selectedAgentId, discardChanges])

  // Undo
  const undo = useCallback(() => {
    if (!selectedAgentId) return
    storeUndo(selectedAgentId)
  }, [selectedAgentId, storeUndo])

  // Redo
  const redo = useCallback(() => {
    if (!selectedAgentId) return
    storeRedo(selectedAgentId)
  }, [selectedAgentId, storeRedo])

  // Convert form state to update request
  const formStateToUpdateRequest = useCallback((state: NormalizedAgentFormState): UpdateAgentRequest => {
    return {
      displayName: state.displayName,
      description: state.description,
      status: state.status,
      appearance: state.appearance,
      tags: state.tags,
      fileOrder: state.fileOrder,
    }
  }, [])

  // Save current agent only
  const saveCurrentAgent = useCallback(async (): Promise<SaveResult> => {
    if (!currentAgent || !isDirty || !validation.valid || !formState) {
      return { success: true, savedCount: 0, failedCount: 0, errors: [] }
    }

    setIsSaving(true)
    try {
      const updateRequest = formStateToUpdateRequest(formState)
      const savedAgent = await onSave(currentAgent.id, updateRequest)
      markAsSaved(currentAgent.id, savedAgent)
      return { success: true, savedCount: 1, failedCount: 0, errors: [] }
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unknown error'
      return {
        success: false,
        savedCount: 0,
        failedCount: 1,
        errors: [{ id: currentAgent.id, name: currentAgent.displayName, message }],
      }
    } finally {
      setIsSaving(false)
    }
  }, [currentAgent, isDirty, validation.valid, formState, formStateToUpdateRequest, onSave, markAsSaved])

  // Save all pending changes
  const saveAllChanges = useCallback(async (): Promise<SaveResult> => {
    const store = useAgentEditorStore.getState()
    const dirtyIds = store.getDirtyAgentIds()

    if (dirtyIds.size === 0) {
      return { success: true, savedCount: 0, failedCount: 0, errors: [] }
    }

    setIsSaving(true)
    try {
      let savedCount = 0
      const errors: Array<{ id: string; name: string; message: string }> = []

      for (const id of dirtyIds) {
        const state = store.getFormState(id)
        const storeValidation = store.getValidation(id)
        const agent = agents.find((a) => a.id === id)

        if (!state || !storeValidation.valid || !agent) continue

        try {
          const updateRequest = formStateToUpdateRequest(state)
          const savedAgent = await onSave(id, updateRequest)
          markAsSaved(id, savedAgent)
          savedCount++
        } catch (error) {
          const message = error instanceof Error ? error.message : 'Unknown error'
          errors.push({ id, name: agent.displayName, message })
        }
      }

      return {
        success: errors.length === 0,
        savedCount,
        failedCount: errors.length,
        errors,
      }
    } finally {
      setIsSaving(false)
    }
  }, [agents, formStateToUpdateRequest, onSave, markAsSaved])

  // Discard current agent changes
  const discardCurrentChanges = useCallback(() => {
    if (!selectedAgentId) return
    discardChanges(selectedAgentId)
  }, [selectedAgentId, discardChanges])

  // Discard all changes
  const discardAllChanges = useCallback(() => {
    storeDiscardAll()
  }, [storeDiscardAll])

  // Delete current agent
  const deleteCurrentAgent = useCallback(async () => {
    if (!currentAgent) return

    setIsDeleting(true)
    try {
      await onDelete(currentAgent.id)
      removeAgent(currentAgent.id)
    } finally {
      setIsDeleting(false)
    }
  }, [currentAgent, onDelete, removeAgent])

  // Return empty state if no form state available
  const effectiveFormState = formState ?? createEmptyAgentState()

  return {
    // Current editor state
    currentAgent,
    formState: effectiveFormState,
    originalState,

    // Form operations
    updateField,
    updateFields,
    renameFileOrderPath,
    resetForm,

    // Validation
    validation,
    isValid: validation.valid,

    // Dirty tracking
    isDirty,
    dirtyAgentIds,
    dirtyCount,

    // Undo/Redo
    undo,
    redo,
    canUndo,
    canRedo,

    // Save operations
    saveCurrentAgent,
    saveAllChanges,
    discardCurrentChanges,
    discardAllChanges,

    // Delete
    deleteCurrentAgent,

    // Loading states
    isSaving,
    isDeleting,
  }
}
