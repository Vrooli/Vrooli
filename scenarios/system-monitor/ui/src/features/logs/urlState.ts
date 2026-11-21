import { DEFAULT_LIMIT, MAX_LIMIT, type LogQueryFilters, emptyFilters } from './types';

const boundedLimit = (value: string | null): number => {
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed > 0 ? Math.min(MAX_LIMIT, Math.floor(parsed)) : DEFAULT_LIMIT;
};

/** Accepts links from metric/detail surfaces and restores only supported log filters. */
export function filtersFromLogUrl(search: string): LogQueryFilters {
  const params = new URLSearchParams(search);
  const priority = params.get('priority') ?? '';
  return {
    ...emptyFilters,
    units: [...new Set([...params.getAll('unit'), ...params.getAll('units').flatMap(value => value.split(','))].map(value => value.trim()).filter(Boolean))],
    kernel: params.get('kernel') === 'true' || params.get('kernel') === '1',
    since: params.get('since') ?? '',
    until: params.get('until') ?? '',
    priority: /^\d+$/.test(priority) ? priority : '',
    grep: params.get('grep') ?? '',
    boot: params.get('boot') ?? '',
    limit: boundedLimit(params.get('limit')),
  };
}

/** Keeps a Logs link actionable when the operator copies or shares the current address. */
export function logUrlWithFilters(pathname: string, search: string, filters: LogQueryFilters, cursor?: string): string {
  const params = new URLSearchParams(search);
  for (const key of ['unit', 'units', 'kernel', 'since', 'until', 'priority', 'grep', 'boot', 'limit', 'cursor']) params.delete(key);
  filters.units.forEach(unit => params.append('unit', unit));
  if (filters.kernel) params.set('kernel', 'true');
  if (filters.since) params.set('since', filters.since);
  if (filters.until) params.set('until', filters.until);
  if (filters.priority) params.set('priority', filters.priority);
  if (filters.grep) params.set('grep', filters.grep);
  if (filters.boot) params.set('boot', filters.boot);
  if (filters.limit !== DEFAULT_LIMIT) params.set('limit', String(filters.limit));
  if (cursor) params.set('cursor', cursor);
  const query = params.toString();
  return `${pathname}${query ? `?${query}` : ''}`;
}
