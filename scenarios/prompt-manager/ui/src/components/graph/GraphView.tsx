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

import { Profiler, useCallback, useMemo, useEffect, useState, useRef, lazy, Suspense } from 'react'
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
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
import { useNavigate } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { useGraphStore, selectFilteredNodes, selectEffectiveHealthScores, type GraphLayoutMode } from '@/stores/graphStore'
import { useShallow } from 'zustand/react/shallow'
import { useIsMobile } from '@/hooks/useMediaQuery'
import { useResolvedTheme } from '@/hooks/use-theme'
import { GraphFlowNode, type GraphNodeData } from './GraphNode'
import { GraphNodePopover } from './GraphNodePopover'
import { collectNeighborhood } from './graphNeighborhood'
import { PanelErrorBoundary } from '../PanelErrorBoundary'
import { selectors } from '@/constants/selectors'
import { onProfilerRender } from '@/lib/profiler'
import { agentDetailPath, skillDetailPath, teamDetailPath } from '@/app/routes/route-paths'
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
const HEAVY_EDGE_COUNT_THRESHOLD = 300
const MINIMAP_NODE_THRESHOLD = 120
const POPOVER_WIDTH = 280
const POPOVER_HEIGHT_ESTIMATE = 360
const POPOVER_MARGIN = 8

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
    const lane = node.data.nodeType
    const list = byLane.get(lane) ?? []
    list.push(node)
    byLane.set(lane, list)
  }

  const cellX = NODE_WIDTH + 60
  const cellY = NODE_HEIGHT + 60
  const laneGap = 120
  const layoutedNodes: FlowNode[] = []
  let laneOffset = 0

  for (let laneIndex = 0; laneIndex < laneOrder.length; laneIndex++) {
    const lane = laneOrder[laneIndex] ?? 'skill'
    const laneNodes = byLane.get(lane) ?? []
    if (laneNodes.length === 0) continue
    const columns = Math.max(1, Math.ceil(Math.sqrt(laneNodes.length)))
    const rows = Math.max(1, Math.ceil(laneNodes.length / columns))
    for (let i = 0; i < laneNodes.length; i++) {
      const laneNode = laneNodes[i]
      if (!laneNode) continue
      const col = i % columns
      const row = Math.floor(i / columns)
      const x = col * cellX
      const y = laneOffset + row * cellY
      layoutedNodes.push({
        ...laneNode,
        position: direction === 'TB'
          ? { x, y }
          : { x: y, y: x },
      })
    }
    laneOffset += rows * cellY + laneGap
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

  // Respect explicit user direction selection (TB/LR) without auto-flipping.
  return runDagreLayout(nodes, edges, direction, mode)
}

function clampPopoverPosition(anchorX: number, anchorY: number): { left: number; top: number } {
  if (typeof window === 'undefined') {
    return { left: anchorX, top: anchorY }
  }

  const maxLeft = Math.max(POPOVER_MARGIN, window.innerWidth - POPOVER_WIDTH - POPOVER_MARGIN)
  const maxTop = Math.max(POPOVER_MARGIN, window.innerHeight - POPOVER_HEIGHT_ESTIMATE - POPOVER_MARGIN)

  return {
    left: Math.min(Math.max(anchorX, POPOVER_MARGIN), maxLeft),
    top: Math.min(Math.max(anchorY, POPOVER_MARGIN), maxTop),
  }
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
}

