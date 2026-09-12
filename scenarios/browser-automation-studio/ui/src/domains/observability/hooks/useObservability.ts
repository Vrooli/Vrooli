/**
 * useObservability Hook
 *
 * Fetches observability data from the playwright-driver with automatic polling.
 * Uses React Query for caching and background updates.
 *
 * Transport: Connect-RPC via `observability` client in src/api/observability.ts.
 */

import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback, useMemo } from 'react';
import { observability } from '@/api/observability';
import { logger } from '@/utils/logger';
import type { ObservabilityResponse, ObservabilityDepth } from '../types';

const QUERY_KEY = 'observability';

interface UseObservabilityOptions {
  depth?: ObservabilityDepth;
  refetchInterval?: number | false;
  noCache?: boolean;
  enabled?: boolean;
}

interface UseObservabilityReturn {
  data: ObservabilityResponse | undefined;
  isLoading: boolean;
  isFetching: boolean;
  error: Error | null;
  refetch: () => Promise<void>;
  invalidate: () => Promise<void>;
  dataUpdatedAt: number | undefined;
  isStale: boolean;
}

async function fetchObservability(
  depth: ObservabilityDepth,
  noCache: boolean
): Promise<ObservabilityResponse> {
  return observability.get<ObservabilityResponse>(depth, noCache);
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
    staleTime: 10000,
    gcTime: 60000,
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
