import * as React from "react";

import {
  usePersistedPreference,
  type PreferenceStorage,
} from "../../../hooks/usePersistedPreference";

/**
 * One entry in the persisted recent-targets list. Scenario is the directory
 * name (e.g. `architecture-cartographer`); `lastOpenedAt` is an ISO timestamp.
 */
export interface RecentTarget {
  scenario: string;
  lastOpenedAt: string;
}

const STORAGE_KEY = "cartographer.recentTargets";
const MAX_ENTRIES = 8;

function validate(raw: unknown): RecentTarget[] | null {
  if (!Array.isArray(raw)) return null;
  const out: RecentTarget[] = [];
  for (const entry of raw) {
    if (!entry || typeof entry !== "object") continue;
    const candidate = entry as Record<string, unknown>;
    const scenario = candidate.scenario;
    const lastOpenedAt = candidate.lastOpenedAt;
    if (typeof scenario !== "string" || scenario.length === 0) continue;
    if (typeof lastOpenedAt !== "string" || lastOpenedAt.length === 0) continue;
    out.push({ scenario, lastOpenedAt });
  }
  return out;
}

export interface UseRecentTargetsOptions {
  /** Test seam — inject a custom storage implementation. */
  storage?: PreferenceStorage;
  /** Test seam — control the timestamp used when recording. */
  now?: () => Date;
}

export interface UseRecentTargetsResult {
  recent: readonly RecentTarget[];
  /** Move (or insert) a scenario to the head of the list with current timestamp. */
  record: (scenario: string) => void;
  /** Remove a scenario entirely from the list. */
  remove: (scenario: string) => void;
  /** Clear the entire list. */
  clear: () => void;
}

/**
 * useRecentTargets — persists the most-recently-opened scenarios in
 * localStorage so the overview page can show a "jump back in" list.
 *
 * Persistence flows through `usePersistedPreference`, which exposes a
 * `storage` seam — tests inject an in-memory adapter. The list is capped at
 * `MAX_ENTRIES` and ordered most-recent-first; duplicate `record(scenario)`
 * calls move the existing entry to the head rather than create a second
 * entry.
 *
 * The mutator callbacks read from a ref-mirrored copy of `value` so that
 * back-to-back calls in the same render tick (`record("a"); record("b");`)
 * see the latest list rather than a stale closure value.
 */
export function useRecentTargets({
  storage,
  now,
}: UseRecentTargetsOptions = {}): UseRecentTargetsResult {
  const [value, setValue] = usePersistedPreference<RecentTarget[]>({
    key: STORAGE_KEY,
    defaultValue: [],
    validate,
    storage,
  });

  const valueRef = React.useRef(value);
  React.useEffect(() => {
    valueRef.current = value;
  }, [value]);

  const nowRef = React.useRef<() => Date>(now ?? (() => new Date()));
  React.useEffect(() => {
    nowRef.current = now ?? (() => new Date());
  }, [now]);

  const record = React.useCallback(
    (scenario: string) => {
      if (scenario.length === 0) return;
      const next = [
        { scenario, lastOpenedAt: nowRef.current().toISOString() },
        ...valueRef.current.filter((entry) => entry.scenario !== scenario),
      ].slice(0, MAX_ENTRIES);
      valueRef.current = next;
      setValue(next);
    },
    [setValue],
  );

  const remove = React.useCallback(
    (scenario: string) => {
      const next = valueRef.current.filter((entry) => entry.scenario !== scenario);
      valueRef.current = next;
      setValue(next);
    },
    [setValue],
  );

  const clear = React.useCallback(() => {
    valueRef.current = [];
    setValue([]);
  }, [setValue]);

  return { recent: value, record, remove, clear };
}
