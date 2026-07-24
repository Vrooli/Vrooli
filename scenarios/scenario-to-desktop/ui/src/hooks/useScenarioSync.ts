/**
 * Hook for scenario state synchronization with server.
 * Handles form state persistence, draft recovery, and staleness detection.
 * Wraps useScenarioState and provides form-specific serialization.
 */

import { useCallback, useEffect, useMemo, useRef } from "react";
import { useScenarioState, type UseScenarioStateResult } from "./useScenarioState";
import { useFormStore } from "../store/formStore";
import type { FormState, BundlePreflightResponse } from "../lib/api";
import type { BundleResult } from "../components/sections/bundle/BundleSection";
import type { ProbeResponse } from "../lib/api";
import type {
  DeploymentMode,
  ServerType,
} from "../domain/deployment";
import type { OutputLocation } from "../domain/generator";
import { DEFAULT_LOCAL_API_ENDPOINT } from "../store/formTypes";

// ============================================================================
// Types
// ============================================================================

export interface UseScenarioSyncProps {
  scenarioName: string;
  enabled?: boolean;
  onTemplateChange?: (template: string) => void;
  onPreflightSeedLoaded?: (seed: PreflightSeed) => void;
  onBundleSeedLoaded?: (result: BundleResult | null) => void;
}

export interface PreflightSeed {
  result: BundlePreflightResponse | null;
  error: string | null;
  override: boolean;
  secrets: Record<string, string>;
}

export interface UseScenarioSyncReturn {
  // Server state
  serverFormState: FormState | null;
  hasInitiallyLoaded: boolean;
  isSaving: boolean;
  isStale: boolean;
  pendingChanges: string[];
  validationStatus: UseScenarioStateResult["validationStatus"];
  timestamps: { createdAt?: string; updatedAt?: string } | null;

  // Actions
  saveStageResult: UseScenarioStateResult["saveStageResult"];
  clearState: () => Promise<void>;
  saveNow: () => Promise<void>;

  // Form state serialization
  getFormStateForServer: () => Partial<FormState>;

  // Bundle result seed
  bundleResultSeed: BundleResult | null;
  setBundleResultSeed: (result: BundleResult | null) => void;
}

// ============================================================================
// Hook Implementation
// ============================================================================

