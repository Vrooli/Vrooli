import { useMemo } from 'react';
import { useAppsStore } from '@/state/appsStore';
import {
  APP_SORT_OPTION_SET,
  DEFAULT_APP_SORT,
  buildAlphabetizedApps,
  buildRecentApps,
  filterAndSortApps,
  type AppSortOption,
} from '@/utils/appCollections';

interface UseAppCatalogOptions {
  search?: string;
  sort?: AppSortOption;
  excludeHistoryIdentifiers?: ReadonlyArray<string>;
  historyLimit?: number;
}

export const useAppCatalog = (options: UseAppCatalogOptions = {}) => {
  const apps = useAppsStore(state => state.apps);
  const {
    search = '',
    sort = DEFAULT_APP_SORT,
    excludeHistoryIdentifiers = [],
    historyLimit = 16,
  } = options;

  const filteredApps = useMemo(
    () => filterAndSortApps(apps, { search, sort }),
    [apps, search, sort],
  );

  const recentApps = useMemo(
    () => buildRecentApps(apps, { excludeIdentifiers: excludeHistoryIdentifiers, limit: historyLimit }),
    [apps, excludeHistoryIdentifiers, historyLimit],
  );

  const alphabetizedApps = useMemo(
    () => buildAlphabetizedApps(apps),
    [apps],
  );

  return {
    apps,
    filteredApps,
    recentApps,
    alphabetizedApps,
  };
};

export const normalizeAppSort = (candidate: string | null | undefined): AppSortOption => {
  if (candidate && APP_SORT_OPTION_SET.has(candidate as AppSortOption)) {
    return candidate as AppSortOption;
  }
  return DEFAULT_APP_SORT;
};

export type { AppSortOption } from '@/utils/appCollections';
