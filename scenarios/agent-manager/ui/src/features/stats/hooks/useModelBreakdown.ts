// React Query hook for fetching model breakdown

import { useQuery } from "@tanstack/react-query";
import { fetchDurableModelBreakdown, fetchDurableModelCohort, statsQueryKeys, type DurableCohort, type DurableRunBreakdown } from "../api/statsClient";
import type { ModelBreakdownResponse, ModelUsageRunsResponse, StatsFilter } from "../api/types";
import { useTimeWindow } from "./useTimeWindow";

export interface UseModelBreakdownOptions {
  filter?: StatsFilter;
  limit?: number;
  enabled?: boolean;
  staleTime?: number;
}

export function useModelBreakdown(options: UseModelBreakdownOptions = {}) {
  const { filter: defaultFilter } = useTimeWindow();
  const filter = options.filter ?? defaultFilter;
  const limit = options.limit ?? 10;

  return useQuery<DurableRunBreakdown, Error, ModelBreakdownResponse>({
    queryKey: [...statsQueryKeys.models(filter, limit), "durable"] as const,
    queryFn: () => fetchDurableModelBreakdown(filter),
    select: (result) => ({
      models: result.rows.slice(0, limit).map((row) => ({
        model: row.value,
        runCount: row.runCount,
        successCount: row.successCount,
        failedCount: row.failedCount,
        totalCostUsd: row.totalCostUsd,
        totalTokens: row.totalTokens,
      })),
    }),
    enabled: options.enabled ?? true,
    staleTime: options.staleTime ?? 30_000,
    refetchInterval: 60_000,
  });
}

export interface UseModelUsageRunsOptions {
  filter?: StatsFilter;
  model?: string;
  limit?: number;
  enabled?: boolean;
  staleTime?: number;
}

export function useModelUsageRuns(options: UseModelUsageRunsOptions = {}) {
  const { filter: defaultFilter } = useTimeWindow();
  const baseFilter = options.filter ?? defaultFilter;
  const limit = options.limit ?? 25;
  const model = options.model ?? "";
  const filter = model ? { ...baseFilter, model } : baseFilter;

  return useQuery<DurableCohort, Error, ModelUsageRunsResponse>({
    queryKey: [...statsQueryKeys.modelRuns(filter, limit), "durable"] as const,
    queryFn: () => fetchDurableModelCohort(baseFilter, model, limit),
    select: (result) => ({ runs: result.runIds.map((runId) => ({ runId, taskId: "", taskTitle: "Run", profileName: "Durable cohort", createdAt: "", status: "unknown", totalCostUsd: 0, totalTokens: 0 })) }),
    enabled: options.enabled ?? !!model,
    staleTime: options.staleTime ?? 30_000,
    refetchInterval: 60_000,
  });
}
