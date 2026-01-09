import React, { useMemo, useCallback } from 'react'
import ReactFlow, {
  Background,
  Controls,
  MiniMap,
  type Connection,
  type EdgeChange,
  type Node,
  type NodeChange,
  type NodeTypes
} from 'react-flow-renderer'
import { EDGE_LABEL_STYLE, EDGE_STYLE } from '../../utils/constants'
import SectorGroupNode from './SectorGroupNode'
import StageNodeComponent from './StageNode'
import ContextMenu from './ContextMenu'
import type { DesignerEdge, DesignerNode, DesignerNodeData, ScenarioNodeData, StageNode } from '../../types/graph'
import type { ContextMenuSection } from './ContextMenu'

interface GraphCanvasProps {
  nodes: DesignerNode[]
  edges: DesignerEdge[]
  isEditMode: boolean
  scenarioOnlyMode: boolean
  onNodesChange: (changes: NodeChange[]) => void
  onEdgesChange: (changes: EdgeChange[]) => void
  onConnect: (connection: Connection) => void
  onNodeClick: (_: unknown, node: Node<DesignerNodeData>) => void
  onPaneClick: () => void
  onExpandSector?: (sectorId: string) => void
  contextMenuVisible?: boolean
  contextMenuX?: number
  contextMenuY?: number
  contextMenuSections?: ContextMenuSection[]
  onContextMenuClose?: () => void
}

/**
 * Pure React Flow graph rendering component.
 * Handles only the visual presentation of the graph, no state management.
 */
const GraphCanvas: React.FC<GraphCanvasProps> = ({
  nodes,
  edges,
  isEditMode,
  scenarioOnlyMode,
  onNodesChange,
  onEdgesChange,
  onConnect,
  onNodeClick,
  onPaneClick,
  onExpandSector,
  contextMenuVisible,
  contextMenuX,
  contextMenuY,
  contextMenuSections,
  onContextMenuClose
}) => {
  // Register custom node types
  const nodeTypes: NodeTypes = useMemo(
    () => ({
      sectorGroup: SectorGroupNode,
      stageNode: StageNodeComponent
    }),
    []
  )

  // Debug logging
  React.useEffect(() => {
    console.log('[GraphCanvas] Rendering with nodes:', nodes.length, 'edges:', edges.length)
    if (nodes.length > 0) {
      console.log('[GraphCanvas] First 3 nodes:', nodes.slice(0, 3))
    }
  }, [nodes, edges])

  // Handle expand-sector custom event from SectorGroupNode
  React.useEffect(() => {
    if (!onExpandSector) return

    const handleExpandEvent = (event: Event) => {
      const customEvent = event as CustomEvent<{ sectorId: string }>
      if (customEvent.detail?.sectorId) {
        onExpandSector(customEvent.detail.sectorId)
      }
    }

    window.addEventListener('expand-sector', handleExpandEvent)
    return () => window.removeEventListener('expand-sector', handleExpandEvent)
  }, [onExpandSector])

  return (
    <div className="tech-tree-flow">
      <ReactFlow
        nodeTypes={nodeTypes}
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onNodeClick={onNodeClick}
        onPaneClick={onPaneClick}
        nodesDraggable={isEditMode && !scenarioOnlyMode}
        nodesConnectable={isEditMode && !scenarioOnlyMode}
        elementsSelectable={isEditMode}
        panOnDrag={!isEditMode}
        defaultEdgeOptions={{
          style: EDGE_STYLE,
          labelStyle: EDGE_LABEL_STYLE,
          labelBgPadding: [4, 2]
        }}
        fitView={nodes.length > 0}
        fitViewOptions={{ padding: 0.2, minZoom: 0.4, maxZoom: 1.4 }}
      >
        <MiniMap
          className="tech-tree-minimap"
          nodeColor={(node) => {
            if (typeof node?.id === 'string' && node.id.startsWith('scenario::')) {
              const scenarioData = node.data as ScenarioNodeData | undefined
              return scenarioData?.hidden ? '#475569' : '#38bdf8'
            }
            const stageData = node?.data as StageNode['data'] | undefined
            return stageData?.sectorColor || '#1e293b'
          }}
          maskColor="rgba(15, 23, 42, 0.65)"
        />
        <Controls showInteractive={false} />
        <Background gap={18} color="rgba(30, 41, 59, 0.35)" />
      </ReactFlow>

      {contextMenuVisible && contextMenuSections && onContextMenuClose && (
        <ContextMenu
          x={contextMenuX || 0}
          y={contextMenuY || 0}
          sections={contextMenuSections}
          onClose={onContextMenuClose}
        />
      )}
    </div>
  )
}

export default GraphCanvas
