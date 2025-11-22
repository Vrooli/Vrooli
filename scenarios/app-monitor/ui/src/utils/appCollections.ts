import type { App } from '@/types';
import {
  collectAppIdentifiers,
  parseTimestampValue,
  resolveAppIdentifier,
} from '@/utils/appPreview';

export type AppSortOption =
  | 'status'
  | 'name-asc'
  | 'name-desc'
  | 'recently-viewed'
  | 'least-recently-viewed'
  | 'recently-updated'
  | 'recently-added'
  | 'most-viewed'
  | 'least-viewed'
  | 'completeness-high'
  | 'completeness-low';

export const APP_SORT_OPTIONS: Array<{ value: AppSortOption; label: string }> = [
  { value: 'status', label: 'Active first' },
  { value: 'completeness-high', label: 'Completeness: High → Low' },
  { value: 'completeness-low', label: 'Completeness: Low → High' },
  { value: 'recently-viewed', label: 'Recently viewed' },
  { value: 'least-recently-viewed', label: 'Least recently viewed' },
  { value: 'recently-updated', label: 'Recently updated' },
  { value: 'recently-added', label: 'Recently added' },
  { value: 'most-viewed', label: 'Most viewed' },
  { value: 'least-viewed', label: 'Least viewed' },
  { value: 'name-asc', label: 'A → Z' },
  { value: 'name-desc', label: 'Z → A' },
];

export const APP_SORT_OPTION_SET = new Set<AppSortOption>(APP_SORT_OPTIONS.map(({ value }) => value));
export const DEFAULT_APP_SORT: AppSortOption = 'status';

const STATUS_PRIORITY: Record<string, number> = {
  running: 0,
  healthy: 0,
  starting: 0,
  booting: 0,
  initializing: 0,
  unknown: 0,
  degraded: 1,
  unhealthy: 1,
  warning: 1,
  syncing: 1,
  partial: 1,
  stopping: 2,
  paused: 2,
  error: 2,
  failed: 2,
  crashed: 2,
  offline: 4,
  stopped: 4,
};

const normalizeSearchTerm = (raw: string | undefined): string => raw?.trim().toLowerCase() ?? '';

const getAppLabel = (app: App): string => {
  const label = app.scenario_name || app.name || app.id || resolveAppIdentifier(app) || '';
  return label;
};

export const matchesAppSearch = (app: App, searchTerm: string): boolean => {
  const query = normalizeSearchTerm(searchTerm);
  if (!query) {
    return true;
  }

  const haystacks: string[] = [
    app.name,
    app.id,
    app.status ?? '',
    app.scenario_name ?? '',
    app.description ?? '',
    ...(app.tags ?? []),
  ];

  Object.entries(app.port_mappings ?? {}).forEach(([key, mapping]) => {
    haystacks.push(key);
    if (mapping == null) {
      return;
    }
    if (typeof mapping === 'string' || typeof mapping === 'number') {
      haystacks.push(String(mapping));
      return;
    }
    if (typeof mapping === 'object') {
      const record = mapping as Record<string, unknown>;
      ['label', 'name', 'value', 'port', 'description', 'url'].forEach((field) => {
        const candidate = record[field];
        if (typeof candidate === 'string' || typeof candidate === 'number') {
          haystacks.push(String(candidate));
        }
      });
    }
  });

  haystacks.push(getAppLabel(app));

  return haystacks.some(entry => typeof entry === 'string' && entry.toLowerCase().includes(query));
};

const compareByName = (a: App, b: App): number => {
  const aLabel = getAppLabel(a).toLowerCase();
  const bLabel = getAppLabel(b).toLowerCase();
  if (aLabel && bLabel) {
    const result = aLabel.localeCompare(bLabel);
    if (result !== 0) {
      return result;
    }
  }
  const aId = (a.id ?? '').toLowerCase();
  const bId = (b.id ?? '').toLowerCase();
  return aId.localeCompare(bId);
};

const compareByViewed = (a: App, b: App, direction: 1 | -1): number => {
  const aTime = parseTimestampValue(a.last_viewed_at) ?? 0;
  const bTime = parseTimestampValue(b.last_viewed_at) ?? 0;
  if (aTime !== bTime) {
    return direction * (bTime - aTime);
  }
  const aCount = Number.isFinite(a.view_count) ? Number(a.view_count) : 0;
  const bCount = Number.isFinite(b.view_count) ? Number(b.view_count) : 0;
  if (aCount !== bCount) {
    return direction * (bCount - aCount);
  }
  return compareByName(a, b);
};

const compareByUpdated = (a: App, b: App): number => {
  const aTime = parseTimestampValue(a.updated_at) ?? 0;
  const bTime = parseTimestampValue(b.updated_at) ?? 0;
  if (aTime !== bTime) {
    return bTime - aTime;
  }
  return compareByName(a, b);
};

const compareByCreated = (a: App, b: App): number => {
  const aTime = parseTimestampValue(a.created_at) ?? 0;
  const bTime = parseTimestampValue(b.created_at) ?? 0;
  if (aTime !== bTime) {
    return bTime - aTime;
  }
  return compareByName(a, b);
};

