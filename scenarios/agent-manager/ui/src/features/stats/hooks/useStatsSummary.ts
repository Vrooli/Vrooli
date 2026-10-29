// React Query hook for fetching stats summary

import { useQuery } from "@tanstack/react-query";
import {
  fetchDurableRunCost,
  fetchDurableRunDurationStatistics,
  fetchDurableRunStatusDistribution,
  fetchDurableRunSuccess,
  runStatusCountsFromRows,
  statsQueryKeys,
} from "../api/statsClient";
import type { StatsFilter, SummaryResponse } from "../api/types";
import { useTimeWindow } from "./useTimeWindow";

export interface UseStatsSummaryOptions {
  filter?: StatsFilter;
  enabled?: boolean;
  staleTime?: number;
}

export function useStatsSummary(options: UseStatsSummaryOptions = {}) {
  const { filter: defaultFilter } = useTimeWindow();
  const filter = options.filter ?? defaultFilter;

  return useQuery<SummaryResponse, Error>({
    queryKey: [...statsQueryKeys.summary(filter), "durable"] as const,
    queryFn: async () => {
      const [status, success, duration, cost] = await Promise.all([
        fetchDurableRunStatusDistribution(filter),
        fetchDurableRunSuccess(filter),
        fetchDurableRunDurationStatistics(filter),
        fetchDurableRunCost(filter),
      ]);
      return {
        summary: {
          statusCounts: runStatusCountsFromRows(status.rows),
          successRate: success.rate,
          duration: {
            avgMs: duration.averageDurationMs,
            p50Ms: duration.p50DurationMs,
            p95Ms: duration.p95DurationMs,
            p99Ms: duration.p99DurationMs,
            minMs: duration.minDurationMs,
            maxMs: duration.maxDurationMs,
            count: duration.count,
          },
          cost: {
            totalCostUsd: cost.totalCostUsd,
            avgCostUsd: cost.averageCostUsd,
            inputTokens: cost.inputTokens,
            outputTokens: cost.outputTokens,
            cacheReadTokens: cost.cacheReadTokens,
            totalTokens: cost.totalTokens,
          },
          runnerBreakdown: [],
        },
      };
    },
    enabled: options.enabled ?? true,
    staleTime: options.staleTime ?? 30_000, // 30 seconds
    refetchInterval: 60_000, // Refetch every minute
  });
}
