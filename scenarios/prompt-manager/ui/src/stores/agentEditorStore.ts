/**
 * Agent Editor Store - Centralized form state management for agents.
 *
 * Mirrors the skill editor store pattern with:
 * - Normalized form state
 * - Per-agent undo/redo history stacks
 * - Debounced localStorage persistence
 * - Staleness cleanup for old edits
 */

import { create } from 'zustand'
import type { Agent, AgentAppearance } from '@/types/agent'
import { DEFAULT_AGENT_COLORS } from '@/types/agent'
import type { EntityEditState, EntitySnapshot, PersistedEntityState, ValidationResult } from '@/types/entityEditorStore'
import {
  MAX_HISTORY_SIZE,
  PERSISTENCE_DEBOUNCE_MS,
  STALENESS_THRESHOLD_MS,
  ENTITY_PERSISTENCE_VERSION,
  ENTITY_STORAGE_KEYS,
} from '@/types/entityEditorStore'

// ============================================================================
// Normalized Agent Form State
// ============================================================================

/**
 * Normalized form state for agents.
 * All fields have consistent types for reliable dirty detection.
 */
export interface NormalizedAgentFormState {
  displayName: string
  description: string
  status: 'active' | 'inactive' | 'suspended'
  appearance: AgentAppearance
  skills: string[]
  tags: string[]
}

/**
 * Normalize an Agent from the API to NormalizedAgentFormState.
 */
export function normalizeAgent(agent: Agent): NormalizedAgentFormState {
  return {
    displayName: agent.displayName,
    description: agent.description ?? '',
    status: agent.status,
    appearance: agent.appearance ?? { ...DEFAULT_AGENT_COLORS },
    skills: [...agent.skills],
    tags: [...agent.tags],
  }
}

/**
 * Create an empty normalized form state for new agents.
 */
export function createEmptyAgentState(): NormalizedAgentFormState {
  return {
    displayName: '',
    description: '',
    status: 'active',
    appearance: { ...DEFAULT_AGENT_COLORS },
    skills: [],
    tags: [],
  }
}

/**
 * Deep equality check for agent form state.
 */
