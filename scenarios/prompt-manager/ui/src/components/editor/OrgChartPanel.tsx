/**
 * OrgChartPanel - React Flow canvas for org chart visualization.
 *
 * Features:
 * - Dagre hierarchical layout
 * - Node selection -> detail panel
 * - Edge drag-drop for manager reassignment
 * - Toolbar with Add Member, auto-layout, zoom controls
 */

import { useCallback, useMemo, useEffect, useRef, useState } from 'react'
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  type Connection,
  type Edge,
  type OnConnect,
  type EdgeMouseHandler,
  Panel,
  MarkerType,
} from '@xyflow/react'
import dagre from '@dagrejs/dagre'
import { UserPlus, LayoutGrid, Users, Info, Trash2, Code } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useIsMobile } from '@/hooks/useMediaQuery'
import { OrgChartNode } from './OrgChartNode'
import type { TeamDetails } from '@/types/team'
import type { Agent, AgentAppearance } from '@/types/agent'
import type { OrgEdge, OrgChartNode as OrgChartNodeType, OrgChartFlowEdge, OrgChartNodeData } from '@/types/orgChart'

import '@xyflow/react/dist/style.css'

// ============================================================================
// Types
// ============================================================================

interface OrgChartPanelProps {
  team: TeamDetails
  edges: OrgEdge[]
  allAgents: Agent[]
  selectedMemberId: string | null
  onSelectMember: (agentId: string | null) => void
  onEdgeUpdate: (agentId: string, managerId: string | null) => void
  onAddMember: () => void
  onSwitchToCode?: () => void
  className?: string
}

// ============================================================================
// Layout Constants
// ============================================================================

const NODE_WIDTH = 200
const NODE_HEIGHT = 80
const LAYOUT_DIRECTION = 'TB' // Top to bottom
type LayoutDirection = 'TB' | 'LR'

// ============================================================================
// Node Types
// ============================================================================

const nodeTypes = {
  orgMember: OrgChartNode,
}

// ============================================================================
// Layout Function
// ============================================================================

function getLayoutedElements(
  nodes: OrgChartNodeType[],
  edges: OrgChartFlowEdge[],
  direction = LAYOUT_DIRECTION
): { nodes: OrgChartNodeType[]; edges: OrgChartFlowEdge[] } {
  const dagreGraph = new dagre.graphlib.Graph()
  dagreGraph.setDefaultEdgeLabel(() => ({}))
  dagreGraph.setGraph({ rankdir: direction, nodesep: 50, ranksep: 80 })

  // Add nodes to dagre
  nodes.forEach((node) => {
    dagreGraph.setNode(node.id, { width: NODE_WIDTH, height: NODE_HEIGHT })
  })

  // Add edges to dagre
  edges.forEach((edge) => {
    dagreGraph.setEdge(edge.source, edge.target)
  })

  // Run layout
  dagre.layout(dagreGraph)

  // Apply positions to nodes
  const layoutedNodes = nodes.map((node) => {
    const nodeWithPosition = dagreGraph.node(node.id)
    return {
      ...node,
      position: {
        x: nodeWithPosition.x - NODE_WIDTH / 2,
        y: nodeWithPosition.y - NODE_HEIGHT / 2,
      },
    }
  })

  return { nodes: layoutedNodes, edges }
}

// ============================================================================
// Component
// ============================================================================

