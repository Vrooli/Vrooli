import type { Edge, Node } from "@xyflow/react";
import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type { GraphLens, EntityType } from "../surfaces/graph/stores/graph-data-store";

interface GraphAPIPosition {
  x: number;
  y: number;
}

interface GraphAPINode {
  id: string;
  type: string;
  data?: Record<string, unknown>;
  position?: GraphAPIPosition;
}

interface GraphAPIEdge {
  id: string;
  source: string;
  target: string;
  type: string;
}

interface GraphAPIMeta {
  lens: GraphLens;
  node_count: number;
  edge_count: number;
  generated_at: string;
  agent_manager_available?: boolean;
}

interface GraphAPIResponse {
  nodes: GraphAPINode[];
  edges: GraphAPIEdge[];
  meta: GraphAPIMeta;
}

export interface GraphProjectionMeta {
  lens: GraphLens;
  nodeCount: number;
  edgeCount: number;
  generatedAt: string;
  agentManagerAvailable: boolean | null;
}

export interface GraphProjection {
  nodes: Node[];
  edges: Edge[];
  meta: GraphProjectionMeta;
}

export interface IGraphService {
  getGraph(lens: GraphLens): Promise<GraphProjection>;
}

const NODE_TYPE_MAP: Record<string, EntityType> = {
  BacklogItem: "backlog",
  Scenario: "scenario",
  ExecutionRecord: "execution",
  Capture: "capture",
  Run: "agent-run",
  Initiative: "initiative",
};

function truncate(value: string, maxLength: number): string {
  return value.length <= maxLength ? value : `${value.slice(0, maxLength - 3)}...`;
}

function shortId(value: string, maxLength = 8): string {
  return value.length <= maxLength ? value : value.slice(0, maxLength);
}

function normalizeNodeLabel(rawType: string, id: string, data: Record<string, unknown>): string {
  switch (rawType) {
    case "BacklogItem":
      return String(data.title ?? data.name ?? id);
    case "Scenario":
      return String(data.name ?? id.replace(/^scenario\//, ""));
    case "Initiative":
      return String(data.title ?? data.name ?? id.replace(/^initiative\//, ""));
    case "Capture":
      return truncate(String(data.text ?? id), 72);
    case "ExecutionRecord": {
      const backlogKind = typeof data.backlog_kind === "string" ? data.backlog_kind : null;
      const backlogName = typeof data.backlog_name === "string" ? data.backlog_name : null;
      if (backlogKind && backlogName) {
        return `${backlogKind}/${backlogName}`;
      }
      const executionId = typeof data.execution_id === "string" ? data.execution_id : id;
      return `Execution ${shortId(executionId)}`;
    }
    case "Run": {
      const runId = typeof data.run_id === "string" ? data.run_id : id;
      return `Run ${shortId(runId)}`;
    }
    default:
      return String(data.label ?? id);
  }
}

function normalizeNode(raw: GraphAPINode): Node {
  const entityType = NODE_TYPE_MAP[raw.type] ?? "backlog";
  const data = raw.data ?? {};

  return {
    id: raw.id,
    type: entityType,
    position: raw.position ?? { x: 0, y: 0 },
    data: {
      ...data,
      label: normalizeNodeLabel(raw.type, raw.id, data),
      entityType,
      rawType: raw.type,
    },
  };
}

function normalizeEdge(raw: GraphAPIEdge): Edge {
  return {
    id: raw.id,
    source: raw.source,
    target: raw.target,
    type: raw.type,
    data: {
      relationship: raw.type,
    },
  };
}

function normalizeMeta(meta: GraphAPIMeta): GraphProjectionMeta {
  return {
    lens: meta.lens,
    nodeCount: meta.node_count,
    edgeCount: meta.edge_count,
    generatedAt: meta.generated_at,
    agentManagerAvailable:
      typeof meta.agent_manager_available === "boolean" ? meta.agent_manager_available : null,
  };
}

export function createGraphService(apiClient: IApiClient = defaultApiClient): IGraphService {
  return {
    async getGraph(lens: GraphLens): Promise<GraphProjection> {
      const response = await apiClient.get<GraphAPIResponse>(
        `${API_ENDPOINTS.graph}?lens=${encodeURIComponent(lens)}`,
      );

      return {
        nodes: (response.nodes ?? []).map(normalizeNode),
        edges: (response.edges ?? []).map(normalizeEdge),
        meta: normalizeMeta(response.meta),
      };
    },
  };
}

export const graphService = createGraphService();
