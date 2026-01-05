import { useState, useCallback, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  triggerInvestigation,
  listInvestigations,
  getInvestigation,
  stopInvestigation,
  applyFixes,
  getAgentManagerStatus,
} from "../lib/api";
import type { Investigation, InvestigationSummary, CreateInvestigationRequest, ApplyFixesRequest } from "../types/investigation";

/**
 * Hook to fetch the agent-manager status.
 */
export function useAgentManagerStatus() {
  return useQuery({
    queryKey: ["agent-manager-status"],
    queryFn: getAgentManagerStatus,
    staleTime: 30000, // Cache for 30s
    retry: false, // Don't retry if agent-manager is down
  });
}

/**
 * Hook to fetch investigations for a deployment.
 */
export function useInvestigations(deploymentId: string | null, limit = 50) {
  return useQuery({
    queryKey: ["investigations", deploymentId, limit],
    queryFn: async () => {
      if (!deploymentId) return [];
      const res = await listInvestigations(deploymentId, limit);
      return res.investigations;
    },
    enabled: !!deploymentId,
    refetchInterval: (query) => {
      const data = query.state.data as InvestigationSummary[] | undefined;
      // Poll frequently if there's a running investigation
      const hasRunning = data?.some(
        (inv) => inv.status === "pending" || inv.status === "running"
      );
      return hasRunning ? 3000 : false;
    },
  });
}

/**
 * Hook to fetch a single investigation.
 */
export function useInvestigationDetails(
  deploymentId: string | null,
  investigationId: string | null
) {
  return useQuery({
    queryKey: ["investigation", deploymentId, investigationId],
    queryFn: async () => {
      if (!deploymentId || !investigationId) return null;
      const res = await getInvestigation(deploymentId, investigationId);
      return res.investigation;
    },
    enabled: !!deploymentId && !!investigationId,
    refetchInterval: (query) => {
      const data = query.state.data as Investigation | null | undefined;
      // Poll frequently while running
      if (data?.status === "pending" || data?.status === "running") {
        return 2000;
      }
      return false;
    },
  });
}

/**
 * Hook to trigger a new investigation.
 */
export function useTriggerInvestigation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      deploymentId,
      options,
    }: {
      deploymentId: string;
      options?: CreateInvestigationRequest;
    }) => {
      const res = await triggerInvestigation(deploymentId, options);
      return res.investigation;
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["investigations", variables.deploymentId],
      });
    },
  });
}

/**
 * Hook to stop a running investigation.
 */
export function useStopInvestigation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      deploymentId,
      investigationId,
    }: {
      deploymentId: string;
      investigationId: string;
    }) => {
      return stopInvestigation(deploymentId, investigationId);
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["investigations", variables.deploymentId],
      });
      queryClient.invalidateQueries({
        queryKey: ["investigation", variables.deploymentId, variables.investigationId],
      });
    },
  });
}

/**
 * Hook to apply fixes from a completed investigation.
 */
export function useApplyFixes() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      deploymentId,
      investigationId,
      options,
    }: {
      deploymentId: string;
      investigationId: string;
      options: ApplyFixesRequest;
    }) => {
      const res = await applyFixes(deploymentId, investigationId, options);
      return res.investigation;
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["investigations", variables.deploymentId],
      });
    },
  });
}

/**
 * Combined hook for managing investigations on a deployment.
 * Provides convenience methods for triggering, viewing, and stopping investigations.
 */
