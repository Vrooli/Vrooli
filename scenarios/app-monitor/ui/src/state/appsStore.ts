import { logger } from '@/services/logger';
import { create } from 'zustand';
import { appService } from '@/services/api';
import type { App } from '@/types';
import { deriveAppKey } from '@/utils/appPreview';

type AppUpdater = App[] | ((current: App[]) => App[]);

interface AppsStoreState {
  apps: App[];
  loadingInitial: boolean;
  loadingDetailed: boolean;
  error: string | null;
  hasInitialized: boolean;
  lastLoadTimestamp: number | null;
  loadApps: (options?: { force?: boolean }) => Promise<void>;
  setAppsState: (updater: AppUpdater) => void;
  mergeApps: (incoming: App[]) => void;
  updateApp: (update: Partial<App> & { id?: string }) => void;
  clearError: () => void;
}

const CACHE_TTL_MS = 30_000; // 30 seconds

export const useAppsStore = create<AppsStoreState>((set, get) => ({
  apps: [],
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
          set({ apps: detailed, lastLoadTimestamp: Date.now() });
        } else if (Array.isArray(detailed)) {
          // Empty array returned - valid but no apps
          set({ apps: detailed });
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
        set({ apps: summaries, lastLoadTimestamp: Date.now() });
        summariesLoaded = true;
      } else if (Array.isArray(summaries)) {
        // Empty array returned - valid but no apps
        set({ apps: summaries });
        summariesLoaded = true;
      } else if (force) {
        set({ apps: [] });
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
      return { apps: next };
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

      return { apps: Array.from(map.values()) };
    });
  },

  updateApp: (update): void => {
    const keyCandidates: Array<string | undefined> = [update.id, update.scenario_name, update.name];
    const identifier = keyCandidates.find((value) => typeof value === 'string' && value.trim().length > 0)?.trim();
    if (!identifier) {
      return;
    }

    set((state) => {
      let matched = false;
      const next = state.apps.map((app) => {
        const key = deriveAppKey(app);
        if (key !== identifier && app.id !== identifier && app.scenario_name !== identifier) {
          return app;
        }
        matched = true;
        return { ...app, ...update, id: update.id ?? app.id ?? identifier };
      });

      if (!matched) {
        next.push({ id: update.id ?? identifier, ...update } as App);
      }

      return { apps: next };
    });
  },

  clearError: (): void => set({ error: null }),

}));
