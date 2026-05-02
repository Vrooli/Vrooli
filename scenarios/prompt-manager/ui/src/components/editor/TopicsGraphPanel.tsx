/**
 * TopicsGraphPanel - Topics-mode visualization of a team's message-flow.
 *
 * Renders the directed graph derived from each member's topics.json:
 * - Member nodes
 * - Boundary nodes (external producers, decision queues, PoR sinks,
 *   capability-gap registry, skill proposals, backlog)
 * - Directed edges labelled with topic prefix; edge style by kind
 * - Validation overlay (red ring on members with error-severity findings)
 *
 * DOC: docs/agent-system/drafts/topics-schema.md
 */

import { useCallback, useMemo, useState } from 'react'
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  Panel,
  MarkerType,
  type EdgeMouseHandler,
} from '@xyflow/react'
import dagre from '@dagrejs/dagre'
import { Loader2, RefreshCw, AlertTriangle } from 'lucide-react'
import { cn } from '@/lib/utils'
import { TopicsGraphNode } from './TopicsGraphNode'
import { TopicsValidationPanel } from './TopicsValidationPanel'
import { useTopicsGraph } from '@/hooks/useTopicsGraph'
import type {
  TopicEdgeKind,
  TopicGraphEdge,
  TopicGraphNode as TopicGraphNodeType,
  TopicNodeKind,
  TopicValidation,
  TopicsFlowEdge,
  TopicsFlowNode,
  TopicsGraphResponse,
} from '@/types/topicsGraph'

import '@xyflow/react/dist/style.css'

interface TopicsGraphPanelProps {
  teamId: string
  /** Optional handler invoked when a member node is clicked. Boundary nodes still highlight locally without firing this. */
  onSelectMember?: (agentId: string) => void
  /** Controlled validation-sidebar visibility (lifted to TeamEditorPanel). */
  showValidation?: boolean
  /** Toggle handler for the validation sidebar (paired with showValidation). */
  onValidationToggle?: () => void
  /** Open a member's file in the Files tab (used by validation-finding CTA). */
  onOpenMemberFile?: (team: string, member: string, fileName: string) => void
  className?: string
}

const NODE_WIDTH = 200
const NODE_HEIGHT = 60

const nodeTypes = {
  topicsNode: TopicsGraphNode,
}

const EDGE_STYLES: Record<TopicEdgeKind, { stroke: string; dashed?: boolean }> = {
  intake: { stroke: 'hsl(var(--primary))' },
  output: { stroke: '#22d3ee' },
  decision_owned: { stroke: '#a855f7' },
  decision_consumed: { stroke: '#a855f7', dashed: true },
  external_producer: { stroke: '#f59e0b', dashed: true },
  capability_gap: { stroke: '#f43f5e', dashed: true },
}

/**
 * Group nodes into ranks so dagre lays them out left-to-right in flow order:
 *   external/decision_consumed sources (left) -> members (middle) -> outputs (right)
 *
 * Dagre's standard rankdir=LR + edge direction handles ordering automatically;
 * we keep the helper for explicit columnization should it become useful.
 */
function getLayouted(
  nodes: TopicsFlowNode[],
  edges: TopicsFlowEdge[],
): { nodes: TopicsFlowNode[]; edges: TopicsFlowEdge[] } {
  const dagreGraph = new dagre.graphlib.Graph()
  dagreGraph.setDefaultEdgeLabel(() => ({}))
  dagreGraph.setGraph({ rankdir: 'LR', nodesep: 30, ranksep: 110 })

  nodes.forEach((node) => {
    dagreGraph.setNode(node.id, { width: NODE_WIDTH, height: NODE_HEIGHT })
  })
  edges.forEach((edge) => {
    dagreGraph.setEdge(edge.source, edge.target)
  })

  dagre.layout(dagreGraph)

  const laidOut = nodes.map((node) => {
    const pos = dagreGraph.node(node.id)
    return {
      ...node,
      position: {
        x: pos.x - NODE_WIDTH / 2,
        y: pos.y - NODE_HEIGHT / 2,
      },
    }
  })

  return { nodes: laidOut, edges }
}

function shortenPrefix(prefix: string): string {
  if (!prefix) return ''
  if (prefix.length <= 28) return prefix
  return prefix.slice(0, 12) + '…' + prefix.slice(-12)
}

