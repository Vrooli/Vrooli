import { create } from 'zustand';
import { appService } from '@/services/api';
import type { App } from '@/types';

const deriveAppKey = (app: App): string => {
  const candidates: Array<string | undefined> = [app.id, app.scenario_name, app.name];
  const key = candidates.find((value) => typeof value === 'string' && value.trim().length > 0);
  if (key && key.trim().length > 0) {
    return key.trim();
  }

  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }

  return `app-${Math.random().toString(36).slice(2)}`;
};

type AppUpdater = App[] | ((current: App[]) => App[]);

interface AppsStoreState {
  apps: App[];
  loadingInitial: boolean;
  loadingDetailed: boolean;
  error: string | null;
  hasInitialized: boolean;
  loadApps: (options?: { force?: boolean }) => Promise<void>;
  setAppsState: (updater: AppUpdater) => void;
  mergeApps: (incoming: App[]) => void;
  updateApp: (update: Partial<App> & { id?: string }) => void;
  clearError: () => void;
}

export const useAppsStore = create<AppsStoreState>((set, get) => ({
  apps: [],
  loadingInitial: false,
  loadingDetailed: false,
  error: null,
  hasInitialized: false,

  loadApps: async ({ force = false } = {}) => {
    const { loadingInitial, hasInitialized, apps } = get();
    if (loadingInitial) {
      return;
    }

    const runDetailedFetch = async () => {
      if (get().loadingDetailed) {
        return;
      }

      set({ loadingDetailed: true });
      try {
        const detailed = await appService.getApps();
        if (Array.isArray(detailed)) {
          set({ apps: detailed });
        }
      } catch (error) {
        console.warn('[appsStore] Failed to fetch detailed app data', error);
        if (!get().error) {
          set({ error: 'Unable to load complete scenario data.' });
        }
      } finally {
        set({ loadingDetailed: false });
      }
    };

    if (!force && hasInitialized && apps.length > 0) {
      void runDetailedFetch();
      return;
    }

    set({ loadingInitial: true, error: null });

    let summariesLoaded = false;

    try {
      const summaries = await appService.getAppSummaries();
      if (Array.isArray(summaries)) {
        set({ apps: summaries });
        summariesLoaded = true;
      } else if (force) {
        set({ apps: [] });
      }
    } catch (error) {
      console.warn('[appsStore] Failed to fetch app summaries', error);
      if (!get().hasInitialized) {
        set({ error: 'Unable to load scenario summaries.' });
      }
    } finally {
      set((state) => ({
        loadingInitial: false,
        hasInitialized: state.hasInitialized || summariesLoaded || state.apps.length > 0,
      }));
    }

    await runDetailedFetch();
  },

  setAppsState: (updater) => {
    set((state) => {
      const next = typeof updater === 'function'
        ? (updater as (current: App[]) => App[])(state.apps)
        : updater;
      return { apps: next };
    });
  },

  mergeApps: (incoming) => {
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

  updateApp: (update) => {
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

  clearError: () => set({ error: null }),

}));
