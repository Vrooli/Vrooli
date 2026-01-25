import { useCallback, useMemo, useState } from 'react';
import { getConfig } from '@/config';
import { logger } from '@/utils/logger';
import { createProfileResourceHook } from './useProfileResource';
import type { HistoryResponse, HistorySettings } from '../types/types';

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

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

// Create base hook using factory
const useHistoryBase = createProfileResourceHook<HistoryResponse>({
  endpoint: 'history',
  componentName: 'useHistory',
  fetchErrorMessage: 'Failed to load history',
  clearAllErrorMessage: 'Clear history failed',
});

export function useHistory(): UseHistoryResult {
  const base = useHistoryBase();
  const {
    data,
    loading,
    error,
    deleting,
    fetch: fetchHistory,
    clear,
    clearAll,
    deleteRequest,
    setLoading,
    setError,
  } = base;

  // Additional state for navigation operations
  const [navigating, setNavigating] = useState(false);

  const deleteHistoryEntry = useCallback(
    async (profileId: string, entryId: string): Promise<boolean> => {
      return deleteRequest(profileId, encodeURIComponent(entryId));
    },
    [deleteRequest]
  );

  const updateSettings = useCallback(
    async (profileId: string, settings: Partial<HistorySettings>): Promise<boolean> => {
      setLoading(true);
      setError(null);
      try {
        const config = await getConfig();
        // Merge with existing settings
        const existingSettings = data?.settings ?? {
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
    },
    [data?.settings, fetchHistory, setLoading, setError]
  );

  const navigateToUrl = useCallback(
    async (profileId: string, url: string): Promise<boolean> => {
      setNavigating(true);
      setError(null);
      try {
        const config = await getConfig();
        const response = await window.fetch(`${config.API_URL}/recordings/sessions/${profileId}/history/navigate`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ url }),
        });
        if (!response.ok) {
          const payloadText = await response.text();
          let payload: unknown = null;
          try {
            payload = JSON.parse(payloadText);
          } catch {
            payload = null;
          }
          const message =
            isRecord(payload) && typeof payload.message === 'string'
              ? payload.message
              : `Navigate failed (${response.status})`;
          throw new Error(message);
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
    },
    [setError]
  );

  return useMemo(
    () => ({
      history: data,
      loading,
      error,
      deleting,
      navigating,
      fetchHistory: fetchHistory,
      clear,
      clearAllHistory: clearAll,
      deleteHistoryEntry,
      updateSettings,
      navigateToUrl,
    }),
    [
      data,
      loading,
      error,
      deleting,
      navigating,
      fetchHistory,
      clear,
      clearAll,
      deleteHistoryEntry,
      updateSettings,
      navigateToUrl,
    ]
  );
}
