import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { logger } from '@/utils/logger';
import { recordingApi } from '../api';
import { createProfileResourceHook } from './useProfileResource';
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

  // AbortController for request cancellation
  const abortControllerRef = useRef<AbortController | null>(null);

  // Clean up on unmount
  useEffect(() => {
    return () => {
      abortControllerRef.current?.abort();
    };
  }, []);

  // Reset abort controller
  useEffect(() => {
    abortControllerRef.current = new AbortController();
  }, []);

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

      const result = await recordingApi.updateHistorySettings(
        profileId,
        {
          maxEntries: mergedSettings.maxEntries,
          retentionDays: mergedSettings.retentionDays,
          captureThumbnails: mergedSettings.captureThumbnails,
        },
        { signal: abortControllerRef.current?.signal }
      );

      if (!result.success) {
        setError(result.error);
        setLoading(false);
        logger.error(result.error, { component: 'useHistory', action: 'updateSettings' });
        return false;
      }

      // Refetch history to get updated data (settings may have triggered pruning)
      await fetchHistory(profileId);
      setLoading(false);
      return true;
    },
    [data?.settings, fetchHistory, setLoading, setError]
  );

  const navigateToUrl = useCallback(
    async (profileId: string, url: string): Promise<boolean> => {
      setNavigating(true);
      setError(null);

      const result = await recordingApi.navigateToHistoryUrl(profileId, url, {
        signal: abortControllerRef.current?.signal,
      });

      setNavigating(false);

      if (!result.success) {
        setError(result.error);
        logger.error(result.error, { component: 'useHistory', action: 'navigateToUrl', url });
        return false;
      }

      return true;
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
