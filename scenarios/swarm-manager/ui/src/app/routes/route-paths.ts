import type { BacklogKind } from "../../types";
import { parseNodeId } from "../../surfaces/graph/lib/node-id-parser";

export const GRAPH_SURFACES = ["plan", "graph", "stats"] as const;
export const GRAPH_MODES = ["topology", "focus"] as const;

export type AppGraphSurface = (typeof GRAPH_SURFACES)[number];
export type GraphMode = (typeof GRAPH_MODES)[number];
export type AppGraphLens = AppGraphSurface | "focus";
export type DetailEntityType = "backlog" | "scenario" | "execution" | "goal" | "capture" | "session";

export interface DetailRouteTarget {
  entityType: DetailEntityType;
  kind?: string;
  name?: string;
  identifier?: string;
  tab?: string;
}

type QueryPrimitive = string | number | boolean | null | undefined;
type QueryParams = Record<string, QueryPrimitive>;

const enc = encodeURIComponent;

function appendQuery(path: string, query?: QueryParams): string {
  if (!query) return path;
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (value === null || value === undefined || value === "") continue;
    params.set(key, String(value));
  }
  const serialized = params.toString();
  return serialized ? `${path}?${serialized}` : path;
}

export function isGraphLens(value: string | undefined): value is AppGraphLens {
  return GRAPH_SURFACES.includes(value as AppGraphSurface);
}

export function isGraphMode(value: string | undefined): value is GraphMode {
  return GRAPH_MODES.includes(value as GraphMode);
}

/**
 * Canonical path for the operator surfaces. Plan is the first-class board
 * route (`/plan`); Graph is the single graph route (`/graph`). Focus is graph
 * query state (`mode=focus`), while topology remains the internal API
 * projection that backs the default graph view.
 */
export function graphPath(options: {
  lens?: AppGraphLens;
  mode?: GraphMode | null;
  focus?: string | null;
  returnLens?: string | null;
  select?: string | null;
  goal?: string | null;
} = {}): string {
  const path = options.lens === "plan" ? "/plan" : options.lens === "stats" ? "/stats" : "/graph";
  const mode = options.mode ?? (options.lens === "focus" ? "focus" : null);
  return appendQuery(path, {
    mode: mode === "focus" ? mode : null,
    focus: options.focus,
    returnLens: options.returnLens,
    select: options.select,
    goal: options.goal,
  });
}

export function backlogDetailPath(kind: string, name: string, query?: QueryParams): string {
  return appendQuery(`/backlog/${enc(kind)}/${enc(name)}`, query);
}

export function scenarioDetailPath(name: string, query?: QueryParams): string {
  return appendQuery(`/scenarios/${enc(name)}`, query);
}

export function executionDetailPath(executionId: string, query?: QueryParams): string {
  return appendQuery(`/executions/${enc(executionId)}`, query);
}

export function recordDetailPath(id: string, query?: QueryParams): string {
	return appendQuery(`/records/${enc(id)}`, query);
}

export function goalDetailPath(name: string, query?: QueryParams): string {
  return appendQuery(`/goals/${enc(name)}`, query);
}

export function captureDetailPath(captureId: string, query?: QueryParams): string {
  return appendQuery(`/captures/${enc(captureId)}`, query);
}

export function sessionDetailPath(sessionId: string, query?: QueryParams): string {
  return appendQuery(`/sessions/${enc(sessionId)}`, query);
}

export function detailPath(target: DetailRouteTarget): string | null {
  switch (target.entityType) {
    case "backlog":
      return target.kind && target.name
        ? backlogDetailPath(target.kind, target.name, target.tab ? { tab: target.tab } : undefined)
        : null;
    case "scenario":
      return target.name ? scenarioDetailPath(target.name, target.tab ? { tab: target.tab } : undefined) : null;
    case "execution":
      return target.identifier ? executionDetailPath(target.identifier, target.tab ? { tab: target.tab } : undefined) : null;
    case "goal":
      return target.name ? goalDetailPath(target.name, target.tab ? { tab: target.tab } : undefined) : null;
    case "capture":
      return target.identifier ? captureDetailPath(target.identifier, target.tab ? { tab: target.tab } : undefined) : null;
    case "session":
      return target.identifier ? sessionDetailPath(target.identifier, target.tab ? { tab: target.tab } : undefined) : null;
  }
}

export function detailPathFromNodeId(nodeId: string): string | null {
  if (nodeId.startsWith("goal/")) {
    const name = nodeId.slice("goal/".length);
    return name ? goalDetailPath(name) : null;
  }
  const parsed = parseNodeId(nodeId);
  if (!parsed) return null;
  switch (parsed.entityType) {
    case "backlog":
      return parsed.kind && parsed.name ? backlogDetailPath(parsed.kind, parsed.name) : null;
    case "scenario":
      return parsed.name ? scenarioDetailPath(parsed.name) : null;
    case "execution":
      return executionDetailPath(parsed.identifier);
    case "capture":
      return captureDetailPath(parsed.identifier);
    default:
      return null;
  }
}

export function routeTargetToNodeId(target: DetailRouteTarget): string | null {
  switch (target.entityType) {
    case "backlog":
      return target.kind && target.name ? `backlog-item/${target.kind}/${target.name}` : null;
    case "scenario":
      return target.name ? `scenario/${target.name}` : null;
    case "execution":
      return target.identifier ? `execution-record/${target.identifier}` : null;
    case "goal":
      return target.name ? `goal/${target.name}` : null;
    case "capture":
      return target.identifier ? `capture/${target.identifier}` : null;
    case "session":
      return null;
  }
}

export function routeBacklogKind(kind: string | undefined): BacklogKind | null {
  if (!kind) return null;
  return kind as BacklogKind;
}
