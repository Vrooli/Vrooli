/**
 * Hook for Generator page state management.
 * Composes micro-hooks into a single interface for the Generator page.
 *
 * This is a "fat hook" that orchestrates:
 * - Form state (useFormState)
 * - Pipeline actions (usePipelineActions)
 * - Server persistence (useScenarioSync)
 * - Signing (useSigningConfig)
 * - Modals (useGeneratorModals)
 */

import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  fetchProxyHints,
  fetchScenarioDesktopStatus,
  fetchBundleManifest,
  type BundleManifestResponse,
  type ProxyHintsResponse,
  type FormState,
} from "../lib/api";
import type { ScenarioDesktopStatus, ScenariosResponse } from "../components/scenario-inventory/types";
import { prepareFormSubmission } from "../controllers/generatorController";
import { useFormState, type UseFormStateReturn } from "./useFormState";
import { usePipelineActions, type UsePipelineActionsReturn } from "./usePipelineActions";
import { useScenarioSync, type UseScenarioSyncReturn, type PreflightSeed } from "./useScenarioSync";
import { useSigningConfig } from "./useSigningConfig";
import { useGeneratorModals, type UseGeneratorModalsReturn } from "./useGeneratorModals";
import type { BundleResult } from "../components/runtime/DeploymentManagerBundleHelper";

// ============================================================================
// Types
// ============================================================================

export interface UseGeneratorPageProps {
  scenarioName: string;
  selectedTemplate: string;
  selectionSource?: "inventory" | "manual" | null;
  onTemplateChange: (template: string) => void;
  onScenarioNameChange: (name: string) => void;
  onBuildStart: (buildId: string) => void;
  onOpenSigningTab: (scenario?: string) => void;
}

export interface UseGeneratorPageReturn {
  // Form state (from useFormState)
  formState: UseFormStateReturn;

  // Modals (from useGeneratorModals)
  modals: UseGeneratorModalsReturn;

  // Pipeline actions (from usePipelineActions)
  pipelineActions: UsePipelineActionsReturn;

  // Scenarios data
  scenarios: ScenarioDesktopStatus[];
  loadingScenarios: boolean;
  selectedScenario: ScenarioDesktopStatus | undefined;

  // Proxy hints
  proxyHints: ProxyHintsResponse | null;

  // Bundle manifest
  bundleManifest: BundleManifestResponse | null;

  // Signing state
  signingConfig: ReturnType<typeof useSigningConfig>["config"];
  signingReadiness: ReturnType<typeof useSigningConfig>["readiness"];
  signingLoading: boolean;
  refreshSigning: () => void;

  // Server state persistence
  serverFormState: FormState | null;
  hasInitiallyLoaded: boolean;
  stateSaving: boolean;
  isStale: boolean;
  pendingChanges: string[];
  validationStatus: UseScenarioSyncReturn["validationStatus"];
  serverTimestamps: { createdAt?: string; updatedAt?: string } | null;

  // Actions
  handleSubmit: (e: FormEvent) => void;
  resetFormState: (resetTemplate: boolean) => void;
  clearDraft: () => void;
  handleBundleComplete: (result: BundleResult) => void;

  // Bundle state
  bundleResultSeed: BundleResult | null;
}

// ============================================================================
// Hook Implementation
// ============================================================================

