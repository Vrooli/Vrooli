// React Query hook for fetching time series data (run trends)

import { useQuery } from "@tanstack/react-query";
import { fetchDurableTerminalTrend, statsQueryKeys, type DurableTerminalTrend } from "../api/statsClient";
import type { StatsFilter, TimeSeriesResponse } from "../api/types";
import { useTimeWindow } from "./useTimeWindow";

export interface UseRunTrendsOptions {
  filter?: StatsFilter;
  bucket?: string;
  enabled?: boolean;
  staleTime?: number;
}

export function useRunTrends(options: UseRunTrendsOptions = {}) {
  const { filter: defaultFilter } = useTimeWindow();
  const filter = options.filter ?? defaultFilter;

  return useQuery<DurableTerminalTrend, Error, TimeSeriesResponse & { measure: DurableTerminalTrend }>({
    queryKey: [...statsQueryKeys.timeSeries(filter, options.bucket), "durable"] as const,
    queryFn: () => fetchDurableTerminalTrend(filter),
    select: (result) => ({
      // The durable series is terminal-time based. `runsStarted` remains zero
      // rather than overstating mutable in-flight lifecycle state.
      buckets: result.rows.map((row) => ({
        timestamp: row.bucket,
        runsStarted: 0,
        runsCompleted: row.completedRuns,
        runsFailed: row.failedRuns,
        runsCancelled: row.cancelledRuns,
        totalCostUsd: row.totalCostUsd,
        avgDurationMs: row.averageDurationMs,
      })),
      bucketDuration: "1h",
      measure: result,
    }),
    enabled: options.enabled ?? true,
    staleTime: options.staleTime ?? 30_000,
    refetchInterval: 60_000,
  });
}
