/**
 * Pipeline controller - orchestrates pipeline business logic.
 * Contains pure functions that can be called from hooks and components.
 * No React imports - keeps business logic separate from React state.
 */

import type { PipelineConfig } from "../lib/api";
import type { PreflightResponse } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/preflight_results_pb";
import type { PipelineRunStatus } from "../store/pipelineTypes";
import {
  deploymentModeFromFormValue,
  PIPELINE_STAGE_BY_FORM_ID,
  platformFromFormValue,
  templateTypeFromFormValue,
} from "../lib/pipeline-enums";

// ============================================================================
// Types
// ============================================================================

export type PipelineStageId = keyof typeof PIPELINE_STAGE_BY_FORM_ID;

export interface BuildPipelineConfigParams {
  scenarioName: string;
  templateType: string;
  deploymentMode: "bundled" | "proxy";
  proxyUrl?: string;
  platforms: string[];
  stopAfterStage?: PipelineStageId;
  bundleManifestPath?: string;
  preflightSecrets?: Record<string, string>;
}

export interface ValidationBeforeRunParams {
  scenarioName: string | null;
  isSubmitting: boolean;
  isBundled: boolean;
  bundleManifestPath?: string;
}

export interface ValidationBeforeRunResult {
  valid: boolean;
  error: string | null;
}

export interface EffectivePreflightParams {
  storeResult: PreflightResponse | null;
  serverResult: PreflightResponse | null | undefined;
}

export interface FormSubmissionParams {
  scenarioName: string;
  templateType: string;
  deploymentMode: "bundled" | "proxy";
  proxyUrl: string;
  platforms: string[];
  bundleManifestPath: string;
  preflightSecrets?: Record<string, string>;
}

// ============================================================================
// Pipeline Configuration
// ============================================================================

/**
 * Build a pipeline config object from form/UI state.
 * This is the canonical way to construct pipeline configs.
 */
export function buildPipelineConfig(
  params: BuildPipelineConfigParams,
): PipelineConfig {
  const {
    scenarioName,
    templateType,
    deploymentMode,
    proxyUrl,
    platforms,
    stopAfterStage,
    bundleManifestPath,
    preflightSecrets,
  } = params;

  const config: PipelineConfig = {
    scenarioName,
    templateType: templateTypeFromFormValue(templateType),
    deploymentMode: deploymentModeFromFormValue(deploymentMode),
    platforms: platforms.map(platformFromFormValue),
  };

  if (stopAfterStage) {
    config.stopAfterStage = PIPELINE_STAGE_BY_FORM_ID[stopAfterStage];
  }

  if (proxyUrl?.trim()) {
    config.proxyUrl = proxyUrl.trim();
  }

  if (bundleManifestPath?.trim()) {
    config.bundleManifestPath = bundleManifestPath.trim();
  }

  if (preflightSecrets && Object.keys(preflightSecrets).length > 0) {
    // Filter out empty secrets
    const filtered = Object.entries(preflightSecrets)
      .filter(([, value]) => value.trim())
      .reduce<Record<string, string>>((acc, [key, value]) => {
        acc[key] = value;
        return acc;
      }, {});
    if (Object.keys(filtered).length > 0) {
      config.preflightSecrets = filtered;
    }
  }

  return config;
}

/**
 * Build a config specifically for the generate stage.
 */
export function buildGenerateConfig(
  params: FormSubmissionParams,
): PipelineConfig {
  return buildPipelineConfig({
    ...params,
    stopAfterStage: "generate",
  });
}

// ============================================================================
// Validation
// ============================================================================

/**
 * Validate parameters before running a pipeline stage.
 * Returns validation result with error message if invalid.
 */
export function validateBeforeRun(
  params: ValidationBeforeRunParams,
): ValidationBeforeRunResult {
  const { scenarioName, isSubmitting, isBundled, bundleManifestPath } = params;

  if (!scenarioName) {
    return { valid: false, error: "No scenario selected" };
  }

  if (isSubmitting) {
    return { valid: false, error: "A pipeline request is already in progress" };
  }

  if (isBundled && !bundleManifestPath?.trim()) {
    return {
      valid: false,
      error: "Bundle manifest path is required for bundled mode",
    };
  }

  return { valid: true, error: null };
}

/**
 * Check if we can proceed to generation based on preflight state.
 */
export function canProceedToGeneration(
  preflightResult: PreflightResponse | null,
  preflightOverride: boolean,
  missingSecretsCount: number,
): { canProceed: boolean; reason: string | null } {
  if (preflightOverride) {
    return { canProceed: true, reason: null };
  }

  if (!preflightResult) {
    return {
      canProceed: false,
      reason: "Preflight validation has not been run",
    };
  }

  if (!preflightResult.validation?.valid) {
    return { canProceed: false, reason: "Bundle validation failed" };
  }

  if (missingSecretsCount > 0) {
    return {
      canProceed: false,
      reason: `Missing ${String(missingSecretsCount)} required secret(s)`,
    };
  }

  if (!preflightResult.ready?.ready) {
    return { canProceed: false, reason: "Services are not ready" };
  }

  return { canProceed: true, reason: null };
}

// ============================================================================
// Preflight Resolution
// ============================================================================

/**
 * Get the effective preflight result, preferring store state over server state.
 * This resolves the "which preflight result to use" question.
 */
export function getEffectivePreflightResult(
  params: EffectivePreflightParams,
): PreflightResponse | null {
  const { storeResult, serverResult } = params;

  // Store result takes priority (it's the most recent)
  if (storeResult) {
    return storeResult;
  }

  // Fall back to server state if available
  if (serverResult) {
    return serverResult;
  }

  return null;
}

/**
 * Calculate effective preflight OK status based on result and missing secrets.
 */
export function getEffectivePreflightOk(
  preflightResult: PreflightResponse | null,
  missingSecretsCount: number,
): boolean {
  if (!preflightResult) {
    return false;
  }

  return (
    Boolean(preflightResult.validation?.valid) &&
    Boolean(preflightResult.ready?.ready) &&
    missingSecretsCount === 0
  );
}

// ============================================================================
// Polling Logic
// ============================================================================

/**
 * Determine if polling should auto-start based on pipeline status.
 */
export function shouldAutoStartPolling(
  status: PipelineRunStatus | null,
): boolean {
  if (!status) return false;
  return status === "running" || status === "starting";
}

/**
 * Determine if the pipeline is in a terminal state where polling should stop.
 */
export function isInTerminalState(status: PipelineRunStatus | null): boolean {
  if (!status) return false;
  return (
    status === "completed" || status === "failed" || status === "cancelled"
  );
}

// ============================================================================
// Secret Filtering
// ============================================================================

/**
 * Filter secrets to only include non-empty values.
 */
export function filterNonEmptySecrets(
  secrets: Record<string, string> | undefined,
): Record<string, string> {
  if (!secrets) return {};

  return Object.entries(secrets)
    .filter(([, value]) => value.trim())
    .reduce<Record<string, string>>((acc, [key, value]) => {
      acc[key] = value;
      return acc;
    }, {});
}

// ============================================================================
// Form State Mapping
// ============================================================================

/**
 * Map domain validation errors to form errors with field names.
 */
export function mapValidationErrorsToFormErrors(
  errors: Array<{ id: string; message: string; field?: string }>,
): Array<{ field: string; message: string; code: string }> {
  return errors.map((e) => ({
    field: e.field || e.id,
    message: e.message,
    code: e.id,
  }));
}
