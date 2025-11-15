import React, { useCallback, useEffect, useMemo } from 'react'
import { useEdgesState, useNodesState, type Node } from 'react-flow-renderer'
import { useGraphContext } from '../../contexts/GraphContext'
import { useGraphModel } from '../../hooks/useGraphModel'
import { useScenarioGraphs } from '../../hooks/useScenarioGraphs'
import { useGraphEditor } from '../../hooks/useGraphEditor'
import { useGraphSelection } from '../../hooks/useGraphSelection'
import { useGraphInteraction } from '../../hooks/useGraphInteraction'
import { useResponsiveLayout } from '../../hooks/useResponsiveLayout'
import { useHierarchicalGraph } from '../../hooks/useHierarchicalGraph'
import { useContextMenu } from '../../hooks/useContextMenu'
import { useNodeExpansion } from '../../hooks/useNodeExpansion'
import GraphCanvas from './GraphCanvas'
import GraphControls from './GraphControls'
import GraphNotifications from './GraphNotifications'
import type { ProgressionStage } from '../../types/techTree'
import type {
  DesignerEdge,
  DesignerNode,
  StageNode,
  StageNodeData
} from '../../types/graph'

interface GraphViewProps {
  graphNotice: string | null
  techTreeCanvasRef: React.RefObject<HTMLDivElement>
  onOpenStageDialog?: (stage: ProgressionStage) => void
  onOpenScenarioDialog?: (scenarioName: string) => void
}

/**
 * GraphView - Main graph interaction component.
 * Orchestrates all graph-related hooks and renders the canvas with controls.
 * Extracted from TechTreeCanvas to reduce complexity and improve maintainability.
 *
 * REFACTORED: Uses GraphContext to access shared data instead of prop drilling.
 */
