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
  getPipelineStatus,
  cancelPipeline as cancelPipelineApi,
  resumePipeline as resumePipelineApi,
  getActivePipeline,
  createNewPipeline,
  resetPipeline,
  getPipelineHistory,
  startActivePipeline,
  type VerbosePipelineStatus,
  type PipelineConfig,
} from "../lib/api";
import { extractStageResults } from "../domain/build";
import { createErrorInfo, logError } from "../lib/error-utils";
import { resetSessionId } from "../lib/pipeline-utils";
import { isTerminalState } from "../services/pipeline.service";
import {
  type PipelineStore,
  type PipelineStage,
  type PipelineRunStatus,
  type PipelineErrorInfo,
  type StatusSubscriber,
  initialPipelineState,
  PIPELINE_CACHE_MAX_SIZE,
} from "./pipelineTypes";

// Re-export types for convenience
export type { PipelineStage, PipelineRunStatus, PipelineErrorInfo, StatusSubscriber };

// Re-export selectors for convenience
export {
  selectProvenance,
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
  selectDeployResult,
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

  // Debounce tracking for loadActivePipeline to prevent rapid repeated calls
  let lastLoadAttemptTime = 0;
  const LOAD_DEBOUNCE_MS = 500; // Minimum time between load attempts

  // Track ongoing load requests to prevent duplicate concurrent calls
  let activeLoadPromise: Promise<void> | null = null;

  // ============================================================================
  // Rate Limiting Protection
  // ============================================================================
  // Prevents runaway pipeline creation from bugs (e.g., infinite effect loops).
  // Uses exponential backoff: if pipelines are created too quickly, the cooldown
  // doubles each time, up to a maximum. Resets after a quiet period.

  const RATE_LIMIT_WINDOW_MS = 5000; // Time window for counting pipeline creations
  const RATE_LIMIT_MAX_CREATIONS = 3; // Max pipelines allowed in window before throttling
  const RATE_LIMIT_INITIAL_COOLDOWN_MS = 1000; // Initial cooldown when rate limited
  const RATE_LIMIT_MAX_COOLDOWN_MS = 30000; // Maximum cooldown (30 seconds)
  const RATE_LIMIT_RESET_AFTER_MS = 60000; // Reset backoff after 1 minute of quiet

  let pipelineCreationTimestamps: number[] = [];
  let currentCooldownMs = RATE_LIMIT_INITIAL_COOLDOWN_MS;
  let cooldownUntil = 0;
  let lastCreationTime = 0;

  /**
   * Check if pipeline creation is rate limited.
   * Returns null if OK to proceed, or an error message if rate limited.
   */
  const checkRateLimit = (): string | null => {
    const now = Date.now();

    // Reset backoff after quiet period
    if (lastCreationTime > 0 && now - lastCreationTime > RATE_LIMIT_RESET_AFTER_MS) {
      currentCooldownMs = RATE_LIMIT_INITIAL_COOLDOWN_MS;
      pipelineCreationTimestamps = [];
    }

    // Check if we're in a cooldown period
    if (now < cooldownUntil) {
      const remainingMs = cooldownUntil - now;
      console.warn(
        `[PipelineStore] Rate limited: too many pipeline creations. ` +
        `Cooldown: ${Math.ceil(remainingMs / 1000)}s remaining. ` +
        `This usually indicates a bug causing infinite pipeline creation.`
      );
      return `Rate limited: please wait ${Math.ceil(remainingMs / 1000)} seconds before creating another pipeline`;
    }

    // Clean old timestamps outside the window
    pipelineCreationTimestamps = pipelineCreationTimestamps.filter(
      (ts) => now - ts < RATE_LIMIT_WINDOW_MS
    );

    // Check if we've exceeded the rate limit
    if (pipelineCreationTimestamps.length >= RATE_LIMIT_MAX_CREATIONS) {
      // Trigger exponential backoff
      cooldownUntil = now + currentCooldownMs;
      console.error(
        `[PipelineStore] RATE LIMIT TRIGGERED: ${pipelineCreationTimestamps.length} pipelines ` +
        `created in ${RATE_LIMIT_WINDOW_MS / 1000}s. Enforcing ${currentCooldownMs / 1000}s cooldown. ` +
        `This is likely a bug - check for infinite effect loops in React components.`
      );

      // Double the cooldown for next time (exponential backoff)
      currentCooldownMs = Math.min(currentCooldownMs * 2, RATE_LIMIT_MAX_COOLDOWN_MS);

      return `Rate limited: ${RATE_LIMIT_MAX_CREATIONS} pipelines created in ${RATE_LIMIT_WINDOW_MS / 1000}s. Please wait.`;
    }

    return null; // OK to proceed
  };

  /**
   * Record a pipeline creation for rate limiting purposes.
   */
  const recordPipelineCreation = () => {
    const now = Date.now();
    pipelineCreationTimestamps.push(now);
    lastCreationTime = now;
  };

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

      // Guard: Prevent redundant calls with same scenario
      if (current === name) {
        console.debug("[pipelineStore] setScenario skipped - same scenario:", name);
        return;
      }

      // Guard: Prevent setting scenario while loading is in progress for another
      const { isLoadingActivePipeline } = get();
      if (isLoadingActivePipeline && name) {
        console.debug("[pipelineStore] setScenario while loading - stopping current load first");
        // We'll proceed but the current load will be discarded due to scenario mismatch check
      }

      // Stop any active polling for previous scenario
      clearPollingTimeout();

      // Save current state to cache before switching
      if (current) {
        get()._updateCache();
      }

      if (!name) {
        // Clearing scenario - reset to initial state
        set({
          ...initialPipelineState,
          scenarioPipelineCache: get().scenarioPipelineCache,
          runningScenarios: get().runningScenarios,
        });
        return;
      }

      // Check cache for the new scenario
      const cache = get().scenarioPipelineCache;
      const cached = cache.get(name);

      if (cached) {
        // Update last accessed time
        cache.set(name, { ...cached, lastAccessed: Date.now() });

        // Restore from cache
        set({
          ...initialPipelineState,
          scenarioName: name,
          pipelineId: cached.pipelineId,
          runStatus: cached.status,
          scenarioPipelineCache: cache,
          runningScenarios: get().runningScenarios,
          isLoadingActivePipeline: false, // Reset loading state
        });

        // Load full status from server (don't auto-create, we have cached data)
        void get().loadActivePipeline(false);
      } else {
        // Not in cache - set scenario and load from server
        set({
          ...initialPipelineState,
          scenarioName: name,
          scenarioPipelineCache: get().scenarioPipelineCache,
          runningScenarios: get().runningScenarios,
          isLoadingActivePipeline: false, // Reset loading state
        });

        // Load active pipeline from server (will auto-create if needed)
        void get().loadActivePipeline(true);
      }
    },

    // ========== Pipeline Execution ==========

    runStage: async (stage, config = {}) => {
      const { scenarioName, isSubmitting } = get();
      if (!scenarioName) {
        throw new Error("No scenario selected");
      }

      // Rate limit check - prevents runaway pipeline creation from bugs
      const rateLimitError = checkRateLimit();
      if (rateLimitError) {
        throw new Error(rateLimitError);
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

      set({
        runStatus: "starting",
        errorInfo: null,
        isSubmitting: true,
      });

      try {
        // Use startActivePipeline to work with the existing active pipeline
        // rather than creating orphaned new pipelines
        const response = await startActivePipeline(scenarioName, {
          stop_after_stage: stage,
          ...config,
        });

        // Record successful pipeline start for rate limiting
        recordPipelineCreation();

        set({
          pipelineId: response.pipeline.pipeline_id,
          runStatus: "running",
          isSubmitting: false,
          pipelineHistory: [...get().pipelineHistory, response.pipeline.pipeline_id],
        });

        // Start polling automatically
        get().startPolling();

        return response.pipeline.pipeline_id;
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

      // Rate limit check - prevents runaway pipeline creation from bugs
      const rateLimitError = checkRateLimit();
      if (rateLimitError) {
        throw new Error(rateLimitError);
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
        // Use startActivePipeline to work with the existing active pipeline
        // rather than creating orphaned new pipelines
        const response = await startActivePipeline(scenarioName, config);

        // Record successful pipeline start for rate limiting
        recordPipelineCreation();

        set({
          pipelineId: response.pipeline.pipeline_id,
          runStatus: "running",
          isSubmitting: false,
          pipelineHistory: [...get().pipelineHistory, response.pipeline.pipeline_id],
        });

        get().startPolling();
        return response.pipeline.pipeline_id;
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

    // ========== Scenario-Based Pipeline Management ==========

    loadActivePipeline: async (autoCreate = true) => {
      const { scenarioName, isLoadingActivePipeline, pipelineId: currentPipelineId } = get();
      if (!scenarioName) return;

      // Guard 1: Prevent concurrent loads - return existing promise if one is active
      if (activeLoadPromise) {
        console.debug("[pipelineStore] loadActivePipeline skipped - returning existing promise");
        return activeLoadPromise;
      }

      // Guard 2: Prevent concurrent loads via flag (belt-and-suspenders)
      if (isLoadingActivePipeline) {
        console.debug("[pipelineStore] loadActivePipeline skipped - already loading flag set");
        return;
      }

      // Guard 3: Debounce - prevent rapid repeated calls (protects against infinite loops)
      const now = Date.now();
      if (now - lastLoadAttemptTime < LOAD_DEBOUNCE_MS) {
        console.debug("[pipelineStore] loadActivePipeline skipped - debounce active", {
          timeSinceLast: now - lastLoadAttemptTime,
          debounceMs: LOAD_DEBOUNCE_MS,
        });
        return;
      }
      lastLoadAttemptTime = now;

      // Guard 4: If we already have a pipeline WITH full status for this scenario, skip.
      // When restoring from cache we have pipelineId but not pipelineStatus,
      // so we must still fetch from the server to get current stage statuses.
      const currentScenario = get().scenarioName;
      const hasPipelineStatus = get().pipelineStatus !== null;
      if (currentPipelineId && hasPipelineStatus && currentScenario === scenarioName && !autoCreate) {
        console.debug("[pipelineStore] loadActivePipeline skipped - already have pipeline status for scenario");
        return;
      }

      set({ isLoadingActivePipeline: true });

      // Create and track the load promise
      const loadPromise = (async () => {
        try {
          // Capture scenario name at start to detect if it changed during async operation
          const targetScenario = scenarioName;

          const response = await getActivePipeline(targetScenario, { autoCreate });

          // Guard: If scenario changed during fetch, discard results
          if (get().scenarioName !== targetScenario) {
            console.debug("[pipelineStore] loadActivePipeline discarding stale results - scenario changed");
            set({ isLoadingActivePipeline: false });
            return;
          }

          if (response.pipeline) {
            // Load full status with verbose details
            const status = await getPipelineStatus(response.pipeline.pipeline_id, { verbose: true });

            // Guard again after second fetch
            if (get().scenarioName !== targetScenario) {
              console.debug("[pipelineStore] loadActivePipeline discarding stale status - scenario changed");
              set({ isLoadingActivePipeline: false });
              return;
            }

            set({
              pipelineId: status.pipeline_id,
              isLoadingActivePipeline: false,
            });
            get()._setPipelineStatus(status);

            // Start polling if pipeline is actively running (not idle, not in terminal state)
            // "idle" means created but not started - don't poll
            const isActivelyRunning = status.status === "running" || status.status === "pending";
            if (isActivelyRunning) {
              get().startPolling();
            }

            // Update running scenarios set
            const runningScenarios = new Set(get().runningScenarios);
            if (isActivelyRunning) {
              runningScenarios.add(targetScenario);
            } else {
              runningScenarios.delete(targetScenario);
            }
            set({ runningScenarios });

            // Update cache immediately when we load a pipeline (don't wait for scenario switch)
            get()._updateCache();
          } else {
            set({ isLoadingActivePipeline: false });
          }
        } catch (err) {
          logError("loadActivePipeline", err);
          const errorInfo = createErrorInfo(err);
          set({ errorInfo, isLoadingActivePipeline: false });
        } finally {
          activeLoadPromise = null;
        }
      })();

      activeLoadPromise = loadPromise;
      return loadPromise;
    },

    createNewPipelineForScenario: async (config?: Partial<PipelineConfig>) => {
      const { scenarioName, isSubmitting } = get();
      if (!scenarioName) {
        throw new Error("No scenario selected");
      }

      if (isSubmitting) {
        const existingId = get().pipelineId;
        if (existingId) return existingId;
        throw new Error("A pipeline request is already in progress");
      }

      clearPollingTimeout();
      set({
        runStatus: "starting",
        errorInfo: null,
        isSubmitting: true,
      });

      try {
        const response = await createNewPipeline(scenarioName, config);
        const status = await getPipelineStatus(response.pipeline.pipeline_id, { verbose: true });

        // New pipelines are created in "idle" state (not running yet)
        set({
          pipelineId: status.pipeline_id,
          isSubmitting: false,
          pipelineHistory: [...get().pipelineHistory, status.pipeline_id],
        });

        // Let _setPipelineStatus determine the correct runStatus from server status
        get()._setPipelineStatus(status);

        // Only start polling if the pipeline is actively running
        const isActivelyRunning = status.status === "running" || status.status === "pending";
        if (isActivelyRunning) {
          get().startPolling();
          // Update running scenarios
          const runningScenarios = new Set(get().runningScenarios);
          runningScenarios.add(scenarioName);
          set({ runningScenarios });
        }

        return status.pipeline_id;
      } catch (err) {
        logError("createNewPipelineForScenario", err);
        const errorInfo = createErrorInfo(err);
        set({
          runStatus: "failed",
          errorInfo,
          isSubmitting: false,
        });
        throw err;
      }
    },

    resetCurrentPipeline: async () => {
      const { scenarioName } = get();
      if (!scenarioName) return;

      clearPollingTimeout();

      try {
        await resetPipeline(scenarioName);

        // Remove from running scenarios
        const runningScenarios = new Set(get().runningScenarios);
        runningScenarios.delete(scenarioName);

        // Reset local state but keep scenario context
        set({
          pipelineId: null,
          pipelineStatus: null,
          runStatus: "idle",
          errorInfo: null,
          isPolling: false,
          bundleResult: null,
          preflightResult: null,
          generateResult: null,
          buildResult: null,
          smokeTestResult: null,
          deployResult: null,
          stageLogs: {},
          runningScenarios,
        });
      } catch (err) {
        logError("resetCurrentPipeline", err);
        const errorInfo = createErrorInfo(err);
        set({ errorInfo });
      }
    },

    loadPipelineHistory: async (limit = 10) => {
      const { scenarioName } = get();
      if (!scenarioName) return [];

      try {
        const response = await getPipelineHistory(scenarioName, { limit });

        // Fetch verbose status for each pipeline
        const pipelines: VerbosePipelineStatus[] = [];
        for (const p of response.pipelines) {
          try {
            const status = await getPipelineStatus(p.pipeline_id, { verbose: true });
            pipelines.push(status);
          } catch {
            // Skip pipelines that can't be fetched
          }
        }

        return pipelines;
      } catch (err) {
        logError("loadPipelineHistory", err);
        return [];
      }
    },

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
        case "idle":
          // Server-side "idle" status means pipeline is created but not started
          runStatus = "idle";
          break;
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

      // Update cache whenever status changes (not just on scenario switch)
      // This ensures the cache has current state after failures, completions, etc.
      get()._updateCache();
    },

    _extractStageResults: (status) => {
      // Use domain function for pure data transformation, then set store state
      const results = extractStageResults(status);
      set(results);
    },

    _notifySubscribers: () => {
      notifySubscribers();
    },

    _updateCache: () => {
      const { scenarioName, pipelineId, runStatus, scenarioPipelineCache } = get();
      if (!scenarioName || !pipelineId) return;

      scenarioPipelineCache.set(scenarioName, {
        pipelineId,
        status: runStatus,
        lastAccessed: Date.now(),
      });

      // Update running scenarios based on status
      const runningScenarios = new Set(get().runningScenarios);
      if (runStatus === "running" || runStatus === "starting") {
        runningScenarios.add(scenarioName);
      } else {
        runningScenarios.delete(scenarioName);
      }

      set({ scenarioPipelineCache, runningScenarios });
      get()._pruneCache();
    },

    _pruneCache: () => {
      const { scenarioPipelineCache, runningScenarios } = get();

      if (scenarioPipelineCache.size <= PIPELINE_CACHE_MAX_SIZE) return;

      // Get entries sorted by lastAccessed (oldest first)
      const entries = Array.from(scenarioPipelineCache.entries())
        .sort((a, b) => a[1].lastAccessed - b[1].lastAccessed);

      // Remove oldest entries that aren't running
      for (const [scenario] of entries) {
        if (scenarioPipelineCache.size <= PIPELINE_CACHE_MAX_SIZE) break;
        if (!runningScenarios.has(scenario)) {
          scenarioPipelineCache.delete(scenario);
        }
      }

      set({ scenarioPipelineCache });
    },
  };
});
