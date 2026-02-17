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
  setAll: (positions) => {
    const current = get().positions
    const currentKeys = Object.keys(current)
    const nextKeys = Object.keys(positions)
    if (currentKeys.length === nextKeys.length) {
      let unchanged = true
      for (const key of nextKeys) {
        const currentPos = current[key]
        const nextPos = positions[key]
        if (
          !currentPos ||
          !nextPos ||
          currentPos[0] !== nextPos[0] ||
          currentPos[1] !== nextPos[1] ||
          currentPos[2] !== nextPos[2]
        ) {
          unchanged = false
          break
        }
      }
      if (unchanged) return
    }
    set({ positions })
  },
  getPosition: (agentId) => get().positions[agentId] ?? null,
}))
