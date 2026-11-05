import { useCallback, useEffect, useRef, useState } from "react";
import type { AgentPreferences as WirePreferences } from "@vrooli/proto-types/web-console/v1/shortcuts/shortcuts_pb";
import { shortcutsClient } from "../../api/shortcuts";

export type AgentOrderMode = "custom" | "recent";

export interface Preferences {
  profileId: string;
  profileInstance: string;
  orderedAgentIds: string[];
  hiddenAgentIds: string[];
  orderMode: AgentOrderMode;
  /** RFC 3339 time of each agent's most recent launch from the launcher. */
  lastLaunchedAt: Record<string, string>;
  revision: bigint;
}

export interface PreferenceChange {
  orderedAgentIds?: string[];
  hiddenAgentIds?: string[];
  orderMode?: AgentOrderMode;
}

function decode(value: WirePreferences): Preferences {
  return {
    profileId: value.profileId,
    profileInstance: value.profileInstance,
    orderedAgentIds: [...value.orderedAgentIds],
    hiddenAgentIds: [...value.hiddenAgentIds],
    orderMode: value.orderMode === "recent" ? "recent" : "custom",
    lastLaunchedAt: { ...value.lastLaunchedAt },
    revision: value.revision,
  };
}

/** Profile-owned agent preferences. A failed or conflicting write never changes shortcuts. */
export function useAgentPreferences(enabled: boolean, expectedProfileId: string | null) {
  const [preferences, setPreferences] = useState<Preferences | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(false);
  const generation = useRef(0);
  const saving = useRef(false);

  const load = useCallback(async () => {
    const request = ++generation.current;
    setBusy(true);
    try {
      const result = await shortcutsClient.getAgentPreferences({});
      if (request !== generation.current) return;
      const value = result.preferences;
      if (!value || value.profileId !== expectedProfileId) throw new Error("Agent preferences profile changed");
      setPreferences(decode(value));
      setError(false);
    } catch {
      if (request === generation.current) { setPreferences(null); setError(true); }
    } finally {
      if (request === generation.current) setBusy(false);
    }
  }, [expectedProfileId]);

  useEffect(() => {
    const requestGeneration = generation;
    if (enabled) void load();
    else { generation.current++; setPreferences(null); setError(false); setBusy(false); }
    return () => { requestGeneration.current++; };
  }, [enabled, load]);

  const update = useCallback(async (change: PreferenceChange) => {
    if (!preferences || saving.current) return;
    saving.current = true;
    setBusy(true);
    const request = generation.current;
    try {
      const result = await shortcutsClient.updateAgentPreferences({
        profileId: preferences.profileId, profileInstance: preferences.profileInstance, expectedRevision: preferences.revision,
        orderedAgentIds: change.orderedAgentIds ?? preferences.orderedAgentIds,
        hiddenAgentIds: change.hiddenAgentIds ?? preferences.hiddenAgentIds,
        orderMode: change.orderMode ?? preferences.orderMode,
      });
      if (request !== generation.current) return;
      if (!result.preferences) throw new Error("Agent preferences missing");
      setPreferences(decode(result.preferences));
      setError(false);
    } catch {
      if (request === generation.current) {
        // Re-read owner state after a conflict; surface the failure so the user
        // knows their change was not accepted and can make it again.
        try {
          const result = await shortcutsClient.getAgentPreferences({});
          if (request === generation.current && result.preferences?.profileId === expectedProfileId) setPreferences(decode(result.preferences));
          else if (request === generation.current) setPreferences(null);
        } catch { if (request === generation.current) setPreferences(null); }
        if (request === generation.current) setError(true);
      }
    } finally {
      saving.current = false;
      if (request === generation.current) setBusy(false);
    }
  }, [expectedProfileId, preferences]);

  /**
   * Records a launch for the Recent order. Nothing waits on it and a lost record
   * costs one position, so it never reports an error; a conflict re-reads once.
   */
  const recordLaunch = useCallback((agentId: string) => {
    if (!preferences) return;
    const write = (current: Preferences) => shortcutsClient.updateAgentPreferences({
      profileId: current.profileId, profileInstance: current.profileInstance, expectedRevision: current.revision,
      orderedAgentIds: current.orderedAgentIds, hiddenAgentIds: current.hiddenAgentIds, orderMode: current.orderMode, launchedAgentId: agentId,
    });
    void write(preferences).catch(async () => {
      const fresh = (await shortcutsClient.getAgentPreferences({})).preferences;
      if (fresh) await write(decode(fresh));
    }).catch(() => undefined);
  }, [preferences]);

  return { preferences, busy, error, reload: load, update, recordLaunch };
}

export type AgentPreferencesController = ReturnType<typeof useAgentPreferences>;
