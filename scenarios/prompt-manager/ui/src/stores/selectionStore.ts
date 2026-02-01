/**
 * Zustand store for centralized selection state.
 *
 * This store synchronizes selection between:
 * - The sidebar tree (single selection for editing)
 * - The 3D skill tree (multi-selection for combining)
 * - Agent selection (single selection for editing)
 */

import { create } from 'zustand'

interface SelectionStore {
  // Single selection for editing (sidebar tree)
  selectedSkillId: string | null

  // Multi-selection for combining (3D skill tree)
  selectedSkillIds: string[]

  // Agent selection for editing
  selectedAgentId: string | null

  // Actions
  setSelectedSkillId: (id: string | null) => void
  toggleSkillSelection: (id: string) => void
  addToSelection: (id: string) => void
  removeFromSelection: (id: string) => void
  setSelectedSkillIds: (ids: string[]) => void
  clearSelection: () => void
  clearAllSelection: () => void
  setSelectedAgentId: (id: string | null) => void
}

export const useSelectionStore = create<SelectionStore>((set, get) => ({
  selectedSkillId: null,
  selectedSkillIds: [],
  selectedAgentId: null,

  setSelectedSkillId: (id) => {
    set({
      selectedSkillId: id,
      // When selecting a single skill for editing, also update multi-selection
      // This ensures the 3D tree highlights the selected skill
      selectedSkillIds: id ? [id] : [],
      // Clear agent selection when selecting a skill
      selectedAgentId: null,
    })
  },

  toggleSkillSelection: (id) => {
    const { selectedSkillIds } = get()
    if (selectedSkillIds.includes(id)) {
      const newIds = selectedSkillIds.filter((sid) => sid !== id)
      set({
        selectedSkillIds: newIds,
        // Update single selection if we just toggled off the selected item
        selectedSkillId: newIds.length === 1 ? newIds[0] : get().selectedSkillId,
      })
    } else {
      const newIds = [...selectedSkillIds, id]
      set({
        selectedSkillIds: newIds,
        // If this is the first selection, also set as single selected
        selectedSkillId: newIds.length === 1 ? id : get().selectedSkillId,
      })
    }
  },

  addToSelection: (id) => {
    const { selectedSkillIds } = get()
    if (!selectedSkillIds.includes(id)) {
      const newIds = [...selectedSkillIds, id]
      set({
        selectedSkillIds: newIds,
        selectedSkillId: newIds.length === 1 ? id : get().selectedSkillId,
      })
    }
  },

  removeFromSelection: (id) => {
    const { selectedSkillIds } = get()
    const newIds = selectedSkillIds.filter((sid) => sid !== id)
    set({
      selectedSkillIds: newIds,
      // Update single selection if we just removed the selected item
      selectedSkillId: get().selectedSkillId === id ? null : get().selectedSkillId,
    })
  },

  setSelectedSkillIds: (ids) => {
    set({
      selectedSkillIds: ids,
      // If there's exactly one selection, also set it as the single selected
      selectedSkillId: ids.length === 1 ? ids[0] : get().selectedSkillId,
    })
  },

  clearSelection: () => {
    set({ selectedSkillIds: [] })
  },

  clearAllSelection: () => {
    set({
      selectedSkillId: null,
      selectedSkillIds: [],
      selectedAgentId: null,
    })
  },

  setSelectedAgentId: (id) => {
    set({
      selectedAgentId: id,
      // Clear skill selection when selecting an agent
      selectedSkillId: null,
      selectedSkillIds: [],
    })
  },
}))