export function useGeneratorPage(props: UseGeneratorPageProps): UseGeneratorPageReturn {
  const {
    scenarioName,
    selectedTemplate,
    selectionSource,
    onTemplateChange,
    onBuildStart,
  } = props;

  // ========== Local State for Preflight Seeds ==========
  const [preflightSeed, setPreflightSeed] = useState<PreflightSeed>({
    result: null,
    error: null,
    override: false,
    secrets: {},
  });
  const [bundleResultSeed, setBundleResultSeed] = useState<BundleResult | null>(null);
  const [lastLoadedScenario, setLastLoadedScenario] = useState<string | null>(null);

  // ========== Compose Micro-Hooks ==========

  const formState = useFormState({
    scenarioName,
    selectionSource,
  });

  const pipelineActions = usePipelineActions({
    scenarioName,
    onBuildStart,
  });

  const scenarioSync = useScenarioSync({
    scenarioName,
    enabled: Boolean(scenarioName),
    onTemplateChange,
    onPreflightSeedLoaded: setPreflightSeed,
    onBundleSeedLoaded: setBundleResultSeed,
  });

  const modalHooks = useGeneratorModals();

  const {
    config: signingConfig,
    readiness: signingReadiness,
    loading: signingLoading,
    refreshAll: refreshSigning,
  } = useSigningConfig({ scenarioName });

  // ========== Queries ==========

  const { data: scenariosData, isLoading: loadingScenarios } = useQuery<ScenariosResponse>({
    queryKey: ["scenarios-desktop-status"],
    queryFn: fetchScenarioDesktopStatus,
  });

  const { data: proxyHints } = useQuery<ProxyHintsResponse | null>({
    queryKey: ["proxy-hints", scenarioName],
    queryFn: async () => {
      if (!scenarioName) return null;
      try {
        return await fetchProxyHints(scenarioName);
      } catch (error) {
        console.warn("Failed to load proxy hints", error);
        return null;
      }
    },
    enabled: Boolean(scenarioName),
    staleTime: 1000 * 60,
  });

  const { data: bundleManifestResp } = useQuery<BundleManifestResponse | null>({
    queryKey: ["bundle-manifest", formState.connection.bundleManifestPath.trim()],
    queryFn: () => {
      const path = formState.connection.bundleManifestPath.trim();
      if (!path) return Promise.resolve(null);
      return fetchBundleManifest({ bundle_manifest_path: path });
    },
    enabled: Boolean(formState.connection.bundleManifestPath.trim()),
  });

  // ========== Computed Values ==========

  const selectedScenario = useMemo(
    () => scenariosData?.scenarios.find((s) => s.name === scenarioName),
    [scenariosData?.scenarios, scenarioName]
  );

  // ========== Effects ==========

  // Initialize preflight state from seed
  useEffect(() => {
    if (preflightSeed.secrets && Object.keys(preflightSeed.secrets).length > 0) {
      pipelineActions.setPreflightSecrets(preflightSeed.secrets);
    }
    if (preflightSeed.override) {
      pipelineActions.setPreflightOverride(preflightSeed.override);
    }
  }, [preflightSeed, pipelineActions]);

  // Reset preflight when switching to non-bundled mode
  useEffect(() => {
    if (!formState.isBundled) {
      pipelineActions.resetPreflight();
    }
  }, [formState.isBundled, pipelineActions]);

  // Apply scenario defaults when selecting a new scenario
  useEffect(() => {
    if (!scenarioName) {
      if (!formState.appMetadata.displayNameEdited) formState.setAppDisplayName("");
      if (!formState.appMetadata.descriptionEdited) formState.setAppDescription("");
      if (!formState.appMetadata.iconPathEdited) formState.setIconPath("");
      return;
    }
    if (!scenarioSync.hasInitiallyLoaded) return;
    if (scenarioSync.serverFormState) return;
    if (!selectedScenario) return;
    formState.applyScenarioDefaults(selectedScenario);
  }, [
    scenarioName,
    selectedScenario,
    scenarioSync.hasInitiallyLoaded,
    scenarioSync.serverFormState,
    formState,
  ]);

  // Apply saved connection config from scenario
  useEffect(() => {
    if (!scenarioName) return;
    const connectionConfig = selectedScenario?.connection_config;
    const updatedAt = connectionConfig?.updated_at;
    if (!updatedAt) return;
    const configKey = `${scenarioName}:${updatedAt}`;
    if (configKey === lastLoadedScenario) return;
    if (scenarioSync.serverFormState && scenarioSync.serverFormState.app_display_name) return;
    formState.applySavedConnection(connectionConfig);
    setLastLoadedScenario(configKey);
  }, [
    scenarioName,
    selectedScenario?.connection_config,
    lastLoadedScenario,
    scenarioSync.serverFormState,
    formState,
  ]);

  // Save preflight result when completed
  const prevPreflightResultRef = useRef<typeof pipelineActions.preflightResult>(null);
  useEffect(() => {
    if (
      pipelineActions.preflightResult &&
      pipelineActions.preflightResult !== prevPreflightResultRef.current &&
      pipelineActions.runStatus === "completed" &&
      scenarioName &&
      scenarioSync.hasInitiallyLoaded
    ) {
      void scenarioSync.saveStageResult("preflight", pipelineActions.preflightResult, {
        preflight_result: pipelineActions.preflightResult,
        preflight_error: null,
      });
    }
    prevPreflightResultRef.current = pipelineActions.preflightResult;
  }, [
    pipelineActions.preflightResult,
    pipelineActions.runStatus,
    scenarioName,
    scenarioSync,
  ]);

  // ========== Handlers ==========

  const handleBundleComplete = useCallback(
    (result: BundleResult) => {
      if (!scenarioName || !scenarioSync.hasInitiallyLoaded) return;
      setBundleResultSeed(result);
      scenarioSync.setBundleResultSeed(result);
      void scenarioSync.saveStageResult("bundle", result, {
        bundle_manifest_path: result.manifestPath ?? undefined,
        bundle_result: result,
      });
    },
    [scenarioName, scenarioSync]
  );

  const handleSubmit = useCallback(
    (e: FormEvent) => {
      e.preventDefault();
      formState.clearValidationErrors();

      // Use controller to prepare form submission
      const result = prepareFormSubmission({
        scenarioName,
        selectedTemplate,
        deploymentMode: formState.deployment.mode === "bundled" ? "bundled" : "proxy",
        proxyUrl: formState.connection.proxyUrl,
        selectedPlatforms: formState.selectedPlatformsList,
        isBundled: formState.isBundled,
        requiresProxyUrl: formState.requiresRemoteConfig,
        bundleManifestPath: formState.connection.bundleManifestPath,
        appDisplayName: formState.appMetadata.displayName,
        appDescription: formState.appMetadata.description,
        locationMode: formState.output.locationMode,
        outputPath: formState.output.outputPath,
        signingEnabledForBuild: formState.signingEnabledForBuild,
        signingConfig,
        signingReadiness,
        storePreflightResult: pipelineActions.preflightResult,
        serverPreflightResult: scenarioSync.serverFormState?.preflight_result,
        missingSecretsCount: pipelineActions.missingPreflightSecrets.length,
        preflightOverride: pipelineActions.preflightOverride,
      });

      if (result.errors.length > 0) {
        formState.setValidationErrors(result.errors);
        window.scrollTo({ top: 0, behavior: "smooth" });
        return;
      }

      if (result.pipelineConfig) {
        pipelineActions.generateDesktop(result.pipelineConfig);
      }
    },
    [
      scenarioName,
      selectedTemplate,
      formState,
      pipelineActions,
      scenarioSync,
      signingConfig,
      signingReadiness,
    ]
  );

  const resetFormState = useCallback(
    (resetTemplate: boolean) => {
      formState.resetFormState(resetTemplate);
      formState.setSigningEnabledForBuild(false);
      setLastLoadedScenario(null);
      setPreflightSeed({ result: null, error: null, override: false, secrets: {} });
      setBundleResultSeed(null);
      pipelineActions.resetPreflight();
      if (resetTemplate) {
        onTemplateChange("basic");
      }
    },
    [formState, pipelineActions, onTemplateChange]
  );

  // ========== Return ==========

  return {
    formState,
    modals: modalHooks,
    pipelineActions,

    scenarios: scenariosData?.scenarios ?? [],
    loadingScenarios,
    selectedScenario,

    proxyHints: proxyHints ?? null,
    bundleManifest: bundleManifestResp ?? null,

    signingConfig,
    signingReadiness,
    signingLoading,
    refreshSigning,

    serverFormState: scenarioSync.serverFormState,
    hasInitiallyLoaded: scenarioSync.hasInitiallyLoaded,
    stateSaving: scenarioSync.isSaving,
    isStale: scenarioSync.isStale,
    pendingChanges: scenarioSync.pendingChanges,
    validationStatus: scenarioSync.validationStatus,
    serverTimestamps: scenarioSync.timestamps,

    handleSubmit,
    resetFormState,
    clearDraft: scenarioSync.clearState,
    handleBundleComplete,

    bundleResultSeed,
  };
}
