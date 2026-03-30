/**
 * useDrillToLens
 *
 * Provides callbacks for cross-lens navigation from detail pages.
 * Closes the detail overlay and drills into flow or operations lens
 * focused on the entity's graph node.
 */

import { useCallback } from "react";
import { useSearchParams } from "react-router-dom";
import { useGraphDataStore } from "../surfaces/graph/stores/graph-data-store";
import { useGraphUIStore } from "../surfaces/graph/stores/graph-ui-store";
import { useDetailSelectionStore } from "../stores/detail-selection-store";
import { getGraphNodeLabel } from "../surfaces/graph/types";
import type { GraphLens } from "../surfaces/graph/stores/graph-data-store";

export function useDrillToLens() {
  const [, setSearchParams] = useSearchParams();
  const lens = useGraphDataStore((s) => s.lens);
  const nodes = useGraphDataStore((s) => s.nodes);
  const setFocusNodeLabel = useGraphUIStore((s) => s.setFocusNodeLabel);
  const clearDetailSelection = useDetailSelectionStore((s) => s.clearSelection);

  const drillToLens = useCallback(
    (nodeId: string, targetLens: GraphLens) => {
      const node = nodes.find((n) => n.id === nodeId);
      if (node) setFocusNodeLabel(getGraphNodeLabel(node));

      clearDetailSelection();

      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        next.set("lens", targetLens);
        next.set("focus", nodeId);
        next.set("returnLens", lens);
        next.delete("select");
        next.delete("detail");
        next.delete("kind");
        next.delete("name");
        next.delete("execId");
        next.delete("tab");
        return next;
      });
    },
    [clearDetailSelection, lens, nodes, setFocusNodeLabel, setSearchParams],
  );

  const drillToFlow = useCallback(
    (nodeId: string) => drillToLens(nodeId, "flow"),
    [drillToLens],
  );

  const drillToOperations = useCallback(
    (nodeId: string) => drillToLens(nodeId, "operations"),
    [drillToLens],
  );

  return { drillToFlow, drillToOperations };
}
