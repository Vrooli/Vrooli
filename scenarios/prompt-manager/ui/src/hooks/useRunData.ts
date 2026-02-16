/**
 * useRunData - Hook for fetching and managing run list data.
 *
 * Features:
 * - Fetches runs via listRuns()
 * - Filters by status and tag prefix
 * - Polls every 10s when active runs exist
 * - Returns { runs, loading, error, refetch }
 */

import { useState, useEffect, useCallback, useRef } from 'react'
import { listRuns, type RunDetails } from '@/services/heartbeatService'

const POLL_INTERVAL = 10_000

interface UseRunDataOptions {
  status?: string
  tagPrefix?: string
}

interface UseRunDataResult {
  runs: RunDetails[]
  loading: boolean
  error: string | null
  refetch: () => void
}

export function useRunData(opts?: UseRunDataOptions): UseRunDataResult {
  const [runs, setRuns] = useState<RunDetails[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const isFirstLoad = useRef(true)

  const fetchRuns = useCallback(async () => {
    if (isFirstLoad.current) {
      setLoading(true)
    }
    try {
      const response = await listRuns({
        status: opts?.status,
        tagPrefix: opts?.tagPrefix,
        limit: 100,
      })
      setRuns(response.runs)
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
  }, [opts?.status, opts?.tagPrefix])

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
