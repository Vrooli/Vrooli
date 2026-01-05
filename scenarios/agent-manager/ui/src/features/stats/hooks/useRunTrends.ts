// React Query hook for fetching time series data (run trends)

import { useQuery } from "@tanstack/react-query";
import { fetchTimeSeries, statsQueryKeys } from "../api/statsClient";
import type { StatsFilter, TimeSeriesResponse } from "../api/types";
import { useTimeWindow } from "./useTimeWindow";

export interface UseRunTrendsOptions {
  filter?: StatsFilter;
  bucket?: string;
  enabled?: boolean;
  staleTime?: number;
}

export function useRunTrends(options: UseRunTrendsOptions = {}) {
  const { filter: defaultFilter } = useTimeWindow();
  const filter = options.filter ?? defaultFilter;

  return useQuery<TimeSeriesResponse, Error>({
    queryKey: statsQueryKeys.timeSeries(filter, options.bucket),
    queryFn: () => fetchTimeSeries(filter, options.bucket),
    enabled: options.enabled ?? true,
    staleTime: options.staleTime ?? 30_000,
    refetchInterval: 60_000,
  });
}
