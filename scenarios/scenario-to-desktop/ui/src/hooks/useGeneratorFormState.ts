/**
 * Centralized form state management for the generator form.
 * Uses useReducer for predictable state updates and easier testing.
 */

import { useCallback, useReducer } from "react";
import type { ProbeResponse } from "../lib/api";
import type { DeploymentMode, ServerType } from "../domain/deployment";
import type { OutputLocation, PlatformSelection } from "../domain/generator";

// ============================================================================
// Types
// ============================================================================

export interface GeneratorFormState {
  // App metadata
  appDisplayName: string;
  appDescription: string;
  iconPath: string;
  displayNameEdited: boolean;
  descriptionEdited: boolean;
  iconPathEdited: boolean;

  // Framework and template
  framework: string;

  // Server configuration
  serverType: ServerType;
  deploymentMode: DeploymentMode;
  proxyUrl: string;
  serverPort: number;
  localServerPath: string;
  localApiEndpoint: string;
  autoManageTier1: boolean;
  vrooliBinaryPath: string;

  // Bundle configuration
  bundleManifestPath: string;

  // Platform and output
  platforms: PlatformSelection;
  locationMode: OutputLocation;
  outputPath: string;

  // Connection state
  connectionResult: ProbeResponse | null;
  connectionError: string | null;

  // Deployment
  deploymentManagerUrl: string | null;

  // Signing
  signingEnabledForBuild: boolean;
}

export const DEFAULT_FORM_STATE: GeneratorFormState = {
  appDisplayName: "",
  appDescription: "",
  iconPath: "",
  displayNameEdited: false,
  descriptionEdited: false,
  iconPathEdited: false,
  framework: "electron",
  serverType: "external",
  deploymentMode: "bundled",
  proxyUrl: "",
  serverPort: 3000,
  localServerPath: "ui/server.js",
  localApiEndpoint: "http://localhost:3001/api",
  autoManageTier1: false,
  vrooliBinaryPath: "vrooli",
  bundleManifestPath: "",
  platforms: { win: true, mac: true, linux: true },
  locationMode: "proper",
  outputPath: "",
  connectionResult: null,
  connectionError: null,
  deploymentManagerUrl: null,
  signingEnabledForBuild: false,
};

// ============================================================================
// Actions
// ============================================================================

export type FormAction =
  | { type: "SET_FIELD"; field: keyof GeneratorFormState; value: GeneratorFormState[keyof GeneratorFormState] }
  | { type: "SET_APP_DISPLAY_NAME"; value: string; markEdited?: boolean }
  | { type: "SET_APP_DESCRIPTION"; value: string; markEdited?: boolean }
  | { type: "SET_ICON_PATH"; value: string; markEdited?: boolean }
  | { type: "SET_PLATFORM"; platform: keyof PlatformSelection; value: boolean }
  | { type: "SET_CONNECTION_RESULT"; result: ProbeResponse | null; error: string | null }
  | { type: "APPLY_SERVER_STATE"; state: Partial<GeneratorFormState> }
  | { type: "APPLY_CONNECTION_CONFIG"; config: ConnectionConfigPayload }
  | { type: "APPLY_SCENARIO_DEFAULTS"; displayName: string; description: string; iconPath: string }
  | { type: "RESET"; keepTemplate?: boolean };

export interface ConnectionConfigPayload {
  deployment_mode?: string;
  proxy_url?: string;
  server_url?: string;
  auto_manage_vrooli?: boolean;
  vrooli_binary_path?: string;
  bundle_manifest_path?: string;
  app_display_name?: string;
  app_description?: string;
  icon?: string;
  server_type?: string;
}

// ============================================================================
// Reducer
// ============================================================================

