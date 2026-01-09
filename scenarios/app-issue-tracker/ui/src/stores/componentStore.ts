import { create } from 'zustand';
import type { Component, ComponentsResponse } from '../types/component';
import { buildApiUrl } from '../utils/issues';

interface ComponentState {
  scenarios: Component[];
  resources: Component[];
  allComponents: Component[];
  loading: boolean;
  error: string | null;
  lastFetched: number | null;

  // Actions
  fetchComponents: (apiBaseUrl: string) => Promise<void>;
  reset: () => void;
  getComponentById: (type: 'scenario' | 'resource', id: string) => Component | undefined;
}

const CACHE_DURATION_MS = 5 * 60 * 1000; // 5 minutes

export const useComponentStore = create<ComponentState>((set, get) => ({
  scenarios: [],
  resources: [],
  allComponents: [],
  loading: false,
  error: null,
  lastFetched: null,

  fetchComponents: async (apiBaseUrl: string) => {
    const state = get();

    // Check if cache is still valid
    if (
      state.lastFetched &&
      Date.now() - state.lastFetched < CACHE_DURATION_MS &&
      state.allComponents.length > 0
    ) {
      // Cache is still valid, skip fetch
      return;
    }

    set({ loading: true, error: null });

    try {
      const response = await fetch(buildApiUrl(apiBaseUrl, '/components'));

      if (!response.ok) {
        throw new Error(`Failed to fetch components: ${response.status} ${response.statusText}`);
      }

      const data = await response.json();

      if (!data.success || !data.data?.components) {
        throw new Error('Invalid response format from components API');
      }

      const componentsData = data.data.components as ComponentsResponse;
      const scenarios = componentsData.scenarios || [];
      const resources = componentsData.resources || [];
      const allComponents = [...scenarios, ...resources];

      set({
        scenarios,
        resources,
        allComponents,
        loading: false,
        error: null,
        lastFetched: Date.now(),
      });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to load components';
      console.error('Error fetching components:', error);

      set({
        loading: false,
        error: errorMessage,
      });
    }
  },

  reset: () => {
    set({
      scenarios: [],
      resources: [],
      allComponents: [],
      loading: false,
      error: null,
      lastFetched: null,
    });
  },

  getComponentById: (type: 'scenario' | 'resource', id: string) => {
    const state = get();
    const list = type === 'scenario' ? state.scenarios : state.resources;
    return list.find((comp) => comp.id === id);
  },
}));
