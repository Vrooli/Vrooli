/**
 * In-surface selection state for the world/graph experiences.
 *
 * Route identity is owned by react-router. This store only tracks transient
 * multi-selection used by the 3D world and combine workflows.
 */

import { create } from 'zustand'
import { usePerformanceStore } from '@/stores/performanceStore'

interface SelectionStore {
  selectedSkillIds: string[]
  toggleSkillSelection: (id: string) => void
  addToSelection: (id: string) => void
  removeFromSelection: (id: string) => void
  setSelectedSkillIds: (ids: string[]) => void
  clearSelection: () => void
  clearAllSelection: () => void
}

function recordSelectionWrite() {
  usePerformanceStore.getState().recordInteractionStoreWrite()
}

export const useSelectionStore = create<SelectionStore>((set, get) => ({
  selectedSkillIds: [],

  toggleSkillSelection: (id) => {
    const { selectedSkillIds } = get()
    const next = selectedSkillIds.includes(id)
      ? selectedSkillIds.filter((sid) => sid !== id)
      : [...selectedSkillIds, id]
    set({ selectedSkillIds: next })
    recordSelectionWrite()
  },

  addToSelection: (id) => {
    const { selectedSkillIds } = get()
    if (selectedSkillIds.includes(id)) return
    set({ selectedSkillIds: [...selectedSkillIds, id] })
    recordSelectionWrite()
  },

  removeFromSelection: (id) => {
    const { selectedSkillIds } = get()
    set({ selectedSkillIds: selectedSkillIds.filter((sid) => sid !== id) })
    recordSelectionWrite()
  },

  setSelectedSkillIds: (ids) => {
    set({ selectedSkillIds: ids })
    recordSelectionWrite()
  },

  clearSelection: () => {
    set({ selectedSkillIds: [] })
    recordSelectionWrite()
  },

  clearAllSelection: () => {
    set({ selectedSkillIds: [] })
    recordSelectionWrite()
  },
}))
