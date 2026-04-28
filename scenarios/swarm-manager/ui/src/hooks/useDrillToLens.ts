/**
 * useDrillToLens
 *
 * Provides callbacks for cross-lens navigation from detail pages.
 * Navigates to graph lens routes with the target node focused.
 *
 * DOC: docs/plans/navigation-header-unification-plan.md#phase-1
 */

import { useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { graphPath } from "../app/routes/route-paths";
import type { GraphLens } from "../surfaces/graph/stores/graph-data-store";

export function useDrillToLens() {
  const navigate = useNavigate();
  const drillToLens = useCallback(
    (nodeId: string, lens: GraphLens) => navigate(graphPath({ lens, focus: nodeId, select: nodeId })),
    [navigate],
  );

  const drillToOperations = useCallback(
    (nodeId: string) => drillToLens(nodeId, "operations"),
    [drillToLens],
  );

  const drillToTopology = useCallback(
    (nodeId: string) => drillToLens(nodeId, "topology"),
    [drillToLens],
  );

  return { drillToLens, drillToTopology, drillToOperations };
}
