/**
 * Hook for form state management.
 * Wraps the formStore and provides integration with scenarios.
 * This is the primary hook for managing form state in the Generator page.
 */

import { useCallback, useEffect, useMemo } from "react";
import { useFormStore } from "../store/formStore";
import {
  selectConnectionDecision,
  selectStandardOutputPath,
  selectStagingPreviewPath,
} from "../store/formSelectors";
import {
  DEFAULT_SERVER_TYPE,
  SERVER_TYPE_OPTIONS,
  decideConnection,
  type DeploymentMode,
  type ServerType,
} from "../domain/deployment";
import {
  getSelectedPlatforms,
  type DesktopFramework,
} from "../domain/generator";
import { getIconPreviewUrl } from "../lib/api";
import type {
  DesktopConnectionConfig,
  ScenarioDesktopStatus,
} from "../components/scenario-inventory/types";
import type {
  HydrateFormData,
  ValidationError,
  AppMetadataState,
  DeploymentState,
  OutputState,
  PlatformsState,
  ConnectionState,
} from "../store/formTypes";

// ============================================================================
// Types
// ============================================================================

export interface UseFormStateProps {
  scenarioName: string;
  selectionSource?: "inventory" | "manual" | null;
}

export interface UseFormStateReturn {
  // App metadata
  appMetadata: AppMetadataState;
  setAppDisplayName: (value: string) => void;
  setAppDescription: (value: string) => void;
  setIconPath: (value: string) => void;
  setIconPreviewError: (value: boolean) => void;

  // Deployment
  deployment: DeploymentState;
  setDeploymentMode: (mode: DeploymentMode) => void;
  setServerType: (type: ServerType) => void;
  setFramework: (framework: DesktopFramework) => void;
  handleDeploymentChange: (nextMode: DeploymentMode) => void;

  // Output
  output: OutputState;
  setLocationMode: (mode: "proper" | "temp" | "custom") => void;
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
  setConnectionResult: (result: ConnectionState["connectionResult"]) => void;
  setConnectionError: (error: string | null) => void;

  // Signing
  signingEnabledForBuild: boolean;
  setSigningEnabledForBuild: (enabled: boolean) => void;

  // Template
  selectedTemplate: string;
  setSelectedTemplate: (template: string) => void;

  // Validation
  validationErrors: ValidationError[];
  setValidationErrors: (errors: ValidationError[]) => void;
  clearValidationErrors: () => void;

  // UI state
  scenarioLocked: boolean;
  setScenarioLocked: (locked: boolean) => void;

  // Derived values
  connectionDecision: ReturnType<typeof selectConnectionDecision>;
  isBundled: boolean;
  requiresRemoteConfig: boolean;
  allowedServerTypes: ServerType[];
  selectedPlatformsList: string[];
  standardOutputPath: string;
  stagingPreviewPath: string;
  iconPreviewUrl: string;
  isCustomLocation: boolean;
  isUpdateMode: boolean;

  // Reset
  resetFormState: (resetTemplate?: boolean) => void;

  // Hydration
  hydrateFromServer: (data: HydrateFormData) => void;

  // Apply saved connection
  applySavedConnection: (config?: DesktopConnectionConfig | null) => void;

  // Apply scenario defaults
  applyScenarioDefaults: (scenario: ScenarioDesktopStatus) => void;
}

// ============================================================================
// Hook Implementation
// ============================================================================