function formReducer(state: GeneratorFormState, action: FormAction): GeneratorFormState {
  switch (action.type) {
    case "SET_FIELD":
      return { ...state, [action.field]: action.value };

    case "SET_APP_DISPLAY_NAME":
      return {
        ...state,
        appDisplayName: action.value,
        displayNameEdited: action.markEdited ?? state.displayNameEdited,
      };

    case "SET_APP_DESCRIPTION":
      return {
        ...state,
        appDescription: action.value,
        descriptionEdited: action.markEdited ?? state.descriptionEdited,
      };

    case "SET_ICON_PATH":
      return {
        ...state,
        iconPath: action.value,
        iconPathEdited: action.markEdited ?? state.iconPathEdited,
      };

    case "SET_PLATFORM":
      return {
        ...state,
        platforms: { ...state.platforms, [action.platform]: action.value },
      };

    case "SET_CONNECTION_RESULT":
      return {
        ...state,
        connectionResult: action.result,
        connectionError: action.error,
      };

    case "APPLY_SERVER_STATE":
      return { ...state, ...action.state };

    case "APPLY_CONNECTION_CONFIG": {
      const config = action.config;
      const updates: Partial<GeneratorFormState> = {};

      if (config.deployment_mode) {
        updates.deploymentMode = config.deployment_mode as DeploymentMode;
      }
      if (config.proxy_url || config.server_url) {
        updates.proxyUrl = config.proxy_url ?? config.server_url ?? "";
      }
      if (config.auto_manage_vrooli !== undefined) {
        updates.autoManageTier1 = config.auto_manage_vrooli;
      }
      if (config.vrooli_binary_path) {
        updates.vrooliBinaryPath = config.vrooli_binary_path;
      }
      if (config.bundle_manifest_path) {
        updates.bundleManifestPath = config.bundle_manifest_path;
      }
      if (config.app_display_name) {
        updates.appDisplayName = config.app_display_name;
        updates.displayNameEdited = true;
      }
      if (config.app_description) {
        updates.appDescription = config.app_description;
        updates.descriptionEdited = true;
      }
      if (config.icon) {
        updates.iconPath = config.icon;
        updates.iconPathEdited = true;
      }
      if (config.server_type) {
        updates.serverType = config.server_type as ServerType;
      }

      return { ...state, ...updates };
    }

    case "APPLY_SCENARIO_DEFAULTS": {
      const updates: Partial<GeneratorFormState> = {};
      if (!state.displayNameEdited) {
        updates.appDisplayName = action.displayName;
      }
      if (!state.descriptionEdited) {
        updates.appDescription = action.description;
      }
      if (!state.iconPathEdited) {
        updates.iconPath = action.iconPath;
      }
      return { ...state, ...updates };
    }

    case "RESET":
      return { ...DEFAULT_FORM_STATE };

    default:
      return state;
  }
}

// ============================================================================
// Hook
// ============================================================================

export interface UseGeneratorFormStateResult {
  state: GeneratorFormState;
  dispatch: React.Dispatch<FormAction>;

  // Convenience setters (memoized)
  setAppDisplayName: (value: string, markEdited?: boolean) => void;
  setAppDescription: (value: string, markEdited?: boolean) => void;
  setIconPath: (value: string, markEdited?: boolean) => void;
  setFramework: (value: string) => void;
  setServerType: (value: ServerType) => void;
  setDeploymentMode: (value: DeploymentMode) => void;
  setProxyUrl: (value: string) => void;
  setServerPort: (value: number) => void;
  setLocalServerPath: (value: string) => void;
  setLocalApiEndpoint: (value: string) => void;
  setAutoManageTier1: (value: boolean) => void;
  setVrooliBinaryPath: (value: string) => void;
  setBundleManifestPath: (value: string) => void;
  setPlatform: (platform: keyof PlatformSelection, value: boolean) => void;
  setLocationMode: (value: OutputLocation) => void;
  setOutputPath: (value: string) => void;
  setConnectionResult: (result: ProbeResponse | null, error: string | null) => void;
  setDeploymentManagerUrl: (value: string | null) => void;
  setSigningEnabledForBuild: (value: boolean) => void;

  // Bulk operations
  applyServerState: (state: Partial<GeneratorFormState>) => void;
  applyConnectionConfig: (config: ConnectionConfigPayload) => void;
  applyScenarioDefaults: (displayName: string, description: string, iconPath: string) => void;
  reset: () => void;
}

