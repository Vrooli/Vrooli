import { create } from 'zustand';
import { appService } from '@/services/api';
import { logger } from '@/services/logger';
import type {
  AppProxyMetadata,
  CompleteDiagnostics,
  CompletenessScore,
  LighthouseHistory,
  LocalhostUsageReport,
} from '@/types';

export const APP_INSIGHTS_DEFAULT_STALE_TIME_MS = 30_000;

export type AppInsightsDataset = 'diagnostics' | 'lighthouse' | 'completeness' | 'proxy';

type DatasetState<T> = {
  data: T | null;
  loading: boolean;
  error: string | null;
  fetchedAt: number | null;
};

export type AppInsightsEntry = {
  diagnostics: DatasetState<CompleteDiagnostics>;
  lighthouse: DatasetState<LighthouseHistory>;
  completeness: DatasetState<CompletenessScore>;
  proxy: DatasetState<{
    proxyMetadata: AppProxyMetadata | null;
    localhostReport: LocalhostUsageReport | null;
  }>;
};

type FetchOptions = {
  force?: boolean;
  staleTimeMs?: number;
};

type PrefetchOptions = FetchOptions & {
  datasets?: AppInsightsDataset[];
};

interface AppInsightsStoreState {
  byAppId: Record<string, AppInsightsEntry>;
  prefetch: (appId: string, options?: PrefetchOptions) => Promise<void>;
  fetchDiagnostics: (appId: string, options?: FetchOptions) => Promise<void>;
  fetchLighthouse: (appId: string, options?: FetchOptions) => Promise<void>;
  fetchCompleteness: (appId: string, options?: FetchOptions) => Promise<void>;
  fetchProxy: (appId: string, options?: FetchOptions) => Promise<void>;
  reset: () => void;
}

const inFlightRequests = new Map<string, Promise<void>>();

const createDatasetState = <T>(): DatasetState<T> => ({
  data: null,
  loading: false,
  error: null,
  fetchedAt: null,
});

const createEntry = (): AppInsightsEntry => ({
  diagnostics: createDatasetState<CompleteDiagnostics>(),
  lighthouse: createDatasetState<LighthouseHistory>(),
  completeness: createDatasetState<CompletenessScore>(),
  proxy: createDatasetState<{ proxyMetadata: AppProxyMetadata | null; localhostReport: LocalhostUsageReport | null }>(),
});

const normalizeAppId = (appId: string): string | null => {
  const trimmed = appId.trim();
  return trimmed.length > 0 ? trimmed : null;
};

const resolveStaleTimeMs = (options?: FetchOptions): number => (
  options?.staleTimeMs ?? APP_INSIGHTS_DEFAULT_STALE_TIME_MS
);

const shouldUseCache = <T>(
  dataset: DatasetState<T> | undefined,
  staleTimeMs: number,
  force: boolean,
): boolean => {
  if (force || !dataset?.fetchedAt) {
    return false;
  }
  return (Date.now() - dataset.fetchedAt) < staleTimeMs;
};

const withInFlightDedup = async (key: string, run: () => Promise<void>): Promise<void> => {
  const existing = inFlightRequests.get(key);
  if (existing) {
    await existing;
    return;
  }

  const request = run().finally(() => {
    inFlightRequests.delete(key);
  });
  inFlightRequests.set(key, request);
  await request;
};

const normalizeError = (error: unknown, fallback: string): string => {
  if (error instanceof Error && error.message.trim().length > 0) {
    return error.message;
  }
  return fallback;
};

const upsertEntry = (
  byAppId: Record<string, AppInsightsEntry>,
  appId: string,
  updater: (entry: AppInsightsEntry) => AppInsightsEntry,
): Record<string, AppInsightsEntry> => {
  const current = byAppId[appId] ?? createEntry();
  return {
    ...byAppId,
    [appId]: updater(current),
  };
};

