import { useCallback } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchActiveRequirementsImprove,
  fetchRequirementsImprove,
  spawnRequirementsImprove,
  stopRequirementsImprove,
  type RequirementsImproveRecord,
  type RequirementImproveInfo,
  type ImproveActionType,
  type ActiveRequirementsImproveResponse,
  type SpawnRequirementsImproveResponse
} from "../lib/api";

/**
 * Hook for managing requirements improve operations on a scenario.
 * Provides access to active improve status, spawning, and stopping improve operations.
 */
export function useRequirementsImprove(scenarioName: string) {
  const queryClient = useQueryClient();

  // Query for active improve (polls while improve is in progress)
  const activeImproveQuery = useQuery<ActiveRequirementsImproveResponse>({
    queryKey: ["requirements-improve", "active", scenarioName],
    queryFn: () => fetchActiveRequirementsImprove(scenarioName),
    enabled: !!scenarioName,
    refetchInterval: (query) => {
      const data = query.state.data;
      if (data?.active && data.improve) {
        const status = data.improve.status;
        // Poll every 3 seconds while pending or running
        if (status === "pending" || status === "running") {
          return 3000;
        }
      }
      return false;
    }
  });

  // Mutation for spawning an improve
  const spawnMutation = useMutation<
    SpawnRequirementsImproveResponse,
    Error,
    { requirements: RequirementImproveInfo[]; actionType: ImproveActionType; message?: string }
  >({
    mutationFn: ({ requirements, actionType, message }) =>
      spawnRequirementsImprove(scenarioName, requirements, actionType, message),
    onSuccess: () => {
      // Invalidate active improve query to refetch
      queryClient.invalidateQueries({
        queryKey: ["requirements-improve", "active", scenarioName]
      });
    }
  });

  // Mutation for stopping an improve
  const stopMutation = useMutation<
    { success: boolean; message: string },
    Error,
    string
  >({
    mutationFn: (improveId) => stopRequirementsImprove(scenarioName, improveId),
    onSuccess: () => {
      // Invalidate active improve query to refetch
      queryClient.invalidateQueries({
        queryKey: ["requirements-improve", "active", scenarioName]
      });
    }
  });

  const spawn = useCallback(
    (requirements: RequirementImproveInfo[], actionType: ImproveActionType, message?: string) => {
      return spawnMutation.mutateAsync({ requirements, actionType, message });
    },
    [spawnMutation]
  );

  const stop = useCallback(
    (improveId: string) => {
      return stopMutation.mutateAsync(improveId);
    },
    [stopMutation]
  );

  return {
    // Active improve data
    activeImprove: activeImproveQuery.data?.improve ?? null,
    isActive: activeImproveQuery.data?.active ?? false,
    isLoading: activeImproveQuery.isLoading,
    error: activeImproveQuery.error,

    // Spawn mutation
    spawn,
    isSpawning: spawnMutation.isPending,
    spawnError: spawnMutation.error,

    // Stop mutation
    stop,
    isStopping: stopMutation.isPending,
    stopError: stopMutation.error,

    // Refetch
    refetch: activeImproveQuery.refetch
  };
}

/**
 * Hook for fetching a specific improve by ID.
 */
export function useRequirementsImproveDetails(
  scenarioName: string,
  improveId: string | null
) {
  return useQuery<RequirementsImproveRecord>({
    queryKey: ["requirements-improve", scenarioName, improveId],
    queryFn: () => fetchRequirementsImprove(scenarioName, improveId!),
    enabled: !!scenarioName && !!improveId,
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
