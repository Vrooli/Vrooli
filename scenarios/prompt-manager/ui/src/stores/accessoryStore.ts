/**
 * Accessory store for managing agent accessories.
 * Stores accessory configurations per agent.
 */

import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { AgentAccessories, AgentStatus } from '@/types/accessory'

interface AgentAccessoryState {
  accessories: AgentAccessories
  status: AgentStatus | null
}

interface AccessoryState {
  /** Accessory configurations by agent ID */
  agentAccessories: Record<string, AgentAccessoryState>
  /** Global default accessories for new agents */
  defaults: Partial<AgentAccessories>
}

interface AccessoryActions {
  /** Set accessories for a specific agent */
  setAgentAccessories: (agentId: string, accessories: Partial<AgentAccessories>) => void
  /** Set status for a specific agent */
  setAgentStatus: (agentId: string, status: AgentStatus | null) => void
  /** Clear status for a specific agent */
  clearAgentStatus: (agentId: string) => void
  /** Get accessories for a agent (with defaults applied) */
  getAgentAccessories: (agentId: string) => AgentAccessories
  /** Get status for a agent */
  getAgentStatus: (agentId: string) => AgentStatus | null
  /** Set default accessories */
  setDefaults: (defaults: Partial<AgentAccessories>) => void
  /** Remove all accessories for a agent */
  removeAgent: (agentId: string) => void
  /** Clear all accessory data */
  reset: () => void
}

type AccessoryStore = AccessoryState & AccessoryActions

const initialState: AccessoryState = {
  agentAccessories: {},
  defaults: {
    head: { type: 'none' },
    back: { type: 'none' },
    held: { type: 'none' },
    clothingTop: { type: 'none' },
    clothingBottom: { type: 'none' },
    footwear: { type: 'none' },
  },
}

/**
 * Zustand store for accessory management with persistence
 */
export const useAccessoryStore = create<AccessoryStore>()(
  persist(
    (set, get) => ({
      ...initialState,

      setAgentAccessories: (agentId, accessories) => {
        const current = get().agentAccessories[agentId] ?? { accessories: {}, status: null }
        set({
          agentAccessories: {
            ...get().agentAccessories,
            [agentId]: {
              ...current,
              accessories: { ...current.accessories, ...accessories },
            },
          },
        })
      },

      setAgentStatus: (agentId, status) => {
        const current = get().agentAccessories[agentId] ?? { accessories: {}, status: null }
        set({
          agentAccessories: {
            ...get().agentAccessories,
            [agentId]: {
              ...current,
              status,
            },
          },
        })
      },

      clearAgentStatus: (agentId) => {
        const current = get().agentAccessories[agentId]
        if (!current) return

        set({
          agentAccessories: {
            ...get().agentAccessories,
            [agentId]: {
              ...current,
              status: null,
            },
          },
        })
      },

      getAgentAccessories: (agentId) => {
        const { agentAccessories, defaults } = get()
        const agentState = agentAccessories[agentId]
        return {
          head: agentState?.accessories.head ?? defaults.head ?? { type: 'none' },
          back: agentState?.accessories.back ?? defaults.back ?? { type: 'none' },
          held: agentState?.accessories.held ?? defaults.held ?? { type: 'none' },
          clothingTop: agentState?.accessories.clothingTop ?? defaults.clothingTop ?? { type: 'none' },
          clothingBottom: agentState?.accessories.clothingBottom ?? defaults.clothingBottom ?? { type: 'none' },
          footwear: agentState?.accessories.footwear ?? defaults.footwear ?? { type: 'none' },
        }
      },

      getAgentStatus: (agentId) => {
        return get().agentAccessories[agentId]?.status ?? null
      },

      setDefaults: (defaults) => {
        set({
          defaults: { ...get().defaults, ...defaults },
        })
      },

      removeAgent: (agentId) => {
        const { agentAccessories } = get()
        const { [agentId]: _, ...rest } = agentAccessories
        void _
        set({ agentAccessories: rest })
      },

      reset: () => set(initialState),
    }),
    {
      name: 'agent-accessories',
      partialize: (state) => ({
        agentAccessories: state.agentAccessories,
        defaults: state.defaults,
      }),
    }
  )
)

/**
 * Hook for getting accessories for a specific agent
 */
export function useAgentAccessoriesSelector(agentId: string) {
  return useAccessoryStore((state) => state.getAgentAccessories(agentId))
}

/**
 * Hook for getting status for a specific agent
 */
export function useAgentStatusSelector(agentId: string) {
  return useAccessoryStore((state) => state.getAgentStatus(agentId))
}
