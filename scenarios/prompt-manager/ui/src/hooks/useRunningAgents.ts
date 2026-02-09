/**
 * useRunningAgents - Polling hook for tracking running heartbeat agents.
 *
 * Polls the running agents endpoint every 5 seconds and provides:
 * - Flat list and grouped-by-team view
 * - Stop action with optimistic removal
 * - Loading and error state
 */

import { useState, useEffect, useMemo, useCallback, useRef } from 'react'
import {
  listRunningAgents,
  stopRunningAgent,
  type RunningAgentEntry,
} from '@/services/heartbeatService'

const POLL_INTERVAL_MS = 5000

export interface TeamGroup {
  teamId: string
  teamName: string
  agents: RunningAgentEntry[]
}

export interface UseRunningAgentsResult {
  runningAgents: RunningAgentEntry[]
  groupedByTeam: TeamGroup[]
  count: number
  isLoading: boolean
  stopAgent: (teamId: string, agentId: string) => Promise<void>
  stoppingIds: Set<string>
}

function agentKey(teamId: string, agentId: string) {
  return `${teamId}/${agentId}`
}

export function useRunningAgents(): UseRunningAgentsResult {
  const [runningAgents, setRunningAgents] = useState<RunningAgentEntry[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [stoppingIds, setStoppingIds] = useState<Set<string>>(new Set())
  const mountedRef = useRef(true)

  // Poll for running agents
  useEffect(() => {
    mountedRef.current = true
    let timeoutId: ReturnType<typeof setTimeout>

    async function poll() {
      try {
        const response = await listRunningAgents()
        if (mountedRef.current) {
          setRunningAgents(response.agents)
          setIsLoading(false)
        }
      } catch {
        // Silently ignore poll errors — will retry
        if (mountedRef.current) {
          setIsLoading(false)
        }
      }
      if (mountedRef.current) {
        timeoutId = setTimeout(() => void poll(), POLL_INTERVAL_MS)
      }
    }

    void poll()

    return () => {
      mountedRef.current = false
      clearTimeout(timeoutId)
    }
  }, [])

  // Group by team
  const groupedByTeam = useMemo(() => {
    const groups = new Map<string, TeamGroup>()
    const order: string[] = []

    for (const agent of runningAgents) {
      if (!groups.has(agent.teamId)) {
        groups.set(agent.teamId, {
          teamId: agent.teamId,
          teamName: agent.teamName || agent.teamId,
          agents: [],
        })
        order.push(agent.teamId)
      }
      groups.get(agent.teamId)?.agents.push(agent)
    }

    return order.map((id) => groups.get(id)).filter(
      (g): g is TeamGroup => g !== undefined
    )
  }, [runningAgents])

  // Stop action with optimistic removal
  const stopAgent = useCallback(async (teamId: string, agentId: string) => {
    const key = agentKey(teamId, agentId)
    setStoppingIds((prev) => new Set(prev).add(key))

    try {
      await stopRunningAgent(teamId, agentId)
      // Optimistic removal
      if (mountedRef.current) {
        setRunningAgents((prev) =>
          prev.filter((a) => !(a.teamId === teamId && a.agentId === agentId))
        )
      }
    } catch {
      // Will be corrected on next poll
    } finally {
      if (mountedRef.current) {
        setStoppingIds((prev) => {
          const next = new Set(prev)
          next.delete(key)
          return next
        })
      }
    }
  }, [])

  return {
    runningAgents,
    groupedByTeam,
    count: runningAgents.length,
    isLoading,
    stopAgent,
    stoppingIds,
  }
}
