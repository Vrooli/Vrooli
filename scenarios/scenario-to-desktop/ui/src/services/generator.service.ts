/**
 * Generator service - pure functions for desktop app generation.
 * Extracted from GeneratorForm.tsx and domain/generator.ts for testability.
 */

import type { PipelineConfig, ProbeResponse } from "../lib/api";
import type { SigningConfig } from "../domain/signing";
import type { PreflightResponse } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/preflight_results_pb";
import {
  deploymentModeFromFormValue,
  platformFromFormValue,
  templateTypeFromFormValue,
} from "../lib/pipeline-enums";
import { StageName } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";
import type { DesktopConnectionConfig } from "../components/scenario-inventory/types";
import type { DeploymentMode, ServerType } from "../domain/deployment";
import type {
  DesktopFramework,
  OutputLocation,
  PlatformSelection,
} from "../domain/generator";
import {
  getSelectedPlatforms,
  type ValidateFormInputsParams,
} from "../domain/generator";
import {
  decideConnection,
  DEFAULT_DEPLOYMENT_MODE,
  DEFAULT_SERVER_TYPE,
} from "../domain/deployment";
import { DEFAULT_LOCAL_API_ENDPOINT } from "../store/formTypes";

// ============================================================================
// Re-exports from domain layer
// ============================================================================

export {
  validateFormInputs,
  resolveEndpoints,
  computeStandardOutputPath,
  computeStagingPreviewPath,
  getSelectedPlatforms,
} from "../domain/generator";

export type {
  ValidationError,
  BuildDesktopConfigOptions,
  ValidateFormInputsParams,
  EndpointResolution,
  OutputLocation,
  PlatformSelection,
} from "../domain/generator";

// ============================================================================
// Form State Types
// ============================================================================

export interface GeneratorFormState {
  appMetadata: {
    displayName: string;
    description: string;
    iconPath: string;
    displayNameEdited: boolean;
    descriptionEdited: boolean;
    iconPathEdited: boolean;
    iconPreviewError: boolean;
  };
  deployment: {
    mode: DeploymentMode;
    serverType: ServerType;
    framework: DesktopFramework;
  };
  output: {
    locationMode: OutputLocation;
    outputPath: string;
  };
  platforms: PlatformSelection;
  connection: {
    proxyUrl: string;
    bundleManifestPath: string;
    serverPort: number;
    localServerPath: string;
    localApiEndpoint: string;
    autoManageTier1: boolean;
    vrooliBinaryPath: string;
    connectionResult: ProbeResponse | null;
    connectionError: string | null;
  };
  selectedTemplate: string;
  signingEnabledForBuild: boolean;
}

// ============================================================================
// Icon Preview
// ============================================================================

/**
 * Build the icon preview URL from an icon path.
 * Returns empty string if no path provided.
 */
export function buildIconPreviewUrl(iconPath: string, apiBaseUrl = ""): string {
  if (!iconPath) return "";
  const encodedPath = encodeURIComponent(iconPath);
  return `${apiBaseUrl}/api/icon-preview?path=${encodedPath}`;
}

// ============================================================================
// Scenario Defaults
// ============================================================================

export interface ScenarioDefaults {
  displayName: string;
  description: string;
  iconPath: string;
}

/**
 * Extract default values from a scenario for form population.
 */
export function extractScenarioDefaults(
  scenario: {
    service_display_name?: string;
    service_description?: string;
    service_icon_path?: string;
  } | null,
): ScenarioDefaults {
  if (!scenario) {
    return {
      displayName: "",
      description: "",
      iconPath: "",
    };
  }
  return {
    displayName: scenario.service_display_name || "",
    description: scenario.service_description || "",
    iconPath: scenario.service_icon_path || "",
  };
}

/**
 * Apply scenario defaults to form state, respecting user edits.
 * Returns the fields that should be updated.
 */
export function applyScenarioDefaults(
  scenario: ScenarioDefaults,
  currentState: {
    displayNameEdited: boolean;
    descriptionEdited: boolean;
    iconPathEdited: boolean;
  },
): Partial<ScenarioDefaults> {
  const updates: Partial<ScenarioDefaults> = {};

  if (!currentState.displayNameEdited) {
    updates.displayName = scenario.displayName;
  }
  if (!currentState.descriptionEdited) {
    updates.description = scenario.description;
  }
  if (!currentState.iconPathEdited) {
    updates.iconPath = scenario.iconPath;
  }

  return updates;
}

