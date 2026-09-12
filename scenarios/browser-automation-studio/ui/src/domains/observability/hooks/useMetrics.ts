/**
 * useMetrics Hook
 *
 * Fetches metrics data from the playwright-driver in JSON format.
 * Used for displaying metrics in the diagnostics tab.
 *
 * Transport: Connect-RPC via `observability` client in src/api/observability.ts.
 */

import { useQuery } from '@tanstack/react-query';
import { observability } from '@/api/observability';
import type { MetricsResponse } from '../types';

const QUERY_KEY = 'observability-metrics';

interface UseMetricsOptions {
  enabled?: boolean;
  refetchInterval?: number | false;
}

interface UseMetricsReturn {
  data: MetricsResponse | undefined;
  isLoading: boolean;
  isFetching: boolean;
  error: Error | null;
  refetch: () => Promise<void>;
}

async function fetchMetrics(): Promise<MetricsResponse> {
  return observability.getMetrics<MetricsResponse>();
}

export function useMetrics(options: UseMetricsOptions = {}): UseMetricsReturn {
  const {
    enabled = true,
    refetchInterval = false,
  } = options;

  const query = useQuery({
    queryKey: [QUERY_KEY],
    queryFn: fetchMetrics,
    enabled,
    refetchInterval: refetchInterval || false,
    staleTime: 5000,
    gcTime: 60000,
    retry: 1,
    refetchOnWindowFocus: false,
  });

  const refetch = async () => {
    await query.refetch();
  };

  return {
    data: query.data,
    isLoading: query.isLoading,
    isFetching: query.isFetching,
    error: query.error,
    refetch,
  };
}

export default useMetrics;
