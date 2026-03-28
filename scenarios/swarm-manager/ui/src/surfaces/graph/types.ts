import type { Edge as FlowEdge, Node as FlowNode } from "@xyflow/react";
import type {
  BacklogKind,
  BacklogStatus,
  CaptureStatus,
  ExecutionMode,
  ExecutionStatus,
} from "../../types";

export type GraphLens = "topology" | "flow" | "operations";

export type GraphEntityType =
  | "backlog"
  | "scenario"
  | "execution"
  | "capture"
  | "agent-run"
  | "initiative";

export type GraphGroupingMode = "initiative" | "none";

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
}

export interface InitiativeGraphNodeData extends GraphBaseNodeData {
  entityType: "initiative";
  rawType: "Initiative";
  name: string;
  title: string;
  status: string;
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
  backlogKind: BacklogKind;
  backlogName: string;
  status: ExecutionStatus;
  mode: ExecutionMode;
  runId?: string;
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
export type GraphEdge = FlowEdge<GraphEdgeData>;

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
