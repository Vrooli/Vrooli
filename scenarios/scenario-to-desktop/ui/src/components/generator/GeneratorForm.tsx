import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import {
  fetchProxyHints,
  fetchScenarioDesktopStatus,
  fetchBundleManifest,
  probeEndpoints,
  type BundlePreflightResponse,
  type BundleManifestResponse,
  type ProbeResponse,
  type ProxyHintsResponse,
  type PipelineConfig,
} from "../../lib/api";
import type { DeploymentManagerBundleHelperHandle, BundleResult } from "../runtime/DeploymentManagerBundleHelper";
import { DeploymentSummarySection } from "./DeploymentSummarySection";
import { PlatformSelector } from "./PlatformSelector";
import type { DesktopConnectionConfig, ScenariosResponse } from "../scenario-inventory/types";
import {
  ScenarioSelector,
  FrameworkTemplateSection,
  SigningInlineSection,
  OutputLocationSelector,
  OutputPathField,
  validateFormInputs,
  AppMetadataSection,
  ConnectionSectionRouter,
  GeneratorFormHeader,
  GeneratorFormFooter,
  GeneratorModalsContainer,
} from ".";
import {
  DEFAULT_DEPLOYMENT_MODE,
  DEFAULT_SERVER_TYPE,
  SERVER_TYPE_OPTIONS,
  type DeploymentMode,
  type ServerType
} from "../../domain/deployment";
import { type OutputLocation } from "../../domain/generator";
import {
  useScenarioState,
  useSigningConfig,
  useGeneratorModals,
  useFormState,
} from "../../hooks";
import {
  usePipelineStore,
  selectIsRunning,
  selectIsSubmitting,
  selectPreflightOk,
  selectMissingSecrets,
  DEFAULT_LOCAL_API_ENDPOINT,
} from "../../store";
import { PendingChangesAlert } from "../state/PendingChangesAlert";
import type { FormState } from "../../lib/api";

/** Exposed form state for sharing with other sections */
export interface ExposedFormState {
  bundleManifestPath: string;
  isBundled: boolean;
  bundleManifest?: unknown;
  // Bundle-related handlers for BundleSection
  onBundleManifestChange: (path: string) => void;
  onBundleComplete: (result: import("../runtime/DeploymentManagerBundleHelper").BundleResult) => void;
  bundleHelperRef: React.RefObject<import("../runtime/DeploymentManagerBundleHelper").DeploymentManagerBundleHelperHandle>;
}

/** Validation state exposed to parent for the submit button in GenerateSection */
export interface ValidationState {
  errors: import("../../domain/generator").ValidationError[];
  clearErrors: () => void;
  isPending: boolean;
  isError: boolean;
  errorMessage: string | null;
  isUpdateMode: boolean;
}

interface GeneratorFormProps {
  selectedTemplate: string;
  onTemplateChange: (template: string) => void;
  onBuildStart?: (buildId: string) => void;
  scenarioName: string;
  onScenarioNameChange: (name: string) => void;
  selectionSource?: "inventory" | "manual" | null;
  onOpenSigningTab: (scenario?: string) => void;
  formId?: string;
  showSubmit?: boolean;
  onGenerateStateChange?: (state: { pending: boolean; error: string | null }) => void;
  /** Callback when form state changes that other sections need */
  onFormStateChange?: (state: ExposedFormState) => void;
  /** Callback when submit handler is ready - allows parent to trigger form submission */
  onSubmitHandlerReady?: (submitFn: () => void) => void;
  /** Callback when validation state changes - used by GenerateSection for submit button */
  onValidationStateChange?: (state: ValidationState) => void;
}

