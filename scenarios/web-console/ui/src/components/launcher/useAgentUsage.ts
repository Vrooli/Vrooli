import { useCallback, useEffect, useState } from "react";
import { getAgentUsage } from "../../api/agentUsage";

export interface AgentUsageWindow {
  pool: string;
  windowMinutes: number;
  usedPercent: number;
  resetsAt: string;
}

/** A provider reading plus the confidence in its association with a launch. */
export interface AgentUsage {
  targetId: string;
  agentId: string;
  match: "verified" | "likely" | "unavailable";
  reason: string;
  source: string;
  observedAt: string;
  windows: AgentUsageWindow[];
}

type Snapshot = { targetId: string; usages: Record<string, AgentUsage | null>; loading: boolean };

/** Requests are scoped to the selected target and cancelled on target changes. */
export function useAgentUsage(enabled: boolean, targetId: string, agentKey: string) {
  const [snapshot, setSnapshot] = useState<Snapshot>({ targetId, usages: {}, loading: false });
  const [revision, setRevision] = useState(0);
  const refresh = useCallback(() => { setRevision((value) => value + 1); }, []);
  useEffect(() => {
    const abort = new AbortController();
    const ids = enabled ? agentKey.split(",").filter(Boolean) : [];
    setSnapshot({ targetId, usages: {}, loading: ids.length > 0 });
    if (!ids.length) return () => { abort.abort(); };
    void Promise.all(ids.map(async (agentId) => {
      let usage: AgentUsage | null = null;
      try {
        const value = await getAgentUsage({ targetId, agentId }, { signal: abort.signal });
        const match = value.match;
        if (value.targetId === targetId && value.agentId === agentId &&
          (match === "verified" || match === "likely" || match === "unavailable") && Array.isArray(value.windows)) usage = { ...value, match };
      } catch { /* The launcher is usable even if the usage service is not. */ }
      if (!abort.signal.aborted) setSnapshot((current) => current.targetId === targetId
        ? { ...current, usages: { ...current.usages, [agentId]: usage } } : current);
    })).then(() => {
      if (!abort.signal.aborted) setSnapshot((current) => current.targetId === targetId ? { ...current, loading: false } : current);
    });
    return () => { abort.abort(); };
  }, [agentKey, enabled, revision, targetId]);
  useEffect(() => {
    if (!enabled) return;
    const timer = window.setInterval(refresh, 5 * 60_000);
    return () => { window.clearInterval(timer); };
  }, [enabled, refresh]);
  return { usages: snapshot.targetId === targetId && enabled ? snapshot.usages : {}, loading: enabled && (snapshot.targetId !== targetId ? Boolean(agentKey) : snapshot.loading), refresh };
}
