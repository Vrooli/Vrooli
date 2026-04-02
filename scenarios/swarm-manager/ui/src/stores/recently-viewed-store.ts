/**
 * Recently Viewed Store
 *
 * Tracks entities the user has recently opened in the detail view.
 * Persisted to localStorage (raw, not via store-utils which expires after cacheTimeMs).
 * Deduplicates by entity identity and caps at MAX_ITEMS.
 */

import { create } from "zustand";
import type { DetailEntityType } from "./detail-selection-store";

const STORAGE_KEY = "swarm-manager.recently-viewed.v1";
const MAX_ITEMS = 50;

export interface RecentlyViewedItem {
  entityType: DetailEntityType;
  kind?: string;
  name?: string;
  identifier?: string;
  label: string;
  viewedAt: string; // ISO timestamp
}

interface RecentlyViewedState {
  items: RecentlyViewedItem[];
  recordView: (item: Omit<RecentlyViewedItem, "viewedAt">) => void;
  clearAll: () => void;
}

function itemKey(item: Pick<RecentlyViewedItem, "entityType" | "kind" | "name" | "identifier">): string {
  return `${item.entityType}:${item.kind ?? ""}:${item.name ?? ""}:${item.identifier ?? ""}`;
}

function loadItems(): RecentlyViewedItem[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    return Array.isArray(parsed) ? (parsed as RecentlyViewedItem[]).slice(0, MAX_ITEMS) : [];
  } catch {
    return [];
  }
}

function persistItems(items: RecentlyViewedItem[]): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(items));
  } catch {
    // Silent failure — localStorage unavailable or quota exceeded.
  }
}

export const useRecentlyViewedStore = create<RecentlyViewedState>((set, get) => ({
  items: loadItems(),

  recordView: (entry) => {
    const newItem: RecentlyViewedItem = { ...entry, viewedAt: new Date().toISOString() };
    const key = itemKey(newItem);
    const filtered = get().items.filter((i) => itemKey(i) !== key);
    const next = [newItem, ...filtered].slice(0, MAX_ITEMS);
    set({ items: next });
    persistItems(next);
  },

  clearAll: () => {
    set({ items: [] });
    persistItems([]);
  },
}));
