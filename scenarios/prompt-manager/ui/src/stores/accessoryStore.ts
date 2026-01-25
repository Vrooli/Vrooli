/**
 * Accessory store for managing member accessories.
 * Stores accessory configurations per member.
 */

import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { MemberAccessories, MemberStatus } from '@/types/accessory'

interface MemberAccessoryState {
  accessories: MemberAccessories
  status: MemberStatus | null
}

interface AccessoryState {
  /** Accessory configurations by member ID */
  memberAccessories: Record<string, MemberAccessoryState>
  /** Global default accessories for new members */
  defaults: Partial<MemberAccessories>
}

interface AccessoryActions {
  /** Set accessories for a specific member */
  setMemberAccessories: (memberId: string, accessories: Partial<MemberAccessories>) => void
  /** Set status for a specific member */
  setMemberStatus: (memberId: string, status: MemberStatus | null) => void
  /** Clear status for a specific member */
  clearMemberStatus: (memberId: string) => void
  /** Get accessories for a member (with defaults applied) */
  getMemberAccessories: (memberId: string) => MemberAccessories
  /** Get status for a member */
  getMemberStatus: (memberId: string) => MemberStatus | null
  /** Set default accessories */
  setDefaults: (defaults: Partial<MemberAccessories>) => void
  /** Remove all accessories for a member */
  removeMember: (memberId: string) => void
  /** Clear all accessory data */
  reset: () => void
}

type AccessoryStore = AccessoryState & AccessoryActions

const initialState: AccessoryState = {
  memberAccessories: {},
  defaults: {
    head: { type: 'none' },
    back: { type: 'none' },
    held: { type: 'none' },
  },
}

/**
 * Zustand store for accessory management with persistence
 */
export const useAccessoryStore = create<AccessoryStore>()(
  persist(
    (set, get) => ({
      ...initialState,

      setMemberAccessories: (memberId, accessories) => {
        const current = get().memberAccessories[memberId] ?? { accessories: {}, status: null }
        set({
          memberAccessories: {
            ...get().memberAccessories,
            [memberId]: {
              ...current,
              accessories: { ...current.accessories, ...accessories },
            },
          },
        })
      },

      setMemberStatus: (memberId, status) => {
        const current = get().memberAccessories[memberId] ?? { accessories: {}, status: null }
        set({
          memberAccessories: {
            ...get().memberAccessories,
            [memberId]: {
              ...current,
              status,
            },
          },
        })
      },

      clearMemberStatus: (memberId) => {
        const current = get().memberAccessories[memberId]
        if (!current) return

        set({
          memberAccessories: {
            ...get().memberAccessories,
            [memberId]: {
              ...current,
              status: null,
            },
          },
        })
      },

      getMemberAccessories: (memberId) => {
        const { memberAccessories, defaults } = get()
        const memberState = memberAccessories[memberId]
        return {
          head: memberState?.accessories.head ?? defaults.head ?? { type: 'none' },
          back: memberState?.accessories.back ?? defaults.back ?? { type: 'none' },
          held: memberState?.accessories.held ?? defaults.held ?? { type: 'none' },
        }
      },

      getMemberStatus: (memberId) => {
        return get().memberAccessories[memberId]?.status ?? null
      },

      setDefaults: (defaults) => {
        set({
          defaults: { ...get().defaults, ...defaults },
        })
      },

      removeMember: (memberId) => {
        const { memberAccessories } = get()
        const { [memberId]: _, ...rest } = memberAccessories
        void _
        set({ memberAccessories: rest })
      },

      reset: () => set(initialState),
    }),
    {
      name: 'member-accessories',
      partialize: (state) => ({
        memberAccessories: state.memberAccessories,
        defaults: state.defaults,
      }),
    }
  )
)

/**
 * Hook for getting accessories for a specific member
 */
export function useMemberAccessoriesSelector(memberId: string) {
  return useAccessoryStore((state) => state.getMemberAccessories(memberId))
}

/**
 * Hook for getting status for a specific member
 */
export function useMemberStatusSelector(memberId: string) {
  return useAccessoryStore((state) => state.getMemberStatus(memberId))
}
