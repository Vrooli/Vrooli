/**
 * useDrillToLens
 *
 * Provides callbacks for cross-lens navigation from detail pages.
 * Navigates to graph lens routes with the target node focused.
 */

import { useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { graphPath, type AppGraphLens } from "../app/routes/route-paths";

export function useDrillToLens() {
  const navigate = useNavigate();
  const drillToLens = useCallback(
    (nodeId: string, lens: AppGraphLens) => navigate(graphPath({ lens, focus: nodeId, select: nodeId })),
    [navigate],
  );

  const drillToFocus = useCallback(
    (nodeId: string) => drillToLens(nodeId, "focus"),
    [drillToLens],
  );

  return { drillToLens, drillToFocus };
}
