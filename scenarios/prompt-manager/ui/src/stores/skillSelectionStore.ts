/**
 * Zustand store for skill selection mode state.
 *
 * This store manages the skill selection mode that allows users to assign
 * prompts as skills to avatars. It's shared between:
 * - WorldCanvas (triggers entering skill selection mode)
 * - PromptTreeSidebar (shows checkboxes and handles selection)
 */

import { create } from 'zustand'
import type { Avatar } from '@/types/avatar'

interface SkillSelectionStore {
  // Mode state
  isActive: boolean
  currentAvatarId: string | null
  currentAvatar: Avatar | null

  // Selection state
  selectedSkillIds: Set<string>

  // Callback for saving
  onSave: ((skillIds: string[]) => Promise<void>) | null

  // Actions
  enterSkillSelectionMode: (
    avatar: Avatar,
    currentSkills: string[],
    onSave: (skillIds: string[]) => Promise<void>
  ) => void
  exitSkillSelectionMode: () => void
  toggleSkillSelection: (promptId: string) => void
  toggleMultipleSkills: (promptIds: string[], select: boolean) => void
  saveAndExit: () => Promise<void>
}

export const useSkillSelectionStore = create<SkillSelectionStore>((set, get) => ({
  isActive: false,
  currentAvatarId: null,
  currentAvatar: null,
  selectedSkillIds: new Set(),
  onSave: null,

  enterSkillSelectionMode: (avatar, currentSkills, onSave) => {
    set({
      isActive: true,
      currentAvatarId: avatar.id,
      currentAvatar: avatar,
      selectedSkillIds: new Set(currentSkills),
      onSave,
    })
  },

  exitSkillSelectionMode: () => {
    set({
      isActive: false,
      currentAvatarId: null,
      currentAvatar: null,
      selectedSkillIds: new Set(),
      onSave: null,
    })
  },

  toggleSkillSelection: (promptId) => {
    set((state) => {
      const next = new Set(state.selectedSkillIds)
      if (next.has(promptId)) {
        next.delete(promptId)
      } else {
        next.add(promptId)
      }
      return { selectedSkillIds: next }
    })
  },

  toggleMultipleSkills: (promptIds, select) => {
    set((state) => {
      const next = new Set(state.selectedSkillIds)
      for (const id of promptIds) {
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
