/**
 * Topics graph types for the dual-mode team graph view.
 *
 * Mirrors `api/memberflow/handlers.go` GraphResponse + ValidationResult.
 * DOC: docs/agent-system/TOPICS_SCHEMA.md
 */

import type { Node, Edge } from '@xyflow/react'

export type TopicNodeKind =
  | 'member'
  | 'external'
  | 'decision'
  | 'por_file'
  | 'capability_gap'
  | 'skill_proposal'
  | 'backlog'
  | 'knowledge_sink'

export type TopicEdgeKind =
  | 'intake'
  | 'output'
  | 'decision_owned'
  | 'decision_consumed'
  | 'external_producer'
  | 'capability_gap'

export interface TopicMemberRef {
  team: string
  member: string
}

export interface TopicIntakeEntry {
  prefix: string
  taxonomy?: string
  classifier_skill?: string
  source_team?: string | null
}

export interface TopicOutputEntry {
  prefix: string
  destination_kind: string
  destination_team?: string | null
  destination_path?: string | null
  schema?: string
}

export interface TopicDeclaration {
  intake?: TopicIntakeEntry[]
  output?: TopicOutputEntry[]
  decisions_owned?: string[]
  decisions_consumed?: string[]
  raises_capability_gaps?: boolean
  external_producers?: string[]
}

export interface TopicGraphNode {
  kind: TopicNodeKind
  id: string
  label?: string
  ref?: TopicMemberRef
  topics?: TopicDeclaration
}

export interface TopicGraphEdge {
  from: string
  to: string
  prefix: string
  kind: TopicEdgeKind
}

export type TopicSeverity = 'error' | 'warning'

export interface TopicFinding {
  rule: string
  severity: TopicSeverity
  member: TopicMemberRef
  prefix?: string
  detail: string
}

export interface TopicValidation {
  findings: TopicFinding[]
  errors: number
  warnings: number
}

export interface TopicsGraphResponse {
  nodes: TopicGraphNode[]
  edges: TopicGraphEdge[]
  validation: TopicValidation
}

/** React Flow data payload for a topics node. */
export interface TopicsFlowNodeData extends Record<string, unknown> {
  graphNode: TopicGraphNode
  errorCount: number
  warningCount: number
  isSelected: boolean
  onSelect?: (nodeId: string) => void
}

export type TopicsFlowNode = Node<TopicsFlowNodeData, 'topicsNode'>

/** React Flow data payload for a topics edge. */
export interface TopicsFlowEdgeData extends Record<string, unknown> {
  graphEdge: TopicGraphEdge
}

export type TopicsFlowEdge = Edge<TopicsFlowEdgeData>

export interface TopicsDrainStatusResponse {
  note?: string
  [key: string]: unknown
}
