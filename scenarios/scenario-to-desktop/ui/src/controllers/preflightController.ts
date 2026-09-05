/**
 * Preflight controller - orchestrates preflight validation operations.
 * Combines API calls and services for preflight functionality.
 */

import type { PipelineConfig, VerbosePipelineStatus } from "../lib/api";
import {
  buildPreflightPayload,
  filterValidSecrets,
  buildPreflightDisplayState,
  getMissingSecrets,
  isPreflightComplete,
  type PreflightExportPayload,
  type PreflightDisplayState,
  type PreflightStepStatus,
  resolveJobStepStatus,
  getValidationStatus,
  getSecretsStatus,
  getRuntimeStatus,
  getServicesStatus,
  getDiagnosticsStatus,
} from "../services/preflight.service";
import { getBundleRootFromManifestPath } from "../lib/preflight-utils";
import type { PreflightJobStep } from "../lib/preflight-status";
import type { StageResult } from "@vrooli/proto-types/scenario-to-desktop/v1/pipeline/types_pb";
import { StageStatus } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";
import type {
  PreflightResponse,
  PreflightSecret,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/preflight_results_pb";

// ============================================================================
// Types
// ============================================================================

export interface PreflightRunConfig {
  scenarioName: string;
  bundleManifestPath: string;
  secrets?: Record<string, string>;
  additionalConfig?: Partial<PipelineConfig>;
}

export interface PreflightStepStatuses {
  validation: PreflightStepStatus;
  secrets: PreflightStepStatus;
  runtime: PreflightStepStatus;
  services: PreflightStepStatus;
  diagnostics: PreflightStepStatus;
}

export interface PreflightSectionState {
  displayState: PreflightDisplayState;
  stepStatuses: PreflightStepStatuses;
  bundleRootPreview: string;
  portSummary: string | null;
  exportPayload: PreflightExportPayload;
  missingSecrets: PreflightSecret[];
}

// ============================================================================
// Pipeline Config Building
// ============================================================================

/**
 * Build a pipeline config for running preflight.
 */
export function buildPreflightPipelineConfig(
  config: PreflightRunConfig,
): Partial<PipelineConfig> {
  const filteredSecrets = config.secrets
    ? filterValidSecrets(config.secrets)
    : undefined;

  return {
    bundleManifestPath: config.bundleManifestPath || undefined,
    preflightSecrets:
      filteredSecrets && Object.keys(filteredSecrets).length > 0
        ? filteredSecrets
        : undefined,
    ...config.additionalConfig,
  };
}

/**
 * Validate preflight config before running.
 */
export function validatePreflightConfig(
  config: PreflightRunConfig,
): string | null {
  if (!config.scenarioName) {
    return "No scenario selected";
  }
  if (!config.bundleManifestPath.trim()) {
    return "Bundle manifest path is required";
  }
  return null;
}

// ============================================================================
// Display State Building
// ============================================================================

/**
 * Build the job step map from pipeline stages.
 */
export function buildJobStepMap(
  pipelineStatus: VerbosePipelineStatus | null,
): Map<string, PreflightJobStep> {
  const map = new Map<string, PreflightJobStep>();
  if (!pipelineStatus?.stages) return map;

  const stageToStep = (
    stageName: string,
    stage: StageResult,
  ): PreflightJobStep | null => {
    const stateMap: Partial<Record<StageStatus, PreflightJobStep["state"]>> = {
      [StageStatus.PENDING]: "pending",
      [StageStatus.RUNNING]: "running",
      [StageStatus.COMPLETED]: "pass",
      [StageStatus.FAILED]: "fail",
      [StageStatus.CANCELLED]: "fail",
      [StageStatus.SKIPPED]: "skipped",
    };
    return {
      id: stageName,
      name: stageName.charAt(0).toUpperCase() + stageName.slice(1),
      state: stateMap[stage.status] || "pending",
      detail: stage.error,
    };
  };

  // Map bundle stage to validation
  if (pipelineStatus.stages.bundle) {
    const step = stageToStep("bundle", pipelineStatus.stages.bundle);
    if (step) map.set("validation", step);
  }

  // Map preflight stage to multiple steps
  if (pipelineStatus.stages.preflight) {
    const step = stageToStep("preflight", pipelineStatus.stages.preflight);
    if (step) {
      map.set("secrets", step);
      map.set("runtime", step);
      map.set("services", step);
      map.set("diagnostics", step);
    }
  }

  return map;
}

/**
 * Resolve all step statuses for the preflight section.
 */
export function resolveAllStepStatuses(
  jobStepById: Map<string, PreflightJobStep>,
  preflightPending: boolean,
  preflightError: string | null,
  hasRun: boolean,
  preflightResult: {
    validation?: { valid: boolean };
    ready?: { ready: boolean };
  } | null,
  diagnosticsAvailable: boolean,
  missingSecretsCount: number,
): PreflightStepStatuses {
  return {
    validation:
      resolveJobStepStatus(jobStepById, "validation") ??
      getValidationStatus(
        preflightPending,
        preflightError,
        hasRun,
        preflightResult?.validation?.valid,
      ),
    secrets:
      resolveJobStepStatus(jobStepById, "secrets") ??
      getSecretsStatus(
        preflightPending,
        preflightError,
        hasRun,
        missingSecretsCount,
      ),
    runtime:
      resolveJobStepStatus(jobStepById, "runtime") ??
      getRuntimeStatus(
        preflightPending,
        preflightError,
        Boolean(preflightResult),
        hasRun,
      ),
    services:
      resolveJobStepStatus(jobStepById, "services") ??
      getServicesStatus(
        preflightPending,
        preflightError,
        hasRun,
        preflightResult?.ready?.ready,
      ),
    diagnostics:
      resolveJobStepStatus(jobStepById, "diagnostics") ??
      getDiagnosticsStatus(
        preflightPending,
        preflightError,
        hasRun,
        diagnosticsAvailable,
      ),
  };
}

/**
 * Build the complete preflight section state from store data.
 */
export function buildPreflightSectionState(
  bundleManifestPath: string,
  preflightResult: PreflightResponse | null,
  preflightError: string | null,
  pipelineStatus: VerbosePipelineStatus | null,
  isRunning: boolean,
): PreflightSectionState {
  const missingSecrets = getMissingSecrets(preflightResult?.secrets);
  const hasRun = Boolean(preflightResult || preflightError || pipelineStatus);

  // Build display state
  const displayState = buildPreflightDisplayState(
    pipelineStatus,
    preflightResult,
    preflightError,
    isRunning,
    missingSecrets.length,
  );

  // Build job step map
  const jobStepById = buildJobStepMap(pipelineStatus);

  // Get port summary
  const portSummary = preflightResult?.ports.length
    ? preflightResult.ports
        .map(
          ({ serviceId, name, port }) =>
            `${serviceId}(${name}:${String(port)})`,
        )
        .join(" · ")
    : null;
  const diagnosticsAvailable = Boolean(
    portSummary ||
    preflightResult?.telemetry?.path ||
    (preflightResult?.logTails && preflightResult.logTails.length > 0),
  );

  // Resolve step statuses
  const stepStatuses = resolveAllStepStatuses(
    jobStepById,
    isRunning,
    preflightError,
    hasRun,
    preflightResult,
    diagnosticsAvailable,
    missingSecrets.length,
  );

  // Build export payload
  const exportPayload = buildPreflightPayload(
    bundleManifestPath,
    preflightResult,
    preflightError,
    missingSecrets,
  );

  return {
    displayState,
    stepStatuses,
    bundleRootPreview: getBundleRootFromManifestPath(bundleManifestPath),
    portSummary,
    exportPayload,
    missingSecrets,
  };
}

// ============================================================================
// Export Functions
// ============================================================================

/**
 * Export preflight result as JSON string.
 */
export function exportPreflightAsJson(
  manifestPath: string,
  result: PreflightResponse | null,
  error: string | null,
  missingSecrets: PreflightSecret[],
): string {
  const payload = buildPreflightPayload(
    manifestPath,
    result,
    error,
    missingSecrets,
  );
  return JSON.stringify(payload, null, 2);
}

/**
 * Create a downloadable blob for preflight JSON.
 */
export function createPreflightJsonBlob(jsonString: string): Blob {
  return new Blob([jsonString], { type: "application/json" });
}

// ============================================================================
// Validation Helpers
// ============================================================================

/**
 * Check if preflight is ready to proceed to generation.
 */
export function isPreflightReadyForGeneration(
  preflightResult: PreflightResponse | null,
  preflightOverride: boolean,
): boolean {
  if (preflightOverride) return true;
  return isPreflightComplete(preflightResult);
}

/**
 * Get a summary of why preflight is not ready.
 */
export function getPreflightBlockingReason(
  preflightResult: PreflightResponse | null,
): string | null {
  if (!preflightResult) {
    return "Preflight validation has not been run";
  }
  if (!preflightResult.validation?.valid) {
    return "Bundle validation failed";
  }
  const missing = getMissingSecrets(preflightResult.secrets);
  if (missing.length > 0) {
    return `Missing ${String(missing.length)} required secret(s)`;
  }
  if (!preflightResult.ready?.ready) {
    return "Services are not ready";
  }
  return null;
}

// ============================================================================
// Preflight Seed Building
// ============================================================================

export interface PreflightSeed {
  result: PreflightResponse | null;
  error: string | null;
  override: boolean;
  secrets: Record<string, string>;
}

export interface ServerFormState {
  preflight_result?: PreflightResponse | null;
  preflight_error?: string | null;
  preflight_override?: boolean;
  preflight_secrets?: Record<string, string>;
}

/**
 * Build a preflight seed from server form state.
 * Used to initialize preflight state when loading from server.
 */
export function buildPreflightSeed(
  serverFormState: ServerFormState | null,
): PreflightSeed {
  if (!serverFormState) {
    return {
      result: null,
      error: null,
      override: false,
      secrets: {},
    };
  }

  return {
    result: serverFormState.preflight_result ?? null,
    error: serverFormState.preflight_error ?? null,
    override: serverFormState.preflight_override ?? false,
    secrets: serverFormState.preflight_secrets ?? {},
  };
}

// ============================================================================
// Preflight Status Calculation
// ============================================================================

export interface PreflightStatus {
  isComplete: boolean;
  isValid: boolean;
  isReady: boolean;
  hasAllSecrets: boolean;
  canProceed: boolean;
  blockingReason: string | null;
}

/**
 * Calculate the comprehensive preflight status from result, secrets, and override.
 * This is the canonical way to determine preflight readiness.
 */
export function calculatePreflightStatus(
  result: PreflightResponse | null,
  missingSecretsCount: number,
  override: boolean,
): PreflightStatus {
  const isValid = Boolean(result?.validation?.valid);
  const isReady = Boolean(result?.ready?.ready);
  const hasAllSecrets = missingSecretsCount === 0;
  const isComplete = isPreflightComplete(result);

  // Can proceed if override is enabled or all checks pass
  const canProceed = override || (isValid && isReady && hasAllSecrets);

  // Get blocking reason if not overridden
  const blockingReason = override ? null : getPreflightBlockingReason(result);

  return {
    isComplete,
    isValid,
    isReady,
    hasAllSecrets,
    canProceed,
    blockingReason,
  };
}

/**
 * Merge preflight secrets from multiple sources, preferring non-empty values.
 */
export function mergePreflightSecrets(
  storeSecrets: Record<string, string>,
  serverSecrets: Record<string, string> | undefined,
): Record<string, string> {
  const merged: Record<string, string> = { ...serverSecrets };

  // Store secrets take precedence
  for (const [key, value] of Object.entries(storeSecrets)) {
    if (value.trim()) {
      merged[key] = value;
    }
  }

  return merged;
}
