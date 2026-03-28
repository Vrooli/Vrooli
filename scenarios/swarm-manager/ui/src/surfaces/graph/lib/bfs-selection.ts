/**
 * BFS Neighborhood Selection
 *
 * Type-constrained BFS traversal for selecting a node and its neighbors.
 * Used when clicking a node to highlight its neighborhood.
 */

import type { Node, Edge } from "@xyflow/react";

interface BFSOptions {
  /** Maximum traversal depth. Default: 1 */
  maxDepth?: number;
  /** If set, only traverse through nodes of these entity types. */
  allowedTypes?: Set<string>;
}

function getOrCreateNeighbors(adjacency: Map<string, Set<string>>, nodeId: string): Set<string> {
  const existing = adjacency.get(nodeId);
  if (existing) {
    return existing;
  }

  const created = new Set<string>();
  adjacency.set(nodeId, created);
  return created;
}

/**
 * Perform BFS from a start node, returning IDs of all reachable nodes
 * within the depth limit and type constraints.
 */
export function bfsNeighborhood(
  startId: string,
  nodes: Node[],
  edges: Edge[],
  options: BFSOptions = {},
): Set<string> {
  const { maxDepth = 1, allowedTypes } = options;

  const nodeMap = new Map<string, Node>();
  for (const node of nodes) {
    nodeMap.set(node.id, node);
  }

  if (!nodeMap.has(startId)) {
    return new Set();
  }

  // Build adjacency list (undirected).
  const adjacency = new Map<string, Set<string>>();
  for (const edge of edges) {
    const sourceNeighbors = getOrCreateNeighbors(adjacency, edge.source);
    const targetNeighbors = getOrCreateNeighbors(adjacency, edge.target);
    sourceNeighbors.add(edge.target);
    targetNeighbors.add(edge.source);
  }

  const visited = new Set<string>([startId]);
  const queue: Array<{ id: string; depth: number }> = [{ id: startId, depth: 0 }];

  while (queue.length > 0) {
    const current = queue.shift();
    if (!current) {
      continue;
    }

    const { id, depth } = current;
    if (depth >= maxDepth) continue;

    const neighbors = adjacency.get(id);
    if (!neighbors) continue;

    for (const neighborId of neighbors) {
      if (visited.has(neighborId)) continue;

      const neighborNode = nodeMap.get(neighborId);
      if (!neighborNode) continue;

      // Type constraint: skip nodes not matching allowed types.
      if (allowedTypes && neighborNode.type && !allowedTypes.has(neighborNode.type)) {
        continue;
      }

      visited.add(neighborId);
      queue.push({ id: neighborId, depth: depth + 1 });
    }
  }

  return visited;
}