// ============================================================================
// Connection Config Transformation
// ============================================================================

/**
 * Transform a connection config from the scenario inventory to form state.
 */
export function transformConnectionConfigToFormState(
  config: DesktopConnectionConfig | null | undefined,
): Partial<GeneratorFormState> {
  if (!config) return {};

  return {
    deployment: {
      mode: config.deployment_mode
        ? (config.deployment_mode as DeploymentMode)
        : DEFAULT_DEPLOYMENT_MODE,
      serverType: config.server_type
        ? (config.server_type as ServerType)
        : DEFAULT_SERVER_TYPE,
      framework: "electron",
    },
    connection: {
      proxyUrl: config.proxy_url ?? config.server_url ?? "",
      bundleManifestPath: config.bundle_manifest_path ?? "",
      serverPort: 3000,
      localServerPath: "ui/server.js",
      localApiEndpoint: DEFAULT_LOCAL_API_ENDPOINT,
      autoManageTier1: config.auto_manage_vrooli ?? false,
      vrooliBinaryPath: config.vrooli_binary_path ?? "vrooli",
      connectionResult: null,
      connectionError: null,
    },
    appMetadata: {
      displayName: config.app_display_name ?? "",
      description: config.app_description ?? "",
      iconPath: config.icon ?? "",
      displayNameEdited: Boolean(config.app_display_name),
      descriptionEdited: Boolean(config.app_description),
      iconPathEdited: Boolean(config.icon),
      iconPreviewError: false,
    },
  };
}

// ============================================================================
// Pipeline Config Building
// ============================================================================

/**
 * Build a pipeline config from form state for submission.
 */
export function buildPipelineConfigFromForm(
  formState: GeneratorFormState,
  scenarioName: string,
): PipelineConfig {
  const selectedPlatforms = getSelectedPlatforms(formState.platforms);

  return {
    scenarioName,
    templateType: templateTypeFromFormValue(formState.selectedTemplate),
    deploymentMode: deploymentModeFromFormValue(formState.deployment.mode),
    proxyUrl: formState.connection.proxyUrl || undefined,
    platforms: selectedPlatforms.map(platformFromFormValue),
    stopAfterStage: StageName.GENERATE,
  };
}

// ============================================================================
// Validation Helpers
// ============================================================================

/**
 * Build validation params from form state for use with validateFormInputs.
 */
export function buildValidationParams(
  formState: GeneratorFormState,
  scenarioName: string,
  preflightResult: PreflightResponse | null,
  preflightOk: boolean,
  signingConfig: SigningConfig | null | undefined,
  signingReadiness: { ready?: boolean; issues?: string[] } | undefined,
): ValidateFormInputsParams {
  const connectionDecision = decideConnection(
    formState.deployment.mode,
    formState.deployment.serverType,
  );
  const isBundled = connectionDecision.kind === "bundled-runtime";
  const requiresProxyUrl = connectionDecision.requiresProxyUrl;
  const selectedPlatforms = getSelectedPlatforms(formState.platforms);
  const outputPath =
    formState.output.locationMode === "custom"
      ? formState.output.outputPath
      : "";

  return {
    scenarioName,
    selectedPlatforms,
    isBundled,
    requiresProxyUrl,
    bundleManifestPath: formState.connection.bundleManifestPath,
    proxyUrl: formState.connection.proxyUrl,
    appDisplayName: formState.appMetadata.displayName,
    appDescription: formState.appMetadata.description,
    locationMode: formState.output.locationMode,
    outputPath,
    preflightResult,
    preflightOk,
    preflightOverride: false, // Will be passed from store
    signingEnabledForBuild: formState.signingEnabledForBuild,
    signingConfig,
    signingReadiness,
  };
}

// ============================================================================
// Server Type Helpers
// ============================================================================

import { SERVER_TYPE_OPTIONS } from "../domain/deployment";

/**
 * Get allowed server types based on deployment mode.
 */
export function getAllowedServerTypes(
  deploymentMode: DeploymentMode,
): ServerType[] {
  if (deploymentMode === "bundled" || deploymentMode === "cloud-api") {
    return ["external"];
  }
  return SERVER_TYPE_OPTIONS.map((option) => option.value);
}

/**
 * Validate and adjust server type based on deployment mode.
 * Returns the adjusted server type if current is not allowed.
 */
