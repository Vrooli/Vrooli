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

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

const isString = (value: unknown): value is string => typeof value === 'string';
const isNumber = (value: unknown): value is number => typeof value === 'number';
const isBoolean = (value: unknown): value is boolean => typeof value === 'boolean';

const isMetricsResponse = (value: unknown): value is MetricsResponse => {
  if (!isRecord(value)) return false;
  if (!isRecord(value.summary)) return false;
  if (!isNumber(value.summary.total_metrics)) return false;
  if (!isString(value.summary.timestamp)) return false;
  if (!isRecord(value.summary.config)) return false;
  if (!isBoolean(value.summary.config.enabled)) return false;
  if (value.summary.config.port !== undefined && !isNumber(value.summary.config.port)) return false;
  if (!isRecord(value.metrics)) return false;
  return true;
};

async function fetchMetrics(): Promise<MetricsResponse> {
  const config = await getConfig();

  const response = await fetch(`${config.API_URL}/observability/metrics`);
  const payload: unknown = await response.json().catch(() => null);

  if (!response.ok) {
    const message =
      isRecord(payload) && typeof payload.message === 'string'
        ? payload.message
        : `Failed to fetch metrics: ${response.statusText}`;
    throw new Error(message);
  }

  if (!isMetricsResponse(payload)) {
    throw new Error('Invalid metrics response');
  }

  return payload;
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
