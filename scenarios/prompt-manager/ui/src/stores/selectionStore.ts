/**
 * Zustand store for centralized selection state.
 *
 * This store synchronizes selection between:
 * - The sidebar tree (single selection for editing)
 * - The 3D skill tree (multi-selection for combining)
 * - Agent selection (single selection for editing)
 */

import { create } from 'zustand'
import { usePerformanceStore } from '@/stores/performanceStore'

const VIEW_STORAGE_KEY = 'pm.viewMode'

function loadGraphViewActive(): boolean {
  try {
    return localStorage.getItem(VIEW_STORAGE_KEY) === 'graph'
  } catch {
    return false
  }
}

interface SelectionStore {
  // Single selection for editing (sidebar tree)
  selectedSkillId: string | null

  // Multi-selection for combining (3D skill tree)
  selectedSkillIds: string[]

  // Agent selection for editing
  selectedAgentId: string | null

  // Team selection for editing
  selectedTeamId: string | null

  // Run selection for detail view
  selectedRunId: string | null

  // Topic selection for editing
  selectedTopicId: string | null

  // Action selection for editing
  selectedActionId: string | null

  // Topic selection wizard active
  topicWizardActive: boolean

  // Graph view toggle (when no skill selected)
  graphViewActive: boolean

  // Actions
  setSelectedSkillId: (id: string | null) => void
  toggleSkillSelection: (id: string) => void
  addToSelection: (id: string) => void
  removeFromSelection: (id: string) => void
  setSelectedSkillIds: (ids: string[]) => void
  clearSelection: () => void
  clearAllSelection: () => void
  setSelectedAgentId: (id: string | null) => void
  setSelectedTeamId: (id: string | null) => void
  setSelectedRunId: (id: string | null) => void
  setSelectedTopicId: (id: string | null) => void
  setSelectedActionId: (id: string | null) => void
  setTopicWizardActive: (v: boolean) => void
  setGraphViewActive: (v: boolean) => void
}

export const useSelectionStore = create<SelectionStore>((set, get) => ({
  selectedSkillId: null,
  selectedSkillIds: [],
  selectedAgentId: null,
  selectedTeamId: null,
  selectedRunId: null,
  selectedTopicId: null,
  selectedActionId: null,
  topicWizardActive: false,
  graphViewActive: loadGraphViewActive(),

  setSelectedSkillId: (id) => {
    set({
      selectedSkillId: id,
      // When selecting a single skill for editing, also update multi-selection
      // This ensures the 3D tree highlights the selected skill
      selectedSkillIds: id ? [id] : [],
      // Clear agent, team, run, and topic selection when selecting a skill
      selectedAgentId: null,
      selectedTeamId: null,
      selectedRunId: null,
      selectedTopicId: null,
      selectedActionId: null,
      topicWizardActive: false,
    })
    usePerformanceStore.getState().recordInteractionStoreWrite()
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
    usePerformanceStore.getState().recordInteractionStoreWrite()
  },

  addToSelection: (id) => {
    const { selectedSkillIds } = get()
    if (!selectedSkillIds.includes(id)) {
      const newIds = [...selectedSkillIds, id]
      set({
        selectedSkillIds: newIds,
        selectedSkillId: newIds.length === 1 ? id : get().selectedSkillId,
      })
      usePerformanceStore.getState().recordInteractionStoreWrite()
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
    usePerformanceStore.getState().recordInteractionStoreWrite()
  },

  setSelectedSkillIds: (ids) => {
    set({
      selectedSkillIds: ids,
      // If there's exactly one selection, also set it as the single selected
      selectedSkillId: ids.length === 1 ? ids[0] : get().selectedSkillId,
    })
    usePerformanceStore.getState().recordInteractionStoreWrite()
  },

  clearSelection: () => {
    set({ selectedSkillIds: [] })
    usePerformanceStore.getState().recordInteractionStoreWrite()
  },

  clearAllSelection: () => {
    set({
      selectedSkillId: null,
      selectedSkillIds: [],
      selectedAgentId: null,
      selectedTeamId: null,
      selectedRunId: null,
      selectedTopicId: null,
      selectedActionId: null,
      topicWizardActive: false,
    })
    usePerformanceStore.getState().recordInteractionStoreWrite()
  },

  setSelectedAgentId: (id) => {
    set({
      selectedAgentId: id,
      // Clear skill, team, run, and topic selection when selecting an agent
      selectedSkillId: null,
      selectedSkillIds: [],
      selectedTeamId: null,
      selectedRunId: null,
      selectedTopicId: null,
      selectedActionId: null,
      topicWizardActive: false,
    })
    usePerformanceStore.getState().recordInteractionStoreWrite()
  },

  setSelectedTeamId: (id) => {
    set({
      selectedTeamId: id,
      // Clear skill, agent, run, and topic selection when selecting a team
      selectedSkillId: null,
      selectedSkillIds: [],
      selectedAgentId: null,
      selectedRunId: null,
      selectedTopicId: null,
      selectedActionId: null,
      topicWizardActive: false,
    })
    usePerformanceStore.getState().recordInteractionStoreWrite()
  },

  setSelectedRunId: (id) => {
    set({
      selectedRunId: id,
      // Clear skill, agent, team, and topic selection when selecting a run
      selectedSkillId: null,
      selectedSkillIds: [],
      selectedAgentId: null,
      selectedTeamId: null,
      selectedTopicId: null,
      selectedActionId: null,
      topicWizardActive: false,
    })
    usePerformanceStore.getState().recordInteractionStoreWrite()
  },

  setSelectedTopicId: (id) => {
    set({
      selectedTopicId: id,
      // Clear skill, agent, team, and run selection when selecting a topic
      selectedSkillId: null,
      selectedSkillIds: [],
      selectedAgentId: null,
      selectedTeamId: null,
      selectedRunId: null,
      selectedActionId: null,
      topicWizardActive: false,
    })
    usePerformanceStore.getState().recordInteractionStoreWrite()
  },

  setSelectedActionId: (id) => {
    set({
      selectedActionId: id,
      selectedSkillId: null,
      selectedSkillIds: [],
      selectedAgentId: null,
      selectedTeamId: null,
      selectedRunId: null,
      selectedTopicId: null,
      topicWizardActive: false,
    })
    usePerformanceStore.getState().recordInteractionStoreWrite()
  },

  setTopicWizardActive: (v) => {
    set({
      topicWizardActive: v,
      // Clear all entity selections when activating wizard
      ...(v ? {
        selectedSkillId: null,
        selectedSkillIds: [],
        selectedAgentId: null,
        selectedTeamId: null,
        selectedRunId: null,
        selectedTopicId: null,
        selectedActionId: null,
      } : {}),
    })
    usePerformanceStore.getState().recordInteractionStoreWrite()
  },

  setGraphViewActive: (v) => {
    set({ graphViewActive: v })
    usePerformanceStore.getState().recordInteractionStoreWrite()
    try {
      localStorage.setItem(VIEW_STORAGE_KEY, v ? 'graph' : 'world')
    } catch { /* ignore quota errors */ }
  },
}))
