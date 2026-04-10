/**
 * Preflight service - pure functions for preflight validation.
 * Extracted from PreflightSection.tsx and lib/preflight-status.ts for testability.
 */

import type {
  BundlePreflightStep,
  BundlePreflightResponse,
  BundlePreflightSecret,
  BundleValidationResult,
  BundlePreflightReady,
} from "../lib/api";

// ============================================================================
// Types
// ============================================================================

export type PreflightStepId = "validation" | "secrets" | "runtime" | "services" | "diagnostics";

export type PreflightStepState =
  | "pending"
  | "testing"
  | "pass"
  | "fail"
  | "warning"
  | "skipped";

export interface PreflightStepStatus {
  state: PreflightStepState;
  label: string;
}

export interface PreflightExportPayload {
  bundle_manifest_path: string;
  start_services: boolean;
  result: BundlePreflightResponse | null;
  error: string | undefined;
  missing_secrets: BundlePreflightSecret[];
}

export interface PreflightDisplayState {
  hasRun: boolean;
  isRunning: boolean;
  isComplete: boolean;
  hasError: boolean;
  diagnosticsAvailable: boolean;
  validationOk: boolean;
  secretsOk: boolean;
  readinessOk: boolean;
  overallOk: boolean;
}

// ============================================================================
// Step Status Resolution
// ============================================================================

/** State label mapping for job step states */
const JOB_STEP_STATE_LABELS: Record<string, string> = {
  pending: "Pending",
  running: "Testing",
  pass: "Pass",
  fail: "Fail",
  warning: "Review",
  skipped: "Skipped",
};

/**
 * Resolve a job step status from the step map.
 * Returns null if the step is not present.
 */
export function resolveJobStepStatus(
  jobStepById: Map<string, BundlePreflightStep>,
  stepId: string
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
  validationValid?: boolean
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
  missingSecretsCount: number
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
  hasRun: boolean
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
  ready?: boolean
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
  diagnosticsAvailable: boolean
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

// ============================================================================
// Payload Building
// ============================================================================

/**
 * Build the export payload for preflight JSON view.
 */
export function buildPreflightPayload(
  manifestPath: string,
  result: BundlePreflightResponse | null,
  error: string | null,
  missingSecrets: BundlePreflightSecret[]
): PreflightExportPayload {
  return {
    bundle_manifest_path: manifestPath,
    start_services: true,
    result,
    error: error || undefined,
    missing_secrets: missingSecrets,
  };
}

/**
 * Filter secrets to only include non-empty values.
 */
export function filterValidSecrets(secrets: Record<string, string>): Record<string, string> {
  return Object.entries(secrets)
    .filter(([, value]) => value.trim())
    .reduce<Record<string, string>>((acc, [key, value]) => {
      acc[key] = value;
      return acc;
    }, {});
}

// ============================================================================
// Display State Computation
// ============================================================================

/**
 * Check if diagnostics data is available.
 */
export function checkDiagnosticsAvailable(
  portSummary: Record<string, number> | string | null,
  telemetryPath: string | undefined,
  logTails: unknown[] | undefined
): boolean {
  const hasPortSummary = portSummary !== null && (typeof portSummary === "string" ? portSummary.length > 0 : Object.keys(portSummary).length > 0);
  return Boolean(hasPortSummary || telemetryPath || (logTails && logTails.length > 0));
}

/**
 * Build the complete display state for the preflight section.
 */
export function buildPreflightDisplayState(
  pipelineStatus: unknown | null,
  result: BundlePreflightResponse | null,
  error: string | null,
  isRunning: boolean,
  missingSecretsCount: number
): PreflightDisplayState {
  const hasRun = Boolean(result || error || pipelineStatus);
  const isComplete = !isRunning && hasRun;
  const hasError = Boolean(error);

  const validation = result?.validation;
  const readiness = result?.ready;
  const ports = result?.ports;
  const telemetry = result?.telemetry;
  const logTails = result?.log_tails;

  const portSummary = ports?.summary || null;
  const diagnosticsAvailable = checkDiagnosticsAvailable(portSummary, telemetry?.path, logTails);

  const validationOk = validation?.valid === true;
  const secretsOk = missingSecretsCount === 0;
  const readinessOk = readiness?.ready === true;
  const overallOk = validationOk && secretsOk && readinessOk;

  return {
    hasRun,
    isRunning,
    isComplete,
    hasError,
    diagnosticsAvailable,
    validationOk,
    secretsOk,
    readinessOk,
    overallOk,
  };
}

// ============================================================================
// Missing Secrets Helpers
// ============================================================================

/**
 * Filter to get only missing required secrets.
 */
export function getMissingSecrets(secrets: BundlePreflightSecret[] | undefined): BundlePreflightSecret[] {
  if (!secrets) return [];
  return secrets.filter((s) => s.required && !s.has_value);
}

/**
 * Check if all required secrets are present.
 */
export function areSecretsComplete(secrets: BundlePreflightSecret[] | undefined): boolean {
  return getMissingSecrets(secrets).length === 0;
}

// ============================================================================
// Validation Result Helpers
// ============================================================================

/**
 * Check if the bundle validation passed.
 */
export function isValidationOk(validation: BundleValidationResult | undefined): boolean {
  return validation?.valid === true;
}

/**
 * Check if the readiness check passed.
 */
export function isReadinessOk(readiness: BundlePreflightReady | undefined): boolean {
  return readiness?.ready === true;
}

/**
 * Check if preflight is fully OK (validation + readiness + secrets).
 */
export function isPreflightComplete(result: BundlePreflightResponse | null): boolean {
  if (!result) return false;
  return (
    isValidationOk(result.validation) &&
    isReadinessOk(result.ready) &&
    areSecretsComplete(result.secrets)
  );
}
