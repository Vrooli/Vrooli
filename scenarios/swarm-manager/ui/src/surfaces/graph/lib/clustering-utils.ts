/**
 * Clustering Utilities
 *
 * Pure functions for initiative-based visual clustering:
 * - Build cluster hierarchy from nodes, edges, and initiative data
 * - Aggregate edges for collapsed clusters
 * - Apply node cap for visual complexity management
 */

import {
  getGraphNodeData,
  type GraphEdge,
  type GraphNode,
} from "../types";

export interface ClusterGroup {
  /** The initiative node ID (e.g., "initiative/my-init") or "__unassigned__" */
  id: string;
  label: string;
  /** Member node IDs */
  members: string[];
  /** Rollup from initiative data (null for unassigned group) */
  rollup: RollupCounts | null;
}

export interface RollupCounts {
  total: number;
  completed: number;
  in_progress: number;
  failed: number;
  pending: number;
}

export const UNASSIGNED_CLUSTER_ID = "__unassigned__";

/**
 * Build cluster hierarchy from flat nodes.
 * Groups backlog items by their initiative membership (via member_of edges).
 * Non-backlog nodes (scenarios, captures) remain unclustered.
 */
export function buildClusterHierarchy(
  nodes: GraphNode[],
  edges: GraphEdge[],
): { clusters: ClusterGroup[]; unclustered: GraphNode[] } {
  // Find member_of edges: source (backlog item) -> target (initiative)
  const memberOf = new Map<string, string>(); // nodeId -> initiativeNodeId
  for (const edge of edges) {
    if (edge.type === "member_of") {
      memberOf.set(edge.source, edge.target);
    }
  }

  // Build initiative info from initiative nodes
  const initiativeNodes = new Map<string, GraphNode>();
  for (const node of nodes) {
    const entityType = getGraphNodeData(node).entityType;
    if (entityType === "initiative") {
      initiativeNodes.set(node.id, node);
    }
  }

  // Group backlog items by initiative
  const clusterMembers = new Map<string, string[]>();
  const unclustered: GraphNode[] = [];

  for (const node of nodes) {
    const entityType = getGraphNodeData(node).entityType;
    if (entityType === "initiative") continue; // Initiative nodes become clusters, not members

    const initId = memberOf.get(node.id);
    if (initId && initiativeNodes.has(initId)) {
      const members = clusterMembers.get(initId) ?? [];
      members.push(node.id);
      clusterMembers.set(initId, members);
    } else if (entityType === "backlog") {
      // Backlog items without an initiative go to unassigned
      const members = clusterMembers.get(UNASSIGNED_CLUSTER_ID) ?? [];
      members.push(node.id);
      clusterMembers.set(UNASSIGNED_CLUSTER_ID, members);
    } else {
      unclustered.push(node);
    }
  }

  const clusters: ClusterGroup[] = [];

  // Build initiative clusters for every initiative node, including those with
  // no active backlog members. Without this, initiatives whose items are all
  // completed/archived disappear from the topology entirely.
  for (const [initId, initNode] of initiativeNodes) {
    const members = clusterMembers.get(initId) ?? [];
    const data = getGraphNodeData(initNode);
    const rollupData = data.rawType === "Initiative" ? data.rollup : undefined;
    const label =
      data.label ??
      ("title" in data && typeof data.title === "string" ? data.title : initId);

    clusters.push({
      id: initId,
      label,
      members,
      rollup: rollupData
        ? {
            total: rollupData.total ?? 0,
            completed: rollupData.completed ?? 0,
            in_progress: rollupData.in_progress ?? 0,
            failed: rollupData.failed ?? 0,
            pending: rollupData.pending ?? 0,
          }
        : null,
    });
  }

  // Build unassigned group
  const unassignedMembers = clusterMembers.get(UNASSIGNED_CLUSTER_ID);
  if (unassignedMembers && unassignedMembers.length > 0) {
    clusters.push({
      id: UNASSIGNED_CLUSTER_ID,
      label: "Unassigned",
      members: unassignedMembers,
      rollup: null,
    });
  }

  return { clusters, unclustered };
}

/**
 * Aggregate edges for collapsed clusters.
 * For each collapsed cluster, merges all edges to/from member nodes into
 * a single aggregated edge per (cluster, external-node, edge-type) triple.
 */
export function aggregateEdgesForCollapsed(
  edges: GraphEdge[],
  collapsedClusterIds: Set<string>,
  clusters: ClusterGroup[],
): GraphEdge[] {
  if (collapsedClusterIds.size === 0) return edges;

  // Build member -> cluster mapping
  const memberToCluster = new Map<string, string>();
  for (const cluster of clusters) {
    if (!collapsedClusterIds.has(cluster.id)) continue;
    for (const memberId of cluster.members) {
      memberToCluster.set(memberId, cluster.id);
    }
  }

  const aggregated = new Map<string, { edge: GraphEdge; count: number }>();
  const passThrough: GraphEdge[] = [];

  for (const edge of edges) {
    const sourceCluster = memberToCluster.get(edge.source);
    const targetCluster = memberToCluster.get(edge.target);

    // Skip intra-cluster edges when cluster is collapsed
    if (sourceCluster && targetCluster && sourceCluster === targetCluster) continue;
    // Also skip member_of edges to collapsed clusters (redundant with cluster display)
    if (edge.type === "member_of" && targetCluster) continue;
    if (edge.type === "member_of" && collapsedClusterIds.has(edge.target)) continue;

    const effectiveSource = sourceCluster ?? edge.source;
    const effectiveTarget = targetCluster ?? edge.target;

    // If neither end is in a collapsed cluster, pass through unchanged
    if (!sourceCluster && !targetCluster) {
      passThrough.push(edge);
      continue;
    }

    const key = `${effectiveSource}|${effectiveTarget}|${edge.type ?? "default"}`;
    const existing = aggregated.get(key);
    if (existing) {
      existing.count++;
    } else {
      aggregated.set(key, {
        edge: {
          id: `agg:${key}`,
          source: effectiveSource,
          target: effectiveTarget,
          type: edge.type,
          data: { aggregatedCount: 1 },
        },
        count: 1,
      });
    }
  }

  // Update counts in aggregated edges
  const result = [...passThrough];
  for (const { edge, count } of aggregated.values()) {
    result.push({
      ...edge,
      data: { ...(edge.data ?? {}), aggregatedCount: count },
    });
  }

  return result;
}

/**
 * Apply node cap to unclustered visible nodes.
 * Sorts by priority descending and caps at the given limit.
 * Returns a "More items" pseudo-node if items were capped.
 */
export function applyNodeCap(
  nodes: GraphNode[],
  limit: number,
): { visible: GraphNode[]; cappedCount: number } {
  if (nodes.length <= limit) {
    return { visible: nodes, cappedCount: 0 };
  }

  // Sort by priority descending (higher priority first)
  const sorted = [...nodes].sort((a, b) => {
    const pA = getGraphNodeData(a).priority ?? 0;
    const pB = getGraphNodeData(b).priority ?? 0;
    return pB - pA;
  });

  const visible = sorted.slice(0, limit);
  const cappedCount = sorted.length - limit;

  // Add "More items" pseudo-node
  visible.push({
    id: "__more-items__",
    type: "backlog",
    position: { x: 0, y: 0 },
    data: {
      label: `More items (${cappedCount})`,
      entityType: "backlog",
      rawType: "Synthetic",
      status: "capped",
      isCapNode: true,
    },
  });

  return { visible, cappedCount };
}
