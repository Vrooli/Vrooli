/**
 * Editor Store - Centralized form state management with Zustand.
 *
 * Features:
 * - Normalized form state (tags as string[], not comma-separated)
 * - Per-prompt undo/redo history stacks
 * - Debounced localStorage persistence
 * - Staleness cleanup for old edits
 */

import { create } from 'zustand'
import type { Skill } from '@/types'
import type {
  NormalizedFormState,
  PromptEditState,
  FormSnapshot,
  PersistedEditorState,
  ValidationResult,
} from '@/types/editorStore'
import {
  MAX_HISTORY_SIZE,
  PERSISTENCE_DEBOUNCE_MS,
  STALENESS_THRESHOLD_MS,
  PERSISTENCE_VERSION,
  STORAGE_KEY,
} from '@/types/editorStore'

// ============================================================================
// Normalization Functions
// ============================================================================

/**
 * Normalize a Skill from the API to NormalizedFormState.
 * Ensures all fields have consistent types.
 */
export function normalizeSkill(skill: Skill): NormalizedFormState {
  return {
    file: skill.file,
    name: skill.name,
    description: skill.description,
    content: skill.content,
    modes: [...skill.modes],
    tags: [...skill.tags], // Already an array from API
    icon: skill.icon ?? '',
    draft: skill.draft,
    folder: skill.folder,
  }
}

/**
 * Create an empty normalized form state.
 */
export function createEmptyNormalizedState(): NormalizedFormState {
  return {
    file: '',
    name: '',
    description: '',
    content: '',
    modes: [],
    tags: [],
    icon: '',
    draft: true,
    folder: 'local',
  }
}

/**
 * Deep equality check for form state.
 */
export function isFormStateEqual(
  a: NormalizedFormState,
  b: NormalizedFormState
): boolean {
  if (a.file !== b.file) return false
  if (a.name !== b.name) return false
  if (a.description !== b.description) return false
  if (a.content !== b.content) return false
  if (a.draft !== b.draft) return false
  if (a.folder !== b.folder) return false
  if (a.icon !== b.icon) return false

  // Compare arrays
  if (!arraysEqual(a.modes, b.modes)) return false
  if (!arraysEqual(a.tags, b.tags)) return false

  return true
}

/**
 * Check if two string arrays are equal (order-sensitive).
 */
function arraysEqual(a: string[], b: string[]): boolean {
  if (a.length !== b.length) return false
  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) return false
  }
  return true
}

/**
 * Validate form state.
 */
export function validateNormalizedState(
  state: NormalizedFormState
): ValidationResult {
  const errors: Record<string, string> = {}

  if (!state.name.trim()) {
    errors.name = 'Name is required'
  } else if (state.name.length > 100) {
    errors.name = 'Name must be 100 characters or less'
  }

  if (state.description.length > 500) {
    errors.description = 'Description must be 500 characters or less'
  }

  if (!state.content.trim()) {
    errors.content = 'Content is required'
  }

  return {
    valid: Object.keys(errors).length === 0,
    errors,
  }
}

// ============================================================================
// Persistence Functions
// ============================================================================

/**
 * Load persisted state from localStorage.
 * Returns null if no state exists or if it's invalid.
 */
function loadPersistedState(): PersistedEditorState | null {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (!stored) return null

    const parsed = JSON.parse(stored) as PersistedEditorState
    if (parsed.version !== PERSISTENCE_VERSION) {
      // Future: handle migrations here
      return null
    }

    return parsed
  } catch {
    return null
  }
}

/**
 * Save state to localStorage (debounced).
 */
let persistTimeout: ReturnType<typeof setTimeout> | null = null

