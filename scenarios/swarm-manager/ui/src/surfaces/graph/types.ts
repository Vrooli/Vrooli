import type { Edge as FlowEdge, Node as FlowNode } from "@xyflow/react";
import {
  AGENT_ACTIVITY_STATUSES,
  AGENT_RUN_STATUSES,
  BACKLOG_STATUSES,
  CAPTURE_STATUSES,
  EXECUTION_STATUSES,
  INITIATIVE_STATUSES,
  SCENARIO_STATUSES,
  type AgentActivityInteractionType,
  type AgentActivityPurpose,
  type AgentActivityStatus,
  type BacklogKind,
  type BacklogStatus,
  type CaptureStatus,
  type ExecutionBacklogKind,
  type ExecutionMode,
  type ExecutionStatus,
} from "../../types";

export type GraphLens = "focus" | "topology" | "operations";

export type GraphEntityType =
  | "backlog"
  | "scenario"
  | "execution"
  | "agent-activity"
  | "capture"
  | "agent-run"
  | "initiative";

export type GraphGroupingMode = "initiative" | "none";

/**
 * Maps each entity type to its known status values.
 */
export const ENTITY_STATUS_REGISTRY: Partial<Record<GraphEntityType, readonly string[]>> = {
  backlog: BACKLOG_STATUSES,
  execution: EXECUTION_STATUSES,
  capture: CAPTURE_STATUSES,
  "agent-activity": AGENT_ACTIVITY_STATUSES,
  "agent-run": AGENT_RUN_STATUSES,
  scenario: SCENARIO_STATUSES,
  initiative: INITIATIVE_STATUSES,
};

export interface InitiativeRollupData {
  total: number;
  completed: number;
  in_progress: number;
  failed: number;
  pending: number;
}

interface GraphBaseNodeData {
  [key: string]: unknown;
  label: string;
  entityType: GraphEntityType;
  rawType: string;
  status?: string;
  kind?: string;
  pulsing?: boolean;
  pulseMode?: "oneshot" | "persistent";
  priority?: number;
}

export interface BacklogGraphNodeData extends GraphBaseNodeData {
  entityType: "backlog";
  rawType: "BacklogItem";
  kind: BacklogKind;
  name: string;
  title: string;
  status: BacklogStatus;
  priority: number;
  activeExecutionStatus?: string;
  activeExecutionCount?: number;
}

export interface InitiativeActiveRoundSummary {
  mode: string;
  phase: string;
  round: number;
  status: string;
}

export interface InitiativeGraphNodeData extends GraphBaseNodeData {
  entityType: "initiative";
  rawType: "Initiative";
  name: string;
  title: string;
  status: string;
  /** Operating mode of the initiative when an active round is in flight. */
  operatingMode?: string;
  /** First non-terminal round, or undefined when no round is active. */
  activeRound?: InitiativeActiveRoundSummary;
  rollup: InitiativeRollupData;
}

export interface CaptureGraphNodeData extends GraphBaseNodeData {
  entityType: "capture";
  rawType: "Capture";
  id: string;
  text: string;
  status: CaptureStatus;
}

export interface ScenarioGraphNodeData extends GraphBaseNodeData {
  entityType: "scenario";
  rawType: "Scenario";
  name: string;
  status: "running" | "stopped" | "error" | "unknown";
}

export interface ExecutionGraphNodeData extends GraphBaseNodeData {
  entityType: "execution";
  rawType: "ExecutionRecord";
  executionId: string;
  backlogKind: ExecutionBacklogKind;
  backlogName: string;
  status: ExecutionStatus;
  mode: ExecutionMode;
  runId?: string;
}

export interface AgentActivityGraphNodeData extends GraphBaseNodeData {
  entityType: "agent-activity";
  rawType: "AgentActivity";
  activityId: string;
  ownerType: "backlog" | "capture" | "scenario";
  ownerKind?: BacklogKind;
  ownerName: string;
  ownerTitle?: string;
  executionId?: string;
  purpose: AgentActivityPurpose;
  interactionType: AgentActivityInteractionType;
  status: AgentActivityStatus;
  requestedAt: string;
  runId?: string;
  taskId?: string;
}

export interface RunGraphNodeData extends GraphBaseNodeData {
  entityType: "agent-run";
  rawType: "Run";
  runId: string;
  taskId?: string;
  status: string;
}

export interface ClusterGraphNodeData extends GraphBaseNodeData {
  label: string;
  entityType: "initiative";
  rawType: "Cluster";
  collapsed: boolean;
  rollup: InitiativeRollupData | null;
  isUnassigned?: boolean;
  pulsing?: boolean;
}

export interface CappedGraphNodeData extends GraphBaseNodeData {
  entityType: "backlog";
  rawType: "Synthetic";
  status: "capped";
  isCapNode: true;
}

export type GraphNodeData =
  | BacklogGraphNodeData
  | InitiativeGraphNodeData
  | CaptureGraphNodeData
  | ScenarioGraphNodeData
  | ExecutionGraphNodeData
  | AgentActivityGraphNodeData
  | RunGraphNodeData
  | ClusterGraphNodeData
  | CappedGraphNodeData;

export interface GraphEdgeData {
  [key: string]: unknown;
  relationship?: string;
  relationshipType?: string;
  aggregatedCount?: number;
}

export type GraphNode = FlowNode<GraphNodeData>;
export type GraphEdge = FlowEdge;

export function getGraphNodeData(node: { data?: unknown }): GraphNodeData {
  return node.data as GraphNodeData;
}

export function getGraphNodeLabel(node: { id: string; data?: unknown }): string {
  const data = getGraphNodeData(node);
  return data.label ?? node.id;
}

export function getGraphNodeStatus(node: { data?: unknown }): string | undefined {
  return getGraphNodeData(node).status;
}

export function getGraphNodeEntityType(node: { data?: unknown }): GraphEntityType {
  return getGraphNodeData(node).entityType;
}

export function isClusterNodeData(data: GraphNodeData): data is ClusterGraphNodeData {
  return data.rawType === "Cluster";
}
