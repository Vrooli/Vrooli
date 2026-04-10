/**
 * Snapshot creation, cloning, and comparison helpers for the graph data store.
 *
 * Extracted from graph-data-store.ts.
 */

import type { GraphProjectionMeta } from "../../../services";
import type { GraphEdge, GraphLens, GraphNode } from "../types";

export interface GraphLensSnapshot {
  nodes: GraphNode[];
  edges: GraphEdge[];
  meta: GraphProjectionMeta | null;
  loading: boolean;
  error: string | null;
  fetchedAtMs: number | null;
}

export function createEmptyLensSnapshot(): GraphLensSnapshot {
  return {
    nodes: [],
    edges: [],
    meta: null,
    loading: false,
    error: null,
    fetchedAtMs: null,
  };
}

export function createEmptyGraphsByLens(): Record<GraphLens, GraphLensSnapshot> {
  return {
    focus: createEmptyLensSnapshot(),
    topology: createEmptyLensSnapshot(),
    operations: createEmptyLensSnapshot(),
  };
}

export function cloneLensSnapshot(snapshot: GraphLensSnapshot): GraphLensSnapshot {
  return {
    nodes: [...snapshot.nodes],
    edges: [...snapshot.edges],
    meta: snapshot.meta ? { ...snapshot.meta } : null,
    loading: snapshot.loading,
    error: snapshot.error,
    fetchedAtMs: snapshot.fetchedAtMs,
  };
}

export function cloneGraphsByLens(
  graphsByLens: Record<GraphLens, GraphLensSnapshot>,
): Record<GraphLens, GraphLensSnapshot> {
  return {
    focus: cloneLensSnapshot(graphsByLens.focus),
    topology: cloneLensSnapshot(graphsByLens.topology),
    operations: cloneLensSnapshot(graphsByLens.operations),
  };
}