function persistState(prompts: Map<string, PromptEditState>): void {
  if (persistTimeout) {
    clearTimeout(persistTimeout)
  }

  persistTimeout = setTimeout(() => {
    try {
      // Only persist dirty prompts (where current !== original)
      const dirtyPrompts: Record<string, PromptEditState> = {}
      for (const [id, state] of prompts) {
        if (!isFormStateEqual(state.original, state.current)) {
          dirtyPrompts[id] = state
        }
      }

      // If no dirty prompts, clear storage
      if (Object.keys(dirtyPrompts).length === 0) {
        localStorage.removeItem(STORAGE_KEY)
        return
      }

      const persisted: PersistedEditorState = {
        version: PERSISTENCE_VERSION,
        prompts: dirtyPrompts,
        lastSavedAt: Date.now(),
      }

      localStorage.setItem(STORAGE_KEY, JSON.stringify(persisted))
    } catch (e) {
      console.error('Failed to persist editor state:', e)
    }
  }, PERSISTENCE_DEBOUNCE_MS)
}

// ============================================================================
// Store Types
// ============================================================================

interface EditorStoreState {
  /** Map of prompt ID to edit state */
  prompts: Map<string, PromptEditState>

  /** Whether the store has been hydrated from localStorage */
  isHydrated: boolean
}

interface EditorStoreActions {
  /**
   * Initialize a prompt for editing.
   * If there's persisted state, restore it; otherwise use the skill data.
   */
  initializePrompt: (promptId: string, skill: Skill) => void

  /**
   * Update a field in the current prompt's form state.
   * Pushes to undo stack.
   */
  updateField: <K extends keyof NormalizedFormState>(
    promptId: string,
    field: K,
    value: NormalizedFormState[K]
  ) => void

  /**
   * Update multiple fields at once (single undo entry).
   */
  updateFields: (
    promptId: string,
    updates: Partial<NormalizedFormState>
  ) => void

  /**
   * Undo the last change for a prompt.
   */
  undo: (promptId: string) => void

  /**
   * Redo a previously undone change.
   */
  redo: (promptId: string) => void

  /**
   * Mark a prompt as saved (update original to match current).
   */
  markAsSaved: (promptId: string, savedSkill: Skill) => void

  /**
   * Discard changes for a specific prompt.
   */
  discardChanges: (promptId: string) => void

  /**
   * Discard all changes across all prompts.
   */
  discardAllChanges: () => void

  /**
   * Remove a prompt from the store (e.g., after deletion).
   */
  removePrompt: (promptId: string) => void

  /**
   * Hydrate the store from localStorage on mount.
   */
  hydrate: () => void

  /**
   * Clean up stale prompts (older than threshold).
   */
  cleanupStalePrompts: (existingSkillIds: Set<string>) => void
}

// Selectors return type
interface EditorStoreSelectors {
  /** Check if a specific prompt has unsaved changes */
  isDirty: (promptId: string) => boolean

  /** Get all dirty prompt IDs */
  getDirtyPromptIds: () => Set<string>

  /** Get count of dirty prompts */
  getDirtyCount: () => number

  /** Check if undo is available for a prompt */
  canUndo: (promptId: string) => boolean

  /** Check if redo is available for a prompt */
  canRedo: (promptId: string) => boolean

  /** Get the current form state for a prompt */
  getFormState: (promptId: string) => NormalizedFormState | null

  /** Get the original state for a prompt */
  getOriginalState: (promptId: string) => NormalizedFormState | null

  /** Get validation for a prompt */
  getValidation: (promptId: string) => ValidationResult
}

type EditorStore = EditorStoreState & EditorStoreActions & EditorStoreSelectors

// ============================================================================
// Store Implementation
// ============================================================================