export function useGeneratorFormState(
  initialState: Partial<GeneratorFormState> = {}
): UseGeneratorFormStateResult {
  const [state, dispatch] = useReducer(formReducer, {
    ...DEFAULT_FORM_STATE,
    ...initialState,
  });

  // Convenience setters - memoized to prevent unnecessary re-renders
  const setAppDisplayName = useCallback((value: string, markEdited = true) => {
    dispatch({ type: "SET_APP_DISPLAY_NAME", value, markEdited });
  }, []);

  const setAppDescription = useCallback((value: string, markEdited = true) => {
    dispatch({ type: "SET_APP_DESCRIPTION", value, markEdited });
  }, []);

  const setIconPath = useCallback((value: string, markEdited = true) => {
    dispatch({ type: "SET_ICON_PATH", value, markEdited });
  }, []);

  const setFramework = useCallback((value: string) => {
    dispatch({ type: "SET_FIELD", field: "framework", value });
  }, []);

  const setServerType = useCallback((value: ServerType) => {
    dispatch({ type: "SET_FIELD", field: "serverType", value });
  }, []);

  const setDeploymentMode = useCallback((value: DeploymentMode) => {
    dispatch({ type: "SET_FIELD", field: "deploymentMode", value });
  }, []);

  const setProxyUrl = useCallback((value: string) => {
    dispatch({ type: "SET_FIELD", field: "proxyUrl", value });
  }, []);

  const setServerPort = useCallback((value: number) => {
    dispatch({ type: "SET_FIELD", field: "serverPort", value });
  }, []);

  const setLocalServerPath = useCallback((value: string) => {
    dispatch({ type: "SET_FIELD", field: "localServerPath", value });
  }, []);

  const setLocalApiEndpoint = useCallback((value: string) => {
    dispatch({ type: "SET_FIELD", field: "localApiEndpoint", value });
  }, []);

  const setAutoManageTier1 = useCallback((value: boolean) => {
    dispatch({ type: "SET_FIELD", field: "autoManageTier1", value });
  }, []);

  const setVrooliBinaryPath = useCallback((value: string) => {
    dispatch({ type: "SET_FIELD", field: "vrooliBinaryPath", value });
  }, []);

  const setBundleManifestPath = useCallback((value: string) => {
    dispatch({ type: "SET_FIELD", field: "bundleManifestPath", value });
  }, []);

  const setPlatform = useCallback((platform: keyof PlatformSelection, value: boolean) => {
    dispatch({ type: "SET_PLATFORM", platform, value });
  }, []);

  const setLocationMode = useCallback((value: OutputLocation) => {
    dispatch({ type: "SET_FIELD", field: "locationMode", value });
  }, []);

  const setOutputPath = useCallback((value: string) => {
    dispatch({ type: "SET_FIELD", field: "outputPath", value });
  }, []);

  const setConnectionResult = useCallback((result: ProbeResponse | null, error: string | null) => {
    dispatch({ type: "SET_CONNECTION_RESULT", result, error });
  }, []);

  const setDeploymentManagerUrl = useCallback((value: string | null) => {
    dispatch({ type: "SET_FIELD", field: "deploymentManagerUrl", value });
  }, []);

  const setSigningEnabledForBuild = useCallback((value: boolean) => {
    dispatch({ type: "SET_FIELD", field: "signingEnabledForBuild", value });
  }, []);

  // Bulk operations
  const applyServerState = useCallback((serverState: Partial<GeneratorFormState>) => {
    dispatch({ type: "APPLY_SERVER_STATE", state: serverState });
  }, []);

  const applyConnectionConfig = useCallback((config: ConnectionConfigPayload) => {
    dispatch({ type: "APPLY_CONNECTION_CONFIG", config });
  }, []);

  const applyScenarioDefaults = useCallback((displayName: string, description: string, iconPath: string) => {
    dispatch({ type: "APPLY_SCENARIO_DEFAULTS", displayName, description, iconPath });
  }, []);

  const reset = useCallback(() => {
    dispatch({ type: "RESET" });
  }, []);

  return {
    state,
    dispatch,
    setAppDisplayName,
    setAppDescription,
    setIconPath,
    setFramework,
    setServerType,
    setDeploymentMode,
    setProxyUrl,
    setServerPort,
    setLocalServerPath,
    setLocalApiEndpoint,
    setAutoManageTier1,
    setVrooliBinaryPath,
    setBundleManifestPath,
    setPlatform,
    setLocationMode,
    setOutputPath,
    setConnectionResult,
    setDeploymentManagerUrl,
    setSigningEnabledForBuild,
    applyServerState,
    applyConnectionConfig,
    applyScenarioDefaults,
    reset,
  };
}
