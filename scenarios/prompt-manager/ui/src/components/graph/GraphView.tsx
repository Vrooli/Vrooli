/**
 * GraphView - Main React Flow canvas for the dependency graph.
 *
 * Features:
 * - Dagre hierarchical layout
 * - Custom node shapes by type
 * - Health-based coloring
 * - Click node -> navigate to editor
 * - Toolbar with filters, layout, zoom controls
 * - Query panel for highlighting subsets
 * - Legend
 * - Tooltip on hover
 */

import { useCallback, useMemo, useEffect, useState, useRef } from 'react'
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  Panel,
  MarkerType,
  useReactFlow,
  ReactFlowProvider,
  type Node,
  type Edge,
  type NodeMouseHandler,
} from '@xyflow/react'
import dagre from '@dagrejs/dagre'
import { cn } from '@/lib/utils'
import { useGraphStore, selectFilteredNodes } from '@/stores/graphStore'
import { useShallow } from 'zustand/react/shallow'
import { useSelectionStore } from '@/stores/selectionStore'
import { GraphFlowNode, type GraphNodeData } from './GraphNode'
import { GraphToolbar } from './GraphToolbar'
import { GraphQueryPanel } from './GraphQueryPanel'
import { GraphLegend } from './GraphLegend'
import { GraphNodeTooltip } from './GraphNodeTooltip'
import { PanelErrorBoundary } from '../PanelErrorBoundary'
import type { GraphNode as GraphNodeType, HealthScore } from '@/lib/schemas'

import '@xyflow/react/dist/style.css'

// ============================================================================
// Constants
// ============================================================================

const NODE_WIDTH = 160
const NODE_HEIGHT = 80
type LayoutDirection = 'TB' | 'LR'

// ============================================================================
// Node Types
// ============================================================================

const nodeTypes = {
  graphNode: GraphFlowNode,
}

// ============================================================================
// Edge kind styling
// ============================================================================

const EDGE_STYLES: Record<string, { stroke: string; strokeDasharray?: string }> = {
  membership: { stroke: 'hsl(var(--muted-foreground))' },
  'cli-read': { stroke: '#8b5cf6' },
  'bold-listed': { stroke: '#8b5cf6', strokeDasharray: '5 3' },
  'path-ref': { stroke: '#a855f7' },
  'default-scope': { stroke: '#3b82f6', strokeDasharray: '8 4' },
  'code-usage': { stroke: '#f97316' },
}

// ============================================================================
// Layout
// ============================================================================

type FlowNode = Node<GraphNodeData, 'graphNode'>
type FlowEdge = Edge

function getLayoutedElements(
  nodes: FlowNode[],
  edges: FlowEdge[],
  direction: LayoutDirection = 'TB',
): { nodes: FlowNode[]; edges: FlowEdge[] } {
  const dagreGraph = new dagre.graphlib.Graph()
  dagreGraph.setDefaultEdgeLabel(() => ({}))
  dagreGraph.setGraph({ rankdir: direction, nodesep: 60, ranksep: 100 })

  nodes.forEach((node) => {
    dagreGraph.setNode(node.id, { width: NODE_WIDTH, height: NODE_HEIGHT })
  })

  edges.forEach((edge) => {
    dagreGraph.setEdge(edge.source, edge.target)
  })

  dagre.layout(dagreGraph)

  const layoutedNodes = nodes.map((node) => {
    const pos = dagreGraph.node(node.id)
    return {
      ...node,
      position: {
        x: pos.x - NODE_WIDTH / 2,
        y: pos.y - NODE_HEIGHT / 2,
      },
    }
  })

  return { nodes: layoutedNodes, edges }
}

// ============================================================================
// Inner Component (needs ReactFlowProvider context)
// ============================================================================

interface GraphViewInnerProps {
  className?: string
}

