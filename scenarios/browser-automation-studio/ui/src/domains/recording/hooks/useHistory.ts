import { useCallback, useMemo, useState } from 'react';
import { ConnectError } from '@connectrpc/connect';
import { recordingsClient } from '@/api/recordings';
import { logger } from '@/utils/logger';
import type { HistoryResponse, HistorySettings } from '../types/types';

export interface UseHistoryResult {
  history: HistoryResponse | null;
  loading: boolean;
  error: string | null;
  deleting: boolean;
  navigating: boolean;
  fetchHistory: (profileId: string) => Promise<void>;
  clear: () => void;
  clearAllHistory: (profileId: string) => Promise<boolean>;
  deleteHistoryEntry: (profileId: string, entryId: string) => Promise<boolean>;
  updateSettings: (profileId: string, settings: Partial<HistorySettings>) => Promise<boolean>;
  navigateToUrl: (profileId: string, url: string) => Promise<boolean>;
}

const COMPONENT = 'useHistory';

function describe(err: unknown): string {
  if (err instanceof ConnectError) return err.message;
  if (err instanceof Error) return err.message;
  return 'History operation failed';
}

const DEFAULT_SETTINGS: HistorySettings = {
  maxEntries: 100,
  retentionDays: 30,
  captureThumbnails: true,
};

export function useHistory(): UseHistoryResult {
  const [history, setHistory] = useState<HistoryResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [navigating, setNavigating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchHistory = useCallback(async (profileId: string) => {
    setLoading(true);
    setError(null);
    try {
      const resp = await recordingsClient.getHistory({ profileId });
      setHistory({
        entries: (resp.entries ?? []).map((e) => ({
          id: e.id,
          url: e.url,
          title: e.title,
          timestamp: e.timestamp,
          thumbnail: e.thumbnail || undefined,
        })),
        settings: resp.settings
          ? {
              maxEntries: resp.settings.maxEntries,
              retentionDays: resp.settings.retentionDays,
              captureThumbnails: resp.settings.captureThumbnails,
            }
          : DEFAULT_SETTINGS,
        stats: {
          totalEntries: resp.stats?.totalEntries ?? 0,
          oldestEntry: resp.stats?.oldestEntry || undefined,
          newestEntry: resp.stats?.newestEntry || undefined,
        },
      });
    } catch (err) {
      const message = describe(err);
      setError(message);
      logger.error(message, { component: COMPONENT, action: 'fetch' }, err);
    } finally {
      setLoading(false);
    }
  }, []);

  const clear = useCallback(() => {
    setHistory(null);
    setError(null);
  }, []);

  const runMutation = useCallback(
    async (profileId: string, op: () => Promise<unknown>, action: string): Promise<boolean> => {
      setDeleting(true);
      setError(null);
      try {
        await op();
        await fetchHistory(profileId);
        return true;
      } catch (err) {
        const message = describe(err);
        setError(message);
        logger.error(message, { component: COMPONENT, action }, err);
        return false;
      } finally {
        setDeleting(false);
      }
    },
    [fetchHistory]
  );

  const clearAllHistory = useCallback(
    (profileId: string) =>
      runMutation(profileId, () => recordingsClient.clearHistory({ profileId }), 'clearAll'),
    [runMutation]
  );

  const deleteHistoryEntry = useCallback(
    (profileId: string, entryId: string) =>
      runMutation(profileId, () => recordingsClient.deleteHistoryEntry({ profileId, entryId }), 'deleteEntry'),
    [runMutation]
  );

  const updateSettings = useCallback(
    async (profileId: string, partial: Partial<HistorySettings>): Promise<boolean> => {
      setLoading(true);
      setError(null);
      const existing = history?.settings ?? DEFAULT_SETTINGS;
      const merged: HistorySettings = { ...existing, ...partial };
      try {
        await recordingsClient.updateHistorySettings({
          profileId,
          settings: {
            maxEntries: merged.maxEntries,
            retentionDays: merged.retentionDays,
            captureThumbnails: merged.captureThumbnails,
          },
        });
        await fetchHistory(profileId);
        return true;
      } catch (err) {
        const message = describe(err);
        setError(message);
        setLoading(false);
        logger.error(message, { component: COMPONENT, action: 'updateSettings' }, err);
        return false;
      }
    },
    [history?.settings, fetchHistory]
  );

  const navigateToUrl = useCallback(
    async (profileId: string, url: string): Promise<boolean> => {
      setNavigating(true);
      setError(null);
      try {
        await recordingsClient.navigateToHistoryURL({ profileId, url });
        return true;
      } catch (err) {
        const message = describe(err);
        setError(message);
        logger.error(message, { component: COMPONENT, action: 'navigateTo', url }, err);
        return false;
      } finally {
        setNavigating(false);
      }
    },
    []
  );

  return useMemo(
    () => ({
      history,
      loading,
      error,
      deleting,
      navigating,
      fetchHistory,
      clear,
      clearAllHistory,
      deleteHistoryEntry,
      updateSettings,
      navigateToUrl,
    }),
    [
      history,
      loading,
      error,
      deleting,
      navigating,
      fetchHistory,
      clear,
      clearAllHistory,
      deleteHistoryEntry,
      updateSettings,
      navigateToUrl,
    ]
  );
}
