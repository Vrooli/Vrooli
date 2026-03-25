import { create } from "zustand";
import { scenariosService } from "../services";
import type { Scenario } from "../types";
import { clearStorage, fetchWithRetry, loadFromStorage, saveToStorage, shouldRefetch, type LoadStatus, type StorePersistConfig } from "./store-utils";

const PERSIST_CONFIG: StorePersistConfig = {
  key: "swarm-manager.scenarios.v1",
  version: 1,
  maxItems: 200,
};

const sortScenarios = (items: Scenario[]): Scenario[] => {
  return [...items].sort((a, b) => {
    if (a.priority !== b.priority) {
      return a.priority - b.priority;
    }
    return a.name.localeCompare(b.name);
  });
};

interface ScenariosStoreState {
  scenarios: Scenario[];
  status: LoadStatus;
  error: Error | null;
  isRefreshing: boolean;
  lastFetchedAt: number | null;
  fetchScenarios: (options?: { force?: boolean }) => Promise<void>;
  setScenarios: (scenarios: Scenario[]) => void;
  upsertScenario: (scenario: Scenario) => void;
  removeScenario: (name: string) => void;
  reset: () => void;
}

const hydrated = loadFromStorage<Scenario[]>(PERSIST_CONFIG, []);

export const scenariosStoreInitialState = {
  scenarios: hydrated.data,
  status: (hydrated.data.length > 0 ? "success" : "idle") as LoadStatus,
  error: null,
  isRefreshing: false,
  lastFetchedAt: hydrated.lastFetchedAt,
};

export const useScenariosStore = create<ScenariosStoreState>((set, get) => ({
  ...scenariosStoreInitialState,

  fetchScenarios: async ({ force = false } = {}): Promise<void> => {
    const { status, scenarios, lastFetchedAt, isRefreshing } = get();

    if (status === "loading" || isRefreshing) {
      return;
    }

    if (!shouldRefetch({ lastFetchedAt, hasData: scenarios.length > 0, force })) {
      return;
    }

    const hasData = scenarios.length > 0;

    set({
      status: hasData ? "success" : "loading",
      isRefreshing: hasData,
      error: null,
    });

    try {
      const result = await fetchWithRetry(() => scenariosService.list());
      const now = Date.now();
      set({
        scenarios: sortScenarios(result),
        status: "success",
        error: null,
        isRefreshing: false,
        lastFetchedAt: now,
      });
      saveToStorage(PERSIST_CONFIG, sortScenarios(result), now);
    } catch (error) {
      set({
        error: error instanceof Error ? error : new Error("Unable to load scenarios."),
        status: hasData ? "success" : "error",
        isRefreshing: false,
      });
    }
  },

  setScenarios: (scenarios: Scenario[]): void => {
    const sorted = sortScenarios(scenarios);
    set({ scenarios: sorted, status: "success", error: null });
    saveToStorage(PERSIST_CONFIG, sorted, get().lastFetchedAt ?? Date.now());
  },

  upsertScenario: (scenario: Scenario): void => {
    set((state) => {
      const next = state.scenarios.some((item) => item.name === scenario.name)
        ? state.scenarios.map((item) => (item.name === scenario.name ? scenario : item))
        : [...state.scenarios, scenario];

      const sorted = sortScenarios(next);
      saveToStorage(PERSIST_CONFIG, sorted, state.lastFetchedAt ?? Date.now());
      return { scenarios: sorted };
    });
  },

  removeScenario: (name: string): void => {
    set((state) => {
      const next = state.scenarios.filter((scenario) => scenario.name !== name);
      saveToStorage(PERSIST_CONFIG, next, state.lastFetchedAt ?? Date.now());
      return { scenarios: next };
    });
  },

  reset: () => {
    clearStorage(PERSIST_CONFIG.key);
    set({
      scenarios: [],
      status: "idle" as LoadStatus,
      error: null,
      isRefreshing: false,
      lastFetchedAt: null,
    });
  },
}));
