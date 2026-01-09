/**
 * Pure domain functions for desktop generator validation and configuration.
 * These functions have no side effects and can be tested in isolation.
 */

import type { ConnectionDecision, DeploymentMode, ServerType } from "./deployment";
import type { DesktopConfig, SigningConfig } from "../lib/api";

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

export interface ValidateGeneratorInputsOptions {
  selectedPlatforms: string[];
  decision: ConnectionDecision;
  bundleManifestPath: string;
  proxyUrl: string;
  appDisplayName: string;
  appDescription: string;
  locationMode: OutputLocation;
  outputPath: string;
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
 * Validate generator form inputs before submission.
 * Returns an error message string if validation fails, null if valid.
 */
export function validateGeneratorInputs(options: ValidateGeneratorInputsOptions): string | null {
  if (options.selectedPlatforms.length === 0) {
    return "Please select at least one target platform";
  }

  if (!options.appDisplayName.trim()) {
    return "Provide an app display name so installers and windows are branded correctly.";
  }

  if (!options.appDescription.trim()) {
    return "Provide a short description for the generated desktop app.";
  }

  if (options.decision.requiresBundleManifest && !options.bundleManifestPath) {
    return "Provide bundle_manifest_path from deployment-manager before generating a bundled build.";
  }

  if (options.decision.requiresProxyUrl && !options.proxyUrl) {
    return "Provide the proxy URL you use in the browser (for example https://app-monitor.example.com/apps/<scenario>/proxy/).";
  }

  if (options.locationMode === "custom" && !options.outputPath.trim()) {
    return "Provide an output path when choosing a custom location.";
  }

  return null;
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
    ? `scenarios/scenario-to-desktop/data/staging/${scenarioName}/<build-id>`
    : "scenarios/scenario-to-desktop/data/staging/<scenario>/<build-id>";
}