export const useAppInsightsStore = create<AppInsightsStoreState>((set, get) => ({
  byAppId: {},

  prefetch: async (appId, options = {}) => {
    const normalizedAppId = normalizeAppId(appId);
    if (!normalizedAppId) {
      return;
    }

    const datasets = options.datasets ?? ['diagnostics', 'lighthouse', 'completeness', 'proxy'];
    const force = options.force ?? false;
    const staleTimeMs = resolveStaleTimeMs(options);

    const tasks: Array<Promise<void>> = [];
    if (datasets.includes('diagnostics')) {
      tasks.push(get().fetchDiagnostics(normalizedAppId, { force, staleTimeMs }));
    }
    if (datasets.includes('lighthouse')) {
      tasks.push(get().fetchLighthouse(normalizedAppId, { force, staleTimeMs }));
    }
    if (datasets.includes('completeness')) {
      tasks.push(get().fetchCompleteness(normalizedAppId, { force, staleTimeMs }));
    }
    if (datasets.includes('proxy')) {
      tasks.push(get().fetchProxy(normalizedAppId, { force, staleTimeMs }));
    }

    await Promise.all(tasks);
  },

  fetchDiagnostics: async (appId, options = {}) => {
    const normalizedAppId = normalizeAppId(appId);
    if (!normalizedAppId) {
      return;
    }

    const key = `diagnostics:${normalizedAppId}`;
    const force = options.force ?? false;
    const staleTimeMs = resolveStaleTimeMs(options);

    await withInFlightDedup(key, async () => {
      const cached = get().byAppId[normalizedAppId]?.diagnostics;
      if (shouldUseCache(cached, staleTimeMs, force)) {
        return;
      }

      set((state) => ({
        byAppId: upsertEntry(state.byAppId, normalizedAppId, (entry) => ({
          ...entry,
          diagnostics: {
            ...entry.diagnostics,
            loading: true,
            error: null,
          },
        })),
      }));

      try {
        const diagnostics = await appService.getCompleteDiagnostics(normalizedAppId, false);
        set((state) => ({
          byAppId: upsertEntry(state.byAppId, normalizedAppId, (entry) => ({
            ...entry,
            diagnostics: {
              ...entry.diagnostics,
              data: diagnostics,
              loading: false,
              error: null,
              fetchedAt: Date.now(),
            },
          })),
        }));
      } catch (error) {
        logger.warn(`[appInsightsStore] Failed to fetch diagnostics for ${normalizedAppId}`, error);
        const message = normalizeError(error, 'Failed to fetch diagnostics');
        set((state) => ({
          byAppId: upsertEntry(state.byAppId, normalizedAppId, (entry) => ({
            ...entry,
            diagnostics: {
              ...entry.diagnostics,
              loading: false,
              error: message,
            },
          })),
        }));
      }
    });
  },

  fetchLighthouse: async (appId, options = {}) => {
    const normalizedAppId = normalizeAppId(appId);
    if (!normalizedAppId) {
      return;
    }

    const key = `lighthouse:${normalizedAppId}`;
    const force = options.force ?? false;
    const staleTimeMs = resolveStaleTimeMs(options);

    await withInFlightDedup(key, async () => {
      const cached = get().byAppId[normalizedAppId]?.lighthouse;
      if (shouldUseCache(cached, staleTimeMs, force)) {
        return;
      }

      set((state) => ({
        byAppId: upsertEntry(state.byAppId, normalizedAppId, (entry) => ({
          ...entry,
          lighthouse: {
            ...entry.lighthouse,
            loading: true,
            error: null,
          },
        })),
      }));

      try {
        const history = await appService.getLighthouseHistory(normalizedAppId);
        set((state) => ({
          byAppId: upsertEntry(state.byAppId, normalizedAppId, (entry) => ({
            ...entry,
            lighthouse: {
              ...entry.lighthouse,
              data: history,
              loading: false,
              error: null,
              fetchedAt: Date.now(),
            },
          })),
        }));
      } catch (error) {
        logger.warn(`[appInsightsStore] Failed to fetch lighthouse history for ${normalizedAppId}`, error);
        const message = normalizeError(error, 'Failed to fetch Lighthouse history');
        set((state) => ({
          byAppId: upsertEntry(state.byAppId, normalizedAppId, (entry) => ({
            ...entry,
            lighthouse: {
              ...entry.lighthouse,
              loading: false,
              error: message,
            },
          })),
        }));
      }
    });
  },

  fetchCompleteness: async (appId, options = {}) => {
    const normalizedAppId = normalizeAppId(appId);
    if (!normalizedAppId) {
      return;
    }

    const key = `completeness:${normalizedAppId}`;
    const force = options.force ?? false;
    const staleTimeMs = resolveStaleTimeMs(options);

    await withInFlightDedup(key, async () => {
      const cached = get().byAppId[normalizedAppId]?.completeness;
      if (shouldUseCache(cached, staleTimeMs, force)) {
        return;
      }

      set((state) => ({
        byAppId: upsertEntry(state.byAppId, normalizedAppId, (entry) => ({
          ...entry,
          completeness: {
            ...entry.completeness,
            loading: true,
            error: null,
          },
        })),
      }));

      try {
        const completeness = await appService.getAppCompleteness(normalizedAppId);
        set((state) => ({
          byAppId: upsertEntry(state.byAppId, normalizedAppId, (entry) => ({
            ...entry,
            completeness: {
              ...entry.completeness,
              data: completeness,
              loading: false,
              error: null,
              fetchedAt: Date.now(),
            },
          })),
        }));
      } catch (error) {
        logger.warn(`[appInsightsStore] Failed to fetch completeness for ${normalizedAppId}`, error);
        const message = normalizeError(error, 'Failed to fetch completeness');
        set((state) => ({
          byAppId: upsertEntry(state.byAppId, normalizedAppId, (entry) => ({
            ...entry,
            completeness: {
              ...entry.completeness,
              loading: false,
              error: message,
            },
          })),
        }));
      }
    });
  },

  fetchProxy: async (appId, options = {}) => {
    const normalizedAppId = normalizeAppId(appId);
    if (!normalizedAppId) {
      return;
    }

    const key = `proxy:${normalizedAppId}`;
    const force = options.force ?? false;
    const staleTimeMs = resolveStaleTimeMs(options);

    await withInFlightDedup(key, async () => {
      const cached = get().byAppId[normalizedAppId]?.proxy;
      if (shouldUseCache(cached, staleTimeMs, force)) {
        return;
      }

      set((state) => ({
        byAppId: upsertEntry(state.byAppId, normalizedAppId, (entry) => ({
          ...entry,
          proxy: {
            ...entry.proxy,
            loading: true,
            error: null,
          },
        })),
      }));

      try {
        const [proxyMetadata, localhostReport] = await Promise.all([
          appService.getAppProxyMetadata(normalizedAppId),
          appService.getAppLocalhostReport(normalizedAppId),
        ]);

        set((state) => ({
          byAppId: upsertEntry(state.byAppId, normalizedAppId, (entry) => ({
            ...entry,
            proxy: {
              ...entry.proxy,
              data: {
                proxyMetadata,
                localhostReport,
              },
              loading: false,
              error: null,
              fetchedAt: Date.now(),
            },
          })),
        }));
      } catch (error) {
        logger.warn(`[appInsightsStore] Failed to fetch proxy insights for ${normalizedAppId}`, error);
        const message = normalizeError(error, 'Failed to fetch proxy metadata');
        set((state) => ({
          byAppId: upsertEntry(state.byAppId, normalizedAppId, (entry) => ({
            ...entry,
            proxy: {
              ...entry.proxy,
              loading: false,
              error: message,
            },
          })),
        }));
      }
    });
  },

  reset: () => {
    inFlightRequests.clear();
    set({ byAppId: {} });
  },
}));
