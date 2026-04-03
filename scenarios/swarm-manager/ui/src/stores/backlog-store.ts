import { backlogService } from "../services";
import type { BacklogItem } from "../types";
import { createCachedStore, type CachedStoreBase } from "./create-cached-store";
import type { LoadStatus, StorePersistConfig } from "./store-utils";

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

interface BacklogStoreState extends CachedStoreBase {
  items: BacklogItem[];
  fetchBacklog: (options?: { force?: boolean }) => Promise<void>;
  setItems: (items: BacklogItem[]) => void;
  upsertItem: (item: BacklogItem) => void;
  removeItem: (name: string, kind: BacklogItem["kind"]) => void;
}

const { useStore, initialState } = createCachedStore<BacklogItem, BacklogStoreState>({
  persist: PERSIST_CONFIG,
  fetchFn: () => backlogService.list(),
  sortFn: sortBacklog,
  errorMessage: "Unable to load backlog.",
  getItems: (state) => state.items,
  setItemsPartial: (items) => ({ items }) as Partial<BacklogStoreState>,
  actions: ({ doFetch, set, get, save, sortFn }) => ({
    items: [] as BacklogItem[], // placeholder; overridden by initialState spread

    fetchBacklog: (options?: { force?: boolean }) => doFetch(options),

    setItems: (items: BacklogItem[]): void => {
      const sorted = sortFn(items);
      set({ items: sorted, status: "success", error: null } as Partial<BacklogStoreState>);
      save(sorted, get().lastFetchedAt ?? Date.now());
    },

    upsertItem: (item: BacklogItem): void => {
      set((state) => {
        const next = state.items.some((entry) => entry.name === item.name && entry.kind === item.kind)
          ? state.items.map((entry) => (entry.name === item.name && entry.kind === item.kind ? item : entry))
          : [...state.items, item];

        const sorted = sortFn(next);
        save(sorted, state.lastFetchedAt ?? Date.now());
        return { items: sorted } as Partial<BacklogStoreState>;
      });
    },

    removeItem: (name: string, kind: BacklogItem["kind"]): void => {
      set((state) => {
        const next = state.items.filter((item) => !(item.name === name && item.kind === kind));
        save(next, state.lastFetchedAt ?? Date.now());
        return { items: next } as Partial<BacklogStoreState>;
      });
    },
  }),
});

export const backlogStoreInitialState = {
  items: initialState.items as BacklogItem[],
  status: initialState.status as LoadStatus,
  error: initialState.error,
  isRefreshing: initialState.isRefreshing,
  lastFetchedAt: initialState.lastFetchedAt,
};

export const useBacklogStore = useStore;

// ---------------------------------------------------------------------------
// Derived selectors
// ---------------------------------------------------------------------------

/** Build a `Set<"kind/name">` from backlog items. Reusable across any component. */
export function buildActiveBacklogKeys(items: BacklogItem[]): Set<string> {
  return new Set(items.map((i) => `${i.kind}/${i.name}`));
}
