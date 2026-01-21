/**
 * Shared hook for pipeline button components.
 * Provides mutation, polling, and status mapping for pipeline operations.
 */

import { useState, useCallback, useEffect, useMemo } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  runPipeline,
  getPipelineStatus,
  checkWineStatus,
  type PipelineConfig,
  type PipelineRunResponse,
  type WineCheckResponse,
  type VerbosePipelineStatus,
  type PipelineStatus,
} from "../lib/api";
import { mapPipelineStatus, type MappedBuildStatus } from "../lib/pipeline-utils";

// ============================================================================
// Platform Selection Hook
// ============================================================================

const DEFAULT_PLATFORMS = ["win", "mac", "linux"];

interface UsePlatformSelectionOptions {
  storageKey: string;
  defaultPlatforms?: string[];
}

interface UsePlatformSelectionReturn {
  selectedPlatforms: string[];
  setSelectedPlatforms: (platforms: string[]) => void;
  togglePlatform: (platform: string) => void;
}

/**
 * Hook for managing platform selection with localStorage persistence.
 */
export function usePlatformSelection({
  storageKey,
  defaultPlatforms = DEFAULT_PLATFORMS,
}: UsePlatformSelectionOptions): UsePlatformSelectionReturn {
  const [selectedPlatforms, setSelectedPlatforms] = useState<string[]>(() => {
    if (typeof window === "undefined") return defaultPlatforms;
    try {
      const stored = window.localStorage.getItem(storageKey);
      if (stored) {
        const parsed = JSON.parse(stored);
        if (Array.isArray(parsed) && parsed.every((item) => typeof item === "string")) {
          return parsed;
        }
      }
    } catch {
      // Ignore parse errors
    }
    return defaultPlatforms;
  });

  useEffect(() => {
    if (typeof window === "undefined") return;
    window.localStorage.setItem(storageKey, JSON.stringify(selectedPlatforms));
  }, [selectedPlatforms, storageKey]);

  const togglePlatform = useCallback((platform: string) => {
    setSelectedPlatforms((prev) =>
      prev.includes(platform) ? prev.filter((v) => v !== platform) : [...prev, platform]
    );
  }, []);

  return { selectedPlatforms, setSelectedPlatforms, togglePlatform };
}

// ============================================================================
// Wine Check Hook
// ============================================================================

interface UseWineCheckReturn {
  wineCheck: WineCheckResponse | undefined;
  showWineDialog: boolean;
  setShowWineDialog: (show: boolean) => void;
  pendingPlatforms: string[];
  setPendingPlatforms: (platforms: string[]) => void;
  needsWineForPlatforms: (platforms: string[]) => boolean;
  handleWineInstallComplete: () => void;
}

/**
 * Hook for managing Wine installation checks and dialog.
 */
export function useWineCheck(): UseWineCheckReturn {
  const queryClient = useQueryClient();
  const [showWineDialog, setShowWineDialog] = useState(false);
  const [pendingPlatforms, setPendingPlatforms] = useState<string[]>([]);

  const { data: wineCheck } = useQuery<WineCheckResponse>({
    queryKey: ["wine-check"],
    queryFn: checkWineStatus,
    staleTime: 60000,
  });

  const needsWineForPlatforms = useCallback(
    (platforms: string[]) => {
      return platforms.includes("win") && wineCheck?.platform === "linux" && !wineCheck?.installed;
    },
    [wineCheck]
  );

  const handleWineInstallComplete = useCallback(() => {
    setShowWineDialog(false);
    queryClient.invalidateQueries({ queryKey: ["wine-check"] });
  }, [queryClient]);

  return {
    wineCheck,
    showWineDialog,
    setShowWineDialog,
    pendingPlatforms,
    setPendingPlatforms,
    needsWineForPlatforms,
    handleWineInstallComplete,
  };
}

// ============================================================================
// Pipeline Mutation Hook
// ============================================================================

