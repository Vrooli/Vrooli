/**
 * useTeamGathering — reacts to teamActivityStore changes and
 * seats/unseats agents at allocated furniture via furnitureStore.
 *
 * - When an activity is added: seat team members at the allocated furniture.
 * - When an activity is removed: unseat members so they walk back.
 * - If an agent is already seated for another active team, skip it.
 */

import { useEffect, useRef } from 'react'
import { useTeamActivityStore, type TeamActivity } from '@/stores/teamActivityStore'
import { useFurnitureStore } from '@/stores/furnitureStore'

export function useTeamGathering() {
  const activities = useTeamActivityStore((s) => s.activities)
  const allocations = useTeamActivityStore((s) => s.allocations)
  const prevActivitiesRef = useRef<TeamActivity[]>([])

  useEffect(() => {
    const prev = prevActivitiesRef.current
    prevActivitiesRef.current = activities

    const prevTeamIds = new Set(prev.map((a) => a.teamId))
    const currTeamIds = new Set(activities.map((a) => a.teamId))

    const store = useFurnitureStore.getState()
    const allocationMap = new Map(allocations.map((a) => [a.teamId, a]))

    // Track which agents are claimed by any *current* activity to avoid conflicts
    const claimedAgents = new Set<string>()
    for (const activity of activities) {
      for (const agentId of activity.memberAgentIds) {
        claimedAgents.add(agentId)
      }
    }

    // Handle removed activities: unseat members
    for (const prevActivity of prev) {
      if (!currTeamIds.has(prevActivity.teamId)) {
        for (const agentId of prevActivity.memberAgentIds) {
          // Only unseat if the agent isn't claimed by another active team
          if (!claimedAgents.has(agentId)) {
            store.unseatAgent(agentId)
          }
        }
      }
    }

    // Handle added activities: seat members
    // Build set of agents already seated for an *active* team (to skip conflicts)
    const agentsSeatedForTeam = new Set<string>()

    for (const activity of activities) {
      if (prevTeamIds.has(activity.teamId)) {
        // Already active — mark its agents as seated
        for (const agentId of activity.memberAgentIds) {
          agentsSeatedForTeam.add(agentId)
        }
        continue
      }

      // New activity
      const allocation = allocationMap.get(activity.teamId)
      if (!allocation?.furnitureId) continue

      for (const agentId of activity.memberAgentIds) {
        if (agentsSeatedForTeam.has(agentId)) continue // skip conflict
        store.seatAgent(agentId, allocation.furnitureId)
        agentsSeatedForTeam.add(agentId)
      }
    }
  }, [activities, allocations])
}
