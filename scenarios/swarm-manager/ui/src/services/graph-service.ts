import type {
  GraphEdge as ProtoGraphEdge,
  GraphNode as ProtoGraphNode,
} from "@vrooli/proto-types/swarm-manager/v1/domain/graph_pb";
import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import {
  graphResponseSchema,
  parseProtoResponse,
  requireProtoField,
} from "./proto-contracts";
import type {
  GraphEdge,
  GraphEntityType,
  GraphLens,
  GraphNode,
  InitiativeRollupData,
} from "../surfaces/graph/types";

export interface GraphProjectionMeta {
  lens: GraphLens;
  nodeCount: number;
  edgeCount: number;
  generatedAt: string;
  agentManagerAvailable: boolean | null;
  hint: string | null;
}

export interface GraphProjection {
  nodes: GraphNode[];
  edges: GraphEdge[];
  meta: GraphProjectionMeta;
}

export interface GraphRequestOptions {
  signal?: AbortSignal;
}

export interface IGraphService {
  getGraph(lens: GraphLens, options?: GraphRequestOptions): Promise<GraphProjection>;
}

const NODE_TYPE_MAP: Record<string, GraphEntityType> = {
  BacklogItem: "backlog",
  Scenario: "scenario",
  ExecutionRecord: "execution",
  AgentActivity: "agent-activity",
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

function mapRollup(proto?: {
  total: number;
  completed: number;
  inProgress: number;
  failed: number;
  pending: number;
}): InitiativeRollupData {
  return {
    total: proto?.total ?? 0,
    completed: proto?.completed ?? 0,
    in_progress: proto?.inProgress ?? 0,
    failed: proto?.failed ?? 0,
    pending: proto?.pending ?? 0,
  };
}

function normalizePosition(raw: ProtoGraphNode["position"]): { x: number; y: number } {
  return {
    x: raw?.x ?? 0,
    y: raw?.y ?? 0,
  };
}

function mapProtoNode(raw: ProtoGraphNode): GraphNode {
  const data = requireProtoField(raw.data, "graph node data");
  const entityType = NODE_TYPE_MAP[raw.type] ?? "backlog";
  const position = normalizePosition(raw.position);

  switch (data.value.case) {
    case "backlog": {
      const backlog = data.value.value;
      // activeExecutionStatus/Count are set by proto_response.go but not yet
      // reflected in the generated proto TS types (proto alignment is a follow-up).
      const backlogAny = backlog as Record<string, unknown>;
      return {
        id: raw.id,
        type: entityType,
        position,
        data: {
          entityType: "backlog",
          rawType: "BacklogItem",
          label: backlog.title || backlog.name || raw.id,
          kind: backlog.kind as "idea" | "research" | "fix" | "execute" | "chore",
          name: backlog.name,
          title: backlog.title,
          status: backlog.status as
            | "backlog"
            | "researching"
            | "ready"
            | "queued"
            | "in_progress"
            | "completed"
            | "failed",
          priority: backlog.priority,
          activeExecutionStatus: (backlogAny.activeExecutionStatus as string) ?? undefined,
          activeExecutionCount: (backlogAny.activeExecutionCount as number) ?? undefined,
        },
      };
    }
    case "initiative": {
      const initiative = data.value.value;
      return {
        id: raw.id,
        type: entityType,
        position,
        data: {
          entityType: "initiative",
          rawType: "Initiative",
          label: initiative.title || initiative.name || raw.id,
          name: initiative.name,
          title: initiative.title,
          status: initiative.status,
          rollup: mapRollup(initiative.rollup),
        },
      };
    }
    case "capture": {
      const capture = data.value.value;
      return {
        id: raw.id,
        type: entityType,
        position,
        data: {
          entityType: "capture",
          rawType: "Capture",
          label: truncate(capture.text || raw.id, 72),
          id: capture.id,
          text: capture.text,
          status: capture.status as "classifying" | "classified" | "failed",
        },
      };
    }
    case "scenario": {
      const scenario = data.value.value;
      return {
        id: raw.id,
        type: entityType,
        position,
        data: {
          entityType: "scenario",
          rawType: "Scenario",
          label: scenario.name || raw.id.replace(/^scenario\//, ""),
          name: scenario.name,
          status: scenario.status as "running" | "stopped" | "error" | "unknown",
        },
      };
    }
    case "execution": {
      const execution = data.value.value;
      return {
        id: raw.id,
        type: entityType,
        position,
        data: {
          entityType: "execution",
          rawType: "ExecutionRecord",
          label: `${execution.backlogKind}/${execution.backlogName}`,
          executionId: execution.executionId,
          backlogKind: execution.backlogKind as "idea" | "research" | "fix" | "execute" | "chore" | "spec-sync",
          backlogName: execution.backlogName,
          status: execution.status as
            | "pending"
            | "starting"
            | "running"
            | "needs_review"
            | "validating"
            | "needs_fixup"
            | "completed"
            | "failed"
            | "canceled",
          mode: execution.mode as "manual" | "yolo",
          runId: execution.runId,
        },
      };
    }
    case "activity": {
      const activity = data.value.value;
      return {
        id: raw.id,
        type: entityType,
        position,
        data: {
          entityType: "agent-activity",
          rawType: "AgentActivity",
          label: activity.ownerTitle || activity.purpose.replace("_", " "),
          activityId: activity.activityId,
          ownerType: activity.ownerType as "backlog" | "capture" | "scenario",
          ownerKind: activity.ownerKind as "idea" | "research" | "fix" | "execute" | "chore" | undefined,
          ownerName: activity.ownerName,
          ownerTitle: activity.ownerTitle,
          executionId: activity.executionId,
          purpose: activity.purpose as
            | "initialize"
            | "workshop"
            | "finalize"
            | "research"
            | "process"
            | "fixup"
            | "followup"
            | "spec_sync"
            | "classify",
          interactionType: activity.interactionType as "spawn" | "continue",
          status: activity.status as
            | "pending"
            | "starting"
            | "running"
            | "needs_review"
            | "complete"
            | "failed"
            | "cancelled"
            | "unspecified",
          requestedAt: activity.requestedAt,
          runId: activity.runId,
          taskId: activity.taskId,
          kind: activity.purpose,
        },
      };
    }
    case "run": {
      const run = data.value.value;
      return {
        id: raw.id,
        type: entityType,
        position,
        data: {
          entityType: "agent-run",
          rawType: "Run",
          label: `Run ${shortId(run.runId)}`,
          runId: run.runId,
          taskId: run.taskId,
          status: run.status,
        },
      };
    }
    default:
      console.warn(`[graph-service] Unknown graph node data case for node ${raw.id}, skipping`);
      return {
        id: raw.id,
        type: entityType,
        position,
        data: {
          entityType: "scenario",
          rawType: "Scenario",
          label: raw.id,
          name: raw.id,
          status: "unknown" as const,
        },
      };
  }
}

function mapProtoEdge(raw: ProtoGraphEdge): GraphEdge {
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

function normalizeMeta(meta: {
  lens: string;
  nodeCount: number;
  edgeCount: number;
  generatedAt: string;
  agentManagerAvailable?: boolean;
  hint?: string;
}): GraphProjectionMeta {
  return {
    lens: meta.lens as GraphLens,
    nodeCount: meta.nodeCount,
    edgeCount: meta.edgeCount,
    generatedAt: meta.generatedAt,
    agentManagerAvailable:
      typeof meta.agentManagerAvailable === "boolean" ? meta.agentManagerAvailable : null,
    hint: meta.hint ?? null,
  };
}

export function createGraphService(apiClient: IApiClient = defaultApiClient): IGraphService {
  return {
    async getGraph(lens: GraphLens, options?: GraphRequestOptions): Promise<GraphProjection> {
      const url = `${API_ENDPOINTS.graph}?lens=${encodeURIComponent(lens)}`;
      const data = await apiClient.get<unknown>(url, { signal: options?.signal });
      const parsed = parseProtoResponse(graphResponseSchema, data, "graph");

      return {
        nodes: parsed.nodes.map(mapProtoNode),
        edges: parsed.edges.map(mapProtoEdge),
        meta: normalizeMeta(requireProtoField(parsed.meta, "graph meta")),
      };
    },
  };
}

export const graphService = createGraphService();
