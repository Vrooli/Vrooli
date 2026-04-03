import { executionService } from "../services";
import type { ExecutionRecord } from "../types";
import { createCachedStore, type CachedStoreBase } from "./create-cached-store";
import type { LoadStatus, StorePersistConfig } from "./store-utils";

const PERSIST_CONFIG: StorePersistConfig = {
  key: "swarm-manager.executions.v1",
  version: 1,
  maxItems: 100,
};

const sortExecutions = (items: ExecutionRecord[]): ExecutionRecord[] => {
  return [...items].sort((a, b) => {
    if (a.createdAt === b.createdAt) {
      return a.executionId.localeCompare(b.executionId);
    }
    return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime();
  });
};

interface ExecutionStoreState extends CachedStoreBase {
  items: ExecutionRecord[];
  fetchExecutions: (options?: { force?: boolean }) => Promise<void>;
  upsertExecution: (record: ExecutionRecord) => void;
}

const { useStore, initialState } = createCachedStore<ExecutionRecord, ExecutionStoreState>({
  persist: PERSIST_CONFIG,
  fetchFn: () => executionService.list(),
  sortFn: sortExecutions,
  errorMessage: "Unable to load executions.",
  getItems: (state) => state.items,
  setItemsPartial: (items) => ({ items }) as Partial<ExecutionStoreState>,
  actions: ({ doFetch, set, save, sortFn }) => ({
    items: [] as ExecutionRecord[],

    fetchExecutions: (options?: { force?: boolean }) => doFetch(options),

    upsertExecution: (record: ExecutionRecord): void => {
      set((state) => {
        const next = state.items.some((entry) => entry.executionId === record.executionId)
          ? state.items.map((entry) => (entry.executionId === record.executionId ? record : entry))
          : [...state.items, record];
        const sorted = sortFn(next);
        save(sorted, state.lastFetchedAt ?? Date.now());
        return { items: sorted } as Partial<ExecutionStoreState>;
      });
    },
  }),
});

export const executionStoreInitialState = {
  items: initialState.items as ExecutionRecord[],
  status: initialState.status as LoadStatus,
  error: initialState.error,
  isRefreshing: initialState.isRefreshing,
  lastFetchedAt: initialState.lastFetchedAt,
};

export const useExecutionStore = useStore;
