import { useMemo } from 'react';
import { useAppsStore } from '@/state/appsStore';
import { useRecentAppsStore, type RecentAppEntry } from '@/state/recentAppsStore';
import {
  APP_SORT_OPTION_SET,
  DEFAULT_APP_SORT,
  buildAlphabetizedApps,
  buildRecentApps,
  filterAndSortApps,
  type AppSortOption,
} from '@/utils/appCollections';
import { collectAppIdentifiers, normalizeIdentifier } from '@/utils/appPreview';
import type { App } from '@/types';

interface UseAppCatalogOptions {
  search?: string;
  sort?: AppSortOption;
  excludeHistoryIdentifiers?: ReadonlyArray<string>;
  historyLimit?: number;
}

export const useAppCatalog = (options: UseAppCatalogOptions = {}) => {
  const apps = useAppsStore(state => state.apps);
  const recentEntries = useRecentAppsStore(state => state.entries);
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

  const recentApps = useMemo(() => {
    const derived = buildRecentApps(apps, { excludeIdentifiers: excludeHistoryIdentifiers, limit: historyLimit });
    if (derived.length > 0 || recentEntries.length === 0) {
      return derived;
    }

    const excludeSet = new Set(
      excludeHistoryIdentifiers
        .map(value => normalizeIdentifier(value))
        .filter((value): value is string => Boolean(value)),
    );

    const appLookup = new Map<string, App>();
    apps.forEach((app) => {
      collectAppIdentifiers(app).forEach(identifier => {
        if (!appLookup.has(identifier)) {
          appLookup.set(identifier, app);
        }
      });
    });

    const entries = recentEntries
      .filter(entry => {
        const entryKey = resolveRecentEntryKey(entry);
        if (!entryKey) {
          return false;
        }
        return !excludeSet.has(entryKey);
      })
      .sort((a, b) => {
        const aTime = Date.parse(a.last_viewed_at ?? '') || 0;
        const bTime = Date.parse(b.last_viewed_at ?? '') || 0;
        return bTime - aTime;
      })
      .map((entry) => {
        const entryKey = resolveRecentEntryKey(entry);
        if (entryKey && appLookup.has(entryKey)) {
          return mergeRecentEntry(appLookup.get(entryKey) as App, entry);
        }
        return buildFallbackApp(entry);
      })
      .slice(0, historyLimit);

    return entries;
  }, [apps, excludeHistoryIdentifiers, historyLimit, recentEntries]);

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

const resolveRecentEntryKey = (entry: RecentAppEntry): string | null => (
  normalizeIdentifier(entry.id)
  ?? normalizeIdentifier(entry.scenario_name)
  ?? normalizeIdentifier(entry.name)
);

const mergeRecentEntry = (app: App, entry: RecentAppEntry): App => {
  const next = { ...app };
  if (!app.last_viewed_at && entry.last_viewed_at) {
    next.last_viewed_at = entry.last_viewed_at;
  }
  const appViewCount = typeof app.view_count === 'number' ? app.view_count : null;
  if ((appViewCount == null || appViewCount <= 0) && typeof entry.view_count === 'number') {
    next.view_count = entry.view_count;
  }
  if (app.status === 'unknown' && entry.status) {
    next.status = entry.status;
  }
  if (typeof app.completeness_score !== 'number' && typeof entry.completeness_score === 'number') {
    next.completeness_score = entry.completeness_score;
  }
  if (!app.completeness_classification && entry.completeness_classification) {
    next.completeness_classification = entry.completeness_classification;
  }
  return next;
};

const buildFallbackApp = (entry: RecentAppEntry): App => {
  const fallbackName = entry.name ?? entry.scenario_name ?? entry.id;
  const createdAt = entry.last_viewed_at ?? new Date(0).toISOString();
  return {
    id: entry.id,
    name: fallbackName,
    scenario_name: entry.scenario_name ?? fallbackName,
    path: '',
    created_at: createdAt,
    updated_at: createdAt,
    status: entry.status ?? 'unknown',
    port_mappings: {},
    environment: {},
    config: {},
    description: '',
    tags: [],
    uptime: '',
    type: 'scenario',
    view_count: entry.view_count,
    last_viewed_at: entry.last_viewed_at ?? null,
    is_partial: true,
    completeness_score: entry.completeness_score,
    completeness_classification: entry.completeness_classification,
  };
};
