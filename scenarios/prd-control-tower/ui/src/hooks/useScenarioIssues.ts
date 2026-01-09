import { useCallback, useEffect, useSyncExternalStore } from 'react'
import type { ScenarioIssuesEntry } from '../state/scenarioIssuesStore'
import { scenarioIssuesStore } from '../state/scenarioIssuesStore'

interface UseScenarioIssuesOptions {
  entityType?: string
  entityName?: string
  autoFetch?: boolean
}

interface UseScenarioIssuesResult extends ScenarioIssuesEntry {
  refresh: () => Promise<ScenarioIssuesEntry>
}

export function useScenarioIssues({ entityType, entityName, autoFetch = true }: UseScenarioIssuesOptions): UseScenarioIssuesResult {
  const subscribe = useCallback((listener: () => void) => scenarioIssuesStore.subscribe(listener), [])

  const getSnapshot = useCallback<() => ScenarioIssuesEntry>(
    () => scenarioIssuesStore.getEntry(entityType, entityName),
    [entityType, entityName],
  )

  const entry = useSyncExternalStore(subscribe, getSnapshot, getSnapshot)

  useEffect(() => {
    if (!autoFetch || !entityType || !entityName) {
      return
    }
    scenarioIssuesStore.fetch(entityType, entityName).catch(() => undefined)
  }, [autoFetch, entityType, entityName])

  const refresh = useCallback(() => {
    if (!entityType || !entityName) {
      return Promise.resolve(entry)
    }
    return scenarioIssuesStore.fetch(entityType, entityName, { force: true })
  }, [entityType, entityName, entry])

  return {
    ...entry,
    refresh,
  }
}
