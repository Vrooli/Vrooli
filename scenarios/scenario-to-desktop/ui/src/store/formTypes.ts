/**
 * Form store type definitions.
 * Separated from formStore.ts for modularity.
 */

import { resolveApiBase } from "@vrooli/api-base";
import type { ProbeResponse } from "../lib/api";
import type {
  DeploymentMode,
  ServerType,
} from "../domain/deployment";
import type { BundleResult } from "../components/runtime/DeploymentManagerBundleHelper";
// Use the correct output location type from domain
export type OutputLocation = "proper" | "temp" | "custom";
// Import ValidationError from domain - single source of truth
import type { PlatformSelection, ValidationError } from "../domain/generator";

// Re-export for consumers
export type { ValidationError };

// ============================================================================
// State Types
// ============================================================================

/** App metadata state group */
export interface AppMetadataState {
  displayName: string;
  description: string;
  iconPath: string;
  /** Track if user manually edited display name */
  displayNameEdited: boolean;
  /** Track if user manually edited description */
  descriptionEdited: boolean;
  /** Track if user manually edited icon path */
  iconPathEdited: boolean;
  /** Icon preview error state */
  iconPreviewError: boolean;
}

/** Deployment configuration state group */
export interface DeploymentState {
  mode: DeploymentMode;
  serverType: ServerType;
  framework: string;
}

/** Output location state group */
export interface OutputState {
  locationMode: OutputLocation;
  outputPath: string;
}

/** Platform selection state */
export type PlatformsState = PlatformSelection;

/** Connection configuration state group */
export interface ConnectionState {
  proxyUrl: string;
  bundleManifestPath: string;
  serverPort: number;
  localServerPath: string;
  localApiEndpoint: string;
  autoManageTier1: boolean;
  vrooliBinaryPath: string;
  connectionResult: ProbeResponse | null;
  connectionError: string | null;
}

/** Complete form store state */
export interface FormStoreState {
  // App metadata
  appMetadata: AppMetadataState;

  // Deployment configuration
  deployment: DeploymentState;

  // Output location
  output: OutputState;

  // Platform selection
  platforms: PlatformsState;

  // Connection configuration
  connection: ConnectionState;

  // Signing state (tracked separately from signing config)
  signingEnabledForBuild: boolean;

  // Template selection
  selectedTemplate: string;

  // Validation errors
  validationErrors: ValidationError[];

  // UI state
  scenarioLocked: boolean;

  // Bundle result seed (persisted stage result)
  bundleResultSeed: BundleResult | null;
}

// ValidationError type imported from domain/generator.ts and re-exported above

/** Form store actions */
export interface FormStoreActions {
  // App metadata setters
  setAppDisplayName: (value: string) => void;
  setAppDescription: (value: string) => void;
  setIconPath: (value: string) => void;
  setIconPreviewError: (value: boolean) => void;

  // Deployment setters
  setDeploymentMode: (mode: DeploymentMode) => void;
  setServerType: (type: ServerType) => void;
  setFramework: (framework: string) => void;

  // Output setters
  setLocationMode: (mode: OutputLocation) => void;
  setOutputPath: (path: string) => void;

  // Platform setters
  setPlatforms: (platforms: PlatformsState) => void;
  handlePlatformChange: (platform: string, checked: boolean) => void;

  // Connection setters
  setProxyUrl: (url: string) => void;
  setBundleManifestPath: (path: string) => void;
  setServerPort: (port: number) => void;
  setLocalServerPath: (path: string) => void;
  setLocalApiEndpoint: (endpoint: string) => void;
  setAutoManageTier1: (auto: boolean) => void;
  setVrooliBinaryPath: (path: string) => void;
  setConnectionResult: (result: ProbeResponse | null) => void;
  setConnectionError: (error: string | null) => void;

  // Signing
  setSigningEnabledForBuild: (enabled: boolean) => void;

  // Template
  setSelectedTemplate: (template: string) => void;

  // Validation
  setValidationErrors: (errors: ValidationError[]) => void;
  clearValidationErrors: () => void;

  // UI state
  setScenarioLocked: (locked: boolean) => void;

  // Bundle result seed
  setBundleResultSeed: (result: BundleResult | null) => void;

  // Reset
  resetFormState: () => void;

  // Hydration from server
  hydrateFromServer: (data: HydrateFormData) => void;
}

/** Data structure for hydrating form from server */
export interface HydrateFormData {
  appMetadata?: Partial<AppMetadataState>;
  deployment?: Partial<DeploymentState>;
  output?: Partial<OutputState>;
  platforms?: PlatformsState;
  connection?: Partial<ConnectionState>;
  signingEnabledForBuild?: boolean;
  selectedTemplate?: string;
  bundleResultSeed?: BundleResult | null;
}

export type FormStore = FormStoreState & FormStoreActions;

// ============================================================================
// Default Values
// ============================================================================

export const defaultAppMetadata: AppMetadataState = {
  displayName: "",
  description: "",
  iconPath: "",
  displayNameEdited: false,
  descriptionEdited: false,
  iconPathEdited: false,
  iconPreviewError: false,
};

export const defaultDeployment: DeploymentState = {
  mode: "bundled",
  serverType: "external",
  framework: "electron",
};

export const defaultOutput: OutputState = {
  locationMode: "proper",
  outputPath: "",
};

export const defaultPlatforms: PlatformsState = {
  win: true,
  mac: true,
  linux: true,
};

/** Default local API endpoint, resolved via @vrooli/api-base with env-var override. */
export const DEFAULT_LOCAL_API_ENDPOINT: string =
  (import.meta.env.VITE_LOCAL_API_ENDPOINT as string | undefined) ?? resolveApiBase({ appendSuffix: true });

export const defaultConnection: ConnectionState = {
  proxyUrl: "",
  bundleManifestPath: "",
  serverPort: 3000,
  localServerPath: "ui/server.js",
  localApiEndpoint: DEFAULT_LOCAL_API_ENDPOINT,
  autoManageTier1: false,
  vrooliBinaryPath: "vrooli",
  connectionResult: null,
  connectionError: null,
};

export const initialFormState: FormStoreState = {
  appMetadata: defaultAppMetadata,
  deployment: defaultDeployment,
  output: defaultOutput,
  platforms: defaultPlatforms,
  connection: defaultConnection,
  signingEnabledForBuild: false,
  selectedTemplate: "basic",
  validationErrors: [],
  scenarioLocked: false,
  bundleResultSeed: null,
};
