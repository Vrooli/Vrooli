/**
 * useTeamEditorStore - Zustand store for team editor state.
 *
 * Manages:
 * - Selected member ID for detail panel
 * - Member documents (responsibilities, heartbeat instructions)
 * - Dirty tracking for save indicators
 * - localStorage persistence for unsaved changes
 */

import { create } from 'zustand'
import {
  PERSISTENCE_DEBOUNCE_MS,
  ENTITY_PERSISTENCE_VERSION,
  ENTITY_STORAGE_KEYS,
} from '@/types/entityEditorStore'

// ============================================================================
// Types
// ============================================================================

interface DirtyState {
  responsibilities: boolean
  heartbeatInstructions: boolean
  schedule: boolean
}

/**
 * Member documents (responsibilities and heartbeat instructions).
 */
export interface MemberDocs {
  /** RESPONSIBILITIES.md content */
  responsibilities: string
  /** HEARTBEAT.md content */
  heartbeatInstructions: string
}

interface TeamEditorState {
  /** Currently selected member ID for the detail panel */
  selectedMemberId: string | null

  /** Member documents cache: agentId -> docs */
  memberDocs: Map<string, MemberDocs>

  /** Dirty tracking per member: agentId -> dirty fields */
  dirtyState: Map<string, DirtyState>

  /** Loading state for documents */
  isLoadingDocs: boolean

  /** Error message if document loading failed */
  docsError: string | null

  /** Whether the store has been hydrated from localStorage */
  isHydrated: boolean
}

interface TeamEditorActions {
  /** Select a member for editing */
  setSelectedMemberId: (id: string | null) => void

  /** Set member documents */
  setMemberDocs: (agentId: string, docs: MemberDocs) => void

  /** Update responsibilities for a member */
  updateResponsibilities: (agentId: string, content: string) => void

  /** Update heartbeat instructions for a member */
  updateHeartbeatInstructions: (agentId: string, content: string) => void

  /** Mark a field as dirty/clean */
  setDirty: (agentId: string, field: keyof DirtyState, isDirty: boolean) => void

  /** Mark all fields as clean for a member */
  clearDirty: (agentId: string) => void

  /** Set loading state for docs */
  setLoadingDocs: (loading: boolean) => void

  /** Set docs error */
  setDocsError: (error: string | null) => void

  /** Clear member from cache (e.g., when removed from team) */
  clearMember: (agentId: string) => void

  /** Reset entire store */
  reset: () => void

  /** Hydrate the store from localStorage on mount */
  hydrate: () => void
}

interface TeamEditorSelectors {
  /** Get documents for a specific member */
  getMemberDocs: (agentId: string) => MemberDocs | undefined

  /** Check if a member has dirty changes */
  isDirty: (agentId: string) => boolean

  /** Get dirty state for a member */
  getDirtyState: (agentId: string) => DirtyState | undefined

  /** Get count of members with dirty changes */
  getDirtyCount: () => number

  /** Get IDs of members with dirty changes */
  getDirtyMemberIds: () => Set<string>
}

type TeamEditorStore = TeamEditorState & TeamEditorActions & TeamEditorSelectors

// ============================================================================
// Initial State
// ============================================================================

const initialState: TeamEditorState = {
  selectedMemberId: null,
  memberDocs: new Map(),
  dirtyState: new Map(),
  isLoadingDocs: false,
  docsError: null,
  isHydrated: false,
}

// ============================================================================
// Persistence
// ============================================================================

const STORAGE_KEY = ENTITY_STORAGE_KEYS.teams

interface PersistedTeamState {
  version: number
  memberDocs: Record<string, MemberDocs>
  dirtyState: Record<string, DirtyState>
  lastSavedAt: number
}

/**
 * Load persisted state from localStorage.
 */
