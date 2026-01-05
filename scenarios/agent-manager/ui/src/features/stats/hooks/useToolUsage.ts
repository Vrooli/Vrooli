// React Query hook for fetching tool usage stats

import { useQuery } from "@tanstack/react-query";
import { fetchToolUsage, statsQueryKeys } from "../api/statsClient";
import type { StatsFilter, ToolUsageResponse } from "../api/types";
import { useTimeWindow } from "./useTimeWindow";

export interface UseToolUsageOptions {
  filter?: StatsFilter;
  limit?: number;
  enabled?: boolean;
  staleTime?: number;
}

export function useToolUsage(options: UseToolUsageOptions = {}) {
  const { filter: defaultFilter } = useTimeWindow();
  const filter = options.filter ?? defaultFilter;
  const limit = options.limit ?? 20;

  return useQuery<ToolUsageResponse, Error>({
    queryKey: statsQueryKeys.tools(filter, limit),
    queryFn: () => fetchToolUsage(filter, limit),
    enabled: options.enabled ?? true,
    staleTime: options.staleTime ?? 30_000,
    refetchInterval: 60_000,
  });
}
