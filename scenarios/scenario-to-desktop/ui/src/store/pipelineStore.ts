/**
 * Unified Zustand store for pipeline state management.
 * Centralizes all pipeline-related state, providing automatic polling,
 * stage-specific results, and actions to run individual stages.
 */

import { create } from "zustand";
import {
  runPipeline,
  getPipelineStatus,
  cancelPipeline as cancelPipelineApi,
  resumePipeline as resumePipelineApi,
  type PipelineConfig,
  type VerbosePipelineStatus,
  type PipelineRunResponse,
  type BundleStageDetails,
  type BundlePreflightResponse,
  type BundlePreflightSecret,
  type GenerateStageDetails,
  type BuildStageDetails,
  type SmokeTestStageDetails,
  type DistributionStageDetails,
} from "../lib/api";
import { extractStageResults } from "../domain/build";
import { createErrorInfo, type ErrorInfo, logError } from "../lib/error-utils";
import { generateUniqueIdempotencyKey, resetSessionId } from "../lib/pipeline-utils";

// ============================================================================
// Types
// ============================================================================

/** Pipeline stages available for stop_after_stage */
export type PipelineStage = "bundle" | "preflight" | "generate" | "build" | "smoketest" | "distribution";

/** Pipeline run status (simplified for UI consumption) */
export type PipelineRunStatus = "idle" | "starting" | "running" | "completed" | "failed" | "cancelled";

/** Structured error information for UI consumption - re-exported from error-utils */
export type PipelineErrorInfo = ErrorInfo;

interface PipelineStoreState {
  // Current scenario context
  scenarioName: string | null;

  // Active pipeline tracking
  pipelineId: string | null;
  pipelineStatus: VerbosePipelineStatus | null;
  runStatus: PipelineRunStatus;
  /** @deprecated Use errorInfo for richer error context */
  error: string | null;
  /** Structured error information with recovery guidance */
  errorInfo: PipelineErrorInfo | null;

  // Polling configuration
  isPolling: boolean;
  pollIntervalMs: number;

  // Stage-specific results (extracted from verbose pipeline status)
  bundleResult: BundleStageDetails | null;
  preflightResult: BundlePreflightResponse | null;
  generateResult: GenerateStageDetails | null;
  buildResult: BuildStageDetails | null;
  smokeTestResult: SmokeTestStageDetails | null;
  distributionResult: DistributionStageDetails | null;

  // Stage logs (from verbose pipeline status)
  stageLogs: Record<string, string[]>;

  // Historical pipeline IDs for this scenario (for resume functionality)
  pipelineHistory: string[];

  // Preflight-specific state (for GeneratorForm integration)
  preflightSecrets: Record<string, string>;
  preflightOverride: boolean;

  // Request deduplication: tracks whether a request is currently in-flight
  // This prevents double-submissions from rapid clicks
  isSubmitting: boolean;
  /** The idempotency key for the current/most recent request */
  currentIdempotencyKey: string | null;
}

interface PipelineStoreActions {
  // Scenario context
  setScenario: (name: string | null) => void;

  // Pipeline execution
  runStage: (stage: PipelineStage, config?: Partial<PipelineConfig>) => Promise<string>;
  runFullPipeline: (config?: Partial<PipelineConfig>) => Promise<string>;
  cancelPipeline: () => Promise<void>;
  resumePipeline: (pipelineId: string) => Promise<string>;

  // Convenience actions for specific stages
  runBundleStage: (config?: Partial<PipelineConfig>) => Promise<string>;
  runPreflightStage: (config?: Partial<PipelineConfig>) => Promise<string>;
  runSmokeTestStage: (config?: Partial<PipelineConfig>) => Promise<string>;

  // Status management
  loadPipelineStatus: (pipelineId: string) => Promise<void>;
  startPolling: () => void;
  stopPolling: () => void;

