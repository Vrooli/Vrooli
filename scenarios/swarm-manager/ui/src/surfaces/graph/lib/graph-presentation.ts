/**
 * Graph Presentation
 *
 * Pure graph-derivation helpers for the canvas. This keeps the filtering and
 * grouping rules out of the React component so the default presentation and
 * topology-compression behavior can be tested directly.
 */

import type { Edge, Node } from "@xyflow/react";
import type { GraphLens, GraphLensSettings } from "../stores/graph-data-store";
import {
  aggregateEdgesForCollapsed,
  applyNodeCap,
  buildClusterHierarchy,
  UNASSIGNED_CLUSTER_ID,
} from "./clustering-utils";
import { SECONDARY_EDGE_TYPES } from "./edge-styles";

export interface GraphPresentationResult {
  processedNodes: Node[];
  processedEdges: Edge[];
  visibleEdgeTypes: string[];
  visibleNodeCount: number;
}

export interface BuildGraphPresentationInput {
  lens: GraphLens;
  nodes: Node[];
  edges: Edge[];
  settings: GraphLensSettings;
  expandedTopologyClusters: Set<string>;
  nodeCapLimit?: number;
}

const DEFAULT_NODE_CAP_LIMIT = 50;

export function filterGraphNodes(nodes: Node[], settings: GraphLensSettings): Node[] {
  return nodes.filter((node) => {
    const data = (node.data as Record<string, unknown> | undefined) ?? {};
    const entityType = data.entityType as keyof typeof settings.entityFilters | undefined;
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
  edges: Edge[],
  visibleNodes: Node[],
  settings: GraphLensSettings,
): Edge[] {
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

function buildVisibleEdgeTypes(edges: Edge[]): string[] {
  return [...new Set(edges.map((edge) => edge.type).filter(Boolean) as string[])];
}

function buildFlatPresentation(
  filteredNodes: Node[],
  filteredEdges: Edge[],
): GraphPresentationResult {
  return {
    processedNodes: filteredNodes,
    processedEdges: filteredEdges,
    visibleEdgeTypes: buildVisibleEdgeTypes(filteredEdges),
    visibleNodeCount: filteredNodes.length,
  };
}

function buildInitiativeTopologyPresentation(
  filteredNodes: Node[],
  filteredEdges: Edge[],
  expandedTopologyClusters: Set<string>,
  nodeCapLimit: number,
): GraphPresentationResult {
  const { clusters, unclustered } = buildClusterHierarchy(filteredNodes, filteredEdges);
  const collapsedClusterIds = new Set(
    clusters
      .filter((cluster) => !expandedTopologyClusters.has(cluster.id))
      .map((cluster) => cluster.id),
  );

  const clusterNodes: Node[] = [];
  const childNodes: Node[] = [];
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
