/**
 * Zustand store for combine/select mode state.
 *
 * This store manages two mutually exclusive selection modes:
 * - 'skill-combine': Multi-select skills for combined copy (original behavior)
 * - 'ai-select': Multi-select AI search results for copy across entity types
 */

import { create } from 'zustand'
import { getSavedFormat, saveFormat } from '@/lib/formatPreference'
import { loadPersistedSelection, savePersistedSelection, clearPersistedSelection } from '@/lib/selectionPersistence'

export type CombineFormat = 'xml' | 'markdown' | 'json' | 'cli'
export type CombineMode = 'skill-combine' | 'ai-select'
export type CombineEntityType = 'skills' | 'agents' | 'teams' | 'topics' | 'actions'

interface CombineStore {
  // Mode state
  isActive: boolean
  mode: CombineMode
  entityType: CombineEntityType

  // Selection state
  selectedIds: Set<string>

  // Format state
  format: CombineFormat

  // Loading state
  isCopying: boolean

  // Budget tracking (AI select / discover mode only)
  contentCharsMap: Map<string, number>
  budgetChars: number | null
  budgetStatus: string | null

  // Actions — skill combine mode
  enterCombineMode: () => void
  exitCombineMode: () => void

  // Actions — AI select mode
  enterAISelectMode: (entityType: CombineEntityType) => void
  setEntityType: (entityType: CombineEntityType) => void

  // Actions — shared
  toggleSelection: (id: string) => void
  toggleMultiple: (ids: string[], select: boolean) => void
  selectMultiple: (ids: string[], contentCharsEntries?: Array<[string, number]>) => void
  setFormat: (format: CombineFormat) => void
  setIsCopying: (copying: boolean) => void
  clearSelection: () => void
  setBudget: (chars: number | null, status: string | null) => void
  getSelectedContentChars: () => number
}

function getInitialState() {
  const defaults = {
    isActive: false,
    mode: 'skill-combine' as CombineMode,
    entityType: 'skills' as CombineEntityType,
    selectedIds: new Set<string>(),
    format: getSavedFormat() as CombineFormat,
    isCopying: false,
    contentCharsMap: new Map<string, number>(),
    budgetChars: null as number | null,
    budgetStatus: null as string | null,
  }

  // Restore selection from localStorage if available
  const persisted = loadPersistedSelection()
  if (persisted) {
    return {
      ...defaults,
      isActive: true,
      mode: persisted.mode,
      entityType: persisted.entityType,
      selectedIds: new Set(persisted.selectedIds),
    }
  }
  return defaults
}

const INITIAL_STATE = getInitialState()

export const useCombineStore = create<CombineStore>((set, get) => ({
  ...INITIAL_STATE,

  enterCombineMode: () => {
    set({
      isActive: true,
      mode: 'skill-combine',
      entityType: 'skills',
      selectedIds: new Set(),
      isCopying: false,
      contentCharsMap: new Map(),
      budgetChars: null,
      budgetStatus: null,
    })
  },

  exitCombineMode: () => {
    set({
      isActive: false,
      selectedIds: new Set(),
      isCopying: false,
      contentCharsMap: new Map(),
      budgetChars: null,
      budgetStatus: null,
    })
  },

  enterAISelectMode: (entityType) => {
    set({
      isActive: true,
      mode: 'ai-select',
      entityType,
      selectedIds: new Set(),
      isCopying: false,
      contentCharsMap: new Map(),
      budgetChars: null,
      budgetStatus: null,
    })
  },

  setEntityType: (entityType) => {
    set({ entityType, selectedIds: new Set(), contentCharsMap: new Map() })
  },

  toggleSelection: (id) => {
    set((state) => {
      const next = new Set(state.selectedIds)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return { selectedIds: next }
    })
  },

  toggleMultiple: (ids, select) => {
    set((state) => {
      const next = new Set(state.selectedIds)
      for (const id of ids) {
        if (select) {
          next.add(id)
        } else {
          next.delete(id)
        }
      }
      return { selectedIds: next }
    })
  },

  selectMultiple: (ids, contentCharsEntries) => {
    set((state) => {
      const next = new Set(state.selectedIds)
      const nextMap = new Map(state.contentCharsMap)
      for (const id of ids) {
        next.add(id)
      }
      if (contentCharsEntries) {
        for (const [id, chars] of contentCharsEntries) {
          nextMap.set(id, chars)
        }
      }
      return { selectedIds: next, contentCharsMap: nextMap }
    })
  },

  setFormat: (format) => {
    saveFormat(format)
    set({ format })
  },

  setIsCopying: (copying) => {
    set({ isCopying: copying })
  },

  clearSelection: () => {
    set({ selectedIds: new Set(), contentCharsMap: new Map() })
  },

  setBudget: (chars, status) => {
    set({ budgetChars: chars, budgetStatus: status })
  },

  getSelectedContentChars: () => {
    const { selectedIds, contentCharsMap } = get()
    let total = 0
    for (const id of selectedIds) {
      total += contentCharsMap.get(id) ?? 0
    }
    return total
  },
}))

// Persist selection state to localStorage on changes
useCombineStore.subscribe((state, prevState) => {
  if (!state.isActive) {
    if (prevState.isActive) {
      clearPersistedSelection()
    }
  } else if (
    state.isActive !== prevState.isActive ||
    state.selectedIds !== prevState.selectedIds ||
    state.entityType !== prevState.entityType
  ) {
    savePersistedSelection({
      isActive: state.isActive,
      mode: state.mode,
      entityType: state.entityType,
      selectedIds: Array.from(state.selectedIds),
    })
  }
})
