import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { defaultQueryOptions } from "../lib";
import { scenariosService, executionService } from "../services";
import { useScenariosStore } from "../stores";
import { useStorePolling } from "./useStorePolling";
import type { PreserveFilesPreset, ScenarioFile } from "../types";

export type SpecSyncPhase = "idle" | "syncing" | "archiving" | "done" | "failed";

export function useScenarioDetailData(name: string | undefined, closeDetail: () => void) {
  const queryClient = useQueryClient();
  const upsertScenario = useScenariosStore((state) => state.upsertScenario);
  const removeScenario = useScenariosStore((state) => state.removeScenario);
  const cachedScenario = useScenariosStore((state) => state.scenarios.find((s) => s.name === name));

  // Local state for optimistic UI updates
  const [localGreenfield, setLocalGreenfield] = useState<boolean | null>(null);

  // Spec-sync progress state
  const [specSyncPhase, setSpecSyncPhase] = useState<SpecSyncPhase>("idle");
  const [specSyncExecutionId, setSpecSyncExecutionId] = useState<string | null>(null);
  const [specSyncError, setSpecSyncError] = useState<string | null>(null);

  // File loading state
  const [scenarioFiles, setScenarioFiles] = useState<ScenarioFile[]>([]);
  const [filesLoading, setFilesLoading] = useState(false);

  // Fetch scenario details
  const {
    data: scenario,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: ["scenarios", name],
    queryFn: () => {
      if (!name) throw new Error("Name is required");
      return scenariosService.get(name);
    },
    enabled: !!name,
    placeholderData: cachedScenario,
    ...defaultQueryOptions,
  });

  // Sync local state when scenario data loads
  useEffect(() => {
    if (scenario) {
      setLocalGreenfield(scenario.isGreenfield);
    }
  }, [scenario]);

  // Update metadata mutation
  // [REQ:REQ-P0-007] PATCH endpoint for toggling metadata
  const updateMutation = useMutation({
    mutationFn: (updates: { isGreenfield?: boolean }) => {
      if (!name) throw new Error("Name is required");
      return scenariosService.updateMetadata(name, updates);
    },
    onSuccess: (updatedScenario) => {
      queryClient.setQueryData(["scenarios", name], updatedScenario);
      upsertScenario(updatedScenario);
    },
    onError: () => {
      if (scenario) {
        setLocalGreenfield(scenario.isGreenfield);
      }
    },
  });

  const actionMutation = useMutation({
    mutationFn: async (action: "start" | "stop" | "restart") => {
      if (!name) throw new Error("Name is required");
      if (action === "start") {
        return scenariosService.start(name);
      }
      if (action === "stop") {
        return scenariosService.stop(name);
      }
      return scenariosService.restart(name);
    },
    onSuccess: (updatedScenario) => {
      queryClient.setQueryData(["scenarios", name], updatedScenario);
      upsertScenario(updatedScenario);
    },
  });

  const handleGreenfieldToggle = () => {
    const newValue = !localGreenfield;
    setLocalGreenfield(newValue);
    updateMutation.mutate({ isGreenfield: newValue });
  };

  // Delete mutation
  // [REQ:REQ-P0-008] DELETE endpoint for scenario deletion with archive option
  const deleteMutationInternal = useMutation({
    mutationFn: (params: {
      archiveOnDelete: boolean;
      effectiveArchiveMode: "preset" | "custom";
      customPaths: string[];
      archivePreset: PreserveFilesPreset;
    }) => {
      if (!name) throw new Error("Name is required");
      const preserveFiles = params.archiveOnDelete
        ? params.effectiveArchiveMode === "custom"
          ? { paths: params.customPaths }
          : { preset: params.archivePreset }
        : undefined;
      return scenariosService.delete(name, {
        archive: params.archiveOnDelete,
        preserveFiles,
      });
    },
    onSuccess: () => {
      if (name) {
        removeScenario(name);
      }
      closeDetail();
    },
  });

  // Spec-sync polling effect
  useStorePolling({
    enabled: specSyncPhase === "syncing" && !!specSyncExecutionId,
    intervalMs: 3000,
    pollFn: async () => {
      if (!specSyncExecutionId) return;
      try {
        const execution = await executionService.get(specSyncExecutionId);
        if (execution.status === "completed") {
          setSpecSyncPhase("done");
          if (name) {
            removeScenario(name);
          }
          closeDetail();
        } else if (execution.status === "failed") {
          setSpecSyncPhase("failed");
          setSpecSyncError(execution.failureReason ?? "Spec sync failed");
        } else if (execution.status === "canceled") {
          setSpecSyncPhase("idle");
          setSpecSyncExecutionId(null);
        }
      } catch {
        // Transient error, keep polling
      }
    },
  });

  const triggerSpecSyncArchive = (
    effectiveArchiveMode: "preset" | "custom",
    customPaths: string[],
    archivePreset: PreserveFilesPreset,
  ) => {
    if (!name) return;
    const preserveFiles = effectiveArchiveMode === "custom"
      ? { paths: customPaths }
      : { preset: archivePreset };
    setSpecSyncPhase("syncing");
    setSpecSyncError(null);
    scenariosService
      .specSyncArchive(name, { preserveFiles })
      .then((response) => {
        setSpecSyncExecutionId(response.executionId);
      })
      .catch((err: unknown) => {
        setSpecSyncPhase("failed");
        setSpecSyncError(err instanceof Error ? err.message : "Failed to start spec sync");
      });
  };

  const handleSpecSyncCancel = () => {
    if (specSyncExecutionId) {
      executionService.cancel(specSyncExecutionId).catch((err) => {
        console.error("[spec-sync] cancel failed:", err);
      });
    }
    setSpecSyncPhase("idle");
    setSpecSyncError(null);
    setSpecSyncExecutionId(null);
  };

  const resetSpecSync = () => {
    setSpecSyncPhase("idle");
    setSpecSyncError(null);
    setSpecSyncExecutionId(null);
  };

  // Load scenario files for file selection dialog
  const loadScenarioFiles = async () => {
    if (!name) return;
    setFilesLoading(true);
    try {
      const files = await scenariosService.getFiles(name);
      setScenarioFiles(Array.isArray(files) ? files : []);
    } catch (error) {
      console.error("Failed to load scenario files:", error);
      setScenarioFiles([]);
    } finally {
      setFilesLoading(false);
    }
  };

  return {
    scenario,
    isLoading,
    error,
    refetch,
    localGreenfield,
    handleGreenfieldToggle,
    updateMutation,
    actionMutation,
    deleteMutationInternal,
    specSyncPhase,
    specSyncError,
    isSpecSyncInProgress: specSyncPhase === "syncing" || specSyncPhase === "archiving",
    triggerSpecSyncArchive,
    handleSpecSyncCancel,
    resetSpecSync,
    scenarioFiles,
    filesLoading,
    loadScenarioFiles,
  };
}