const GraphView: React.FC<GraphViewProps> = ({
  graphNotice,
  techTreeCanvasRef,
  onOpenStageDialog,
  onOpenScenarioDialog
}) => {
  // Access graph data and operations from context
  const {
    sectors,
    dependencies,
    scenarioCatalog,
    scenarioEntryMap,
    selectedTreeId,
    showLiveScenarios,
    scenarioOnlyMode,
    showHiddenScenarios,
    isFullscreen,
    isLayoutFullscreen,
    canFullscreen,
    onGraphPersist,
    setGraphNotice,
    buildTreeAwarePath,
    toggleFullscreen,
    onLinkScenario,
    onCreateSector,
    onCreateStage,
    onGenerateAIStageIdeas,
    onDeleteStage,
    onDeleteSector,
    onUnlinkScenario,
    handleScenarioVisibility
  } = useGraphContext()
  // Build initial graph model from sectors and dependencies
  const { initialNodes, initialEdges, stageLookup } = useGraphModel(sectors, dependencies)

  // React Flow state management
  const [nodes, setNodes, onNodesStateChange] = useNodesState<StageNodeData>(
    initialNodes as Node<StageNodeData>[]
  )
  const [edges, setEdges, onEdgesStateChange] = useEdgesState(initialEdges as DesignerEdge[])

  const stageNodes = nodes as StageNode[]
  const designerEdges = edges as DesignerEdge[]

  // Node expansion management (for hierarchical children loading)
  const handleChildrenLoaded = useCallback(
    (parentStageId: string, children: ProgressionStage[]) => {
      if (!children || children.length === 0) {
        setGraphNotice('No children found for this stage')
        return
      }

      // Find parent node's sector for positioning and color
      const parentNode = stageNodes.find((n) => n.id === parentStageId)
      if (!parentNode) return

      const parentData = parentNode.data
      const sector = sectors.find((s) => s.id === parentData.sectorId)
      if (!sector) return

      // Calculate positions for children (stagger below parent)
      const parentX = parentNode.position.x
      const parentY = parentNode.position.y
      const childSpacing = 240 // horizontal spacing between children
      const verticalOffset = 150 // vertical offset from parent

      // Create nodes for children
      const childNodes: StageNode[] = children.map((child, index) => {
        const childX = parentX + (index - (children.length - 1) / 2) * childSpacing
        const childY = parentY + verticalOffset

        return {
          id: child.id,
          position: { x: childX, y: childY },
          data: {
            label: child.name,
            progress: child.progress_percentage || 0,
            type: child.stage_type,
            sectorName: sector.name,
            sectorColor: sector.color,
            sectorId: sector.id,
            stageId: child.id,
            hasChildren: child.has_children || false,
            childrenLoaded: false,
            parentStageId: parentStageId,
            isExpanded: false,
            isLoading: false
          },
          type: 'stageNode'
        }
      })

      // Update parent node to mark children as loaded
      setNodes((prevNodes) => {
        const updatedNodes = prevNodes.map((node) => {
          if (node.id === parentStageId) {
            return {
              ...node,
              data: {
                ...node.data,
                childrenLoaded: true
              }
            }
          }
          return node
        })

        // Add child nodes
        return [...updatedNodes, ...childNodes] as Node<StageNodeData>[]
      })

      // Create edges from parent to children
      const parentChildEdges = children.map((child) => ({
        id: `${parentStageId}->${child.id}`,
        source: parentStageId,
        target: child.id,
        type: 'default',
        style: { stroke: sector.color || '#3B82F6', strokeWidth: 2 },
        label: 'child',
        data: {
          dependencyType: 'parent-child',
          dependencyStrength: 1,
          description: 'Hierarchical parent-child relationship'
        }
      }))

      setEdges((prevEdges) => [...prevEdges, ...parentChildEdges] as DesignerEdge[])
      setGraphNotice(`Loaded ${children.length} child stage${children.length > 1 ? 's' : ''}`)
    },
    [stageNodes, sectors, setNodes, setEdges, setGraphNotice]
  )

  const { expandedNodes, loadingNodes, toggleNodeExpansion } = useNodeExpansion({
    selectedTreeId,
    onChildrenLoaded: handleChildrenLoaded,
    onError: (error) => setGraphNotice(`Error loading children: ${error}`)
  })

  // Hierarchical graph management (collapsible sectors)
  const { collapsedSectors, toggleSectorCollapse, filteredStageNodes, sectorGroupNodes } =
    useHierarchicalGraph({
      sectors,
      stageNodes
    })

  // Generate scenario overlay nodes/edges
  const { overlayNodes, overlayEdges, scenarioOnlyNodes, scenarioOnlyEdges } = useScenarioGraphs({
    scenarioOnlyMode,
    showLiveScenarios,
    scenarioCatalog,
    scenarioEntryMap,
    showHiddenScenarios,
    nodes: stageNodes,
    sectors
  })

  // Merge stage nodes with scenario nodes and inject expansion state
  const graphNodes: DesignerNode[] = useMemo(() => {
    // Inject expansion state and toggle handler into stage nodes
    const enrichedStageNodes = filteredStageNodes.map((node) => ({
      ...node,
      data: {
        ...node.data,
        isExpanded: expandedNodes.has(node.id),
        isLoading: loadingNodes.has(node.id),
        onToggleExpand: toggleNodeExpansion
      }
    }))

    if (scenarioOnlyMode) {
      return scenarioOnlyNodes
    }
    return [
      ...(enrichedStageNodes as unknown as DesignerNode[]),
      ...sectorGroupNodes,
      ...overlayNodes
    ]
  }, [
    overlayNodes,
    scenarioOnlyMode,
    scenarioOnlyNodes,
    filteredStageNodes,
    sectorGroupNodes,
    expandedNodes,
    loadingNodes,
    toggleNodeExpansion
  ])

  const graphEdges: DesignerEdge[] = useMemo(() => {
    if (scenarioOnlyMode) {
      return scenarioOnlyEdges
    }
    return [...designerEdges, ...overlayEdges]
  }, [designerEdges, overlayEdges, scenarioOnlyEdges, scenarioOnlyMode])

  // Edit mode management
  const {
    isEditMode,
    hasGraphChanges,
    isPersisting,
    editError,
    setHasGraphChanges,
    setEditError,
    handleToggleEditMode
  } = useGraphEditor({
    stageNodes,
    designerEdges,
    scenarioOnlyMode,
    buildTreeAwarePath,
    setEdges,
    onGraphPersist,
    setGraphNotice
  })

  // Responsive layout detection
  const { isCompactLayout } = useResponsiveLayout(1120)

  // Node selection management
  const {
    selectedStageId,
    selectedScenarioName,
    setSelectedStageId,
    setSelectedScenarioName,
    handleNodeClick,
    handleCloseDetail
  } = useGraphSelection({
    isEditMode,
    isFullscreen,
    isLayoutFullscreen,
    isCompactLayout
  })

  // Graph interaction handlers (drag, connect, etc.)
  const { handleNodesChange, handleEdgesChange, handleConnect } = useGraphInteraction({
    isEditMode,
    scenarioOnlyMode,
    onNodesStateChange,
    onEdgesStateChange,
    setEdges,
    setHasGraphChanges,
    setEditError
  })

  // Context menu handlers
  const handleAddChildStage = useCallback((parentStageId: string, parentSectorId: string) => {
    const parentStage = stageLookup.get(parentStageId)
    const nextStageOrder = (parentStage?.stage_order || 0) + 1
    const stageTypeMap: Record<number, string> = {
      1: 'foundation',
      2: 'operational',
      3: 'analytics',
      4: 'integration',
      5: 'digital_twin'
    }
    const nextStageType = stageTypeMap[nextStageOrder] || 'foundation'
    onCreateStage?.(parentSectorId, nextStageType, parentStageId)
  }, [stageLookup, onCreateStage])

  const handleGenerateAISuggestions = useCallback((stageId: string, sectorId: string) => {
    onGenerateAIStageIdeas?.(sectorId)
  }, [onGenerateAIStageIdeas])

  const handleEditStage = useCallback((stageId: string) => {
    const stage = stageLookup.get(stageId)
    if (stage && onOpenStageDialog) {
      onOpenStageDialog(stage)
    }
  }, [stageLookup, onOpenStageDialog])

  const handleDeleteStage = useCallback((stageId: string) => {
    const stage = stageLookup.get(stageId)
    if (!stage) {
      return
    }
    if (onDeleteStage) {
      onDeleteStage(stageId, stage.name)
    } else {
      setGraphNotice(`Delete stage "${stage.name}" - handler not wired up yet`)
    }
  }, [stageLookup, onDeleteStage, setGraphNotice])

  const handleUnlinkScenario = useCallback((scenarioName: string) => {
    if (onUnlinkScenario) {
      onUnlinkScenario(scenarioName)
    } else {
      setGraphNotice(`Unlink scenario "${scenarioName}" - handler not wired up yet`)
    }
  }, [onUnlinkScenario, setGraphNotice])

  const handleAddStageToSector = useCallback((sectorId: string) => {
    onCreateStage?.(sectorId)
  }, [onCreateStage])

  // Context menu state and sections
  const { menuState, openMenu, closeMenu, buildMenuSections } = useContextMenu({
    isEditMode,
    scenarioOnlyMode,
    onAddChildStage: handleAddChildStage,
    onGenerateAISuggestions: handleGenerateAISuggestions,
    onLinkScenario,
    onEditStage: handleEditStage,
    onDeleteStage: handleDeleteStage,
    onToggleScenarioVisibility: handleScenarioVisibility,
    onUnlinkScenario: handleUnlinkScenario,
    onExpandSector: toggleSectorCollapse,
    onAddStageToSector: handleAddStageToSector,
    onAddSector: onCreateSector,
    stageLookup
  })

  const menuSections = buildMenuSections()

  // Handle right-click to open context menu
  const handleContextMenu = useCallback((event: React.MouseEvent) => {
    event.preventDefault()
    const target = event.target as HTMLElement
    const nodeElement = target.closest('[data-id]')

    if (nodeElement) {
      const nodeId = nodeElement.getAttribute('data-id')
      const clickedNode = graphNodes.find(n => n.id === nodeId)
      openMenu(event.clientX, event.clientY, clickedNode || null)
    } else {
      openMenu(event.clientX, event.clientY, null)
    }
  }, [graphNodes, openMenu])

  // Sync nodes/edges when source data changes (but not during editing)
  useEffect(() => {
    if (isEditMode && hasGraphChanges) {
      return
    }
    setNodes(initialNodes as Node<StageNodeData>[])
  }, [initialNodes, isEditMode, hasGraphChanges, setNodes])

  useEffect(() => {
    if (isEditMode && hasGraphChanges) {
      return
    }
    setEdges(initialEdges as DesignerEdge[])
  }, [initialEdges, isEditMode, hasGraphChanges, setEdges])

  // Clear selection when entering edit mode
  useEffect(() => {
    if (isEditMode) {
      setSelectedStageId(null)
      setSelectedScenarioName(null)
    }
  }, [isEditMode, setSelectedScenarioName, setSelectedStageId])

  // Notify parent about selection changes for side panel
  useEffect(() => {
    if (selectedStageId) {
      const stage = stageLookup.get(selectedStageId)
      if (stage) {
        onOpenStageDialog?.(stage)
      }
    } else if (selectedScenarioName) {
      onOpenScenarioDialog?.(selectedScenarioName)
    }
  }, [selectedStageId, selectedScenarioName, stageLookup, onOpenStageDialog, onOpenScenarioDialog])

  return (
    <div
      ref={techTreeCanvasRef}
      className={[
        'tech-tree-canvas',
        isFullscreen && 'tech-tree-canvas--fullscreen',
        isLayoutFullscreen && 'tech-tree-canvas--immersive'
      ]
        .filter(Boolean)
        .join(' ')}
      role="application"
      aria-label="Tech tree graph"
    >
      <div className="tech-tree-canvas__header">
        <div>
          <h2>Tech Tree Graph</h2>
          <p>
            Navigate nodes, inspect dependencies, or toggle edit mode to restructure the graph
            directly within the designer.
          </p>
        </div>
        <GraphControls
          isFullscreen={isFullscreen}
          canFullscreen={canFullscreen}
          isEditMode={isEditMode}
          isPersisting={isPersisting}
          hasGraphChanges={hasGraphChanges}
          onToggleFullscreen={toggleFullscreen}
          onToggleEditMode={handleToggleEditMode}
        />
      </div>

      <GraphNotifications
        editError={editError}
        isEditMode={isEditMode}
        hasGraphChanges={hasGraphChanges}
        graphNotice={graphNotice}
      />

      <div onContextMenu={handleContextMenu}>
        <GraphCanvas
          nodes={graphNodes}
          edges={graphEdges}
          isEditMode={isEditMode}
          scenarioOnlyMode={scenarioOnlyMode}
          onNodesChange={handleNodesChange}
          onEdgesChange={handleEdgesChange}
          onConnect={handleConnect}
          onNodeClick={handleNodeClick}
          onPaneClick={handleCloseDetail}
          onExpandSector={toggleSectorCollapse}
          contextMenuVisible={menuState.visible}
          contextMenuX={menuState.x}
          contextMenuY={menuState.y}
          contextMenuSections={menuSections}
          onContextMenuClose={closeMenu}
        />
      </div>
    </div>
  )
}

export default GraphView
