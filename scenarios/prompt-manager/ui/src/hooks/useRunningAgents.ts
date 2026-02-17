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

interface UseRunningAgentsOptions {
  enabled?: boolean
}

function agentKey(teamId: string, agentId: string) {
  return `${teamId}/${agentId}`
}

function areRunningAgentsEqual(a: RunningAgentEntry[], b: RunningAgentEntry[]): boolean {
  if (a.length !== b.length) return false
  for (let i = 0; i < a.length; i++) {
    const left = a[i]
    const right = b[i]
    if (!left || !right) return false
    if (
      left.teamId !== right.teamId ||
      left.agentId !== right.agentId ||
      left.agentName !== right.agentName ||
      left.teamName !== right.teamName ||
      left.runId !== right.runId ||
      left.startedAt !== right.startedAt ||
      left.duration !== right.duration
    ) {
      return false
    }
  }
  return true
}

interface SharedRunningAgentsState {
  runningAgents: RunningAgentEntry[]
  isLoading: boolean
}

const INITIAL_SHARED_STATE: SharedRunningAgentsState = {
  runningAgents: [],
  isLoading: true,
}

let sharedState: SharedRunningAgentsState = INITIAL_SHARED_STATE
let pollTimeoutId: ReturnType<typeof setTimeout> | null = null
let isPolling = false
let hasPrimed = false
const subscribers = new Set<(state: SharedRunningAgentsState) => void>()

export function resetRunningAgentsPollingForTests() {
  sharedState = INITIAL_SHARED_STATE
  clearPollTimeout()
  isPolling = false
  hasPrimed = false
  subscribers.clear()
}

function emitSharedState() {
  for (const listener of subscribers) {
    listener(sharedState)
  }
}

function clearPollTimeout() {
  if (pollTimeoutId !== null) {
    clearTimeout(pollTimeoutId)
    pollTimeoutId = null
  }
}

function schedulePoll() {
  if (pollTimeoutId !== null || subscribers.size === 0) {
    return
  }
  pollTimeoutId = setTimeout(() => {
    pollTimeoutId = null
    void pollRunningAgents()
  }, POLL_INTERVAL_MS)
}

async function pollRunningAgents() {
  if (isPolling || subscribers.size === 0) {
    return
  }
  isPolling = true
  try {
    if (document.hidden) {
      schedulePoll()
      return
    }
    const response = await listRunningAgents()
    if (
      !areRunningAgentsEqual(sharedState.runningAgents, response.agents)
      || sharedState.isLoading
    ) {
      sharedState = {
        runningAgents: response.agents,
        isLoading: false,
      }
      emitSharedState()
    } else if (!hasPrimed) {
      // Ensure first successful poll exits loading state for late subscribers.
      sharedState = {
        ...sharedState,
        isLoading: false,
      }
      emitSharedState()
    }
    hasPrimed = true
  } catch {
    if (sharedState.isLoading) {
      sharedState = {
        ...sharedState,
        isLoading: false,
      }
      emitSharedState()
    }
  } finally {
    isPolling = false
    schedulePoll()
  }
}

function subscribeToRunningAgents(listener: (state: SharedRunningAgentsState) => void): () => void {
  subscribers.add(listener)
  listener(sharedState)

  if (subscribers.size === 1) {
    clearPollTimeout()
    void pollRunningAgents()
  }

  return () => {
    subscribers.delete(listener)
    if (subscribers.size === 0) {
      clearPollTimeout()
    }
  }
}

function updateSharedRunningAgents(mutator: (current: RunningAgentEntry[]) => RunningAgentEntry[]) {
  const nextAgents = mutator(sharedState.runningAgents)
  if (areRunningAgentsEqual(sharedState.runningAgents, nextAgents)) {
    return
  }
  sharedState = {
    ...sharedState,
    runningAgents: nextAgents,
  }
  emitSharedState()
}

export function useRunningAgents(options?: UseRunningAgentsOptions): UseRunningAgentsResult {
  const enabled = options?.enabled ?? true
  const [runningAgents, setRunningAgents] = useState<RunningAgentEntry[]>(() =>
    enabled ? sharedState.runningAgents : []
  )
  const [isLoading, setIsLoading] = useState(() => enabled && sharedState.isLoading)
  const [stoppingIds, setStoppingIds] = useState<Set<string>>(new Set())
  const mountedRef = useRef(true)

  useEffect(() => {
    mountedRef.current = true
    if (!enabled) {
      setRunningAgents([])
      setIsLoading(false)
      return () => {
        mountedRef.current = false
      }
    }

    const unsubscribe = subscribeToRunningAgents((state) => {
      if (!mountedRef.current) return
      setRunningAgents(state.runningAgents)
      setIsLoading(state.isLoading)
    })

    return () => {
      mountedRef.current = false
      unsubscribe()
    }
  }, [enabled])

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
        updateSharedRunningAgents((prev) =>
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
