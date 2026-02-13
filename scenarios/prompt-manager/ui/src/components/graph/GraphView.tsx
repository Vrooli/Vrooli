/**
 * GraphView - Main React Flow canvas for the dependency graph.
 *
 * Features:
 * - Dagre hierarchical layout
 * - Custom node shapes by type
 * - Health-based coloring
 * - Click node -> detail popover with health breakdown + navigation
 * - Popover tracks node position during pan/zoom
 *
 * Query panel and settings/help controls are rendered externally via ViewOverlay.
 */

import { useCallback, useMemo, useEffect, useState, useRef, lazy, Suspense } from 'react'
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  MarkerType,
  useReactFlow,
  ReactFlowProvider,
  type Node,
  type Edge,
  type NodeMouseHandler,
  type OnMove,
} from '@xyflow/react'
import dagre from '@dagrejs/dagre'
import { Network, Braces } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useGraphStore, selectFilteredNodes, type GraphLayoutMode } from '@/stores/graphStore'
import { useShallow } from 'zustand/react/shallow'
import { useSelectionStore } from '@/stores/selectionStore'
import { GraphFlowNode, type GraphNodeData } from './GraphNode'
import { GraphNodePopover } from './GraphNodePopover'
import { PanelErrorBoundary } from '../PanelErrorBoundary'
import { selectors } from '@/constants/selectors'
import type { GraphNode as GraphNodeType, GraphEdge as GraphEdgeType, HealthScore } from '@/lib/schemas'

const GraphJsonView = lazy(() => import('./GraphJsonView').then((m) => ({ default: m.GraphJsonView })))

import '@xyflow/react/dist/style.css'

// ============================================================================
// Constants
// ============================================================================

const NODE_WIDTH = 160
const NODE_HEIGHT = 80
const CLI_CLUSTER_ID = '__pm_cli_cluster__'
const LOW_SIGNAL_EDGE_KINDS = new Set(['bold-listed', 'path-ref'])

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

function getBounds(nodes: FlowNode[]): { width: number; height: number } {
  if (nodes.length === 0) return { width: 0, height: 0 }
  let minX = Number.POSITIVE_INFINITY
  let maxX = Number.NEGATIVE_INFINITY
  let minY = Number.POSITIVE_INFINITY
  let maxY = Number.NEGATIVE_INFINITY
  for (const node of nodes) {
    minX = Math.min(minX, node.position.x)
    maxX = Math.max(maxX, node.position.x + NODE_WIDTH)
    minY = Math.min(minY, node.position.y)
    maxY = Math.max(maxY, node.position.y + NODE_HEIGHT)
  }
  return { width: Math.max(1, maxX - minX), height: Math.max(1, maxY - minY) }
}

