// React Query hook for fetching error patterns

import { useQuery } from "@tanstack/react-query";
import { fetchDurableErrorPatterns, statsQueryKeys, type ErrorPatternsMeasure, type MeasureWindow } from "../api/statsClient";
import type { StatsFilter } from "../api/types";
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
  const now = new Date();
  const hours = filter.preset === "6h" ? 6 : filter.preset === "12h" ? 12 : filter.preset === "7d" ? 24 * 7 : filter.preset === "30d" ? 24 * 30 : 24;
  const window: MeasureWindow = { window: { custom: { from: new Date(now.getTime() - hours * 60 * 60 * 1000).toISOString(), to: now.toISOString() } } };

  return useQuery<ErrorPatternsMeasure, Error>({
    queryKey: [...statsQueryKeys.errors(filter, limit), "durable"] as const,
    queryFn: () => fetchDurableErrorPatterns(window),
    enabled: options.enabled ?? true,
    staleTime: options.staleTime ?? 30_000,
    refetchInterval: 60_000,
  });
}
