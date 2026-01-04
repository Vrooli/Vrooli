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

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.message || `Failed to fetch observability: ${response.statusText}`);
  }

  return response.json();
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