export function OrgChartPanel({
  team,
  edges: orgEdges,
  allAgents,
  selectedMemberId,
  onSelectMember,
  onEdgeUpdate,
  onAddMember,
  onSwitchToCode,
  className,
}: OrgChartPanelProps) {
  const [selectedEdgeId, setSelectedEdgeId] = useState<string | null>(null)
  const [layoutDirection, setLayoutDirection] = useState<LayoutDirection>(LAYOUT_DIRECTION)
  const [showReportingHelp, setShowReportingHelp] = useState(false)
  const reportingHelpRef = useRef<HTMLDivElement>(null)
  const isMobile = useIsMobile()

  useEffect(() => {
    if (!showReportingHelp) return
    const handleClick = (e: MouseEvent) => {
      if (reportingHelpRef.current && !reportingHelpRef.current.contains(e.target as Node)) {
        setShowReportingHelp(false)
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [showReportingHelp])

  // Build agent appearance map
  const agentAppearances = useMemo(() => {
    const map = new Map<string, AgentAppearance>()
    allAgents.forEach((agent) => {
      if (agent.appearance) {
        map.set(agent.id, agent.appearance)
      }
    })
    return map
  }, [allAgents])

  const memberNames = useMemo(() => {
    const map = new Map<string, string>()
    team.members.forEach((member) => {
      map.set(member.agentId, member.displayName)
    })
    return map
  }, [team.members])

  const managerByReport = useMemo(() => {
    const map = new Map<string, string>()
    orgEdges.forEach((edge) => {
      map.set(edge.reportId, edge.managerId)
    })
    return map
  }, [orgEdges])

  const reportsByManager = useMemo(() => {
    const map = new Map<string, string[]>()
    orgEdges.forEach((edge) => {
      const existing = map.get(edge.managerId)
      if (existing) {
        existing.push(edge.reportId)
      } else {
        map.set(edge.managerId, [edge.reportId])
      }
    })
    return map
  }, [orgEdges])

  // Convert team members to React Flow nodes
  const initialNodes = useMemo((): OrgChartNodeType[] => {
    return team.members.map((member): OrgChartNodeType => {
      const managerId = managerByReport.get(member.agentId)
      return {
        id: member.agentId,
        type: 'orgMember',
        position: { x: 0, y: 0 }, // Will be set by dagre
        data: {
          member,
          appearance: agentAppearances.get(member.agentId),
          teamRoles: team.roles,
          isSelected: member.agentId === selectedMemberId,
          managerName: managerId ? (memberNames.get(managerId) ?? managerId) : undefined,
          directReportCount: reportsByManager.get(member.agentId)?.length ?? 0,
          onSelect: onSelectMember,
        },
      }
    })
  }, [
    team.members,
    team.roles,
    agentAppearances,
    selectedMemberId,
    managerByReport,
    memberNames,
    reportsByManager,
    onSelectMember,
  ])

  // Convert org edges to React Flow edges
  const initialEdges = useMemo((): OrgChartFlowEdge[] => {
    return orgEdges.map((edge): OrgChartFlowEdge => ({
      id: edge.id,
      source: edge.managerId,
      target: edge.reportId,
      type: 'smoothstep',
      animated: false,
      markerEnd: {
        type: MarkerType.ArrowClosed,
        width: 20,
        height: 20,
        color: 'hsl(var(--muted-foreground))',
      },
      style: {
        stroke: 'hsl(var(--muted-foreground))',
        strokeWidth: 2,
      },
      data: { originalEdge: edge },
    }))
  }, [orgEdges])

  // Apply layout
  const { nodes: layoutedNodes, edges: layoutedEdges } = useMemo(
    () => getLayoutedElements(initialNodes, initialEdges, layoutDirection),
    [initialNodes, initialEdges, layoutDirection]
  )

  // React Flow state - use type assertions since hooks don't preserve generics
  const [nodes, setNodes, onNodesChange] = useNodesState<OrgChartNodeType>(layoutedNodes)
  const [flowEdges, setEdges, onEdgesChange] = useEdgesState<OrgChartFlowEdge>(layoutedEdges)

  // Update nodes when selection or data changes
  useEffect(() => {
    const updatedNodes: OrgChartNodeType[] = layoutedNodes.map((node) => ({
      ...node,
      data: {
        ...node.data,
        isSelected: node.id === selectedMemberId,
      },
    }))
    setNodes(updatedNodes)
  }, [layoutedNodes, selectedMemberId, setNodes])

  // Update edges when org edges change
  useEffect(() => {
    setEdges(layoutedEdges)
  }, [layoutedEdges, setEdges])

  useEffect(() => {
    if (selectedEdgeId && !layoutedEdges.some((edge) => edge.id === selectedEdgeId)) {
      setSelectedEdgeId(null)
    }
  }, [layoutedEdges, selectedEdgeId])

  // Handle new edge connection (drag to connect)
  const onConnect: OnConnect = useCallback(
    (connection: Connection) => {
      if (connection.source && connection.target) {
        if (connection.source === connection.target) return
        // source = manager, target = report
        onEdgeUpdate(connection.target, connection.source)
      }
    },
    [onEdgeUpdate]
  )

  // Handle edge deletion
  const onEdgeDelete = useCallback(
    (deletedEdges: Edge[]) => {
      deletedEdges.forEach((edge) => {
        onEdgeUpdate(edge.target, null)
      })
      setSelectedEdgeId(null)
    },
    [onEdgeUpdate]
  )

  // Handle auto-layout button
  const handleAutoLayout = useCallback(() => {
    const nextDirection: LayoutDirection = layoutDirection === 'TB' ? 'LR' : 'TB'
    const { nodes: newNodes, edges: newEdges } = getLayoutedElements(nodes, flowEdges, nextDirection)
    setLayoutDirection(nextDirection)
    setNodes(newNodes)
    setEdges(newEdges)
  }, [layoutDirection, nodes, flowEdges, setNodes, setEdges])

  // Handle background click to deselect
  const onPaneClick = useCallback(() => {
    onSelectMember(null)
    setSelectedEdgeId(null)
  }, [onSelectMember])

  const onEdgeClick = useCallback<EdgeMouseHandler<OrgChartFlowEdge>>((event, edge) => {
    event.stopPropagation()
    setSelectedEdgeId(edge.id)
  }, [])

  const onNodeClick = useCallback(() => {
    setSelectedEdgeId(null)
  }, [])

  const selectedEdge = selectedEdgeId
    ? flowEdges.find((edge) => edge.id === selectedEdgeId) ?? null
    : null
  const selectedManagerName = selectedEdge
    ? memberNames.get(selectedEdge.source) ?? selectedEdge.source
    : null
  const selectedReportName = selectedEdge
    ? memberNames.get(selectedEdge.target) ?? selectedEdge.target
    : null

  const handleRemoveSelectedEdge = useCallback(() => {
    if (!selectedEdge) return
    onEdgeUpdate(selectedEdge.target, null)
    setSelectedEdgeId(null)
  }, [onEdgeUpdate, selectedEdge])

  // Empty state
  if (team.members.length === 0) {
    return (
      <div className={cn('h-full flex flex-col items-center justify-center', className)}>
        <div className="text-center">
          <Users className="h-16 w-16 mx-auto mb-4 text-muted-foreground/50" />
          <h3 className="text-lg font-medium text-muted-foreground mb-2">
            No Team Members
          </h3>
          <p className="text-sm text-muted-foreground/70 max-w-xs mx-auto mb-4">
            Add your first team member to start building your org chart.
          </p>
          <div className="flex flex-wrap items-center justify-center gap-2">
            <button
              type="button"
              onClick={onAddMember}
              className={cn(
                'inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg',
                'bg-primary text-primary-foreground hover:bg-primary/90 transition-colors'
              )}
            >
              <UserPlus className="h-4 w-4" />
              Add First Member
            </button>
            {onSwitchToCode && (
              <button
                type="button"
                onClick={onSwitchToCode}
                className={cn(
                  'inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg',
                  'bg-card border border-border text-foreground hover:bg-muted transition-colors'
                )}
              >
                <Code className="h-4 w-4" />
                Code View
              </button>
            )}
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className={cn('h-full', className)}>
      <ReactFlow
        nodes={nodes}
        edges={flowEdges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onEdgesDelete={onEdgeDelete}
        onEdgeClick={onEdgeClick}
        onNodeClick={onNodeClick}
        onPaneClick={onPaneClick}
        nodeTypes={nodeTypes}
        fitView
        fitViewOptions={{ padding: 0.2 }}
        minZoom={0.5}
        maxZoom={1.5}
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
            const data = node.data as OrgChartNodeData
            return data.appearance?.body ?? '#6366f1'
          }}
          maskColor="rgba(0, 0, 0, 0.5)"
        />

        {/* Legend + Selected Edge Panel */}
        <Panel position="top-left" className="flex flex-col gap-2 max-w-xs">
          {isMobile ? (
            <div ref={reportingHelpRef} className="relative">
              <button
                type="button"
                onClick={() => setShowReportingHelp(!showReportingHelp)}
                className={cn(
                  'p-2 rounded-lg bg-card border border-border transition-colors',
                  showReportingHelp ? 'text-foreground' : 'text-muted-foreground hover:text-foreground'
                )}
                title="Reporting lines help"
                aria-label="Reporting lines help"
              >
                <Info className="h-4 w-4" />
              </button>
              {showReportingHelp && (
                <div className="absolute top-full left-0 mt-1 p-2 rounded-lg bg-card border border-border shadow-lg z-10 w-56">
                  <p className="text-xs font-medium mb-1">Reporting Lines</p>
                  <p className="text-xs text-muted-foreground">
                    Drag from a manager to a report to set relationships. Each member can report to one
                    manager. Click an edge to remove it.
                  </p>
                </div>
              )}
            </div>
          ) : (
            <div className="flex items-start gap-2 p-2 rounded-lg bg-card border border-border">
              <Info className="h-4 w-4 text-muted-foreground mt-0.5" />
              <div className="space-y-1">
                <p className="text-xs font-medium">Reporting Lines</p>
                <p className="text-xs text-muted-foreground">
                  Drag from a manager to a report to set relationships. Each member can report to one
                  manager. Click an edge to remove it.
                </p>
              </div>
            </div>
          )}
          {selectedEdge && (
            <div className="flex items-center justify-between gap-2 p-2 rounded-lg bg-card border border-border">
              <div className="min-w-0">
                <p className="text-xs font-medium">Selected Relationship</p>
                <p className="text-xs text-muted-foreground truncate">
                  {selectedManagerName} → {selectedReportName}
                </p>
              </div>
              <button
                type="button"
                onClick={handleRemoveSelectedEdge}
                className={cn(
                  'p-1.5 rounded-md border border-border',
                  'text-destructive hover:bg-destructive/10 transition-colors'
                )}
                title="Remove relationship"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            </div>
          )}
        </Panel>

        {/* Toolbar Panel */}
        <Panel position="top-right" className="flex gap-2">
          {onSwitchToCode && (
            <button
              type="button"
              onClick={onSwitchToCode}
              className={cn(
                'flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg',
                'bg-card border border-border text-foreground',
                'hover:bg-muted transition-colors'
              )}
              title="Switch to code view"
              aria-label="Code View"
            >
              <Code className="h-3.5 w-3.5" />
              {!isMobile && <span>Code View</span>}
            </button>
          )}
          <button
            type="button"
            onClick={handleAutoLayout}
            className={cn(
              'flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg',
              'bg-card border border-border text-foreground',
              'hover:bg-muted transition-colors'
            )}
            title={`Switch layout direction (${layoutDirection === 'TB' ? 'vertical' : 'horizontal'})`}
            aria-label="Toggle layout direction"
          >
            <LayoutGrid className="h-3.5 w-3.5" />
            {!isMobile && <span>{layoutDirection === 'TB' ? 'Layout: Vertical' : 'Layout: Horizontal'}</span>}
          </button>
          <button
            type="button"
            onClick={onAddMember}
            className={cn(
              'flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg',
              'bg-primary text-primary-foreground hover:bg-primary/90 transition-colors'
            )}
            title="Add member"
            aria-label="Add Member"
          >
            <UserPlus className="h-3.5 w-3.5" />
            {!isMobile && <span>Add Member</span>}
          </button>
        </Panel>
      </ReactFlow>
    </div>
  )
}
