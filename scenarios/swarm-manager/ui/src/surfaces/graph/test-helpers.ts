import type {
  BacklogKind,
  BacklogStatus,
  CaptureStatus,
  ExecutionMode,
  ExecutionStatus,
} from "../../types";
import type {
  BacklogGraphNodeData,
  CaptureGraphNodeData,
  ClusterGraphNodeData,
  ExecutionGraphNodeData,
  GraphEdge,
  GraphEntityType,
  GraphNode,
  InitiativeGraphNodeData,
  RunGraphNodeData,
  ScenarioGraphNodeData,
} from "./types";

const DEFAULT_POSITION = { x: 0, y: 0 };

function lastSegment(id: string): string {
  const parts = id.split("/");
  return parts[parts.length - 1] || id;
}

function parseBacklogId(id: string): { kind: BacklogKind; name: string } {
  const [, rawKind, ...nameParts] = id.split("/");
  const kind = (rawKind || "execute") as BacklogKind;
  const name = nameParts.join("/") || lastSegment(id);
  return { kind, name };
}

export function makeBacklogNode(
  id: string,
  overrides: Partial<BacklogGraphNodeData> = {},
): GraphNode {
  const { kind, name } = parseBacklogId(id);
  const data: BacklogGraphNodeData = {
    label: overrides.label ?? overrides.title ?? name,
    entityType: "backlog",
    rawType: "BacklogItem",
    kind,
    name,
    title: overrides.title ?? name,
    status: overrides.status ?? ("backlog" as BacklogStatus),
    priority: overrides.priority ?? 0,
    pulsing: overrides.pulsing,
    ...overrides,
  };

  return {
    id,
    type: "backlog",
    position: DEFAULT_POSITION,
    data,
  };
}

export function makeInitiativeNode(
  id: string,
  overrides: Partial<InitiativeGraphNodeData> = {},
): GraphNode {
  const name = id.replace(/^initiative\//, "") || lastSegment(id);
  const data: InitiativeGraphNodeData = {
    label: overrides.label ?? overrides.title ?? name,
    entityType: "initiative",
    rawType: "Initiative",
    name,
    title: overrides.title ?? name,
    status: overrides.status ?? "active",
    rollup: overrides.rollup ?? {
      total: 0,
      completed: 0,
      in_progress: 0,
      failed: 0,
      pending: 0,
    },
    pulsing: overrides.pulsing,
    ...overrides,
  };

  return {
    id,
    type: "initiative",
    position: DEFAULT_POSITION,
    data,
  };
}

export function makeCaptureNode(
  id: string,
  overrides: Partial<CaptureGraphNodeData> = {},
): GraphNode {
  const captureId = overrides.id ?? lastSegment(id);
  const text = overrides.text ?? `Capture ${captureId}`;
  const data: CaptureGraphNodeData = {
    label: overrides.label ?? text,
    entityType: "capture",
    rawType: "Capture",
    id: captureId,
    text,
    status: overrides.status ?? ("classified" as CaptureStatus),
    pulsing: overrides.pulsing,
    ...overrides,
  };

  return {
    id,
    type: "capture",
    position: DEFAULT_POSITION,
    data,
  };
}

export function makeScenarioNode(
  id: string,
  overrides: Partial<ScenarioGraphNodeData> = {},
): GraphNode {
  const name = id.replace(/^scenario\//, "") || lastSegment(id);
  const data: ScenarioGraphNodeData = {
    label: overrides.label ?? name,
    entityType: "scenario",
    rawType: "Scenario",
    name,
    status: overrides.status ?? "running",
    pulsing: overrides.pulsing,
    ...overrides,
  };

  return {
    id,
    type: "scenario",
    position: DEFAULT_POSITION,
    data,
  };
}

export function makeExecutionNode(
  id: string,
  overrides: Partial<ExecutionGraphNodeData> = {},
): GraphNode {
  const executionId = overrides.executionId ?? lastSegment(id);
  const backlogKind = overrides.backlogKind ?? ("execute" as BacklogKind);
  const backlogName = overrides.backlogName ?? "task";
  const data: ExecutionGraphNodeData = {
    label: overrides.label ?? `${backlogKind}/${backlogName}`,
    entityType: "execution",
    rawType: "ExecutionRecord",
    executionId,
    backlogKind,
    backlogName,
    status: overrides.status ?? ("running" as ExecutionStatus),
    mode: overrides.mode ?? ("manual" as ExecutionMode),
    runId: overrides.runId,
    pulsing: overrides.pulsing,
    ...overrides,
  };

  return {
    id,
    type: "execution",
    position: DEFAULT_POSITION,
    data,
  };
}

export function makeRunNode(
  id: string,
  overrides: Partial<RunGraphNodeData> = {},
): GraphNode {
  const runId = overrides.runId ?? lastSegment(id);
  const data: RunGraphNodeData = {
    label: overrides.label ?? `Run ${runId}`,
    entityType: "agent-run",
    rawType: "Run",
    runId,
    taskId: overrides.taskId,
    status: overrides.status ?? "running",
    pulsing: overrides.pulsing,
    ...overrides,
  };

  return {
    id,
    type: "agent-run",
    position: DEFAULT_POSITION,
    data,
  };
}

export function makeGraphNode(
  id: string,
  entityType: GraphEntityType,
  overrides:
    | Partial<BacklogGraphNodeData>
    | Partial<InitiativeGraphNodeData>
    | Partial<CaptureGraphNodeData>
    | Partial<ScenarioGraphNodeData>
    | Partial<ExecutionGraphNodeData>
    | Partial<RunGraphNodeData> = {},
): GraphNode {
  switch (entityType) {
    case "backlog":
      return makeBacklogNode(id, overrides as Partial<BacklogGraphNodeData>);
    case "initiative":
      return makeInitiativeNode(id, overrides as Partial<InitiativeGraphNodeData>);
    case "capture":
      return makeCaptureNode(id, overrides as Partial<CaptureGraphNodeData>);
    case "scenario":
      return makeScenarioNode(id, overrides as Partial<ScenarioGraphNodeData>);
    case "execution":
      return makeExecutionNode(id, overrides as Partial<ExecutionGraphNodeData>);
    case "agent-run":
      return makeRunNode(id, overrides as Partial<RunGraphNodeData>);
  }
}

export function makeClusterNodeData(
  overrides: Partial<ClusterGraphNodeData> = {},
): ClusterGraphNodeData {
  return {
    label: overrides.label ?? "Cluster",
    entityType: "initiative",
    rawType: "Cluster",
    collapsed: overrides.collapsed ?? true,
    rollup: overrides.rollup ?? null,
    isUnassigned: overrides.isUnassigned,
    pulsing: overrides.pulsing,
    ...overrides,
  };
}

export function makeGraphEdge(
  id: string,
  source: string,
  target: string,
  type?: string,
  data?: GraphEdge["data"],
): GraphEdge {
  return {
    id,
    source,
    target,
    type,
    data,
  };
}
