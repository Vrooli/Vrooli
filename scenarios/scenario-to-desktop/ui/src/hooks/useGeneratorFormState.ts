/**
 * Hook for managing Generator form state.
 * Groups related useState calls into logical clusters.
 */

import { useCallback, useState } from "react";
import type { ProbeResponse } from "../lib/api";
import {
  DEFAULT_DEPLOYMENT_MODE,
  DEFAULT_SERVER_TYPE,
  type DeploymentMode,
  type ServerType,
} from "../domain/deployment";
import type { OutputLocation, PlatformSelection } from "../domain/generator";

// ============================================================================
// Types
// ============================================================================

export interface AppMetadata {
  displayName: string;
  description: string;
  iconPath: string;
  displayNameEdited: boolean;
  descriptionEdited: boolean;
  iconPathEdited: boolean;
  iconPreviewError: boolean;
}

export interface DeploymentState {
  mode: DeploymentMode;
  serverType: ServerType;
  framework: string;
}

export interface OutputState {
  locationMode: OutputLocation;
  outputPath: string;
}

export interface PlatformsState extends PlatformSelection {}

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

export interface UseGeneratorFormStateReturn {
  // App metadata
  appMetadata: AppMetadata;
  setAppDisplayName: (value: string) => void;
  setAppDescription: (value: string) => void;
  setIconPath: (value: string) => void;
  setIconPreviewError: (value: boolean) => void;

  // Deployment
  deployment: DeploymentState;
  setDeploymentMode: (mode: DeploymentMode) => void;
  setServerType: (type: ServerType) => void;
  setFramework: (framework: string) => void;

  // Output
  output: OutputState;
  setLocationMode: (mode: OutputLocation) => void;
  setOutputPath: (path: string) => void;

  // Platforms
  platforms: PlatformsState;
  setPlatforms: (platforms: PlatformsState) => void;
  handlePlatformChange: (platform: string, checked: boolean) => void;

  // Connection
  connection: ConnectionState;
  setProxyUrl: (url: string) => void;
  setBundleManifestPath: (path: string) => void;
  setServerPort: (port: number) => void;
  setLocalServerPath: (path: string) => void;
  setLocalApiEndpoint: (endpoint: string) => void;
  setAutoManageTier1: (auto: boolean) => void;
  setVrooliBinaryPath: (path: string) => void;
  setConnectionResult: (result: ProbeResponse | null) => void;
  setConnectionError: (error: string | null) => void;

  // Reset
  resetFormState: () => void;

  // Hydration
  hydrateFromServer: (data: {
    appMetadata?: Partial<AppMetadata>;
    deployment?: Partial<DeploymentState>;
    output?: Partial<OutputState>;
    platforms?: PlatformsState;
    connection?: Partial<ConnectionState>;
  }) => void;
}

// ============================================================================
// Default Values
// ============================================================================

const defaultAppMetadata: AppMetadata = {
  displayName: "",
  description: "",
  iconPath: "",
  displayNameEdited: false,
  descriptionEdited: false,
  iconPathEdited: false,
  iconPreviewError: false,
};

const defaultDeployment: DeploymentState = {
  mode: DEFAULT_DEPLOYMENT_MODE,
  serverType: DEFAULT_SERVER_TYPE,
  framework: "electron",
};

const defaultOutput: OutputState = {
  locationMode: "proper",
  outputPath: "",
};

const defaultPlatforms: PlatformsState = {
  win: true,
  mac: true,
  linux: true,
};

const defaultConnection: ConnectionState = {
  proxyUrl: "",
  bundleManifestPath: "",
  serverPort: 3000,
  localServerPath: "ui/server.js",
  localApiEndpoint: "http://localhost:3001/api",
  autoManageTier1: false,
  vrooliBinaryPath: "vrooli",
  connectionResult: null,
  connectionError: null,
};

// ============================================================================
// Hook
// ============================================================================

/**
 * Hook for managing all generator form state.
 * Organizes state into logical groups for better maintainability.
 */
