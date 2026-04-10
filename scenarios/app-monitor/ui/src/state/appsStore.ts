import { logger } from '@/services/logger';
import { create } from 'zustand';
import { appService } from '@/services/api';
import type { App } from '@/types';
import { deriveAppKey } from '@/utils/appPreview';

// AI_CHECK: APP_MONITOR_RENDER_PERF=1 | LAST: 2026-02-13
type AppUpdater = App[] | ((current: App[]) => App[]);

interface AppsStoreState {
  apps: App[];
  appIndex: Record<string, number>;
  loadingInitial: boolean;
  loadingDetailed: boolean;
  error: string | null;
  hasInitialized: boolean;
  lastLoadTimestamp: number | null;
  loadApps: (options?: { force?: boolean }) => Promise<void>;
  setAppsState: (updater: AppUpdater) => void;
  mergeApps: (incoming: App[]) => void;
  updateApp: (update: Partial<App> & { id?: string }) => void;
  updateAppsBatch: (updates: Array<Partial<App> & { id?: string }>) => void;
  clearError: () => void;
}

const CACHE_TTL_MS = 30_000; // 30 seconds

const normalizeIdentifier = (value: string | undefined | null): string | null => {
  if (!value) {
    return null;
  }
  const trimmed = value.trim();
  return trimmed.length > 0 ? trimmed : null;
};

const collectIdentifiers = (app: App): string[] => {
  const identifiers = new Set<string>();
  const derived = normalizeIdentifier(deriveAppKey(app));
  if (derived) {
    identifiers.add(derived);
  }
  const id = normalizeIdentifier(app.id);
  if (id) {
    identifiers.add(id);
  }
  const scenarioName = normalizeIdentifier(app.scenario_name);
  if (scenarioName) {
    identifiers.add(scenarioName);
  }
  const name = normalizeIdentifier(app.name);
  if (name) {
    identifiers.add(name);
  }
  return Array.from(identifiers);
};

const buildAppIndex = (apps: App[]): Record<string, number> => {
  const index: Record<string, number> = {};
  apps.forEach((app, appIndex) => {
    collectIdentifiers(app).forEach((identifier) => {
      if (index[identifier] === undefined) {
        index[identifier] = appIndex;
      }
    });
  });
  return index;
};

const getUpdateLookupCandidates = (update: Partial<App> & { id?: string }): string[] => (
  [update.id, update.scenario_name, update.name]
    .map((value) => normalizeIdentifier(value))
    .filter((value): value is string => Boolean(value))
);

const getPreferredUpdateIdentifier = (update: Partial<App> & { id?: string }): string | null => {
  const lookupCandidates = getUpdateLookupCandidates(update);
  return lookupCandidates[0] ?? null;
};

const hasMeaningfulAppUpdate = (existing: App, update: Partial<App> & { id?: string }, identifier: string): boolean => {
  const nextId = update.id ?? existing.id ?? identifier;
  if (nextId !== existing.id) {
    return true;
  }

  for (const [key, value] of Object.entries(update) as Array<[keyof App, App[keyof App]]>) {
    if (typeof value === 'undefined') {
      continue;
    }
    if (!Object.is(existing[key], value)) {
      return true;
    }
  }

  return false;
};

const applyAppUpdates = (
  state: Pick<AppsStoreState, 'apps' | 'appIndex'>,
  updates: Array<Partial<App> & { id?: string }>,
): Pick<AppsStoreState, 'apps' | 'appIndex'> => {
  if (!Array.isArray(updates) || updates.length === 0) {
    return state;
  }

  const nextApps = state.apps.slice();
  const nextIndex = { ...state.appIndex };
  let changed = false;

  updates.forEach((update) => {
    const identifier = getPreferredUpdateIdentifier(update);
    if (!identifier) {
      return;
    }

    const lookupCandidates = getUpdateLookupCandidates(update);
    const matchedIndex = lookupCandidates.reduce<number>((found, candidate) => {
      if (found >= 0) {
        return found;
      }
      const index = nextIndex[candidate];
      return typeof index === 'number' ? index : -1;
    }, -1);

    if (matchedIndex >= 0 && matchedIndex < nextApps.length) {
      const existing = nextApps[matchedIndex];
      if (!existing) {
        return;
      }

      if (!hasMeaningfulAppUpdate(existing, update, identifier)) {
        return;
      }

      const updated = { ...existing, ...update, id: update.id ?? existing.id ?? identifier } as App;
      nextApps[matchedIndex] = updated;
      changed = true;

      const oldIdentifiers = collectIdentifiers(existing);
      const newIdentifierSet = new Set(collectIdentifiers(updated));
      oldIdentifiers.forEach((oldIdentifier) => {
        if (!newIdentifierSet.has(oldIdentifier) && nextIndex[oldIdentifier] === matchedIndex) {
          delete nextIndex[oldIdentifier];
        }
      });
      newIdentifierSet.forEach((newIdentifier) => {
        nextIndex[newIdentifier] = matchedIndex;
      });
      return;
    }

    const created = { id: update.id ?? identifier, ...update } as App;
    const createdIndex = nextApps.length;
    nextApps.push(created);
    collectIdentifiers(created).forEach((candidate) => {
      nextIndex[candidate] = createdIndex;
    });
    changed = true;
  });

  if (!changed) {
    return state;
  }

  return { apps: nextApps, appIndex: nextIndex };
};