function loadPersistedState(): PersistedTeamState | null {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (!stored) return null

    const parsed = JSON.parse(stored) as PersistedTeamState
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

function persistState(memberDocs: Map<string, MemberDocs>, dirtyState: Map<string, DirtyState>): void {
  if (persistTimeout) {
    clearTimeout(persistTimeout)
  }

  persistTimeout = setTimeout(() => {
    try {
      // Only persist if there are dirty members
      const hasDirty = Array.from(dirtyState.values()).some(
        (d) => d.responsibilities || d.heartbeatInstructions || d.schedule
      )

      if (!hasDirty) {
        localStorage.removeItem(STORAGE_KEY)
        return
      }

      // Convert Maps to objects for JSON serialization
      const docsObj: Record<string, MemberDocs> = {}
      for (const [id, docs] of memberDocs) {
        docsObj[id] = docs
      }

      const dirtyObj: Record<string, DirtyState> = {}
      for (const [id, dirty] of dirtyState) {
        dirtyObj[id] = dirty
      }

      const persisted: PersistedTeamState = {
        version: ENTITY_PERSISTENCE_VERSION,
        memberDocs: docsObj,
        dirtyState: dirtyObj,
        lastSavedAt: Date.now(),
      }

      localStorage.setItem(STORAGE_KEY, JSON.stringify(persisted))
    } catch (e) {
      console.error('Failed to persist team editor state:', e)
    }
  }, PERSISTENCE_DEBOUNCE_MS)
}

// ============================================================================
// Store
// ============================================================================

export const useTeamEditorStore = create<TeamEditorStore>((set, get) => ({
  // State
  ...initialState,

  // Actions
  setSelectedMemberId: (id) => {
    set({ selectedMemberId: id, docsError: null })
  },

  setMemberDocs: (agentId, docs) => {
    const { memberDocs, dirtyState } = get()
    const newDocs = new Map(memberDocs)
    newDocs.set(agentId, docs)
    set({ memberDocs: newDocs })
    persistState(newDocs, dirtyState)
  },

  updateResponsibilities: (agentId, content) => {
    const { memberDocs, dirtyState } = get()
    const existing = memberDocs.get(agentId) ?? {
      responsibilities: '',
      heartbeatInstructions: '',
    }

    const newDocs = new Map(memberDocs)
    newDocs.set(agentId, { ...existing, responsibilities: content })

    const newDirtyState = new Map(dirtyState)
    const existingDirty = dirtyState.get(agentId) ?? {
      responsibilities: false,
      heartbeatInstructions: false,
      schedule: false,
    }
    newDirtyState.set(agentId, { ...existingDirty, responsibilities: true })

    set({ memberDocs: newDocs, dirtyState: newDirtyState })
    persistState(newDocs, newDirtyState)
  },

  updateHeartbeatInstructions: (agentId, content) => {
    const { memberDocs, dirtyState } = get()
    const existing = memberDocs.get(agentId) ?? {
      responsibilities: '',
      heartbeatInstructions: '',
    }

    const newDocs = new Map(memberDocs)
    newDocs.set(agentId, { ...existing, heartbeatInstructions: content })

    const newDirtyState = new Map(dirtyState)
    const existingDirty = dirtyState.get(agentId) ?? {
      responsibilities: false,
      heartbeatInstructions: false,
      schedule: false,
    }
    newDirtyState.set(agentId, { ...existingDirty, heartbeatInstructions: true })

    set({ memberDocs: newDocs, dirtyState: newDirtyState })
    persistState(newDocs, newDirtyState)
  },

  setDirty: (agentId, field, isDirty) => {
    const { memberDocs, dirtyState } = get()
    const newDirtyState = new Map(dirtyState)
    const existing = dirtyState.get(agentId) ?? {
      responsibilities: false,
      heartbeatInstructions: false,
      schedule: false,
    }
    newDirtyState.set(agentId, { ...existing, [field]: isDirty })
    set({ dirtyState: newDirtyState })
    persistState(memberDocs, newDirtyState)
  },

  clearDirty: (agentId) => {
    const { memberDocs, dirtyState } = get()
    const newDirtyState = new Map(dirtyState)
    newDirtyState.set(agentId, {
      responsibilities: false,
      heartbeatInstructions: false,
      schedule: false,
    })
    set({ dirtyState: newDirtyState })
    persistState(memberDocs, newDirtyState)
  },

  setLoadingDocs: (loading) => {
    set({ isLoadingDocs: loading })
  },

  setDocsError: (error) => {
    set({ docsError: error })
  },

  clearMember: (agentId) => {
    const { memberDocs, dirtyState, selectedMemberId } = get()

    const newDocs = new Map(memberDocs)
    newDocs.delete(agentId)

    const newDirtyState = new Map(dirtyState)
    newDirtyState.delete(agentId)

    set({
      memberDocs: newDocs,
      dirtyState: newDirtyState,
      selectedMemberId: selectedMemberId === agentId ? null : selectedMemberId,
    })
    persistState(newDocs, newDirtyState)
  },

  reset: () => {
    set({ ...initialState, isHydrated: true })
    localStorage.removeItem(STORAGE_KEY)
  },

  hydrate: () => {
    const persisted = loadPersistedState()
    if (!persisted) {
      set({ isHydrated: true })
      return
    }

    // Convert objects back to Maps
    const memberDocs = new Map<string, MemberDocs>()
    for (const [id, docs] of Object.entries(persisted.memberDocs)) {
      memberDocs.set(id, docs)
    }

    const dirtyState = new Map<string, DirtyState>()
    for (const [id, dirty] of Object.entries(persisted.dirtyState)) {
      dirtyState.set(id, dirty)
    }

    set({ memberDocs, dirtyState, isHydrated: true })
  },

  // Selectors
  getMemberDocs: (agentId) => {
    return get().memberDocs.get(agentId)
  },

  isDirty: (agentId) => {
    const dirty = get().dirtyState.get(agentId)
    if (!dirty) return false
    return dirty.responsibilities || dirty.heartbeatInstructions || dirty.schedule
  },

  getDirtyState: (agentId) => {
    return get().dirtyState.get(agentId)
  },

  /** Get count of members with dirty changes */
  getDirtyCount: () => {
    const { dirtyState } = get()
    let count = 0
    for (const dirty of dirtyState.values()) {
      if (dirty.responsibilities || dirty.heartbeatInstructions || dirty.schedule) {
        count++
      }
    }
    return count
  },

  /** Get IDs of members with dirty changes */
  getDirtyMemberIds: () => {
    const { dirtyState } = get()
    const ids = new Set<string>()
    for (const [id, dirty] of dirtyState) {
      if (dirty.responsibilities || dirty.heartbeatInstructions || dirty.schedule) {
        ids.add(id)
      }
    }
    return ids
  },
}))
