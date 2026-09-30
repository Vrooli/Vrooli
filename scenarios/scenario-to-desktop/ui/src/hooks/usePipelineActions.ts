/**
 * Hook for pipeline actions.
 * Wraps pipelineStore and provides actions for running pipeline stages.
 * This replaces direct pipeline API calls in components.
 */

import { useCallback, useEffect } from "react";
import { useMutation } from "@tanstack/react-query";
import {
  usePipelineStore,
  selectIsRunning,
  selectPreflightOk,
  selectMissingSecrets,
  selectIsSubmitting,
  selectIsBusy,
  selectCurrentStage,
  selectProgress,
  selectCanResume,
  selectStoppedAfterStage,
  type PipelineStage,
} from "../store";
import {
  runPipeline,
  probeEndpoints,
  type PipelineConfig,
  type ProbeResponse,
} from "../lib/api";
import { filterNonEmptySecrets } from "../controllers/pipelineController";
import type { StageName } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";
import type { PreflightSecret } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/preflight_results_pb";

// ============================================================================
// Types
// ============================================================================

export interface UsePipelineActionsProps {
  scenarioName: string;
  onBuildStart?: (buildId: string) => void;
}

export interface UsePipelineActionsReturn {
  // Pipeline state from store
  pipelineId: string | null;
  pipelineStatus: ReturnType<
    typeof usePipelineStore.getState
  >["pipelineStatus"];
  runStatus: ReturnType<typeof usePipelineStore.getState>["runStatus"];
  error: string | null;
  errorInfo: ReturnType<typeof usePipelineStore.getState>["errorInfo"];

  // Derived state
  isRunning: boolean;
  isSubmitting: boolean;
  isBusy: boolean;
  currentStage: StageName | null;
  progress: number;
  canResume: boolean;
  stoppedAfterStage: StageName | null;

  // Preflight state
  preflightResult: ReturnType<
    typeof usePipelineStore.getState
  >["preflightResult"];
  preflightSecrets: Record<string, string>;
  preflightOverride: boolean;
  preflightOk: boolean;
  missingPreflightSecrets: PreflightSecret[];

  // Preflight actions
  runPreflight: (
    secretsOverride?: Record<string, string>,
    configOverride?: Partial<PipelineConfig>,
  ) => Promise<void>;
  resetPreflight: () => void;
  setPreflightSecrets: (secrets: Record<string, string>) => void;
  setPreflightSecret: (id: string, value: string) => void;
  setPreflightOverride: (override: boolean) => void;

  // Pipeline actions
  runStage: (
    stage: PipelineStage,
    config?: Partial<PipelineConfig>,
  ) => Promise<string>;
  runFullPipeline: (config?: Partial<PipelineConfig>) => Promise<string>;
  cancelPipeline: () => Promise<void>;
  resumePipeline: (pipelineId: string) => Promise<string>;

  // Stage-specific actions
  runBundleStage: (config?: Partial<PipelineConfig>) => Promise<string>;
  runPreflightStage: (config?: Partial<PipelineConfig>) => Promise<string>;
  runSmokeTestStage: (config?: Partial<PipelineConfig>) => Promise<string>;

  // Generate action (with callback)
  generateDesktop: (config: PipelineConfig) => void;
  generatePending: boolean;
  generateError: string | null;

  // Connection testing
  testConnection: (proxyUrl: string) => void;
  connectionTestPending: boolean;
  connectionTestResult: ProbeResponse | null;
  connectionTestError: string | null;

  // State management
  reset: () => void;
  clearError: () => void;
  resetForRetry: () => void;

  // Status management
  startPolling: () => void;
  stopPolling: () => void;
  loadPipelineStatus: (pipelineId: string) => Promise<void>;
}

// ============================================================================
// Hook Implementation
// ============================================================================