  // State management
  reset: () => void;
  clearError: () => void;
  /**
   * Forces fresh idempotency keys for future requests.
   * Call this when the user explicitly wants to retry a failed operation,
   * ensuring the retry is treated as a new request rather than deduplicated.
   */
  resetForRetry: () => void;

  // Preflight-specific actions
  setPreflightSecrets: (secrets: Record<string, string>) => void;
  setPreflightSecret: (id: string, value: string) => void;
  setPreflightOverride: (override: boolean) => void;
  resetPreflight: () => void;

  // Internal helpers (prefixed with _ to indicate private)
  _setPipelineStatus: (status: VerbosePipelineStatus | null) => void;
  _extractStageResults: (status: VerbosePipelineStatus) => void;
}

type PipelineStore = PipelineStoreState & PipelineStoreActions;

// ============================================================================
// Initial State
// ============================================================================

const initialState: PipelineStoreState = {
  scenarioName: null,
  pipelineId: null,
  pipelineStatus: null,
  runStatus: "idle",
  error: null,
  errorInfo: null,
  isPolling: false,
  pollIntervalMs: 2000,
  bundleResult: null,
  preflightResult: null,
  generateResult: null,
  buildResult: null,
  smokeTestResult: null,
  distributionResult: null,
  stageLogs: {},
  pipelineHistory: [],
  preflightSecrets: {},
  preflightOverride: false,
  // Idempotency and deduplication
  isSubmitting: false,
  currentIdempotencyKey: null,
};

// ============================================================================
// Store Implementation
// ============================================================================

