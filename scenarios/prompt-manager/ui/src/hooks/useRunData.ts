/**
 * useRunData - Hook for fetching and managing run list data.
 *
 * Features:
 * - Fetches runs via listRuns()
 * - Filters by status and tag prefix
 * - Polls every 10s when active runs exist
 * - Returns { runs, loading, error, refetch }
 */
// AI_CHECK: RUN_POLL_STATE_CHURN=2 | LAST: 2026-02-18

import { useState, useEffect, useCallback, useRef } from 'react'
import { listHeartbeatAttempts, listRuns, type HeartbeatAttempt, type RunDetails } from '@/services/heartbeatService'

const POLL_INTERVAL = 10_000
const DEFAULT_PROFILE_KEY = 'prompt-manager-heartbeat'

interface UseRunDataOptions {
  status?: string
  tagPrefix?: string
  profileKey?: string
}

interface UseRunDataResult {
  runs: RunDetails[]
  loading: boolean
  error: string | null
  refetch: () => void
}

export function areRunsEqual(a: RunDetails[], b: RunDetails[]): boolean {
  if (a.length !== b.length) return false
  for (let i = 0; i < a.length; i++) {
    const left = a[i]
    const right = b[i]
    if (!left || !right) return false
    if (
      left.id !== right.id ||
      left.tag !== right.tag ||
      left.status !== right.status ||
      left.startedAt !== right.startedAt ||
      left.endedAt !== right.endedAt ||
      left.error !== right.error ||
      left.taskId !== right.taskId ||
      left.profileId !== right.profileId ||
      left.sessionId !== right.sessionId
      || left.source !== right.source
      || left.phase !== right.phase
      || left.recovery !== right.recovery
      || left.errorCategory !== right.errorCategory
    ) {
      return false
    }
  }
  return true
}

function attemptToRun(attempt: HeartbeatAttempt): RunDetails {
  return {
    id: attempt.runId || `attempt:${attempt.id}`,
    taskId: attempt.taskId || '',
    status: attempt.status,
    startedAt: attempt.startedAt,
    endedAt: attempt.endedAt,
    error: attempt.error,
    tag: attempt.tag || `heartbeat-${attempt.teamId}-${attempt.agentId}`,
    teamId: attempt.teamId,
    agentId: attempt.agentId,
    source: 'heartbeat-attempt',
    phase: attempt.phase,
    recovery: attempt.recovery,
    errorCategory: attempt.errorCategory,
  }
}

function mergeRunsWithAttempts(runs: RunDetails[], attempts: HeartbeatAttempt[]): RunDetails[] {
  const runIds = new Set(runs.map((run) => run.id))
  const attemptRuns = attempts
    .filter((attempt) => !attempt.runId || !runIds.has(attempt.runId))
    .map(attemptToRun)
  return [...runs, ...attemptRuns].sort((a, b) => {
    const left = new Date(a.startedAt ?? a.endedAt ?? 0).getTime()
    const right = new Date(b.startedAt ?? b.endedAt ?? 0).getTime()
    return right - left
  })
}

export function useRunData(opts?: UseRunDataOptions): UseRunDataResult {
  const [runs, setRuns] = useState<RunDetails[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const isFirstLoad = useRef(true)

  const fetchRuns = useCallback(async () => {
    if (typeof document !== 'undefined' && document.hidden) {
      return
    }
    if (isFirstLoad.current) {
      setLoading(true)
    }
    try {
      const profileKey = opts?.profileKey ?? DEFAULT_PROFILE_KEY
      const [response, attemptsResponse] = await Promise.all([
        listRuns({
          status: opts?.status,
          tagPrefix: opts?.tagPrefix,
          profileKey,
          limit: 100,
        }),
        listHeartbeatAttempts({
          status: opts?.status,
          profileKey,
          limit: 100,
        }),
      ])
      const mergedRuns = mergeRunsWithAttempts(response.runs, attemptsResponse.attempts)
      // Avoid rerendering the runs tab when polled data is unchanged.
      setRuns((prev) => (areRunsEqual(prev, mergedRuns) ? prev : mergedRuns))
      setError(null)
    } catch (err) {
      if (isFirstLoad.current) {
        setError(err instanceof Error ? err.message : 'Failed to load runs')
      }
    } finally {
      if (isFirstLoad.current) {
        setLoading(false)
        isFirstLoad.current = false
      }
    }
  }, [opts?.status, opts?.tagPrefix, opts?.profileKey])

  // Initial fetch
  useEffect(() => {
    isFirstLoad.current = true
    void fetchRuns()
  }, [fetchRuns])

  // Poll when active runs exist
  const hasActiveRuns = runs.some((r) => r.status === 'running' || r.status === 'pending')

  useEffect(() => {
    if (!hasActiveRuns) return
    const interval = setInterval(() => void fetchRuns(), POLL_INTERVAL)
    return () => clearInterval(interval)
  }, [hasActiveRuns, fetchRuns])

  const refetch = useCallback(() => {
    void fetchRuns()
  }, [fetchRuns])

  return { runs, loading, error, refetch }
}
