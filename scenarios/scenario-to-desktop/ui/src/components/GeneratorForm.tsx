import { useEffect, useMemo, useRef, useState, type FormEvent } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import {
  fetchProxyHints,
  fetchScenarioDesktopStatus,
  getIconPreviewUrl,
  generateDesktop,
  fetchBundleManifest,
  probeEndpoints,
  type BundlePreflightResponse,
  type BundleManifestResponse,
  type ProbeResponse,
  type ProxyHintsResponse,
  type SigningConfig,
  type SigningReadinessResponse
} from "../lib/api";
import { Input } from "./ui/input";
import { Label } from "./ui/label";
import { Button } from "./ui/button";
import { AlertTriangle, ExternalLink, RefreshCw, ShieldCheck } from "lucide-react";
import { TemplateModal } from "./TemplateModal";
import { ScenarioModal } from "./ScenarioModal";
import { FrameworkModal } from "./FrameworkModal";
import { DeploymentModal } from "./DeploymentModal";
import { BundledPreflightSection } from "./BundledPreflightSection";
import type { DeploymentManagerBundleHelperHandle } from "./DeploymentManagerBundleHelper";
import { BundledRuntimeSection } from "./BundledRuntimeSection";
import { ExternalServerSection } from "./ExternalServerSection";
import { EmbeddedServerSection } from "./EmbeddedServerSection";
import { DeploymentSummarySection } from "./DeploymentSummarySection";
import { PlatformSelector } from "./PlatformSelector";
import type { DesktopConnectionConfig, ScenarioDesktopStatus, ScenariosResponse } from "./scenario-inventory/types";
import {
  DEFAULT_DEPLOYMENT_MODE,
  DEFAULT_SERVER_TYPE,
  SERVER_TYPE_OPTIONS,
  decideConnection,
  type DeploymentMode,
  type ServerType
} from "../domain/deployment";
import {
  buildDesktopConfig,
  computeStandardOutputPath,
  computeStagingPreviewPath,
  getSelectedPlatforms,
  resolveEndpoints,
  validateGeneratorInputs,
  type OutputLocation,
  type PlatformSelection
} from "../domain/generator";
import {
  useGeneratorDraft,
  usePreflightSession,
  useSigningConfig,
  type GeneratorDraftState
} from "../hooks";

const TEMPLATE_SUMMARIES: Record<string, { name: string; description: string }> = {
  basic: { name: "Basic", description: "Balanced single window wrapper" },
  advanced: { name: "Advanced", description: "Tray, shortcuts, deep OS touches" },
  multi_window: { name: "Multi-Window", description: "Multiple coordinated windows" },
  kiosk: { name: "Kiosk Mode", description: "Locked-down fullscreen kiosk" },
  universal: { name: "Universal Desktop App", description: "All-purpose desktop wrapper" }
};

const FRAMEWORK_SUMMARIES: Record<string, { name: string; description: string }> = {
  electron: { name: "Electron", description: "Most compatible and battle-tested for desktop web apps" },
  tauri: { name: "Tauri", description: "Rust + system webview for smaller, more secure apps" },
  neutralino: { name: "Neutralino", description: "Ultra-lightweight desktop wrapper with minimal runtime" }
};

interface ScenarioSelectorProps {
  scenarioName: string;
  loadingScenarios: boolean;
  selectedScenario?: ScenarioDesktopStatus;
  onOpenScenarioModal: () => void;
  onLoadSaved?: () => void;
  locked?: boolean;
  onUnlock?: () => void;
}