interface UsePipelineMutationOptions {
  onSuccess?: (data: PipelineRunResponse) => void;
  onError?: (error: Error) => void;
  invalidateOnSuccess?: string[];
  invalidateDelay?: number;
}

interface PipelineMutationState {
  buildId: string | null;
  error: string | null;
}

interface UsePipelineMutationReturn {
  state: PipelineMutationState;
  mutation: ReturnType<typeof useMutation<PipelineRunResponse, Error, PipelineConfig>>;
  runPipelineWithConfig: (config: PipelineConfig) => void;
  reset: () => void;
  clearBuildId: () => void;
}

/**
 * Hook for pipeline mutation with state management.
 */
export function usePipelineMutation(
  options: UsePipelineMutationOptions = {}
): UsePipelineMutationReturn {
  const { onSuccess, onError, invalidateOnSuccess, invalidateDelay = 3000 } = options;
  const queryClient = useQueryClient();
  const [buildId, setBuildId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const mutation = useMutation<PipelineRunResponse, Error, PipelineConfig>({
    mutationFn: runPipeline,
    onSuccess: (data) => {
      setBuildId(data.pipeline_id);
      setError(null);
      onSuccess?.(data);

      if (invalidateOnSuccess?.length) {
        setTimeout(() => {
          for (const key of invalidateOnSuccess) {
            queryClient.invalidateQueries({ queryKey: [key] });
          }
        }, invalidateDelay);
      }
    },
    onError: (err: Error) => {
      setError(err.message);
      onError?.(err);
    },
  });

  const runPipelineWithConfig = useCallback(
    (config: PipelineConfig) => {
      mutation.mutate(config);
    },
    [mutation]
  );

  const reset = useCallback(() => {
    setBuildId(null);
    setError(null);
    mutation.reset();
  }, [mutation]);

  const clearBuildId = useCallback(() => {
    setBuildId(null);
  }, []);

  return {
    state: { buildId, error },
    mutation,
    runPipelineWithConfig,
    reset,
    clearBuildId,
  };
}

// ============================================================================
// Pipeline Status Polling Hook
// ============================================================================

interface UsePipelineStatusOptions {
  buildId: string | null;
  verbose?: boolean;
  pollInterval?: number;
  queryKeyPrefix?: string;
}

interface UsePipelineStatusReturn {
  pipelineStatus: VerbosePipelineStatus | PipelineStatus | null | undefined;
  mappedStatus: MappedBuildStatus | null;
  isBuilding: boolean;
  isComplete: boolean;
  isFailed: boolean;
}

/**
 * Hook for polling pipeline status with automatic stopping on completion.
 */
export function usePipelineStatus(options: UsePipelineStatusOptions): UsePipelineStatusReturn {
  const { buildId, verbose = false, pollInterval = 2000, queryKeyPrefix = "pipeline-status" } = options;

  const { data: pipelineStatus } = useQuery({
    queryKey: [queryKeyPrefix, buildId, verbose],
    queryFn: async () => {
      if (!buildId) return null;
      // Use type assertion to avoid overload issues - the runtime handles this correctly
      return verbose
        ? getPipelineStatus(buildId, { verbose: true })
        : getPipelineStatus(buildId);
    },
    enabled: !!buildId,
    refetchInterval: (query) => {
      const data = query.state.data;
      if (data?.status === "completed" || data?.status === "failed" || data?.status === "cancelled") {
        return false;
      }
      return pollInterval;
    },
  });

  const mappedStatus = useMemo((): MappedBuildStatus | null => {
    if (!pipelineStatus) return null;
    return mapPipelineStatus(pipelineStatus.status);
  }, [pipelineStatus]);

  const isBuilding = mappedStatus === "building";
  const isComplete = mappedStatus === "ready" || mappedStatus === "partial";
  const isFailed = mappedStatus === "failed";

  return {
    pipelineStatus,
    mappedStatus,
    isBuilding,
    isComplete,
    isFailed,
  };
}

