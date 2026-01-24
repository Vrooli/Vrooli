/**
 * Unified Zustand store for pipeline state management.
 * Centralizes all pipeline-related state, providing automatic polling,
 * stage-specific results, and actions to run individual stages.
 *
 * Types are defined in ./pipelineTypes.ts
 * Selectors are defined in ./pipelineSelectors.ts
 */

import { create } from "zustand";
import {
  runPipeline,
  getPipelineStatus,
  cancelPipeline as cancelPipelineApi,
  resumePipeline as resumePipelineApi,
  type VerbosePipelineStatus,
  type PipelineRunResponse,
} from "../lib/api";
import { extractStageResults } from "../domain/build";
import { createErrorInfo, logError } from "../lib/error-utils";
import { generateUniqueIdempotencyKey, resetSessionId } from "../lib/pipeline-utils";
import { isTerminalState } from "../services/pipeline.service";
import {
  type PipelineStore,
  type PipelineStage,
  type PipelineRunStatus,
  type PipelineErrorInfo,
  type StatusSubscriber,
  initialPipelineState,
} from "./pipelineTypes";

// Re-export types for convenience
export type { PipelineStage, PipelineRunStatus, PipelineErrorInfo, StatusSubscriber };

// Re-export selectors for convenience
export {
  selectIsRunning,
  selectCurrentStage,
  selectProgress,
  selectStageStatus,
  selectCanResume,
  selectStoppedAfterStage,
  selectIsSubmitting,
  selectIsBusy,
  selectPreflightValidationOk,
  selectPreflightReadinessOk,
  selectPreflightSecretsOk,
  selectPreflightOk,
  selectMissingSecrets,
  selectBundleResult,
  selectPreflightResult,
  selectGenerateResult,
  selectBuildResult,
  selectSmokeTestResult,
  selectDistributionResult,
  selectStageLogs,
  selectError,
  selectErrorMessage,
  selectErrorInfo,
  selectHasError,
  selectPipelineHistory,
  selectLatestPipelineId,
  selectPreflightSecrets,
  selectPreflightOverride,
} from "./pipelineSelectors";

// ============================================================================
// Store Implementation
// ============================================================================

