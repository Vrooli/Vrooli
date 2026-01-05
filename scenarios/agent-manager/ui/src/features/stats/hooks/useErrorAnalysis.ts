// React Query hook for fetching error patterns

import { useQuery } from "@tanstack/react-query";
import { fetchErrorPatterns, statsQueryKeys } from "../api/statsClient";
import type { ErrorPatternsResponse, StatsFilter } from "../api/types";
import { useTimeWindow } from "./useTimeWindow";

export interface UseErrorAnalysisOptions {
  filter?: StatsFilter;
  limit?: number;
  enabled?: boolean;
  staleTime?: number;
}

export function useErrorAnalysis(options: UseErrorAnalysisOptions = {}) {
  const { filter: defaultFilter } = useTimeWindow();
  const filter = options.filter ?? defaultFilter;
  const limit = options.limit ?? 10;

  return useQuery<ErrorPatternsResponse, Error>({
    queryKey: statsQueryKeys.errors(filter, limit),
    queryFn: () => fetchErrorPatterns(filter, limit),
    enabled: options.enabled ?? true,
    staleTime: options.staleTime ?? 30_000,
    refetchInterval: 60_000,
  });
}
