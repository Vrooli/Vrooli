// React Query hook for fetching profile breakdown

import { useQuery } from "@tanstack/react-query";
import { fetchDurableProfileBreakdown, statsQueryKeys, type DurableRunBreakdown } from "../api/statsClient";
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

  return useQuery<DurableRunBreakdown, Error, ProfileBreakdownResponse>({
    queryKey: [...statsQueryKeys.profiles(filter, limit), "durable"] as const,
    queryFn: () => fetchDurableProfileBreakdown(filter),
    select: (result) => ({
      profiles: result.rows.slice(0, limit).map((row) => ({
        profileId: row.key,
        profileName: row.value,
        runCount: row.runCount,
        successCount: row.successCount,
        failedCount: row.failedCount,
        totalCostUsd: row.totalCostUsd,
      })),
    }),
    enabled: options.enabled ?? true,
    staleTime: options.staleTime ?? 30_000,
    refetchInterval: 60_000,
  });
}
