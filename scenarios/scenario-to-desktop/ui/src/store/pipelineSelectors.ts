/**
 * Pipeline store selectors.
 * Extracted from pipelineStore.ts for modularity and reuse.
 */

import type { PreflightSecret } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/preflight_results_pb";
import {
  StageName,
  StageStatus,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";
import type { PipelineStore } from "./pipelineTypes";

// ============================================================================
// Pipeline Status Selectors
// ============================================================================

/** Check if pipeline is currently running or starting */
export const selectIsRunning = (state: PipelineStore) =>
  state.runStatus === "running" || state.runStatus === "starting";

/** Get the current active stage name */
export const selectCurrentStage = (state: PipelineStore) =>
  state.pipelineStatus?.currentStage ?? null;

/** Calculate overall progress (0-1) based on completed stages */
export const selectProgress = (state: PipelineStore) => {
  if (!state.pipelineStatus) return 0;
  // A persisted or in-flight status can legitimately be partial while the API
  // is assembling its stage map. Treat missing collections as no progress
  // instead of crashing the owning screen.
  const stageOrder = state.pipelineStatus.stageOrder ?? [];
  const stages = state.pipelineStatus.stages ?? {};
  if (!stageOrder.length) return 0;
  const completed = stageOrder.filter(
    (s) =>
      stages[stageResultKey(s)]?.status === StageStatus.COMPLETED ||
      stages[stageResultKey(s)]?.status === StageStatus.SKIPPED,
  ).length;
  return completed / stageOrder.length;
};

/** Get status of a specific stage */
export const selectStageStatus =
  (stage: StageName) =>
  (state: PipelineStore): StageStatus =>
    state.pipelineStatus?.stages[stageResultKey(stage)]?.status ??
    StageStatus.PENDING;

/** Check if pipeline can be resumed (stopped after a stage) */
export const selectCanResume = (state: PipelineStore) =>
  state.pipelineStatus?.status === StageStatus.COMPLETED &&
  Boolean(state.pipelineStatus.stoppedAfterStage);

/** Get the stage where pipeline stopped (for resume) */
export const selectStoppedAfterStage = (state: PipelineStore) =>
  state.pipelineStatus?.stoppedAfterStage ?? null;

export function stageResultKey(stage: StageName): string {
  switch (stage) {
    case StageName.RESOLVE_DEPLOYMENT:
      return "resolve-deployment";
    case StageName.BUNDLE:
      return "bundle";
    case StageName.PREFLIGHT:
      return "preflight";
    case StageName.GENERATE:
      return "generate";
    case StageName.BUILD:
      return "build";
    case StageName.SMOKE_TEST:
      return "smoketest";
    case StageName.DEPLOY:
      return "deploy";
    default:
      return "";
  }
}

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
  state.isSubmitting ||
  state.runStatus === "running" ||
  state.runStatus === "starting";

// ============================================================================
// Preflight Selectors
// ============================================================================

/**
 * Check if preflight validation passed.
 * Returns true when validation.valid is true (schema validation passed).
 * This checks that the bundle manifest structure is valid and all referenced
 * assets/binaries exist with matching checksums.
 */
export const selectPreflightValidationOk = (state: PipelineStore) =>
  state.preflightResult?.validation?.valid ?? false;

/**
 * Check if preflight readiness check passed.
 * Returns true when ready.ready is true (services are reachable).
 * This checks that bundled services can start and respond to health checks.
 */
export const selectPreflightReadinessOk = (state: PipelineStore) =>
  state.preflightResult?.ready?.ready ?? false;

/**
 * Stable empty array reference to avoid creating new arrays on every selector call.
 * This prevents infinite re-renders when Zustand compares selector results with Object.is.
 */
const EMPTY_SECRETS_ARRAY: PreflightSecret[] = [];

/**
 * Get missing required secrets from preflight result.
 * Returns a stable empty array reference when there are no missing secrets.
 *
 * Note: For components that need stable references when there ARE missing secrets,
 * use `useShallow` from zustand/react/shallow.
 */
export const selectMissingSecrets = (state: PipelineStore) => {
  const pf = state.preflightResult;
  if (!pf?.secrets) return EMPTY_SECRETS_ARRAY;

  const missing = pf.secrets.filter((s) => s.required && !s.hasValue);
  return missing.length === 0 ? EMPTY_SECRETS_ARRAY : missing;
};

/**
 * Check if all required secrets are provided.
 * Returns true when no secrets are missing values.
 * Missing secrets are those marked as required but without has_value set.
 */
export const selectPreflightSecretsOk = (state: PipelineStore) =>
  selectMissingSecrets(state).length === 0;

/**
 * Check if preflight is fully OK (all checks pass).
 * Returns true only when ALL of the following are true:
 * - validation passed (manifest structure valid, assets exist)
 * - readiness passed (services are reachable)
 * - secrets passed (no required secrets are missing)
 *
 * When this returns false, the UI shows the "Override preflight" checkbox
 * allowing users to bypass validation for development/debugging.
 */
export const selectPreflightOk = (state: PipelineStore) => {
  if (!state.preflightResult) return false;
  return (
    selectPreflightValidationOk(state) &&
    selectPreflightReadinessOk(state) &&
    selectPreflightSecretsOk(state)
  );
};

// ============================================================================
// Stage Result Selectors
// ============================================================================

/** Get bundle stage result */
export const selectBundleResult = (state: PipelineStore) => state.bundleResult;

/** Get preflight stage result */
export const selectPreflightResult = (state: PipelineStore) =>
  state.preflightResult;

/** Get generate stage result */
export const selectGenerateResult = (state: PipelineStore) =>
  state.generateResult;

/** Get build stage result */
export const selectBuildResult = (state: PipelineStore) => state.buildResult;

/** Get smoke test stage result */
export const selectSmokeTestResult = (state: PipelineStore) =>
  state.smokeTestResult;

/** Get deploy stage result */
export const selectDeployResult = (state: PipelineStore) => state.deployResult;

/** Get logs for a specific stage */
export const selectStageLogs =
  (stage: string) =>
  (state: PipelineStore): string[] =>
    state.stageLogs[stage] ?? [];

// ============================================================================
// Error Selectors
// ============================================================================

/**
 * Get the current error message from errorInfo.
 */
export const selectError = (state: PipelineStore) =>
  state.errorInfo?.message ?? null;

/**
 * Get the error message (alias for selectError).
 * Use this when you specifically want just the message string.
 */
export const selectErrorMessage = selectError;

/** Get structured error info with category and suggestions */
export const selectErrorInfo = (state: PipelineStore) => state.errorInfo;

/** Check if there's an error state */
export const selectHasError = (state: PipelineStore) =>
  Boolean(state.errorInfo);

// ============================================================================
// History Selectors
// ============================================================================

/** Get pipeline history for the current scenario */
export const selectPipelineHistory = (state: PipelineStore) =>
  state.pipelineHistory;

/** Get the most recent pipeline ID from history */
export const selectLatestPipelineId = (state: PipelineStore) =>
  state.pipelineHistory.length > 0
    ? state.pipelineHistory[state.pipelineHistory.length - 1]
    : null;

// ============================================================================
// Preflight Input Selectors
// ============================================================================

/** Get preflight secrets input state */
export const selectPreflightSecrets = (state: PipelineStore) =>
  state.preflightSecrets;

/** Get preflight override flag */
export const selectPreflightOverride = (state: PipelineStore) =>
  state.preflightOverride;
