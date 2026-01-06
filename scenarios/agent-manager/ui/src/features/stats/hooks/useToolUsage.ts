// React Query hook for fetching tool usage stats

import { useQuery } from "@tanstack/react-query";
import { fetchToolUsage, fetchToolUsageModels, fetchToolUsageRuns, statsQueryKeys } from "../api/statsClient";
import type { StatsFilter, ToolUsageModelsResponse, ToolUsageResponse, ToolUsageRunsResponse } from "../api/types";
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

export interface UseToolUsageRunsOptions {
  filter?: StatsFilter;
  toolName?: string;
  limit?: number;
  enabled?: boolean;
  staleTime?: number;
}

export function useToolUsageRuns(options: UseToolUsageRunsOptions = {}) {
  const { filter: defaultFilter } = useTimeWindow();
  const filter = options.filter ?? defaultFilter;
  const limit = options.limit ?? 25;
  const toolName = options.toolName ?? "";

  return useQuery<ToolUsageRunsResponse, Error>({
    queryKey: statsQueryKeys.toolRuns(filter, toolName, limit),
    queryFn: () => fetchToolUsageRuns(filter, toolName, limit),
    enabled: options.enabled ?? !!toolName,
    staleTime: options.staleTime ?? 30_000,
    refetchInterval: 60_000,
  });
}

export interface UseToolUsageModelsOptions {
  filter?: StatsFilter;
  toolName?: string;
  limit?: number;
  enabled?: boolean;
  staleTime?: number;
}

export function useToolUsageModels(options: UseToolUsageModelsOptions = {}) {
  const { filter: defaultFilter } = useTimeWindow();
  const filter = options.filter ?? defaultFilter;
  const limit = options.limit ?? 25;
  const toolName = options.toolName ?? "";

  return useQuery<ToolUsageModelsResponse, Error>({
    queryKey: statsQueryKeys.toolModels(filter, toolName, limit),
    queryFn: () => fetchToolUsageModels(filter, toolName, limit),
    enabled: options.enabled ?? !!toolName,
    staleTime: options.staleTime ?? 30_000,
    refetchInterval: 60_000,
  });
}
