import { useCallback, useEffect, useState } from "react";

/**
 * Persist the recently-inspected scenario slugs in localStorage so the
 * scenario picker can offer one-click recall across the matrix, wizard, and
 * findings surfaces. Kept intentionally tiny: a capped, de-duplicated,
 * most-recent-first list under a single namespaced key.
 *
 * Degrades gracefully when storage is unavailable (private mode, SSR): the
 * hook still tracks recents in memory for the session.
 */
const STORAGE_KEY = "vrooli.business-health.recent-scenarios";
const MAX_RECENTS = 8;

const readStorage = (): string[] => {
  if (typeof window === "undefined") return [];
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((v): v is string => typeof v === "string").slice(0, MAX_RECENTS);
  } catch {
    return [];
  }
};

const writeStorage = (values: string[]): void => {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(values));
  } catch {
    // Ignore quota / access errors — recents are a convenience, not state.
  }
};

export interface RecentScenarios {
  readonly recents: string[];
  readonly remember: (scenario: string) => void;
  readonly clear: () => void;
}

export function useRecentScenarios(): RecentScenarios {
  const [recents, setRecents] = useState<string[]>(() => readStorage());

  useEffect(() => {
    writeStorage(recents);
  }, [recents]);

  const remember = useCallback((scenario: string) => {
    const slug = scenario.trim();
    if (!slug) return;
    setRecents((prev) => {
      const next = [slug, ...prev.filter((s) => s !== slug)].slice(0, MAX_RECENTS);
      // Avoid a state churn (and storage write) when nothing actually moved.
      if (next.length === prev.length && next.every((s, i) => s === prev[i])) {
        return prev;
      }
      return next;
    });
  }, []);

  const clear = useCallback(() => setRecents([]), []);

  return { recents, remember, clear };
}
