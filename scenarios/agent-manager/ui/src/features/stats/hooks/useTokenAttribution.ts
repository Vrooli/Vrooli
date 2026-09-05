import { useQuery } from "@tanstack/react-query";
import {
  fetchDurableTokenAttribution,
  statsQueryKeys,
  type DurableTokenAttribution,
  type TokenAttributionGroupBy,
  type TokenAttributionView,
} from "../api/statsClient";
import type { StatsFilter } from "../api/types";
import { useTimeWindow } from "./useTimeWindow";

export function useTokenAttribution(options: {
  filter?: StatsFilter;
  groupBy: TokenAttributionGroupBy;
  view: TokenAttributionView;
  limit?: number;
}): ReturnType<typeof useQuery<DurableTokenAttribution, Error>> {
  const { filter: defaultFilter } = useTimeWindow();
  const filter = options.filter ?? defaultFilter;
  const limit = options.limit ?? 20;
  return useQuery({
    queryKey: statsQueryKeys.tokenAttribution(filter, options.groupBy, options.view, limit),
    queryFn: () => fetchDurableTokenAttribution(filter, options.groupBy, options.view, limit),
    staleTime: 30_000,
    refetchInterval: 60_000,
  });
}
