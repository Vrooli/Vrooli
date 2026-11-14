import { useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";

/** What happened on the screen the reader just came from, to say once on arrival. */
export type ReachNotice =
  | { kind: "removed"; name: string; uncertain: boolean }
  | { kind: "channelSaved"; channel: string; uncertain: boolean };

/**
 * Read the notice a screen was navigated to with, then clear it from history so
 * a reload does not repeat news that is no longer news.
 */
export function useRouteNotice(): ReachNotice | null {
  const location = useLocation();
  const navigate = useNavigate();
  const [notice] = useState(() => (location.state as { notice?: ReachNotice } | null)?.notice ?? null);
  useEffect(() => {
    if (notice) navigate(`${location.pathname}${location.search}`, { replace: true, state: null });
    // Run once for the notice that arrived with this mount.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  return notice;
}
