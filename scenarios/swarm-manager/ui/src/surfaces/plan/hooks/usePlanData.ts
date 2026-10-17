/**
 * usePlanData — subscribes the Plan board to its projection.
 *
 * Initial fetch on mount; freshness afterwards comes from /ws/graph
 * invalidations carrying the "plan" lens (routed through
 * graph-data-store.fetchGraph("plan") → plan-data-store.fetchBoard), plus a
 * slow safety poll for out-of-band changes the socket may miss.
 */

import { useEffect } from "react";
import { usePlanDataStore } from "../stores/plan-data-store";

const PLAN_SAFETY_POLL_MS = 60_000;

export function usePlanData() {
  const board = usePlanDataStore((s) => s.board);
  const loading = usePlanDataStore((s) => s.loading);
  const error = usePlanDataStore((s) => s.error);
  const windowSeconds = usePlanDataStore((s) => s.windowSeconds);
  const setWindowSeconds = usePlanDataStore((s) => s.setWindowSeconds);
  const fetchBoard = usePlanDataStore((s) => s.fetchBoard);

  useEffect(() => {
    void fetchBoard();
    const interval = window.setInterval(() => {
      void fetchBoard({ silent: true });
    }, PLAN_SAFETY_POLL_MS);
    return () => window.clearInterval(interval);
  }, [fetchBoard]);

  return { board, loading, error, windowSeconds, setWindowSeconds, refresh: () => fetchBoard({ force: true }) };
}