export const useEditorStore = create<EditorStore>((set, get) => ({
  // State
  prompts: new Map(),
  isHydrated: false,

  // Actions
  initializePrompt: (promptId, skill) => {
    const { prompts } = get()

    // Check if we already have this prompt in the store
    const existing = prompts.get(promptId)
    if (existing) {
      // Update original if the skill changed (e.g., saved elsewhere)
      const newOriginal = normalizeSkill(skill)
      if (!isFormStateEqual(existing.original, newOriginal)) {
        const newPrompts = new Map(prompts)
        newPrompts.set(promptId, {
          ...existing,
          original: newOriginal,
        })
        set({ prompts: newPrompts })
        persistState(newPrompts)
      }
      return
    }

    // Create new edit state
    const normalized = normalizeSkill(skill)
    const newState: PromptEditState = {
      original: normalized,
      current: { ...normalized },
      undoStack: [],
      redoStack: [],
      editStartedAt: Date.now(),
      lastModifiedAt: Date.now(),
    }

    const newPrompts = new Map(prompts)
    newPrompts.set(promptId, newState)
    set({ prompts: newPrompts })
    persistState(newPrompts)
  },

  updateField: (promptId, field, value) => {
    const { prompts } = get()
    const state = prompts.get(promptId)
    if (!state) return

    // Push current state to undo stack
    const snapshot: FormSnapshot = {
      state: { ...state.current },
      timestamp: Date.now(),
    }

    const newUndoStack = [...state.undoStack, snapshot].slice(-MAX_HISTORY_SIZE)

    // Update the field
    const newCurrent = {
      ...state.current,
      [field]: value,
    }

    const newState: PromptEditState = {
      ...state,
      current: newCurrent,
      undoStack: newUndoStack,
      redoStack: [], // Clear redo stack on new edit
      lastModifiedAt: Date.now(),
    }

    const newPrompts = new Map(prompts)
    newPrompts.set(promptId, newState)
    set({ prompts: newPrompts })
    persistState(newPrompts)
  },

  updateFields: (promptId, updates) => {
    const { prompts } = get()
    const state = prompts.get(promptId)
    if (!state) return

    // Push current state to undo stack
    const snapshot: FormSnapshot = {
      state: { ...state.current },
      timestamp: Date.now(),
    }

    const newUndoStack = [...state.undoStack, snapshot].slice(-MAX_HISTORY_SIZE)

    // Apply all updates
    const newCurrent = {
      ...state.current,
      ...updates,
    }

    const newState: PromptEditState = {
      ...state,
      current: newCurrent,
      undoStack: newUndoStack,
      redoStack: [], // Clear redo stack on new edit
      lastModifiedAt: Date.now(),
    }

    const newPrompts = new Map(prompts)
    newPrompts.set(promptId, newState)
    set({ prompts: newPrompts })
    persistState(newPrompts)
  },

  undo: (promptId) => {
    const { prompts } = get()
    const state = prompts.get(promptId)
    if (!state || state.undoStack.length === 0) return

    // Pop from undo stack
    const newUndoStack = [...state.undoStack]
    const previousSnapshot = newUndoStack.pop()
    if (!previousSnapshot) return // Shouldn't happen due to length check above

    // Push current to redo stack
    const redoSnapshot: FormSnapshot = {
      state: { ...state.current },
      timestamp: Date.now(),
    }
    const newRedoStack = [...state.redoStack, redoSnapshot]

    const newState: PromptEditState = {
      ...state,
      current: { ...previousSnapshot.state },
      undoStack: newUndoStack,
      redoStack: newRedoStack,
      lastModifiedAt: Date.now(),
    }

    const newPrompts = new Map(prompts)
    newPrompts.set(promptId, newState)
    set({ prompts: newPrompts })
    persistState(newPrompts)
  },

  redo: (promptId) => {
    const { prompts } = get()
    const state = prompts.get(promptId)
    if (!state || state.redoStack.length === 0) return

    // Pop from redo stack
    const newRedoStack = [...state.redoStack]
    const nextSnapshot = newRedoStack.pop()
    if (!nextSnapshot) return // Shouldn't happen due to length check above

    // Push current to undo stack
    const undoSnapshot: FormSnapshot = {
      state: { ...state.current },
      timestamp: Date.now(),
    }
    const newUndoStack = [...state.undoStack, undoSnapshot].slice(
      -MAX_HISTORY_SIZE
    )

    const newState: PromptEditState = {
      ...state,
      current: { ...nextSnapshot.state },
      undoStack: newUndoStack,
      redoStack: newRedoStack,
      lastModifiedAt: Date.now(),
    }

    const newPrompts = new Map(prompts)
    newPrompts.set(promptId, newState)
    set({ prompts: newPrompts })
    persistState(newPrompts)
  },

  markAsSaved: (promptId, savedSkill) => {
    const { prompts } = get()
    const state = prompts.get(promptId)
    if (!state) return

    const newOriginal = normalizeSkill(savedSkill)
    const newState: PromptEditState = {
      ...state,
      original: newOriginal,
      current: { ...newOriginal },
      undoStack: [],
      redoStack: [],
      lastModifiedAt: Date.now(),
    }

    const newPrompts = new Map(prompts)
    newPrompts.set(promptId, newState)
    set({ prompts: newPrompts })
    persistState(newPrompts)
  },

  discardChanges: (promptId) => {
    const { prompts } = get()
    const state = prompts.get(promptId)
    if (!state) return

    const newState: PromptEditState = {
      ...state,
      current: { ...state.original },
      undoStack: [],
      redoStack: [],
      lastModifiedAt: Date.now(),
    }

    const newPrompts = new Map(prompts)
    newPrompts.set(promptId, newState)
    set({ prompts: newPrompts })
    persistState(newPrompts)
  },

  discardAllChanges: () => {
    const { prompts } = get()
    const newPrompts = new Map<string, PromptEditState>()

    for (const [id, state] of prompts) {
      newPrompts.set(id, {
        ...state,
        current: { ...state.original },
        undoStack: [],
        redoStack: [],
        lastModifiedAt: Date.now(),
      })
    }

    set({ prompts: newPrompts })
    persistState(newPrompts)
  },

  removePrompt: (promptId) => {
    const { prompts } = get()
    const newPrompts = new Map(prompts)
    newPrompts.delete(promptId)
    set({ prompts: newPrompts })
    persistState(newPrompts)
  },

  hydrate: () => {
    const persisted = loadPersistedState()
    if (!persisted) {
      set({ isHydrated: true })
      return
    }

    const prompts = new Map<string, PromptEditState>()
    for (const [id, state] of Object.entries(persisted.prompts)) {
      prompts.set(id, state)
    }

    set({ prompts, isHydrated: true })
  },

  cleanupStalePrompts: (existingSkillIds) => {
    const { prompts } = get()
    const now = Date.now()
    const newPrompts = new Map<string, PromptEditState>()

    for (const [id, state] of prompts) {
      const skillExists = existingSkillIds.has(id)
      const isDirty = !isFormStateEqual(state.original, state.current)
      const isStale = now - state.lastModifiedAt > STALENESS_THRESHOLD_MS

      // Always keep dirty prompts to prevent data loss during race conditions
      // (e.g., when skills list hasn't fully loaded yet)
      // Also keep non-stale prompts for existing skills
      if (isDirty || (skillExists && !isStale)) {
        newPrompts.set(id, state)
      }
    }

    if (newPrompts.size !== prompts.size) {
      set({ prompts: newPrompts })
      persistState(newPrompts)
    }
  },

  // Selectors
  isDirty: (promptId) => {
    const state = get().prompts.get(promptId)
    if (!state) return false
    return !isFormStateEqual(state.original, state.current)
  },

  getDirtyPromptIds: () => {
    const { prompts } = get()
    const dirty = new Set<string>()
    for (const [id, state] of prompts) {
      if (!isFormStateEqual(state.original, state.current)) {
        dirty.add(id)
      }
    }
    return dirty
  },

  getDirtyCount: () => {
    return get().getDirtyPromptIds().size
  },

  canUndo: (promptId) => {
    const state = get().prompts.get(promptId)
    return state ? state.undoStack.length > 0 : false
  },

  canRedo: (promptId) => {
    const state = get().prompts.get(promptId)
    return state ? state.redoStack.length > 0 : false
  },

  getFormState: (promptId) => {
    const state = get().prompts.get(promptId)
    return state ? state.current : null
  },

  getOriginalState: (promptId) => {
    const state = get().prompts.get(promptId)
    return state ? state.original : null
  },

  getValidation: (promptId) => {
    const state = get().prompts.get(promptId)
    if (!state) {
      return { valid: true, errors: {} }
    }
    return validateNormalizedState(state.current)
  },
}))
