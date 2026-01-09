// React Query hook for fetching stats summary

import { useQuery } from "@tanstack/react-query";
import { fetchStatsSummary, statsQueryKeys } from "../api/statsClient";
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
    queryKey: statsQueryKeys.summary(filter),
    queryFn: () => fetchStatsSummary(filter),
    enabled: options.enabled ?? true,
    staleTime: options.staleTime ?? 30_000, // 30 seconds
    refetchInterval: 60_000, // Refetch every minute
  });
}
