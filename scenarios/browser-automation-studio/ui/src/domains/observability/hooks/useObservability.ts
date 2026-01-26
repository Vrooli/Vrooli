/**
 * useObservability Hook
 *
 * Fetches observability data from the playwright-driver with automatic polling.
 * Uses React Query for caching and background updates.
 */

import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback, useMemo } from 'react';
import { getConfig } from '@/config';
import { logger } from '@/utils/logger';
import type { ObservabilityResponse, ObservabilityDepth } from '../types';

const QUERY_KEY = 'observability';

interface UseObservabilityOptions {
  /**
   * Data depth to request
   * - 'quick': Status and summary only (~10ms)
   * - 'standard': Component details + config (~50ms)
   * - 'deep': Full diagnostics (~500ms-5s)
   * @default 'standard'
   */
  depth?: ObservabilityDepth;

  /**
   * Polling interval in milliseconds.
   * Set to 0 or false to disable polling.
   * @default 30000 (30 seconds)
   */
  refetchInterval?: number | false;

  /**
   * Whether to skip cache and fetch fresh data
   * @default false
   */
  noCache?: boolean;

  /**
   * Whether to enable the query
   * @default true
   */
  enabled?: boolean;
}

interface UseObservabilityReturn {
  /** The observability data */
  data: ObservabilityResponse | undefined;

  /** Whether the initial load is in progress */
  isLoading: boolean;

  /** Whether a refetch is in progress */
  isFetching: boolean;

  /** Any error that occurred */
  error: Error | null;

  /** Force a refetch of the data */
  refetch: () => Promise<void>;

  /** Invalidate cached data and refetch */
  invalidate: () => Promise<void>;

  /** Last successful fetch timestamp */
  dataUpdatedAt: number | undefined;

  /** Whether the data is stale */
  isStale: boolean;
}

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

const isString = (value: unknown): value is string => typeof value === 'string';
const isNumber = (value: unknown): value is number => typeof value === 'number';
const isBoolean = (value: unknown): value is boolean => typeof value === 'boolean';

const isObservabilityResponse = (value: unknown): value is ObservabilityResponse => {
  if (!isRecord(value)) return false;
  if (!isString(value.status) || !['ok', 'degraded', 'error'].includes(value.status)) return false;
  if (!isBoolean(value.ready)) return false;
  if (!isString(value.timestamp)) return false;
  if (!isString(value.version)) return false;
  if (!isNumber(value.uptime_ms)) return false;
  if (!isString(value.depth) || !['quick', 'standard', 'deep'].includes(value.depth)) return false;
  if (!isRecord(value.summary)) return false;
  if (!isNumber(value.summary.sessions)) return false;
  if (!isNumber(value.summary.recordings)) return false;
  if (!isBoolean(value.summary.browser_connected)) return false;
  return true;
};

async function fetchObservability(
  depth: ObservabilityDepth,
  noCache: boolean
): Promise<ObservabilityResponse> {
  const config = await getConfig();
  const params = new URLSearchParams();
  params.set('depth', depth);
  if (noCache) {
    params.set('no_cache', 'true');
  }

  const response = await fetch(`${config.API_URL}/observability?${params.toString()}`);

  const payload: unknown = await response.json().catch(() => null);

  if (!response.ok) {
    const message =
      isRecord(payload) && typeof payload.message === 'string'
        ? payload.message
        : `Failed to fetch observability: ${response.statusText}`;
    throw new Error(message);
  }

  if (!isObservabilityResponse(payload)) {
    throw new Error('Invalid observability response');
  }

  return payload;
}

export function useObservability(options: UseObservabilityOptions = {}): UseObservabilityReturn {
  const {
    depth = 'standard',
    refetchInterval = 30000,
    noCache = false,
    enabled = true,
  } = options;

  const queryClient = useQueryClient();

  const queryKey = useMemo(() => [QUERY_KEY, depth], [depth]);

  const query = useQuery({
    queryKey,
    queryFn: () => fetchObservability(depth, noCache),
    enabled,
    refetchInterval: refetchInterval || false,
    staleTime: 10000, // Consider data stale after 10 seconds
    gcTime: 60000, // Keep in cache for 1 minute (formerly cacheTime)
    retry: 1,
    refetchOnWindowFocus: false,
  });

  const refetch = useCallback(async () => {
    try {
      await query.refetch();
    } catch (error) {
      logger.error('Failed to refetch observability data', { component: 'useObservability', action: 'refetch' }, error);
    }
  }, [query]);

  const invalidate = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: [QUERY_KEY] });
  }, [queryClient]);

  return {
    data: query.data,
    isLoading: query.isLoading,
    isFetching: query.isFetching,
    error: query.error,
    refetch,
    invalidate,
    dataUpdatedAt: query.dataUpdatedAt,
    isStale: query.isStale,
  };
}

export default useObservability;
