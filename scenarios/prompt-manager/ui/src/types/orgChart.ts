/**
 * Org Chart types for the team editor visualization.
 *
 * These types extend React Flow's node/edge types for our specific use case.
 */

import type { Node, Edge } from '@xyflow/react'
import type { TeamMember, TeamRole } from '@/types/team'
import type { AgentAppearance } from '@/types/agent'

/**
 * Data payload for an org chart node (team member).
 * Uses index signature to satisfy React Flow's Record<string, unknown> constraint.
 */
export interface OrgChartNodeData extends Record<string, unknown> {
  /** The team member data */
  member: TeamMember
  /** Agent appearance for avatar colors */
  appearance?: AgentAppearance
  /** Roles available in the team for display */
  teamRoles: TeamRole[]
  /** Whether this node is currently selected */
  isSelected: boolean
  /** Manager display name if assigned */
  managerName?: string
  /** Number of direct reports */
  directReportCount: number
  /** Callback when node is clicked */
  onSelect: (agentId: string) => void
}

/**
 * React Flow node for org chart visualization.
 */
export type OrgChartNode = Node<OrgChartNodeData, 'orgMember'>

/**
 * Edge representing a reporting relationship.
 * source = manager, target = report
 */
export interface OrgEdge {
  /** Unique edge ID */
  id: string
  /** Manager's agent ID */
  managerId: string
  /** Report's agent ID */
  reportId: string
}

/**
 * React Flow edge for org chart visualization.
 */
export type OrgChartFlowEdge = Edge<{ originalEdge: OrgEdge }>

/**
 * Full org chart data structure.
 */
export interface OrgChartData {
  /** Team members as nodes */
  nodes: OrgChartNode[]
  /** Reporting relationships as edges */
  edges: OrgChartFlowEdge[]
}

/**
 * Member documents (responsibilities and heartbeat instructions).
 */
export interface MemberDocs {
  /** RESPONSIBILITIES.md content */
  responsibilities: string
  /** HEARTBEAT.md content */
  heartbeatInstructions: string
}

/**
 * Backend API response format for org chart.
 * Field names match Go struct JSON tags (camelCase).
 */
export interface OrgChartApiResponse {
  teamId: string
  edges: Array<{
    managerAgentId: string
    reportAgentId: string
  }>
}

/**
 * Request to set all org chart edges (backend format).
 */
export interface SetOrgChartRequest {
  edges: Array<{
    managerAgentId: string
    reportAgentId: string
  }>
}

/**
 * Request to update an edge (change manager).
 * Used by the frontend API layer.
 */
export interface UpdateEdgeRequest {
  managerId: string | null
}