export function useFormState(props: UseFormStateProps): UseFormStateReturn {
  const { scenarioName, selectionSource } = props;

  // ========== Store State ==========
  const store = useFormStore();
  const {
    appMetadata,
    deployment,
    output,
    platforms,
    connection,
    signingEnabledForBuild,
    selectedTemplate,
    validationErrors,
    scenarioLocked,
    setAppDisplayName,
    setAppDescription,
    setIconPath,
    setIconPreviewError,
    setDeploymentMode,
    setServerType,
    setFramework,
    setLocationMode,
    setOutputPath,
    setPlatforms,
    handlePlatformChange,
    setProxyUrl,
    setBundleManifestPath,
    setServerPort,
    setLocalServerPath,
    setLocalApiEndpoint,
    setAutoManageTier1,
    setVrooliBinaryPath,
    setConnectionResult,
    setConnectionError,
    setSigningEnabledForBuild,
    setSelectedTemplate,
    setValidationErrors,
    clearValidationErrors,
    setScenarioLocked,
    resetFormState: resetFormStateStore,
    hydrateFromServer,
  } = store;

  // ========== Derived Values ==========
  const connectionDecision = useMemo(
    () => decideConnection(deployment.mode, deployment.serverType),
    [deployment.mode, deployment.serverType],
  );
  const isBundled = connectionDecision.kind === "bundled-runtime";
  const requiresRemoteConfig = connectionDecision.requiresProxyUrl;
  const isUpdateMode = selectionSource === "inventory";

  const allowedServerTypes = useMemo<ServerType[]>(() => {
    const { mode } = deployment;
    if (mode === "bundled" || mode === "cloud-api") {
      return ["external"];
    }
    return SERVER_TYPE_OPTIONS.map((option) => option.value);
  }, [deployment]);

  const selectedPlatformsList = useMemo(
    () => getSelectedPlatforms(platforms),
    [platforms],
  );

  const standardOutputPath = useMemo(
    () => selectStandardOutputPath(scenarioName),
    [scenarioName],
  );

  const stagingPreviewPath = useMemo(
    () => selectStagingPreviewPath(scenarioName),
    [scenarioName],
  );

  const iconPreviewUrl = useMemo(
    () => (appMetadata.iconPath ? getIconPreviewUrl(appMetadata.iconPath) : ""),
    [appMetadata.iconPath],
  );

  const isCustomLocation = output.locationMode === "custom";

  // ========== Effects ==========

  // Sync scenario locked state with selection source
  useEffect(() => {
    setScenarioLocked(selectionSource === "inventory");
  }, [selectionSource, setScenarioLocked]);

  // Reset icon preview error on URL change
  useEffect(() => {
    setIconPreviewError(false);
  }, [iconPreviewUrl, setIconPreviewError]);

  // Adjust server type when not allowed
  useEffect(() => {
    if (!allowedServerTypes.includes(deployment.serverType)) {
      setServerType(allowedServerTypes[0] ?? DEFAULT_SERVER_TYPE);
    }
  }, [allowedServerTypes, deployment.serverType, setServerType]);

  // Sync connection decision effects
  useEffect(() => {
    if (connectionDecision.kind === "bundled-runtime") {
      if (deployment.serverType !== connectionDecision.effectiveServerType) {
        setServerType(connectionDecision.effectiveServerType);
      }
      if (
        !connectionDecision.allowsAutoManageTier1 &&
        connection.autoManageTier1
      ) {
        setAutoManageTier1(false);
      }
    }
  }, [
    connectionDecision,
    deployment.serverType,
    connection.autoManageTier1,
    setServerType,
    setAutoManageTier1,
  ]);

  // ========== Handlers ==========

  const handleDeploymentChange = useCallback(
    (nextMode: DeploymentMode) => {
      setDeploymentMode(nextMode);
      const nextAllowed: ServerType[] =
        nextMode === "bundled" || nextMode === "cloud-api"
          ? ["external"]
          : SERVER_TYPE_OPTIONS.map((option) => option.value);
      if (!nextAllowed.includes(deployment.serverType)) {
        setServerType(nextAllowed[0] ?? DEFAULT_SERVER_TYPE);
      }
    },
    [deployment.serverType, setDeploymentMode, setServerType],
  );

  const resetFormState = useCallback(
    (resetTemplate = true) => {
      resetFormStateStore();
      if (resetTemplate) {
        setSelectedTemplate("basic");
      }
    },
    [resetFormStateStore, setSelectedTemplate],
  );

  const applySavedConnection = useCallback(
    (config?: DesktopConnectionConfig | null) => {
      if (!config) return;
      setDeploymentMode(config.deployment_mode as DeploymentMode);
      setProxyUrl(config.proxy_url ?? config.server_url ?? "");
      setAutoManageTier1(config.auto_manage_vrooli ?? false);
      setVrooliBinaryPath(config.vrooli_binary_path ?? "vrooli");
      setBundleManifestPath(config.bundle_manifest_path ?? "");
      if (config.app_display_name) setAppDisplayName(config.app_display_name);
      if (config.app_description) setAppDescription(config.app_description);
      if (config.icon) setIconPath(config.icon);
      if (config.server_type) setServerType(config.server_type as ServerType);
    },
    [
      setDeploymentMode,
      setProxyUrl,
      setAutoManageTier1,
      setVrooliBinaryPath,
      setBundleManifestPath,
      setAppDisplayName,
      setAppDescription,
      setIconPath,
      setServerType,
    ],
  );

  const applyScenarioDefaults = useCallback(
    (scenario: ScenarioDesktopStatus) => {
      if (!appMetadata.displayNameEdited) {
        setAppDisplayName(scenario.service_display_name || "");
      }
      if (!appMetadata.descriptionEdited) {
        setAppDescription(scenario.service_description || "");
      }
      if (!appMetadata.iconPathEdited) {
        setIconPath(scenario.service_icon_path || "");
      }
    },
    [
      appMetadata.displayNameEdited,
      appMetadata.descriptionEdited,
      appMetadata.iconPathEdited,
      setAppDisplayName,
      setAppDescription,
      setIconPath,
    ],
  );

  // ========== Return ==========

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
    handleDeploymentChange,

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

    // Signing
    signingEnabledForBuild,
    setSigningEnabledForBuild,

    // Template
    selectedTemplate,
    setSelectedTemplate,

    // Validation
    validationErrors,
    setValidationErrors,
    clearValidationErrors,

    // UI state
    scenarioLocked,
    setScenarioLocked,

    // Derived values
    connectionDecision,
    isBundled,
    requiresRemoteConfig,
    allowedServerTypes,
    selectedPlatformsList,
    standardOutputPath,
    stagingPreviewPath,
    iconPreviewUrl,
    isCustomLocation,
    isUpdateMode,

    // Reset
    resetFormState,

    // Hydration
    hydrateFromServer,

    // Apply saved connection
    applySavedConnection,

    // Apply scenario defaults
    applyScenarioDefaults,
  };
}
