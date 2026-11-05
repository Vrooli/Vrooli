/**
 * Recent-search history, persisted in localStorage.
 *
 * Kept deliberately tiny and synchronous: the search page reads the list on
 * mount and pushes each executed query. Entries are de-duplicated by
 * (query, mode) with most-recent-first ordering and capped at MAX_ENTRIES so
 * the store never grows unbounded. All access is guarded against the
 * localStorage being unavailable (private mode, SSR, quota) so a failure here
 * never breaks search.
 */

import { useSyncExternalStore } from "react";

export type SearchMode = "live" | "learnings";

export interface HistoryEntry {
  query: string;
  mode: SearchMode;
  /** Epoch millis of the most recent run of this (query, mode). */
  at: number;
}

const STORAGE_KEY = "web-search:history";
const MAX_ENTRIES = 20;

const isEntry = (value: unknown): value is HistoryEntry => {
  if (typeof value !== "object" || value === null) return false;
  const e = value as Record<string, unknown>;
  return (
    typeof e.query === "string" &&
    (e.mode === "live" || e.mode === "learnings") &&
    typeof e.at === "number"
  );
};

export function loadHistory(): HistoryEntry[] {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter(isEntry).slice(0, MAX_ENTRIES);
  } catch {
    return [];
  }
}

/**
 * Record a run of `(query, mode)`, returning the updated list (most-recent
 * first). A blank query is ignored and returns the existing list unchanged.
 */
export function recordSearch(
  query: string,
  mode: SearchMode,
  existing: HistoryEntry[] = snapshot,
): HistoryEntry[] {
  const trimmed = query.trim();
  if (!trimmed) return existing;

  const without = existing.filter(
    (e) => !(e.query === trimmed && e.mode === mode),
  );
  const next: HistoryEntry[] = [
    { query: trimmed, mode, at: Date.now() },
    ...without,
  ].slice(0, MAX_ENTRIES);

  persist(next);
  return next;
}

export function clearHistory(): HistoryEntry[] {
  persist([]);
  return [];
}

const listeners = new Set<() => void>();
let snapshot: HistoryEntry[] = loadHistory();

function persist(entries: HistoryEntry[]): void {
  snapshot = entries;
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(entries));
  } catch {
    // Best-effort: a quota/availability failure must not break search.
  }
  listeners.forEach((l) => l());
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function getSnapshot(): HistoryEntry[] {
  return snapshot;
}

/**
 * Reactive view of the recent-search history. Re-renders subscribers whenever
 * `recordSearch` / `clearHistory` mutate the store.
 */
export function useSearchHistory(): HistoryEntry[] {
  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
}