function ScenarioSelector({
  scenarioName,
  loadingScenarios,
  selectedScenario,
  onOpenScenarioModal,
  onLoadSaved,
  locked = false,
  onUnlock
}: ScenarioSelectorProps) {
  const displayName = selectedScenario?.display_name || scenarioName;

  if (locked && scenarioName) {
    return (
      <div className="rounded-lg border border-slate-800 bg-slate-950/60 p-3 flex items-center justify-between gap-3">
        <div>
          <p className="text-xs uppercase tracking-wide text-slate-400">Scenario</p>
          <p className="text-sm font-semibold text-slate-100">{scenarioName}</p>
        </div>
        {onUnlock && (
          <button
            type="button"
            onClick={onUnlock}
            className="text-xs text-blue-300 underline hover:text-blue-200"
          >
            Change scenario
          </button>
        )}
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-1.5 gap-3">
        <Label htmlFor="scenarioName">Scenario Name</Label>
        <div className="flex items-center gap-2">
          {selectedScenario?.connection_config && onLoadSaved && (
            <button
              type="button"
              onClick={onLoadSaved}
              className="text-xs text-blue-300 hover:text-blue-200"
            >
              Load saved URLs
            </button>
          )}
          <button
            type="button"
            onClick={onOpenScenarioModal}
            className="text-xs text-blue-400 hover:text-blue-300"
          >
            Browse scenarios
          </button>
        </div>
      </div>

      <div className="mt-1.5 rounded-md border border-slate-800 bg-slate-950/60 p-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <p className="text-sm font-semibold text-slate-100">
              {displayName || (loadingScenarios ? "Loading scenarios..." : "Select a scenario")}
            </p>
            <p className="text-xs text-slate-400">
              {scenarioName ? `Slug: ${scenarioName}` : "Browse scenarios to choose a desktop target."}
            </p>
          </div>
          <Button type="button" variant="outline" size="sm" onClick={onOpenScenarioModal}>
            Browse scenarios
          </Button>
        </div>
      </div>
      <p className="mt-1.5 text-xs text-slate-400">
        Select from available scenarios or enter a slug from the modal.
      </p>
    </div>
  );
}

interface FrameworkTemplateSectionProps {
  framework: string;
  onFrameworkChange: (framework: string) => void;
  onOpenFrameworkModal: () => void;
  selectedTemplate: string;
  onOpenTemplateModal: () => void;
}

function FrameworkTemplateSection({
  framework,
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  onFrameworkChange,
  onOpenFrameworkModal,
  selectedTemplate,
  onOpenTemplateModal
}: FrameworkTemplateSectionProps) {
  const frameworkSummary = FRAMEWORK_SUMMARIES[framework] ?? {
    name: framework,
    description: "Desktop framework"
  };
  const summary =
    TEMPLATE_SUMMARIES[selectedTemplate] ?? {
      name: selectedTemplate.replace(/_/g, " "),
      description: "Custom template"
    };
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      <div>
        <Label>Framework</Label>
        <div className="mt-1.5 rounded-md border border-slate-800 bg-slate-950/60 p-3">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p className="text-sm font-semibold text-slate-100">{frameworkSummary.name}</p>
              <p className="text-xs text-slate-400">{frameworkSummary.description}</p>
            </div>
            <Button type="button" variant="outline" size="sm" onClick={onOpenFrameworkModal}>
              Browse frameworks
            </Button>
          </div>
        </div>
        <p className="mt-1.5 text-xs text-slate-400">
          Electron is fully supported today. Tauri and Neutralino are planned but not yet available.
        </p>
      </div>

      <div>
        <Label>Template</Label>
        <div className="mt-1.5 rounded-md border border-slate-800 bg-slate-950/60 p-3">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p className="text-sm font-semibold text-slate-100">{summary?.name ?? "Select template"}</p>
              <p className="text-xs text-slate-400">
                {summary?.description ?? "Browse templates to choose the best fit."}
              </p>
            </div>
            <Button type="button" variant="outline" size="sm" onClick={onOpenTemplateModal}>
              Browse templates
            </Button>
          </div>
        </div>
        <p className="mt-1.5 text-xs text-slate-400">
          All templates share the same codebase. If you change your mind later, switch templates here or from the
          Generated Apps tab - your scenario stays intact.
        </p>
      </div>
    </div>
  );
}

interface SigningInlineSectionProps {
  scenarioName: string;
  signingEnabled: boolean;
  signingConfig?: SigningConfig | null;
  readiness?: SigningReadinessResponse;
  loading: boolean;
  onToggleSigning: (enabled: boolean) => void;
  onOpenSigning: () => void;
  onRefresh: () => void;
}

