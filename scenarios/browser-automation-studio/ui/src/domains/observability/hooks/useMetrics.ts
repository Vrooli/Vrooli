/**
 * useMetrics Hook
 *
 * Fetches metrics data from the playwright-driver in JSON format.
 * Used for displaying metrics in the diagnostics tab.
 */

import { useQuery } from '@tanstack/react-query';
import { getConfig } from '@/config';
import type { MetricsResponse } from '../types';

const QUERY_KEY = 'observability-metrics';

interface UseMetricsOptions {
  /**
   * Whether to enable the query
   * @default true
   */
  enabled?: boolean;

  /**
   * Polling interval in milliseconds.
   * Set to 0 or false to disable polling.
   * @default 0 (no polling)
   */
  refetchInterval?: number | false;
}

interface UseMetricsReturn {
  /** The metrics data */
  data: MetricsResponse | undefined;

  /** Whether the initial load is in progress */
  isLoading: boolean;

  /** Whether a refetch is in progress */
  isFetching: boolean;

  /** Any error that occurred */
  error: Error | null;

  /** Force a refetch of the data */
  refetch: () => Promise<void>;
}

async function fetchMetrics(): Promise<MetricsResponse> {
  const config = await getConfig();

  const response = await fetch(`${config.API_URL}/observability/metrics`);

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.message || `Failed to fetch metrics: ${response.statusText}`);
  }

  return response.json();
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
    staleTime: 5000, // Consider data stale after 5 seconds
    gcTime: 60000, // Keep in cache for 1 minute
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
