/**
 * Zustand store for skill selection mode state.
 *
 * This store manages the skill selection mode that allows users to assign
 * skills to members. It's shared between:
 * - WorldCanvas (triggers entering skill selection mode)
 * - SkillTreeSidebar (shows checkboxes and handles selection)
 */

import { create } from 'zustand'
import type { Member } from '@/types/member'

interface SkillSelectionStore {
  // Mode state
  isActive: boolean
  currentMemberId: string | null
  currentMember: Member | null

  // Selection state
  selectedSkillIds: Set<string>

  // Callback for saving
  onSave: ((skillIds: string[]) => Promise<void>) | null

  // Actions
  enterSkillSelectionMode: (
    member: Member,
    currentSkills: string[],
    onSave: (skillIds: string[]) => Promise<void>
  ) => void
  exitSkillSelectionMode: () => void
  toggleSkillSelection: (skillId: string) => void
  toggleMultipleSkills: (skillIds: string[], select: boolean) => void
  saveAndExit: () => Promise<void>
}

export const useSkillSelectionStore = create<SkillSelectionStore>((set, get) => ({
  isActive: false,
  currentMemberId: null,
  currentMember: null,
  selectedSkillIds: new Set(),
  onSave: null,

  enterSkillSelectionMode: (member, currentSkills, onSave) => {
    set({
      isActive: true,
      currentMemberId: member.id,
      currentMember: member,
      selectedSkillIds: new Set(currentSkills),
      onSave,
    })
  },

  exitSkillSelectionMode: () => {
    set({
      isActive: false,
      currentMemberId: null,
      currentMember: null,
      selectedSkillIds: new Set(),
      onSave: null,
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

  saveAndExit: async () => {
    const { onSave, selectedSkillIds } = get()
    if (onSave) {
      await onSave(Array.from(selectedSkillIds))
    }
    get().exitSkillSelectionMode()
  },
}))
