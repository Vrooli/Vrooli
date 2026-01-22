/**
 * Zustand store for centralized prompt selection state.
 *
 * This store synchronizes selection between:
 * - The sidebar tree (single selection for editing)
 * - The 3D skill tree (multi-selection for combining)
 */

import { create } from 'zustand'

interface SelectionStore {
  // Single selection for editing (sidebar tree)
  selectedPromptId: string | null

  // Multi-selection for combining (3D skill tree)
  selectedPromptIds: string[]

  // Actions
  setSelectedPromptId: (id: string | null) => void
  togglePromptSelection: (id: string) => void
  addToSelection: (id: string) => void
  removeFromSelection: (id: string) => void
  setSelectedPromptIds: (ids: string[]) => void
  clearSelection: () => void
  clearAllSelection: () => void
}

export const useSelectionStore = create<SelectionStore>((set, get) => ({
  selectedPromptId: null,
  selectedPromptIds: [],

  setSelectedPromptId: (id) => {
    set({
      selectedPromptId: id,
      // When selecting a single prompt for editing, also update multi-selection
      // This ensures the 3D tree highlights the selected prompt
      selectedPromptIds: id ? [id] : [],
    })
  },

  togglePromptSelection: (id) => {
    const { selectedPromptIds } = get()
    if (selectedPromptIds.includes(id)) {
      const newIds = selectedPromptIds.filter((sid) => sid !== id)
      set({
        selectedPromptIds: newIds,
        // Update single selection if we just toggled off the selected item
        selectedPromptId: newIds.length === 1 ? newIds[0] : get().selectedPromptId,
      })
    } else {
      const newIds = [...selectedPromptIds, id]
      set({
        selectedPromptIds: newIds,
        // If this is the first selection, also set as single selected
        selectedPromptId: newIds.length === 1 ? id : get().selectedPromptId,
      })
    }
  },

  addToSelection: (id) => {
    const { selectedPromptIds } = get()
    if (!selectedPromptIds.includes(id)) {
      const newIds = [...selectedPromptIds, id]
      set({
        selectedPromptIds: newIds,
        selectedPromptId: newIds.length === 1 ? id : get().selectedPromptId,
      })
    }
  },

  removeFromSelection: (id) => {
    const { selectedPromptIds } = get()
    const newIds = selectedPromptIds.filter((sid) => sid !== id)
    set({
      selectedPromptIds: newIds,
      // Update single selection if we just removed the selected item
      selectedPromptId: get().selectedPromptId === id ? null : get().selectedPromptId,
    })
  },

  setSelectedPromptIds: (ids) => {
    set({
      selectedPromptIds: ids,
      // If there's exactly one selection, also set it as the single selected
      selectedPromptId: ids.length === 1 ? ids[0] : get().selectedPromptId,
    })
  },

  clearSelection: () => {
    set({ selectedPromptIds: [] })
  },

  clearAllSelection: () => {
    set({
      selectedPromptId: null,
      selectedPromptIds: [],
    })
  },
}))
