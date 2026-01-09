import { useSyncExternalStore } from 'react';
import { api } from '@/lib/api';
import type { Resource, Scenario } from '@/types/api';

type DiscoveryState = {
  resources: Resource[];
  scenarios: Scenario[];
  loading: boolean;
  initialized: boolean;
  error?: string;
};

type Listener = () => void;

const listeners = new Set<Listener>();

let state: DiscoveryState = {
  resources: [],
  scenarios: [],
  loading: false,
  initialized: false,
  error: undefined,
};

let inFlight: Promise<void> | null = null;

function notify() {
  listeners.forEach(listener => listener());
}

function setState(partial: Partial<DiscoveryState>) {
  state = { ...state, ...partial };
  notify();
}

async function fetchDiscovery() {
  if (inFlight) {
    return inFlight;
  }

  inFlight = (async () => {
    setState({ loading: true, error: undefined });
    try {
      const [resources, scenarios] = await Promise.all([api.getResources(), api.getScenarios()]);
      setState({
        resources: (resources || []).map(resource => ({
          ...resource,
          display_name: resource.display_name || resource.name,
        })),
        scenarios: (scenarios || []).map(scenario => ({
          ...scenario,
          display_name: scenario.display_name || scenario.name,
        })),
        initialized: true,
      });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load discovery data';
      setState({ error: message });
    } finally {
      setState({ loading: false });
      inFlight = null;
    }
  })();

  return inFlight;
}

export function ensureDiscoveryLoaded() {
  return fetchDiscovery();
}

export function refreshDiscovery() {
  return fetchDiscovery();
}

export function useDiscoveryStore<T>(selector: (state: DiscoveryState) => T): T {
  const snapshot = useSyncExternalStore(
    (listener) => {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
    () => state,
    () => state,
  );

  return selector(snapshot);
}