export const usePipelineStore = create<PipelineStore>((set, get) => {
  // Track polling timeout to allow cleanup
  let pollingTimeoutId: ReturnType<typeof setTimeout> | null = null;

  const clearPollingTimeout = () => {
    if (pollingTimeoutId) {
      clearTimeout(pollingTimeoutId);
      pollingTimeoutId = null;
    }
  };

  return {
    // Initial state
    ...initialState,

    // ========== Scenario Context ==========

    setScenario: (name) => {
      const current = get().scenarioName;
      if (current !== name) {
        // Stop any active polling
        clearPollingTimeout();

        // Reset state when scenario changes
        set({
          ...initialState,
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
      // This ensures the backend deduplicates rapid retries
      const idempotencyKey = generateUniqueIdempotencyKey(scenarioName, stage);

      set({
        runStatus: "starting",
        error: null,
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
          error: errorInfo.message,
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
        error: null,
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
          error: errorInfo.message,
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
        set({ error: errorInfo.message, errorInfo });
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
        error: null,
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
          error: errorInfo.message,
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
        set({ error: errorInfo.message, errorInfo });
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
          const terminalStates = ["completed", "failed", "cancelled"];
          if (!terminalStates.includes(status.status)) {
            pollingTimeoutId = setTimeout(poll, get().pollIntervalMs);
          } else {
            set({ isPolling: false });
          }
        } catch (err) {
          logError("pollPipelineStatus", err);
          const errorInfo = createErrorInfo(err);
          set({
            error: errorInfo.message,
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

    // ========== State Management ==========

    reset: () => {
      clearPollingTimeout();
      const { scenarioName } = get();
      set({
        ...initialState,
        scenarioName, // Keep scenario name when resetting
      });
    },

    clearError: () => set({ error: null, errorInfo: null }),

    resetForRetry: () => {
      // Reset session ID to get fresh idempotency keys for explicit retries
      resetSessionId();
      // Clear submitting state in case it got stuck
      set({
        isSubmitting: false,
        currentIdempotencyKey: null,
        error: null,
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
        error: status.error ?? null,
      });

      get()._extractStageResults(status);
    },

    _extractStageResults: (status) => {
      // Use domain function for pure data transformation, then set store state
      const results = extractStageResults(status);
      set(results);
    },
  };
});

// ============================================================================
// Selectors
// ============================================================================

/** Check if pipeline is currently running or starting */
export const selectIsRunning = (state: PipelineStore) =>
  state.runStatus === "running" || state.runStatus === "starting";

/** Get the current active stage name */
export const selectCurrentStage = (state: PipelineStore) =>
  state.pipelineStatus?.current_stage ?? null;

/** Calculate overall progress (0-1) based on completed stages */
export const selectProgress = (state: PipelineStore) => {
  if (!state.pipelineStatus) return 0;
  const { stage_order, stages } = state.pipelineStatus;
  if (!stage_order?.length) return 0;
  const completed = stage_order.filter(
    (s) => stages?.[s]?.status === "completed" || stages?.[s]?.status === "skipped"
  ).length;
  return completed / stage_order.length;
};

/** Get status of a specific stage */
export const selectStageStatus =
  (stage: PipelineStage) =>
  (state: PipelineStore): string =>
    state.pipelineStatus?.stages?.[stage]?.status ?? "pending";

/** Check if pipeline can be resumed (stopped after a stage) */
export const selectCanResume = (state: PipelineStore) =>
  state.pipelineStatus?.status === "completed" && Boolean(state.pipelineStatus?.stopped_after_stage);

/** Get the stage where pipeline stopped (for resume) */
export const selectStoppedAfterStage = (state: PipelineStore) =>
  state.pipelineStatus?.stopped_after_stage ?? null;

/**
 * Check if a pipeline request is currently being submitted.
 * Use this to disable submit buttons and prevent double-clicks.
 */
export const selectIsSubmitting = (state: PipelineStore) => state.isSubmitting;

/**
 * Check if any pipeline operation is in progress (either submitting or running).
 * This is a combined guard for UI that should block all interactions.
 */
export const selectIsBusy = (state: PipelineStore) =>
  state.isSubmitting || state.runStatus === "running" || state.runStatus === "starting";

// ============================================================================
// Preflight Selectors
// ============================================================================

/** Check if preflight validation passed */
export const selectPreflightValidationOk = (state: PipelineStore) =>
  state.preflightResult?.validation?.valid ?? false;

/** Check if preflight readiness check passed */
export const selectPreflightReadinessOk = (state: PipelineStore) =>
  state.preflightResult?.ready?.ready ?? false;

/**
 * Stable empty array reference to avoid creating new arrays on every selector call.
 * This prevents infinite re-renders when Zustand compares selector results with Object.is.
 */
const EMPTY_SECRETS_ARRAY: BundlePreflightSecret[] = [];

/**
 * Cache for missing secrets to avoid creating new arrays on every call
 * when the underlying data hasn't changed.
 */
let _cachedMissingSecrets: BundlePreflightSecret[] = EMPTY_SECRETS_ARRAY;
let _cachedPreflightSecretsRef: unknown = null;

/** Get missing required secrets from preflight result */
export const selectMissingSecrets = (state: PipelineStore) => {
  const pf = state.preflightResult;
  if (!pf?.secrets) return EMPTY_SECRETS_ARRAY;

  // Return cached result if the secrets array reference hasn't changed
  if (pf.secrets === _cachedPreflightSecretsRef) {
    return _cachedMissingSecrets;
  }

  // Filter and cache the result
  const missing = pf.secrets.filter((s) => s.required && !s.has_value);
  _cachedPreflightSecretsRef = pf.secrets;
  _cachedMissingSecrets = missing.length === 0 ? EMPTY_SECRETS_ARRAY : missing;
  return _cachedMissingSecrets;
};

/** Check if all required secrets are provided */
export const selectPreflightSecretsOk = (state: PipelineStore) =>
  selectMissingSecrets(state).length === 0;

/** Check if preflight is fully OK (validation + readiness + secrets) */
export const selectPreflightOk = (state: PipelineStore) => {
  if (!state.preflightResult) return false;
  return (
    selectPreflightValidationOk(state) &&
    selectPreflightReadinessOk(state) &&
    selectPreflightSecretsOk(state)
  );
};
