/**
 * Structural sharing utilities.
 *
 * Backend polls return fresh object references on every request even when the
 * underlying data is unchanged. That breaks downstream `useMemo` chains in the
 * canvas because they key on identity, not content — a quiet poll still busts
 * filter/style/layout caches and re-renders every node.
 *
 * These helpers run a cheap content comparison against the previous snapshot
 * and reuse old refs for entries that didn't actually change. Unchanged nodes
 * and edges keep their identity through the pipeline, so downstream memos
 * preserve their cached output.
 */

import {
  getGraphNodeData,
  type GraphEdge,
  type GraphNode,
  type GraphNodeData,
} from "../types";

function isDataEqual(a: unknown, b: unknown): boolean {
  if (Object.is(a, b)) return true;
  if (a === null || b === null) return false;
  if (typeof a !== "object" || typeof b !== "object") return false;

  if (Array.isArray(a) || Array.isArray(b)) {
    if (!Array.isArray(a) || !Array.isArray(b)) return false;
    if (a.length !== b.length) return false;
    for (let i = 0; i < a.length; i += 1) {
      if (!isDataEqual(a[i], b[i])) return false;
    }
    return true;
  }

  const aObj = a as Record<string, unknown>;
  const bObj = b as Record<string, unknown>;
  const aKeys = Object.keys(aObj);
  const bKeys = Object.keys(bObj);
  if (aKeys.length !== bKeys.length) return false;
  for (const key of aKeys) {
    if (!Object.prototype.hasOwnProperty.call(bObj, key)) return false;
    if (!isDataEqual(aObj[key], bObj[key])) return false;
  }
  return true;
}

export function isGraphNodeContentEqual(a: GraphNode, b: GraphNode): boolean {
  if (a.id !== b.id) return false;
  if (a.type !== b.type) return false;
  if (a.parentId !== b.parentId) return false;
  return isDataEqual(a.data, b.data);
}

export function isGraphEdgeContentEqual(a: GraphEdge, b: GraphEdge): boolean {
  if (a.id !== b.id) return false;
  if (a.source !== b.source) return false;
  if (a.target !== b.target) return false;
  if (a.type !== b.type) return false;
  if (a.sourceHandle !== b.sourceHandle) return false;
  if (a.targetHandle !== b.targetHandle) return false;
  return isDataEqual(a.data, b.data);
}

/**
 * Merge runtime node state (pulsing) from the current snapshot into the
 * incoming nodes, then reuse existing node refs whenever the resulting content
 * is equal to the previous snapshot. The returned array is always fresh, but
 * individual entries keep their previous identity when unchanged.
 */
export function reconcileNodes(currentNodes: GraphNode[], nextNodes: GraphNode[]): GraphNode[] {
  const currentById = new Map<string, GraphNode>();
  for (const node of currentNodes) {
    currentById.set(node.id, node);
  }

  return nextNodes.map((next) => {
    const prev = currentById.get(next.id);
    const prevPulsing = prev ? getGraphNodeData(prev).pulsing : undefined;

    let candidate: GraphNode;
    if (typeof prevPulsing === "boolean") {
      const nextData = getGraphNodeData(next);
      if (nextData.pulsing === prevPulsing) {
        candidate = next;
      } else {
        candidate = {
          ...next,
          data: {
            ...nextData,
            pulsing: prevPulsing,
          } as GraphNodeData,
        };
      }
    } else {
      candidate = next;
    }

    if (prev && isGraphNodeContentEqual(prev, candidate)) {
      return prev;
    }
    return candidate;
  });
}

/**
 * Reuse existing edge refs whenever their content matches the incoming edge.
 */
export function reconcileEdges(currentEdges: GraphEdge[], nextEdges: GraphEdge[]): GraphEdge[] {
  const currentById = new Map<string, GraphEdge>();
  for (const edge of currentEdges) {
    currentById.set(edge.id, edge);
  }

  return nextEdges.map((next) => {
    const prev = currentById.get(next.id);
    if (prev && isGraphEdgeContentEqual(prev, next)) {
      return prev;
    }
    return next;
  });
}
