import { useCallback, useState } from 'react'
import type { Node } from 'react-flow-renderer'
import type { ContextMenuSection } from '../components/graph/ContextMenu'
import type { DesignerNodeData } from '../types/graph'
import type { ProgressionStage } from '../types/techTree'
import { buildContextMenu, type MenuBuilderContext, type MenuCallbacks } from '../utils/contextMenuBuilders'

export interface ContextMenuState {
  visible: boolean
  x: number
  y: number
  nodeId: string | null
  nodeType: 'stage' | 'scenario' | 'sectorGroup' | 'pane' | null
  nodeData: DesignerNodeData | null
}

interface UseContextMenuParams extends MenuCallbacks {
  isEditMode: boolean
  scenarioOnlyMode: boolean
  stageLookup?: Map<string, ProgressionStage & { sector: any }>
}

/**
 * Hook for managing context menu state and generating menu items based on node type.
 * Delegates menu building to pure functions for better testability and maintainability.
 */
export const useContextMenu = (params: UseContextMenuParams) => {
  const [menuState, setMenuState] = useState<ContextMenuState>({
    visible: false,
    x: 0,
    y: 0,
    nodeId: null,
    nodeType: null,
    nodeData: null
  })

  const openMenu = useCallback(
    (x: number, y: number, node: Node<DesignerNodeData> | null) => {
      if (!node) {
        // Clicked on pane
        setMenuState({
          visible: true,
          x,
          y,
          nodeId: null,
          nodeType: 'pane',
          nodeData: null
        })
        return
      }

      const nodeId = node.id
      let nodeType: ContextMenuState['nodeType'] = null

      if (nodeId.startsWith('scenario::')) {
        nodeType = 'scenario'
      } else if (nodeId.startsWith('sector-group::')) {
        nodeType = 'sectorGroup'
      } else {
        nodeType = 'stage'
      }

      setMenuState({
        visible: true,
        x,
        y,
        nodeId,
        nodeType,
        nodeData: node.data
      })
    },
    []
  )

  const closeMenu = useCallback(() => {
    setMenuState((prev) => ({ ...prev, visible: false }))
  }, [])

  const buildMenuSections = useCallback((): ContextMenuSection[] => {
    if (!menuState.visible) {
      return []
    }

    const context: MenuBuilderContext = {
      isEditMode: params.isEditMode,
      scenarioOnlyMode: params.scenarioOnlyMode,
      stageLookup: params.stageLookup
    }

    const callbacks: MenuCallbacks = {
      onAddChildStage: params.onAddChildStage,
      onGenerateAISuggestions: params.onGenerateAISuggestions,
      onLinkScenario: params.onLinkScenario,
      onEditStage: params.onEditStage,
      onDeleteStage: params.onDeleteStage,
      onToggleScenarioVisibility: params.onToggleScenarioVisibility,
      onUnlinkScenario: params.onUnlinkScenario,
      onExpandSector: params.onExpandSector,
      onAddStageToSector: params.onAddStageToSector,
      onAddSector: params.onAddSector
    }

    return buildContextMenu(
      menuState.nodeType,
      menuState.nodeId,
      menuState.nodeData,
      context,
      callbacks
    )
  }, [menuState, params])

  return {
    menuState,
    openMenu,
    closeMenu,
    buildMenuSections
  }
}
