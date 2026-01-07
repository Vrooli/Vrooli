import { useCallback, useMemo, useRef, useState } from 'react';
import { getConfig } from '@/config';
import { logger } from '@/utils/logger';

export interface TabInfo {
  url: string;
  title?: string;
  isActive: boolean;
  order: number;
}

export interface TabsResponse {
  tabs: TabInfo[];
}

export interface UseTabsResult {
  tabs: TabInfo[];
  loading: boolean;
  error: string | null;
  deleting: boolean;
  fetchTabs: (profileId: string) => Promise<void>;
  clear: () => void;
  clearAllTabs: (profileId: string) => Promise<boolean>;
  deleteTab: (profileId: string, order: number) => Promise<boolean>;
}

export function useTabs(): UseTabsResult {
  const [tabs, setTabs] = useState<TabInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const currentProfileId = useRef<string | null>(null);

  const fetchTabs = useCallback(async (profileId: string) => {
    currentProfileId.current = profileId;
    setLoading(true);
    setError(null);
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/sessions/${profileId}/tabs`);
      if (!response.ok) {
        throw new Error(`Failed to fetch tabs (${response.status})`);
      }
      const data = (await response.json()) as TabsResponse;
      setTabs(data.tabs ?? []);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load tabs';
      setError(message);
      logger.error(message, { component: 'useTabs', action: 'fetchTabs' }, err);
    } finally {
      setLoading(false);
    }
  }, []);

  const clear = useCallback(() => {
    setTabs([]);
    setError(null);
    currentProfileId.current = null;
  }, []);

  const clearAllTabs = useCallback(async (profileId: string): Promise<boolean> => {
    setDeleting(true);
    setError(null);
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/sessions/${profileId}/tabs`, {
        method: 'DELETE',
      });
      if (!response.ok) {
        throw new Error(`Clear tabs failed (${response.status})`);
      }
      // Refetch tabs to update UI
      await fetchTabs(profileId);
      return true;
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Clear tabs failed';
      setError(message);
      logger.error(message, { component: 'useTabs', action: 'clearAllTabs' }, err);
      return false;
    } finally {
      setDeleting(false);
    }
  }, [fetchTabs]);

  const deleteTab = useCallback(async (profileId: string, order: number): Promise<boolean> => {
    setDeleting(true);
    setError(null);
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/sessions/${profileId}/tabs/${order}`, {
        method: 'DELETE',
      });
      if (!response.ok) {
        throw new Error(`Delete tab failed (${response.status})`);
      }
      // Refetch tabs to update UI
      await fetchTabs(profileId);
      return true;
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Delete tab failed';
      setError(message);
      logger.error(message, { component: 'useTabs', action: 'deleteTab', order }, err);
      return false;
    } finally {
      setDeleting(false);
    }
  }, [fetchTabs]);

  return useMemo(
    () => ({
      tabs,
      loading,
      error,
      deleting,
      fetchTabs,
      clear,
      clearAllTabs,
      deleteTab,
    }),
    [tabs, loading, error, deleting, fetchTabs, clear, clearAllTabs, deleteTab]
  );
}
