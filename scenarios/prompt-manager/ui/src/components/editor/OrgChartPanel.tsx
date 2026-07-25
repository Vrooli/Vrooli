/**
 * OrgChartPanel - React Flow canvas for org chart visualization.
 *
 * Features:
 * - Dagre hierarchical layout
 * - Node selection -> detail panel
 * - Edge drag-drop for manager reassignment
 * - Toolbar with Add Member, auto-layout, zoom controls
 */

import { memo, useCallback, useMemo, useEffect, useRef, useState } from 'react'
import {
  useNodesState,
  useEdgesState,
  type Connection,
  type Edge,
  type OnConnect,
  type EdgeMouseHandler,
  type Node as RFNode,
  type NodeProps,
  Panel,
  MarkerType,
} from '@xyflow/react'
import { UserPlus, Users, Info, Trash2, AlertTriangle } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useIsMobile } from '@/hooks/useMediaQuery'
import { OrgChartNode } from './OrgChartNode'
import { FlowShell, layoutFlowDagre } from '@/components/graph/FlowShell'
import type { TeamDetails } from '@/types/team'
import type { Agent, AgentAppearance } from '@/types/agent'
import type { OrgEdge, OrgChartNode as OrgChartNodeType, OrgChartFlowEdge, OrgChartNodeData } from '@/types/orgChart'

import * as heartbeatService from '@/services/heartbeatService'
import type { HeartbeatConfig } from '@/services/heartbeatService'
import { useRunningAgentsStore } from '@/stores/runningAgentsStore'

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
  /** Controlled layout direction (TB vertical, LR horizontal). Default 'TB'. */
  layoutDirection?: LayoutDirection
  className?: string
}

// ============================================================================
// Layout Constants
// ============================================================================

const NODE_WIDTH = 200
const NODE_HEIGHT = 80
const NODE_GAP_X = 32
const NODE_GAP_Y = 24
const GROUP_HEADER_HEIGHT = 28
const GROUP_GAP = 24
type LayoutDirection = 'TB' | 'LR'

// ============================================================================
// Node Types
// ============================================================================

interface OrgGroupHeaderData extends Record<string, unknown> {
  label: string
  roleId: string
}

type OrgGroupHeaderNode = RFNode<OrgGroupHeaderData, 'orgGroup'>
type OrgChartFlowNode = OrgChartNodeType | OrgGroupHeaderNode

const OrgGroupHeader = memo(function OrgGroupHeader({
  data,
}: NodeProps<OrgGroupHeaderNode>) {
  return (
    <div
      className="px-2 py-0.5 text-[11px] uppercase tracking-wide text-muted-foreground/80 font-medium pointer-events-none select-none"
      data-testid={`org-group-header-${data.roleId}`}
    >
      {data.label}
    </div>
  )
})

const nodeTypes = {
  orgMember: OrgChartNode,
  orgGroup: OrgGroupHeader,
}

// ============================================================================
// Layout Function
// ============================================================================

/**
 * Wrapped grid layout for hierarchies with zero edges. Members are placed in
 * rows grouped by their first role, with a labeled header node per group so
 * the label tracks the React-Flow viewport during pan/zoom.
 */
