// React Query hook for fetching model breakdown

import { useQuery } from "@tanstack/react-query";
import { fetchModelBreakdown, statsQueryKeys } from "../api/statsClient";
import type { ModelBreakdownResponse, StatsFilter } from "../api/types";
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
