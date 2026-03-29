/**
 * Layout Utilities
 *
 * Dagre layout configuration for hierarchical and compact modes.
 * Lane-based grouped layout that organizes nodes by entity type.
 * Disconnected nodes (no edges) are arranged in a grid below the
 * connected subgraph to prevent Dagre's default single-line placement.
 */

import dagre from "dagre";
import type { Node, Edge } from "@xyflow/react";
import type { LayoutDirection, LayoutMode } from "../stores/graph-ui-store";
import type { GraphEntityType } from "../types";
import { getShapeDimensions } from "./entity-shapes";

const DEFAULT_NODE_WIDTH = 140;
const DEFAULT_NODE_HEIGHT = 80;

interface DagreConfig {
  rankdir: LayoutDirection;
  ranksep: number;
  nodesep: number;
  ranker: string;
}

export function getDagreConfig(mode: Exclude<LayoutMode, "grouped">, direction: LayoutDirection): DagreConfig {
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
  }
}

/**
 * Lane order for grouped layout. Nodes are arranged in lanes by entity type,
 * matching prompt-manager's approach where each entity type occupies its own
 * row (TB) or column (LR).
 */
const GROUPED_LANE_ORDER: GraphEntityType[] = [
  "initiative",
  "backlog",
  "scenario",
  "execution",
  "capture",
  "agent-run",
  "agent-activity",
];

/**
 * Resolve the entity type from a node's data for lane grouping.
 * Falls back to the node's `type` field (which maps to nodeTypes in GraphCanvas)
 * when `data.entityType` is not present.
 */
function getNodeEntityType(node: Node): GraphEntityType | undefined {
  const data = node.data as { entityType?: GraphEntityType } | undefined;
  if (data?.entityType) return data.entityType;
  // Node `type` in React Flow maps to the nodeTypes registry keys (backlog, scenario, etc.)
  if (node.type && GROUPED_LANE_ORDER.includes(node.type as GraphEntityType)) {
    return node.type as GraphEntityType;
  }
  return undefined;
}

/**
 * Grouped layout: arrange nodes in lanes by entity type.
 *
 * Each entity type gets its own row (TB direction) or column (LR direction).
 * Within each lane, nodes are arranged in a grid. Lanes are separated by a gap.
 */
export function applyGroupedLayout<NodeType extends Node>(
  nodes: NodeType[],
  direction: LayoutDirection,
): NodeType[] {
  if (nodes.length === 0) return [];

  const cellX = DEFAULT_NODE_WIDTH + 60;
  const cellY = DEFAULT_NODE_HEIGHT + 60;
  const laneGap = 120;

  // Bucket nodes by entity type into ordered lanes.
  const byLane = new Map<GraphEntityType, NodeType[]>();
  for (const lane of GROUPED_LANE_ORDER) byLane.set(lane, []);

  const untyped: NodeType[] = [];
  for (const node of nodes) {
    const entityType = getNodeEntityType(node);
    const laneList = entityType ? byLane.get(entityType) : undefined;
    if (laneList) {
      laneList.push(node);
    } else {
      untyped.push(node);
    }
  }

  const layoutedNodes: NodeType[] = [];
  let laneOffset = 0;

  for (const lane of GROUPED_LANE_ORDER) {
    const laneNodes = byLane.get(lane) ?? [];
    if (laneNodes.length === 0) continue;

    const columns = Math.max(1, Math.ceil(Math.sqrt(laneNodes.length)));
    const rows = Math.max(1, Math.ceil(laneNodes.length / columns));

    for (let i = 0; i < laneNodes.length; i++) {
      const laneNode = laneNodes[i];
      if (!laneNode) continue;
      const col = i % columns;
      const row = Math.floor(i / columns);
      const x = col * cellX;
      const y = laneOffset + row * cellY;
      layoutedNodes.push({
        ...laneNode,
        position: direction === "TB" ? { x, y } : { x: y, y: x },
      });
    }
    laneOffset += rows * cellY + laneGap;
  }

  // Append any untyped nodes at the end.
  if (untyped.length > 0) {
    const columns = Math.max(1, Math.ceil(Math.sqrt(untyped.length)));
    for (let i = 0; i < untyped.length; i++) {
      const node = untyped[i];
      if (!node) continue;
      const col = i % columns;
      const row = Math.floor(i / columns);
      const x = col * cellX;
      const y = laneOffset + row * cellY;
      layoutedNodes.push({
        ...node,
        position: direction === "TB" ? { x, y } : { x: y, y: x },
      });
    }
  }

  return layoutedNodes;
}

/**
 * Apply Dagre layout to nodes and edges, returning new positioned nodes.
 *
 * For "grouped" mode, delegates to applyGroupedLayout which arranges nodes
 * by entity type in lanes. For other modes, uses Dagre for hierarchical layout.
 *
 * Nodes with no edges are separated and arranged in a compact grid below
 * the connected subgraph to prevent Dagre's default single-line placement.
 */
export function applyDagreLayout<NodeType extends Node, EdgeType extends Edge>(
  nodes: NodeType[],
  edges: EdgeType[],
  mode: LayoutMode,
  direction: LayoutDirection,
): NodeType[] {
  if (nodes.length === 0) return [];

  // Grouped mode uses lane-based layout, not Dagre.
  if (mode === "grouped") {
    return applyGroupedLayout(nodes, direction);
  }

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
    const entityType = getNodeEntityType(node);
    const shapeDims = entityType ? getShapeDimensions(entityType) : null;
    const width = (node.measured?.width ?? node.width ?? shapeDims?.width ?? DEFAULT_NODE_WIDTH) as number;
    const height = (node.measured?.height ?? node.height ?? shapeDims?.height ?? DEFAULT_NODE_HEIGHT) as number;
    g.setNode(node.id, { width, height });
  }

  for (const edge of edges) {
    if (connectedIds.has(edge.source) && connectedIds.has(edge.target)) {
      g.setEdge(edge.source, edge.target);
    }
  }

  dagre.layout(g);

  const positioned: NodeType[] = connectedNodes.map((node) => {
    const pos = g.node(node.id) as { x: number; y: number } | undefined;
    const entityType = getNodeEntityType(node);
    const shapeDims = entityType ? getShapeDimensions(entityType) : null;
    const width = (node.measured?.width ?? node.width ?? shapeDims?.width ?? DEFAULT_NODE_WIDTH) as number;
    const height = (node.measured?.height ?? node.height ?? shapeDims?.height ?? DEFAULT_NODE_HEIGHT) as number;
    if (!pos) {
      return { ...node, position: { x: 0, y: 0 } };
    }
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
  mode: Exclude<LayoutMode, "grouped">,
  startX: number,
  startY: number,
): NodeType[] {
  if (nodes.length === 0) return [];

  const gap = mode === "compact" ? 20 : 30;
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