export function useDeploymentInvestigation(deploymentId: string | null) {
  const queryClient = useQueryClient();
  const [activeInvestigationId, setActiveInvestigationId] = useState<string | null>(null);
  const [showReport, setShowReport] = useState(false);

  // Agent manager status
  const agentStatus = useAgentManagerStatus();

  // List of investigations for this deployment
  const investigationsQuery = useInvestigations(deploymentId);

  // Active investigation details
  const activeInvestigation = useInvestigationDetails(deploymentId, activeInvestigationId);

  // Mutations
  const triggerMutation = useTriggerInvestigation();
  const stopMutation = useStopInvestigation();
  const applyFixesMutation = useApplyFixes();

  useEffect(() => {
    setActiveInvestigationId(null);
    setShowReport(false);
  }, [deploymentId]);

  // Auto-track latest investigation (prefer running, fallback to most recent)
  useEffect(() => {
    const investigations = investigationsQuery.data ?? [];
    if (investigations.length === 0) {
      if (activeInvestigationId) {
        setActiveInvestigationId(null);
      }
      return;
    }

    // Prefer running investigation
    const running = investigations.find(
      (inv) => inv.status === "pending" || inv.status === "running"
    );
    if (running && running.id !== activeInvestigationId) {
      setActiveInvestigationId(running.id);
      return;
    }

    // If no running investigation and none tracked, track the most recent one
    const hasActive = activeInvestigationId
      ? investigations.some((inv) => inv.id === activeInvestigationId)
      : false;
    if (!activeInvestigationId || !hasActive) {
      // Investigations are sorted by created_at desc, so first is most recent
      setActiveInvestigationId(investigations[0].id);
    }
  }, [investigationsQuery.data, activeInvestigationId]);

  // Trigger investigation
  const trigger = useCallback(
    async (options?: CreateInvestigationRequest) => {
      if (!deploymentId) return;
      const inv = await triggerMutation.mutateAsync({ deploymentId, options });
      setActiveInvestigationId(inv.id);
      return inv;
    },
    [deploymentId, triggerMutation]
  );

  // Stop active investigation
  const stop = useCallback(async () => {
    if (!deploymentId || !activeInvestigationId) return;
    await stopMutation.mutateAsync({ deploymentId, investigationId: activeInvestigationId });
  }, [deploymentId, activeInvestigationId, stopMutation]);

  // Apply fixes from an investigation
  const applyFixesFromInvestigation = useCallback(
    async (investigationId: string, options: Omit<ApplyFixesRequest, "note"> & { note?: string }) => {
      if (!deploymentId) return;
      const inv = await applyFixesMutation.mutateAsync({
        deploymentId,
        investigationId,
        options: {
          immediate: options.immediate,
          permanent: options.permanent,
          prevention: options.prevention,
          note: options.note,
        },
      });
      // Track the new fix investigation
      setActiveInvestigationId(inv.id);
      return inv;
    },
    [deploymentId, applyFixesMutation]
  );

  // View investigation report
  const viewReport = useCallback((investigationId: string) => {
    setActiveInvestigationId(investigationId);
    setShowReport(true);
  }, []);

  // Close report modal
  const closeReport = useCallback(() => {
    setShowReport(false);
  }, []);

  // Computed state
  const isAgentAvailable = agentStatus.data?.available ?? false;
  const isAgentEnabled = agentStatus.data?.enabled ?? false;
  const isRunning = activeInvestigation.data?.status === "running" || activeInvestigation.data?.status === "pending";
  const isTriggering = triggerMutation.isPending;
  const isStopping = stopMutation.isPending;

  return {
    // Agent manager status
    agentStatus: agentStatus.data,
    isAgentAvailable,
    isAgentEnabled,
    isAgentLoading: agentStatus.isLoading,

    // Investigations list
    investigations: investigationsQuery.data ?? [],
    isLoadingInvestigations: investigationsQuery.isLoading,

    // Active investigation
    activeInvestigation: activeInvestigation.data,
    activeInvestigationId,
    isRunning,

    // Report modal
    showReport,
    viewReport,
    closeReport,

    // Actions
    trigger,
    stop,
    applyFixes: applyFixesFromInvestigation,
    isTriggering,
    isStopping,
    isApplyingFixes: applyFixesMutation.isPending,
    triggerError: triggerMutation.error,
    stopError: stopMutation.error,
    applyFixesError: applyFixesMutation.error,

    // Refresh
    refresh: () => {
      queryClient.invalidateQueries({ queryKey: ["investigations", deploymentId] });
      if (activeInvestigationId) {
        queryClient.invalidateQueries({
          queryKey: ["investigation", deploymentId, activeInvestigationId],
        });
      }
    },
  };
}
