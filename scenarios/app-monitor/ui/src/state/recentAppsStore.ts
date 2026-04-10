import { create } from 'zustand';
import { createJSONStorage, persist, type StateStorage } from 'zustand/middleware';
import type { App } from '@/types';
import { normalizeIdentifier, resolveAppIdentifier } from '@/utils/appPreview';

export interface RecentAppEntry {
  id: string;
  name?: string;
  scenario_name?: string;
  last_viewed_at?: string | null;
  view_count?: number;
  status?: App['status'] | null;
  completeness_score?: number;
  completeness_classification?: string;
}

type RecentAppInput = Partial<RecentAppEntry>;

interface RecentAppsState {
  entries: RecentAppEntry[];
  recordAppView: (input: RecentAppInput) => void;
  clearRecentApps: () => void;
}

const RECENT_APPS_STORAGE_KEY = 'app-monitor:recent-apps';
const RECENT_APPS_LIMIT = 24;

const noopStorage: StateStorage = {
  getItem: () => null,
  setItem: () => undefined,
  removeItem: () => undefined,
};

const recentAppsStorage = createJSONStorage<Pick<RecentAppsState, 'entries'>>(() => {
  if (typeof window !== 'undefined' && window.localStorage) {
    return window.localStorage;
  }
  return noopStorage;
});

const APP_STATUS_SET = new Set<App['status']>([
  'running',
  'stopped',
  'error',
  'degraded',
  'healthy',
  'unknown',
  'unhealthy',
]);

const normalizeEntryStatus = (value: unknown): App['status'] | null => {
  if (typeof value !== 'string') {
    return null;
  }
  const normalized = value.trim().toLowerCase() as App['status'];
  return APP_STATUS_SET.has(normalized) ? normalized : null;
};

const normalizeRecentEntry = (input: unknown): RecentAppEntry | null => {
  if (!input || typeof input !== 'object') {
    return null;
  }
  const record = input as Partial<RecentAppEntry>;
  const identifier = resolveAppIdentifier({
    id: record.id ?? '',
    name: record.name ?? '',
    scenario_name: record.scenario_name ?? '',
  } as App);
  if (!identifier) {
    return null;
  }
  const viewCount = typeof record.view_count === 'number' && Number.isFinite(record.view_count)
    ? record.view_count
    : undefined;
  const completenessScore = typeof record.completeness_score === 'number' && Number.isFinite(record.completeness_score)
    ? record.completeness_score
    : undefined;
  return {
    id: identifier,
    name: typeof record.name === 'string' ? record.name : undefined,
    scenario_name: typeof record.scenario_name === 'string' ? record.scenario_name : undefined,
    last_viewed_at: typeof record.last_viewed_at === 'string' ? record.last_viewed_at : null,
    view_count: viewCount,
    status: normalizeEntryStatus(record.status),
    completeness_score: completenessScore,
    completeness_classification: typeof record.completeness_classification === 'string'
      ? record.completeness_classification
      : undefined,
  };
};

const resolveEntryKey = (entry: RecentAppEntry): string | null => (
  normalizeIdentifier(entry.id)
  ?? normalizeIdentifier(entry.scenario_name)
  ?? normalizeIdentifier(entry.name)
);

const resolveInputIdentifier = (input: RecentAppInput): string | null => (
  normalizeIdentifier(input.id)
  ?? normalizeIdentifier(input.scenario_name)
  ?? normalizeIdentifier(input.name)
);

export const useRecentAppsStore = create<RecentAppsState>()(persist(
  (set) => ({
    entries: [],

    recordAppView: (input) => {
      const identifier = resolveInputIdentifier(input);
      if (!identifier) {
        return;
      }
      const nowIso = new Date().toISOString();
      set((state) => {
        const existingIndex = state.entries.findIndex((entry) => {
          const entryKey = resolveEntryKey(entry);
          return entryKey === identifier;
        });
        const existing = existingIndex >= 0 ? state.entries[existingIndex] : null;
        const viewCount = typeof input.view_count === 'number' && Number.isFinite(input.view_count)
          ? Math.max(input.view_count, existing?.view_count ?? 0)
          : (existing?.view_count ?? 0) + 1;
        const merged: RecentAppEntry = {
          id: input.id ?? existing?.id ?? identifier,
          name: input.name ?? existing?.name,
          scenario_name: input.scenario_name ?? existing?.scenario_name,
          last_viewed_at: input.last_viewed_at ?? existing?.last_viewed_at ?? nowIso,
          view_count: viewCount,
          status: input.status ?? existing?.status ?? null,
          completeness_score: input.completeness_score ?? existing?.completeness_score,
          completeness_classification: input.completeness_classification ?? existing?.completeness_classification,
        };

        const next = [...state.entries];
        if (existingIndex >= 0) {
          next.splice(existingIndex, 1);
        }
        next.unshift(merged);
        return { entries: next.slice(0, RECENT_APPS_LIMIT) };
      });
    },

    clearRecentApps: () => set({ entries: [] }),
  }),
  {
    name: RECENT_APPS_STORAGE_KEY,
    storage: recentAppsStorage,
    version: 1,
    partialize: (state) => ({ entries: state.entries }),
    migrate: (persisted) => {
      if (!persisted || typeof persisted !== 'object') {
        return { entries: [] };
      }
      const raw = (persisted as Partial<RecentAppsState>).entries;
      if (!Array.isArray(raw)) {
        return { entries: [] };
      }
      const normalized = raw
        .map(normalizeRecentEntry)
        .filter((entry): entry is RecentAppEntry => Boolean(entry));
      return { entries: normalized.slice(0, RECENT_APPS_LIMIT) };
    },
  },
));
