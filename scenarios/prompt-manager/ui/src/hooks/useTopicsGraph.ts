/**
 * useTopicsGraph - shared topics-graph fetch with module-level cache.
 *
 * Multiple consumers (TopicsGraphPanel, TeamEditorPanel's validation badge)
 * subscribe to the same per-teamId entry, so the graph is fetched at most
 * once per team selection. `refresh()` forces a re-fetch.
 */

import { useCallback, useEffect, useState } from 'react'
import * as memberFlowService from '@/services/memberFlowService'
import type { TopicsGraphResponse } from '@/types/topicsGraph'

interface CacheEntry {
  graph: TopicsGraphResponse | null
  loading: boolean
  error: string | null
  promise: Promise<void> | null
  listeners: Set<() => void>
}

const cache = new Map<string, CacheEntry>()

function getEntry(teamId: string): CacheEntry {
  let entry = cache.get(teamId)
  if (!entry) {
    entry = {
      graph: null,
      loading: false,
      error: null,
      promise: null,
      listeners: new Set(),
    }
    cache.set(teamId, entry)
  }
  return entry
}

function notify(entry: CacheEntry): void {
  for (const listener of entry.listeners) listener()
}

async function fetchInto(teamId: string, force: boolean): Promise<void> {
  const entry = getEntry(teamId)
  if (entry.loading && entry.promise) {
    return entry.promise
  }
  if (entry.graph && !force) return
  entry.loading = true
  entry.error = null
  notify(entry)
  const promise = (async () => {
    try {
      const data = await memberFlowService.getTopicsGraph(teamId)
      entry.graph = data
    } catch (err) {
      entry.error = err instanceof Error ? err.message : String(err)
    } finally {
      entry.loading = false
      entry.promise = null
      notify(entry)
    }
  })()
  entry.promise = promise
  return promise
}

export interface UseTopicsGraphResult {
  graph: TopicsGraphResponse | null
  loading: boolean
  error: string | null
  refresh: () => Promise<void>
}

export function useTopicsGraph(teamId: string | undefined | null): UseTopicsGraphResult {
  const [, setTick] = useState(0)

  useEffect(() => {
    if (!teamId) return
    const entry = getEntry(teamId)
    const listener = () => setTick((t) => t + 1)
    entry.listeners.add(listener)
    if (!entry.graph && !entry.loading) {
      void fetchInto(teamId, false)
    }
    return () => {
      entry.listeners.delete(listener)
    }
  }, [teamId])

  const refresh = useCallback(async () => {
    if (!teamId) return
    await fetchInto(teamId, true)
  }, [teamId])

  if (!teamId) {
    return { graph: null, loading: false, error: null, refresh }
  }
  const entry = getEntry(teamId)
  return {
    graph: entry.graph,
    loading: entry.loading,
    error: entry.error,
    refresh,
  }
}

/** Test-only: clear the module-level cache so unit tests stay isolated. */
export function __resetTopicsGraphCache(): void {
  cache.clear()
}
