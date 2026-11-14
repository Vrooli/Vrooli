import { useCallback, useEffect, useState } from "react";

const STORAGE_PREFIX = "vrooli.tm.disclosure.";

/**
 * A boolean open/closed state that survives reloads. Used by collapsible
 * `DetailSection`s so an operator's choice to fold a heavy section (e.g. a long
 * findings list) sticks across navigation. Falls back to in-memory state when
 * `localStorage` is unavailable (SSR / privacy modes / tests).
 */
export function usePersistedDisclosure(
  storageKey: string,
  defaultOpen = true,
): readonly [boolean, () => void] {
  const fullKey = `${STORAGE_PREFIX}${storageKey}`;

  const [open, setOpen] = useState<boolean>(() => {
    if (typeof window === "undefined") return defaultOpen;
    try {
      const stored = window.localStorage.getItem(fullKey);
      if (stored === "open") return true;
      if (stored === "closed") return false;
    } catch {
      // Ignore storage access errors; use the default.
    }
    return defaultOpen;
  });

  useEffect(() => {
    if (typeof window === "undefined") return;
    try {
      window.localStorage.setItem(fullKey, open ? "open" : "closed");
    } catch {
      // Non-fatal: persistence is a convenience, not a correctness requirement.
    }
  }, [fullKey, open]);

  const toggle = useCallback(() => setOpen((current) => !current), []);

  return [open, toggle] as const;
}
