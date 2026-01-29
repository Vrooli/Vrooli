import { create } from "zustand";
import { backlogService } from "../services";
import type { BacklogItem } from "../types";
import { fetchWithRetry, shouldRefetch, type LoadStatus } from "./store-utils";

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

export const backlogStoreInitialState = {
  items: [],
  status: "idle" as LoadStatus,
  error: null,
  isRefreshing: false,
  lastFetchedAt: null,
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
      set({
        items: sortBacklog(result),
        status: "success",
        error: null,
        isRefreshing: false,
        lastFetchedAt: Date.now(),
      });
    } catch (error) {
      set({
        error: error instanceof Error ? error : new Error("Unable to load backlog."),
        status: hasData ? "success" : "error",
        isRefreshing: false,
      });
    }
  },

  setItems: (items: BacklogItem[]): void => {
    set({ items: sortBacklog(items), status: "success", error: null });
  },

  upsertItem: (item: BacklogItem): void => {
    set((state) => {
      const next = state.items.some((entry) => entry.name === item.name && entry.kind === item.kind)
        ? state.items.map((entry) => (entry.name === item.name && entry.kind === item.kind ? item : entry))
        : [...state.items, item];

      return { items: sortBacklog(next) };
    });
  },

  removeItem: (name: string, kind: BacklogItem["kind"]): void => {
    set((state) => ({
      items: state.items.filter((item) => !(item.name === name && item.kind === kind)),
    }));
  },

  reset: () => {
    set({ ...backlogStoreInitialState });
  },
}));
