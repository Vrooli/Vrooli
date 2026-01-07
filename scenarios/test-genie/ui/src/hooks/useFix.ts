import { useCallback } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchActiveFix,
  fetchFix,
  spawnFix,
  stopFix,
  type FixRecord,
  type FixPhaseInfo,
  type ActiveFixResponse,
  type SpawnFixResponse
} from "../lib/api";

/**
 * Hook for managing fix operations on a scenario.
 * Provides access to active fix status, spawning, and stopping fixes.
 */
export function useFix(scenarioName: string) {
  const queryClient = useQueryClient();

  // Query for active fix (polls while fix is in progress)
  const activeFixQuery = useQuery<ActiveFixResponse>({
    queryKey: ["fix", "active", scenarioName],
    queryFn: () => fetchActiveFix(scenarioName),
    enabled: !!scenarioName,
    refetchInterval: (query) => {
      const data = query.state.data;
      if (data?.active && data.fix) {
        const status = data.fix.status;
        // Poll every 3 seconds while pending or running
        if (status === "pending" || status === "running") {
          return 3000;
        }
      }
      return false;
    }
  });

  // Mutation for spawning a fix
  const spawnMutation = useMutation<SpawnFixResponse, Error, FixPhaseInfo[]>({
    mutationFn: (phases) => spawnFix(scenarioName, phases),
    onSuccess: () => {
      // Invalidate active fix query to refetch
      queryClient.invalidateQueries({ queryKey: ["fix", "active", scenarioName] });
    }
  });

  // Mutation for stopping a fix
  const stopMutation = useMutation<{ success: boolean; message: string }, Error, string>({
    mutationFn: (fixId) => stopFix(scenarioName, fixId),
    onSuccess: () => {
      // Invalidate active fix query to refetch
      queryClient.invalidateQueries({ queryKey: ["fix", "active", scenarioName] });
    }
  });

  const spawn = useCallback(
    (phases: FixPhaseInfo[]) => {
      return spawnMutation.mutateAsync(phases);
    },
    [spawnMutation]
  );

  const stop = useCallback(
    (fixId: string) => {
      return stopMutation.mutateAsync(fixId);
    },
    [stopMutation]
  );

  return {
    // Active fix data
    activeFix: activeFixQuery.data?.fix ?? null,
    isActive: activeFixQuery.data?.active ?? false,
    isLoading: activeFixQuery.isLoading,
    error: activeFixQuery.error,

    // Spawn mutation
    spawn,
    isSpawning: spawnMutation.isPending,
    spawnError: spawnMutation.error,

    // Stop mutation
    stop,
    isStopping: stopMutation.isPending,
    stopError: stopMutation.error,

    // Refetch
    refetch: activeFixQuery.refetch
  };
}

/**
 * Hook for fetching a specific fix by ID.
 */
export function useFixDetails(scenarioName: string, fixId: string | null) {
  return useQuery<FixRecord>({
    queryKey: ["fix", scenarioName, fixId],
    queryFn: () => fetchFix(scenarioName, fixId!),
    enabled: !!scenarioName && !!fixId,
    refetchInterval: (query) => {
      const data = query.state.data;
      if (data) {
        const status = data.status;
        // Poll every 3 seconds while pending or running
        if (status === "pending" || status === "running") {
          return 3000;
        }
      }
      return false;
    }
  });
}