function SigningInlineSection({
  scenarioName,
  signingEnabled,
  signingConfig,
  readiness,
  loading,
  onToggleSigning,
  onOpenSigning,
  onRefresh
}: SigningInlineSectionProps) {
  const hasConfig = Boolean(signingConfig);
  const ready = readiness?.ready;
  const readinessNote =
    readiness?.issues && readiness.issues.length > 0 ? readiness.issues[0] : undefined;
  const [expiryWarning, setExpiryWarning] = useState<string | null>(null);

  useEffect(() => {
    if (typeof window === "undefined") return;
    const stored = window.localStorage.getItem("std_signing_expiry_warning");
    if (stored) {
      setExpiryWarning(stored);
    }
  }, []);

  return (
    <div className="rounded-lg border border-slate-800 bg-slate-900/60 p-4 space-y-3">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="space-y-1">
          <p className="text-sm font-semibold text-slate-100">Signing (recommended for production)</p>
          <p className="text-xs text-slate-400">
            Signed installers avoid OS warnings. Configure details in the Signing tab or reuse saved config here.
          </p>
        </div>
        <label className="flex items-center gap-2 text-sm text-slate-100">
          <input
            type="checkbox"
            className="h-4 w-4"
            checked={signingEnabled}
            disabled={!scenarioName}
            onChange={(e) => onToggleSigning(e.target.checked)}
          />
          <span>{signingEnabled ? "Sign this build" : "Skip signing for this build"}</span>
        </label>
      </div>

      <div className="flex flex-wrap items-center gap-3 text-xs text-slate-300">
        {loading ? (
          <div className="flex items-center gap-2">
            <RefreshCw className="h-4 w-4 animate-spin text-blue-300" />
            Checking signing status…
          </div>
        ) : signingEnabled ? (
          hasConfig ? (
            ready ? (
              <div className="flex items-center gap-2 text-green-300">
                <ShieldCheck className="h-4 w-4" />
                Signing ready for at least one platform.
              </div>
            ) : (
              <div className="flex items-center gap-2 text-amber-300">
                <AlertTriangle className="h-4 w-4" />
                {readinessNote || "Signing config needs fixes before packaging."}
              </div>
            )
          ) : (
            <div className="flex items-center gap-2 text-amber-300">
              <AlertTriangle className="h-4 w-4" />
              No signing config saved yet. Open the Signing tab to add certificates.
            </div>
          )
        ) : (
          <div className="flex items-center gap-2 text-slate-300">
            <AlertTriangle className="h-4 w-4 text-slate-400" />
            Unsigned installers may trigger OS warnings. Enable signing when you&apos;re ready.
          </div>
        )}

        {signingEnabled && expiryWarning && (
          <div className="flex items-center gap-2 text-amber-300">
            <AlertTriangle className="h-4 w-4" />
            {expiryWarning}
          </div>
        )}

        <div className="flex items-center gap-2">
          <Button type="button" size="sm" variant="outline" onClick={onRefresh} disabled={!scenarioName || loading}>
            <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
            Refresh
          </Button>
          {!signingEnabled && (
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={() => onToggleSigning(true)}
              disabled={!scenarioName}
              className="gap-1 text-blue-200 hover:text-blue-100"
            >
              Enable signing now
            </Button>
          )}
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={onOpenSigning}
            disabled={!scenarioName}
            className="gap-1 text-blue-200 hover:text-blue-100"
          >
            Open Signing tab
            <ExternalLink className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}

interface OutputPathFieldProps {
  outputPath: string;
  onOutputPathChange: (value: string) => void;
}

interface OutputLocationSelectorProps {
  locationMode: OutputLocation;
  standardPath: string;
  stagingPreview: string;
  onChange: (value: OutputLocation) => void;
}

function OutputLocationSelector({ locationMode, onChange, standardPath, stagingPreview }: OutputLocationSelectorProps) {
  return (
    <div className="space-y-2">
      <Label>Output location</Label>
      <div className="space-y-1.5">
        <label className="flex gap-3 text-sm text-slate-200">
          <input
            type="radio"
            name="outputLocation"
            value="proper"
            checked={locationMode === "proper"}
            onChange={() => onChange("proper")}
          />
          <span>
            Proper (recommended): <code className="text-xs">{standardPath}</code>
          </span>
        </label>
        <label className="flex gap-3 text-sm text-slate-200">
          <input
            type="radio"
            name="outputLocation"
            value="temp"
            checked={locationMode === "temp"}
            onChange={() => onChange("temp")}
          />
          <span>
            Temporary (gitignored staging): <code className="text-xs">{stagingPreview}</code>
          </span>
        </label>
        <label className="flex gap-3 text-sm text-slate-200">
          <input
            type="radio"
            name="outputLocation"
            value="custom"
            checked={locationMode === "custom"}
            onChange={() => onChange("custom")}
          />
          <span>Custom path</span>
        </label>
      </div>
      <p className="text-xs text-slate-400">
        Proper keeps wrappers beside their scenarios. Temporary routes output to a gitignored staging area so you can
        review before moving. Custom is for one-off locations.
      </p>
    </div>
  );
}

function OutputPathField({ outputPath, onOutputPathChange }: OutputPathFieldProps) {
  return (
    <div>
      <Label htmlFor="outputPath">Output Directory</Label>
      <Input
        id="outputPath"
        value={outputPath}
        onChange={(e) => onOutputPathChange(e.target.value)}
        placeholder="/absolute/or/relative/path"
        className="mt-1.5"
      />
      <p className="mt-1 text-xs text-slate-400">
        Used only when choosing a custom location. Leave blank to fall back to the selected mode&apos;s default.
      </p>
    </div>
  );
}

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
    secrets: {} as Record<string, string>,
    sessionId: null as string | null,
    sessionExpiresAt: null as string | null,
    sessionTTL: 120,
    startServices: true,
    autoRefresh: true
  });
  const [deploymentManagerUrl, setDeploymentManagerUrl] = useState<string | null>(null);
  const [lastLoadedScenario, setLastLoadedScenario] = useState<string | null>(null);
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
    jobStatus: preflightJobStatus,
    sessionId: preflightSessionId,
    sessionExpiresAt: preflightSessionExpiresAt,
    sessionTTL: preflightSessionTTL,
    override: preflightOverride,
    secrets: preflightSecrets,
    startServices: preflightStartServices,
    autoRefresh: preflightAutoRefresh,
    missingSecrets: missingPreflightSecrets,
    preflightOk,
    setSessionTTL: setPreflightSessionTTL,
    setOverride: setPreflightOverride,
    setSecret: setPreflightSecret,
    runPreflight,
    stopSession: stopPreflightSession,
    reset: resetPreflight
  } = usePreflightSession({
    bundleManifestPath,
    isBundled,
    initialSessionId: preflightSeed.sessionId,
    initialSessionExpiresAt: preflightSeed.sessionExpiresAt,
    initialResult: preflightSeed.result,
    initialError: preflightSeed.error,
    initialOverride: preflightSeed.override,
    initialSecrets: preflightSeed.secrets,
    initialSessionTTL: preflightSeed.sessionTTL,
    initialStartServices: preflightSeed.startServices,
    initialAutoRefresh: preflightSeed.autoRefresh
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

  const resetFormState = (resetTemplate: boolean) => {
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
    setPreflightSeed({
      result: null,
      error: null,
      override: false,
      secrets: {},
      sessionId: null,
      sessionExpiresAt: null,
      sessionTTL: 120,
      startServices: true,
      autoRefresh: true
    });
    resetPreflight();
    if (resetTemplate) {
      onTemplateChange("basic");
    }
  };

  const applyDraftState = (draft: GeneratorDraftState) => {
    setAppDisplayName(draft.appDisplayName);
    setAppDescription(draft.appDescription);
    setIconPath(draft.iconPath);
    setDisplayNameEdited(draft.displayNameEdited);
    setDescriptionEdited(draft.descriptionEdited);
    setIconPathEdited(draft.iconPathEdited);
    setFramework(draft.framework || "electron");
    setServerType(draft.serverType ?? DEFAULT_SERVER_TYPE);
    setDeploymentMode(draft.deploymentMode ?? DEFAULT_DEPLOYMENT_MODE);
    setPlatforms(draft.platforms ?? { win: true, mac: true, linux: true });
    setLocationMode(draft.locationMode ?? "proper");
    setOutputPath(draft.outputPath ?? "");
    setProxyUrl(draft.proxyUrl ?? "");
    setBundleManifestPath(draft.bundleManifestPath ?? "");
    setServerPort(draft.serverPort ?? 3000);
    setLocalServerPath(draft.localServerPath ?? "ui/server.js");
    setLocalApiEndpoint(draft.localApiEndpoint ?? "http://localhost:3001/api");
    setAutoManageTier1(draft.autoManageTier1 ?? false);
    setVrooliBinaryPath(draft.vrooliBinaryPath ?? "vrooli");
    setConnectionResult(draft.connectionResult ?? null);
    setConnectionError(draft.connectionError ?? null);
    setDeploymentManagerUrl(draft.deploymentManagerUrl ?? null);
    setSigningEnabledForBuild(draft.signingEnabledForBuild ?? false);
    if (draft.selectedTemplate) {
      onTemplateChange(draft.selectedTemplate);
    }
  };

  const draftState = useMemo(() => ({
    selectedTemplate,
    appDisplayName,
    appDescription,
    iconPath,
    displayNameEdited,
    descriptionEdited,
    iconPathEdited,
    framework,
    serverType,
    deploymentMode,
    platforms,
    locationMode,
    outputPath,
    proxyUrl,
    bundleManifestPath,
    serverPort,
    localServerPath,
    localApiEndpoint,
    autoManageTier1,
    vrooliBinaryPath,
    connectionResult,
    connectionError,
    preflightResult,
    preflightError,
    preflightOverride,
    preflightSecrets,
    preflightStartServices,
    preflightAutoRefresh,
    preflightSessionId,
    preflightSessionExpiresAt,
    preflightSessionTTL,
    deploymentManagerUrl,
    signingEnabledForBuild
  }), [
    selectedTemplate,
    appDisplayName,
    appDescription,
    iconPath,
    displayNameEdited,
    descriptionEdited,
    iconPathEdited,
    framework,
    serverType,
    deploymentMode,
    platforms,
    locationMode,
    outputPath,
    proxyUrl,
    bundleManifestPath,
    serverPort,
    localServerPath,
    localApiEndpoint,
    autoManageTier1,
    vrooliBinaryPath,
    connectionResult,
    connectionError,
    preflightResult,
    preflightError,
    preflightOverride,
    preflightSecrets,
    preflightStartServices,
    preflightAutoRefresh,
    preflightSessionId,
    preflightSessionExpiresAt,
    preflightSessionTTL,
    deploymentManagerUrl,
    signingEnabledForBuild
  ]);

  const {
    timestamps: draftTimestamps,
    loadedScenario: draftLoadedScenario,
    clearDraft
  } = useGeneratorDraft({
    scenarioName,
    draftState,
    onDraftLoaded: (draft) => {
      applyDraftState(draft);
      setPreflightSeed({
        result: draft.preflightResult,
        error: draft.preflightError,
        override: draft.preflightOverride,
        secrets: draft.preflightSecrets,
        sessionId: draft.preflightSessionId,
        sessionExpiresAt: draft.preflightSessionExpiresAt,
        sessionTTL: draft.preflightSessionTTL,
        startServices: draft.preflightStartServices,
        autoRefresh: draft.preflightAutoRefresh
      });
    },
    onDraftCleared: () => {
      resetFormState(true);
      setPreflightSeed({
        result: null,
        error: null,
        override: false,
        secrets: {},
        sessionId: null,
        sessionExpiresAt: null,
        sessionTTL: 120,
        startServices: true,
        autoRefresh: true
      });
      resetPreflight();
    }
  });

  // Fetch available scenarios
  const { data: scenariosData, isLoading: loadingScenarios } = useQuery<ScenariosResponse>({
    queryKey: ['scenarios-desktop-status'],
    queryFn: fetchScenarioDesktopStatus,
  });

  const generateMutation = useMutation({
    mutationFn: generateDesktop,
    onSuccess: (data) => {
      onBuildStart(data.build_id);
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

  const selectedScenario = scenariosData?.scenarios.find((s) => s.name === scenarioName);
  const iconPreviewUrl = useMemo(
    () => (iconPath ? getIconPreviewUrl(iconPath) : ""),
    [iconPath]
  );

  useEffect(() => {
    setIconPreviewError(false);
  }, [iconPreviewUrl]);

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
    const updatedAt = selectedScenario?.connection_config?.updated_at;
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
    applySavedConnection(selectedScenario?.connection_config);
    setLastLoadedScenario(configKey);
  }, [
    scenarioName,
    selectedScenario?.connection_config?.updated_at,
    lastLoadedScenario,
    selectedScenario?.connection_config,
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

    if (!scenarioName) {
      alert("Please select a scenario before generating a desktop app.");
      return;
    }

    const outputPathForRequest = locationMode === "custom" ? outputPath : "";

    const validationMessage = validateGeneratorInputs({
      selectedPlatforms: selectedPlatformsList,
      decision: connectionDecision,
      bundleManifestPath,
      proxyUrl,
      appDisplayName,
      appDescription,
      locationMode,
      outputPath: outputPathForRequest
    });

    if (validationMessage) {
      alert(validationMessage);
      return;
    }

    if (isBundled) {
      if (!preflightResult) {
        alert("Run preflight validation before generating a bundled desktop app.");
        return;
      }
      if (!preflightOk && !preflightOverride) {
        alert("Preflight is not green. Fix the issues or enable override to continue.");
        return;
      }
    }

    if (signingEnabledForBuild) {
      if (!signingConfig || !signingConfig.enabled) {
        alert("Signing is enabled for this build but no signing config is saved. Open the Signing tab to add certificates first.");
        return;
      }
      if (signingReadiness && !signingReadiness.ready) {
        const issue = signingReadiness.issues?.[0] || "Resolve signing prerequisites before packaging.";
        alert(`Signing is not ready: ${issue}`);
        return;
      }
    }

    const config = buildDesktopConfig({
      scenarioName,
      selectedTemplate,
      framework,
      serverType: connectionDecision.effectiveServerType,
      serverPort,
      outputPath: outputPathForRequest,
      selectedPlatforms: selectedPlatformsList,
      deploymentMode,
      autoManageTier1,
      vrooliBinaryPath,
      proxyUrl,
      bundleManifestPath,
      isBundled,
      requiresRemoteConfig,
      resolvedEndpoints,
      locationMode,
      appDisplayName,
      appDescription,
      iconPath,
      includeSigning: signingEnabledForBuild,
      codeSigning: signingEnabledForBuild ? (signingConfig ?? undefined) : { enabled: false }
    });

    generateMutation.mutate(config);
  };

  const handlePlatformChange = (platform: string, checked: boolean) => {
    setPlatforms((prev) => ({ ...prev, [platform]: checked }));
  };

  const handleDeploymentChange = (nextMode: DeploymentMode) => {
    setDeploymentMode(nextMode);
    const nextAllowed = nextMode === "bundled" || nextMode === "cloud-api"
      ? ["external"]
      : SERVER_TYPE_OPTIONS.map((option) => option.value);
    if (!nextAllowed.includes(serverType)) {
      setServerType((nextAllowed[0] as ServerType) ?? DEFAULT_SERVER_TYPE);
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
          runPreflight(undefined, manifestPath);
        }}
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
        {(draftCreatedLabel || draftUpdatedLabel) && (
          <div className="flex flex-wrap items-center gap-2 text-xs text-slate-400">
            {draftCreatedLabel && (
              <span>Draft started {draftCreatedLabel}</span>
            )}
            {draftUpdatedLabel && (
              <span>Updated {draftUpdatedLabel}</span>
            )}
          </div>
        )}
        <Button
          type="button"
          size="sm"
          variant="outline"
          className="sm:ml-auto"
          disabled={!scenarioName}
          onClick={() => {
            if (!scenarioName) return;
            if (preflightSessionId && bundleManifestPath.trim()) {
              void stopPreflightSession();
            }
            clearDraft();
            resetFormState(true);
          }}
        >
          Reset progress
        </Button>
      </div>
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
            onFrameworkChange={setFramework}
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
              preflightJobStatus={preflightJobStatus}
              missingSecrets={missingPreflightSecrets}
              secretInputs={preflightSecrets}
              preflightOk={preflightOk}
              preflightOverride={preflightOverride}
              preflightSessionTTL={preflightSessionTTL}
              preflightSessionId={preflightSessionId}
              preflightSessionExpiresAt={preflightSessionExpiresAt}
              preflightLogTails={preflightResult?.log_tails}
              onOverrideChange={setPreflightOverride}
              onSessionTTLChange={setPreflightSessionTTL}
              onSecretChange={setPreflightSecret}
              onRun={runPreflight}
              onStopSession={stopPreflightSession}
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
          {showSubmit && (
            <>
              <Button
                type="submit"
                className="w-full"
                disabled={generateMutation.isPending}
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
