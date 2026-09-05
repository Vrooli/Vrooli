// React Query hook for fetching tool usage stats

import { useQuery } from "@tanstack/react-query";
import { fetchDurableToolCohort, fetchDurableToolModels, fetchDurableToolUsage, fetchDurableToolCommands, statsQueryKeys, type DurableCohort, type DurableRunBreakdown, type DurableToolUsage, type DurableToolCommandBreakdown } from "../api/statsClient";
import type { StatsFilter, ToolUsageModelsResponse, ToolUsageResponse, ToolUsageRunsResponse, ToolCommandBreakdownResponse } from "../api/types";
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

  return useQuery<DurableToolUsage, Error, ToolUsageResponse & { measure: DurableToolUsage }>({
    queryKey: [...statsQueryKeys.tools(filter, limit), "durable"] as const,
    queryFn: () => fetchDurableToolUsage(filter),
    select: (result) => ({ tools: result.rows.slice(0, limit).map((row) => ({ toolName: row.toolName, callCount: row.callCount, successCount: row.successCount, failedCount: row.failedCount })), measure: result }),
    enabled: options.enabled ?? true,
    staleTime: options.staleTime ?? 30_000,
    refetchInterval: 60_000,
  });
}

export function useToolCommandBreakdown(options: UseToolUsageRunsOptions = {}) {
  const { filter: defaultFilter } = useTimeWindow();
  const filter = options.filter ?? defaultFilter;
  const limit = options.limit ?? 20;
  const toolName = options.toolName ?? "";
  return useQuery<DurableToolCommandBreakdown, Error, ToolCommandBreakdownResponse>({
    queryKey: [...statsQueryKeys.toolCommands(filter, toolName, limit), "durable"] as const,
    queryFn: () => fetchDurableToolCommands(filter, toolName, limit),
    select: (result) => ({ rows: result.rows }),
    enabled: options.enabled ?? !!toolName,
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

  return useQuery<DurableCohort, Error, ToolUsageRunsResponse>({
    queryKey: [...statsQueryKeys.toolRuns(filter, toolName, limit), "durable"] as const,
    queryFn: () => fetchDurableToolCohort(filter, toolName, limit),
    select: (result) => ({ runs: result.rows.map((run) => ({ runId: run.runId, taskTitle: run.taskTitle, profileName: run.profileName, createdAt: run.createdAt, status: run.status, model: run.model, callCount: run.toolCallCount })) }),
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

  return useQuery<DurableRunBreakdown, Error, ToolUsageModelsResponse>({
    queryKey: [...statsQueryKeys.toolModels(filter, toolName, limit), "durable"] as const,
    queryFn: () => fetchDurableToolModels(filter, toolName),
    select: (result) => ({ models: result.rows.slice(0, limit).map((row) => ({ model: row.value, runCount: row.runCount, callCount: undefined, successCount: row.successCount, failedCount: row.failedCount })) }),
    enabled: options.enabled ?? !!toolName,
    staleTime: options.staleTime ?? 30_000,
    refetchInterval: 60_000,
  });
}