function GraphViewInner({ className }: GraphViewInnerProps) {
  const { fitView } = useReactFlow()
  const [layoutDirection, setLayoutDirection] = useState<LayoutDirection>('TB')
  const [hoveredNode, setHoveredNode] = useState<{ node: GraphNodeType; healthScore?: HealthScore; x: number; y: number } | null>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  // Store data
  const graph = useGraphStore((s) => s.graph)
  const loading = useGraphStore((s) => s.loading)
  const error = useGraphStore((s) => s.error)
  const fetchGraph = useGraphStore((s) => s.fetchGraph)
  const highlightedNodeIds = useGraphStore((s) => s.highlightedNodeIds)
  // useShallow prevents infinite re-render: selectFilteredNodes returns a new
  // array reference on every call (.filter()), but useShallow compares elements
  // by identity so the result is stable when the underlying data hasn't changed.
  const filteredNodes = useGraphStore(useShallow(selectFilteredNodes))

  // Selection store for navigation
  const setSelectedSkillId = useSelectionStore((s) => s.setSelectedSkillId)
  const setSelectedAgentId = useSelectionStore((s) => s.setSelectedAgentId)
  const setSelectedTeamId = useSelectionStore((s) => s.setSelectedTeamId)

  // Fetch on mount
  useEffect(() => {
    void fetchGraph()
  }, [fetchGraph])

  // Build node ID set for filtering edges
  const filteredNodeIds = useMemo(() => new Set(filteredNodes.map((n) => n.id)), [filteredNodes])

  // Map for quick node lookup
  const nodeMap = useMemo(() => {
    const map = new Map<string, GraphNodeType>()
    if (graph) {
      for (const n of graph.graph.nodes) {
        map.set(n.id, n)
      }
    }
    return map
  }, [graph])

  // Build health score map from graph data
  const healthMap = useMemo(() => {
    const map = new Map<string, number>()
    if (graph) {
      for (const hs of graph.graph.healthScores) {
        map.set(hs.nodeId, hs.score)
      }
    }
    return map
  }, [graph])

  // Build flow nodes
  const flowNodes = useMemo((): FlowNode[] => {
    return filteredNodes.map((node): FlowNode => ({
      id: node.id,
      type: 'graphNode',
      position: { x: 0, y: 0 },
      data: {
        label: node.label,
        nodeType: node.type,
        healthScore: healthMap.get(node.id) ?? null,
        isHighlighted: highlightedNodeIds.has(node.id),
      },
    }))
  }, [filteredNodes, highlightedNodeIds, healthMap])

  // Build flow edges (only between visible nodes)
  const flowEdges = useMemo((): FlowEdge[] => {
    if (!graph) return []
    return graph.graph.edges
      .filter((e) => filteredNodeIds.has(e.from) && filteredNodeIds.has(e.to))
      .map((edge, i): FlowEdge => {
        const defaultStyle: { stroke: string; strokeDasharray?: string } = { stroke: 'hsl(var(--muted-foreground))' }
        const edgeStyle = EDGE_STYLES[edge.kind] ?? defaultStyle
        return {
          id: `e-${edge.from}-${edge.to}-${edge.kind}-${i}`,
          source: edge.from,
          target: edge.to,
          type: 'smoothstep',
          animated: edge.kind === 'default-scope',
          markerEnd: {
            type: MarkerType.ArrowClosed,
            width: 16,
            height: 16,
            color: edgeStyle.stroke,
          },
          style: {
            stroke: edgeStyle.stroke,
            strokeWidth: 1.5,
            strokeDasharray: edgeStyle.strokeDasharray,
          },
        }
      })
  }, [graph, filteredNodeIds])

  // Apply layout
  const { nodes: layoutedNodes, edges: layoutedEdges } = useMemo(
    () => getLayoutedElements(flowNodes, flowEdges, layoutDirection),
    [flowNodes, flowEdges, layoutDirection],
  )

  const [nodes, setNodes, onNodesChange] = useNodesState<FlowNode>(layoutedNodes)
  const [edges, setEdges, onEdgesChange] = useEdgesState<FlowEdge>(layoutedEdges)

  // Update when layout/data changes
  useEffect(() => {
    setNodes(layoutedNodes)
  }, [layoutedNodes, setNodes])

  useEffect(() => {
    setEdges(layoutedEdges)
  }, [layoutedEdges, setEdges])

  // Handle node click -> navigate
  const onNodeClick = useCallback<NodeMouseHandler>((_event, node) => {
    const graphNode = nodeMap.get(node.id)
    if (!graphNode) return

    if (graphNode.type === 'skill') {
      setSelectedSkillId(graphNode.id)
    } else if (graphNode.type === 'agent') {
      setSelectedAgentId(graphNode.id)
    } else if (graphNode.type === 'team') {
      setSelectedTeamId(graphNode.id)
    }
  }, [nodeMap, setSelectedSkillId, setSelectedAgentId, setSelectedTeamId])

  // Build health score map for tooltip
  const healthScoreMap = useMemo(() => {
    const map = new Map<string, HealthScore>()
    if (graph) {
      for (const hs of graph.graph.healthScores) {
        map.set(hs.nodeId, hs)
      }
    }
    return map
  }, [graph])

  // Handle node hover -> tooltip
  const onNodeMouseEnter = useCallback<NodeMouseHandler>((_event, node) => {
    const graphNode = nodeMap.get(node.id)
    if (!graphNode) return
    const nodePos = node.position
    setHoveredNode({
      node: graphNode,
      healthScore: healthScoreMap.get(node.id),
      x: nodePos.x + NODE_WIDTH + 20,
      y: nodePos.y,
    })
  }, [nodeMap, healthScoreMap])

  const onNodeMouseLeave = useCallback(() => {
    setHoveredNode(null)
  }, [])

  const handleToggleLayout = useCallback(() => {
    setLayoutDirection((d) => (d === 'TB' ? 'LR' : 'TB'))
  }, [])

  const handleFitView = useCallback(() => {
    void fitView({ padding: 0.2 })
  }, [fitView])

  // Loading / error states
  if (loading && !graph) {
    return (
      <div className={cn('h-full flex items-center justify-center', className)}>
        <div className="text-center text-muted-foreground">
          <div className="animate-spin h-8 w-8 border-2 border-primary border-t-transparent rounded-full mx-auto mb-3" />
          <p className="text-sm">Loading graph...</p>
        </div>
      </div>
    )
  }

  if (error && !graph) {
    return (
      <div className={cn('h-full flex items-center justify-center', className)}>
        <div className="text-center text-muted-foreground">
          <p className="text-sm text-destructive mb-2">Failed to load graph</p>
          <p className="text-xs">{error}</p>
          <button
            type="button"
            onClick={() => void fetchGraph(true)}
            className="mt-3 px-3 py-1.5 text-xs bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
          >
            Retry
          </button>
        </div>
      </div>
    )
  }

  if (!graph || graph.graph.nodes.length === 0) {
    return (
      <div className={cn('h-full flex items-center justify-center', className)}>
        <div className="text-center text-muted-foreground">
          <p className="text-sm mb-2">No graph data available</p>
          <p className="text-xs">The graph will be generated when teams, agents, and skills are configured.</p>
        </div>
      </div>
    )
  }

  return (
    <div ref={containerRef} className={cn('h-full relative', className)}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onNodeClick={onNodeClick}
        onNodeMouseEnter={onNodeMouseEnter}
        onNodeMouseLeave={onNodeMouseLeave}
        nodeTypes={nodeTypes}
        fitView
        fitViewOptions={{ padding: 0.2 }}
        minZoom={0.1}
        maxZoom={2}
        proOptions={{ hideAttribution: true }}
        className="bg-background"
      >
        <Background color="hsl(var(--border))" gap={20} />
        <Controls
          className="!bg-card !border-border !rounded-lg overflow-hidden"
          showInteractive={false}
        />
        <MiniMap
          className="!bg-card !border-border !rounded-lg"
          nodeColor={(node) => {
            const data = node.data as unknown as GraphNodeData
            switch (data.nodeType) {
              case 'team': return '#3b82f6'
              case 'agent': return '#10b981'
              case 'skill': return '#8b5cf6'
              default: return '#f97316'
            }
          }}
          maskColor="rgba(0, 0, 0, 0.5)"
        />

        {/* Toolbar */}
        <Panel position="top-right">
          <PanelErrorBoundary panelName="Graph Toolbar" minimal>
            <GraphToolbar
              layoutDirection={layoutDirection}
              onToggleLayout={handleToggleLayout}
              onFitView={handleFitView}
            />
          </PanelErrorBoundary>
        </Panel>

        {/* Legend + Query Panel */}
        <Panel position="top-left" className="flex flex-col gap-2 max-w-xs">
          <GraphLegend />
          <PanelErrorBoundary panelName="Graph Queries">
            <GraphQueryPanel />
          </PanelErrorBoundary>
        </Panel>
      </ReactFlow>

      {/* Tooltip overlay */}
      {hoveredNode && (
        <PanelErrorBoundary panelName="Graph Tooltip" minimal>
          <GraphNodeTooltip
            node={hoveredNode.node}
            healthScore={hoveredNode.healthScore}
            x={hoveredNode.x}
            y={hoveredNode.y}
          />
        </PanelErrorBoundary>
      )}
    </div>
  )
}

// ============================================================================
// Exported Component (wraps with ReactFlowProvider)
// ============================================================================

interface GraphViewProps {
  className?: string
}

export function GraphView({ className }: GraphViewProps) {
  return (
    <ReactFlowProvider>
      <GraphViewInner className={className} />
    </ReactFlowProvider>
  )
}
