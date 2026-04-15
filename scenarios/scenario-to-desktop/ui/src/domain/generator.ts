/**
 * Pure domain functions for desktop generator validation and configuration.
 * These functions have no side effects and can be tested in isolation.
 */

import type { ConnectionDecision, DeploymentMode, ServerType } from "./deployment";
import type { DesktopConfig, SigningConfig, BundlePreflightResponse } from "../lib/api";

export type PlatformSelection = {
  win: boolean;
  mac: boolean;
  linux: boolean;
};

export type EndpointResolution = {
  serverPath: string;
  apiEndpoint: string;
};

export type OutputLocation = "proper" | "temp" | "custom";

export interface BuildDesktopConfigOptions {
  scenarioName: string;
  appDisplayName: string;
  appDescription: string;
  iconPath: string;
  selectedTemplate: string;
  framework: string;
  serverType: ServerType;
  serverPort: number;
  outputPath: string;
  selectedPlatforms: string[];
  deploymentMode: DeploymentMode;
  autoManageTier1: boolean;
  vrooliBinaryPath: string;
  proxyUrl: string;
  bundleManifestPath: string;
  isBundled: boolean;
  requiresRemoteConfig: boolean;
  resolvedEndpoints: EndpointResolution;
  locationMode: OutputLocation;
  includeSigning: boolean;
  codeSigning?: SigningConfig;
}

/**
 * Validation error with field association for UI feedback.
 * Part of the domain layer - consumed by presentation components.
 */
export interface ValidationError {
  id: string;
  message: string;
  field?: string;
}

/**
 * Parameters for comprehensive form validation.
 */
export interface ValidateFormInputsParams {
  scenarioName: string;
  selectedPlatforms: string[];
  isBundled: boolean;
  requiresProxyUrl: boolean;
  bundleManifestPath: string;
  proxyUrl: string;
  appDisplayName: string;
  appDescription: string;
  locationMode: string;
  outputPath: string;
  preflightResult: BundlePreflightResponse | null | undefined;
  preflightOk: boolean;
  preflightOverride: boolean;
  signingEnabledForBuild: boolean;
  signingConfig: { enabled?: boolean } | null | undefined;
  signingReadiness: { ready?: boolean; issues?: string[] } | undefined;
}

export const TEMPLATE_SUMMARIES: Record<string, { name: string; description: string }> = {
  basic: { name: "Basic", description: "Balanced single window wrapper" },
  advanced: { name: "Advanced", description: "Tray, shortcuts, deep OS touches" },
  multi_window: { name: "Multi-Window", description: "Multiple coordinated windows" },
  kiosk: { name: "Kiosk Mode", description: "Locked-down fullscreen kiosk" },
  universal: { name: "Universal Desktop App", description: "All-purpose desktop wrapper" }
};

export const FRAMEWORK_SUMMARIES: Record<string, { name: string; description: string }> = {
  electron: { name: "Electron", description: "Most compatible and battle-tested for desktop web apps" },
  tauri: { name: "Tauri", description: "Rust + system webview for smaller, more secure apps" },
  neutralino: { name: "Neutralino", description: "Ultra-lightweight desktop wrapper with minimal runtime" }
};

/**
 * Convert platform selection object to array of platform strings.
 */
export function getSelectedPlatforms(platforms: PlatformSelection): string[] {
  return Object.entries(platforms)
    .filter(([, enabled]) => enabled)
    .map(([platform]) => platform);
}

/**
 * Comprehensive form validation with field associations.
 * Returns an array of ValidationError objects for UI feedback.
 * This is the canonical validation function - presentation layer components should use this.
 *
 * ASSUMPTION: params is a valid object with all required fields. TypeScript ensures this at compile time,
 * but runtime guards are added for defensive programming against invalid data from API responses or JSON parsing.
 */
