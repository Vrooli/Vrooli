import { useCallback, useMemo, useRef, useState } from 'react';
import { getConfig } from '@/config';
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
  // History operations
  clearAllHistory: (profileId: string) => Promise<boolean>;
  deleteHistoryEntry: (profileId: string, entryId: string) => Promise<boolean>;
  updateSettings: (profileId: string, settings: Partial<HistorySettings>) => Promise<boolean>;
  navigateToUrl: (profileId: string, url: string) => Promise<boolean>;
}

export function useHistory(): UseHistoryResult {
  const [history, setHistory] = useState<HistoryResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [navigating, setNavigating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const currentProfileId = useRef<string | null>(null);

  const fetchHistory = useCallback(async (profileId: string) => {
    currentProfileId.current = profileId;
    setLoading(true);
    setError(null);
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/sessions/${profileId}/history`);
      if (!response.ok) {
        throw new Error(`Failed to fetch history (${response.status})`);
      }
      const data = (await response.json()) as HistoryResponse;
      setHistory(data);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load history';
      setError(message);
      logger.error(message, { component: 'useHistory', action: 'fetchHistory' }, err);
    } finally {
      setLoading(false);
    }
  }, []);

  const clear = useCallback(() => {
    setHistory(null);
    setError(null);
    currentProfileId.current = null;
  }, []);

  const clearAllHistory = useCallback(async (profileId: string): Promise<boolean> => {
    setDeleting(true);
    setError(null);
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/sessions/${profileId}/history`, {
        method: 'DELETE',
      });
      if (!response.ok) {
        throw new Error(`Clear history failed (${response.status})`);
      }
      // Refetch history to update UI
      await fetchHistory(profileId);
      return true;
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Clear history failed';
      setError(message);
      logger.error(message, { component: 'useHistory', action: 'clearAllHistory' }, err);
      return false;
    } finally {
      setDeleting(false);
    }
  }, [fetchHistory]);

  const deleteHistoryEntry = useCallback(async (profileId: string, entryId: string): Promise<boolean> => {
    setDeleting(true);
    setError(null);
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/sessions/${profileId}/history/${encodeURIComponent(entryId)}`, {
        method: 'DELETE',
      });
      if (!response.ok) {
        throw new Error(`Delete entry failed (${response.status})`);
      }
      // Refetch history to update UI
      await fetchHistory(profileId);
      return true;
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Delete entry failed';
      setError(message);
      logger.error(message, { component: 'useHistory', action: 'deleteHistoryEntry', entryId }, err);
      return false;
    } finally {
      setDeleting(false);
    }
  }, [fetchHistory]);

  const updateSettings = useCallback(async (profileId: string, settings: Partial<HistorySettings>): Promise<boolean> => {
    setLoading(true);
    setError(null);
    try {
      const config = await getConfig();
      // Merge with existing settings
      const existingSettings = history?.settings ?? {
        maxEntries: 100,
        retentionDays: 30,
        captureThumbnails: true,
      };
      const mergedSettings: HistorySettings = {
        ...existingSettings,
        ...settings,
      };

      const response = await fetch(`${config.API_URL}/recordings/sessions/${profileId}/history/settings`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          maxEntries: mergedSettings.maxEntries,
          retentionDays: mergedSettings.retentionDays,
          captureThumbnails: mergedSettings.captureThumbnails,
        }),
      });
      if (!response.ok) {
        throw new Error(`Update settings failed (${response.status})`);
      }
      // Refetch history to get updated data (settings may have triggered pruning)
      await fetchHistory(profileId);
      return true;
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Update settings failed';
      setError(message);
      logger.error(message, { component: 'useHistory', action: 'updateSettings' }, err);
      return false;
    } finally {
      setLoading(false);
    }
  }, [fetchHistory, history?.settings]);

  const navigateToUrl = useCallback(async (profileId: string, url: string): Promise<boolean> => {
    setNavigating(true);
    setError(null);
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/sessions/${profileId}/history/navigate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url }),
      });
      if (!response.ok) {
        const data = await response.json().catch(() => ({}));
        throw new Error(data.message || `Navigate failed (${response.status})`);
      }
      return true;
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Navigate failed';
      setError(message);
      logger.error(message, { component: 'useHistory', action: 'navigateToUrl', url }, err);
      return false;
    } finally {
      setNavigating(false);
    }
  }, []);

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