export function useScenarioSync(props: UseScenarioSyncProps): UseScenarioSyncReturn {
  const {
    scenarioName,
    enabled = true,
    onTemplateChange,
    onPreflightSeedLoaded,
    onBundleSeedLoaded,
  } = props;

  // ========== Form Store ==========
  const formStore = useFormStore();

  // ========== Scenario State ==========
  const {
    formState: serverFormState,
    hasInitiallyLoaded,
    isSaving,
    isStale,
    pendingChanges,
    validationStatus,
    timestamps,
    updateFormState,
    saveStageResult,
    clearState,
    saveNow,
  } = useScenarioState({
    scenarioName,
    enabled: enabled && Boolean(scenarioName),
    onStateLoaded: (state) => {
      if (!state.form_state) return;
      const fs = state.form_state;

      // Hydrate form store from server
      formStore.hydrateFromServer({
        appMetadata: {
          displayName: fs.app_display_name || "",
          description: fs.app_description || "",
          iconPath: fs.icon_path || "",
          displayNameEdited: fs.display_name_edited || false,
          descriptionEdited: fs.description_edited || false,
          iconPathEdited: fs.icon_path_edited || false,
          iconPreviewError: false,
        },
        deployment: {
          framework: fs.framework || "electron",
          serverType: (fs.server_type as ServerType) ?? "external",
          mode: (fs.deployment_mode as DeploymentMode) ?? "bundled",
        },
        output: {
          locationMode: (fs.location_mode as OutputLocation) ?? "proper",
          outputPath: fs.output_path ?? "",
        },
        platforms: {
          win: fs.platforms?.win ?? true,
          mac: fs.platforms?.mac ?? true,
          linux: fs.platforms?.linux ?? true,
        },
        connection: {
          proxyUrl: fs.proxy_url ?? "",
          bundleManifestPath: fs.bundle_manifest_path ?? "",
          serverPort: fs.server_port ?? 3000,
          localServerPath: fs.local_server_path ?? "ui/server.js",
          localApiEndpoint: fs.local_api_endpoint ?? DEFAULT_LOCAL_API_ENDPOINT,
          autoManageTier1: fs.auto_manage_tier1 ?? false,
          vrooliBinaryPath: fs.vrooli_binary_path ?? "vrooli",
          connectionResult: (fs.connection_result as ProbeResponse | null) ?? null,
          connectionError: fs.connection_error ?? null,
        },
        signingEnabledForBuild: fs.signing_enabled_for_build ?? false,
        selectedTemplate: fs.selected_template ?? "basic",
      });

      // Handle non-store state
      if (fs.selected_template && onTemplateChange) {
        onTemplateChange(fs.selected_template);
      }

      // Load preflight seed
      const preflightSeed: PreflightSeed = {
        result: fs.preflight_result ?? null,
        error: fs.preflight_error ?? null,
        override: fs.preflight_override ?? false,
        secrets: fs.preflight_secrets ?? {},
      };
      onPreflightSeedLoaded?.(preflightSeed);

      // Load bundle result seed
      const bundleSeed = (fs.bundle_result as BundleResult) ?? null;
      formStore.setBundleResultSeed(bundleSeed);
      onBundleSeedLoaded?.(bundleSeed);
    },
    onStateCleared: () => {
      formStore.resetFormState();
      onBundleSeedLoaded?.(null);
    },
  });

  // ========== Form State Serialization ==========

  const getFormStateForServer = useCallback((): Partial<FormState> => {
    const { appMetadata, deployment, output, platforms, connection, signingEnabledForBuild, selectedTemplate, bundleResultSeed } = formStore;

    return {
      selected_template: selectedTemplate,
      app_display_name: appMetadata.displayName,
      app_description: appMetadata.description,
      icon_path: appMetadata.iconPath,
      display_name_edited: appMetadata.displayNameEdited,
      description_edited: appMetadata.descriptionEdited,
      icon_path_edited: appMetadata.iconPathEdited,
      framework: deployment.framework,
      server_type: deployment.serverType,
      deployment_mode: deployment.mode,
      platforms,
      location_mode: output.locationMode,
      output_path: output.outputPath,
      proxy_url: connection.proxyUrl,
      bundle_manifest_path: connection.bundleManifestPath,
      server_port: connection.serverPort,
      local_server_path: connection.localServerPath,
      local_api_endpoint: connection.localApiEndpoint,
      auto_manage_tier1: connection.autoManageTier1,
      vrooli_binary_path: connection.vrooliBinaryPath,
      connection_result: connection.connectionResult,
      connection_error: connection.connectionError,
      signing_enabled_for_build: signingEnabledForBuild,
      bundle_result: bundleResultSeed,
    };
  }, [formStore]);

  // ========== Auto-save Effect ==========

  const prevFormStateRef = useRef<string>("");
  const prevScenarioForSaveRef = useRef<string>(scenarioName);

  // Reset tracking when scenario changes
  useEffect(() => {
    if (prevScenarioForSaveRef.current !== scenarioName) {
      prevScenarioForSaveRef.current = scenarioName;
      prevFormStateRef.current = "";
    }
  }, [scenarioName]);

  // Auto-save form state to server
  useEffect(() => {
    if (!scenarioName) return;
    if (!hasInitiallyLoaded) return;

    const formStateForServer = getFormStateForServer();
    const serialized = JSON.stringify(formStateForServer);
    if (serialized === prevFormStateRef.current) return;
    prevFormStateRef.current = serialized;
    updateFormState(formStateForServer);
  }, [scenarioName, hasInitiallyLoaded, getFormStateForServer, updateFormState]);

  // ========== Timestamps ==========

  const timestampsFormatted = useMemo(() => {
    if (!timestamps) return null;
    return {
      createdAt: timestamps.createdAt,
      updatedAt: timestamps.updatedAt,
    };
  }, [timestamps]);

  // ========== Return ==========

  return {
    serverFormState,
    hasInitiallyLoaded,
    isSaving,
    isStale,
    pendingChanges: pendingChanges.map((c) => (c as unknown as { field: string }).field || String(c)),
    validationStatus,
    timestamps: timestampsFormatted,
    saveStageResult,
    clearState,
    saveNow,
    getFormStateForServer,
    bundleResultSeed: formStore.bundleResultSeed,
    setBundleResultSeed: formStore.setBundleResultSeed,
  };
}
