import { create } from "zustand";
import { executionService } from "../services";
import type { ExecutionRecord } from "../types";
import { fetchWithRetry, shouldRefetch, type LoadStatus } from "./store-utils";

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

export const executionStoreInitialState = {
  items: [],
  status: "idle" as LoadStatus,
  error: null,
  isRefreshing: false,
  lastFetchedAt: null,
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
      set({
        items: sortExecutions(result),
        status: "success",
        error: null,
        isRefreshing: false,
        lastFetchedAt: Date.now(),
      });
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
      return { items: sortExecutions(next) };
    });
  },

  reset: (): void => {
    set({ ...executionStoreInitialState });
  },
}));
