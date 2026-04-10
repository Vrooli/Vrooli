/**
 * Pure BFS helper for computing a node's neighborhood in the dependency graph.
 *
 * Extracted from GraphView for independent testability — this is a testing seam
 * that isolates graph traversal logic from React rendering concerns.
 *
 * Traversal rule: each hop must move to a DIFFERENT entity type than any type
 * already visited on that path. With 4 types (team, agent, skill, cli), this
 * naturally caps at 3 hops and prevents lateral spread within the same type.
 */

import type { GraphEdge, GraphNode, NodeType } from '@/lib/schemas'

/** Bitmask per entity type — used to efficiently track visited types per path. */
const TYPE_BIT: Record<NodeType, number> = {
  team: 1,
  agent: 2,
  skill: 4,
  cli: 8,
}

/**
 * BFS from a starting node, following edges bidirectionally. Each hop must
 * reach a node whose type has NOT been visited on the current path.
 *
 * Returns the set of all reachable node IDs (including the start).
 *
 * @param nodeId - The starting node ID
 * @param adjacentEdgesMap - Map from node ID to all edges touching that node
 * @param nodeMap - Map from node ID to node (provides type info + validates existence)
 */
export function collectNeighborhood(
  nodeId: string,
  adjacentEdgesMap: Map<string, GraphEdge[]>,
  nodeMap: Map<string, GraphNode>,
): Set<string> {
  const result = new Set<string>()
  const startNode = nodeMap.get(nodeId)
  if (!startNode) return result

  const startBit = TYPE_BIT[startNode.type]
  result.add(nodeId)

  // BFS queue carries the bitmask of entity types visited on the path so far.
  const queue: Array<{ id: string; typeMask: number }> = [
    { id: nodeId, typeMask: startBit },
  ]
  // Track (nodeId, typeMask) pairs to avoid re-processing the same state.
  const visited = new Set<string>()
  visited.add(`${nodeId}:${startBit}`)

  while (queue.length > 0) {
    const current = queue.shift()
    if (!current) continue
    const edges = adjacentEdgesMap.get(current.id) ?? []

    for (const edge of edges) {
      const neighborId = edge.from === current.id ? edge.to : edge.from
      const neighbor = nodeMap.get(neighborId)
      if (!neighbor) continue

      const neighborBit = TYPE_BIT[neighbor.type]
      // Skip if this entity type was already visited on this path
      if (current.typeMask & neighborBit) continue

      const newMask = current.typeMask | neighborBit
      const stateKey = `${neighborId}:${newMask}`
      if (visited.has(stateKey)) continue

      visited.add(stateKey)
      result.add(neighborId)
      queue.push({ id: neighborId, typeMask: newMask })
    }
  }

  return result
}
