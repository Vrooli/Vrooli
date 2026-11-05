import { logger } from '@/services/logger';
import { create } from 'zustand';
import { appService, type ScenarioFixesSummary } from '@/services/api';
import { normalizeIdentifier } from '@/utils/appPreview';
import type { FetchStatus } from '@/components/report/reportTypes';

const CACHE_TTL_MS = 2 * 60 * 1000; // 2 minutes

export type ScenarioIssuesStatus = FetchStatus;

export interface ScenarioIssuesEntry {
  status: ScenarioIssuesStatus;
  fetchedAt: number | null;
  summary: ScenarioFixesSummary | null;
  activeCount: number | null;
  archivedCount: number | null;
  totalCount: number | null;
  stale: boolean;
  error: string | null;
}

interface ScenarioIssuesStoreState {
  entries: Record<string, ScenarioIssuesEntry>;
  fetchIssues: (identifier: string, options?: { force?: boolean }) => Promise<ScenarioIssuesEntry | undefined>;
  flagIssueReported: (identifier: string, timestamp?: number) => void;
}

const baseEntry: ScenarioIssuesEntry = {
  status: 'idle',
  fetchedAt: null,
  summary: null,
  activeCount: null,
  archivedCount: null,
  totalCount: null,
  stale: false,
  error: null,
};

const ensureKey = (identifier: string | null | undefined): string | null => normalizeIdentifier(identifier);

const pendingRequests = new Map<string, Promise<ScenarioIssuesEntry | undefined>>();

const hasFreshEntry = (entry: ScenarioIssuesEntry | undefined, now: number): boolean => {
  if (!entry) {
    return false;
  }
  if (entry.status !== 'ready') {
    return false;
  }
  if (!entry.fetchedAt) {
    return false;
  }
  if (entry.stale) {
    return false;
  }
  return now - entry.fetchedAt < CACHE_TTL_MS;
};

export const useScenarioIssuesStore = create<ScenarioIssuesStoreState>((set, get) => ({
  entries: {},

  fetchIssues: async (identifier, options) => {
    const key = ensureKey(identifier);
    if (!key) {
      return undefined;
    }

    const now = Date.now();
    const currentEntry = get().entries[key];
    const force = options?.force ?? false;
    if (!force && hasFreshEntry(currentEntry, now)) {
      return currentEntry;
    }

    if (!force && pendingRequests.has(key)) {
      return pendingRequests.get(key);
    }

    const request = (async (): Promise<ScenarioIssuesEntry | undefined> => {
      set((state) => ({
        entries: {
          ...state.entries,
          [key]: {
            ...(state.entries[key] ?? baseEntry),
            status: 'loading',
            error: null,
          },
        },
      }));

      try {
        const summary = await appService.getScenarioFixes(identifier);
        const active = Array.isArray(summary?.active) ? summary.active : [];
        const archived = Array.isArray(summary?.archived) ? summary.archived : [];
        const activeCount = typeof summary?.active_count === 'number'
          ? summary?.active_count
          : active.length;
        const archivedCount = typeof summary?.archived_count === 'number'
          ? summary.archived_count
          : archived.length;
        const totalCount = typeof summary?.total_count === 'number' ? summary?.total_count : activeCount + archivedCount;

        const nextEntry: ScenarioIssuesEntry = {
          status: 'ready',
          fetchedAt: Date.now(),
          summary: summary ?? null,
          activeCount,
          archivedCount,
          totalCount,
          stale: Boolean(summary?.stale),
          error: null,
        };

        set((state) => ({
          entries: {
            ...state.entries,
            [key]: nextEntry,
          },
        }));

        return nextEntry;
      } catch (error) {
        logger.warn('[scenarioIssuesStore] Failed to fetch fixes', error);
        const nextEntry: ScenarioIssuesEntry = {
          status: 'error',
          fetchedAt: Date.now(),
          summary: null,
          activeCount: null,
          archivedCount: null,
          totalCount: null,
          stale: false,
          error: (error as { message?: string })?.message ?? 'Failed to load fix status.',
        };

        set((state) => ({
          entries: {
            ...state.entries,
            [key]: nextEntry,
          },
        }));

        return nextEntry;
      } finally {
        pendingRequests.delete(key);
      }
    })();

    pendingRequests.set(key, request);
    return request;
  },

  flagIssueReported: (identifier, timestamp) => {
    const key = ensureKey(identifier);
    if (!key) {
      return;
    }

    const resolvedTimestamp = timestamp ?? Date.now();

    set((state) => {
      const existing = state.entries[key] ?? baseEntry;
      const nextEntry: ScenarioIssuesEntry = {
        ...existing,
        status: 'ready',
        fetchedAt: existing.fetchedAt ?? resolvedTimestamp,
        summary: existing.summary,
        activeCount: Math.max(existing.activeCount ?? 0, 1),
        archivedCount: existing.archivedCount ?? 0,
        totalCount: existing.totalCount != null ? Math.max(existing.totalCount, 1) : 1,
        stale: true,
        error: null,
      };

      return {
        entries: {
          ...state.entries,
          [key]: nextEntry,
        },
      };
    });
  },
}));

export const scenarioIssuesGetState = () => useScenarioIssuesStore.getState();
export type ScenarioIssuesStateSnapshot = ReturnType<typeof scenarioIssuesGetState>;
