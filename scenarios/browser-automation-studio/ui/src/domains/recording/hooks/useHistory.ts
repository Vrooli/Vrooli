import { useCallback, useMemo, useState } from 'react';
import { getConfig } from '@/config';
import { logger } from '@/utils/logger';
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

  // Additional state for navigation operations
  const [navigating, setNavigating] = useState(false);

  const deleteHistoryEntry = useCallback(
    async (profileId: string, entryId: string): Promise<boolean> => {
      return base.deleteRequest(profileId, encodeURIComponent(entryId));
    },
    [base.deleteRequest]
  );

  const updateSettings = useCallback(
    async (profileId: string, settings: Partial<HistorySettings>): Promise<boolean> => {
      base.setLoading(true);
      base.setError(null);
      try {
        const config = await getConfig();
        // Merge with existing settings
        const existingSettings = base.data?.settings ?? {
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
        await base.fetch(profileId);
        return true;
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Update settings failed';
        base.setError(message);
        logger.error(message, { component: 'useHistory', action: 'updateSettings' }, err);
        return false;
      } finally {
        base.setLoading(false);
      }
    },
    [base.data?.settings, base.fetch, base.setLoading, base.setError]
  );

  const navigateToUrl = useCallback(
    async (profileId: string, url: string): Promise<boolean> => {
      setNavigating(true);
      base.setError(null);
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
        base.setError(message);
        logger.error(message, { component: 'useHistory', action: 'navigateToUrl', url }, err);
        return false;
      } finally {
        setNavigating(false);
      }
    },
    [base.setError]
  );

  return useMemo(
    () => ({
      history: base.data,
      loading: base.loading,
      error: base.error,
      deleting: base.deleting,
      navigating,
      fetchHistory: base.fetch,
      clear: base.clear,
      clearAllHistory: base.clearAll,
      deleteHistoryEntry,
      updateSettings,
      navigateToUrl,
    }),
    [
      base.data,
      base.loading,
      base.error,
      base.deleting,
      navigating,
      base.fetch,
      base.clear,
      base.clearAll,
      deleteHistoryEntry,
      updateSettings,
      navigateToUrl,
    ]
  );
}
