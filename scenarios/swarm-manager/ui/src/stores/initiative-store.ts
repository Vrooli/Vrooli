import { create } from "zustand";
import { initiativeService } from "../services/initiative-service";
import type { InitiativeWithRollup } from "../types";
import { clearStorage, fetchWithRetry, loadFromStorage, saveToStorage, shouldRefetch, type LoadStatus, type StorePersistConfig } from "./store-utils";

const PERSIST_CONFIG: StorePersistConfig = {
  key: "swarm-manager.initiatives.v1",
  version: 1,
  maxItems: 100,
};

interface InitiativeStoreState {
  items: InitiativeWithRollup[];
  status: LoadStatus;
  error: Error | null;
  isRefreshing: boolean;
  lastFetchedAt: number | null;
  fetchInitiatives: (options?: { force?: boolean }) => Promise<void>;
  reset: () => void;
}

const hydrated = loadFromStorage<InitiativeWithRollup[]>(PERSIST_CONFIG, []);

export const initiativeStoreInitialState = {
  items: hydrated.data,
  status: (hydrated.data.length > 0 ? "success" : "idle") as LoadStatus,
  error: null,
  isRefreshing: false,
  lastFetchedAt: hydrated.lastFetchedAt,
};

export const useInitiativeStore = create<InitiativeStoreState>((set, get) => ({
  ...initiativeStoreInitialState,

  fetchInitiatives: async ({ force = false } = {}): Promise<void> => {
    const { status, items, lastFetchedAt, isRefreshing } = get();

    if (status === "loading" || isRefreshing) return;

    if (!shouldRefetch({ lastFetchedAt, hasData: items.length > 0, force })) return;

    const hasData = items.length > 0;
    set({
      status: hasData ? "success" : "loading",
      isRefreshing: hasData,
      error: null,
    });

    try {
      const result = await fetchWithRetry(() => initiativeService.list());
      const now = Date.now();
      set({
        items: result,
        status: "success",
        error: null,
        isRefreshing: false,
        lastFetchedAt: now,
      });
      saveToStorage(PERSIST_CONFIG, result, now);
    } catch (error) {
      set({
        error: error instanceof Error ? error : new Error("Unable to load initiatives."),
        status: hasData ? "success" : "error",
        isRefreshing: false,
      });
    }
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
