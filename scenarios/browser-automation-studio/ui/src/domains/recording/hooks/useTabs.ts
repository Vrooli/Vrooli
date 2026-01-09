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

  const deleteTab = useCallback(
    async (profileId: string, order: number): Promise<boolean> => {
      return base.deleteRequest(profileId, String(order));
    },
    [base.deleteRequest]
  );

  return useMemo(
    () => ({
      tabs: base.data ?? [],
      loading: base.loading,
      error: base.error,
      deleting: base.deleting,
      fetchTabs: base.fetch,
      clear: base.clear,
      clearAllTabs: base.clearAll,
      deleteTab,
    }),
    [base.data, base.loading, base.error, base.deleting, base.fetch, base.clear, base.clearAll, deleteTab]
  );
}
