import type { BacklogKind } from "../../types";
import type { GraphLens } from "../../surfaces/graph/stores/graph-data-store";
import { parseNodeId } from "../../surfaces/graph/lib/node-id-parser";

export const GRAPH_LENSES = ["focus", "topology", "operations"] as const satisfies readonly GraphLens[];

export type AppGraphLens = (typeof GRAPH_LENSES)[number];
export type DetailEntityType = "backlog" | "scenario" | "execution" | "initiative" | "capture" | "session" | "operatingMode";

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
  return GRAPH_LENSES.includes(value as AppGraphLens);
}

export function graphPath(options: {
  lens?: AppGraphLens;
  focus?: string | null;
  returnLens?: string | null;
  select?: string | null;
} = {}): string {
  const path = options.lens ? `/graph/${options.lens}` : "/graph";
  return appendQuery(path, {
    focus: options.focus,
    returnLens: options.returnLens,
    select: options.select,
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

export function initiativeDetailPath(name: string, query?: QueryParams): string {
  return appendQuery(`/initiatives/${enc(name)}`, query);
}

export function captureDetailPath(captureId: string, query?: QueryParams): string {
  return appendQuery(`/captures/${enc(captureId)}`, query);
}

export function sessionDetailPath(sessionId: string, query?: QueryParams): string {
  return appendQuery(`/sessions/${enc(sessionId)}`, query);
}

export function operatingModeDetailPath(mode: string, query?: QueryParams): string {
  return appendQuery(`/operating-modes/${enc(mode)}`, query);
}

export function commandPostPath(query?: QueryParams): string {
  return appendQuery("/command-post", query);
}

export function decisionStreamPath(query?: QueryParams): string {
  return appendQuery("/command-post/decisions", query);
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
    case "initiative":
      return target.name ? initiativeDetailPath(target.name, target.tab ? { tab: target.tab } : undefined) : null;
    case "capture":
      return target.identifier ? captureDetailPath(target.identifier, target.tab ? { tab: target.tab } : undefined) : null;
    case "session":
      return target.identifier ? sessionDetailPath(target.identifier, target.tab ? { tab: target.tab } : undefined) : null;
    case "operatingMode":
      return target.identifier ? operatingModeDetailPath(target.identifier, target.tab ? { tab: target.tab } : undefined) : null;
  }
}

export function detailPathFromNodeId(nodeId: string): string | null {
  // Operating-mode node IDs aren't part of the graph entity registry — handle
  // them before parseNodeId, which only knows about graph entities.
  if (nodeId.startsWith("operatingMode/")) {
    const mode = nodeId.slice("operatingMode/".length);
    return mode ? operatingModeDetailPath(mode) : null;
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
    case "initiative":
      return parsed.name ? initiativeDetailPath(parsed.name) : null;
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
    case "initiative":
      return target.name ? `initiative/${target.name}` : null;
    case "capture":
      return target.identifier ? `capture/${target.identifier}` : null;
    case "session":
      return null;
    case "operatingMode":
      return target.identifier ? `operatingMode/${target.identifier}` : null;
  }
}

export function routeBacklogKind(kind: string | undefined): BacklogKind | null {
  if (!kind) return null;
  return kind as BacklogKind;
}
