import { create } from 'zustand';
import { logger } from '../utils/logger';
import { scenariosClient } from '../api/scenarios';

export interface Scenario {
  name: string;
  description: string;
  status: string;
}

interface ScenarioState {
  scenarios: Scenario[];
  isLoading: boolean;
  error: string | null;
  lastFetchTime: number | null;

  // Actions
  fetchScenarios: () => Promise<void>;
  clearError: () => void;
}

const CACHE_DURATION_MS = 30000; // 30 seconds

export const useScenarioStore = create<ScenarioState>((set, get) => ({
  scenarios: [],
  isLoading: false,
  error: null,
  lastFetchTime: null,

  fetchScenarios: async () => {
    const { isLoading, lastFetchTime } = get();

    // Avoid redundant fetches if we recently loaded scenarios
    const now = Date.now();
    if (lastFetchTime && now - lastFetchTime < CACHE_DURATION_MS && !isLoading) {
      return;
    }

    // Prevent concurrent fetches
    if (isLoading) {
      return;
    }

    set({ isLoading: true, error: null });

    try {
      const resp = await scenariosClient.list({});
      const mapped: Scenario[] = resp.scenarios
        .filter((s) => Boolean(s.name))
        .map((s) => ({ name: s.name, description: s.description, status: s.status }));

      set({
        scenarios: mapped,
        isLoading: false,
        lastFetchTime: Date.now(),
        error: mapped.length === 0 ? 'No scenarios found. Install or start a scenario, then refresh.' : null,
      });
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unable to load scenarios';
      logger.error('Failed to load scenarios', { component: 'ScenarioStore', action: 'fetchScenarios' }, error);
      set({
        error: message,
        isLoading: false,
        lastFetchTime: Date.now(), // Still update time to avoid hammering on errors
      });
    }
  },

  clearError: () => {
    set({ error: null });
  },
}));
