import { create } from "zustand";
import { executionService } from "../services";
import type { ExecutionRecord } from "../types";
import { clearStorage, fetchWithRetry, loadFromStorage, saveToStorage, shouldRefetch, type LoadStatus, type StorePersistConfig } from "./store-utils";

const PERSIST_CONFIG: StorePersistConfig = {
  key: "swarm-manager.executions.v1",
  version: 1,
  maxItems: 100,
};

interface ExecutionStoreState {
  items: ExecutionRecord[];
  status: LoadStatus;
  error: Error | null;
  isRefreshing: boolean;
  lastFetchedAt: number | null;
  fetchExecutions: (options?: { force?: boolean }) => Promise<void>;
  upsertExecution: (record: ExecutionRecord) => void;
  reset: () => void;
}

const hydrated = loadFromStorage<ExecutionRecord[]>(PERSIST_CONFIG, []);

export const executionStoreInitialState = {
  items: hydrated.data,
  status: (hydrated.data.length > 0 ? "success" : "idle") as LoadStatus,
  error: null,
  isRefreshing: false,
  lastFetchedAt: hydrated.lastFetchedAt,
};

const sortExecutions = (items: ExecutionRecord[]): ExecutionRecord[] => {
  return [...items].sort((a, b) => {
    if (a.createdAt === b.createdAt) {
      return a.executionId.localeCompare(b.executionId);
    }
    return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime();
  });
};

export const useExecutionStore = create<ExecutionStoreState>((set, get) => ({
  ...executionStoreInitialState,

  fetchExecutions: async ({ force = false } = {}): Promise<void> => {
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
      const result = await fetchWithRetry(() => executionService.list());
      const now = Date.now();
      set({
        items: sortExecutions(result),
        status: "success",
        error: null,
        isRefreshing: false,
        lastFetchedAt: now,
      });
      saveToStorage(PERSIST_CONFIG, sortExecutions(result), now);
    } catch (error) {
      set({
        error: error instanceof Error ? error : new Error("Unable to load executions."),
        status: hasData ? "success" : "error",
        isRefreshing: false,
      });
    }
  },

  upsertExecution: (record: ExecutionRecord): void => {
    set((state) => {
      const next = state.items.some((entry) => entry.executionId === record.executionId)
        ? state.items.map((entry) => (entry.executionId === record.executionId ? record : entry))
        : [...state.items, record];
      const sorted = sortExecutions(next);
      saveToStorage(PERSIST_CONFIG, sorted, state.lastFetchedAt ?? Date.now());
      return { items: sorted };
    });
  },

  reset: (): void => {
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
