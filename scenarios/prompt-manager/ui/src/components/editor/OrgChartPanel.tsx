/**
 * OrgChartPanel - React Flow canvas for org chart visualization.
 *
 * Features:
 * - Dagre hierarchical layout
 * - Node selection -> detail panel
 * - Edge drag-drop for manager reassignment
 * - Toolbar with Add Member, auto-layout, zoom controls
 */

import { useCallback, useMemo, useEffect } from 'react'
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
  Panel,
  MarkerType,
} from '@xyflow/react'
import dagre from '@dagrejs/dagre'
import { UserPlus, LayoutGrid, Users } from 'lucide-react'
import { cn } from '@/lib/utils'
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
  className?: string
}

// ============================================================================
// Layout Constants
// ============================================================================

const NODE_WIDTH = 200
const NODE_HEIGHT = 80
const LAYOUT_DIRECTION = 'TB' // Top to bottom

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
  className,
}: OrgChartPanelProps) {
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

  // Convert team members to React Flow nodes
  const initialNodes = useMemo((): OrgChartNodeType[] => {
    return team.members.map((member): OrgChartNodeType => ({
      id: member.agentId,
      type: 'orgMember',
      position: { x: 0, y: 0 }, // Will be set by dagre
      data: {
        member,
        appearance: agentAppearances.get(member.agentId),
        teamRoles: team.roles,
        isSelected: member.agentId === selectedMemberId,
        onSelect: onSelectMember,
      },
    }))
  }, [team.members, team.roles, agentAppearances, selectedMemberId, onSelectMember])

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
    () => getLayoutedElements(initialNodes, initialEdges),
    [initialNodes, initialEdges]
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

  // Handle new edge connection (drag to connect)
  const onConnect: OnConnect = useCallback(
    (connection: Connection) => {
      if (connection.source && connection.target) {
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
    },
    [onEdgeUpdate]
  )

  // Handle auto-layout button
  const handleAutoLayout = useCallback(() => {
    const { nodes: newNodes, edges: newEdges } = getLayoutedElements(nodes, flowEdges)
    setNodes(newNodes)
    setEdges(newEdges)
  }, [nodes, flowEdges, setNodes, setEdges])

  // Handle background click to deselect
  const onPaneClick = useCallback(() => {
    onSelectMember(null)
  }, [onSelectMember])

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
              'bg-primary text-primary-foreground hover:bg-primary/90 transition-colors'
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
    <div className={cn('h-full', className)}>
      <ReactFlow
        nodes={nodes}
        edges={flowEdges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onEdgesDelete={onEdgeDelete}
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

        {/* Toolbar Panel */}
        <Panel position="top-right" className="flex gap-2">
          <button
            type="button"
            onClick={handleAutoLayout}
            className={cn(
              'flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg',
              'bg-card border border-border text-foreground',
              'hover:bg-muted transition-colors'
            )}
            title="Auto-layout nodes"
          >
            <LayoutGrid className="h-3.5 w-3.5" />
            Layout
          </button>
          <button
            type="button"
            onClick={onAddMember}
            className={cn(
              'flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg',
              'bg-primary text-primary-foreground hover:bg-primary/90 transition-colors'
            )}
          >
            <UserPlus className="h-3.5 w-3.5" />
            Add Member
          </button>
        </Panel>
      </ReactFlow>
    </div>
  )
}
