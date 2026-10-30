import { useCallback, useEffect, useRef, useState } from "react";
import { shortcutsClient } from "../../api/shortcuts";

interface Preferences {
  profileId: string;
  profileInstance: string;
  orderedAgentIds: string[];
  hiddenAgentIds: string[];
  revision: bigint;
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
      setPreferences({ profileId: value.profileId, profileInstance: value.profileInstance, orderedAgentIds: [...value.orderedAgentIds], hiddenAgentIds: [...value.hiddenAgentIds], revision: value.revision });
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

  const update = useCallback(async (orderedAgentIds: string[], hiddenAgentIds: string[]) => {
    if (!preferences || saving.current) return;
    saving.current = true;
    setBusy(true);
    const request = generation.current;
    try {
      const result = await shortcutsClient.updateAgentPreferences({ profileId: preferences.profileId, profileInstance: preferences.profileInstance, expectedRevision: preferences.revision, orderedAgentIds, hiddenAgentIds });
      if (request !== generation.current) return;
      const value = result.preferences;
      if (!value) throw new Error("Agent preferences missing");
      setPreferences({ profileId: value.profileId, profileInstance: value.profileInstance, orderedAgentIds: [...value.orderedAgentIds], hiddenAgentIds: [...value.hiddenAgentIds], revision: value.revision });
      setError(false);
    } catch {
      if (request === generation.current) {
        // Re-read owner state after a conflict; surface the failure so the user
        // knows their change was not accepted and can make it again.
        try {
          const result = await shortcutsClient.getAgentPreferences({});
          if (request === generation.current && result.preferences?.profileId === expectedProfileId) {
            const value = result.preferences;
            setPreferences({ profileId: value.profileId, profileInstance: value.profileInstance, orderedAgentIds: [...value.orderedAgentIds], hiddenAgentIds: [...value.hiddenAgentIds], revision: value.revision });
          } else if (request === generation.current) setPreferences(null);
        } catch { if (request === generation.current) setPreferences(null); }
        if (request === generation.current) setError(true);
      }
    } finally {
      saving.current = false;
      if (request === generation.current) setBusy(false);
    }
  }, [expectedProfileId, preferences]);

  return { preferences, busy, error, reload: load, update };
}
