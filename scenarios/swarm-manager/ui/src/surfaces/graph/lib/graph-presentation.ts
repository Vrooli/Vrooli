/**
 * Graph Presentation
 *
 * Pure graph-derivation helpers for the canvas. This keeps the filtering and
 * grouping rules out of the React component so the default presentation and
 * topology-compression behavior can be tested directly.
 */

import type { GraphLensSettings } from "../stores/graph-settings-store";
import {
  getGraphNodeData,
  type GraphEdge,
  type GraphNode,
  type GraphLens,
} from "../types";
import { computeNodeAttention } from "./attention";
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
}

export function filterGraphNodes(nodes: GraphNode[], settings: GraphLensSettings): GraphNode[] {
  return nodes.filter((node) => {
    const data = getGraphNodeData(node);
    const entityType = data.entityType;
    if (entityType && !settings.entityFilters[entityType]) {
      return false;
    }

    const status = data.status;
    if (typeof status === "string" && entityType) {
      const entityGroup = settings.statusFilters[entityType];
      if (entityGroup && entityGroup[status] === false) {
        return false;
      }
    }

    return true;
  });
}

export function filterGraphEdges(
  edges: GraphEdge[],
  visibleNodes: GraphNode[],
  settings: GraphLensSettings,
  lens?: GraphLens,
): GraphEdge[] {
  const visibleIds = new Set(visibleNodes.map((node) => node.id));
  // Focus mode has already filtered to attention-worthy items plus their
  // structural context — any remaining edge between two visible nodes is
  // load-bearing, so skip the secondary-edge filter here. Without this,
  // context edges like "targets" (scenario connections) disappear and
  // scenarios show up floating with no visible link to the work.
  const applySecondaryFilter = lens !== "focus" && !settings.showSecondaryEdges;
  return edges.filter((edge) => {
    if (!visibleIds.has(edge.source) || !visibleIds.has(edge.target)) {
      return false;
    }
    if (applySecondaryFilter && SECONDARY_EDGE_TYPES.has(edge.type ?? "")) {
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

function applyAttentionHighlighting(nodes: GraphNode[]): GraphNode[] {
  return nodes.map((node) => {
    const data = getGraphNodeData(node);
    const result = computeNodeAttention(data);
    if (result.needsAttention) {
      return {
        ...node,
        data: {
          ...data,
          pulsing: true,
          pulseMode: "persistent" as const,
        },
      };
    }
    return node;
  });
}

export function buildGraphPresentation({
  lens,
  nodes,
  edges,
  settings,
}: BuildGraphPresentationInput): GraphPresentationResult {
  let filteredNodes = filterGraphNodes(nodes, settings);
  const filteredEdges = filterGraphEdges(edges, filteredNodes, settings, lens);

  if (settings.highlightActionableNodes) {
    filteredNodes = applyAttentionHighlighting(filteredNodes);
  }

  return buildFlatPresentation(filteredNodes, filteredEdges);
}