function getGridLayout(
  nodes: OrgChartNodeType[],
  isMobile: boolean,
): { nodes: OrgChartFlowNode[] } {
  const cols = isMobile ? 2 : 4
  const stepX = NODE_WIDTH + NODE_GAP_X
  const stepY = NODE_HEIGHT + NODE_GAP_Y

  // Group by first role.
  const order: string[] = []
  const groups = new Map<string, OrgChartNodeType[]>()
  for (const node of nodes) {
    const roles = node.data.member.roles
    const groupKey = roles[0] ?? 'unassigned'
    let group = groups.get(groupKey)
    if (!group) {
      group = []
      groups.set(groupKey, group)
      order.push(groupKey)
    }
    group.push(node)
  }

  const placedNodes: OrgChartFlowNode[] = []
  let yCursor = 0

  for (const key of order) {
    const groupNodes = groups.get(key) ?? []
    const label = labelForRoleKey(key, groupNodes[0]?.data.teamRoles ?? [])
    placedNodes.push({
      id: `__group__${key}`,
      type: 'orgGroup',
      position: { x: 0, y: yCursor },
      data: { label, roleId: key },
      draggable: false,
      selectable: false,
      connectable: false,
      width: NODE_WIDTH * cols,
      height: GROUP_HEADER_HEIGHT,
    })
    yCursor += GROUP_HEADER_HEIGHT
    groupNodes.forEach((node, idx) => {
      const col = idx % cols
      const row = Math.floor(idx / cols)
      placedNodes.push({
        ...node,
        position: {
          x: col * stepX,
          y: yCursor + row * stepY,
        },
      })
    })
    const rows = Math.ceil(groupNodes.length / cols) || 1
    yCursor += rows * stepY + GROUP_GAP
  }

  return { nodes: placedNodes }
}

function labelForRoleKey(key: string, teamRoles: { id: string; name: string }[]): string {
  if (key === 'unassigned') return 'Unassigned'
  const match = teamRoles.find((r) => r.id === key)
  return match?.name ?? key
}

