/**
 * Hook for managing existing fix backlog state in the report dialog.
 * Wraps the scenarioIssuesStore to provide a consistent interface.
 */

import { useEffect, useMemo } from 'react';
import type { App } from '@/types';
import { useScenarioIssuesStore } from '@/state/scenarioIssuesStore';

interface UseReportExistingIssuesParams {
  app: App | null;
  appId?: string;
  isOpen: boolean;
}

export interface ReportExistingIssuesState {
  status: 'idle' | 'loading' | 'ready' | 'error';
  active: Array<{
    id?: string;
    kind?: string;
    name?: string;
    title?: string;
    status?: string;
    url?: string;
  }>;
  archived: Array<{
    id?: string;
    kind?: string;
    name?: string;
    title?: string;
    status?: string;
    url?: string;
  }>;
  activeCount: number;
  archivedCount: number;
  totalCount: number;
  swarmUrl: string | null;
  lastFetched: string | null;
  stale: boolean;
  fromCache: boolean;
  error: string | null;
  refresh: () => void;
}

/**
 * Hook for accessing existing Swarm Manager fixes from the scenario store.
 * Automatically fetches fixes when the dialog opens.
 */
export function useReportExistingIssues({
  app,
  appId,
  isOpen,
}: UseReportExistingIssuesParams): ReportExistingIssuesState {
  const fetchIssues = useScenarioIssuesStore(state => state.fetchIssues);

  const resolvedAppId = useMemo(() => {
    const candidate = (app?.id ?? appId ?? '').trim();
    return candidate === '' ? null : candidate;
  }, [app?.id, appId]);

  // Auto-fetch when dialog opens
  useEffect(() => {
    if (!isOpen || !resolvedAppId) {
      return;
    }
    void fetchIssues(resolvedAppId);
  }, [isOpen, resolvedAppId, fetchIssues]);

  // Get current state from store
  const storeEntry = useScenarioIssuesStore(state =>
    resolvedAppId ? state.entries[resolvedAppId] : null
  );

  // Build return state
  const state: ReportExistingIssuesState = useMemo(() => {
    if (!storeEntry) {
      return {
        status: 'idle',
        active: [],
        archived: [],
        activeCount: 0,
        archivedCount: 0,
        totalCount: 0,
        swarmUrl: null,
        lastFetched: null,
        stale: false,
        fromCache: false,
        error: null,
        refresh: () => {
          if (resolvedAppId) {
            void fetchIssues(resolvedAppId, { force: true });
          }
        },
      };
    }

    return {
      status: storeEntry.status,
      active: storeEntry.summary?.active ?? [],
      archived: storeEntry.summary?.archived ?? [],
      activeCount: storeEntry.activeCount ?? 0,
      archivedCount: storeEntry.archivedCount ?? 0,
      totalCount: storeEntry.totalCount ?? 0,
      swarmUrl: storeEntry.summary?.swarm_url ?? null,
      lastFetched: storeEntry.summary?.last_fetched ?? null,
      stale: storeEntry.stale,
      fromCache: storeEntry.summary?.from_cache ?? false,
      error: storeEntry.error,
      refresh: () => {
        if (resolvedAppId) {
          void fetchIssues(resolvedAppId, { force: true });
        }
      },
    };
  }, [storeEntry, resolvedAppId, fetchIssues]);

  return state;
}