export function useGeneratorFormState(): UseGeneratorFormStateReturn {
  // App metadata state
  const [appMetadata, setAppMetadataState] = useState<AppMetadata>(defaultAppMetadata);

  // Deployment state
  const [deployment, setDeploymentState] = useState<DeploymentState>(defaultDeployment);

  // Output state
  const [output, setOutputState] = useState<OutputState>(defaultOutput);

  // Platforms state
  const [platforms, setPlatformsState] = useState<PlatformsState>(defaultPlatforms);

  // Connection state
  const [connection, setConnectionState] = useState<ConnectionState>(defaultConnection);

  // App metadata setters
  const setAppDisplayName = useCallback((value: string) => {
    setAppMetadataState((prev) => ({
      ...prev,
      displayName: value,
      displayNameEdited: true,
    }));
  }, []);

  const setAppDescription = useCallback((value: string) => {
    setAppMetadataState((prev) => ({
      ...prev,
      description: value,
      descriptionEdited: true,
    }));
  }, []);

  const setIconPath = useCallback((value: string) => {
    setAppMetadataState((prev) => ({
      ...prev,
      iconPath: value,
      iconPathEdited: true,
    }));
  }, []);

  const setIconPreviewError = useCallback((value: boolean) => {
    setAppMetadataState((prev) => ({ ...prev, iconPreviewError: value }));
  }, []);

  // Deployment setters
  const setDeploymentMode = useCallback((mode: DeploymentMode) => {
    setDeploymentState((prev) => ({ ...prev, mode }));
  }, []);

  const setServerType = useCallback((serverType: ServerType) => {
    setDeploymentState((prev) => ({ ...prev, serverType }));
  }, []);

  const setFramework = useCallback((framework: string) => {
    setDeploymentState((prev) => ({ ...prev, framework }));
  }, []);

  // Output setters
  const setLocationMode = useCallback((locationMode: OutputLocation) => {
    setOutputState((prev) => ({ ...prev, locationMode }));
  }, []);

  const setOutputPath = useCallback((outputPath: string) => {
    setOutputState((prev) => ({ ...prev, outputPath }));
  }, []);

  // Platform setters
  const setPlatforms = useCallback((platforms: PlatformsState) => {
    setPlatformsState(platforms);
  }, []);

  const handlePlatformChange = useCallback((platform: string, checked: boolean) => {
    setPlatformsState((prev) => ({ ...prev, [platform]: checked }));
  }, []);

  // Connection setters
  const setProxyUrl = useCallback((proxyUrl: string) => {
    setConnectionState((prev) => ({ ...prev, proxyUrl }));
  }, []);

  const setBundleManifestPath = useCallback((bundleManifestPath: string) => {
    setConnectionState((prev) => ({ ...prev, bundleManifestPath }));
  }, []);

  const setServerPort = useCallback((serverPort: number) => {
    setConnectionState((prev) => ({ ...prev, serverPort }));
  }, []);

  const setLocalServerPath = useCallback((localServerPath: string) => {
    setConnectionState((prev) => ({ ...prev, localServerPath }));
  }, []);

  const setLocalApiEndpoint = useCallback((localApiEndpoint: string) => {
    setConnectionState((prev) => ({ ...prev, localApiEndpoint }));
  }, []);

  const setAutoManageTier1 = useCallback((autoManageTier1: boolean) => {
    setConnectionState((prev) => ({ ...prev, autoManageTier1 }));
  }, []);

  const setVrooliBinaryPath = useCallback((vrooliBinaryPath: string) => {
    setConnectionState((prev) => ({ ...prev, vrooliBinaryPath }));
  }, []);

  const setConnectionResult = useCallback((connectionResult: ProbeResponse | null) => {
    setConnectionState((prev) => ({ ...prev, connectionResult }));
  }, []);

  const setConnectionError = useCallback((connectionError: string | null) => {
    setConnectionState((prev) => ({ ...prev, connectionError }));
  }, []);

  // Reset all state
  const resetFormState = useCallback(() => {
    setAppMetadataState(defaultAppMetadata);
    setDeploymentState(defaultDeployment);
    setOutputState(defaultOutput);
    setPlatformsState(defaultPlatforms);
    setConnectionState(defaultConnection);
  }, []);

  // Hydrate state from server (preserves edited flags)
  const hydrateFromServer = useCallback((data: {
    appMetadata?: Partial<AppMetadata>;
    deployment?: Partial<DeploymentState>;
    output?: Partial<OutputState>;
    platforms?: PlatformsState;
    connection?: Partial<ConnectionState>;
  }) => {
    if (data.appMetadata) {
      setAppMetadataState((prev) => ({ ...prev, ...data.appMetadata }));
    }
    if (data.deployment) {
      setDeploymentState((prev) => ({ ...prev, ...data.deployment }));
    }
    if (data.output) {
      setOutputState((prev) => ({ ...prev, ...data.output }));
    }
    if (data.platforms) {
      setPlatformsState(data.platforms);
    }
    if (data.connection) {
      setConnectionState((prev) => ({ ...prev, ...data.connection }));
    }
  }, []);

  return {
    // App metadata
    appMetadata,
    setAppDisplayName,
    setAppDescription,
    setIconPath,
    setIconPreviewError,

    // Deployment
    deployment,
    setDeploymentMode,
    setServerType,
    setFramework,

    // Output
    output,
    setLocationMode,
    setOutputPath,

    // Platforms
    platforms,
    setPlatforms,
    handlePlatformChange,

    // Connection
    connection,
    setProxyUrl,
    setBundleManifestPath,
    setServerPort,
    setLocalServerPath,
    setLocalApiEndpoint,
    setAutoManageTier1,
    setVrooliBinaryPath,
    setConnectionResult,
    setConnectionError,

    // Reset
    resetFormState,

    // Hydration
    hydrateFromServer,
  };
}