function buildFlow(
  graph: TopicsGraphResponse,
  selectedNodeId: string | null,
  onSelectNode: (id: string) => void,
) {
  const errorByNode = new Map<string, number>()
  const warnByNode = new Map<string, number>()

  for (const f of graph.validation.findings) {
    const memberID = `member:${f.member.team}/${f.member.member}`
    const map = f.severity === 'error' ? errorByNode : warnByNode
    map.set(memberID, (map.get(memberID) ?? 0) + 1)
  }

  const flowNodes: TopicsFlowNode[] = graph.nodes.map((n: TopicGraphNodeType) => ({
    id: n.id,
    type: 'topicsNode' as const,
    position: { x: 0, y: 0 },
    data: {
      graphNode: n,
      errorCount: errorByNode.get(n.id) ?? 0,
      warningCount: warnByNode.get(n.id) ?? 0,
      isSelected: n.id === selectedNodeId,
      onSelect: onSelectNode,
    },
  }))

  const flowEdges: TopicsFlowEdge[] = graph.edges.map((e: TopicGraphEdge, i) => {
    const style = EDGE_STYLES[e.kind]
    return {
      id: `edge-${i}-${e.kind}-${e.from}-${e.to}-${e.prefix}`,
      source: e.from,
      target: e.to,
      type: 'smoothstep' as const,
      label: shortenPrefix(e.prefix),
      labelStyle: { fontSize: 10, fontFamily: 'monospace' },
      labelBgStyle: { fill: 'hsl(var(--background))' },
      labelBgPadding: [3, 1] as [number, number],
      labelBgBorderRadius: 3,
      style: {
        stroke: style.stroke,
        strokeWidth: 1.5,
        strokeDasharray: style.dashed ? '4 3' : undefined,
      },
      markerEnd: {
        type: MarkerType.ArrowClosed,
        width: 14,
        height: 14,
        color: style.stroke,
      },
      data: { graphEdge: e },
    }
  })

  return getLayouted(flowNodes, flowEdges)
}

