/**
 * Form store selectors.
 * Provides derived state from the form store.
 */

import {
  decideConnection,
  SERVER_TYPE_OPTIONS,
  type ServerType,
} from "../domain/deployment";
import {
  computeStandardOutputPath,
  computeStagingPreviewPath,
  getSelectedPlatforms,
} from "../domain/generator";
import { getIconPreviewUrl } from "../lib/api";
import type { FormStore } from "./formTypes";

// ============================================================================
// Connection Decision Selectors
// ============================================================================

/**
 * Get the connection decision based on deployment mode and server type.
 * This determines if the app is bundled, external, or embedded.
 */
export const selectConnectionDecision = (state: FormStore) =>
  decideConnection(state.deployment.mode, state.deployment.serverType);

/**
 * Check if the current configuration is for bundled runtime.
 */
export const selectIsBundled = (state: FormStore) =>
  selectConnectionDecision(state).kind === "bundled-runtime";

/**
 * Check if the current configuration requires a remote proxy URL.
 */
export const selectRequiresRemoteConfig = (state: FormStore) =>
  selectConnectionDecision(state).requiresProxyUrl;

// ============================================================================
// Server Type Selectors
// ============================================================================

/**
 * Get the list of allowed server types based on deployment mode.
 */
export const selectAllowedServerTypes = (state: FormStore): ServerType[] => {
  const { mode } = state.deployment;
  if (mode === "bundled" || mode === "cloud-api") {
    return ["external"];
  }
  return SERVER_TYPE_OPTIONS.map((option) => option.value);
};

// ============================================================================
// Platform Selectors
// ============================================================================

/**
 * Get the list of selected platforms as an array of strings.
 */
export const selectSelectedPlatformsList = (state: FormStore): string[] =>
  getSelectedPlatforms(state.platforms);

// ============================================================================
// Output Path Selectors
// ============================================================================

/**
 * These selectors depend on scenarioName which is not in the form store.
 * They need to be called with the scenarioName as a parameter.
 */
export const selectStandardOutputPath = (scenarioName: string): string =>
  computeStandardOutputPath(scenarioName);

export const selectStagingPreviewPath = (scenarioName: string): string =>
  computeStagingPreviewPath(scenarioName);

/**
 * Check if using custom output location.
 */
export const selectIsCustomLocation = (state: FormStore): boolean =>
  state.output.locationMode === "custom";

// ============================================================================
// Icon Selectors
// ============================================================================

/**
 * Get the icon preview URL for the current icon path.
 */
export const selectIconPreviewUrl = (state: FormStore): string => {
  const { iconPath } = state.appMetadata;
  return iconPath ? getIconPreviewUrl(iconPath) : "";
};

// ============================================================================
// Validation Selectors
// ============================================================================

/**
 * Check if there are any validation errors.
 */
export const selectHasValidationErrors = (state: FormStore): boolean =>
  state.validationErrors.length > 0;

/**
 * Get validation errors for a specific field.
 */
export const selectFieldErrors = (field: string) => (state: FormStore) =>
  state.validationErrors.filter((e) => e.field === field);

// ============================================================================
// Combined State Selectors (for backward compatibility)
// ============================================================================

/**
 * Get all app metadata at once.
 */
export const selectAppMetadata = (state: FormStore) => state.appMetadata;

/**
 * Get all deployment config at once.
 */
export const selectDeployment = (state: FormStore) => state.deployment;

/**
 * Get all output config at once.
 */
export const selectOutput = (state: FormStore) => state.output;

/**
 * Get all platforms at once.
 */
export const selectPlatforms = (state: FormStore) => state.platforms;

/**
 * Get all connection config at once.
 */
export const selectConnection = (state: FormStore) => state.connection;