export function usePipelineActions(
  props: UsePipelineActionsProps,
): UsePipelineActionsReturn {
  const { scenarioName, onBuildStart } = props;

  // ========== Pipeline Store ==========
  const store = usePipelineStore();
  const {
    pipelineId,
    pipelineStatus,
    runStatus,
    errorInfo,
    preflightResult,
    preflightSecrets,
    preflightOverride,
    setScenario,
    runStage,
    runFullPipeline,
    cancelPipeline,
    resumePipeline,
    runBundleStage,
    runPreflightStage,
    runSmokeTestStage,
    setPreflightSecrets,
    setPreflightSecret,
    setPreflightOverride,
    resetPreflight,
    reset,
    clearError,
    resetForRetry,
    startPolling,
    stopPolling,
    loadPipelineStatus,
  } = store;

  // ========== Derived State ==========
  const error = errorInfo?.message ?? null;

  // ========== Selectors ==========
  const isRunning = usePipelineStore(selectIsRunning);
  const isSubmitting = usePipelineStore(selectIsSubmitting);
  const isBusy = usePipelineStore(selectIsBusy);
  const preflightOk = usePipelineStore(selectPreflightOk);
  const missingPreflightSecrets = usePipelineStore(selectMissingSecrets);
  const currentStage = usePipelineStore(selectCurrentStage);
  const progress = usePipelineStore(selectProgress);
  const canResume = usePipelineStore(selectCanResume);
  const stoppedAfterStage = usePipelineStore(selectStoppedAfterStage);

  // ========== Effects ==========

  // Set scenario in pipeline store when it changes
  useEffect(() => {
    if (scenarioName) {
      setScenario(scenarioName);
    }
  }, [scenarioName, setScenario]);

  // ========== Generate Mutation ==========

  const generateMutation = useMutation({
    mutationFn: (config: PipelineConfig) => runPipeline(config),
    onSuccess: (data) => {
      onBuildStart?.(data.pipelineId);
    },
  });

  const generateDesktop = useCallback(
    (config: PipelineConfig) => {
      generateMutation.mutate(config);
    },
    [generateMutation],
  );

  const generateError = generateMutation.isError
    ? generateMutation.error.message
    : null;

  // ========== Connection Test Mutation ==========

  const connectionMutation = useMutation({
    mutationFn: (proxyUrl: string) => {
      if (!proxyUrl) {
        throw new Error("Enter the proxy URL above before testing.");
      }
      return probeEndpoints({ proxy_url: proxyUrl });
    },
  });

  const testConnection = useCallback(
    (proxyUrl: string) => {
      connectionMutation.mutate(proxyUrl);
    },
    [connectionMutation],
  );

  // ========== Preflight Action ==========

  const runPreflight = useCallback(
    async (
      secretsOverride?: Record<string, string>,
      configOverride?: Partial<PipelineConfig>,
    ) => {
      if (!scenarioName) return;

      const filteredSecrets = filterNonEmptySecrets(
        secretsOverride ?? preflightSecrets,
      );

      setPreflightOverride(false);

      await runPreflightStage({
        preflightSecrets:
          Object.keys(filteredSecrets).length > 0 ? filteredSecrets : undefined,
        ...configOverride,
      });
    },
    [scenarioName, preflightSecrets, runPreflightStage, setPreflightOverride],
  );

  // ========== Return ==========

  return {
    // Pipeline state
    pipelineId,
    pipelineStatus,
    runStatus,
    error,
    errorInfo,

    // Derived state
    isRunning,
    isSubmitting,
    isBusy,
    currentStage,
    progress,
    canResume,
    stoppedAfterStage,

    // Preflight state
    preflightResult,
    preflightSecrets,
    preflightOverride,
    preflightOk,
    missingPreflightSecrets,

    // Preflight actions
    runPreflight,
    resetPreflight,
    setPreflightSecrets,
    setPreflightSecret,
    setPreflightOverride,

    // Pipeline actions
    runStage,
    runFullPipeline,
    cancelPipeline,
    resumePipeline,

    // Stage-specific actions
    runBundleStage,
    runPreflightStage,
    runSmokeTestStage,

    // Generate action
    generateDesktop,
    generatePending: generateMutation.isPending,
    generateError,

    // Connection testing
    testConnection,
    connectionTestPending: connectionMutation.isPending,
    connectionTestResult: connectionMutation.data ?? null,
    connectionTestError: connectionMutation.error
      ? connectionMutation.error.message
      : null,

    // State management
    reset,
    clearError,
    resetForRetry,

    // Status management
    startPolling,
    stopPolling,
    loadPipelineStatus,
  };
}
