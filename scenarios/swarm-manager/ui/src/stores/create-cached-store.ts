import { create } from "zustand";
import { clearStorage, fetchWithRetry, loadFromStorage, saveToStorage, shouldRefetch, type LoadStatus, type StorePersistConfig } from "./store-utils";

// ---------------------------------------------------------------------------
// Public types
// ---------------------------------------------------------------------------

/** Base state that every cached store includes. */
export interface CachedStoreBase {
  status: LoadStatus;
  error: Error | null;
  isRefreshing: boolean;
  lastFetchedAt: number | null;
  reset: () => void;
}

/** Helpers passed to the `actions` callback so stores can build domain logic. */
export interface StoreHelpers<TItem, TState extends CachedStoreBase> {
  set: (partial: Partial<TState> | ((state: TState) => Partial<TState>)) => void;
  get: () => TState;
  persistConfig: StorePersistConfig;
  sortFn: (items: TItem[]) => TItem[];
  getItems: (state: TState) => TItem[];
  /** Save items + timestamp to localStorage. */
  save: (items: TItem[], lastFetchedAt: number) => void;
  /**
   * Standard fetch-with-retry lifecycle. Call this from your named fetch
   * action (e.g. `fetchBacklog`, `fetchScenarios`).
   */
  doFetch: (options?: { force?: boolean }) => Promise<void>;
}

/** Configuration required by the factory. */
export interface CachedStoreConfig<TItem, TState extends CachedStoreBase> {
  /** localStorage persistence config. */
  persist: StorePersistConfig;
  /** The async function that fetches the list from the API. */
  fetchFn: () => Promise<TItem[]>;
  /** Sort function applied after every mutation. */
  sortFn: (items: TItem[]) => TItem[];
  /** Error message when fetch fails and there's no cached data. */
  errorMessage: string;
  /** Read the items array from the store state. */
  getItems: (state: TState) => TItem[];
  /** Return a partial state update that sets the items array. */
  setItemsPartial: (items: TItem[]) => Partial<TState>;
  /**
   * Build all store actions (including the named fetch action).
   * The `helpers.doFetch` provides the standard fetch lifecycle.
   */
  actions: (helpers: StoreHelpers<TItem, TState>) => Omit<TState, "status" | "error" | "isRefreshing" | "lastFetchedAt" | "reset"> & Record<string, unknown>;
}

// ---------------------------------------------------------------------------
// Factory
// ---------------------------------------------------------------------------

/**
 * Creates a Zustand store with standardised fetch-with-retry, localStorage
 * caching, and a reset action. Each store defines its own state shape and
 * domain-specific actions (including the named fetch method) via config.
 *
 * Returns `{ useStore, initialState }` so consuming modules can re-export both.
 */
export function createCachedStore<TItem, TState extends CachedStoreBase>(config: CachedStoreConfig<TItem, TState>) {
  const { persist: persistConfig, fetchFn, sortFn, errorMessage, getItems, setItemsPartial, actions } = config;

  const hydrated = loadFromStorage<TItem[]>(persistConfig, []);

  const initialState = {
    ...setItemsPartial(hydrated.data),
    status: (hydrated.data.length > 0 ? "success" : "idle") as LoadStatus,
    error: null as Error | null,
    isRefreshing: false,
    lastFetchedAt: hydrated.lastFetchedAt,
  };

  const useStore = create<TState>((rawSet, rawGet) => {
    const set = rawSet as (partial: Partial<TState> | ((state: TState) => Partial<TState>)) => void;
    const get = rawGet as () => TState;

    const save = (items: TItem[], lastFetchedAt: number) =>
      saveToStorage(persistConfig, items, lastFetchedAt);

    const doFetch = async ({ force = false } = {}): Promise<void> => {
      const state = get();
      const items = getItems(state);
      const { status, lastFetchedAt, isRefreshing } = state;

      if (status === "loading" || isRefreshing) return;
      if (!shouldRefetch({ lastFetchedAt, hasData: items.length > 0, force })) return;

      const hasData = items.length > 0;

      set({
        status: hasData ? "success" : "loading",
        isRefreshing: hasData,
        error: null,
      } as Partial<TState>);

      try {
        const result = await fetchWithRetry(fetchFn);
        const now = Date.now();
        const sorted = sortFn(result);
        set({
          ...setItemsPartial(sorted),
          status: "success",
          error: null,
          isRefreshing: false,
          lastFetchedAt: now,
        } as Partial<TState>);
        save(sorted, now);
      } catch (error) {
        set({
          error: error instanceof Error ? error : new Error(errorMessage),
          status: hasData ? "success" : "error",
          isRefreshing: false,
        } as Partial<TState>);
      }
    };

    const helpers: StoreHelpers<TItem, TState> = {
      set,
      get,
      persistConfig,
      sortFn,
      getItems,
      save,
      doFetch,
    };

    const domainActions = actions(helpers);

    return {
      ...initialState,
      ...domainActions,
      reset: (): void => {
        clearStorage(persistConfig.key);
        set({
          ...setItemsPartial([]),
          status: "idle" as LoadStatus,
          error: null,
          isRefreshing: false,
          lastFetchedAt: null,
        } as Partial<TState>);
      },
    } as TState;
  });

  return { useStore, initialState };
}
