/**
 * Team activity store — ephemeral runtime state (no persistence).
 *
 * Tracks which teams are upcoming or running, and which furniture
 * has been allocated for team gatherings.
 */

import { create } from 'zustand'

export interface TeamActivity {
  teamId: string
  teamName: string
  memberAgentIds: string[]
  status: 'upcoming' | 'running'
  /** upcoming: nextExecution ISO; running: startedAt ISO */
  referenceTime: string
  /** Which member's heartbeat triggers */
  heartbeatAgentId?: string
}

export interface TeamFurnitureAllocation {
  teamId: string
  furnitureId: string | null
  fallbackPosition?: [number, number, number]
}

interface TeamActivityState {
  activities: TeamActivity[]
  allocations: TeamFurnitureAllocation[]
  getActivitiesForFurniture: (furnitureId: string) => TeamActivity[]
  getTeamAllocation: (teamId: string) => TeamFurnitureAllocation | undefined
  setActivities: (activities: TeamActivity[]) => void
  setAllocations: (allocations: TeamFurnitureAllocation[]) => void
  clear: () => void
}

export const useTeamActivityStore = create<TeamActivityState>()((set, get) => ({
  activities: [],
  allocations: [],

  getActivitiesForFurniture: (furnitureId) => {
    const { activities, allocations } = get()
    const teamIds = new Set(
      allocations
        .filter((a) => a.furnitureId === furnitureId)
        .map((a) => a.teamId),
    )
    return activities.filter((a) => teamIds.has(a.teamId))
  },

  getTeamAllocation: (teamId) => {
    return get().allocations.find((a) => a.teamId === teamId)
  },

  setActivities: (activities) => set({ activities }),
  setAllocations: (allocations) => set({ allocations }),

  clear: () => set({ activities: [], allocations: [] }),
}))
