/**
 * Generator controller - orchestrates page data loading and form submission.
 * Combines API calls and services for the Generator page.
 */

import {
  fetchScenarioDesktopStatus,
  fetchProxyHints,
  fetchBundleManifest,
  probeEndpoints,
  runPipeline,
  type ProxyHintsResponse,
  type BundleManifestResponse,
  type ProbeResponse,
} from "../lib/api";
import type { ScenarioDesktopStatus } from "../components/scenario-inventory/types";
import {
  buildPipelineConfigFromForm,
  buildValidationParams,
  type GeneratorFormState,
} from "../services/generator.service";
import { validateFormInputs, type ValidationError } from "../domain/generator";
import type { SigningConfig, BundlePreflightResponse, SigningReadinessResponse } from "../lib/api";
import {
  buildPipelineConfig,
  getEffectivePreflightResult,
  getEffectivePreflightOk,
} from "./pipelineController";

// ============================================================================
// Types
// ============================================================================

export interface GeneratorPageData {
  scenarios: ScenarioDesktopStatus[];
  proxyHints: ProxyHintsResponse | null;
  bundleManifest: BundleManifestResponse | null;
  error: string | null;
}

export interface SubmitGeneratorFormResult {
  pipelineId: string | null;
  error: string | null;
}

export interface ProxyConnectionTestResult {
  result: ProbeResponse | null;
  error: string | null;
}

// ============================================================================
// Page Data Loading
// ============================================================================

/**
 * Load all data needed for the Generator page.
 * Combines multiple API calls for efficient initial page load.
 */
export async function loadGeneratorPageData(
  scenarioName: string | null,
  bundleManifestPath: string | null
): Promise<GeneratorPageData> {
  try {
    // Fetch scenarios (always needed)
    const scenariosResponse = await fetchScenarioDesktopStatus();

    // Optionally fetch proxy hints and bundle manifest if scenario is selected
    let proxyHints: ProxyHintsResponse | null = null;
    let bundleManifest: BundleManifestResponse | null = null;

    if (scenarioName) {
      try {
        proxyHints = await fetchProxyHints(scenarioName);
      } catch (error) {
        console.warn("Failed to load proxy hints:", error);
      }
    }

    if (bundleManifestPath?.trim()) {
      try {
        bundleManifest = await fetchBundleManifest({
          bundle_manifest_path: bundleManifestPath.trim(),
        });
      } catch (error) {
        console.warn("Failed to load bundle manifest:", error);
      }
    }

    return {
      scenarios: scenariosResponse.scenarios,
      proxyHints,
      bundleManifest,
      error: null,
    };
  } catch (error) {
    return {
      scenarios: [],
      proxyHints: null,
      bundleManifest: null,
      error: error instanceof Error ? error.message : "Failed to load page data",
    };
  }
}

/**
 * Load scenarios list.
 */
export async function loadScenarios(): Promise<{
  scenarios: ScenarioDesktopStatus[];
  error: string | null;
}> {
  try {
    const response = await fetchScenarioDesktopStatus();
    return { scenarios: response.scenarios, error: null };
  } catch (error) {
    return {
      scenarios: [],
      error: error instanceof Error ? error.message : "Failed to load scenarios",
    };
  }
}

/**
 * Load proxy hints for a scenario.
 */
export async function loadProxyHints(
  scenarioName: string
): Promise<{ hints: ProxyHintsResponse | null; error: string | null }> {
  if (!scenarioName) {
    return { hints: null, error: null };
  }

  try {
    const hints = await fetchProxyHints(scenarioName);
    return { hints, error: null };
  } catch (error) {
    return {
      hints: null,
      error: error instanceof Error ? error.message : "Failed to load proxy hints",
    };
  }
}

/**
 * Load bundle manifest.
 */
export async function loadBundleManifest(
  manifestPath: string
): Promise<{ manifest: BundleManifestResponse | null; error: string | null }> {
  const path = manifestPath.trim();
  if (!path) {
    return { manifest: null, error: null };
  }

  try {
    const manifest = await fetchBundleManifest({ bundle_manifest_path: path });
    return { manifest, error: null };
  } catch (error) {
    return {
      manifest: null,
      error: error instanceof Error ? error.message : "Failed to load bundle manifest",
    };
  }
}

// ============================================================================
// Form Submission
// ============================================================================

/**
 * Validate and submit the generator form.
 * Returns validation errors if invalid, or pipeline ID if successful.
 */
export async function submitGeneratorForm(
  formState: GeneratorFormState,
  scenarioName: string,
  preflightResult: BundlePreflightResponse | null,
  preflightOk: boolean,
  preflightOverride: boolean,
  signingConfig: SigningConfig | null | undefined,
  signingReadiness: { ready?: boolean; issues?: string[] } | undefined
): Promise<{
  pipelineId: string | null;
  validationErrors: ValidationError[];
  error: string | null;
}> {
  // Build validation params
  const validationParams = buildValidationParams(
    formState,
    scenarioName,
    preflightResult,
    preflightOk,
    signingConfig,
    signingReadiness
  );

  // Add preflightOverride from store
  validationParams.preflightOverride = preflightOverride;

  // Validate
  const errors = validateFormInputs(validationParams);
  if (errors.length > 0) {
    return {
      pipelineId: null,
      validationErrors: errors,
      error: null,
    };
  }

  // Build pipeline config and submit
  try {
    const pipelineConfig = buildPipelineConfigFromForm(formState, scenarioName);
    const response = await runPipeline(pipelineConfig);

    return {
      pipelineId: response.pipeline_id,
      validationErrors: [],
      error: null,
    };
  } catch (error) {
    return {
      pipelineId: null,
      validationErrors: [],
      error: error instanceof Error ? error.message : "Failed to submit form",
    };
  }
}