function getLayoutedElements(
  nodes: OrgChartNodeType[],
  edges: OrgChartFlowEdge[],
  direction: LayoutDirection,
): { nodes: OrgChartNodeType[]; edges: OrgChartFlowEdge[] } {
  return layoutFlowDagre(nodes, edges, { direction, nodeWidth: NODE_WIDTH, nodeHeight: NODE_HEIGHT, nodeSep: 50, rankSep: 80 })
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
  layoutDirection = 'TB',
  className,
}: OrgChartPanelProps) {
  const [selectedEdgeId, setSelectedEdgeId] = useState<string | null>(null)
  const [showReportingHelp, setShowReportingHelp] = useState(false)
  const reportingHelpRef = useRef<HTMLDivElement>(null)
  const isMobile = useIsMobile()
  const reportingMode = team.coordination.reportingMode
  const isLeaderless = reportingMode === 'none'

  // Heartbeat configs keyed by agentId
  const [heartbeatConfigs, setHeartbeatConfigs] = useState<Map<string, HeartbeatConfig>>(new Map())
  const runningAgentMap = useRunningAgentsStore((s) => s.agentMap)

  useEffect(() => {
    let isActive = true
    heartbeatService.listHeartbeats(team.id).then((configs) => {
      if (!isActive) return
      const map = new Map<string, HeartbeatConfig>()
      for (const cfg of configs) {
        map.set(cfg.agentId, cfg)
      }
      setHeartbeatConfigs(map)
    }).catch((err: unknown) => {
      console.warn('Failed to load heartbeat configs for org chart:', err)
    })
    return () => { isActive = false }
  }, [team.id])

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
      const hbConfig = heartbeatConfigs.get(member.agentId)
      const isRunning = runningAgentMap.has(member.agentId)

      // Determine heartbeat status: running agent takes priority, then last execution
      let heartbeatStatus: 'running' | 'completed' | 'failed' | 'cancelled' | null = null
      if (hbConfig) {
        if (isRunning) {
          heartbeatStatus = 'running'
        } else if (hbConfig.lastExecution) {
          heartbeatStatus = hbConfig.lastExecution.status
        }
      }

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
          heartbeatEnabled: hbConfig?.enabled,
          heartbeatStatus,
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
    heartbeatConfigs,
    runningAgentMap,
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

  // Apply layout: dagre when edges exist, wrapped grid when leaderless/empty.
  const usingGridLayout = initialEdges.length === 0
  const { nodes: layoutedNodes, edges: layoutedEdges } = useMemo<{
    nodes: OrgChartFlowNode[]
    edges: OrgChartFlowEdge[]
  }>(() => {
    if (usingGridLayout) {
      const { nodes: gridNodes } = getGridLayout(initialNodes, isMobile)
      return { nodes: gridNodes, edges: initialEdges }
    }
    const dagre = getLayoutedElements(initialNodes, initialEdges, layoutDirection)
    return { nodes: dagre.nodes, edges: dagre.edges }
  }, [initialNodes, initialEdges, layoutDirection, usingGridLayout, isMobile])

  // reportingMode-aware info chip variant.
  const reportingChip = useMemo(() => {
    const hasEdges = initialEdges.length > 0
    if (isLeaderless && !hasEdges) {
      return {
        tone: 'info' as const,
        title: 'Leaderless team',
        message:
          'No reporting lines expected for this team. See the Topics view for cross-member message flow.',
      }
    }
    if (isLeaderless && hasEdges) {
      return {
        tone: 'warning' as const,
        title: 'Reporting lines + leaderless team',
        message:
          'Reporting lines are defined but the team is set to leaderless. Either change reportingMode to org-chart, or remove the edges.',
      }
    }
    if (!isLeaderless && !hasEdges) {
      return {
        tone: 'warning' as const,
        title: 'No reporting lines yet',
        message:
          'No reporting lines defined yet. Drag from a manager handle to a report to start.',
      }
    }
    return null
  }, [isLeaderless, initialEdges.length])

  // React Flow state - use type assertions since hooks don't preserve generics
  const [nodes, setNodes, onNodesChange] = useNodesState<OrgChartFlowNode>(layoutedNodes)
  const [flowEdges, setEdges, onEdgesChange] = useEdgesState<OrgChartFlowEdge>(layoutedEdges)

  // Update nodes when selection or data changes (member nodes only)
  useEffect(() => {
    const updatedNodes: OrgChartFlowNode[] = layoutedNodes.map((node) => {
      if (node.type !== 'orgMember') return node
      return {
        ...node,
        data: {
          ...node.data,
          isSelected: node.id === selectedMemberId,
        },
      }
    })
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
          <button
            type="button"
            onClick={onAddMember}
            className={cn(
              'inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg',
              'bg-primary text-primary-foreground hover:bg-primary/90 transition-colors',
            )}
          >
            <UserPlus className="h-4 w-4" />
            Add First Member
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className={cn('h-full flex flex-col', className)}>
      {reportingChip && (
        <div
          className={cn(
            'flex-shrink-0 flex items-start gap-2 mx-2 mt-2 p-2 rounded-lg border',
            reportingChip.tone === 'warning'
              ? 'bg-amber-500/10 border-amber-500/30'
              : 'bg-card border-border',
          )}
          data-testid={`reporting-chip-${reportingChip.tone}`}
        >
          {reportingChip.tone === 'warning' ? (
            <AlertTriangle className="h-4 w-4 text-amber-400 mt-0.5 flex-shrink-0" />
          ) : (
            <Info className="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
          )}
          <div className="min-w-0">
            <p className="text-xs font-medium">{reportingChip.title}</p>
            <p className="text-xs text-muted-foreground">{reportingChip.message}</p>
          </div>
        </div>
      )}
      <div className="flex-1 min-h-0 relative">
        <FlowShell
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
          miniMapNodeColor={(node) => {
            const data = node.data as OrgChartNodeData
            return data.appearance?.body ?? '#6366f1'
          }}
        >

          {/* Reporting-lines help (only shown when dagre layout is active — drag-drop is disabled in grid mode) */}
          {!usingGridLayout && (
            <Panel position="top-left" className="flex flex-col gap-2 max-w-xs">
              {isMobile ? (
                <div ref={reportingHelpRef} className="relative">
                  <button
                    type="button"
                    onClick={() => setShowReportingHelp(!showReportingHelp)}
                    className={cn(
                      'p-2 rounded-lg bg-card border border-border transition-colors',
                      showReportingHelp ? 'text-foreground' : 'text-muted-foreground hover:text-foreground',
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
                      'text-destructive hover:bg-destructive/10 transition-colors',
                    )}
                    title="Remove relationship"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              )}
            </Panel>
          )}

        </FlowShell>
      </div>
    </div>
  )
}
