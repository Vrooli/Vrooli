import { useCallback, useMemo } from 'react';
import { createProfileResourceHook } from './useProfileResource';

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

// Create base hook using factory
const useTabsBase = createProfileResourceHook<TabInfo[], TabsResponse>({
  endpoint: 'tabs',
  componentName: 'useTabs',
  transform: (raw) => raw.tabs ?? [],
  initialData: [],
  fetchErrorMessage: 'Failed to load tabs',
  clearAllErrorMessage: 'Clear tabs failed',
});

export function useTabs(): UseTabsResult {
  const base = useTabsBase();
  const { data, loading, error, deleting, fetch, clear, clearAll, deleteRequest } = base;

  const deleteTab = useCallback(
    async (profileId: string, order: number): Promise<boolean> => {
      return deleteRequest(profileId, String(order));
    },
    [deleteRequest]
  );

  return useMemo(
    () => ({
      tabs: data ?? [],
      loading,
      error,
      deleting,
      fetchTabs: fetch,
      clear,
      clearAllTabs: clearAll,
      deleteTab,
    }),
    [data, loading, error, deleting, fetch, clear, clearAll, deleteTab]
  );
}
