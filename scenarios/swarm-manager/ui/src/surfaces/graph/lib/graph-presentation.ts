/**
 * Graph Presentation
 *
 * Pure graph-derivation helpers for the canvas. This keeps the filtering and
 * grouping rules out of the React component so the default presentation and
 * topology-compression behavior can be tested directly.
 */

import type { GraphLensSettings } from "../stores/graph-data-store";
import {
  getGraphNodeData,
  type GraphEdge,
  type GraphNode,
  type GraphLens,
} from "../types";
import {
  aggregateEdgesForCollapsed,
  applyNodeCap,
  buildClusterHierarchy,
  UNASSIGNED_CLUSTER_ID,
} from "./clustering-utils";
import { SECONDARY_EDGE_TYPES } from "./edge-styles";

export interface GraphPresentationResult {
  processedNodes: GraphNode[];
  processedEdges: GraphEdge[];
  visibleEdgeTypes: string[];
  visibleNodeCount: number;
}

export interface BuildGraphPresentationInput {
  lens: GraphLens;
  nodes: GraphNode[];
  edges: GraphEdge[];
  settings: GraphLensSettings;
  expandedTopologyClusters: Set<string>;
  nodeCapLimit?: number;
}

const DEFAULT_NODE_CAP_LIMIT = 50;

export function filterGraphNodes(nodes: GraphNode[], settings: GraphLensSettings): GraphNode[] {
  return nodes.filter((node) => {
    const data = getGraphNodeData(node);
    const entityType = data.entityType;
    if (entityType && settings.entityFilters[entityType] === false) {
      return false;
    }

    const status = data.status;
    if (typeof status === "string" && settings.statusFilters[status] === false) {
      return false;
    }

    return true;
  });
}

export function filterGraphEdges(
  edges: GraphEdge[],
  visibleNodes: GraphNode[],
  settings: GraphLensSettings,
): GraphEdge[] {
  const visibleIds = new Set(visibleNodes.map((node) => node.id));
  return edges.filter((edge) => {
    if (!visibleIds.has(edge.source) || !visibleIds.has(edge.target)) {
      return false;
    }
    if (!settings.showSecondaryEdges && SECONDARY_EDGE_TYPES.has(edge.type ?? "")) {
      return false;
    }
    return true;
  });
}

function buildVisibleEdgeTypes(edges: GraphEdge[]): string[] {
  return [...new Set(edges.map((edge) => edge.type).filter(Boolean) as string[])];
}

function buildFlatPresentation(
  filteredNodes: GraphNode[],
  filteredEdges: GraphEdge[],
): GraphPresentationResult {
  return {
    processedNodes: filteredNodes,
    processedEdges: filteredEdges,
    visibleEdgeTypes: buildVisibleEdgeTypes(filteredEdges),
    visibleNodeCount: filteredNodes.length,
  };
}

function buildInitiativeTopologyPresentation(
  filteredNodes: GraphNode[],
  filteredEdges: GraphEdge[],
  expandedTopologyClusters: Set<string>,
  nodeCapLimit: number,
): GraphPresentationResult {
  const { clusters, unclustered } = buildClusterHierarchy(filteredNodes, filteredEdges);
  const collapsedClusterIds = new Set(
    clusters
      .filter((cluster) => !expandedTopologyClusters.has(cluster.id))
      .map((cluster) => cluster.id),
  );

  const clusterNodes: GraphNode[] = [];
  const childNodes: GraphNode[] = [];
  const nodeById = new Map(filteredNodes.map((node) => [node.id, node]));

  for (const cluster of clusters) {
    const isCollapsed = collapsedClusterIds.has(cluster.id);
    const isUnassigned = cluster.id === UNASSIGNED_CLUSTER_ID;

    clusterNodes.push({
      id: cluster.id,
      type: "cluster",
      position: { x: 0, y: 0 },
      data: {
        label: cluster.label,
        collapsed: isCollapsed,
        rollup: cluster.rollup,
        isUnassigned,
        entityType: "initiative",
        rawType: "Cluster",
      },
      style: isCollapsed ? undefined : { padding: 20 },
    });

    if (!isCollapsed) {
      for (const memberId of cluster.members) {
        const memberNode = nodeById.get(memberId);
        if (!memberNode) {
          continue;
        }

        childNodes.push({
          ...memberNode,
          parentId: cluster.id,
          extent: "parent" as const,
          position: { x: 0, y: 40 },
        });
      }
    }
  }

  const { visible: cappedUnclustered } = applyNodeCap(unclustered, nodeCapLimit);
  const groupedEdges = aggregateEdgesForCollapsed(filteredEdges, collapsedClusterIds, clusters);

  return {
    processedNodes: [...clusterNodes, ...childNodes, ...cappedUnclustered],
    processedEdges: groupedEdges,
    visibleEdgeTypes: buildVisibleEdgeTypes(groupedEdges),
    visibleNodeCount: clusterNodes.length + childNodes.length + cappedUnclustered.length,
  };
}

export function buildGraphPresentation({
  lens,
  nodes,
  edges,
  settings,
  expandedTopologyClusters,
  nodeCapLimit = DEFAULT_NODE_CAP_LIMIT,
}: BuildGraphPresentationInput): GraphPresentationResult {
  const filteredNodes = filterGraphNodes(nodes, settings);
  const filteredEdges = filterGraphEdges(edges, filteredNodes, settings);

  const useInitiativeGrouping = lens === "topology" && settings.groupingMode === "initiative";
  if (!useInitiativeGrouping) {
    return buildFlatPresentation(filteredNodes, filteredEdges);
  }

  return buildInitiativeTopologyPresentation(
    filteredNodes,
    filteredEdges,
    expandedTopologyClusters,
    nodeCapLimit,
  );
}
