import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import {
  fetchProxyHints,
  fetchScenarioDesktopStatus,
  getIconPreviewUrl,
  runPipeline,
  fetchBundleManifest,
  probeEndpoints,
  type BundlePreflightResponse,
  type BundleManifestResponse,
  type ProbeResponse,
  type ProxyHintsResponse,
  type PipelineConfig,
} from "../lib/api";
import { Input } from "./ui/input";
import { Label } from "./ui/label";
import { Button } from "./ui/button";
import { TemplateModal } from "./TemplateModal";
import { ScenarioModal } from "./ScenarioModal";
import { FrameworkModal } from "./FrameworkModal";
import { DeploymentModal } from "./DeploymentModal";
import { BundledPreflightSection } from "./BundledPreflightSection";
import type { DeploymentManagerBundleHelperHandle, BundleResult } from "./DeploymentManagerBundleHelper";
import { BundledRuntimeSection } from "./BundledRuntimeSection";
import { ExternalServerSection } from "./ExternalServerSection";
import { EmbeddedServerSection } from "./EmbeddedServerSection";
import { DeploymentSummarySection } from "./DeploymentSummarySection";
import { PlatformSelector } from "./PlatformSelector";
import type { DesktopConnectionConfig, ScenariosResponse } from "./scenario-inventory/types";
import {
  ScenarioSelector,
  FrameworkTemplateSection,
  SigningInlineSection,
  OutputLocationSelector,
  OutputPathField,
  ValidationErrors,
  validateFormInputs,
  type ValidationError,
} from "./generator";
import {
  DEFAULT_DEPLOYMENT_MODE,
  DEFAULT_SERVER_TYPE,
  SERVER_TYPE_OPTIONS,
  decideConnection,
  type DeploymentMode,
  type ServerType
} from "../domain/deployment";
import {
  computeStandardOutputPath,
  computeStagingPreviewPath,
  getSelectedPlatforms,
  resolveEndpoints,
  type OutputLocation,
  type PlatformSelection
} from "../domain/generator";
import {
  useScenarioState,
  usePreflightSession,
  useSigningConfig,
} from "../hooks";
import { PipelineStatusSummary } from "./state/PipelineStatusOverview";
import { PendingChangesAlert } from "./state/PendingChangesAlert";
import type { FormState } from "../lib/api";

