import { useCallback, useEffect, useState } from 'react'
import type { ScenarioCatalogSnapshot, ScenarioVisibilityPayload } from '../types/techTree'
import {
  fetchScenarioCatalog,
  refreshScenarioCatalog,
  updateScenarioVisibility
} from '../services/techTree'

const defaultCatalogState: ScenarioCatalogSnapshot = {
  scenarios: [],
  edges: [],
  hidden: [],
  last_synced: null
}

/**
 * Hook for managing the scenario catalog.
 * Handles fetching, refreshing, and updating visibility of scenarios.
 */
const useScenarioCatalog = () => {
  const [scenarioCatalog, setScenarioCatalog] = useState<ScenarioCatalogSnapshot>(defaultCatalogState)
  const [catalogLoading, setCatalogLoading] = useState(true)
  const [isSyncingCatalog, setIsSyncingCatalog] = useState(false)
  const [catalogNotice, setCatalogNotice] = useState<string | null>(null)

  const loadScenarioCatalog = useCallback(async () => {
    setCatalogLoading(true)
    try {
      const snapshot = await fetchScenarioCatalog()
      setScenarioCatalog(snapshot || defaultCatalogState)
    } catch (error) {
      console.warn('Failed to load scenario catalog, using fallback', error)
      setScenarioCatalog(defaultCatalogState)
    } finally {
      setCatalogLoading(false)
    }
  }, [])

  useEffect(() => {
    loadScenarioCatalog()
  }, [loadScenarioCatalog])

  const handleScenarioCatalogRefresh = useCallback(async () => {
    if (isSyncingCatalog) {
      return
    }
    setIsSyncingCatalog(true)
    setCatalogNotice(null)
    try {
      await refreshScenarioCatalog()
      await loadScenarioCatalog()
      setCatalogNotice('Scenario catalog synced successfully!')
      // Clear success message after 3 seconds
      setTimeout(() => {
        setCatalogNotice(null)
      }, 3000)
    } catch (error) {
      console.error('Scenario catalog refresh failed', error)
      setCatalogNotice('Scenario catalog refresh failed. Using cached data.')
    } finally {
      setIsSyncingCatalog(false)
    }
  }, [isSyncingCatalog, loadScenarioCatalog])

  const handleScenarioVisibility = useCallback(
    async (scenarioName: string, hidden: boolean) => {
      if (!scenarioName) {
        return
      }
      try {
        await updateScenarioVisibility({ scenario: scenarioName, hidden })
        await loadScenarioCatalog()
      } catch (error) {
        console.error('Scenario visibility update failed', error)
        setCatalogNotice('Unable to update scenario visibility right now.')
      }
    },
    [loadScenarioCatalog]
  )

  return {
    scenarioCatalog,
    catalogLoading,
    isSyncingCatalog,
    catalogNotice,
    handleScenarioCatalogRefresh,
    handleScenarioVisibility
  }
}

export default useScenarioCatalog