export function isAgentFormStateEqual(
  a: NormalizedAgentFormState,
  b: NormalizedAgentFormState
): boolean {
  if (a.displayName !== b.displayName) return false
  if (a.description !== b.description) return false
  if (a.status !== b.status) return false

  // Compare appearance
  if (a.appearance.body !== b.appearance.body) return false
  if (a.appearance.head !== b.appearance.head) return false
  if (a.appearance.accent !== b.appearance.accent) return false

  // Compare arrays
  if (!arraysEqual(a.skills, b.skills)) return false
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
 * Validate agent form state.
 */
export function validateAgentState(state: NormalizedAgentFormState): ValidationResult {
  const errors: Record<string, string> = {}

  if (!state.displayName.trim()) {
    errors.displayName = 'Display name is required'
  } else if (state.displayName.length > 100) {
    errors.displayName = 'Display name must be 100 characters or less'
  }

  if (state.description.length > 500) {
    errors.description = 'Description must be 500 characters or less'
  }

  return {
    valid: Object.keys(errors).length === 0,
    errors,
  }
}

// ============================================================================
// Persistence Functions
// ============================================================================

type AgentEditState = EntityEditState<NormalizedAgentFormState>
type AgentSnapshot = EntitySnapshot<NormalizedAgentFormState>
type PersistedAgentState = PersistedEntityState<NormalizedAgentFormState>

const STORAGE_KEY = ENTITY_STORAGE_KEYS.agents

/**
 * Load persisted state from localStorage.
 */
function loadPersistedState(): PersistedAgentState | null {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (!stored) return null

    const parsed = JSON.parse(stored) as PersistedAgentState
    if (parsed.version !== ENTITY_PERSISTENCE_VERSION) {
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

function persistState(agents: Map<string, AgentEditState>): void {
  if (persistTimeout) {
    clearTimeout(persistTimeout)
  }

  persistTimeout = setTimeout(() => {
    try {
      // Only persist dirty agents
      const dirtyAgents: Record<string, AgentEditState> = {}
      for (const [id, state] of agents) {
        if (!isAgentFormStateEqual(state.original, state.current)) {
          dirtyAgents[id] = state
        }
      }

      // If no dirty agents, clear storage
      if (Object.keys(dirtyAgents).length === 0) {
        localStorage.removeItem(STORAGE_KEY)
        return
      }

      const persisted: PersistedAgentState = {
        version: ENTITY_PERSISTENCE_VERSION,
        entities: dirtyAgents,
        lastSavedAt: Date.now(),
      }

      localStorage.setItem(STORAGE_KEY, JSON.stringify(persisted))
    } catch (e) {
      console.error('Failed to persist agent editor state:', e)
    }
  }, PERSISTENCE_DEBOUNCE_MS)
}

// ============================================================================
// Store Types
// ============================================================================

interface AgentEditorStoreState {
  /** Map of agent ID to edit state */
  agents: Map<string, AgentEditState>

  /** Whether the store has been hydrated from localStorage */
  isHydrated: boolean
}

interface AgentEditorStoreActions {
  /** Initialize an agent for editing */
  initializeAgent: (agentId: string, agent: Agent) => void

  /** Update a field in the current agent's form state */
  updateField: <K extends keyof NormalizedAgentFormState>(
    agentId: string,
    field: K,
    value: NormalizedAgentFormState[K]
  ) => void

  /** Update multiple fields at once (single undo entry) */
  updateFields: (agentId: string, updates: Partial<NormalizedAgentFormState>) => void

  /** Undo the last change for an agent */
  undo: (agentId: string) => void

  /** Redo a previously undone change */
  redo: (agentId: string) => void

  /** Mark an agent as saved (update original to match current) */
  markAsSaved: (agentId: string, savedAgent: Agent) => void

  /** Discard changes for a specific agent */
  discardChanges: (agentId: string) => void

  /** Discard all changes across all agents */
  discardAllChanges: () => void

  /** Remove an agent from the store */
  removeAgent: (agentId: string) => void

  /** Hydrate the store from localStorage on mount */
  hydrate: () => void

  /** Clean up stale agents */
  cleanupStaleAgents: (existingAgentIds: Set<string>) => void
}

interface AgentEditorStoreSelectors {
  /** Check if a specific agent has unsaved changes */
  isDirty: (agentId: string) => boolean

  /** Get all dirty agent IDs */
  getDirtyAgentIds: () => Set<string>

  /** Get count of dirty agents */
  getDirtyCount: () => number

  /** Check if undo is available for an agent */
  canUndo: (agentId: string) => boolean

  /** Check if redo is available for an agent */
  canRedo: (agentId: string) => boolean

  /** Get the current form state for an agent */
  getFormState: (agentId: string) => NormalizedAgentFormState | null

  /** Get the original state for an agent */
  getOriginalState: (agentId: string) => NormalizedAgentFormState | null

  /** Get validation for an agent */
  getValidation: (agentId: string) => ValidationResult
}

type AgentEditorStore = AgentEditorStoreState & AgentEditorStoreActions & AgentEditorStoreSelectors

// ============================================================================
// Store Implementation
// ============================================================================

export const useAgentEditorStore = create<AgentEditorStore>((set, get) => ({
  // State
  agents: new Map(),
  isHydrated: false,

  // Actions
  initializeAgent: (agentId, agent) => {
    const { agents } = get()

    // Check if we already have this agent in the store
    const existing = agents.get(agentId)
    if (existing) {
      // Update original if the agent changed (e.g., saved elsewhere)
      const newOriginal = normalizeAgent(agent)
      if (!isAgentFormStateEqual(existing.original, newOriginal)) {
        const newAgents = new Map(agents)
        newAgents.set(agentId, {
          ...existing,
          original: newOriginal,
        })
        set({ agents: newAgents })
        persistState(newAgents)
      }
      return
    }

    // Create new edit state
    const normalized = normalizeAgent(agent)
    const newState: AgentEditState = {
      original: normalized,
      current: { ...normalized, appearance: { ...normalized.appearance } },
      undoStack: [],
      redoStack: [],
      editStartedAt: Date.now(),
      lastModifiedAt: Date.now(),
    }

    const newAgents = new Map(agents)
    newAgents.set(agentId, newState)
    set({ agents: newAgents })
    persistState(newAgents)
  },

  updateField: (agentId, field, value) => {
    const { agents } = get()
    const state = agents.get(agentId)
    if (!state) return

    // Push current state to undo stack
    const snapshot: AgentSnapshot = {
      state: { ...state.current, appearance: { ...state.current.appearance } },
      timestamp: Date.now(),
    }

    const newUndoStack = [...state.undoStack, snapshot].slice(-MAX_HISTORY_SIZE)

    // Update the field
    const newCurrent = {
      ...state.current,
      [field]: value,
    }

    const newState: AgentEditState = {
      ...state,
      current: newCurrent,
      undoStack: newUndoStack,
      redoStack: [], // Clear redo stack on new edit
      lastModifiedAt: Date.now(),
    }

    const newAgents = new Map(agents)
    newAgents.set(agentId, newState)
    set({ agents: newAgents })
    persistState(newAgents)
  },

  updateFields: (agentId, updates) => {
    const { agents } = get()
    const state = agents.get(agentId)
    if (!state) return

    // Push current state to undo stack
    const snapshot: AgentSnapshot = {
      state: { ...state.current, appearance: { ...state.current.appearance } },
      timestamp: Date.now(),
    }

    const newUndoStack = [...state.undoStack, snapshot].slice(-MAX_HISTORY_SIZE)

    // Apply all updates
    const newCurrent = {
      ...state.current,
      ...updates,
    }

    const newState: AgentEditState = {
      ...state,
      current: newCurrent,
      undoStack: newUndoStack,
      redoStack: [],
      lastModifiedAt: Date.now(),
    }

    const newAgents = new Map(agents)
    newAgents.set(agentId, newState)
    set({ agents: newAgents })
    persistState(newAgents)
  },

  undo: (agentId) => {
    const { agents } = get()
    const state = agents.get(agentId)
    if (!state || state.undoStack.length === 0) return

    const newUndoStack = [...state.undoStack]
    const previousSnapshot = newUndoStack.pop()
    if (!previousSnapshot) return

    const redoSnapshot: AgentSnapshot = {
      state: { ...state.current, appearance: { ...state.current.appearance } },
      timestamp: Date.now(),
    }
    const newRedoStack = [...state.redoStack, redoSnapshot]

    const newState: AgentEditState = {
      ...state,
      current: { ...previousSnapshot.state },
      undoStack: newUndoStack,
      redoStack: newRedoStack,
      lastModifiedAt: Date.now(),
    }

    const newAgents = new Map(agents)
    newAgents.set(agentId, newState)
    set({ agents: newAgents })
    persistState(newAgents)
  },

  redo: (agentId) => {
    const { agents } = get()
    const state = agents.get(agentId)
    if (!state || state.redoStack.length === 0) return

    const newRedoStack = [...state.redoStack]
    const nextSnapshot = newRedoStack.pop()
    if (!nextSnapshot) return

    const undoSnapshot: AgentSnapshot = {
      state: { ...state.current, appearance: { ...state.current.appearance } },
      timestamp: Date.now(),
    }
    const newUndoStack = [...state.undoStack, undoSnapshot].slice(-MAX_HISTORY_SIZE)

    const newState: AgentEditState = {
      ...state,
      current: { ...nextSnapshot.state },
      undoStack: newUndoStack,
      redoStack: newRedoStack,
      lastModifiedAt: Date.now(),
    }

    const newAgents = new Map(agents)
    newAgents.set(agentId, newState)
    set({ agents: newAgents })
    persistState(newAgents)
  },

  markAsSaved: (agentId, savedAgent) => {
    const { agents } = get()
    const state = agents.get(agentId)
    if (!state) return

    const newOriginal = normalizeAgent(savedAgent)
    const newState: AgentEditState = {
      ...state,
      original: newOriginal,
      current: { ...newOriginal, appearance: { ...newOriginal.appearance } },
      undoStack: [],
      redoStack: [],
      lastModifiedAt: Date.now(),
    }

    const newAgents = new Map(agents)
    newAgents.set(agentId, newState)
    set({ agents: newAgents })
    persistState(newAgents)
  },

  discardChanges: (agentId) => {
    const { agents } = get()
    const state = agents.get(agentId)
    if (!state) return

    const newState: AgentEditState = {
      ...state,
      current: { ...state.original, appearance: { ...state.original.appearance } },
      undoStack: [],
      redoStack: [],
      lastModifiedAt: Date.now(),
    }

    const newAgents = new Map(agents)
    newAgents.set(agentId, newState)
    set({ agents: newAgents })
    persistState(newAgents)
  },

  discardAllChanges: () => {
    const { agents } = get()
    const newAgents = new Map<string, AgentEditState>()

    for (const [id, state] of agents) {
      newAgents.set(id, {
        ...state,
        current: { ...state.original, appearance: { ...state.original.appearance } },
        undoStack: [],
        redoStack: [],
        lastModifiedAt: Date.now(),
      })
    }

    set({ agents: newAgents })
    persistState(newAgents)
  },

  removeAgent: (agentId) => {
    const { agents } = get()
    const newAgents = new Map(agents)
    newAgents.delete(agentId)
    set({ agents: newAgents })
    persistState(newAgents)
  },

  hydrate: () => {
    const persisted = loadPersistedState()
    if (!persisted) {
      set({ isHydrated: true })
      return
    }

    const agents = new Map<string, AgentEditState>()
    for (const [id, state] of Object.entries(persisted.entities)) {
      agents.set(id, state)
    }

    set({ agents, isHydrated: true })
  },

  cleanupStaleAgents: (existingAgentIds) => {
    const { agents } = get()
    const now = Date.now()
    const newAgents = new Map<string, AgentEditState>()

    for (const [id, state] of agents) {
      const agentExists = existingAgentIds.has(id)
      const isDirty = !isAgentFormStateEqual(state.original, state.current)
      const isStale = now - state.lastModifiedAt > STALENESS_THRESHOLD_MS

      // Always keep dirty agents to prevent data loss
      // Also keep non-stale agents for existing entities
      if (isDirty || (agentExists && !isStale)) {
        newAgents.set(id, state)
      }
    }

    if (newAgents.size !== agents.size) {
      set({ agents: newAgents })
      persistState(newAgents)
    }
  },

  // Selectors
  isDirty: (agentId) => {
    const state = get().agents.get(agentId)
    if (!state) return false
    return !isAgentFormStateEqual(state.original, state.current)
  },

  getDirtyAgentIds: () => {
    const { agents } = get()
    const dirty = new Set<string>()
    for (const [id, state] of agents) {
      if (!isAgentFormStateEqual(state.original, state.current)) {
        dirty.add(id)
      }
    }
    return dirty
  },

  getDirtyCount: () => {
    return get().getDirtyAgentIds().size
  },

  canUndo: (agentId) => {
    const state = get().agents.get(agentId)
    return state ? state.undoStack.length > 0 : false
  },

  canRedo: (agentId) => {
    const state = get().agents.get(agentId)
    return state ? state.redoStack.length > 0 : false
  },

  getFormState: (agentId) => {
    const state = get().agents.get(agentId)
    return state ? state.current : null
  },

  getOriginalState: (agentId) => {
    const state = get().agents.get(agentId)
    return state ? state.original : null
  },

  getValidation: (agentId) => {
    const state = get().agents.get(agentId)
    if (!state) {
      return { valid: true, errors: {} }
    }
    return validateAgentState(state.current)
  },
}))
