import { useSyncExternalStore } from "react";

/**
 * Last live-search health snapshot — a small cross-route store so the
 * Operations panel can surface the most-recent live-search response signals
 * (cached / degraded / degraded_reason / result count) even though it renders
 * on a different route than the search page that produced them.
 *
 * Backed by localStorage so the readout survives a reload, and exposed through
 * a `useSyncExternalStore` hook so any mounted reader re-renders when the
 * search page records a new snapshot.
 */
export interface LiveSearchHealth {
  query: string;
  cached: boolean;
  degraded: boolean;
  degradedReason: string;
  resultCount: number;
  /** Epoch millis of the response. */
  at: number;
}

const STORAGE_KEY = "web-search:live-health";

let current: LiveSearchHealth | null = read();
const listeners = new Set<() => void>();

function read(): LiveSearchHealth | null {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed: unknown = JSON.parse(raw);
    if (typeof parsed !== "object" || parsed === null) return null;
    const p = parsed as Record<string, unknown>;
    if (
      typeof p.query !== "string" ||
      typeof p.cached !== "boolean" ||
      typeof p.degraded !== "boolean" ||
      typeof p.degradedReason !== "string" ||
      typeof p.resultCount !== "number" ||
      typeof p.at !== "number"
    ) {
      return null;
    }
    return p as unknown as LiveSearchHealth;
  } catch {
    return null;
  }
}

export function recordLiveSearchHealth(snapshot: LiveSearchHealth): void {
  current = snapshot;
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(snapshot));
  } catch {
    // Best-effort persistence; the in-memory value still updates subscribers.
  }
  listeners.forEach((l) => l());
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function getSnapshot(): LiveSearchHealth | null {
  return current;
}

export function useLiveSearchHealth(): LiveSearchHealth | null {
  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
}
