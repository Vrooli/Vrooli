/**
 * useDrillToLens
 *
 * Provides callbacks for cross-lens navigation from detail pages.
 * Delegates to useDetailNavigation for consistent sidebar/detail coordination.
 *
 * DOC: docs/plans/navigation-header-unification-plan.md#phase-1
 */

import { useCallback } from "react";
import { useDetailNavigation } from "./useDetailNavigation";

export function useDrillToLens() {
  const { drillToLens } = useDetailNavigation();

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