export function validateFormInputs(params: ValidateFormInputsParams): ValidationError[] {
  const errors: ValidationError[] = [];

  // Defensive guard: ensure params is valid
  if (!params || typeof params !== "object") {
    return [{ id: "invalid-params", message: "Internal error: invalid form parameters", field: undefined }];
  }

  // Scenario selection
  if (!params.scenarioName) {
    errors.push({
      id: "no-scenario",
      message: "Please select a scenario before generating a desktop app.",
      field: "scenarioName",
    });
  }

  // Platform selection - guard against undefined/null array
  const platforms = params.selectedPlatforms ?? [];
  if (!Array.isArray(platforms) || platforms.length === 0) {
    errors.push({
      id: "no-platforms",
      message: "Select at least one target platform (Windows, macOS, or Linux).",
      field: "platforms",
    });
  }

  // Bundled mode requirements
  if (params.isBundled) {
    // Guard against undefined bundleManifestPath
    const manifestPath = params.bundleManifestPath ?? "";
    if (typeof manifestPath !== "string" || !manifestPath.trim()) {
      errors.push({
        id: "no-bundle-manifest",
        message: "Provide a bundle manifest path for bundled runtime mode.",
        field: "bundleManifestPath",
      });
    }

    if (!params.preflightResult) {
      errors.push({
        id: "no-preflight",
        message: "Run preflight validation before generating a bundled desktop app.",
        field: "preflight",
      });
    } else if (!params.preflightOk && !params.preflightOverride) {
      errors.push({
        id: "preflight-failed",
        message: "Preflight validation failed. Fix the issues or enable override to continue.",
        field: "preflight",
      });
    }
  }

  // Remote server requirements - guard against undefined proxyUrl
  const proxyUrl = params.proxyUrl ?? "";
  if (params.requiresProxyUrl && (typeof proxyUrl !== "string" || !proxyUrl.trim())) {
    errors.push({
      id: "no-proxy-url",
      message: "Provide a proxy URL for remote server mode.",
      field: "proxyUrl",
    });
  }

  // App metadata - guard against undefined/null values
  const displayName = params.appDisplayName ?? "";
  if (typeof displayName !== "string" || !displayName.trim()) {
    errors.push({
      id: "no-display-name",
      message: "Provide an app display name.",
      field: "appDisplayName",
    });
  }

  const description = params.appDescription ?? "";
  if (typeof description !== "string" || !description.trim()) {
    errors.push({
      id: "no-description",
      message: "Provide an app description.",
      field: "appDescription",
    });
  }

  // Custom output path - guard against undefined/null values
  const outputPath = params.outputPath ?? "";
  if (params.locationMode === "custom" && (typeof outputPath !== "string" || !outputPath.trim())) {
    errors.push({
      id: "no-output-path",
      message: "Provide an output path when using custom location mode.",
      field: "outputPath",
    });
  }

  // Signing
  if (params.signingEnabledForBuild) {
    if (!params.signingConfig || !params.signingConfig.enabled) {
      errors.push({
        id: "no-signing-config",
        message: "Signing is enabled but no signing config is saved. Open the Signing tab to add certificates.",
        field: "signing",
      });
    } else if (params.signingReadiness && !params.signingReadiness.ready) {
      const issue = params.signingReadiness.issues?.[0] || "Signing prerequisites not met.";
      errors.push({
        id: "signing-not-ready",
        message: `Signing is not ready: ${issue}`,
        field: "signing",
      });
    }
  }

  return errors;
}

/**
 * Resolve server endpoints based on connection decision and user inputs.
 */
export function resolveEndpoints(input: {
  decision: ConnectionDecision;
  proxyUrl: string;
  localServerPath: string;
  localApiEndpoint: string;
}): EndpointResolution {
  if (input.decision.kind === "bundled-runtime") {
    return { serverPath: "http://127.0.0.1", apiEndpoint: "http://127.0.0.1" };
  }
  if (input.decision.requiresProxyUrl) {
    return { serverPath: input.proxyUrl, apiEndpoint: input.proxyUrl };
  }
  return { serverPath: input.localServerPath, apiEndpoint: input.localApiEndpoint };
}

/**
 * Build a DesktopConfig object from form inputs.
 */
export function buildDesktopConfig(options: BuildDesktopConfigOptions): DesktopConfig {
  return {
    app_name: options.scenarioName,
    app_display_name: options.appDisplayName,
    app_description: options.appDescription,
    version: "1.0.0",
    author: "Vrooli Platform",
    license: "MIT",
    app_id: `com.vrooli.${options.scenarioName.replace(/-/g, ".")}`,
    icon: options.iconPath || undefined,
    server_type: options.serverType,
    server_port: options.serverPort,
    server_path: options.resolvedEndpoints.serverPath,
    api_endpoint: options.resolvedEndpoints.apiEndpoint,
    framework: options.framework,
    template_type: options.selectedTemplate,
    platforms: options.selectedPlatforms,
    output_path: options.outputPath,
    location_mode: options.locationMode,
    features: {
      splash: true,
      autoUpdater: true,
      devTools: true
    },
    window: {
      width: 1200,
      height: 800,
      background: "#f5f5f5"
    },
    deployment_mode: options.deploymentMode,
    auto_manage_vrooli: options.autoManageTier1,
    vrooli_binary_path: options.vrooliBinaryPath,
    proxy_url: options.requiresRemoteConfig ? options.proxyUrl : undefined,
    external_server_url: options.requiresRemoteConfig ? options.proxyUrl : undefined,
    external_api_url: !options.requiresRemoteConfig && !options.isBundled ? options.resolvedEndpoints.apiEndpoint : undefined,
    bundle_manifest_path: options.isBundled ? options.bundleManifestPath : undefined,
    code_signing: options.includeSigning ? options.codeSigning : { enabled: false }
  };
}

/**
 * Compute standard output path for a scenario.
 */
export function computeStandardOutputPath(scenarioName: string): string {
  return scenarioName
    ? `scenarios/${scenarioName}/platforms/electron`
    : "scenarios/<scenario>/platforms/electron";
}

/**
 * Compute staging preview path for a scenario.
 */
export function computeStagingPreviewPath(scenarioName: string): string {
  return scenarioName
    ? `<cache-root>/vrooli/scenario-to-desktop/staging/${scenarioName}/<build-id>`
    : "<cache-root>/vrooli/scenario-to-desktop/staging/<scenario>/<build-id>";
}