function GraphViewInner({ className }: GraphViewInnerProps) {
  const navigate = useNavigate()
  const isMobile = useIsMobile()
  const resolvedTheme = useResolvedTheme()
  const { fitView, setViewport, getViewport, flowToScreenPosition } = useReactFlow()
  const [viewMode, setViewMode] = useState<'visual' | 'json'>('visual')
  const [selectedNode, setSelectedNode] = useState<SelectedNodeState | null>(null)
  const selectedNodeRef = useRef<SelectedNodeState | null>(null)
  const hasInitializedViewport = useRef(false)

  // Store data
  const {
    graph,
    loading,
    error,
    fetchGraph,
    highlightedNodeIds,
    highlightSource,
    focusNodes,
    clearHighlights,
    queryDisplayMode,
    filters,
    layoutDirection,
    layoutMode,
    fitViewRequested,
    savedViewport,
    setSavedViewport,
  } = useGraphStore(useShallow((s) => ({
    graph: s.graph,
    loading: s.loading,
    error: s.error,
    fetchGraph: s.fetchGraph,
    highlightedNodeIds: s.highlightedNodeIds,
    highlightSource: s.highlightSource,
    focusNodes: s.focusNodes,
    clearHighlights: s.clearHighlights,
    queryDisplayMode: s.queryDisplayMode,
    filters: s.filters,
    layoutDirection: s.layoutDirection,
    layoutMode: s.layoutMode,
    fitViewRequested: s.fitViewRequested,
    savedViewport: s.viewport,
    setSavedViewport: s.setViewport,
  })))

  // useShallow prevents infinite re-render: selectFilteredNodes returns a new
  // array reference on every call (.filter()), but useShallow compares elements
  // by identity so the result is stable when the underlying data hasn't changed.
  const filteredNodes = useGraphStore(useShallow(selectFilteredNodes))
  const effectiveHealthScores = useGraphStore(useShallow(selectEffectiveHealthScores))

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

  const querySelectedNodeIds = useMemo(() => {
    if (highlightedNodeIds.size === 0) return new Set<string>()

    // Focus highlights use raw IDs directly — no CLI remapping needed
    // because the BFS already computed the correct neighborhood.
    if (highlightSource === 'focus') {
      // Still handle CLI collapsing: if CLIs are collapsed into a cluster
      // node, map individual CLI IDs to the cluster ID.
      if (!filters.collapseCLIs) return new Set(highlightedNodeIds)

      const selected = new Set<string>()
      let hasSelectedCLI = false
      for (const id of highlightedNodeIds) {
        if (id.startsWith('cli:')) {
          hasSelectedCLI = true
        } else {
          selected.add(id)
        }
      }
      if (hasSelectedCLI) selected.add(CLI_CLUSTER_ID)
      return selected
    }

    // Query highlights: CLI IDs from query results are only meaningful
    // when collapseCLIs is on (they map to the cluster node).
    const selected = new Set<string>()
    let hasSelectedCLI = false

    for (const id of highlightedNodeIds) {
      if (id.startsWith('cli:')) {
        hasSelectedCLI = true
        continue
      }
      selected.add(id)
    }

    if (filters.collapseCLIs && hasSelectedCLI) {
      selected.add(CLI_CLUSTER_ID)
    }

    return selected
  }, [highlightedNodeIds, highlightSource, filters.collapseCLIs])

  const hasQuerySelection = querySelectedNodeIds.size > 0
  const hideNonSelected = hasQuerySelection && queryDisplayMode === 'hide-others'
  const dimNonSelected = hasQuerySelection && queryDisplayMode === 'dim-others'

  const nodesAfterQueryMode = useMemo(() => {
    if (!hideNonSelected) return viewModel.nodes
    return viewModel.nodes.filter((node) => querySelectedNodeIds.has(node.id))
  }, [viewModel.nodes, hideNonSelected, querySelectedNodeIds])

  const nodeIdsAfterQueryMode = useMemo(() => new Set(nodesAfterQueryMode.map((node) => node.id)), [nodesAfterQueryMode])

  const edgesAfterQueryMode = useMemo(() => {
    if (!hideNonSelected) {
      return viewModel.edges.filter((edge) => nodeIdsAfterQueryMode.has(edge.from) && nodeIdsAfterQueryMode.has(edge.to))
    }

    return viewModel.edges.filter((edge) => querySelectedNodeIds.has(edge.from) && querySelectedNodeIds.has(edge.to))
  }, [viewModel.edges, nodeIdsAfterQueryMode, hideNonSelected, querySelectedNodeIds])

  // Build node ID set for filtering edges
  const renderedNodeIds = useMemo(() => new Set(nodesAfterQueryMode.map((n) => n.id)), [nodesAfterQueryMode])

  // Build health score map for nodes and popovers.
  const healthScoreMap = useMemo(() => {
    const map = new Map<string, HealthScore>()
    for (const hs of effectiveHealthScores) {
      map.set(hs.nodeId, hs)
    }
    return map
  }, [effectiveHealthScores])

  // Build flow nodes
  const flowNodes = useMemo((): FlowNode[] => {
    return nodesAfterQueryMode.map((node): FlowNode => {
      const isQuerySelected = querySelectedNodeIds.has(node.id)
      const queryState: GraphNodeData['queryState'] = hasQuerySelection
        ? (isQuerySelected ? 'selected' : (dimNonSelected ? 'dimmed' : 'normal'))
        : 'normal'

      return {
        id: node.id,
        type: 'graphNode',
        position: { x: 0, y: 0 },
        data: {
          label: node.label,
          nodeType: node.type,
          healthScore: healthScoreMap.get(node.id)?.score ?? null,
          queryState,
        },
      }
    })
  }, [nodesAfterQueryMode, querySelectedNodeIds, hasQuerySelection, dimNonSelected, healthScoreMap])

  // Build flow edges (only between visible nodes)
  const useLightweightEdges = edgesAfterQueryMode.length > HEAVY_EDGE_COUNT_THRESHOLD
  const flowEdges = useMemo((): FlowEdge[] => {
    return edgesAfterQueryMode
      .filter((e) => renderedNodeIds.has(e.from) && renderedNodeIds.has(e.to))
      .map((edge, i): FlowEdge => {
        const defaultStyle: { stroke: string; strokeDasharray?: string } = { stroke: 'hsl(var(--muted-foreground))' }
        const edgeStyle = EDGE_STYLES[edge.kind] ?? defaultStyle
        const isConnectedToSelection = querySelectedNodeIds.has(edge.from) || querySelectedNodeIds.has(edge.to)
        const edgeOpacity = dimNonSelected && !isConnectedToSelection ? 0.15 : 1

        return {
          id: `e-${edge.from}-${edge.to}-${edge.kind}-${i}`,
          source: edge.from,
          target: edge.to,
          type: useLightweightEdges ? 'straight' : 'smoothstep',
          selectable: false,
          focusable: false,
          reconnectable: false,
          animated: false,
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
            opacity: edgeOpacity,
            pointerEvents: 'none',
          },
        }
      })
  }, [edgesAfterQueryMode, renderedNodeIds, useLightweightEdges, dimNonSelected, querySelectedNodeIds])

  // Apply layout
  const { nodes: layoutedNodes, edges: layoutedEdges } = useMemo(
    () => getLayoutedElements(flowNodes, flowEdges, layoutDirection, layoutMode),
    [flowNodes, flowEdges, layoutDirection, layoutMode],
  )

  const nodes = layoutedNodes
  const edges = layoutedEdges
  const showMiniMap = nodes.length <= MINIMAP_NODE_THRESHOLD

  const autoFitSignature = useMemo(() => {
    const highlighted = Array.from(highlightedNodeIds).sort().join('|')
    return [
      layoutMode,
      layoutDirection,
      filters.collapseCLIs ? 'collapse' : 'expand',
      filters.showLowSignalEdges ? 'low-on' : 'low-off',
      queryDisplayMode,
      flowNodes.length,
      flowEdges.length,
      highlighted,
    ].join('::')
  }, [
    layoutMode,
    layoutDirection,
    filters.collapseCLIs,
    filters.showLowSignalEdges,
    queryDisplayMode,
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

  // Keep selected node data in sync when graph data updates.
  useEffect(() => {
    if (!selectedNodeRef.current) return
    const nextNode = nodeMap.get(selectedNodeRef.current.nodeId)
    if (!nextNode) {
      selectedNodeRef.current = null
      setSelectedNode(null)
      return
    }

    const nextState: SelectedNodeState = {
      nodeId: selectedNodeRef.current.nodeId,
      node: nextNode,
      healthScore: healthScoreMap.get(selectedNodeRef.current.nodeId),
      edges: adjacentEdgesMap.get(selectedNodeRef.current.nodeId) ?? [],
    }
    selectedNodeRef.current = nextState
    setSelectedNode(nextState)
  }, [nodeMap, healthScoreMap, adjacentEdgesMap])

  // Handle node click -> toggle popover + focus neighborhood
  const onNodeClick = useCallback<NodeMouseHandler>((_event, node) => {
    const graphNode = nodeMap.get(node.id)
    if (!graphNode) return

    // Toggle off if clicking the same node
    if (selectedNodeRef.current?.nodeId === node.id) {
      selectedNodeRef.current = null
      setSelectedNode(null)
      // If we're in focus mode, clear the focus highlight
      if (highlightSource === 'focus') {
        clearHighlights()
      }
      return
    }

    const state: SelectedNodeState = {
      nodeId: node.id,
      node: graphNode,
      healthScore: healthScoreMap.get(node.id),
      edges: adjacentEdgesMap.get(node.id) ?? [],
    }
    selectedNodeRef.current = state
    setSelectedNode(state)

    // When no query is active (or already in focus mode), compute and focus on the neighborhood
    if (highlightSource !== 'query') {
      const neighborhood = collectNeighborhood(node.id, adjacentEdgesMap, nodeMap)
      focusNodes(Array.from(neighborhood))
    }
  }, [nodeMap, healthScoreMap, adjacentEdgesMap, highlightSource, clearHighlights, focusNodes])

  // Close popover on pane click + clear focus
  const onPaneClick = useCallback(() => {
    selectedNodeRef.current = null
    setSelectedNode(null)
    if (highlightSource === 'focus') {
      clearHighlights()
    }
  }, [highlightSource, clearHighlights])

  // Navigate to editor from popover
  const navigateToEditor = useCallback(() => {
    if (!selectedNode) return
    const { node: n } = selectedNode
    if (n.type === 'skill') navigate(skillDetailPath(n.id))
    else if (n.type === 'agent') navigate(agentDetailPath(n.id))
    else if (n.type === 'team') navigate(teamDetailPath(n.id))
    selectedNodeRef.current = null
    setSelectedNode(null)
  }, [navigate, selectedNode])

  // Ensure selected node remains tracked while panning/zooming.
  const onMove = useCallback<OnMove>(() => {
    if (!selectedNodeRef.current) return
    setSelectedNode((current) => (current ? { ...current } : current))
  }, [])

  const desktopPopoverPosition = useMemo(() => {
    if (!selectedNode || isMobile) return null
    const flowNode = nodes.find((n) => n.id === selectedNode.nodeId)
    if (!flowNode) return null
    const screenPos = flowToScreenPosition(flowNode.position)
    return clampPopoverPosition(screenPos.x + NODE_WIDTH + 16, screenPos.y - 8)
  }, [selectedNode, isMobile, nodes, flowToScreenPosition])

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
    <div className={cn('h-full relative', className)}>
      {/* Visual / JSON mode toggle */}
      <div
        data-testid={selectors.graph.modeToggle}
        className="absolute top-4 left-1/2 -translate-x-1/2 z-30 flex rounded-lg border border-border bg-card/90 backdrop-blur-sm overflow-hidden"
      >
        <button
          type="button"
          data-testid={selectors.graph.modeVisual}
          onClick={() => setViewMode('visual')}
          className={cn(
            'flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium transition-colors',
            viewMode === 'visual'
              ? 'bg-indigo-500/30 text-indigo-300'
              : 'text-muted-foreground hover:text-foreground',
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
              : 'text-muted-foreground hover:text-foreground',
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
            onNodeClick={onNodeClick}
            onPaneClick={onPaneClick}
            onMove={onMove}
            onMoveEnd={onMoveEnd}
            nodeTypes={nodeTypes}
            onlyRenderVisibleElements
            nodesDraggable={false}
            nodesConnectable={false}
            elementsSelectable={false}
            minZoom={0.1}
            maxZoom={2}
            proOptions={{ hideAttribution: true }}
            className="bg-background"
          >
            <Background color="hsl(var(--border))" gap={20} />
            <Controls showInteractive={false} />
            {showMiniMap && (
              <MiniMap
                nodeColor={(node) => {
                  const data = node.data as unknown as GraphNodeData
                  switch (data.nodeType) {
                    case 'team': return '#3b82f6'
                    case 'agent': return '#10b981'
                    case 'skill': return '#8b5cf6'
                    default: return '#f97316'
                  }
                }}
                maskColor={resolvedTheme === 'dark' ? 'rgba(0, 0, 0, 0.5)' : 'rgba(255, 255, 255, 0.5)'}
              />
            )}
          </ReactFlow>

          {/* Click-anchored node detail popover */}
          {selectedNode && (
            <PanelErrorBoundary panelName="Graph Popover" minimal>
              <GraphNodePopover
                node={selectedNode.node}
                healthScore={selectedNode.healthScore}
                edges={selectedNode.edges}
                screenX={desktopPopoverPosition?.left ?? 0}
                screenY={desktopPopoverPosition?.top ?? 0}
                onClose={onPaneClick}
                onNavigate={navigateToEditor}
                variant={isMobile ? 'mobile' : 'desktop'}
              />
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
    <Profiler id="GraphView" onRender={onProfilerRender}>
      <ReactFlowProvider>
        <GraphViewInner className={className} />
      </ReactFlowProvider>
    </Profiler>
  )
}
