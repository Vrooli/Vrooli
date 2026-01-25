import { create } from 'zustand';
import { getConfig } from '../config';
import { logger } from '../utils/logger';
import { safeParse, parseArrayFiltered } from '../shared/api/safeParse';
import { ListScenariosResponseSchema, ScenarioSchema } from '../shared/api/schemas';

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

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
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/scenarios`);

      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Failed to load scenarios (${response.status})`);
      }

      const rawData: unknown = await response.json();
      const rawRecord = isRecord(rawData) ? rawData : {};

      // Validate with safeParse, filtering out invalid items
      const result = safeParse(ListScenariosResponseSchema, rawData, 'ListScenarios');
      let mapped: Scenario[];

      if (result.success) {
        // Filter out scenarios without names
        mapped = result.data.scenarios.filter((scenario) => Boolean(scenario.name));
      } else {
        // Fall back to filtered array parsing for partial data recovery
        const items = Array.isArray(rawRecord.scenarios) ? rawRecord.scenarios : [];
        mapped = parseArrayFiltered(ScenarioSchema, items, 'Scenario')
          .filter((scenario) => Boolean(scenario.name));
      }

      set({
        scenarios: mapped,
        isLoading: false,
        lastFetchTime: Date.now(),
        error: mapped.length === 0 ? 'No scenarios found. Install or start a scenario, then refresh.' : null
      });
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unable to load scenarios';
      logger.error('Failed to load scenarios', { component: 'ScenarioStore', action: 'fetchScenarios' }, error);
      set({
        error: message,
        isLoading: false,
        lastFetchTime: Date.now() // Still update time to avoid hammering on errors
      });
    }
  },

  clearError: () => {
    set({ error: null });
  },
}));