export const usePipelineStore = create<PipelineStore>((set, get) => {
  // Track polling timeout to allow cleanup
  let pollingTimeoutId: ReturnType<typeof setTimeout> | null = null;

  // Status subscribers for component notifications
  const statusSubscribers = new Set<(status: VerbosePipelineStatus | null) => void>();

  const clearPollingTimeout = () => {
    if (pollingTimeoutId) {
      clearTimeout(pollingTimeoutId);
      pollingTimeoutId = null;
    }
  };

  // Notify all subscribers when status changes
  const notifySubscribers = () => {
    const status = get().pipelineStatus;
    statusSubscribers.forEach((callback) => {
      try {
        callback(status);
      } catch (err) {
        console.error("Error in status subscriber:", err);
      }
    });
  };

  return {
    // Initial state
    ...initialPipelineState,

    // ========== Scenario Context ==========

    setScenario: (name) => {
      const current = get().scenarioName;
      if (current !== name) {
        // Stop any active polling
        clearPollingTimeout();

        // Reset state when scenario changes
        set({
          ...initialPipelineState,
          scenarioName: name,
        });
      }
    },

    // ========== Pipeline Execution ==========

    runStage: async (stage, config = {}) => {
      const { scenarioName, isSubmitting } = get();
      if (!scenarioName) {
        throw new Error("No scenario selected");
      }

      // Prevent double-submission: if already submitting, return existing pipeline ID or throw
      if (isSubmitting) {
        const existingId = get().pipelineId;
        if (existingId) {
          // Return the existing pipeline ID - idempotent behavior
          return existingId;
        }
        throw new Error("A pipeline request is already in progress");
      }

      // Stop any existing polling
      clearPollingTimeout();

      // Generate idempotency key for this request
      const idempotencyKey = generateUniqueIdempotencyKey(scenarioName, stage);

      set({
        runStatus: "starting",
        errorInfo: null,
        isSubmitting: true,
        currentIdempotencyKey: idempotencyKey,
      });

      try {
        const response: PipelineRunResponse = await runPipeline({
          scenario_name: scenarioName,
          stop_after_stage: stage,
          idempotency_key: idempotencyKey,
          ...config,
        });

        set({
          pipelineId: response.pipeline_id,
          runStatus: "running",
          isSubmitting: false,
          pipelineHistory: [...get().pipelineHistory, response.pipeline_id],
        });

        // Start polling automatically
        get().startPolling();

        return response.pipeline_id;
      } catch (err) {
        logError("runStage", err);
        const errorInfo = createErrorInfo(err);
        set({
          runStatus: "failed",
          errorInfo,
          isSubmitting: false,
        });
        throw err;
      }
    },

    runBundleStage: (config) => get().runStage("bundle", config),
    runPreflightStage: (config) => get().runStage("preflight", config),
    runSmokeTestStage: (config) => get().runStage("smoketest", config),

    runFullPipeline: async (config = {}) => {
      const { scenarioName, isSubmitting } = get();
      if (!scenarioName) {
        throw new Error("No scenario selected");
      }

      // Prevent double-submission
      if (isSubmitting) {
        const existingId = get().pipelineId;
        if (existingId) {
          return existingId;
        }
        throw new Error("A pipeline request is already in progress");
      }

      // Stop any existing polling
      clearPollingTimeout();

      // Generate idempotency key for full pipeline
      const idempotencyKey = generateUniqueIdempotencyKey(scenarioName, "full");

      set({
        runStatus: "starting",
        errorInfo: null,
        isSubmitting: true,
        currentIdempotencyKey: idempotencyKey,
      });

      try {
        const response = await runPipeline({
          scenario_name: scenarioName,
          idempotency_key: idempotencyKey,
          ...config,
        });

        set({
          pipelineId: response.pipeline_id,
          runStatus: "running",
          isSubmitting: false,
          pipelineHistory: [...get().pipelineHistory, response.pipeline_id],
        });

        get().startPolling();
        return response.pipeline_id;
      } catch (err) {
        logError("runFullPipeline", err);
        const errorInfo = createErrorInfo(err);
        set({
          runStatus: "failed",
          errorInfo,
          isSubmitting: false,
        });
        throw err;
      }
    },

    cancelPipeline: async () => {
      const { pipelineId } = get();
      if (!pipelineId) return;

      try {
        await cancelPipelineApi(pipelineId);
        // Status will update via polling
      } catch (err) {
        logError("cancelPipeline", err);
        const errorInfo = createErrorInfo(err);
        set({ errorInfo });
      }
    },

    resumePipeline: async (parentPipelineId) => {
      const { scenarioName, isSubmitting } = get();
      if (!scenarioName) {
        throw new Error("No scenario selected");
      }

      // Prevent double-submission
      if (isSubmitting) {
        const existingId = get().pipelineId;
        if (existingId) {
          return existingId;
        }
        throw new Error("A pipeline request is already in progress");
      }

      // Stop any existing polling
      clearPollingTimeout();

      set({
        runStatus: "starting",
        errorInfo: null,
        isSubmitting: true,
      });

      try {
        const response = await resumePipelineApi(parentPipelineId);

        set({
          pipelineId: response.pipeline_id,
          runStatus: "running",
          isSubmitting: false,
          pipelineHistory: [...get().pipelineHistory, response.pipeline_id],
        });

        get().startPolling();
        return response.pipeline_id;
      } catch (err) {
        logError("resumePipeline", err);
        const errorInfo = createErrorInfo(err);
        set({
          runStatus: "failed",
          errorInfo,
          isSubmitting: false,
        });
        throw err;
      }
    },

    // ========== Status Management ==========

    loadPipelineStatus: async (pipelineId) => {
      try {
        const status = await getPipelineStatus(pipelineId, { verbose: true });
        get()._setPipelineStatus(status);
      } catch (err) {
        logError("loadPipelineStatus", err);
        const errorInfo = createErrorInfo(err);
        set({ errorInfo });
      }
    },

    startPolling: () => {
      const state = get();
      if (state.isPolling || !state.pipelineId) return;

      set({ isPolling: true });

      const poll = async () => {
        const { pipelineId, isPolling } = get();
        if (!pipelineId || !isPolling) return;

        try {
          const status = await getPipelineStatus(pipelineId, { verbose: true });
          get()._setPipelineStatus(status);

          // Continue polling if not in terminal state
          if (!isTerminalState(status.status)) {
            pollingTimeoutId = setTimeout(poll, get().pollIntervalMs);
          } else {
            set({ isPolling: false });
          }
        } catch (err) {
          logError("pollPipelineStatus", err);
          const errorInfo = createErrorInfo(err);
          set({
            errorInfo,
            isPolling: false,
          });
        }
      };

      poll();
    },

    stopPolling: () => {
      clearPollingTimeout();
      set({ isPolling: false });
    },

    subscribeToStatus: (callback) => {
      statusSubscribers.add(callback);
      // Immediately call with current status
      callback(get().pipelineStatus);
      // Return unsubscribe function
      return () => {
        statusSubscribers.delete(callback);
      };
    },

    // ========== State Management ==========

    reset: () => {
      clearPollingTimeout();
      const { scenarioName } = get();
      set({
        ...initialPipelineState,
        scenarioName, // Keep scenario name when resetting
      });
    },

    clearError: () => set({ errorInfo: null }),

    resetForRetry: () => {
      // Reset session ID to get fresh idempotency keys for explicit retries
      resetSessionId();
      // Clear submitting state in case it got stuck
      set({
        isSubmitting: false,
        currentIdempotencyKey: null,
        errorInfo: null,
      });
    },

    // ========== Preflight-Specific Actions ==========

    setPreflightSecrets: (secrets) => set({ preflightSecrets: secrets }),

    setPreflightSecret: (id, value) =>
      set((state) => ({
        preflightSecrets: { ...state.preflightSecrets, [id]: value },
      })),

    setPreflightOverride: (override) => set({ preflightOverride: override }),

    resetPreflight: () =>
      set({
        preflightResult: null,
        preflightSecrets: {},
        preflightOverride: false,
      }),

    // ========== Internal Helpers ==========

    _setPipelineStatus: (status) => {
      if (!status) {
        set({ pipelineStatus: null });
        get()._notifySubscribers();
        return;
      }

      // Map pipeline status to runStatus
      let runStatus: PipelineRunStatus;
      switch (status.status) {
        case "pending":
        case "running":
          runStatus = "running";
          break;
        case "completed":
          runStatus = "completed";
          break;
        case "failed":
          runStatus = "failed";
          break;
        case "cancelled":
          runStatus = "cancelled";
          break;
        default:
          runStatus = "idle";
      }

      set({
        pipelineStatus: status,
        runStatus,
      });

      get()._extractStageResults(status);
      get()._notifySubscribers();
    },

    _extractStageResults: (status) => {
      // Use domain function for pure data transformation, then set store state
      const results = extractStageResults(status);
      set(results);
    },

    _notifySubscribers: () => {
      notifySubscribers();
    },
  };
});
