/**
 * useTeamActivity — polling hook that detects upcoming and running teams,
 * then allocates furniture for gatherings.
 *
 * Runs every 10 s. Only fetches team details when the set of active teamIds changes.
 */

import { useEffect, useRef, useCallback } from 'react'
import { listRunningAgents, listHeartbeats } from '@/services/heartbeatService'
import { getTeams, getTeam } from '@/services/teamService'
import { useFurnitureStore } from '@/stores/furnitureStore'
import { getSeats } from '@/stores/worldSeatsStore'
import { useTeamActivityStore, type TeamActivity, type TeamFurnitureAllocation } from '@/stores/teamActivityStore'

const POLL_INTERVAL_MS = 10_000
const UPCOMING_THRESHOLD_MS = 5 * 60 * 1000 // 5 minutes

export function useTeamActivity() {
  const setActivities = useTeamActivityStore((s) => s.setActivities)
  const setAllocations = useTeamActivityStore((s) => s.setAllocations)

  // Cache team details keyed by teamId to avoid re-fetching every cycle
  const teamDetailsCacheRef = useRef<Map<string, { memberAgentIds: string[]; teamName: string }>>(new Map())
  const prevTeamIdsRef = useRef<string>('')

  const poll = useCallback(async () => {
    try {
      const now = Date.now()
      const activities: TeamActivity[] = []

      // 1. Find running teams
      const runningResponse = await listRunningAgents()
      const runningByTeam = new Map<string, { agentIds: string[]; startedAt: string }>()
      for (const entry of runningResponse.agents) {
        const existing = runningByTeam.get(entry.teamId)
        if (existing) {
          existing.agentIds.push(entry.agentId)
          // Use earliest startedAt
          if (entry.startedAt < existing.startedAt) {
            existing.startedAt = entry.startedAt
          }
        } else {
          runningByTeam.set(entry.teamId, {
            agentIds: [entry.agentId],
            startedAt: entry.startedAt,
          })
        }
      }

      // 2. Find upcoming teams (heartbeats within threshold)
      const teams = await getTeams()
      const upcomingTeams = new Map<string, { nextExecution: string; agentId: string }>()

      await Promise.all(
        teams.map(async (team) => {
          // Skip teams that are already running
          if (runningByTeam.has(team.id)) return
          try {
            const heartbeats = await listHeartbeats(team.id)
            for (const hb of heartbeats) {
              if (!hb.enabled || !hb.nextExecution) continue
              const nextTime = new Date(hb.nextExecution).getTime()
              if (nextTime - now <= UPCOMING_THRESHOLD_MS && nextTime > now) {
                const existing = upcomingTeams.get(team.id)
                // Use the soonest upcoming execution
                if (!existing || hb.nextExecution < existing.nextExecution) {
                  upcomingTeams.set(team.id, {
                    nextExecution: hb.nextExecution,
                    agentId: hb.agentId,
                  })
                }
              }
            }
          } catch {
            // Skip teams whose heartbeats can't be fetched
          }
        }),
      )

      // 3. Determine which teamIds need detail lookups
      const activeTeamIds = new Set([...runningByTeam.keys(), ...upcomingTeams.keys()])
      const sortedIds = [...activeTeamIds].sort().join(',')

      // Re-fetch team details only when the active set changes
      if (sortedIds !== prevTeamIdsRef.current) {
        prevTeamIdsRef.current = sortedIds
        const cache = teamDetailsCacheRef.current
        // Evict stale entries
        for (const key of cache.keys()) {
          if (!activeTeamIds.has(key)) cache.delete(key)
        }
        // Fetch missing
        await Promise.all(
          [...activeTeamIds].map(async (teamId) => {
            if (cache.has(teamId)) return
            try {
              const details = await getTeam(teamId)
              if (details) {
                cache.set(teamId, {
                  memberAgentIds: details.members.map((m) => m.agentId),
                  teamName: details.displayName,
                })
              }
            } catch {
              // Skip
            }
          }),
        )
      }

      // 4. Build activity list
      for (const [teamId, info] of runningByTeam) {
        const cached = teamDetailsCacheRef.current.get(teamId)
        activities.push({
          teamId,
          teamName: cached?.teamName ?? teamId,
          memberAgentIds: cached?.memberAgentIds ?? info.agentIds,
          status: 'running',
          referenceTime: info.startedAt,
        })
      }

      for (const [teamId, info] of upcomingTeams) {
        const cached = teamDetailsCacheRef.current.get(teamId)
        activities.push({
          teamId,
          teamName: cached?.teamName ?? teamId,
          memberAgentIds: cached?.memberAgentIds ?? [],
          status: 'upcoming',
          referenceTime: info.nextExecution,
          heartbeatAgentId: info.agentId,
        })
      }

      setActivities(activities)

      // 5. Allocate furniture
      allocateFurniture(activities, setAllocations)
    } catch (err) {
      console.error('[useTeamActivity] Poll error:', err)
    }
  }, [setActivities, setAllocations])

  useEffect(() => {
    void poll()
    const id = setInterval(() => void poll(), POLL_INTERVAL_MS)
    return () => {
      clearInterval(id)
      useTeamActivityStore.getState().clear()
    }
  }, [poll])
}

/**
 * Allocate furniture to team activities. Larger teams pick first.
 * Falls back to a random position near center when no furniture fits.
 */
function allocateFurniture(
  activities: TeamActivity[],
  setAllocations: (a: TeamFurnitureAllocation[]) => void,
) {
  const store = useFurnitureStore.getState()
  const furnitureList = Object.values(store.scenes).flat().filter(Boolean) as { id: string; type: string }[]

  // Build seat counts
  const seatCounts = new Map<string, number>()
  for (const f of furnitureList) {
    seatCounts.set(f.id, getSeats(f.type as Parameters<typeof getSeats>[0]).length)
  }

  // Sort teams by member count descending (larger teams pick first)
  const sorted = [...activities].sort((a, b) => b.memberAgentIds.length - a.memberAgentIds.length)

  const allocated = new Set<string>()
  const allocations: TeamFurnitureAllocation[] = []

  // Sort furniture by seat count descending for greedy matching
  const availableFurniture = [...seatCounts.entries()]
    .sort(([, a], [, b]) => b - a)

  for (const activity of sorted) {
    const memberCount = activity.memberAgentIds.length

    // Find first unallocated furniture with enough seats
    let chosen: string | null = null
    for (const [fId, seats] of availableFurniture) {
      if (allocated.has(fId)) continue
      if (seats >= memberCount) {
        chosen = fId
        break
      }
    }

    // Fallback: furniture with most available seats
    if (!chosen) {
      for (const [fId] of availableFurniture) {
        if (!allocated.has(fId)) {
          chosen = fId
          break
        }
      }
    }

    if (chosen) {
      allocated.add(chosen)
      allocations.push({ teamId: activity.teamId, furnitureId: chosen })
    } else {
      // No furniture at all — assign a random fallback position
      allocations.push({
        teamId: activity.teamId,
        furnitureId: null,
        fallbackPosition: [
          (Math.random() - 0.5) * 4,
          0,
          (Math.random() - 0.5) * 4,
        ],
      })
    }
  }

  setAllocations(allocations)
}
