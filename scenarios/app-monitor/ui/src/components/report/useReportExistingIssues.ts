/**
 * Hook for managing existing issues state in the report dialog.
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
  issues: Array<{
    id?: string;
    title?: string;
    status?: string;
    issue_url?: string;
  }>;
  openCount: number;
  activeCount: number;
  totalCount: number;
  trackerUrl: string | null;
  lastFetched: string | null;
  stale: boolean;
  fromCache: boolean;
  error: string | null;
  refresh: () => void;
}

/**
 * Hook for accessing existing issues from the scenario issues store.
 * Automatically fetches issues when the dialog opens.
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
        issues: [],
        openCount: 0,
        activeCount: 0,
        totalCount: 0,
        trackerUrl: null,
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
      issues: storeEntry.summary?.issues ?? [],
      openCount: storeEntry.openCount ?? 0,
      activeCount: storeEntry.activeCount ?? 0,
      totalCount: storeEntry.totalCount ?? 0,
      trackerUrl: storeEntry.summary?.tracker_url ?? null,
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
