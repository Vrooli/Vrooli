import { apiFetch } from '../../shared/api/apiFetch';
import {
  type BootsResponse,
  DEFAULT_LIMIT,
  type LogDirection,
  type LogQueryFilters,
  type LogsResponse,
  MAX_LIMIT,
  type UnitsResponse,
} from './types';

export interface FetchLogsArgs {
  filters: LogQueryFilters;
  cursor?: string;
  direction?: LogDirection;
  signal?: AbortSignal;
}

export function buildLogsQueryString(args: FetchLogsArgs): string {
  const { filters, cursor, direction } = args;
  const params = new URLSearchParams();
  for (const u of filters.units) params.append('unit', u);
  if (filters.kernel) params.set('kernel', 'true');
  if (filters.since) params.set('since', filters.since);
  if (filters.until) params.set('until', filters.until);
  if (filters.priority) params.set('priority', filters.priority);
  if (filters.grep) params.set('grep', filters.grep);
  if (filters.boot) params.set('boot', filters.boot);
  const limit = Math.min(Math.max(1, filters.limit || DEFAULT_LIMIT), MAX_LIMIT);
  params.set('limit', String(limit));
  if (cursor) params.set('cursor', cursor);
  if (direction) params.set('direction', direction);
  return params.toString();
}

export function fetchLogs(args: FetchLogsArgs): Promise<LogsResponse> {
  const qs = buildLogsQueryString(args);
  return apiFetch<LogsResponse>(`/logs?${qs}`, { signal: args.signal });
}

export function fetchUnits(signal?: AbortSignal): Promise<UnitsResponse> {
  return apiFetch<UnitsResponse>('/logs/units', { signal });
}

export function fetchBoots(signal?: AbortSignal): Promise<BootsResponse> {
  return apiFetch<BootsResponse>('/logs/boots', { signal });
}
