import { useCallback, useEffect, useState } from "react";
import { DEFAULT_SHORTCUTS, type ShortcutEntry } from "../../consts/shortcuts";
import { shortcutsClient } from "../../api/shortcuts";

/**
 * Wire shortcuts to the UI shape. Proto string fields are never undefined, so
 * blank optionals are normalized here rather than at every read site.
 */
function decodeShortcuts(wire: readonly { label: string; command: string; description: string; agentId: string }[]): ShortcutEntry[] {
  return wire.map((entry) => ({
    label: entry.label,
    command: entry.command,
    description: entry.description || undefined,
    agentId: entry.agentId || undefined,
  }));
}

/** The shortcuts a launcher shows: the caller's list, or the effective profile's with its id. */
export function useEffectiveShortcuts(open: boolean, shortcutsProp: ShortcutEntry[] | undefined) {
  const [apiShortcuts, setApiShortcuts] = useState<ShortcutEntry[] | null>(null);
  const [profileId, setProfileId] = useState<string | null>(null);

  const fetchEffective = useCallback(() => shortcutsClient.getEffective({}).then((response) => {
    setApiShortcuts(decodeShortcuts(response.shortcuts));
    setProfileId(response.profileId);
    return response.profileId;
  }), []);

  useEffect(() => {
    if (!open || shortcutsProp) return;
    let cancelled = false;
    setProfileId(null);
    shortcutsClient.getEffective({})
      .then((response) => {
        if (cancelled) return;
        setApiShortcuts(decodeShortcuts(response.shortcuts));
        setProfileId(response.profileId);
      })
      .catch(() => {
        if (!cancelled) { setApiShortcuts(null); setProfileId(null); }
      });
    return () => { cancelled = true; };
  }, [open, shortcutsProp]);

  return { shortcuts: shortcutsProp ?? apiShortcuts ?? DEFAULT_SHORTCUTS, profileId, fetchEffective };
}
