/**
 * useStats - React Query hook for fetching stats data.
 *
 * Conditionally fetches when the stats panel is open, with auto-refresh
 * every 60s. Stats are analytical (not transactional), so staleTime is
 * generous at 30s.
 */

import { useQuery } from "@tanstack/react-query";
import { statsService } from "../services";
import type { StatsResponse } from "../types/stats";

export const STATS_QUERY_KEY = ["stats"] as const;

export function useStats(enabled: boolean) {
  return useQuery<StatsResponse>({
    queryKey: STATS_QUERY_KEY,
    queryFn: () => statsService.getStats(),
    enabled,
    staleTime: 30_000,
    gcTime: 5 * 60_000,
    refetchInterval: enabled ? 60_000 : false,
    refetchIntervalInBackground: false,
  });
}
