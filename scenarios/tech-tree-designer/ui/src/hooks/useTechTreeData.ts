import { useCallback, useMemo } from 'react'
import useTechTreeSelection from './useTechTreeSelection'
import useTreeData from './useTreeData'
import useScenarioCatalog from './useScenarioCatalog'

/**
 * Orchestrator hook that combines tree selection, tree data, and scenario catalog management.
 * This hook previously handled all concerns directly (320 lines), but has been refactored
 * into three focused hooks for better maintainability.
 *
 * Migration: This hook maintains the exact same API as before, so no changes needed in App.tsx
 */
const useTechTreeData = () => {
  // Tree selection and list management
  const {
    techTrees,
    treeLoading,
    selectedTreeId,
    setSelectedTreeId,
    selectedTreeSummary,
    buildTreeAwarePath
  } = useTechTreeSelection()

  // Tree data (sectors, dependencies, milestones)
  const {
    sectors,
    milestones,
    dependencies,
    loading,
    graphNotice,
    setGraphNotice,
    refreshTreeData
  } = useTreeData({ selectedTreeId, treeLoading })

  // Scenario catalog management
  const {
    scenarioCatalog,
    catalogLoading,
    isSyncingCatalog,
    catalogNotice,
    handleScenarioCatalogRefresh,
    handleScenarioVisibility
  } = useScenarioCatalog()

  // Merge graph notice and catalog notice
  const combinedNotice = useMemo(() => {
    if (catalogNotice && graphNotice) {
      return `${graphNotice} | ${catalogNotice}`
    }
    return catalogNotice || graphNotice
  }, [catalogNotice, graphNotice])

  // Provide a unified setGraphNotice that handles both sources
  const setUnifiedGraphNotice = useCallback((notice: string | null) => {
    setGraphNotice(notice)
  }, [setGraphNotice])

  return {
    // Tree selection
    techTrees,
    treeLoading,
    selectedTreeId,
    setSelectedTreeId,
    selectedTreeSummary,
    buildTreeAwarePath,

    // Tree data
    sectors,
    milestones,
    dependencies,
    loading,
    graphNotice: combinedNotice,
    setGraphNotice: setUnifiedGraphNotice,
    refreshTreeData,

    // Scenario catalog
    scenarioCatalog,
    catalogLoading,
    isSyncingCatalog,
    handleScenarioCatalogRefresh,
    handleScenarioVisibility
  }
}

export default useTechTreeData
