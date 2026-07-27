/**
 * Status resolution helpers for preflight validation.
 * These pure functions compute step statuses based on preflight state.
 */

import {
  JOB_STEP_STATE_LABELS,
  type PreflightStepState,
  type PreflightStepStatus,
} from "./preflight-constants";

/**
 * Local projection of pipeline stage progress for the preflight view.
 * This is derived from the generated pipeline status, not an API payload.
 */
export interface PreflightJobStep {
  id: string;
  name: string;
  state: "pending" | "running" | "pass" | "fail" | "warning" | "skipped";
  detail?: string;
}

/**
 * Resolve a job step status from the step map.
 * Returns null if the step is not present.
 */
export function resolveJobStepStatus(
  jobStepById: Map<string, PreflightJobStep>,
  stepId: string,
): PreflightStepStatus | null {
  const step = jobStepById.get(stepId);
  if (!step) {
    return null;
  }
  const state = step.state === "running" ? "testing" : step.state;
  return {
    state: state as PreflightStepState,
    label: JOB_STEP_STATE_LABELS[step.state] || step.state,
  };
}

/**
 * Get the status for the validation step.
 */
export function getValidationStatus(
  preflightPending: boolean,
  preflightError: string | null,
  hasRun: boolean,
  validationValid?: boolean,
): PreflightStepStatus {
  if (preflightPending) {
    return { state: "testing", label: "Testing" };
  }
  if (preflightError) {
    return { state: "fail", label: "Failed" };
  }
  if (!hasRun) {
    return { state: "pending", label: "Pending" };
  }
  if (validationValid === true) {
    return { state: "pass", label: "Pass" };
  }
  if (validationValid === false) {
    return { state: "fail", label: "Fail" };
  }
  return { state: "warning", label: "Review" };
}

/**
 * Get the status for the secrets step.
 */
export function getSecretsStatus(
  preflightPending: boolean,
  preflightError: string | null,
  hasRun: boolean,
  missingSecretsCount: number,
): PreflightStepStatus {
  if (preflightPending) {
    return { state: "testing", label: "Checking" };
  }
  if (preflightError) {
    return { state: "fail", label: "Failed" };
  }
  if (!hasRun) {
    return { state: "pending", label: "Pending" };
  }
  if (missingSecretsCount > 0) {
    return { state: "warning", label: "Missing" };
  }
  return { state: "pass", label: "Ready" };
}

/**
 * Get the status for the runtime step.
 */
export function getRuntimeStatus(
  preflightPending: boolean,
  preflightError: string | null,
  hasResult: boolean,
  hasRun: boolean,
): PreflightStepStatus {
  if (preflightPending) {
    return { state: "testing", label: "Starting" };
  }
  if (preflightError) {
    return { state: "fail", label: "Failed" };
  }
  if (hasResult) {
    return { state: "pass", label: "Running" };
  }
  if (!hasRun) {
    return { state: "pending", label: "Pending" };
  }
  return { state: "warning", label: "Unknown" };
}

/**
 * Get the status for the services step.
 */
export function getServicesStatus(
  preflightPending: boolean,
  preflightError: string | null,
  hasRun: boolean,
  ready?: boolean,
): PreflightStepStatus {
  if (preflightPending) {
    return { state: "testing", label: "Starting" };
  }
  if (preflightError) {
    return { state: "fail", label: "Failed" };
  }
  if (!hasRun) {
    return { state: "pending", label: "Pending" };
  }
  if (ready === true) {
    return { state: "pass", label: "Ready" };
  }
  if (ready === false) {
    return { state: "warning", label: "Waiting (snapshot)" };
  }
  return { state: "warning", label: "Unknown" };
}

/**
 * Get the status for the diagnostics step.
 */
export function getDiagnosticsStatus(
  preflightPending: boolean,
  preflightError: string | null,
  hasRun: boolean,
  diagnosticsAvailable: boolean,
): PreflightStepStatus {
  if (preflightPending) {
    return { state: "testing", label: "Collecting" };
  }
  if (preflightError) {
    return { state: "fail", label: "Failed" };
  }
  if (!hasRun) {
    return { state: "pending", label: "Pending" };
  }
  if (diagnosticsAvailable) {
    return { state: "pass", label: "Available" };
  }
  return { state: "warning", label: "Empty" };
}
