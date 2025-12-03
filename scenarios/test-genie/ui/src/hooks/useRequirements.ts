import { useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchScenarioRequirements,
  syncScenarioRequirements,
  type RequirementsSnapshot,
  type SyncRequirementsInput
} from "../lib/api";

export interface RequirementsCoverage {
  total: number;
  passed: number;
  failed: number;
  notRun: number;
  completionRate: number;
  passRate: number;
}

export function useRequirements(scenarioName: string | null) {
  const queryClient = useQueryClient();

  const query = useQuery<RequirementsSnapshot | null>({
    queryKey: ["requirements", scenarioName],
    queryFn: () => (scenarioName ? fetchScenarioRequirements(scenarioName) : null),
    enabled: !!scenarioName,
    refetchInterval: 30000, // Refresh every 30 seconds
    staleTime: 10000 // Consider data stale after 10 seconds
  });

  const syncMutation = useMutation({
    mutationFn: (input?: SyncRequirementsInput) => {
      if (!scenarioName) {
        return Promise.reject(new Error("No scenario selected"));
      }
      return syncScenarioRequirements(scenarioName, input);
    },
    onSuccess: () => {
      // Invalidate requirements query to refetch
      queryClient.invalidateQueries({ queryKey: ["requirements", scenarioName] });
    }
  });

  const coverage = useMemo<RequirementsCoverage>(() => {
    const snapshot = query.data;
    if (!snapshot) {
      return {
        total: 0,
        passed: 0,
        failed: 0,
        notRun: 0,
        completionRate: 0,
        passRate: 0
      };
    }

    const byLive = snapshot.summary.byLiveStatus ?? {};
    const passed = byLive["passed"] ?? 0;
    const failed = byLive["failed"] ?? 0;
    const notRun = (byLive["not_run"] ?? 0) + (byLive["skipped"] ?? 0) + (byLive["unknown"] ?? 0);

    return {
      total: snapshot.summary.totalRequirements,
      passed,
      failed,
      notRun,
      completionRate: snapshot.summary.completionRate,
      passRate: snapshot.summary.passRate
    };
  }, [query.data]);

  const syncStatus = useMemo(() => {
    return query.data?.syncStatus ?? null;
  }, [query.data]);

  const modules = useMemo(() => {
    return query.data?.modules ?? [];
  }, [query.data]);

  return {
    ...query,
    snapshot: query.data,
    coverage,
    syncStatus,
    modules,
    sync: syncMutation.mutate,
    syncAsync: syncMutation.mutateAsync,
    isSyncing: syncMutation.isPending
  };
}

export function useRequirementsCoverage(scenarioName: string | null) {
  const { coverage, isLoading, isError } = useRequirements(scenarioName);
  return { coverage, isLoading, isError };
}
