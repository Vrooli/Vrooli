/**
 * usePendingDecisions - Polling hook for tracking pending decisions across all teams.
 * Follows the same shared-state polling pattern as useRunningAgents.
 */
import { useState, useEffect, useMemo, useCallback, useRef } from 'react'
import {
  getAllPendingDecisions,
  updateDecision,
  type PendingDecisionTeamGroup,
} from '@/services/heartbeatService'
import { refreshHeartbeatControlStatus } from './useHeartbeatControlStatus'

const POLL_INTERVAL_MS = 10_000

export interface UsePendingDecisionsResult {
  groupedByTeam: PendingDecisionTeamGroup[]
  count: number
  isLoading: boolean
  acceptDecision: (teamId: string, decisionId: string) => Promise<void>
  rejectDecision: (teamId: string, decisionId: string) => Promise<void>
  selectOption: (teamId: string, decisionId: string, selected: string, freeform?: string, notes?: string) => Promise<void>
  processingIds: Set<string>
}

function areGroupsEqual(a: PendingDecisionTeamGroup[], b: PendingDecisionTeamGroup[]): boolean {
  if (a.length !== b.length) return false
  for (let i = 0; i < a.length; i++) {
    const left = a[i]
    const right = b[i]
    if (!left || !right) return false
    if (left.teamId !== right.teamId || left.entries.length !== right.entries.length) return false
    for (let j = 0; j < left.entries.length; j++) {
      if (left.entries[j]?.id !== right.entries[j]?.id) return false
    }
  }
  return true
}

interface SharedPendingState {
  groups: PendingDecisionTeamGroup[]
  total: number
  isLoading: boolean
}

const INITIAL_STATE: SharedPendingState = { groups: [], total: 0, isLoading: true }

let sharedState: SharedPendingState = INITIAL_STATE
let pollTimeoutId: ReturnType<typeof setTimeout> | null = null
let isPolling = false
let hasPrimed = false
const subscribers = new Set<(state: SharedPendingState) => void>()

export function resetPendingDecisionsPollingForTests() {
  sharedState = INITIAL_STATE
  clearPollTimeout()
  isPolling = false
  hasPrimed = false
  subscribers.clear()
}

function emitState() {
  for (const listener of subscribers) listener(sharedState)
}

function clearPollTimeout() {
  if (pollTimeoutId !== null) {
    clearTimeout(pollTimeoutId)
    pollTimeoutId = null
  }
}

function schedulePoll() {
  if (pollTimeoutId !== null || subscribers.size === 0) return
  pollTimeoutId = setTimeout(() => {
    pollTimeoutId = null
    void poll()
  }, POLL_INTERVAL_MS)
}

async function poll() {
  if (isPolling || subscribers.size === 0) return
  isPolling = true
  try {
    if (document.hidden) {
      schedulePoll()
      return
    }
    const response = await getAllPendingDecisions()
    if (!areGroupsEqual(sharedState.groups, response.teams) || sharedState.isLoading) {
      sharedState = { groups: response.teams, total: response.totalCount, isLoading: false }
      emitState()
    } else if (!hasPrimed) {
      sharedState = { ...sharedState, isLoading: false }
      emitState()
    }
    hasPrimed = true
  } catch {
    if (sharedState.isLoading) {
      sharedState = { ...sharedState, isLoading: false }
      emitState()
    }
  } finally {
    isPolling = false
    schedulePoll()
  }
}

function subscribe(listener: (state: SharedPendingState) => void): () => void {
  subscribers.add(listener)
  listener(sharedState)
  if (subscribers.size === 1) {
    clearPollTimeout()
    void poll()
  }
  return () => {
    subscribers.delete(listener)
    if (subscribers.size === 0) clearPollTimeout()
  }
}

function updateSharedGroups(mutator: (groups: PendingDecisionTeamGroup[]) => PendingDecisionTeamGroup[]) {
  const next = mutator(sharedState.groups)
  const nextTotal = next.reduce((sum, g) => sum + g.entries.length, 0)
  if (areGroupsEqual(sharedState.groups, next)) return
  sharedState = { ...sharedState, groups: next, total: nextTotal }
  emitState()
}

export function usePendingDecisions(): UsePendingDecisionsResult {
  const [groups, setGroups] = useState<PendingDecisionTeamGroup[]>(() => sharedState.groups)
  const [isLoading, setIsLoading] = useState(() => sharedState.isLoading)
  const [processingIds, setProcessingIds] = useState<Set<string>>(new Set())
  const mountedRef = useRef(true)

  useEffect(() => {
    mountedRef.current = true
    const unsubscribe = subscribe((state) => {
      if (!mountedRef.current) return
      setGroups(state.groups)
      setIsLoading(state.isLoading)
    })
    return () => {
      mountedRef.current = false
      unsubscribe()
    }
  }, [])

  const count = useMemo(() => groups.reduce((sum, g) => sum + g.entries.length, 0), [groups])

  const processDecision = useCallback(async (teamId: string, decisionId: string, status: string) => {
    setProcessingIds((prev) => new Set(prev).add(decisionId))
    try {
      await updateDecision(teamId, decisionId, { status })
      if (mountedRef.current) {
        // Optimistic removal
        updateSharedGroups((prev) =>
          prev
            .map((g) => g.teamId === teamId ? { ...g, entries: g.entries.filter((e) => e.id !== decisionId) } : g)
            .filter((g) => g.entries.length > 0),
        )
      }
      void refreshHeartbeatControlStatus()
    } catch {
      // Will be corrected on next poll
    } finally {
      if (mountedRef.current) {
        setProcessingIds((prev) => {
          const next = new Set(prev)
          next.delete(decisionId)
          return next
        })
      }
    }
  }, [])

  const acceptDecision = useCallback((teamId: string, decisionId: string) => processDecision(teamId, decisionId, 'accepted'), [processDecision])
  const rejectDecision = useCallback((teamId: string, decisionId: string) => processDecision(teamId, decisionId, 'rejected'), [processDecision])

  const selectOption = useCallback(async (teamId: string, decisionId: string, selected: string, freeform?: string, notes?: string) => {
    setProcessingIds((prev) => new Set(prev).add(decisionId))
    try {
      await updateDecision(teamId, decisionId, {
        selected,
        freeform: freeform ?? null,
        notes: notes ?? null,
        status: 'accepted',
      })
      if (mountedRef.current) {
        updateSharedGroups((prev) =>
          prev
            .map((g) => g.teamId === teamId ? { ...g, entries: g.entries.filter((e) => e.id !== decisionId) } : g)
            .filter((g) => g.entries.length > 0),
        )
      }
      void refreshHeartbeatControlStatus()
    } catch {
      // Will be corrected on next poll
    } finally {
      if (mountedRef.current) {
        setProcessingIds((prev) => {
          const next = new Set(prev)
          next.delete(decisionId)
          return next
        })
      }
    }
  }, [])

  return { groupedByTeam: groups, count, isLoading, acceptDecision, rejectDecision, selectOption, processingIds }
}
