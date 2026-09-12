// React Query hook for fetching runner breakdown

import { useQuery } from "@tanstack/react-query";
import { fetchDurableRunnerBreakdown, statsQueryKeys, type DurableRunBreakdown } from "../api/statsClient";
import type { RunnerBreakdownResponse, StatsFilter } from "../api/types";
import { useTimeWindow } from "./useTimeWindow";

export interface UseRunnerPerformanceOptions {
  filter?: StatsFilter;
  enabled?: boolean;
  staleTime?: number;
}

export function useRunnerPerformance(options: UseRunnerPerformanceOptions = {}) {
  const { filter: defaultFilter } = useTimeWindow();
  const filter = options.filter ?? defaultFilter;

  return useQuery<DurableRunBreakdown, Error, RunnerBreakdownResponse & { measure: DurableRunBreakdown }>({
    queryKey: [...statsQueryKeys.runners(filter), "durable"] as const,
    queryFn: () => fetchDurableRunnerBreakdown(filter),
    select: (result) => ({
      runners: result.rows.map((row) => ({
        runnerType: row.value,
        runCount: row.runCount,
        successCount: row.successCount,
        failedCount: row.failedCount,
        totalCostUsd: row.totalCostUsd,
        totalTokens: row.totalTokens,
        totalChargeMicroUsd: row.totalChargeMicroUsd,
        avgDurationMs: row.averageDurationMs,
      })),
      measure: result,
    }),
    enabled: options.enabled ?? true,
    staleTime: options.staleTime ?? 30_000,
    refetchInterval: 60_000,
  });
}
