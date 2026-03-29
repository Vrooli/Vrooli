/**
 * Layout Utilities
 *
 * Dagre layout configuration for three layout modes.
 * Transforms React Flow nodes/edges into positioned layouts.
 * Disconnected nodes (no edges) are arranged in a grid below the
 * connected subgraph to prevent Dagre's default single-line placement.
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
 *
 * Nodes with no edges are separated and arranged in a compact grid below
 * the connected subgraph, preventing the single-line layout that Dagre
 * produces for isolated nodes.
 */
export function applyDagreLayout<NodeType extends Node, EdgeType extends Edge>(
  nodes: NodeType[],
  edges: EdgeType[],
  mode: LayoutMode,
  direction: LayoutDirection,
): NodeType[] {
  if (nodes.length === 0) return [];

  // Partition nodes into connected (has ≥1 edge) and isolated (no edges).
  const connectedIds = new Set<string>();
  for (const edge of edges) {
    connectedIds.add(edge.source);
    connectedIds.add(edge.target);
  }

  const connectedNodes: NodeType[] = [];
  const isolatedNodes: NodeType[] = [];
  for (const node of nodes) {
    if (connectedIds.has(node.id)) {
      connectedNodes.push(node);
    } else {
      isolatedNodes.push(node);
    }
  }

  // If all nodes are isolated (no edges), arrange them all in a grid.
  if (connectedNodes.length === 0) {
    return arrangeGrid(isolatedNodes, mode, 0, 0);
  }

  // Layout the connected subgraph with Dagre.
  const config = getDagreConfig(mode, direction);
  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({
    rankdir: direction,
    ranksep: config.ranksep,
    nodesep: config.nodesep,
    ranker: config.ranker,
  });

  for (const node of connectedNodes) {
    const width = (node.measured?.width ?? node.width ?? DEFAULT_NODE_WIDTH) as number;
    const height = (node.measured?.height ?? node.height ?? DEFAULT_NODE_HEIGHT) as number;
    g.setNode(node.id, { width, height });
  }

  for (const edge of edges) {
    if (connectedIds.has(edge.source) && connectedIds.has(edge.target)) {
      g.setEdge(edge.source, edge.target);
    }
  }

  dagre.layout(g);

  const positioned: NodeType[] = connectedNodes.map((node) => {
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

  // Place isolated nodes in a grid below the connected subgraph.
  if (isolatedNodes.length > 0) {
    let maxY = -Infinity;
    let minX = Infinity;
    for (const node of positioned) {
      const height = (node.measured?.height ?? node.height ?? DEFAULT_NODE_HEIGHT) as number;
      maxY = Math.max(maxY, node.position.y + height);
      minX = Math.min(minX, node.position.x);
    }

    const gridStartY = maxY + config.ranksep;
    positioned.push(...arrangeGrid(isolatedNodes, mode, minX, gridStartY));
  }

  return positioned;
}

/**
 * Arrange nodes in a compact grid starting at the given origin.
 */
function arrangeGrid<NodeType extends Node>(
  nodes: NodeType[],
  mode: LayoutMode,
  startX: number,
  startY: number,
): NodeType[] {
  if (nodes.length === 0) return [];

  const gap = mode === "compact" ? 20 : mode === "grouped" ? 50 : 30;
  const cols = Math.max(1, Math.ceil(Math.sqrt(nodes.length)));

  return nodes.map((node, i) => {
    const col = i % cols;
    const row = Math.floor(i / cols);
    const width = (node.measured?.width ?? node.width ?? DEFAULT_NODE_WIDTH) as number;
    const height = (node.measured?.height ?? node.height ?? DEFAULT_NODE_HEIGHT) as number;
    return {
      ...node,
      position: {
        x: startX + col * (width + gap),
        y: startY + row * (height + gap),
      },
    };
  });
}
