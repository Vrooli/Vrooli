/**
 * Pipeline store selectors.
 * Extracted from pipelineStore.ts for modularity and reuse.
 */

import type { BundlePreflightSecret } from "../lib/api";
import type { PipelineStore, PipelineStage } from "./pipelineTypes";

// ============================================================================
// Pipeline Status Selectors
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
  state.pipelineStatus?.status === "completed" &&
  Boolean(state.pipelineStatus?.stopped_after_stage);

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
  state.isSubmitting ||
  state.runStatus === "running" ||
  state.runStatus === "starting";

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
 * Get missing required secrets from preflight result.
 * Returns a stable empty array reference when there are no missing secrets.
 *
 * Note: For components that need stable references when there ARE missing secrets,
 * use `useShallow` from zustand/react/shallow.
 */
export const selectMissingSecrets = (state: PipelineStore) => {
  const pf = state.preflightResult;
  if (!pf?.secrets) return EMPTY_SECRETS_ARRAY;

  const missing = pf.secrets.filter((s) => s.required && !s.has_value);
  return missing.length === 0 ? EMPTY_SECRETS_ARRAY : missing;
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

// ============================================================================
// Stage Result Selectors
// ============================================================================

/** Get bundle stage result */
export const selectBundleResult = (state: PipelineStore) => state.bundleResult;

/** Get preflight stage result */
export const selectPreflightResult = (state: PipelineStore) => state.preflightResult;

/** Get generate stage result */
export const selectGenerateResult = (state: PipelineStore) => state.generateResult;

/** Get build stage result */
export const selectBuildResult = (state: PipelineStore) => state.buildResult;

/** Get smoke test stage result */
export const selectSmokeTestResult = (state: PipelineStore) => state.smokeTestResult;

/** Get distribution stage result */
export const selectDistributionResult = (state: PipelineStore) => state.distributionResult;

/** Get logs for a specific stage */
export const selectStageLogs =
  (stage: string) =>
  (state: PipelineStore): string[] =>
    state.stageLogs[stage] ?? [];

// ============================================================================
// Error Selectors
// ============================================================================

/** Get the current error message */
export const selectError = (state: PipelineStore) => state.error;

/** Get structured error info */
export const selectErrorInfo = (state: PipelineStore) => state.errorInfo;

/** Check if there's an error */
export const selectHasError = (state: PipelineStore) =>
  Boolean(state.error || state.errorInfo);

// ============================================================================
// History Selectors
// ============================================================================

/** Get pipeline history for the current scenario */
export const selectPipelineHistory = (state: PipelineStore) => state.pipelineHistory;

/** Get the most recent pipeline ID from history */
export const selectLatestPipelineId = (state: PipelineStore) =>
  state.pipelineHistory.length > 0
    ? state.pipelineHistory[state.pipelineHistory.length - 1]
    : null;

// ============================================================================
// Preflight Input Selectors
// ============================================================================

/** Get preflight secrets input state */
export const selectPreflightSecrets = (state: PipelineStore) => state.preflightSecrets;

/** Get preflight override flag */
export const selectPreflightOverride = (state: PipelineStore) => state.preflightOverride;
