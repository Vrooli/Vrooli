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
  /** Which member heartbeats are scheduled at the reference time */
  scheduledAgentIds?: string[]
}

export interface TeamFurnitureAllocation {
  teamId: string
  furnitureId: string | null
  fallbackPosition?: [number, number, number]
}

function areActivitiesEqual(a: TeamActivity[], b: TeamActivity[]): boolean {
  if (a.length !== b.length) return false
  for (let i = 0; i < a.length; i++) {
    const left = a[i]
    const right = b[i]
    if (!left || !right) return false
    if (
      left.teamId !== right.teamId ||
      left.teamName !== right.teamName ||
      left.status !== right.status ||
      left.referenceTime !== right.referenceTime ||
      (left.scheduledAgentIds?.join(',') ?? '') !== (right.scheduledAgentIds?.join(',') ?? '') ||
      left.memberAgentIds.length !== right.memberAgentIds.length
    ) {
      return false
    }
    for (let j = 0; j < left.memberAgentIds.length; j++) {
      if (left.memberAgentIds[j] !== right.memberAgentIds[j]) {
        return false
      }
    }
  }
  return true
}

function areAllocationsEqual(a: TeamFurnitureAllocation[], b: TeamFurnitureAllocation[]): boolean {
  if (a.length !== b.length) return false
  for (let i = 0; i < a.length; i++) {
    const left = a[i]
    const right = b[i]
    if (!left || !right) return false
    if (
      left.teamId !== right.teamId ||
      left.furnitureId !== right.furnitureId
    ) {
      return false
    }
    const leftPos = left.fallbackPosition
    const rightPos = right.fallbackPosition
    if (!leftPos && !rightPos) continue
    if (!leftPos || !rightPos) return false
    if (leftPos[0] !== rightPos[0] || leftPos[1] !== rightPos[1] || leftPos[2] !== rightPos[2]) {
      return false
    }
  }
  return true
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

  setActivities: (activities) => {
    if (areActivitiesEqual(get().activities, activities)) return
    set({ activities })
  },
  setAllocations: (allocations) => {
    if (areAllocationsEqual(get().allocations, allocations)) return
    set({ allocations })
  },

  clear: () => set({ activities: [], allocations: [] }),
}))
