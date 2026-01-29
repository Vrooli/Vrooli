import { create } from "zustand";
import { scenariosService } from "../services";
import type { Scenario } from "../types";
import { fetchWithRetry, shouldRefetch, type LoadStatus } from "./store-utils";

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

export const scenariosStoreInitialState = {
  scenarios: [],
  status: "idle" as LoadStatus,
  error: null,
  isRefreshing: false,
  lastFetchedAt: null,
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
      set({
        scenarios: sortScenarios(result),
        status: "success",
        error: null,
        isRefreshing: false,
        lastFetchedAt: Date.now(),
      });
    } catch (error) {
      set({
        error: error instanceof Error ? error : new Error("Unable to load scenarios."),
        status: hasData ? "success" : "error",
        isRefreshing: false,
      });
    }
  },

  setScenarios: (scenarios: Scenario[]): void => {
    set({ scenarios: sortScenarios(scenarios), status: "success", error: null });
  },

  upsertScenario: (scenario: Scenario): void => {
    set((state) => {
      const next = state.scenarios.some((item) => item.name === scenario.name)
        ? state.scenarios.map((item) => (item.name === scenario.name ? scenario : item))
        : [...state.scenarios, scenario];

      return { scenarios: sortScenarios(next) };
    });
  },

  removeScenario: (name: string): void => {
    set((state) => ({
      scenarios: state.scenarios.filter((scenario) => scenario.name !== name),
    }));
  },

  reset: () => {
    set({ ...scenariosStoreInitialState });
  },
}));
