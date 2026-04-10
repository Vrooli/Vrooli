import { backlogService } from "../services";
import type { BacklogItem, ItemBlockingInfo } from "../types";
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
  /** Per-item blocking info from the list endpoint, keyed by "kind/name". */
  blockingMap: Record<string, ItemBlockingInfo>;
  fetchBacklog: (options?: { force?: boolean }) => Promise<void>;
  setItems: (items: BacklogItem[]) => void;
  upsertItem: (item: BacklogItem) => void;
  removeItem: (name: string, kind: BacklogItem["kind"]) => void;
}

// Module-level cache for the blocking map from the latest list() call.
// Populated by the fetchFn wrapper and read by the doFetch onSuccess in actions.
let _latestBlockingMap: Record<string, ItemBlockingInfo> = {};

const { useStore, initialState } = createCachedStore<BacklogItem, BacklogStoreState>({
  persist: PERSIST_CONFIG,
  fetchFn: async () => {
    const result = await backlogService.list();
    _latestBlockingMap = result.blocking;
    return result.items;
  },
  sortFn: sortBacklog,
  errorMessage: "Unable to load backlog.",
  getItems: (state) => state.items,
  setItemsPartial: (items) => ({ items, blockingMap: _latestBlockingMap }) as Partial<BacklogStoreState>,
  actions: ({ doFetch, set, get, save, sortFn }) => ({
    items: [] as BacklogItem[], // placeholder; overridden by initialState spread
    blockingMap: {} as Record<string, ItemBlockingInfo>,

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

/** Build a `Set<"kind/name">` from non-archived backlog items. Used by the
 *  Command Post to exclude pending questions from archived items. */
export function buildActiveBacklogKeys(items: BacklogItem[]): Set<string> {
  return new Set(
    items.filter((i) => i.archivedAt == null).map((i) => `${i.kind}/${i.name}`),
  );
}
