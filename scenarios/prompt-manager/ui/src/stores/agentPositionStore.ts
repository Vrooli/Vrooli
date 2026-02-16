/**
 * Agent position store - maps agentId to 3D world position.
 * WorldCanvas is the sole writer; any component can read.
 */

import { create } from 'zustand'

interface AgentPositionStore {
  positions: Record<string, [number, number, number]>
  setAll: (positions: Record<string, [number, number, number]>) => void
  getPosition: (agentId: string) => [number, number, number] | null
}

export const useAgentPositionStore = create<AgentPositionStore>((set, get) => ({
  positions: {},
  setAll: (positions) => set({ positions }),
  getPosition: (agentId) => get().positions[agentId] ?? null,
}))
