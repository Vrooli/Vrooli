// React Query hook for fetching profile breakdown

import { useQuery } from "@tanstack/react-query";
import { fetchProfileBreakdown, statsQueryKeys } from "../api/statsClient";
import type { ProfileBreakdownResponse, StatsFilter } from "../api/types";
import { useTimeWindow } from "./useTimeWindow";

export interface UseProfileBreakdownOptions {
  filter?: StatsFilter;
  limit?: number;
  enabled?: boolean;
  staleTime?: number;
}

export function useProfileBreakdown(options: UseProfileBreakdownOptions = {}) {
  const { filter: defaultFilter } = useTimeWindow();
  const filter = options.filter ?? defaultFilter;
  const limit = options.limit ?? 10;

  return useQuery<ProfileBreakdownResponse, Error>({
    queryKey: statsQueryKeys.profiles(filter, limit),
    queryFn: () => fetchProfileBreakdown(filter, limit),
    enabled: options.enabled ?? true,
    staleTime: options.staleTime ?? 30_000,
    refetchInterval: 60_000,
  });
}
