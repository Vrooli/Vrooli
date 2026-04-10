import { scenariosService } from "../services";
import type { Scenario } from "../types";
import { createCachedStore, type CachedStoreBase } from "./create-cached-store";
import type { LoadStatus, StorePersistConfig } from "./store-utils";

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

interface ScenariosStoreState extends CachedStoreBase {
  scenarios: Scenario[];
  fetchScenarios: (options?: { force?: boolean }) => Promise<void>;
  setScenarios: (scenarios: Scenario[]) => void;
  upsertScenario: (scenario: Scenario) => void;
  removeScenario: (name: string) => void;
}

const { useStore, initialState } = createCachedStore<Scenario, ScenariosStoreState>({
  persist: PERSIST_CONFIG,
  fetchFn: () => scenariosService.list(),
  sortFn: sortScenarios,
  errorMessage: "Unable to load scenarios.",
  getItems: (state) => state.scenarios,
  setItemsPartial: (scenarios) => ({ scenarios }) as Partial<ScenariosStoreState>,
  actions: ({ doFetch, set, get, save, sortFn }) => ({
    scenarios: [] as Scenario[], // placeholder; overridden by initialState spread

    fetchScenarios: (options?: { force?: boolean }) => doFetch(options),

    setScenarios: (scenarios: Scenario[]): void => {
      const sorted = sortFn(scenarios);
      set({ scenarios: sorted, status: "success", error: null } as Partial<ScenariosStoreState>);
      save(sorted, get().lastFetchedAt ?? Date.now());
    },

    upsertScenario: (scenario: Scenario): void => {
      set((state) => {
        const next = state.scenarios.some((item) => item.name === scenario.name)
          ? state.scenarios.map((item) => (item.name === scenario.name ? scenario : item))
          : [...state.scenarios, scenario];

        const sorted = sortFn(next);
        save(sorted, state.lastFetchedAt ?? Date.now());
        return { scenarios: sorted } as Partial<ScenariosStoreState>;
      });
    },

    removeScenario: (name: string): void => {
      set((state) => {
        const next = state.scenarios.filter((scenario) => scenario.name !== name);
        save(next, state.lastFetchedAt ?? Date.now());
        return { scenarios: next } as Partial<ScenariosStoreState>;
      });
    },
  }),
});

export const scenariosStoreInitialState = {
  scenarios: initialState.scenarios as Scenario[],
  status: initialState.status as LoadStatus,
  error: initialState.error,
  isRefreshing: initialState.isRefreshing,
  lastFetchedAt: initialState.lastFetchedAt,
};

export const useScenariosStore = useStore;
