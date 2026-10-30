import { useEffect, useState } from "react";
import { capabilitiesClient } from "../../api/capabilities";

export interface AgentUpdate {
  installed: string;
  latest: string;
  pinned: string;
  status: string;
  checkedAt: string;
  sourceKind: string;
  sourceId: string;
  probeVerified: boolean;
  note: string;
}

/** Read checks for this target only while management is open. Failure is unknown. */
export function useAgentUpdates(open: boolean, targetId: string, agentKey: string) {
  const [updates, setUpdates] = useState<Record<string, AgentUpdate | null>>({});
  const [loading, setLoading] = useState(false);
  useEffect(() => {
    setUpdates({});
    if (!open || !agentKey) { setLoading(false); return; }
    const abort = new AbortController();
    setLoading(true);
    const ids = agentKey.split(",").filter(Boolean);
    void Promise.all(ids.map(async (agentId) => {
      let update: AgentUpdate | null = null;
      try {
        const value = await capabilitiesClient.agentUpdate({ targetId, agentId }, { signal: abort.signal });
        if (value.targetId === targetId && value.agentId === agentId) {
          update = { installed: value.installed, latest: value.latest, pinned: value.pinned, status: value.status, checkedAt: value.checkedAt, sourceKind: value.sourceKind, sourceId: value.sourceId, probeVerified: value.probeVerified, note: value.note };
        }
      } catch { /* Unavailable is not an update. */ }
      if (!abort.signal.aborted) setUpdates((current) => ({ ...current, [agentId]: update }));
    })).then(() => {
      if (!abort.signal.aborted) setLoading(false);
    });
    return () => { abort.abort(); };
  }, [agentKey, open, targetId]);
  return { updates, loading };
}
