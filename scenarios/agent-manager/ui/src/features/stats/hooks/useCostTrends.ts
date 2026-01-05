// React Query hook for fetching cost stats

import { useQuery } from "@tanstack/react-query";
import { fetchCostStats, statsQueryKeys } from "../api/statsClient";
import type { CostResponse, StatsFilter } from "../api/types";
import { useTimeWindow } from "./useTimeWindow";

export interface UseCostTrendsOptions {
  filter?: StatsFilter;
  enabled?: boolean;
  staleTime?: number;
}

export function useCostTrends(options: UseCostTrendsOptions = {}) {
  const { filter: defaultFilter } = useTimeWindow();
  const filter = options.filter ?? defaultFilter;

  return useQuery<CostResponse, Error>({
    queryKey: statsQueryKeys.cost(filter),
    queryFn: () => fetchCostStats(filter),
    enabled: options.enabled ?? true,
    staleTime: options.staleTime ?? 30_000,
    refetchInterval: 60_000,
  });
}