export function TopicsGraphPanel({
  teamId,
  onSelectMember,
  showValidation: showValidationProp,
  onValidationToggle,
  onOpenMemberFile,
  className,
}: TopicsGraphPanelProps) {
  const { graph, loading, error, refresh } = useTopicsGraph(teamId)
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null)
  const [selectedEdge, setSelectedEdge] = useState<TopicGraphEdge | null>(null)
  const [internalShowValidation, setInternalShowValidation] = useState(true)
  const showValidation = showValidationProp ?? internalShowValidation
  const handleValidationToggle = useCallback(() => {
    if (onValidationToggle) onValidationToggle()
    else setInternalShowValidation((v) => !v)
  }, [onValidationToggle])

  const handleSelectNode = useCallback((nodeId: string) => {
    setSelectedNodeId(nodeId)
    if (nodeId.startsWith('member:')) {
      const slashIdx = nodeId.indexOf('/', 'member:'.length)
      if (slashIdx > 0) {
        const agentId = nodeId.slice(slashIdx + 1)
        if (agentId) onSelectMember?.(agentId)
      }
    }
  }, [onSelectMember])

  const fetchGraph = useCallback(() => refresh(), [refresh])

  const { nodes, edges } = useMemo(() => {
    if (!graph) {
      return { nodes: [] as TopicsFlowNode[], edges: [] as TopicsFlowEdge[] }
    }
    const built = buildFlow(graph, selectedNodeId, handleSelectNode)
    return { nodes: built.nodes, edges: built.edges }
  }, [graph, selectedNodeId, handleSelectNode])

  const handleEdgeClick: EdgeMouseHandler<TopicsFlowEdge> = (event, edge) => {
    event.stopPropagation()
    const data = edge.data as { graphEdge: TopicGraphEdge } | undefined
    setSelectedEdge(data?.graphEdge ?? null)
  }

  const handlePaneClick = () => {
    setSelectedNodeId(null)
    setSelectedEdge(null)
  }

  const validation: TopicValidation = graph?.validation ?? {
    findings: [],
    errors: 0,
    warnings: 0,
  }

  const summary = (() => {
    if (!graph) return null
    const memberCount = graph.nodes.filter((n) => n.kind === 'member').length
    const boundaryByKind: Record<TopicNodeKind, number> = {
      member: 0,
      external: 0,
      decision: 0,
      por_file: 0,
      capability_gap: 0,
      skill_proposal: 0,
      backlog: 0,
      knowledge_sink: 0,
    }
    for (const n of graph.nodes) {
      boundaryByKind[n.kind] = boundaryByKind[n.kind] + 1
    }
    return { memberCount, boundaryByKind }
  })()

  if (error) {
    return (
      <div className={cn('h-full flex flex-col items-center justify-center text-center p-6', className)}>
        <AlertTriangle className="h-10 w-10 text-rose-500 mb-3" />
        <p className="text-sm text-muted-foreground max-w-sm mb-3">
          Failed to load topics graph: {error}
        </p>
        <button
          type="button"
          onClick={() => void fetchGraph()}
          className="px-3 py-1.5 text-xs rounded-lg bg-card border border-border hover:bg-muted"
        >
          Retry
        </button>
      </div>
    )
  }

  if (loading && !graph) {
    return (
      <div className={cn('h-full flex flex-col items-center justify-center', className)}>
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (graph && graph.nodes.length === 0) {
    return (
      <div className={cn('h-full flex flex-col items-center justify-center text-center p-6', className)}>
        <p className="text-sm text-muted-foreground max-w-sm">
          No topic flow declared for this team yet. Add `topics.json` to one or more members
          to populate the graph.
        </p>
      </div>
    )
  }

  return (
    <div className={cn('h-full flex', className)}>
      <div className="flex-1 min-w-0 h-full relative">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onEdgeClick={handleEdgeClick}
          onPaneClick={handlePaneClick}
          nodeTypes={nodeTypes}
          fitView
          fitViewOptions={{ padding: 0.18 }}
          minZoom={0.3}
          maxZoom={1.5}
          proOptions={{ hideAttribution: true }}
          nodesDraggable={false}
          nodesConnectable={false}
          elementsSelectable
          className="bg-background"
        >
          <Background color="hsl(var(--border))" gap={20} />
          <Controls showInteractive={false} />
          <MiniMap
            nodeColor={(n) => {
              const kind = (n.data as { graphNode?: { kind?: TopicNodeKind } } | undefined)?.graphNode?.kind
              switch (kind) {
                case 'member': return '#6366f1'
                case 'external': return '#f59e0b'
                case 'decision': return '#a855f7'
                case 'por_file': return '#3b82f6'
                case 'capability_gap': return '#f43f5e'
                case 'skill_proposal': return '#10b981'
                case 'backlog': return '#64748b'
                case 'knowledge_sink': return '#06b6d4'
                default: return '#94a3b8'
              }
            }}
            maskColor="rgba(0, 0, 0, 0.5)"
          />

          <Panel position="top-left" className="flex flex-col gap-2 max-w-xs">
            <div className="flex items-start gap-2 p-2 rounded-lg bg-card border border-border">
              <div className="space-y-1 min-w-0">
                <p className="text-xs font-medium">Topics Mode</p>
                {summary && (
                  <p className="text-[10px] text-muted-foreground">
                    {summary.memberCount} member(s) · {graph?.nodes.length ?? 0} node(s) ·{' '}
                    {graph?.edges.length ?? 0} edge(s)
                  </p>
                )}
                {validation.errors === 0 && validation.warnings === 0 ? (
                  <p className="text-[10px] text-emerald-400">Validation: clean</p>
                ) : (
                  <p className="text-[10px] text-muted-foreground">
                    Validation:{' '}
                    {validation.errors > 0 && (
                      <span className="text-rose-400">{validation.errors} error(s)</span>
                    )}
                    {validation.errors > 0 && validation.warnings > 0 && ' · '}
                    {validation.warnings > 0 && (
                      <span className="text-amber-400">{validation.warnings} warning(s)</span>
                    )}
                  </p>
                )}
              </div>
            </div>
            {selectedEdge && (
              <div className="p-2 rounded-lg bg-card border border-border">
                <p className="text-xs font-medium">Edge</p>
                <p className="text-[11px] font-mono text-muted-foreground break-all">
                  {selectedEdge.prefix || '(no prefix)'}
                </p>
                <p className="text-[10px] text-muted-foreground/80 mt-1">
                  kind: {selectedEdge.kind}
                </p>
                <p className="text-[10px] text-muted-foreground/80 truncate">
                  {selectedEdge.from} → {selectedEdge.to}
                </p>
              </div>
            )}
          </Panel>

          <Panel position="top-right" className="flex gap-2">
            <button
              type="button"
              onClick={() => void fetchGraph()}
              className={cn(
                'flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg',
                'bg-card border border-border text-foreground',
                'hover:bg-muted transition-colors',
              )}
              title="Refresh"
              aria-label="Refresh topics graph"
            >
              <RefreshCw className={cn('h-3.5 w-3.5', loading && 'animate-spin')} />
              <span>Refresh</span>
            </button>
            <button
              type="button"
              onClick={handleValidationToggle}
              className={cn(
                'flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg',
                'bg-card border border-border text-foreground',
                'hover:bg-muted transition-colors',
                showValidation && 'border-primary text-primary',
              )}
              title="Toggle validation panel"
              aria-label="Toggle validation panel"
              data-testid="topics-validation-toggle"
            >
              <AlertTriangle className="h-3.5 w-3.5" />
              <span>{validation.errors + validation.warnings}</span>
            </button>
          </Panel>
        </ReactFlow>
      </div>

      {showValidation && (
        <div className="w-72 flex-shrink-0 border-l border-border h-full overflow-hidden">
          <TopicsValidationPanel
            validation={validation}
            onSelectMember={(team, member) => {
              setSelectedNodeId(`member:${team}/${member}`)
            }}
            onOpenMemberFile={onOpenMemberFile}
          />
        </div>
      )}
    </div>
  )
}
