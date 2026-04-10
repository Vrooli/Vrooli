/**
 * Visual Focus
 *
 * Shared logic for applying visual focus (selection + BFS highlight + dim)
 * to a graph node. This is used by three entry points that all need the
 * same behavior:
 *
 * 1. **Node click** (all entity types):
 *    User clicks any node and sees the graph with that node highlighted
 *    and neighbors visible. The NodeInspectorPanel shows entity info.
 *
 * 2. **Page refresh** (restoring selection from URL):
 *    The `select` URL param survives refresh, but the highlight state is
 *    in-memory only. When nodes arrive from the API, we recompute the
 *    BFS neighborhood and reapply the dim effect.
 *
 * 3. **Lens drill** (arriving at a new lens via detail page lens buttons):
 *    `drillToLens` sets a `focus` URL param. When the new lens's graph
 *    data loads, we select the focus node and apply dim highlighting so
 *    the user sees the entity in context.
 *
 * WHY THIS EXISTS:
 * Without this, the same BFS + select + highlight logic was duplicated
 * (or missing) across these three paths, causing bugs:
 * - Lens drill set focusNodeId but never applied visual effects
 * - Page refresh restored selectedNodeId but not the highlight state
 * - Node click had the logic but it wasn't reusable
 */

import type { Node, Edge } from "@xyflow/react";
import { bfsNeighborhood } from "./bfs-selection";
import type { NodeHighlightState } from "../stores/graph-ui-store";

export interface VisualFocusResult {
  /** The node ID to mark as selected in the UI store. */
  selectedNodeId: string;
  /** The highlight state to apply (BFS neighborhood + dim mode). */
  highlightState: NodeHighlightState;
}

/**
 * Compute the visual focus state for a given node.
 *
 * Returns null if the node doesn't exist in the current graph
 * (e.g., it was filtered out or hasn't loaded yet).
 */
export function computeVisualFocus(
  nodeId: string,
  nodes: Node[],
  edges: Edge[],
): VisualFocusResult | null {
  const nodeExists = nodes.some((n) => n.id === nodeId);
  if (!nodeExists) return null;

  return {
    selectedNodeId: nodeId,
    highlightState: {
      highlighted: bfsNeighborhood(nodeId, nodes, edges),
      mode: "dim",
    },
  };
}

/**
 * Clear all visual focus state (selection + highlight).
 * Returns the "reset" state values to apply to the stores.
 */
export function clearVisualFocus(): {
  selectedNodeId: null;
  highlightState: NodeHighlightState;
} {
  return {
    selectedNodeId: null,
    highlightState: {
      highlighted: new Set<string>(),
      mode: "normal",
    },
  };
}