export function adjustServerTypeForMode(
  currentServerType: ServerType,
  deploymentMode: DeploymentMode,
): ServerType {
  const allowed = getAllowedServerTypes(deploymentMode);
  if (!allowed.includes(currentServerType)) {
    return allowed[0] ?? DEFAULT_SERVER_TYPE;
  }
  return currentServerType;
}

// ============================================================================
// Form State Serialization
// ============================================================================

export interface SerializedFormState {
  selected_template: string;
  app_display_name: string;
  app_description: string;
  icon_path: string;
  display_name_edited: boolean;
  description_edited: boolean;
  icon_path_edited: boolean;
  framework: DesktopFramework;
  server_type: string;
  deployment_mode: string;
  platforms: PlatformSelection;
  location_mode: string;
  output_path: string;
  proxy_url: string;
  bundle_manifest_path: string;
  server_port: number;
  local_server_path: string;
  local_api_endpoint: string;
  auto_manage_tier1: boolean;
  vrooli_binary_path: string;
  connection_result: ProbeResponse | null;
  connection_error: string | null;
  signing_enabled_for_build: boolean;
}

/**
 * Serialize form state for server persistence.
 */
export function serializeFormStateForServer(
  formState: GeneratorFormState,
): SerializedFormState {
  return {
    selected_template: formState.selectedTemplate,
    app_display_name: formState.appMetadata.displayName,
    app_description: formState.appMetadata.description,
    icon_path: formState.appMetadata.iconPath,
    display_name_edited: formState.appMetadata.displayNameEdited,
    description_edited: formState.appMetadata.descriptionEdited,
    icon_path_edited: formState.appMetadata.iconPathEdited,
    framework: formState.deployment.framework,
    server_type: formState.deployment.serverType,
    deployment_mode: formState.deployment.mode,
    platforms: formState.platforms,
    location_mode: formState.output.locationMode,
    output_path: formState.output.outputPath,
    proxy_url: formState.connection.proxyUrl,
    bundle_manifest_path: formState.connection.bundleManifestPath,
    server_port: formState.connection.serverPort,
    local_server_path: formState.connection.localServerPath,
    local_api_endpoint: formState.connection.localApiEndpoint,
    auto_manage_tier1: formState.connection.autoManageTier1,
    vrooli_binary_path: formState.connection.vrooliBinaryPath,
    connection_result: formState.connection.connectionResult,
    connection_error: formState.connection.connectionError,
    signing_enabled_for_build: formState.signingEnabledForBuild,
  };
}

/**
 * Deserialize form state from server response.
 */
export function deserializeFormStateFromServer(
  data: Partial<SerializedFormState>,
): Partial<GeneratorFormState> {
  return {
    selectedTemplate: data.selected_template,
    appMetadata: {
      displayName: data.app_display_name ?? "",
      description: data.app_description ?? "",
      iconPath: data.icon_path ?? "",
      displayNameEdited: data.display_name_edited ?? false,
      descriptionEdited: data.description_edited ?? false,
      iconPathEdited: data.icon_path_edited ?? false,
      iconPreviewError: false,
    },
    deployment: {
      mode: data.deployment_mode
        ? (data.deployment_mode as DeploymentMode)
        : DEFAULT_DEPLOYMENT_MODE,
      serverType: data.server_type
        ? (data.server_type as ServerType)
        : DEFAULT_SERVER_TYPE,
      framework: data.framework ?? "electron",
    },
    output: {
      locationMode: data.location_mode
        ? (data.location_mode as OutputLocation)
        : "proper",
      outputPath: data.output_path ?? "",
    },
    platforms: data.platforms ?? { win: true, mac: true, linux: true },
    connection: {
      proxyUrl: data.proxy_url ?? "",
      bundleManifestPath: data.bundle_manifest_path ?? "",
      serverPort: data.server_port ?? 3000,
      localServerPath: data.local_server_path ?? "ui/server.js",
      localApiEndpoint: data.local_api_endpoint ?? DEFAULT_LOCAL_API_ENDPOINT,
      autoManageTier1: data.auto_manage_tier1 ?? false,
      vrooliBinaryPath: data.vrooli_binary_path ?? "vrooli",
      connectionResult: data.connection_result ?? null,
      connectionError: data.connection_error ?? null,
    },
    signingEnabledForBuild: data.signing_enabled_for_build ?? false,
  };
}
