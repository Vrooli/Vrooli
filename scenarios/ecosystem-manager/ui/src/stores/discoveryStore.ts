import { useSyncExternalStore } from 'react';
import { discoveryClient } from '@/api/discovery';
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
      const [resourcesRes, scenariosRes] = await Promise.all([
        discoveryClient.listResources({}),
        discoveryClient.listScenarios({}),
      ]);
      const resources: Resource[] = resourcesRes.resources.map(r => ({
        name: r.name,
        display_name: r.displayName || r.name,
        description: r.description,
        path: r.path,
        port: r.port,
        category: r.category,
        version: r.version,
        healthy: r.healthy,
        status: r.status,
      }));
      const scenarios: Scenario[] = scenariosRes.scenarios.map(s => ({
        name: s.name,
        display_name: s.displayName || s.name,
        description: s.description,
        path: s.path,
        category: s.category,
        version: s.version,
        status: s.status,
      }));
      setState({ resources, scenarios, initialized: true });
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
