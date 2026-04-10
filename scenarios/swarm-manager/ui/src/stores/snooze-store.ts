/**
 * Snooze Store
 *
 * Zustand store for snoozing actionable items (backlog, execution, capture).
 * Persists to localStorage via store-utils pattern. Auto-purges expired entries.
 */

import { useMemo } from "react";
import { create } from "zustand";
import { subscribeWithSelector } from "zustand/middleware";
import type { SnoozeEntry } from "../lib/snooze-utils";
import { isExpired } from "../lib/snooze-utils";
import { loadFromStorage, saveToStorage, type StorePersistConfig } from "./store-utils";

// ---------------------------------------------------------------------------
// Persistence config
// ---------------------------------------------------------------------------

const PERSIST_CONFIG: StorePersistConfig = {
  key: "swarm-manager.snooze.v1",
  version: 1,
};

const AUTO_PURGE_INTERVAL_MS = 60_000;

// ---------------------------------------------------------------------------
// Store types
// ---------------------------------------------------------------------------

interface SnoozeState {
  entries: Map<string, SnoozeEntry>;
  snooze: (key: string, expiresAt: number) => void;
  unsnooze: (key: string) => void;
  isSnoozed: (key: string) => boolean;
  snoozedKeys: () => Set<string>;
  purgeExpired: () => void;
}

// ---------------------------------------------------------------------------
// Serialization helpers (Map ↔ array for JSON)
// ---------------------------------------------------------------------------

type SerializedEntries = [string, SnoozeEntry][];

function serializeEntries(entries: Map<string, SnoozeEntry>): SerializedEntries {
  return Array.from(entries.entries());
}

function deserializeEntries(arr: SerializedEntries): Map<string, SnoozeEntry> {
  return new Map(arr);
}

// ---------------------------------------------------------------------------
// Hydrate from localStorage and purge expired on load
// ---------------------------------------------------------------------------

function hydrateAndPurge(): Map<string, SnoozeEntry> {
  const hydrated = loadFromStorage<SerializedEntries>(PERSIST_CONFIG, []);
  const entries = deserializeEntries(hydrated.data);
  // Purge expired on hydration
  for (const [key, entry] of entries) {
    if (isExpired(entry)) entries.delete(key);
  }
  return entries;
}

// ---------------------------------------------------------------------------
// Persist helper
// ---------------------------------------------------------------------------

function persist(entries: Map<string, SnoozeEntry>): void {
  saveToStorage(PERSIST_CONFIG, serializeEntries(entries), Date.now());
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useSnoozeStore = create<SnoozeState>()(
  subscribeWithSelector((set, get) => ({
    entries: hydrateAndPurge(),

    snooze(key: string, expiresAt: number) {
      set((state) => {
        const next = new Map(state.entries);
        next.set(key, { key, expiresAt });
        persist(next);
        return { entries: next };
      });
    },

    unsnooze(key: string) {
      set((state) => {
        const next = new Map(state.entries);
        next.delete(key);
        persist(next);
        return { entries: next };
      });
    },

    isSnoozed(key: string): boolean {
      const entry = get().entries.get(key);
      return entry !== undefined && !isExpired(entry);
    },

    snoozedKeys(): Set<string> {
      const result = new Set<string>();
      for (const [key, entry] of get().entries) {
        if (!isExpired(entry)) result.add(key);
      }
      return result;
    },

    purgeExpired() {
      set((state) => {
        let changed = false;
        const next = new Map(state.entries);
        for (const [key, entry] of next) {
          if (isExpired(entry)) {
            next.delete(key);
            changed = true;
          }
        }
        if (changed) persist(next);
        return changed ? { entries: next } : state;
      });
    },
  })),
);

// ---------------------------------------------------------------------------
// Auto-purge side-effect (60s interval)
// ---------------------------------------------------------------------------

let purgeInterval: ReturnType<typeof setInterval> | null = null;

export function startAutoPurge(): void {
  if (purgeInterval) return;
  purgeInterval = setInterval(() => {
    useSnoozeStore.getState().purgeExpired();
  }, AUTO_PURGE_INTERVAL_MS);
}

export function stopAutoPurge(): void {
  if (purgeInterval) {
    clearInterval(purgeInterval);
    purgeInterval = null;
  }
}

// Start auto-purge when module loads in browser
if (typeof window !== "undefined") {
  startAutoPurge();
}

// ---------------------------------------------------------------------------
// React hook for stable snoozedKeys derivation
// ---------------------------------------------------------------------------

/**
 * Derive snoozed keys from the store's `entries` Map with a stable Set reference.
 *
 * IMPORTANT: Do NOT use `useSnoozeStore((s) => s.snoozedKeys())` in components.
 * That pattern calls the method inside a zustand selector, creating a new Set on
 * every evaluation. Because `Object.is(newSet, oldSet)` is always false, zustand
 * always considers the value changed, causing infinite re-render loops with
 * React 18's `useSyncExternalStore`.
 *
 * This hook subscribes to the raw `entries` Map (stable reference when unchanged)
 * and derives the Set in a `useMemo`.
 */
export function useSnoozedKeys(): Set<string> {
  const entries = useSnoozeStore((s) => s.entries);
  return useMemoSnoozedKeys(entries);
}

function useMemoSnoozedKeys(entries: Map<string, SnoozeEntry>): Set<string> {
  return useMemo(() => {
    const result = new Set<string>();
    for (const [key, entry] of entries) {
      if (!isExpired(entry)) result.add(key);
    }
    return result;
  }, [entries]);
}