export const useAppsStore = create<AppsStoreState>((set, get) => ({
  apps: [],
  appIndex: {},
  loadingInitial: false,
  loadingDetailed: false,
  error: null,
  hasInitialized: false,
  lastLoadTimestamp: null,

  loadApps: async ({ force = false } = {}): Promise<void> => {
    const { loadingInitial, hasInitialized, apps, lastLoadTimestamp } = get();
    if (loadingInitial) {
      return;
    }

    // Check cache validity to prevent stale data issues
    const now = Date.now();
    const isCacheValid = lastLoadTimestamp !== null && (now - lastLoadTimestamp) < CACHE_TTL_MS;

    const runDetailedFetch = async () => {
      if (get().loadingDetailed) {
        return;
      }

      set({ loadingDetailed: true });
      try {
        const detailed = await appService.getApps();
        if (Array.isArray(detailed) && detailed.length > 0) {
          set({ apps: detailed, appIndex: buildAppIndex(detailed), lastLoadTimestamp: Date.now() });
        } else if (Array.isArray(detailed)) {
          // Empty array returned - valid but no apps
          set({ apps: detailed, appIndex: {} });
        }
      } catch (error) {
        logger.warn('[appsStore] Failed to fetch detailed app data', error);
        // Set error if we have no apps at all, otherwise degrade gracefully to summary data
        const currentState = get();
        if (currentState.apps.length === 0) {
          const errorMessage = error instanceof Error ? error.message : 'Unable to load complete scenario data.';
          set({ error: errorMessage });
        }
      } finally {
        set({ loadingDetailed: false });
      }
    };

    // If already initialized with data and cache is valid, just run detailed fetch in background
    if (!force && hasInitialized && apps.length > 0 && isCacheValid) {
      void runDetailedFetch();
      return;
    }

    set({ loadingInitial: true, error: null });

    let summariesLoaded = false;

    try {
      const summaries = await appService.getAppSummaries();
      if (Array.isArray(summaries) && summaries.length > 0) {
        set({ apps: summaries, appIndex: buildAppIndex(summaries), lastLoadTimestamp: Date.now() });
        summariesLoaded = true;
      } else if (Array.isArray(summaries)) {
        // Empty array returned - valid but no apps
        set({ apps: summaries, appIndex: {} });
        summariesLoaded = true;
      } else if (force) {
        set({ apps: [], appIndex: {} });
      }
    } catch (error) {
      logger.warn('[appsStore] Failed to fetch app summaries', error);
      const errorMessage = error instanceof Error ? error.message : 'Unable to load scenario summaries.';
      set({ error: errorMessage });
    } finally {
      set((state) => ({
        loadingInitial: false,
        hasInitialized: state.hasInitialized || summariesLoaded || state.apps.length > 0,
      }));
    }

    await runDetailedFetch();
  },

  setAppsState: (updater): void => {
    set((state) => {
      const next = typeof updater === 'function'
        ? (updater as (current: App[]) => App[])(state.apps)
        : updater;
      return { apps: next, appIndex: buildAppIndex(next) };
    });
  },

  mergeApps: (incoming): void => {
    if (!Array.isArray(incoming) || incoming.length === 0) {
      return;
    }

    set((state) => {
      const map = new Map<string, App>();
      state.apps.forEach((app) => {
        map.set(deriveAppKey(app), app);
      });
      incoming.forEach((app) => {
        const key = deriveAppKey(app);
        const existing = map.get(key) ?? {};
        map.set(key, { ...existing, ...app });
      });

      const next = Array.from(map.values());
      return { apps: next, appIndex: buildAppIndex(next) };
    });
  },

  updateApp: (update): void => {
    set((state) => applyAppUpdates(state, [update]));
  },

  updateAppsBatch: (updates): void => {
    set((state) => applyAppUpdates(state, updates));
  },

  clearError: (): void => set({ error: null }),

}));