function runDagreLayout(
  nodes: FlowNode[],
  edges: FlowEdge[],
  direction: 'TB' | 'LR',
  mode: Extract<GraphLayoutMode, 'hierarchical' | 'compact'>,
): { nodes: FlowNode[]; edges: FlowEdge[] } {
  const dagreGraph = new dagre.graphlib.Graph()
  dagreGraph.setDefaultEdgeLabel(() => ({}))
  dagreGraph.setGraph({
    rankdir: direction,
    nodesep: mode === 'compact' ? 35 : 60,
    ranksep: mode === 'compact' ? 60 : 100,
    ranker: mode === 'compact' ? 'tight-tree' : 'network-simplex',
  })

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

function getGroupedLayout(nodes: FlowNode[], edges: FlowEdge[], direction: 'TB' | 'LR'): { nodes: FlowNode[]; edges: FlowEdge[] } {
  const laneOrder: GraphNodeData['nodeType'][] = ['team', 'agent', 'skill', 'cli']
  const byLane = new Map<GraphNodeData['nodeType'], FlowNode[]>()
  for (const lane of laneOrder) byLane.set(lane, [])
  for (const node of nodes) {
    const lane = node.data?.nodeType ?? 'skill'
    const list = byLane.get(lane) ?? []
    list.push(node)
    byLane.set(lane, list)
  }

  const laneGap = 260
  const cellX = NODE_WIDTH + 60
  const cellY = NODE_HEIGHT + 60
  const layoutedNodes: FlowNode[] = []

  for (let laneIndex = 0; laneIndex < laneOrder.length; laneIndex++) {
    const lane = laneOrder[laneIndex] ?? 'skill'
    const laneNodes = byLane.get(lane) ?? []
    if (laneNodes.length === 0) continue
    const columns = Math.max(1, Math.ceil(Math.sqrt(laneNodes.length)))
    for (let i = 0; i < laneNodes.length; i++) {
      const laneNode = laneNodes[i]
      if (!laneNode) continue
      const col = i % columns
      const row = Math.floor(i / columns)
      const x = col * cellX
      const y = laneIndex * laneGap + row * cellY
      layoutedNodes.push({
        ...laneNode,
        position: direction === 'TB'
          ? { x, y }
          : { x: y, y: x },
      })
    }
  }
  return { nodes: layoutedNodes, edges }
}

function getLayoutedElements(
  nodes: FlowNode[],
  edges: FlowEdge[],
  direction: 'TB' | 'LR' = 'TB',
  mode: GraphLayoutMode = 'compact',
): { nodes: FlowNode[]; edges: FlowEdge[] } {
  if (mode === 'grouped') {
    return getGroupedLayout(nodes, edges, direction)
  }

  const primary = runDagreLayout(nodes, edges, direction, mode)
  const primaryBounds = getBounds(primary.nodes)
  const primaryRatio = primaryBounds.width / primaryBounds.height
  const tooElongated = primaryRatio > 3 || primaryRatio < 1 / 3
  if (!tooElongated) {
    return primary
  }

  const alternateDirection = direction === 'TB' ? 'LR' : 'TB'
  const alternate = runDagreLayout(nodes, edges, alternateDirection, mode)
  const alternateBounds = getBounds(alternate.nodes)
  const alternateRatio = alternateBounds.width / alternateBounds.height

  const primaryDistance = Math.abs(Math.log(primaryRatio))
  const alternateDistance = Math.abs(Math.log(alternateRatio))
  return alternateDistance < primaryDistance ? alternate : primary
}

// ============================================================================
// Inner Component (needs ReactFlowProvider context)
// ============================================================================

interface GraphViewInnerProps {
  className?: string
}

interface SelectedNodeState {
  nodeId: string
  node: GraphNodeType
  healthScore?: HealthScore
  edges: GraphEdgeType[]
  screenX: number
  screenY: number
}

function GraphViewInner({ className }: GraphViewInnerProps) {
  const { fitView, setViewport, getViewport, flowToScreenPosition } = useReactFlow()
  const [viewMode, setViewMode] = useState<'visual' | 'json'>('visual')
  const [selectedNode, setSelectedNode] = useState<SelectedNodeState | null>(null)
  const selectedNodeRef = useRef<SelectedNodeState | null>(null)
  const popoverRef = useRef<HTMLDivElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const hasInitializedViewport = useRef(false)

  // Store data
  const graph = useGraphStore((s) => s.graph)
  const loading = useGraphStore((s) => s.loading)
  const error = useGraphStore((s) => s.error)
  const fetchGraph = useGraphStore((s) => s.fetchGraph)
  const highlightedNodeIds = useGraphStore((s) => s.highlightedNodeIds)
  const filters = useGraphStore((s) => s.filters)
  const layoutDirection = useGraphStore((s) => s.layoutDirection)
  const layoutMode = useGraphStore((s) => s.layoutMode)
  const fitViewRequested = useGraphStore((s) => s.fitViewRequested)
  const savedViewport = useGraphStore((s) => s.viewport)
  const setSavedViewport = useGraphStore((s) => s.setViewport)
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

  // Watch for fit view requests from the store
  const prevFitView = useRef(fitViewRequested)
  useEffect(() => {
    if (fitViewRequested !== prevFitView.current) {
      prevFitView.current = fitViewRequested
      void (async () => {
        await fitView({ padding: 0.2 })
        setSavedViewport(getViewport())
      })()
    }
  }, [fitViewRequested, fitView, getViewport, setSavedViewport])

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

  // Apply edge filters and optional CLI collapsing before rendering.
  const viewModel = useMemo(() => {
    if (!graph) {
      return {
        nodes: [] as GraphNodeType[],
        edges: [] as GraphEdgeType[],
      }
    }

    let visibleNodes = filteredNodes
    const visibleNodeIDs = new Set(visibleNodes.map((n) => n.id))
    let visibleEdges = graph.graph.edges.filter((e) => visibleNodeIDs.has(e.from) && visibleNodeIDs.has(e.to))

    if (!filters.showLowSignalEdges) {
      visibleEdges = visibleEdges.filter((edge) => !LOW_SIGNAL_EDGE_KINDS.has(edge.kind))
    }

    if (!filters.collapseCLIs) {
      return {
        nodes: visibleNodes,
        edges: visibleEdges,
      }
    }

    const cliNodes = visibleNodes.filter((n) => n.type === 'cli')
    if (cliNodes.length === 0) {
      return {
        nodes: visibleNodes,
        edges: visibleEdges,
      }
    }

    const collapsedNode: GraphNodeType = {
      id: CLI_CLUSTER_ID,
      type: 'cli',
      label: `CLI Tools (${cliNodes.length})`,
      description: 'Collapsed CLI cluster',
      status: '',
      tags: [],
    }
    const replacement = new Map<string, string>()
    for (const cli of cliNodes) replacement.set(cli.id, CLI_CLUSTER_ID)

    const deduped = new Map<string, GraphEdgeType>()
    for (const edge of visibleEdges) {
      const source = replacement.get(edge.from) ?? edge.from
      const target = replacement.get(edge.to) ?? edge.to
      if (source === target) continue
      const dedupeKey = `${source}|${target}|${edge.kind}`
      if (!deduped.has(dedupeKey)) {
        deduped.set(dedupeKey, { ...edge, from: source, to: target })
      }
    }

    visibleNodes = [...visibleNodes.filter((n) => n.type !== 'cli'), collapsedNode]
    return {
      nodes: visibleNodes,
      edges: Array.from(deduped.values()),
    }
  }, [graph, filteredNodes, filters.collapseCLIs, filters.showLowSignalEdges])

  // Build node ID set for filtering edges
  const renderedNodeIds = useMemo(() => new Set(viewModel.nodes.map((n) => n.id)), [viewModel.nodes])

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
    return viewModel.nodes.map((node): FlowNode => ({
      id: node.id,
      type: 'graphNode',
      position: { x: 0, y: 0 },
      data: {
        label: node.label,
        nodeType: node.type,
        healthScore: healthMap.get(node.id) ?? null,
        isHighlighted: node.id === CLI_CLUSTER_ID
          ? Array.from(highlightedNodeIds).some((id) => id.startsWith('cli:'))
          : highlightedNodeIds.has(node.id),
      },
    }))
  }, [viewModel.nodes, highlightedNodeIds, healthMap])

  // Build flow edges (only between visible nodes)
  const flowEdges = useMemo((): FlowEdge[] => {
    return viewModel.edges
      .filter((e) => renderedNodeIds.has(e.from) && renderedNodeIds.has(e.to))
      .map((edge, i): FlowEdge => {
        const defaultStyle: { stroke: string; strokeDasharray?: string } = { stroke: 'hsl(var(--muted-foreground))' }
        const edgeStyle = EDGE_STYLES[edge.kind] ?? defaultStyle
        return {
          id: `e-${edge.from}-${edge.to}-${edge.kind}-${i}`,
          source: edge.from,
          target: edge.to,
          type: 'smoothstep',
          selectable: false,
          focusable: false,
          reconnectable: false,
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
            pointerEvents: 'none',
          },
        }
      })
  }, [viewModel.edges, renderedNodeIds])

  // Apply layout
  const { nodes: layoutedNodes, edges: layoutedEdges } = useMemo(
    () => getLayoutedElements(flowNodes, flowEdges, layoutDirection, layoutMode),
    [flowNodes, flowEdges, layoutDirection, layoutMode],
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

  const autoFitSignature = useMemo(() => {
    const highlighted = Array.from(highlightedNodeIds).sort().join('|')
    return [
      layoutMode,
      layoutDirection,
      filters.collapseCLIs ? 'collapse' : 'expand',
      filters.showLowSignalEdges ? 'low-on' : 'low-off',
      flowNodes.length,
      flowEdges.length,
      highlighted,
    ].join('::')
  }, [
    layoutMode,
    layoutDirection,
    filters.collapseCLIs,
    filters.showLowSignalEdges,
    flowNodes.length,
    flowEdges.length,
    highlightedNodeIds,
  ])
  const lastAutoFitSignature = useRef('')
  useEffect(() => {
    if (!filters.autoFitOnChange) return
    if (!hasInitializedViewport.current) return
    if (!graph || flowNodes.length === 0) return
    if (autoFitSignature === lastAutoFitSignature.current) return
    lastAutoFitSignature.current = autoFitSignature
    void (async () => {
      await fitView({ padding: 0.2, duration: 200 })
      setSavedViewport(getViewport())
    })()
  }, [
    autoFitSignature,
    filters.autoFitOnChange,
    graph,
    flowNodes.length,
    fitView,
    getViewport,
    setSavedViewport,
  ])

  // Restore persisted viewport once after nodes are available.
  useEffect(() => {
    if (!graph || nodes.length === 0 || hasInitializedViewport.current) return

    hasInitializedViewport.current = true

    if (savedViewport) {
      void setViewport(savedViewport, { duration: 0 })
      return
    }

    void (async () => {
      await fitView({ padding: 0.2 })
      setSavedViewport(getViewport())
    })()
  }, [graph, nodes.length, savedViewport, setViewport, fitView, getViewport, setSavedViewport])

  const onMoveEnd = useCallback<OnMove>((_event, viewport) => {
    setSavedViewport(viewport)
  }, [setSavedViewport])

  // Build health score map for popover
  const healthScoreMap = useMemo(() => {
    const map = new Map<string, HealthScore>()
    if (graph) {
      for (const hs of graph.graph.healthScores) {
        map.set(hs.nodeId, hs)
      }
    }
    return map
  }, [graph])

  // Build adjacency map for edge lookup
  const adjacentEdgesMap = useMemo(() => {
    const map = new Map<string, GraphEdgeType[]>()
    if (graph) {
      for (const edge of graph.graph.edges) {
        const fromEdges = map.get(edge.from) ?? []
        fromEdges.push(edge)
        map.set(edge.from, fromEdges)
        if (edge.from !== edge.to) {
          const toEdges = map.get(edge.to) ?? []
          toEdges.push(edge)
          map.set(edge.to, toEdges)
        }
      }
    }
    return map
  }, [graph])

  // Handle node click -> toggle popover
  const onNodeClick = useCallback<NodeMouseHandler>((_event, node) => {
    const graphNode = nodeMap.get(node.id)
    if (!graphNode) return

    // Toggle off if clicking the same node
    if (selectedNodeRef.current?.nodeId === node.id) {
      selectedNodeRef.current = null
      setSelectedNode(null)
      return
    }

    const screenPos = flowToScreenPosition(node.position)
    const state: SelectedNodeState = {
      nodeId: node.id,
      node: graphNode,
      healthScore: healthScoreMap.get(node.id),
      edges: adjacentEdgesMap.get(node.id) ?? [],
      screenX: screenPos.x + NODE_WIDTH,
      screenY: screenPos.y,
    }
    selectedNodeRef.current = state
    setSelectedNode(state)
  }, [nodeMap, healthScoreMap, adjacentEdgesMap, flowToScreenPosition])

  // Close popover on pane click
  const onPaneClick = useCallback(() => {
    selectedNodeRef.current = null
    setSelectedNode(null)
  }, [])

  // Navigate to editor from popover
  const navigateToEditor = useCallback(() => {
    if (!selectedNode) return
    const { node: n } = selectedNode
    if (n.type === 'skill') setSelectedSkillId(n.id)
    else if (n.type === 'agent') setSelectedAgentId(n.id)
    else if (n.type === 'team') setSelectedTeamId(n.id)
    selectedNodeRef.current = null
    setSelectedNode(null)
  }, [selectedNode, setSelectedSkillId, setSelectedAgentId, setSelectedTeamId])

  // Update popover screen position during pan/zoom
  const onMove = useCallback<OnMove>((_event, _viewport) => {
    const sel = selectedNodeRef.current
    if (!sel) return

    // Find the current flow node to get its position
    const flowNode = nodes.find((n) => n.id === sel.nodeId)
    if (!flowNode) return

    const screenPos = flowToScreenPosition(flowNode.position)
    const el = popoverRef.current
    if (el) {
      el.style.left = `${screenPos.x + NODE_WIDTH + 16}px`
      el.style.top = `${screenPos.y - 8}px`
    }
  }, [nodes, flowToScreenPosition])

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
      {/* Visual / JSON mode toggle */}
      <div
        data-testid={selectors.graph.modeToggle}
        className="absolute top-4 left-1/2 -translate-x-1/2 z-30 flex rounded-lg border border-slate-700 bg-slate-800/90 backdrop-blur-sm overflow-hidden"
      >
        <button
          type="button"
          data-testid={selectors.graph.modeVisual}
          onClick={() => setViewMode('visual')}
          className={cn(
            'flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium transition-colors',
            viewMode === 'visual'
              ? 'bg-indigo-500/30 text-indigo-300'
              : 'text-slate-400 hover:text-slate-200',
          )}
        >
          <Network className="h-3.5 w-3.5" />
          Visual
        </button>
        <button
          type="button"
          data-testid={selectors.graph.modeJson}
          onClick={() => setViewMode('json')}
          className={cn(
            'flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium transition-colors',
            viewMode === 'json'
              ? 'bg-indigo-500/30 text-indigo-300'
              : 'text-slate-400 hover:text-slate-200',
          )}
        >
          <Braces className="h-3.5 w-3.5" />
          JSON
        </button>
      </div>

      {viewMode === 'visual' ? (
        <>
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onNodeClick={onNodeClick}
            onPaneClick={onPaneClick}
            onMove={onMove}
            onMoveEnd={onMoveEnd}
            nodeTypes={nodeTypes}
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
          </ReactFlow>

          {/* Click-anchored node detail popover */}
          {selectedNode && (
            <PanelErrorBoundary panelName="Graph Popover" minimal>
              <div ref={popoverRef}>
                <GraphNodePopover
                  node={selectedNode.node}
                  healthScore={selectedNode.healthScore}
                  edges={selectedNode.edges}
                  screenX={selectedNode.screenX}
                  screenY={selectedNode.screenY}
                  onClose={onPaneClick}
                  onNavigate={navigateToEditor}
                />
              </div>
            </PanelErrorBoundary>
          )}
        </>
      ) : (
        <Suspense fallback={
          <div className="h-full flex items-center justify-center">
            <div className="animate-spin h-6 w-6 border-2 border-primary border-t-transparent rounded-full" />
          </div>
        }>
          <GraphJsonView />
        </Suspense>
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
