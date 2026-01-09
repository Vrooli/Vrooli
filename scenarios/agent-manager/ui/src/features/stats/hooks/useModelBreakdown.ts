// React Query hook for fetching model breakdown

import { useQuery } from "@tanstack/react-query";
import { fetchModelBreakdown, fetchModelUsageRuns, statsQueryKeys } from "../api/statsClient";
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

  return useQuery<ModelBreakdownResponse, Error>({
    queryKey: statsQueryKeys.models(filter, limit),
    queryFn: () => fetchModelBreakdown(filter, limit),
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

  return useQuery<ModelUsageRunsResponse, Error>({
    queryKey: statsQueryKeys.modelRuns(filter, limit),
    queryFn: () => fetchModelUsageRuns(filter, limit),
    enabled: options.enabled ?? !!model,
    staleTime: options.staleTime ?? 30_000,
    refetchInterval: 60_000,
  });
}
