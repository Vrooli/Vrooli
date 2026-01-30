/**
 * Zustand store for combine mode state.
 *
 * This store manages the combine mode that allows users to select multiple
 * skills and copy their combined output to clipboard in different formats.
 */

import { create } from 'zustand'

export type CombineFormat = 'xml' | 'markdown' | 'json'

interface CombineStore {
  // Mode state
  isActive: boolean

  // Selection state
  selectedSkillIds: Set<string>

  // Format state
  format: CombineFormat

  // Loading state
  isCopying: boolean

  // Actions
  enterCombineMode: () => void
  exitCombineMode: () => void
  toggleSkillSelection: (skillId: string) => void
  toggleMultipleSkills: (skillIds: string[], select: boolean) => void
  setFormat: (format: CombineFormat) => void
  setIsCopying: (copying: boolean) => void
  clearSelection: () => void
}

export const useCombineStore = create<CombineStore>((set) => ({
  isActive: false,
  selectedSkillIds: new Set(),
  format: 'xml',
  isCopying: false,

  enterCombineMode: () => {
    set({
      isActive: true,
      selectedSkillIds: new Set(),
      isCopying: false,
    })
  },

  exitCombineMode: () => {
    set({
      isActive: false,
      selectedSkillIds: new Set(),
      isCopying: false,
    })
  },

  toggleSkillSelection: (skillId) => {
    set((state) => {
      const next = new Set(state.selectedSkillIds)
      if (next.has(skillId)) {
        next.delete(skillId)
      } else {
        next.add(skillId)
      }
      return { selectedSkillIds: next }
    })
  },

  toggleMultipleSkills: (skillIds, select) => {
    set((state) => {
      const next = new Set(state.selectedSkillIds)
      for (const id of skillIds) {
        if (select) {
          next.add(id)
        } else {
          next.delete(id)
        }
      }
      return { selectedSkillIds: next }
    })
  },

  setFormat: (format) => {
    set({ format })
  },

  setIsCopying: (copying) => {
    set({ isCopying: copying })
  },

  clearSelection: () => {
    set({ selectedSkillIds: new Set() })
  },
}))
