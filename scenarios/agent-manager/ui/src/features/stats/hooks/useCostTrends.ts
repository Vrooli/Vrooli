// React Query hook for fetching cost stats

import { useQuery } from "@tanstack/react-query";
import { fetchDurableRunCost, statsQueryKeys, type DurableRunCost } from "../api/statsClient";
import type { StatsFilter } from "../api/types";
import { useTimeWindow } from "./useTimeWindow";

export interface UseCostTrendsOptions {
  filter?: StatsFilter;
  enabled?: boolean;
  staleTime?: number;
}

export function useCostTrends(options: UseCostTrendsOptions = {}) {
  const { filter: defaultFilter } = useTimeWindow();
  const filter = options.filter ?? defaultFilter;

  return useQuery<DurableRunCost, Error>({
    queryKey: [...statsQueryKeys.cost(filter), "durable"] as const,
    queryFn: () => fetchDurableRunCost(filter),
    enabled: options.enabled ?? true,
    staleTime: options.staleTime ?? 30_000,
    refetchInterval: 60_000,
  });
}
