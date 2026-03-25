import { create } from "zustand";
import { backlogService } from "../services";
import type { BacklogItem } from "../types";
import { clearStorage, fetchWithRetry, loadFromStorage, saveToStorage, shouldRefetch, type LoadStatus, type StorePersistConfig } from "./store-utils";

const PERSIST_CONFIG: StorePersistConfig = {
  key: "swarm-manager.backlog.v1",
  version: 1,
  maxItems: 200,
};

const sortBacklog = (items: BacklogItem[]): BacklogItem[] => {
  return [...items].sort((a, b) => {
    if (a.priority !== b.priority) {
      return a.priority - b.priority;
    }
    return new Date(b.updated).getTime() - new Date(a.updated).getTime();
  });
};

interface BacklogStoreState {
  items: BacklogItem[];
  status: LoadStatus;
  error: Error | null;
  isRefreshing: boolean;
  lastFetchedAt: number | null;
  fetchBacklog: (options?: { force?: boolean }) => Promise<void>;
  setItems: (items: BacklogItem[]) => void;
  upsertItem: (item: BacklogItem) => void;
  removeItem: (name: string, kind: BacklogItem["kind"]) => void;
  reset: () => void;
}

const hydrated = loadFromStorage<BacklogItem[]>(PERSIST_CONFIG, []);

export const backlogStoreInitialState = {
  items: hydrated.data,
  status: (hydrated.data.length > 0 ? "success" : "idle") as LoadStatus,
  error: null,
  isRefreshing: false,
  lastFetchedAt: hydrated.lastFetchedAt,
};

export const useBacklogStore = create<BacklogStoreState>((set, get) => ({
  ...backlogStoreInitialState,

  fetchBacklog: async ({ force = false } = {}): Promise<void> => {
    const { status, items, lastFetchedAt, isRefreshing } = get();

    if (status === "loading" || isRefreshing) {
      return;
    }

    if (!shouldRefetch({ lastFetchedAt, hasData: items.length > 0, force })) {
      return;
    }

    const hasData = items.length > 0;

    set({
      status: hasData ? "success" : "loading",
      isRefreshing: hasData,
      error: null,
    });

    try {
      const result = await fetchWithRetry(() => backlogService.list());
      const now = Date.now();
      set({
        items: sortBacklog(result),
        status: "success",
        error: null,
        isRefreshing: false,
        lastFetchedAt: now,
      });
      saveToStorage(PERSIST_CONFIG, sortBacklog(result), now);
    } catch (error) {
      set({
        error: error instanceof Error ? error : new Error("Unable to load backlog."),
        status: hasData ? "success" : "error",
        isRefreshing: false,
      });
    }
  },

  setItems: (items: BacklogItem[]): void => {
    const sorted = sortBacklog(items);
    set({ items: sorted, status: "success", error: null });
    saveToStorage(PERSIST_CONFIG, sorted, get().lastFetchedAt ?? Date.now());
  },

  upsertItem: (item: BacklogItem): void => {
    set((state) => {
      const next = state.items.some((entry) => entry.name === item.name && entry.kind === item.kind)
        ? state.items.map((entry) => (entry.name === item.name && entry.kind === item.kind ? item : entry))
        : [...state.items, item];

      const sorted = sortBacklog(next);
      saveToStorage(PERSIST_CONFIG, sorted, state.lastFetchedAt ?? Date.now());
      return { items: sorted };
    });
  },

  removeItem: (name: string, kind: BacklogItem["kind"]): void => {
    set((state) => {
      const next = state.items.filter((item) => !(item.name === name && item.kind === kind));
      saveToStorage(PERSIST_CONFIG, next, state.lastFetchedAt ?? Date.now());
      return { items: next };
    });
  },

  reset: () => {
    clearStorage(PERSIST_CONFIG.key);
    set({
      items: [],
      status: "idle" as LoadStatus,
      error: null,
      isRefreshing: false,
      lastFetchedAt: null,
    });
  },
}));