interface GeneratorFormProps {
  selectedTemplate: string;
  onTemplateChange: (template: string) => void;
  onBuildStart: (buildId: string) => void;
  scenarioName: string;
  onScenarioNameChange: (name: string) => void;
  selectionSource?: "inventory" | "manual" | null;
  onOpenSigningTab: (scenario?: string) => void;
  formId?: string;
  showSubmit?: boolean;
  onGenerateStateChange?: (state: { pending: boolean; error: string | null }) => void;
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
  onGenerateStateChange
}: GeneratorFormProps) {
  const [scenarioLocked, setScenarioLocked] = useState(selectionSource === "inventory");
  const [appDisplayName, setAppDisplayName] = useState("");
  const [appDescription, setAppDescription] = useState("");
  const [iconPath, setIconPath] = useState("");
  const [displayNameEdited, setDisplayNameEdited] = useState(false);
  const [descriptionEdited, setDescriptionEdited] = useState(false);
  const [iconPathEdited, setIconPathEdited] = useState(false);
  const [iconPreviewError, setIconPreviewError] = useState(false);
  const [framework, setFramework] = useState("electron");
  const [frameworkModalOpen, setFrameworkModalOpen] = useState(false);
  const [serverType, setServerType] = useState<ServerType>(DEFAULT_SERVER_TYPE);
  const [deploymentMode, setDeploymentMode] = useState<DeploymentMode>(DEFAULT_DEPLOYMENT_MODE);
  const [platforms, setPlatforms] = useState<PlatformSelection>({
    win: true,
    mac: true,
    linux: true
  });
  const [locationMode, setLocationMode] = useState<OutputLocation>("proper");
  const [outputPath, setOutputPath] = useState("");
  const [proxyUrl, setProxyUrl] = useState("");
  const [bundleManifestPath, setBundleManifestPath] = useState("");
  const [serverPort, setServerPort] = useState(3000);
  const [localServerPath, setLocalServerPath] = useState("ui/server.js");
  const [localApiEndpoint, setLocalApiEndpoint] = useState("http://localhost:3001/api");
  const [autoManageTier1, setAutoManageTier1] = useState(false);
  const [vrooliBinaryPath, setVrooliBinaryPath] = useState("vrooli");
  const [scenarioModalOpen, setScenarioModalOpen] = useState(false);
  const [templateModalOpen, setTemplateModalOpen] = useState(false);
  const [deploymentModalOpen, setDeploymentModalOpen] = useState(false);
  const [connectionResult, setConnectionResult] = useState<ProbeResponse | null>(null);
  const [connectionError, setConnectionError] = useState<string | null>(null);
  const [preflightSeed, setPreflightSeed] = useState({
    result: null as BundlePreflightResponse | null,
    error: null as string | null,
    override: false,
    secrets: {} as Record<string, string>
  });
  // Bundle result seed from server state - similar pattern to preflightSeed
  const [bundleResultSeed, setBundleResultSeed] = useState<BundleResult | null>(null);
  const [deploymentManagerUrl, setDeploymentManagerUrl] = useState<string | null>(null);
  const [lastLoadedScenario, setLastLoadedScenario] = useState<string | null>(null);
  const [validationErrors, setValidationErrors] = useState<ValidationError[]>([]);
  const isUpdateMode = selectionSource === "inventory";
  const bundleHelperRef = useRef<DeploymentManagerBundleHelperHandle>(null);

  const connectionDecision = useMemo(
    () => decideConnection(deploymentMode, serverType),
    [deploymentMode, serverType]
  );
  const isBundled = connectionDecision.kind === "bundled-runtime";
  const requiresRemoteConfig = connectionDecision.requiresProxyUrl;
  const standardOutputPath = useMemo(
    () => computeStandardOutputPath(scenarioName),
    [scenarioName]
  );
  const stagingPreviewPath = useMemo(
    () => computeStagingPreviewPath(scenarioName),
    [scenarioName]
  );
  const selectedPlatformsList = useMemo(
    () => getSelectedPlatforms(platforms),
    [platforms]
  );
  const resolvedEndpoints = useMemo(
    () =>
      resolveEndpoints({
        decision: connectionDecision,
        proxyUrl,
        localServerPath,
        localApiEndpoint
      }),
    [connectionDecision, proxyUrl, localServerPath, localApiEndpoint]
  );
  const {
    result: preflightResult,
    error: preflightError,
    pending: preflightPending,
    pipelineId: preflightPipelineId,
    pipelineStatus: preflightPipelineStatus,
    override: preflightOverride,
    secrets: preflightSecrets,
    missingSecrets: missingPreflightSecrets,
    preflightOk,
    setOverride: setPreflightOverride,
    setSecret: setPreflightSecret,
    runPreflight,
    cancelPreflight: cancelPreflightPipeline,
    reset: resetPreflight
  } = usePreflightSession({
    scenarioName,
    bundleManifestPath,
    isBundled,
    initialResult: preflightSeed.result,
    initialError: preflightSeed.error,
    initialOverride: preflightSeed.override,
    initialSecrets: preflightSeed.secrets,
    onPreflightComplete: (result) => {
      if (scenarioName && hasInitiallyLoaded) {
        void saveStageResult("preflight", result, {
          preflight_result: result,
          preflight_error: null,
        });
      }
    }
  });

  const {
    config: signingConfig,
    readiness: signingReadiness,
    loading: signingLoading,
    enabledForBuild: signingEnabledForBuild,
    setEnabledForBuild: setSigningEnabledForBuild,
    refreshAll: refreshSigning
  } = useSigningConfig({ scenarioName });


  const isCustomLocation = locationMode === "custom";
  const allowedServerTypes = useMemo<ServerType[]>(() => {
    if (deploymentMode === "bundled") {
      return ["external"];
    }
    if (deploymentMode === "cloud-api") {
      return ["external"];
    }
    return SERVER_TYPE_OPTIONS.map((option) => option.value);
  }, [deploymentMode]);

  useEffect(() => {
    if (!allowedServerTypes.includes(serverType)) {
      setServerType(allowedServerTypes[0] ?? DEFAULT_SERVER_TYPE);
    }
  }, [allowedServerTypes, serverType]);

  useEffect(() => {
    setScenarioLocked(selectionSource === "inventory");
  }, [selectionSource]);

  const resetFormState = useCallback((resetTemplate: boolean) => {
    setAppDisplayName("");
    setAppDescription("");
    setIconPath("");
    setDisplayNameEdited(false);
    setDescriptionEdited(false);
    setIconPathEdited(false);
    setIconPreviewError(false);
    setFramework("electron");
    setServerType(DEFAULT_SERVER_TYPE);
    setDeploymentMode(DEFAULT_DEPLOYMENT_MODE);
    setPlatforms({ win: true, mac: true, linux: true });
    setLocationMode("proper");
    setOutputPath("");
    setProxyUrl("");
    setBundleManifestPath("");
    setServerPort(3000);
    setLocalServerPath("ui/server.js");
    setLocalApiEndpoint("http://localhost:3001/api");
    setAutoManageTier1(false);
    setVrooliBinaryPath("vrooli");
    setConnectionResult(null);
    setConnectionError(null);
    setDeploymentManagerUrl(null);
    setSigningEnabledForBuild(false);
    setLastLoadedScenario(null);
    setValidationErrors([]);
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
  }, [onTemplateChange, resetPreflight]);

  // Server-side state persistence via useScenarioState
  const {
    formState: serverFormState,
    isLoading: stateLoading,
    hasInitiallyLoaded,
    isSaving: stateSaving,
    isStale,
    pendingChanges,
    validationStatus,
    timestamps: serverTimestamps,
    updateFormState,
    saveStageResult,
    clearState,
    stages,
  } = useScenarioState({
    scenarioName,
    enabled: Boolean(scenarioName),
    onStateLoaded: (state) => {
      if (!state.form_state) return;
      const fs = state.form_state;
      // Apply form state from server
      setAppDisplayName(fs.app_display_name || "");
      setAppDescription(fs.app_description || "");
      setIconPath(fs.icon_path || "");
      setDisplayNameEdited(fs.display_name_edited || false);
      setDescriptionEdited(fs.description_edited || false);
      setIconPathEdited(fs.icon_path_edited || false);
      setFramework(fs.framework || "electron");
      setServerType((fs.server_type as ServerType) ?? DEFAULT_SERVER_TYPE);
      setDeploymentMode((fs.deployment_mode as DeploymentMode) ?? DEFAULT_DEPLOYMENT_MODE);
      setPlatforms({
        win: fs.platforms?.win ?? true,
        mac: fs.platforms?.mac ?? true,
        linux: fs.platforms?.linux ?? true,
      });
      setLocationMode((fs.location_mode as OutputLocation) ?? "proper");
      setOutputPath(fs.output_path ?? "");
      setProxyUrl(fs.proxy_url ?? "");
      setBundleManifestPath(fs.bundle_manifest_path ?? "");
      setServerPort(fs.server_port ?? 3000);
      setLocalServerPath(fs.local_server_path ?? "ui/server.js");
      setLocalApiEndpoint(fs.local_api_endpoint ?? "http://localhost:3001/api");
      setAutoManageTier1(fs.auto_manage_tier1 ?? false);
      setVrooliBinaryPath(fs.vrooli_binary_path ?? "vrooli");
      setConnectionResult((fs.connection_result as ProbeResponse | null) ?? null);
      setConnectionError(fs.connection_error ?? null);
      setDeploymentManagerUrl(fs.deployment_manager_url ?? null);
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

  // Convert local state to FormState for server persistence
  const formStateForServer = useMemo((): Partial<FormState> => ({
    selected_template: selectedTemplate,
    app_display_name: appDisplayName,
    app_description: appDescription,
    icon_path: iconPath,
    display_name_edited: displayNameEdited,
    description_edited: descriptionEdited,
    icon_path_edited: iconPathEdited,
    framework,
    server_type: serverType,
    deployment_mode: deploymentMode,
    platforms,
    location_mode: locationMode,
    output_path: outputPath,
    proxy_url: proxyUrl,
    bundle_manifest_path: bundleManifestPath,
    server_port: serverPort,
    local_server_path: localServerPath,
    local_api_endpoint: localApiEndpoint,
    auto_manage_tier1: autoManageTier1,
    vrooli_binary_path: vrooliBinaryPath,
    connection_result: connectionResult,
    connection_error: connectionError,
    preflight_result: preflightResult,
    preflight_error: preflightError,
    preflight_override: preflightOverride,
    preflight_secrets: preflightSecrets,
    deployment_manager_url: deploymentManagerUrl,
    signing_enabled_for_build: signingEnabledForBuild,
    bundle_result: bundleResultSeed
  }), [
    selectedTemplate, appDisplayName, appDescription, iconPath,
    displayNameEdited, descriptionEdited, iconPathEdited,
    framework, serverType, deploymentMode, platforms,
    locationMode, outputPath, proxyUrl, bundleManifestPath,
    serverPort, localServerPath, localApiEndpoint,
    autoManageTier1, vrooliBinaryPath, connectionResult, connectionError,
    preflightResult, preflightError, preflightOverride, preflightSecrets,
    deploymentManagerUrl, signingEnabledForBuild, bundleResultSeed
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

  // Note: Preflight results are now saved via the onPreflightComplete callback in usePreflightSession

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
        deployment_manager_url: result.deploymentManagerUrl,
        bundle_result: result,
      });
    },
    [scenarioName, hasInitiallyLoaded, saveStageResult]
  );

  // Bundle result for restoration - use seed loaded from form_state.bundle_result
  // This mirrors how preflight uses preflightResultSeed from form_state.preflight_result
  const initialBundleResult = bundleResultSeed;

  // Fetch available scenarios
  const { data: scenariosData, isLoading: loadingScenarios } = useQuery<ScenariosResponse>({
    queryKey: ['scenarios-desktop-status'],
    queryFn: fetchScenarioDesktopStatus,
  });

  const generateMutation = useMutation({
    mutationFn: (config: PipelineConfig) => runPipeline(config),
    onSuccess: (data) => {
      onBuildStart(data.pipeline_id);
    }
  });
  const generateErrorMessage = generateMutation.isError
    ? (generateMutation.error as Error).message
    : null;

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
    onGenerateStateChange({ pending: generateMutation.isPending, error: generateErrorMessage });
  }, [generateMutation.isPending, generateErrorMessage, onGenerateStateChange]);

  // Memoize selectedScenario to avoid recalculating on every render
  const selectedScenario = useMemo(
    () => scenariosData?.scenarios.find((s) => s.name === scenarioName),
    [scenariosData?.scenarios, scenarioName]
  );

  const iconPreviewUrl = useMemo(
    () => (iconPath ? getIconPreviewUrl(iconPath) : ""),
    [iconPath]
  );

  useEffect(() => {
    setIconPreviewError(false);
  }, [iconPreviewUrl]);

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
    // Skip applying scenario defaults when we've loaded state from the server.
    // This prevents overwriting persisted values with scenario metadata on refresh.
    if (serverFormState) {
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
    iconPathEdited
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

  const applySavedConnection = (config?: DesktopConnectionConfig | null) => {
    if (!config) return;
    setDeploymentMode((config.deployment_mode as DeploymentMode) ?? DEFAULT_DEPLOYMENT_MODE);
    setProxyUrl(config.proxy_url ?? config.server_url ?? "");
    setAutoManageTier1(config.auto_manage_vrooli ?? false);
    setVrooliBinaryPath(config.vrooli_binary_path ?? "vrooli");
    setBundleManifestPath(config.bundle_manifest_path ?? "");
    if (config.app_display_name) {
      setAppDisplayName(config.app_display_name);
      setDisplayNameEdited(true);
    }
    if (config.app_description) {
      setAppDescription(config.app_description);
      setDescriptionEdited(true);
    }
    if (config.icon) {
      setIconPath(config.icon);
      setIconPathEdited(true);
    }
    if (config.server_type) {
      setServerType((config.server_type as ServerType) ?? DEFAULT_SERVER_TYPE);
    }
  };

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
    draftLoadedScenario
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
  }, [autoManageTier1, connectionDecision, serverType]);

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();

    // Clear previous validation errors
    setValidationErrors([]);

    const outputPathForRequest = locationMode === "custom" ? outputPath : "";

    // Use effective preflight values - fall back to server state when hook hasn't synced yet.
    // This handles the race condition where usePreflightSession's effect hasn't run yet
    // but server state has been loaded with valid preflight data.
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
      // Scroll to show validation errors
      window.scrollTo({ top: 0, behavior: "smooth" });
      return;
    }

    // Build pipeline config from form values
    const pipelineConfig: PipelineConfig = {
      scenario_name: scenarioName,
      template_type: selectedTemplate,
      deployment_mode: deploymentMode === "bundled" ? "bundled" : "proxy",
      proxy_url: proxyUrl || undefined,
      platforms: selectedPlatformsList,
      stop_after_stage: "generate", // Only run up to generate stage
    };

    generateMutation.mutate(pipelineConfig);
  };

  const handlePlatformChange = (platform: string, checked: boolean) => {
    setPlatforms((prev) => ({ ...prev, [platform]: checked }));
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

  const connectionSection =
    connectionDecision.kind === "bundled-runtime" ? (
      <BundledRuntimeSection
        bundleManifestPath={bundleManifestPath}
        onBundleManifestChange={setBundleManifestPath}
        scenarioName={scenarioName}
        bundleHelperRef={bundleHelperRef}
        onDeploymentManagerUrlChange={setDeploymentManagerUrl}
        onBundleExported={(manifestPath) => {
          runPreflight(undefined, { bundle_manifest_path: manifestPath });
        }}
        onBundleComplete={handleBundleComplete}
        initialBundleResult={initialBundleResult}
      />
    ) : connectionDecision.kind === "remote-server" ? (
      <ExternalServerSection
        proxyUrl={proxyUrl}
        onProxyUrlChange={setProxyUrl}
        scenarioName={scenarioName}
        proxyHints={proxyHints}
        connectionTester={{ isPending: connectionMutation.isPending, mutate: () => connectionMutation.mutate() }}
        connectionResult={connectionResult}
        connectionError={connectionError}
        autoManageTier1={autoManageTier1}
        onAutoManageTier1Change={setAutoManageTier1}
        vrooliBinaryPath={vrooliBinaryPath}
        onVrooliBinaryPathChange={setVrooliBinaryPath}
      />
    ) : (
      <EmbeddedServerSection
        serverPort={serverPort}
        onServerPortChange={setServerPort}
        localServerPath={localServerPath}
        onLocalServerPathChange={setLocalServerPath}
        localApiEndpoint={localApiEndpoint}
        onLocalApiEndpointChange={setLocalApiEndpoint}
      />
    );

  const draftUpdatedLabel = draftTimestamps?.updatedAt
    ? new Date(draftTimestamps.updatedAt).toLocaleString()
    : null;
  const draftCreatedLabel = draftTimestamps?.createdAt
    ? new Date(draftTimestamps.createdAt).toLocaleString()
    : null;

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex flex-wrap items-center gap-3">
          {scenarioName && (
            <PipelineStatusSummary validationStatus={validationStatus} />
          )}
          {(draftCreatedLabel || draftUpdatedLabel) && (
            <div className="flex flex-wrap items-center gap-2 text-xs text-slate-400">
              {draftCreatedLabel && (
                <span>Started {draftCreatedLabel}</span>
              )}
              {draftUpdatedLabel && (
                <span>Saved {draftUpdatedLabel}</span>
              )}
              {stateSaving && (
                <span className="text-blue-400">Saving...</span>
              )}
            </div>
          )}
        </div>
        <Button
          type="button"
          size="sm"
          variant="outline"
          className="sm:ml-auto"
          disabled={!scenarioName}
          onClick={() => {
            if (!scenarioName) return;
            if (preflightPipelineId && preflightPending) {
              void cancelPreflightPipeline();
            }
            clearDraft();
            resetFormState(true);
          }}
        >
          Reset progress
        </Button>
      </div>

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
            <div className="rounded-lg border border-blue-800/60 bg-blue-950/30 px-3 py-2 text-sm text-blue-100">
              <div className="font-semibold text-blue-50">
                Loaded from Scenario Inventory: <span className="font-semibold">{scenarioName}</span>.
              </div>
              <p className="text-blue-100/90">
                We&apos;ll regenerate the desktop wrapper with the settings below—your scenario code stays the same.
              </p>
            </div>
          )}
          <ScenarioSelector
            scenarioName={scenarioName}
            loadingScenarios={loadingScenarios}
            selectedScenario={selectedScenario}
            onOpenScenarioModal={() => setScenarioModalOpen(true)}
            onLoadSaved={
              selectedScenario?.connection_config
                ? () => applySavedConnection(selectedScenario.connection_config)
                : undefined
            }
            locked={scenarioLocked}
            onUnlock={() => setScenarioLocked(false)}
          />

          <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4 space-y-3">
            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <Label htmlFor="appDisplayName">App display name</Label>
                <Input
                  id="appDisplayName"
                  value={appDisplayName}
                  onChange={(e) => {
                    setAppDisplayName(e.target.value);
                    setDisplayNameEdited(true);
                  }}
                  placeholder={`${scenarioName || "scenario"} Desktop`}
                  className="mt-1.5"
                />
                <p className="mt-1 text-xs text-slate-400">
                  Controls window titles, installer product name, and tray labels.
                </p>
              </div>
              <div>
                <Label htmlFor="iconPath">Icon path (PNG)</Label>
                <Input
                  id="iconPath"
                  value={iconPath}
                  onChange={(e) => {
                    setIconPath(e.target.value);
                    setIconPathEdited(true);
                  }}
                  placeholder="/home/you/Vrooli/scenarios/picker-wheel/icon.png"
                  className="mt-1.5"
                />
                <p className="mt-1 text-xs text-slate-400">
                  Optional 256px+ PNG; it will be copied into <code>assets/icon.png</code> for the build.
                </p>
                <div className="mt-3 flex items-center gap-3 rounded-md border border-slate-800 bg-slate-950/60 p-3">
                  <div className="flex h-12 w-12 items-center justify-center rounded-md border border-slate-700 bg-slate-900">
                    {iconPreviewUrl && !iconPreviewError ? (
                      <img
                        src={iconPreviewUrl}
                        alt="Icon preview"
                        className="h-10 w-10 rounded object-contain"
                        onError={() => setIconPreviewError(true)}
                      />
                    ) : (
                      <span className="text-xs text-slate-500">No icon</span>
                    )}
                  </div>
                  <div className="text-xs text-slate-400">
                    {iconPreviewUrl && !iconPreviewError
                      ? "Previewing selected icon."
                      : "Preview will appear once a valid PNG path is set."}
                  </div>
                </div>
              </div>
            </div>
            <div>
              <Label htmlFor="appDescription">App description</Label>
              <textarea
                id="appDescription"
                value={appDescription}
                onChange={(e) => {
                  setAppDescription(e.target.value);
                  setDescriptionEdited(true);
                }}
                className="mt-1.5 w-full rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 shadow-sm focus:border-blue-600 focus:outline-none"
                rows={3}
                placeholder={`Desktop application for ${scenarioName || "your scenario"} scenario`}
              />
              <p className="mt-1 text-xs text-slate-400">
                Shown in generated README and metadata.
              </p>
            </div>
          </div>

          <FrameworkTemplateSection
            framework={framework}
            onOpenFrameworkModal={() => setFrameworkModalOpen(true)}
            selectedTemplate={selectedTemplate}
            onOpenTemplateModal={() => setTemplateModalOpen(true)}
          />

          <DeploymentSummarySection
            deploymentMode={deploymentMode}
            serverType={serverType}
            onOpenDeploymentModal={() => setDeploymentModalOpen(true)}
          />

          {connectionSection}

          {isBundled && (
            <BundledPreflightSection
              bundleManifestPath={bundleManifestPath}
              bundleManifest={bundleManifestResp?.manifest}
              preflightResult={preflightResult}
              preflightPending={preflightPending}
              preflightError={preflightError}
              pipelineStatus={preflightPipelineStatus}
              missingSecrets={missingPreflightSecrets}
              secretInputs={preflightSecrets}
              preflightOk={preflightOk}
              preflightOverride={preflightOverride}
              preflightLogTails={preflightResult?.log_tails}
              onOverrideChange={setPreflightOverride}
              onSecretChange={setPreflightSecret}
              onRun={runPreflight}
              onCancel={cancelPreflightPipeline}
            />
          )}

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

          {/* Validation errors - shown above submit button */}
          <ValidationErrors
            errors={validationErrors}
            onDismiss={() => setValidationErrors([])}
          />

          {showSubmit && (
            <>
              <Button
                type="submit"
                className="w-full"
                disabled={generateMutation.isPending || validationErrors.length > 0}
              >
                {generateMutation.isPending
                  ? "Generating..."
                  : isUpdateMode
                    ? "Update Desktop Application"
                    : "Generate Desktop Application"}
              </Button>

              {generateMutation.isError && (
                <div className="rounded-lg bg-red-900/20 p-3 text-sm text-red-300">
                  <strong>Error:</strong> {(generateMutation.error as Error).message}
                </div>
              )}
            </>
          )}
      </form>
      <ScenarioModal
          open={scenarioModalOpen}
          loading={loadingScenarios}
          scenarios={scenariosData?.scenarios ?? []}
          selectedScenarioName={scenarioName}
          onClose={() => setScenarioModalOpen(false)}
          onSelect={(name) => {
            onScenarioNameChange(name);
            setScenarioModalOpen(false);
          }}
        />
        <TemplateModal
          open={templateModalOpen}
          selectedTemplate={selectedTemplate}
          onClose={() => setTemplateModalOpen(false)}
          onSelect={(template) => {
            onTemplateChange(template);
            setTemplateModalOpen(false);
          }}
        />
        <FrameworkModal
          open={frameworkModalOpen}
          selectedFramework={framework}
          onClose={() => setFrameworkModalOpen(false)}
          onSelect={(nextFramework) => {
            setFramework(nextFramework);
            setFrameworkModalOpen(false);
          }}
        />
        <DeploymentModal
          open={deploymentModalOpen}
          deploymentMode={deploymentMode}
          serverType={serverType}
          allowedServerTypes={allowedServerTypes}
          onClose={() => setDeploymentModalOpen(false)}
          onChange={(nextMode, nextServerType) => {
            handleDeploymentChange(nextMode);
            if (nextServerType) {
              setServerType(nextServerType);
            }
          }}
        />
    </div>
  );
}