// ============================================================================
// Connection Testing
// ============================================================================

/**
 * Test proxy connection.
 */
export async function testProxyConnection(
  proxyUrl: string
): Promise<ProxyConnectionTestResult> {
  if (!proxyUrl) {
    return {
      result: null,
      error: "Enter the proxy URL before testing",
    };
  }

  try {
    const result = await probeEndpoints({ proxy_url: proxyUrl });
    return { result, error: null };
  } catch (error) {
    return {
      result: null,
      error: error instanceof Error ? error.message : "Connection test failed",
    };
  }
}

// ============================================================================
// Form State Synchronization
// ============================================================================

/**
 * Find a scenario by name from the list.
 */
export function findScenarioByName(
  scenarios: ScenarioDesktopStatus[],
  name: string
): ScenarioDesktopStatus | undefined {
  return scenarios.find((s) => s.name === name);
}

/**
 * Get default values from a scenario.
 */
export function getScenarioDefaults(scenario: ScenarioDesktopStatus | undefined): {
  displayName: string;
  description: string;
  iconPath: string;
} {
  if (!scenario) {
    return { displayName: "", description: "", iconPath: "" };
  }
  return {
    displayName: scenario.service_display_name || "",
    description: scenario.service_description || "",
    iconPath: scenario.service_icon_path || "",
  };
}

// ============================================================================
// Form Submission Preparation
// ============================================================================

export interface PrepareFormSubmissionParams {
  scenarioName: string;
  selectedTemplate: string;
  deploymentMode: "bundled" | "proxy";
  proxyUrl: string;
  selectedPlatforms: string[];
  isBundled: boolean;
  requiresProxyUrl: boolean;
  bundleManifestPath: string;
  appDisplayName: string;
  appDescription: string;
  locationMode: string; // "proper" | "temp" | "custom"
  outputPath: string;
  signingEnabledForBuild: boolean;
  signingConfig: SigningConfig | null | undefined;
  signingReadiness: SigningReadinessResponse | null | undefined;
  storePreflightResult: BundlePreflightResponse | null;
  serverPreflightResult: BundlePreflightResponse | null | undefined;
  missingSecretsCount: number;
  preflightOverride: boolean;
}

export interface FormSubmissionResult {
  errors: ValidationError[];
  pipelineConfig: ReturnType<typeof buildPipelineConfig> | null;
}

/**
 * Prepare form submission by validating inputs and building pipeline config.
 * This consolidates the validation and config building logic from useGeneratorPage.
 */
export function prepareFormSubmission(params: PrepareFormSubmissionParams): FormSubmissionResult {
  const {
    scenarioName,
    selectedTemplate,
    deploymentMode,
    proxyUrl,
    selectedPlatforms,
    isBundled,
    requiresProxyUrl,
    bundleManifestPath,
    appDisplayName,
    appDescription,
    locationMode,
    outputPath,
    signingEnabledForBuild,
    signingConfig,
    signingReadiness,
    storePreflightResult,
    serverPreflightResult,
    missingSecretsCount,
    preflightOverride,
  } = params;

  // Resolve effective preflight result
  const effectivePreflightResult = getEffectivePreflightResult({
    storeResult: storePreflightResult,
    serverResult: serverPreflightResult,
  });

  // Calculate effective preflight OK status
  const effectivePreflightOk = effectivePreflightResult
    ? getEffectivePreflightOk(effectivePreflightResult, missingSecretsCount)
    : false;

  // Prepare output path for request
  const outputPathForRequest = locationMode === "custom" ? outputPath : "";

  // Build validation params - extract only the fields validateFormInputs needs
  const validationParams = {
    scenarioName,
    selectedPlatforms,
    isBundled,
    requiresProxyUrl,
    bundleManifestPath,
    proxyUrl,
    appDisplayName,
    appDescription,
    locationMode,
    outputPath: outputPathForRequest,
    preflightResult: effectivePreflightResult,
    preflightOk: effectivePreflightOk,
    preflightOverride,
    signingEnabledForBuild,
    signingConfig,
    signingReadiness: signingReadiness
      ? { ready: signingReadiness.ready, issues: signingReadiness.issues }
      : undefined,
  };

  // Validate form inputs
  const domainErrors = validateFormInputs(validationParams);

  if (domainErrors.length > 0) {
    return {
      errors: domainErrors,
      pipelineConfig: null,
    };
  }

  // Build pipeline config
  const pipelineConfig = buildPipelineConfig({
    scenarioName,
    templateType: selectedTemplate,
    deploymentMode,
    proxyUrl: proxyUrl || undefined,
    platforms: selectedPlatforms,
    stopAfterStage: "generate",
    bundleManifestPath: bundleManifestPath || undefined,
  });

  return {
    errors: [],
    pipelineConfig,
  };
}