export function GeneratorForm({
  selectedTemplate,
  onTemplateChange,
  onBuildStart,
  scenarioName,
  onScenarioNameChange,
  selectionSource,
  onOpenSigningTab,
  formId,
  showSubmit = true,
  onGenerateStateChange,
  onFormStateChange,
  onSubmitHandlerReady,
  onValidationStateChange,
}: GeneratorFormProps) {
  const { modals, openModal, closeModal } = useGeneratorModals();

  // Form state hook - manages all form fields that get persisted to server
  const formState = useFormState({ scenarioName, selectionSource });
  const {
    appMetadata,
    setAppDisplayName,
    setAppDescription,
    setIconPath,
    setIconPreviewError,
    deployment,
    setDeploymentMode,
    setServerType,
    setFramework,
    output,
    setLocationMode,
    setOutputPath,
    platforms,
    handlePlatformChange,
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
    resetFormState: resetHookState,
    hydrateFromServer,
    // Derived values from useFormState
    connectionDecision,
    isBundled,
    requiresRemoteConfig,
    standardOutputPath,
    stagingPreviewPath,
    selectedPlatformsList,
    allowedServerTypes,
    isCustomLocation,
    isUpdateMode,
    iconPreviewUrl,
    // Scenario lock state
    scenarioLocked,
    setScenarioLocked,
    // Validation
    validationErrors,
    setValidationErrors,
    clearValidationErrors,
  } = formState;

  // Destructure for easier access
  const { displayName: appDisplayName, description: appDescription, iconPath, iconPreviewError } = appMetadata;
  const { displayNameEdited, descriptionEdited, iconPathEdited } = appMetadata;
  const { mode: deploymentMode, serverType, framework } = deployment;
  const { locationMode, outputPath } = output;
  const { proxyUrl, bundleManifestPath, serverPort, localServerPath, localApiEndpoint, autoManageTier1, vrooliBinaryPath, connectionResult, connectionError } = connection;
  const [preflightSeed, setPreflightSeed] = useState({
    result: null as BundlePreflightResponse | null,
    error: null as string | null,
    override: false,
    secrets: {} as Record<string, string>
  });
  // Bundle result seed from server state - similar pattern to preflightSeed
  const [bundleResultSeed, setBundleResultSeed] = useState<BundleResult | null>(null);
  const [lastLoadedScenario, setLastLoadedScenario] = useState<string | null>(null);
  const bundleHelperRef = useRef<DeploymentManagerBundleHelperHandle>(null);
  // Note: resolveEndpoints is available via the domain layer if needed
  // for server_path and api_endpoint in buildDesktopConfig
  // Pipeline store for preflight
  const {
    setScenario: setPipelineScenario,
    runPreflightStage,
    cancelPipeline: cancelPreflightPipeline,
    resetPreflight,
    resetCurrentPipeline,
    preflightResult,
    preflightSecrets,
    preflightOverride,
    setPreflightSecrets,
    setPreflightOverride,
    pipelineId: preflightPipelineId,
    runStatus: preflightRunStatus,
    errorInfo: preflightErrorInfo,
  } = usePipelineStore();
  const preflightError = preflightErrorInfo?.message ?? null;
  const preflightPending = usePipelineStore(selectIsRunning);
  const preflightOk = usePipelineStore(selectPreflightOk);
  const missingPreflightSecrets = usePipelineStore(selectMissingSecrets);

  // Set scenario in pipeline store when it changes
  useEffect(() => {
    if (scenarioName) {
      setPipelineScenario(scenarioName);
    }
  }, [scenarioName, setPipelineScenario]);

  // Initialize preflight state from server-persisted seed
  useEffect(() => {
    if (preflightSeed.secrets && Object.keys(preflightSeed.secrets).length > 0) {
      setPreflightSecrets(preflightSeed.secrets);
    }
    if (preflightSeed.override) {
      setPreflightOverride(preflightSeed.override);
    }
  }, [preflightSeed, setPreflightSecrets, setPreflightOverride]);

  // Track previous preflight result to detect new completions
  const prevPreflightResultRef = useRef<typeof preflightResult>(null);

  // Reset preflight state when switching to non-bundled mode
  useEffect(() => {
    if (!isBundled) {
      resetPreflight();
    }
  }, [isBundled, resetPreflight]);

  // Wrapper for running preflight with the right config
  const _runPreflight = useCallback(
    async (secretsOverride?: Record<string, string>, configOverride?: Partial<PipelineConfig>) => {
      if (!scenarioName) return;
      const manifestPath = bundleManifestPath.trim();
      if (!manifestPath && isBundled) return;

      const filteredSecrets = Object.entries(secretsOverride ?? preflightSecrets)
        .filter(([, value]) => value.trim())
        .reduce<Record<string, string>>((acc, [key, value]) => {
          acc[key] = value;
          return acc;
        }, {});

      setPreflightOverride(false);

      await runPreflightStage({
        bundle_manifest_path: manifestPath || undefined,
        preflight_secrets: Object.keys(filteredSecrets).length > 0 ? filteredSecrets : undefined,
        ...configOverride,
      });
    },
    [scenarioName, bundleManifestPath, isBundled, preflightSecrets, runPreflightStage, setPreflightOverride]
  );

  const {
    config: signingConfig,
    readiness: signingReadiness,
    loading: signingLoading,
    enabledForBuild: signingEnabledForBuild,
    setEnabledForBuild: setSigningEnabledForBuild,
    refreshAll: refreshSigning
  } = useSigningConfig({ scenarioName });

  const resetFormState = useCallback((resetTemplate: boolean) => {
    // Reset all hook-managed form state
    resetHookState();
    // Reset remaining non-hook state
    setSigningEnabledForBuild(false);
    setLastLoadedScenario(null);
    clearValidationErrors();
    setPreflightSeed({
      result: null,
      error: null,
      override: false,
      secrets: {}
    });
    resetPreflight();
    if (resetTemplate) {
      onTemplateChange("basic");
    }
  }, [resetHookState, onTemplateChange, resetPreflight, setSigningEnabledForBuild, clearValidationErrors]);

  // Server-side state persistence via useScenarioState
  const {
    formState: serverFormState,
    hasInitiallyLoaded,
    isSaving: stateSaving,
    isStale,
    pendingChanges,
    validationStatus,
    timestamps: serverTimestamps,
    updateFormState,
    saveStageResult,
    clearState,
  } = useScenarioState({
    scenarioName,
    enabled: Boolean(scenarioName),
    onStateLoaded: (state) => {
      if (!state.form_state) return;
      const fs = state.form_state;

      // Hydrate form state from server using the hook's hydration method
      hydrateFromServer({
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
          serverType: (fs.server_type as ServerType) ?? DEFAULT_SERVER_TYPE,
          mode: (fs.deployment_mode as DeploymentMode) ?? DEFAULT_DEPLOYMENT_MODE,
        },
        platforms: {
          win: fs.platforms?.win ?? true,
          mac: fs.platforms?.mac ?? true,
          linux: fs.platforms?.linux ?? true,
        },
        output: {
          locationMode: (fs.location_mode as OutputLocation) ?? "proper",
          outputPath: fs.output_path ?? "",
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
      });

      // Handle non-hook state
      setSigningEnabledForBuild(fs.signing_enabled_for_build ?? false);
      if (fs.selected_template) {
        onTemplateChange(fs.selected_template);
      }
      // Apply preflight seed from server state
      setPreflightSeed({
        result: fs.preflight_result ?? null,
        error: fs.preflight_error ?? null,
        override: fs.preflight_override ?? false,
        secrets: fs.preflight_secrets ?? {}
      });
      // Apply bundle result seed from server state
      setBundleResultSeed((fs.bundle_result as BundleResult) ?? null);
    },
    onStateCleared: () => {
      resetFormState(true);
      setPreflightSeed({
        result: null,
        error: null,
        override: false,
        secrets: {}
      });
      setBundleResultSeed(null);
      resetPreflight();
    },
  });

  // Save preflight result when it completes (equivalent to onPreflightComplete callback)
  useEffect(() => {
    // Only trigger on new completions, not on initial load
    if (
      preflightResult &&
      preflightResult !== prevPreflightResultRef.current &&
      preflightRunStatus === "completed" &&
      scenarioName &&
      hasInitiallyLoaded
    ) {
      void saveStageResult("preflight", preflightResult, {
        preflight_result: preflightResult,
        preflight_error: null,
      });
    }
    prevPreflightResultRef.current = preflightResult;
  }, [preflightResult, preflightRunStatus, scenarioName, hasInitiallyLoaded, saveStageResult]);

  // Convert local state to FormState for server persistence
  const formStateForServer = useMemo((): Partial<FormState> => ({
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
    preflight_result: preflightResult,
    preflight_error: preflightError,
    preflight_override: preflightOverride,
    preflight_secrets: preflightSecrets,
    signing_enabled_for_build: signingEnabledForBuild,
    bundle_result: bundleResultSeed
  }), [
    selectedTemplate, appMetadata, deployment, output, platforms, connection,
    preflightResult, preflightError, preflightOverride, preflightSecrets,
    signingEnabledForBuild, bundleResultSeed
  ]);

  // Debounced save to server when form state changes
  // CRITICAL: Only save after initial load to prevent race condition
  const prevFormStateRef = useRef<string>("");
  const prevScenarioForSaveRef = useRef<string>(scenarioName);

  // Reset the previous form state ref when scenario changes to prevent stale comparisons
  useEffect(() => {
    if (prevScenarioForSaveRef.current !== scenarioName) {
      prevScenarioForSaveRef.current = scenarioName;
      prevFormStateRef.current = "";
    }
  }, [scenarioName]);

  useEffect(() => {
    if (!scenarioName) return;
    // Wait for initial load before saving - prevents overwriting server state with defaults
    if (!hasInitiallyLoaded) return;

    const serialized = JSON.stringify(formStateForServer);
    if (serialized === prevFormStateRef.current) return;
    prevFormStateRef.current = serialized;
    updateFormState(formStateForServer);
  }, [scenarioName, formStateForServer, updateFormState, hasInitiallyLoaded]);

  // Legacy compatibility - keep draft timestamps working
  const draftTimestamps = serverTimestamps;
  const draftLoadedScenario = serverFormState ? scenarioName : null;
  const clearDraft = clearState;

  // Handler for bundle export completion - saves bundle stage result
  const handleBundleComplete = useCallback(
    (result: BundleResult) => {
      if (!scenarioName || !hasInitiallyLoaded) return;
      // Update local seed state for immediate availability
      setBundleResultSeed(result);
      // Save bundle stage result with manifest path and bundle_result in form_state
      void saveStageResult("bundle", result, {
        bundle_manifest_path: result.manifestPath ?? undefined,
        bundle_result: result,
      });
    },
    [scenarioName, hasInitiallyLoaded, saveStageResult]
  );


  // Fetch available scenarios
  const { data: scenariosData, isLoading: loadingScenarios } = useQuery<ScenariosResponse>({
    queryKey: ['scenarios-desktop-status'],
    queryFn: fetchScenarioDesktopStatus,
  });

  // Use pipeline store for generating - this properly sets store state and starts polling
  const runStage = usePipelineStore((s) => s.runStage);
  const isGenerating = usePipelineStore(selectIsRunning);
  const isSubmittingGenerate = usePipelineStore(selectIsSubmitting);
  const storeErrorInfo = usePipelineStore((s) => s.errorInfo);
  const [generateError, setGenerateError] = useState<string | null>(null);

  const generateErrorMessage = generateError ?? (storeErrorInfo?.message || null);
  const isGenerateError = Boolean(generateError) || Boolean(storeErrorInfo);

  const connectionMutation = useMutation({
    mutationFn: async () => {
      if (!proxyUrl) {
        throw new Error("Enter the proxy URL above before testing.");
      }
      return probeEndpoints({ proxy_url: proxyUrl });
    },
    onSuccess: (result) => {
      setConnectionResult(result);
      setConnectionError(null);
    },
    onError: (error: Error) => {
      setConnectionError(error.message);
      setConnectionResult(null);
    }
  });

  useEffect(() => {
    if (!onGenerateStateChange) {
      return;
    }
    onGenerateStateChange({ pending: isSubmittingGenerate || isGenerating, error: generateErrorMessage });
  }, [isSubmittingGenerate, isGenerating, generateErrorMessage, onGenerateStateChange]);

  // Memoize selectedScenario to avoid recalculating on every render
  const selectedScenario = useMemo(
    () => scenariosData?.scenarios.find((s) => s.name === scenarioName),
    [scenariosData?.scenarios, scenarioName]
  );

  // Apply scenario defaults ONLY when selecting a new scenario (not when loading from server)
  useEffect(() => {
    if (!scenarioName) {
      if (!displayNameEdited) {
        setAppDisplayName("");
      }
      if (!descriptionEdited) {
        setAppDescription("");
      }
      if (!iconPathEdited) {
        setIconPath("");
      }
      return;
    }
    // CRITICAL: Wait for initial load to complete before deciding whether to apply defaults.
    // This prevents race conditions where defaults are applied before server state is processed.
    // hasInitiallyLoaded is set AFTER localFormState is updated from server response.
    if (!hasInitiallyLoaded) {
      return;
    }
    // Only skip applying defaults if the server has actual user-entered app metadata.
    // An empty serverFormState or one with empty metadata fields should allow defaults.
    const hasServerAppMetadata = serverFormState && (
      serverFormState.app_display_name ||
      serverFormState.app_description ||
      serverFormState.icon_path
    );
    if (hasServerAppMetadata) {
      return;
    }
    if (!selectedScenario) {
      return;
    }
    if (!displayNameEdited) {
      setAppDisplayName(selectedScenario.service_display_name || "");
    }
    if (!descriptionEdited) {
      setAppDescription(selectedScenario.service_description || "");
    }
    if (!iconPathEdited) {
      setIconPath(selectedScenario.service_icon_path || "");
    }
  }, [
    scenarioName,
    selectedScenario,
    hasInitiallyLoaded,
    serverFormState,
    displayNameEdited,
    descriptionEdited,
    iconPathEdited,
    setAppDisplayName,
    setAppDescription,
    setIconPath
  ]);

  const { data: proxyHints } = useQuery<ProxyHintsResponse | null>({
    queryKey: ['proxy-hints', scenarioName],
    queryFn: async () => {
      if (!scenarioName) return null;
      try {
        return await fetchProxyHints(scenarioName);
      } catch (error) {
        console.warn('Failed to load proxy hints', error);
        return null;
      }
    },
    enabled: Boolean(scenarioName),
    staleTime: 1000 * 60,
  });

  const { data: bundleManifestResp } = useQuery<BundleManifestResponse | null>({
    queryKey: ["bundle-manifest", bundleManifestPath.trim()],
    queryFn: () => {
      const path = bundleManifestPath.trim();
      if (!path) {
        return Promise.resolve(null);
      }
      return fetchBundleManifest({ bundle_manifest_path: path });
    },
    enabled: Boolean(bundleManifestPath.trim())
  });

  // Notify parent of form state changes needed by other sections (e.g., PreflightSection, BundleSection)
  useEffect(() => {
    if (!onFormStateChange) {
      return;
    }
    onFormStateChange({
      bundleManifestPath,
      isBundled,
      bundleManifest: bundleManifestResp?.manifest,
      onBundleManifestChange: setBundleManifestPath,
      onBundleComplete: handleBundleComplete,
      bundleHelperRef,
    });
  }, [
    bundleManifestPath,
    isBundled,
    bundleManifestResp?.manifest,
    onFormStateChange,
    setBundleManifestPath,
    handleBundleComplete,
    bundleHelperRef,
  ]);

  const applySavedConnection = useCallback((config?: DesktopConnectionConfig | null) => {
    if (!config) return;
    setDeploymentMode((config.deployment_mode as DeploymentMode) ?? DEFAULT_DEPLOYMENT_MODE);
    setProxyUrl(config.proxy_url ?? config.server_url ?? "");
    setAutoManageTier1(config.auto_manage_vrooli ?? false);
    setVrooliBinaryPath(config.vrooli_binary_path ?? "vrooli");
    setBundleManifestPath(config.bundle_manifest_path ?? "");
    if (config.app_display_name) {
      setAppDisplayName(config.app_display_name);
    }
    if (config.app_description) {
      setAppDescription(config.app_description);
    }
    if (config.icon) {
      setIconPath(config.icon);
    }
    if (config.server_type) {
      setServerType((config.server_type as ServerType) ?? DEFAULT_SERVER_TYPE);
    }
  }, [setDeploymentMode, setProxyUrl, setAutoManageTier1, setVrooliBinaryPath, setBundleManifestPath, setAppDisplayName, setAppDescription, setIconPath, setServerType]);

  useEffect(() => {
    if (!scenarioName) {
      return;
    }
    const connectionConfig = selectedScenario?.connection_config;
    const updatedAt = connectionConfig?.updated_at;
    if (!updatedAt) {
      return;
    }
    const configKey = `${scenarioName}:${updatedAt}`;
    if (configKey === lastLoadedScenario) {
      return;
    }
    if (draftLoadedScenario === scenarioName) {
      return;
    }
    applySavedConnection(connectionConfig);
    setLastLoadedScenario(configKey);
  }, [
    scenarioName,
    selectedScenario?.connection_config,
    lastLoadedScenario,
    draftLoadedScenario,
    applySavedConnection
  ]);

  useEffect(() => {
    if (connectionDecision.kind === "bundled-runtime") {
      if (serverType !== connectionDecision.effectiveServerType) {
        setServerType(connectionDecision.effectiveServerType);
      }
      if (!connectionDecision.allowsAutoManageTier1 && autoManageTier1) {
        setAutoManageTier1(false);
      }
    }
  }, [autoManageTier1, connectionDecision, serverType, setAutoManageTier1, setServerType]);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();

    try {
      // Clear previous errors
      clearValidationErrors();
      setGenerateError(null);

      const outputPathForRequest = locationMode === "custom" ? outputPath : "";

      // Use effective preflight values - fall back to server state when local state hasn't synced yet.
      // This handles the race condition where server state has been loaded with valid preflight data
      // but local state hasn't been updated yet.
      const effectivePreflightResult = preflightResult ?? serverFormState?.preflight_result ?? null;
      const effectivePreflightOk = effectivePreflightResult
        ? Boolean(
            effectivePreflightResult.validation?.valid &&
            effectivePreflightResult.ready?.ready &&
            missingPreflightSecrets.length === 0
          )
        : preflightOk;

      // Validate all inputs using the centralized validation function
      const errors = validateFormInputs({
        scenarioName,
        selectedPlatforms: selectedPlatformsList,
        isBundled,
        requiresProxyUrl: requiresRemoteConfig,
        bundleManifestPath,
        proxyUrl,
        appDisplayName,
        appDescription,
        locationMode,
        outputPath: outputPathForRequest,
        preflightResult: effectivePreflightResult,
        preflightOk: effectivePreflightOk,
        preflightOverride,
        signingEnabledForBuild,
        signingConfig,
        signingReadiness,
      });

      if (errors.length > 0) {
        setValidationErrors(errors);
        return;
      }

      // Use the pipeline store's runStage which properly sets store state,
      // starts polling, and updates all status-reading components
      const pipelineConfig: Partial<PipelineConfig> = {
        template_type: selectedTemplate,
        deployment_mode: deploymentMode === "bundled" ? "bundled" : "proxy",
        proxy_url: proxyUrl || undefined,
        platforms: selectedPlatformsList,
      };

      const pipelineId = await runStage("generate", pipelineConfig);
      onBuildStart?.(pipelineId);
    } catch (err) {
      console.error("[GeneratorForm] Unexpected error during submit:", err);
      const message = err instanceof Error ? err.message : "An unexpected error occurred. Check the browser console for details.";
      setGenerateError(message);
    }
  };

  const handleDeploymentChange = (nextMode: DeploymentMode) => {
    setDeploymentMode(nextMode);
    const nextAllowed: ServerType[] = nextMode === "bundled" || nextMode === "cloud-api"
      ? ["external"]
      : SERVER_TYPE_OPTIONS.map((option) => option.value);
    if (!nextAllowed.includes(serverType)) {
      setServerType(nextAllowed[0] ?? DEFAULT_SERVER_TYPE);
    }
  };

  const connectionTester = {
    isPending: connectionMutation.isPending,
    mutate: () => connectionMutation.mutate(),
  };

  const draftUpdatedLabel = draftTimestamps?.updatedAt
    ? new Date(draftTimestamps.updatedAt).toLocaleString()
    : null;
  const draftCreatedLabel = draftTimestamps?.createdAt
    ? new Date(draftTimestamps.createdAt).toLocaleString()
    : null;

  const handleReset = () => {
    if (!scenarioName) return;
    if (preflightPipelineId && preflightPending) {
      void cancelPreflightPipeline();
    }
    // Reset the pipeline on the backend (archives current, clears active slot)
    void resetCurrentPipeline();
    clearDraft();
    resetFormState(true);
  };

  // Use a ref to store the latest handleSubmit to avoid stale closures
  const handleSubmitRef = useRef(handleSubmit);
  handleSubmitRef.current = handleSubmit;

  // Create a stable submit function for external triggering
  const triggerSubmit = useCallback(() => {
    const syntheticEvent = {
      preventDefault: () => {},
    } as React.FormEvent;
    handleSubmitRef.current(syntheticEvent);
  }, []);

  // Expose submit handler to parent when requested
  useEffect(() => {
    if (!onSubmitHandlerReady) {
      return;
    }
    onSubmitHandlerReady(triggerSubmit);
  }, [onSubmitHandlerReady, triggerSubmit]);

  // Expose validation state to parent when requested
  useEffect(() => {
    if (!onValidationStateChange) {
      return;
    }
    onValidationStateChange({
      errors: validationErrors,
      clearErrors: clearValidationErrors,
      isPending: isSubmittingGenerate || isGenerating,
      isError: isGenerateError,
      errorMessage: generateErrorMessage,
      isUpdateMode,
    });
  }, [
    onValidationStateChange,
    validationErrors,
    clearValidationErrors,
    isSubmittingGenerate,
    isGenerating,
    isGenerateError,
    generateErrorMessage,
    isUpdateMode,
  ]);

  return (
    <div className="space-y-4">
      <GeneratorFormHeader
        scenarioName={scenarioName}
        validationStatus={validationStatus}
        createdLabel={draftCreatedLabel}
        updatedLabel={draftUpdatedLabel}
        isSaving={stateSaving}
        onReset={handleReset}
      />

      {isStale && pendingChanges.length > 0 && (
        <PendingChangesAlert
          changes={pendingChanges}
          onDismiss={() => {
            // Dismiss by refreshing the page or invalidating cache
          }}
        />
      )}
      <form id={formId} onSubmit={handleSubmit} className="space-y-4">
          {selectionSource === "inventory" && scenarioName && (
            <div className="rounded-md border border-blue-800/60 bg-blue-950/30 px-2.5 py-1.5 text-xs md:text-sm text-blue-100">
              <span className="font-semibold text-blue-50">
                Loaded: {scenarioName}.
              </span>{" "}
              <span className="text-blue-100/90">
                Settings below regenerate the desktop wrapper&mdash;scenario code stays the same.
              </span>
            </div>
          )}
          <ScenarioSelector
            scenarioName={scenarioName}
            loadingScenarios={loadingScenarios}
            selectedScenario={selectedScenario}
            onOpenScenarioModal={() => openModal('scenario')}
            onLoadSaved={
              selectedScenario?.connection_config
                ? () => applySavedConnection(selectedScenario.connection_config)
                : undefined
            }
            locked={scenarioLocked}
            onUnlock={() => setScenarioLocked(false)}
          />

          <AppMetadataSection
            scenarioName={scenarioName}
            appDisplayName={appDisplayName}
            onAppDisplayNameChange={setAppDisplayName}
            iconPath={iconPath}
            onIconPathChange={setIconPath}
            iconPreviewUrl={iconPreviewUrl}
            iconPreviewError={iconPreviewError}
            onIconPreviewError={setIconPreviewError}
            appDescription={appDescription}
            onAppDescriptionChange={setAppDescription}
          />

          <FrameworkTemplateSection
            framework={framework}
            onOpenFrameworkModal={() => openModal('framework')}
            selectedTemplate={selectedTemplate}
            onOpenTemplateModal={() => openModal('template')}
          />

          <DeploymentSummarySection
            deploymentMode={deploymentMode}
            serverType={serverType}
            onOpenDeploymentModal={() => openModal('deployment')}
          />

          <ConnectionSectionRouter
            connectionDecision={connectionDecision}
            scenarioName={scenarioName}
            proxyUrl={proxyUrl}
            onProxyUrlChange={setProxyUrl}
            proxyHints={proxyHints}
            connectionTester={connectionTester}
            connectionResult={connectionResult}
            connectionError={connectionError}
            autoManageTier1={autoManageTier1}
            onAutoManageTier1Change={setAutoManageTier1}
            vrooliBinaryPath={vrooliBinaryPath}
            onVrooliBinaryPathChange={setVrooliBinaryPath}
            serverPort={serverPort}
            onServerPortChange={setServerPort}
            localServerPath={localServerPath}
            onLocalServerPathChange={setLocalServerPath}
            localApiEndpoint={localApiEndpoint}
            onLocalApiEndpointChange={setLocalApiEndpoint}
          />

          <PlatformSelector platforms={platforms} onPlatformChange={handlePlatformChange} />

          <SigningInlineSection
            scenarioName={scenarioName}
            signingEnabled={signingEnabledForBuild}
            signingConfig={signingConfig}
            readiness={signingReadiness}
            loading={signingLoading}
            onToggleSigning={setSigningEnabledForBuild}
            onOpenSigning={() => onOpenSigningTab(scenarioName)}
            onRefresh={() => {
              refreshSigning();
            }}
          />

          <OutputLocationSelector
            locationMode={locationMode}
            onChange={(mode) => {
              setLocationMode(mode);
              if (mode !== "custom") {
                setOutputPath("");
              }
            }}
            standardPath={standardOutputPath}
            stagingPreview={stagingPreviewPath}
          />

          {isCustomLocation && (
            <OutputPathField
              outputPath={outputPath}
              onOutputPathChange={(value) => {
                setOutputPath(value);
              }}
            />
          )}

          <input type="hidden" name="scenarioName" value={scenarioName} />

          {showSubmit && (
            <GeneratorFormFooter
              validationErrors={validationErrors}
              onDismissErrors={clearValidationErrors}
              isPending={isSubmittingGenerate || isGenerating}
              isError={isGenerateError}
              errorMessage={generateErrorMessage}
              isUpdateMode={isUpdateMode}
            />
          )}
      </form>
      <GeneratorModalsContainer
        modals={modals}
        closeModal={closeModal}
        loadingScenarios={loadingScenarios}
        scenarios={scenariosData?.scenarios ?? []}
        selectedScenarioName={scenarioName}
        onScenarioSelect={onScenarioNameChange}
        selectedTemplate={selectedTemplate}
        onTemplateSelect={onTemplateChange}
        selectedFramework={framework}
        onFrameworkSelect={setFramework}
        deploymentMode={deploymentMode}
        serverType={serverType}
        allowedServerTypes={allowedServerTypes}
        onDeploymentChange={(nextMode, nextServerType) => {
          handleDeploymentChange(nextMode);
          if (nextServerType) {
            setServerType(nextServerType);
          }
        }}
      />
    </div>
  );
}
