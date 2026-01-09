// React Query hook for fetching runner breakdown

import { useQuery } from "@tanstack/react-query";
import { fetchRunnerBreakdown, statsQueryKeys } from "../api/statsClient";
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

  return useQuery<RunnerBreakdownResponse, Error>({
    queryKey: statsQueryKeys.runners(filter),
    queryFn: () => fetchRunnerBreakdown(filter),
    enabled: options.enabled ?? true,
    staleTime: options.staleTime ?? 30_000,
    refetchInterval: 60_000,
  });
}