const compareByViews = (a: App, b: App, direction: 1 | -1): number => {
  const aCount = Number.isFinite(a.view_count) ? Number(a.view_count) : 0;
  const bCount = Number.isFinite(b.view_count) ? Number(b.view_count) : 0;
  if (aCount !== bCount) {
    return direction * (bCount - aCount);
  }
  return compareByName(a, b);
};

const compareByStatus = (a: App, b: App): number => {
  const aStatus = (a.status ?? 'unknown').toLowerCase();
  const bStatus = (b.status ?? 'unknown').toLowerCase();
  const aPriority = STATUS_PRIORITY[aStatus] ?? 3;
  const bPriority = STATUS_PRIORITY[bStatus] ?? 3;
  if (aPriority !== bPriority) {
    return aPriority - bPriority;
  }
  if (aStatus !== bStatus) {
    return aStatus.localeCompare(bStatus);
  }
  return compareByViewed(a, b, 1);
};

const compareByCompleteness = (a: App, b: App, direction: 1 | -1): number => {
  const aScore = Number.isFinite(a.completeness_score) ? Number(a.completeness_score) : -1;
  const bScore = Number.isFinite(b.completeness_score) ? Number(b.completeness_score) : -1;

  // Place apps without scores at the end
  if (aScore === -1 && bScore === -1) {
    return compareByName(a, b);
  }
  if (aScore === -1) {
    return 1;
  }
  if (bScore === -1) {
    return -1;
  }

  if (aScore !== bScore) {
    return direction * (bScore - aScore);
  }
  return compareByName(a, b);
};

export const sortApps = (apps: App[], sortOption: AppSortOption): App[] => {
  switch (sortOption) {
    case 'status':
      return [...apps].sort(compareByStatus);
    case 'name-asc':
      return [...apps].sort(compareByName);
    case 'name-desc':
      return [...apps].sort((a, b) => compareByName(b, a));
    case 'recently-viewed':
      return [...apps].sort((a, b) => compareByViewed(a, b, 1));
    case 'least-recently-viewed':
      return [...apps].sort((a, b) => compareByViewed(a, b, -1));
    case 'recently-updated':
      return [...apps].sort(compareByUpdated);
    case 'recently-added':
      return [...apps].sort(compareByCreated);
    case 'most-viewed':
      return [...apps].sort((a, b) => compareByViews(a, b, 1));
    case 'least-viewed':
      return [...apps].sort((a, b) => compareByViews(a, b, -1));
    case 'completeness-high':
      return [...apps].sort((a, b) => compareByCompleteness(a, b, 1));
    case 'completeness-low':
      return [...apps].sort((a, b) => compareByCompleteness(a, b, -1));
    default:
      return apps;
  }
};

export const filterAndSortApps = (
  apps: App[],
  options: { search?: string; sort?: AppSortOption },
): App[] => {
  const { search = '', sort = DEFAULT_APP_SORT } = options;
  const normalized = normalizeSearchTerm(search);
  const filtered = normalized
    ? apps.filter(app => matchesAppSearch(app, normalized))
    : apps;
  return sortApps(filtered, sort);
};

export const buildRecentApps = (
  apps: App[],
  options: {
    excludeIdentifiers?: ReadonlyArray<string>;
    limit?: number;
  } = {},
): App[] => {
  const { excludeIdentifiers = [], limit = 16 } = options;
  if (!apps.length) {
    return [];
  }

  const excludeSet = new Set(
    excludeIdentifiers.map(value => value.trim().toLowerCase()).filter(Boolean),
  );

  return apps
    .filter(app => {
      const lastViewed = parseTimestampValue(app.last_viewed_at);
      const viewCountRaw = Number(app.view_count ?? 0);
      const viewCount = Number.isFinite(viewCountRaw) ? viewCountRaw : 0;
      const hasHistory = lastViewed !== null || viewCount > 0;
      if (!hasHistory) {
        return false;
      }

      const identifiers = collectAppIdentifiers(app);
      return identifiers.every(identifier => !excludeSet.has(identifier.toLowerCase()));
    })
    .sort((a, b) => {
      const aTime = parseTimestampValue(a.last_viewed_at) ?? parseTimestampValue(a.updated_at) ?? 0;
      const bTime = parseTimestampValue(b.last_viewed_at) ?? parseTimestampValue(b.updated_at) ?? 0;
      if (aTime !== bTime) {
        return bTime - aTime;
      }

      const aCountRaw = Number(a.view_count ?? 0);
      const bCountRaw = Number(b.view_count ?? 0);
      const aCount = Number.isFinite(aCountRaw) ? aCountRaw : 0;
      const bCount = Number.isFinite(bCountRaw) ? bCountRaw : 0;
      if (aCount !== bCount) {
        return bCount - aCount;
      }

      return compareByName(a, b);
    })
    .slice(0, limit);
};

export const buildAlphabetizedApps = (apps: App[]): App[] => {
  if (!apps.length) {
    return [];
  }
  return [...apps].sort(compareByName);
};
