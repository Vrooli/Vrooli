/**
 * Zustand store for form state management.
 * Centralizes all form-related state, separate from pipeline execution state.
 *
 * Types are defined in ./formTypes.ts
 * Selectors are defined in ./formSelectors.ts
 */

import { create } from "zustand";
import type { DeploymentMode, ServerType } from "../domain/deployment";
import type { ProbeResponse } from "../lib/api";
import {
  type FormStore,
  type PlatformsState,
  type HydrateFormData,
  type ValidationError,
  type OutputLocation,
  initialFormState,
  defaultAppMetadata,
  defaultDeployment,
  defaultOutput,
  defaultPlatforms,
  defaultConnection,
} from "./formTypes";

// Re-export types for convenience
export type {
  FormStore,
  FormStoreState,
  FormStoreActions,
  AppMetadataState,
  DeploymentState,
  OutputState,
  PlatformsState,
  ConnectionState,
  ValidationError,
  HydrateFormData,
} from "./formTypes";

// Re-export selectors for convenience
export {
  selectIsBundled,
  selectConnectionDecision,
  selectRequiresRemoteConfig,
  selectAllowedServerTypes,
  selectSelectedPlatformsList,
  selectStandardOutputPath,
  selectStagingPreviewPath,
  selectIconPreviewUrl,
  selectIsCustomLocation,
} from "./formSelectors";

// ============================================================================
// Store Implementation
// ============================================================================

export const useFormStore = create<FormStore>((set) => ({
  // Initial state
  ...initialFormState,

  // ========== App Metadata Setters ==========

  setAppDisplayName: (value: string) =>
    set((state) => ({
      appMetadata: {
        ...state.appMetadata,
        displayName: value,
        displayNameEdited: true,
      },
    })),

  setAppDescription: (value: string) =>
    set((state) => ({
      appMetadata: {
        ...state.appMetadata,
        description: value,
        descriptionEdited: true,
      },
    })),

  setIconPath: (value: string) =>
    set((state) => ({
      appMetadata: {
        ...state.appMetadata,
        iconPath: value,
        iconPathEdited: true,
      },
    })),

  setIconPreviewError: (value: boolean) =>
    set((state) => ({
      appMetadata: { ...state.appMetadata, iconPreviewError: value },
    })),

  // ========== Deployment Setters ==========

  setDeploymentMode: (mode: DeploymentMode) =>
    set((state) => ({
      deployment: { ...state.deployment, mode },
    })),

  setServerType: (serverType: ServerType) =>
    set((state) => ({
      deployment: { ...state.deployment, serverType },
    })),

  setFramework: (framework: string) =>
    set((state) => ({
      deployment: { ...state.deployment, framework },
    })),

  // ========== Output Setters ==========

  setLocationMode: (locationMode: OutputLocation) =>
    set((state) => ({
      output: { ...state.output, locationMode },
    })),

  setOutputPath: (outputPath: string) =>
    set((state) => ({
      output: { ...state.output, outputPath },
    })),

  // ========== Platform Setters ==========

  setPlatforms: (platforms: PlatformsState) =>
    set({ platforms }),

  handlePlatformChange: (platform: string, checked: boolean) =>
    set((state) => ({
      platforms: { ...state.platforms, [platform]: checked },
    })),

  // ========== Connection Setters ==========

  setProxyUrl: (proxyUrl: string) =>
    set((state) => ({
      connection: { ...state.connection, proxyUrl },
    })),

  setBundleManifestPath: (bundleManifestPath: string) =>
    set((state) => ({
      connection: { ...state.connection, bundleManifestPath },
    })),

  setServerPort: (serverPort: number) =>
    set((state) => ({
      connection: { ...state.connection, serverPort },
    })),

  setLocalServerPath: (localServerPath: string) =>
    set((state) => ({
      connection: { ...state.connection, localServerPath },
    })),

  setLocalApiEndpoint: (localApiEndpoint: string) =>
    set((state) => ({
      connection: { ...state.connection, localApiEndpoint },
    })),

  setAutoManageTier1: (autoManageTier1: boolean) =>
    set((state) => ({
      connection: { ...state.connection, autoManageTier1 },
    })),

  setVrooliBinaryPath: (vrooliBinaryPath: string) =>
    set((state) => ({
      connection: { ...state.connection, vrooliBinaryPath },
    })),

  setConnectionResult: (connectionResult: ProbeResponse | null) =>
    set((state) => ({
      connection: { ...state.connection, connectionResult },
    })),

  setConnectionError: (connectionError: string | null) =>
    set((state) => ({
      connection: { ...state.connection, connectionError },
    })),

  // ========== Signing ==========

  setSigningEnabledForBuild: (enabled: boolean) =>
    set({ signingEnabledForBuild: enabled }),

  // ========== Template ==========

  setSelectedTemplate: (template: string) =>
    set({ selectedTemplate: template }),

  // ========== Validation ==========

  setValidationErrors: (errors: ValidationError[]) =>
    set({ validationErrors: errors }),

  clearValidationErrors: () =>
    set({ validationErrors: [] }),

  // ========== UI State ==========

  setScenarioLocked: (locked: boolean) =>
    set({ scenarioLocked: locked }),

  // ========== Bundle Result Seed ==========

  setBundleResultSeed: (result) =>
    set({ bundleResultSeed: result }),

  // ========== Reset ==========

  resetFormState: () =>
    set({
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
    }),

  // ========== Hydration ==========

  hydrateFromServer: (data: HydrateFormData) =>
    set((state) => ({
      appMetadata: data.appMetadata
        ? { ...state.appMetadata, ...data.appMetadata }
        : state.appMetadata,
      deployment: data.deployment
        ? { ...state.deployment, ...data.deployment }
        : state.deployment,
      output: data.output
        ? { ...state.output, ...data.output }
        : state.output,
      platforms: data.platforms ?? state.platforms,
      connection: data.connection
        ? { ...state.connection, ...data.connection }
        : state.connection,
      signingEnabledForBuild: data.signingEnabledForBuild ?? state.signingEnabledForBuild,
      selectedTemplate: data.selectedTemplate ?? state.selectedTemplate,
      bundleResultSeed: data.bundleResultSeed ?? state.bundleResultSeed,
    })),
}));
