/**
 * Pending decisions store — ephemeral runtime state (no persistence).
 * Tracks pending decisions across all teams for sidebar badge and world view.
 */
import { create } from 'zustand'
import type { PendingDecisionTeamGroup } from '@/services/heartbeatService'

interface PendingDecisionsState {
  groups: PendingDecisionTeamGroup[]
  total: number
  /** Get pending count for a specific team */
  getTeamPendingCount: (teamId: string) => number
  /** Get all pending decision submitter agent IDs */
  getPendingAgentIds: () => string[]
  setGroups: (groups: PendingDecisionTeamGroup[], total: number) => void
  clear: () => void
}

function areGroupsEqual(a: PendingDecisionTeamGroup[], b: PendingDecisionTeamGroup[]): boolean {
  if (a.length !== b.length) return false
  for (let i = 0; i < a.length; i++) {
    const left = a[i]
    const right = b[i]
    if (!left || !right) return false
    if (left.teamId !== right.teamId || left.teamName !== right.teamName || left.entries.length !== right.entries.length) return false
    for (let j = 0; j < left.entries.length; j++) {
      if (left.entries[j]?.id !== right.entries[j]?.id || left.entries[j]?.status !== right.entries[j]?.status) return false
    }
  }
  return true
}

export const usePendingDecisionsStore = create<PendingDecisionsState>()((set, get) => ({
  groups: [],
  total: 0,

  getTeamPendingCount: (teamId) => {
    const group = get().groups.find((g) => g.teamId === teamId)
    return group ? group.entries.length : 0
  },

  getPendingAgentIds: () => {
    const ids: string[] = []
    for (const group of get().groups) {
      for (const entry of group.entries) {
        if (entry.by && !ids.includes(entry.by)) {
          ids.push(entry.by)
        }
      }
    }
    return ids
  },

  setGroups: (groups, total) => {
    if (areGroupsEqual(get().groups, groups) && get().total === total) return
    set({ groups, total })
  },

  clear: () => set({ groups: [], total: 0 }),
}))
