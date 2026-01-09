import { useCallback, useEffect, useState } from 'react'
import type { Node } from 'react-flow-renderer'
import type { DesignerNodeData, ScenarioNodeData } from '../types/graph'

interface UseGraphSelectionParams {
  isEditMode: boolean
  isFullscreen: boolean
  isLayoutFullscreen: boolean
  isCompactLayout: boolean
}

interface UseGraphSelectionResult {
  selectedStageId: string | null
  selectedScenarioName: string | null
  shouldShowDialog: boolean
  setSelectedStageId: (id: string | null) => void
  setSelectedScenarioName: (name: string | null) => void
  handleNodeClick: (_: unknown, node: Node<DesignerNodeData>) => void
  handleCloseDetail: () => void
}

/**
 * Custom hook for managing node selection state and interaction handlers.
 * Handles both stage nodes and scenario nodes with appropriate dialog visibility.
 */
export const useGraphSelection = ({
  isEditMode,
  isFullscreen,
  isLayoutFullscreen,
  isCompactLayout
}: UseGraphSelectionParams): UseGraphSelectionResult => {
  const [selectedStageId, setSelectedStageId] = useState<string | null>(null)
  const [selectedScenarioName, setSelectedScenarioName] = useState<string | null>(null)

  const shouldShowDialog = Boolean(isFullscreen || isLayoutFullscreen || isCompactLayout)

  const handleNodeClick = useCallback(
    (_: unknown, node: Node<DesignerNodeData>) => {
      if (isEditMode || !node) {
        return
      }

      // Scenario nodes have IDs starting with 'scenario::'
      if (typeof node.id === 'string' && node.id.startsWith('scenario::')) {
        const scenarioName =
          (node.data as ScenarioNodeData | undefined)?.scenarioName ||
          node.id.replace('scenario::', '')
        setSelectedScenarioName(scenarioName)
        setSelectedStageId(null)
        return
      }

      // Regular stage nodes
      setSelectedScenarioName(null)
      setSelectedStageId(node.id || null)
    },
    [isEditMode]
  )

  const handleCloseDetail = useCallback(() => {
    setSelectedStageId(null)
    setSelectedScenarioName(null)
  }, [])

  // Keyboard shortcut for closing dialogs (ESC key)
  useEffect(() => {
    if (
      !shouldShowDialog ||
      (!selectedStageId && !selectedScenarioName) ||
      typeof document === 'undefined'
    ) {
      return undefined
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        handleCloseDetail()
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [handleCloseDetail, selectedScenarioName, selectedStageId, shouldShowDialog])

  return {
    selectedStageId,
    selectedScenarioName,
    shouldShowDialog,
    setSelectedStageId,
    setSelectedScenarioName,
    handleNodeClick,
    handleCloseDetail
  }
}
