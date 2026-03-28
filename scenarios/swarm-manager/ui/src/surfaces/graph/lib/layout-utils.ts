/**
 * Layout Utilities
 *
 * Dagre layout configuration for three layout modes.
 * Transforms React Flow nodes/edges into positioned layouts.
 */

import dagre from "dagre";
import type { Node, Edge } from "@xyflow/react";
import type { LayoutDirection, LayoutMode } from "../stores/graph-ui-store";

const DEFAULT_NODE_WIDTH = 180;
const DEFAULT_NODE_HEIGHT = 72;

interface DagreConfig {
  rankdir: LayoutDirection;
  ranksep: number;
  nodesep: number;
  ranker: string;
}

export function getDagreConfig(mode: LayoutMode, direction: LayoutDirection): DagreConfig {
  switch (mode) {
    case "hierarchical":
      return {
        rankdir: direction,
        ranksep: 80,
        nodesep: 40,
        ranker: "network-simplex",
      };
    case "compact":
      return {
        rankdir: direction,
        ranksep: 60,
        nodesep: 30,
        ranker: "tight-tree",
      };
    case "grouped":
      return {
        rankdir: direction,
        ranksep: 100,
        nodesep: 60,
        ranker: "network-simplex",
      };
  }
}

/**
 * Apply Dagre layout to nodes and edges, returning new positioned nodes.
 */
export function applyDagreLayout(
  nodes: Node[],
  edges: Edge[],
  mode: LayoutMode,
  direction: LayoutDirection,
): Node[] {
  if (nodes.length === 0) return [];

  const config = getDagreConfig(mode, direction);
  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({
    rankdir: direction,
    ranksep: config.ranksep,
    nodesep: config.nodesep,
    ranker: config.ranker,
  });

  for (const node of nodes) {
    const width = (node.measured?.width ?? node.width ?? DEFAULT_NODE_WIDTH) as number;
    const height = (node.measured?.height ?? node.height ?? DEFAULT_NODE_HEIGHT) as number;
    g.setNode(node.id, { width, height });
  }

  for (const edge of edges) {
    g.setEdge(edge.source, edge.target);
  }

  dagre.layout(g);

  return nodes.map((node) => {
    const pos = g.node(node.id);
    const width = (node.measured?.width ?? node.width ?? DEFAULT_NODE_WIDTH) as number;
    const height = (node.measured?.height ?? node.height ?? DEFAULT_NODE_HEIGHT) as number;
    return {
      ...node,
      position: {
        x: pos.x - width / 2,
        y: pos.y - height / 2,
      },
    };
  });
}
