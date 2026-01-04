import { forwardRef, useCallback, useEffect, useImperativeHandle, useMemo, useRef, useState, type FormEvent, type Ref } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import {
  exportBundleFromDeploymentManager,
  startDeploymentManagerAutoBuild,
  fetchDeploymentManagerAutoBuildStatus,
  fetchProxyHints,
  fetchScenarioDesktopStatus,
  fetchBundleManifest,
  getIconPreviewUrl,
  generateDesktop,
  fetchSigningConfig,
  checkSigningReadiness,
  probeEndpoints,
  runBundlePreflight,
  type BundlePreflightLogTail,
  type BundlePreflightResponse,
  type BundlePreflightSecret,
  type BundleManifestResponse,
  type AutoBuildStatusResponse,
  type DesktopConfig,
  type ProbeResponse,
  type ProxyHintsResponse,
  type SigningConfig,
  type SigningReadinessResponse
} from "../lib/api";
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card";
import { Input } from "./ui/input";
import { Label } from "./ui/label";
import { Select } from "./ui/select";
import { Checkbox } from "./ui/checkbox";
import { Button } from "./ui/button";
import { Progress } from "./ui/progress";
import { AlertTriangle, Braces, CheckCircle, Copy, Download, ExternalLink, LayoutList, Loader2, RefreshCw, Rocket, ShieldCheck } from "lucide-react";
import { TemplateModal } from "./TemplateModal";
import { ScenarioModal } from "./ScenarioModal";
import { FrameworkModal } from "./FrameworkModal";
import { DeploymentModal } from "./DeploymentModal";
import type { DesktopConnectionConfig, ScenarioDesktopStatus, ScenariosResponse } from "./scenario-inventory/types";
import {
  DEFAULT_DEPLOYMENT_MODE,
  DEFAULT_SERVER_TYPE,
  SERVER_TYPE_OPTIONS,
  decideConnection,
  findDeploymentOption,
  findServerTypeOption,
  type ConnectionDecision,
  type DeploymentMode,
  type ServerType
} from "../domain/deployment";

type PlatformSelection = {
  win: boolean;
  mac: boolean;
  linux: boolean;
};

type EndpointResolution = {
  serverPath: string;
  apiEndpoint: string;
};

type OutputLocation = "proper" | "temp" | "custom";

interface BuildDesktopConfigOptions {
  scenarioName: string;
  appDisplayName: string;
  appDescription: string;
  iconPath: string;
  selectedTemplate: string;
  framework: string;
  serverType: ServerType;
  serverPort: number;
  outputPath: string;
  selectedPlatforms: string[];
  deploymentMode: DeploymentMode;
  autoManageTier1: boolean;
  vrooliBinaryPath: string;
  proxyUrl: string;
  bundleManifestPath: string;
  isBundled: boolean;
  requiresRemoteConfig: boolean;
  resolvedEndpoints: EndpointResolution;
  locationMode: OutputLocation;
  includeSigning: boolean;
  codeSigning?: SigningConfig;
}

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

function getSelectedPlatforms(platforms: PlatformSelection): string[] {
  return Object.entries(platforms)
    .filter(([, enabled]) => enabled)
    .map(([platform]) => platform);
}

function validateGeneratorInputs(options: {
  selectedPlatforms: string[];
  decision: ConnectionDecision;
  bundleManifestPath: string;
  proxyUrl: string;
  appDisplayName: string;
  appDescription: string;
  locationMode: OutputLocation;
  outputPath: string;
}): string | null {
  if (options.selectedPlatforms.length === 0) {
    return "Please select at least one target platform";
  }

  if (!options.appDisplayName.trim()) {
    return "Provide an app display name so installers and windows are branded correctly.";
  }

  if (!options.appDescription.trim()) {
    return "Provide a short description for the generated desktop app.";
  }

  if (options.decision.requiresBundleManifest && !options.bundleManifestPath) {
    return "Provide bundle_manifest_path from deployment-manager before generating a bundled build.";
  }

  if (options.decision.requiresProxyUrl && !options.proxyUrl) {
    return "Provide the proxy URL you use in the browser (for example https://app-monitor.example.com/apps/<scenario>/proxy/).";
  }

  if (options.locationMode === "custom" && !options.outputPath.trim()) {
    return "Provide an output path when choosing a custom location.";
  }

  return null;
}

function resolveEndpoints(input: {
  decision: ConnectionDecision;
  proxyUrl: string;
  localServerPath: string;
  localApiEndpoint: string;
}): EndpointResolution {
  if (input.decision.kind === "bundled-runtime") {
    return { serverPath: "http://127.0.0.1", apiEndpoint: "http://127.0.0.1" };
  }
  if (input.decision.requiresProxyUrl) {
    return { serverPath: input.proxyUrl, apiEndpoint: input.proxyUrl };
  }
  return { serverPath: input.localServerPath, apiEndpoint: input.localApiEndpoint };
}

function buildDesktopConfig(options: BuildDesktopConfigOptions): DesktopConfig {
  return {
    app_name: options.scenarioName,
    app_display_name: options.appDisplayName,
    app_description: options.appDescription,
    version: "1.0.0",
    author: "Vrooli Platform",
    license: "MIT",
    app_id: `com.vrooli.${options.scenarioName.replace(/-/g, ".")}`,
    icon: options.iconPath || undefined,
    server_type: options.serverType,
    server_port: options.serverPort,
    server_path: options.resolvedEndpoints.serverPath,
    api_endpoint: options.resolvedEndpoints.apiEndpoint,
    framework: options.framework,
    template_type: options.selectedTemplate,
    platforms: options.selectedPlatforms,
    output_path: options.outputPath,
    location_mode: options.locationMode,
    features: {
      splash: true,
      autoUpdater: true,
      devTools: true
    },
    window: {
      width: 1200,
      height: 800,
      background: "#f5f5f5"
    },
    deployment_mode: options.deploymentMode,
    auto_manage_vrooli: options.autoManageTier1,
    vrooli_binary_path: options.vrooliBinaryPath,
    proxy_url: options.requiresRemoteConfig ? options.proxyUrl : undefined,
    external_server_url: options.requiresRemoteConfig ? options.proxyUrl : undefined,
    external_api_url: !options.requiresRemoteConfig && !options.isBundled ? options.resolvedEndpoints.apiEndpoint : undefined,
    bundle_manifest_path: options.isBundled ? options.bundleManifestPath : undefined,
    code_signing: options.includeSigning ? options.codeSigning : { enabled: false }
  };
}

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

interface DeploymentSummarySectionProps {
  deploymentMode: DeploymentMode;
  serverType: ServerType;
  onOpenDeploymentModal: () => void;
}

function DeploymentSummarySection({
  deploymentMode,
  serverType,
  onOpenDeploymentModal
}: DeploymentSummarySectionProps) {
  const deployment = findDeploymentOption(deploymentMode);
  const server = findServerTypeOption(serverType);

  return (
    <div className="rounded-lg border border-slate-800 bg-slate-950/60 p-4">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="space-y-2">
          <div>
            <p className="text-xs uppercase tracking-wide text-slate-400">Deployment intent</p>
            <p className="text-sm font-semibold text-slate-100">{deployment.label}</p>
            <p className="text-xs text-slate-400">{deployment.description}</p>
          </div>
          <div>
            <p className="text-xs uppercase tracking-wide text-slate-400">Data source</p>
            <p className="text-sm font-semibold text-slate-100">{server.label}</p>
            <p className="text-xs text-slate-400">{server.description}</p>
          </div>
        </div>
        <Button type="button" variant="outline" size="sm" onClick={onOpenDeploymentModal}>
          Choose deployment
        </Button>
      </div>
    </div>
  );
}

interface BundledRuntimeSectionProps {
  bundleManifestPath: string;
  onBundleManifestChange: (value: string) => void;
  scenarioName: string;
  bundleHelperRef: Ref<DeploymentManagerBundleHelperHandle>;
  onBundleExported?: (manifestPath: string) => void;
  onDeploymentManagerUrlChange?: (url: string | null) => void;
}

function BundledRuntimeSection({
  bundleManifestPath,
  onBundleManifestChange,
  scenarioName,
  bundleHelperRef,
  onBundleExported,
  onDeploymentManagerUrlChange
}: BundledRuntimeSectionProps) {
  const [viewMode, setViewMode] = useState<"summary" | "json">("summary");
  const [manifestResult, setManifestResult] = useState<BundleManifestResponse | null>(null);
  const [manifestError, setManifestError] = useState<string | null>(null);
  const [manifestLoading, setManifestLoading] = useState(false);
  const [manifestCopyStatus, setManifestCopyStatus] = useState<"idle" | "copied" | "error">("idle");

  const manifestName = bundleManifestPath.trim()
    ? bundleManifestPath.trim().split(/[\\/]/).pop() || "bundle.json"
    : "bundle.json";

  useEffect(() => {
    setManifestResult(null);
    setManifestError(null);
    setManifestLoading(false);
  }, [bundleManifestPath]);

  useEffect(() => {
    if (viewMode !== "json") {
      return;
    }
    if (!bundleManifestPath.trim()) {
      setManifestResult(null);
      setManifestError("Add a bundle manifest path to view the JSON.");
      return;
    }
    let isActive = true;
    setManifestLoading(true);
    setManifestError(null);
    fetchBundleManifest({ bundle_manifest_path: bundleManifestPath })
      .then((result) => {
        if (!isActive) return;
        setManifestResult(result);
        setManifestError(null);
      })
      .catch((error) => {
        if (!isActive) return;
        setManifestResult(null);
        setManifestError((error as Error).message);
      })
      .finally(() => {
        if (!isActive) return;
        setManifestLoading(false);
      });
    return () => {
      isActive = false;
    };
  }, [bundleManifestPath, viewMode]);

  const manifestJson = manifestResult?.manifest ? JSON.stringify(manifestResult.manifest, null, 2) : "";

  const copyManifestJson = async () => {
    if (!manifestJson) {
      return;
    }
    try {
      await navigator.clipboard.writeText(manifestJson);
      setManifestCopyStatus("copied");
      setTimeout(() => setManifestCopyStatus("idle"), 1500);
    } catch (error) {
      console.warn("Failed to copy manifest JSON", error);
      setManifestCopyStatus("error");
      setTimeout(() => setManifestCopyStatus("idle"), 2000);
    }
  };

  const downloadManifestJson = () => {
    if (!manifestJson) {
      return;
    }
    const blob = new Blob([manifestJson], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = manifestName;
    link.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="rounded-lg border border-slate-700 bg-slate-900/40 p-4 space-y-3">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <Label htmlFor="bundleManifest">Bundle manifest</Label>
          <p className="text-xs text-slate-400 mt-1">
            Generate a bundle manifest (bundle.json) and staged files for offline use.
          </p>
        </div>
        <div className="flex items-center gap-1 rounded-full border border-slate-800/70 bg-slate-950/40 p-1 text-[11px]">
          <Button
            type="button"
            size="sm"
            variant={viewMode === "summary" ? "default" : "ghost"}
            onClick={() => setViewMode("summary")}
            aria-label="Show bundle manifest summary"
            className="h-10 w-10 p-0"
          >
            <LayoutList className="h-4 w-4" />
          </Button>
          <Button
            type="button"
            size="sm"
            variant={viewMode === "json" ? "default" : "ghost"}
            onClick={() => setViewMode("json")}
            aria-label="Show bundle manifest JSON"
            className="h-10 w-10 p-0"
          >
            <Braces className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {viewMode === "summary" && (
        <>
          <Input
            id="bundleManifest"
            value={bundleManifestPath}
            onChange={(e) => onBundleManifestChange(e.target.value)}
            placeholder="/home/you/Vrooli/scenarios/my-scenario/platforms/electron/bundle/bundle.json"
          />
          <div className="rounded-md border border-slate-800 bg-slate-950/60 p-3 text-xs text-slate-200 space-y-2">
            <p className="font-semibold text-slate-100">Current manifest</p>
            {bundleManifestPath.trim() ? (
              <>
                <p className="text-[11px] text-slate-300">{bundleManifestPath}</p>
                <p className="text-[11px] text-slate-400">
                  Expect this file to live alongside staged binaries/assets.
                </p>
              </>
            ) : (
              <p className="text-[11px] text-slate-400">
                Paste a bundle.json path or export one from deployment-manager below.
              </p>
            )}
          </div>
        </>
      )}

      {viewMode === "json" && (
        <div className="rounded-md border border-slate-800 bg-slate-950/60 p-3 space-y-2">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <p className="text-xs font-semibold text-slate-200">Bundle manifest JSON</p>
            <div className="flex items-center gap-2">
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={copyManifestJson}
                disabled={!manifestJson}
                aria-label="Copy bundle manifest JSON"
                className="h-10 w-10 p-0"
              >
                <Copy className="h-4 w-4" />
              </Button>
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={downloadManifestJson}
                disabled={!manifestJson}
                aria-label="Download bundle manifest JSON"
                className="h-10 w-10 p-0"
              >
                <Download className="h-4 w-4" />
              </Button>
            </div>
          </div>
          {manifestLoading && (
            <p className="text-[11px] text-slate-400">Loading bundle manifest...</p>
          )}
          {manifestError && (
            <p className="text-[11px] text-amber-200">{manifestError}</p>
          )}
          {!manifestLoading && !manifestError && (
            <pre className="max-h-96 overflow-auto whitespace-pre-wrap rounded-md border border-slate-800/70 bg-slate-950/80 p-3 text-[11px] text-slate-200">
              {manifestJson || "No manifest loaded."}
            </pre>
          )}
          {manifestCopyStatus !== "idle" && (
            <p className="text-[11px] text-slate-400">
              {manifestCopyStatus === "copied" ? "Copied to clipboard." : "Copy failed."}
            </p>
          )}
        </div>
      )}

      <DeploymentManagerBundleHelper
        ref={bundleHelperRef}
        scenarioName={scenarioName}
        onBundleManifestChange={onBundleManifestChange}
        onBundleExported={onBundleExported}
        onDeploymentManagerUrlChange={onDeploymentManagerUrlChange}
      />
    </div>
  );
}

type PreflightStepState = "pending" | "testing" | "pass" | "fail" | "warning" | "skipped";

type PreflightStepStatus = {
  state: PreflightStepState;
  label: string;
};

const PREFLIGHT_STEP_STYLES: Record<PreflightStepState, string> = {
  pending: "border-slate-800/70 bg-slate-950/70 text-slate-200",
  testing: "border-blue-800/70 bg-blue-950/60 text-blue-200",
  pass: "border-emerald-800/70 bg-emerald-950/40 text-emerald-200",
  fail: "border-red-800/70 bg-red-950/60 text-red-200",
  warning: "border-amber-800/70 bg-amber-950/60 text-amber-200",
  skipped: "border-slate-800/70 bg-slate-950/60 text-slate-300"
};

interface PreflightStepHeaderProps {
  index: number;
  title: string;
  status: PreflightStepStatus;
  subtitle?: string;
}

function PreflightStepHeader({ index, title, status, subtitle }: PreflightStepHeaderProps) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <div className="flex items-start gap-3">
        <div className="flex h-7 w-7 items-center justify-center rounded-full border border-slate-800 bg-slate-950 text-xs font-semibold text-slate-200">
          {index}
        </div>
        <div>
          <p className="text-sm font-semibold text-slate-100">{title}</p>
          {subtitle && <p className="text-[11px] text-slate-400">{subtitle}</p>}
        </div>
      </div>
      <span className={`rounded-full border px-2 py-0.5 text-[11px] uppercase tracking-wide ${PREFLIGHT_STEP_STYLES[status.state]}`}>
        {status.label}
      </span>
    </div>
  );
}

type PreflightIssueGuidance = {
  title: string;
  meaning: string;
  remediation: string[];
};

const PREFLIGHT_ISSUE_GUIDANCE: Record<string, PreflightIssueGuidance> = {
  manifest_invalid: {
    title: "Bundle manifest is invalid",
    meaning: "The manifest structure does not pass schema or platform validation.",
    remediation: [
      "Re-export the bundle from deployment-manager for the current OS/arch.",
      "Confirm the manifest points to assets that exist inside the bundle root."
    ]
  },
  binary_not_defined: {
    title: "Binary missing in manifest",
    meaning: "The service has no binary entry for this platform in bundle.json.",
    remediation: [
      "Ensure the service is built for this OS/arch, then re-export the bundle.",
      "Check deployment-manager export settings include service binaries."
    ]
  },
  binary_missing: {
    title: "Binary file not found",
    meaning: "The manifest references a binary path that doesn't exist in the bundle root.",
    remediation: [
      "Rebuild the service binary and re-export the bundle.",
      "Verify the binary path in bundle.json matches the staged file path."
    ]
  },
  binary_is_directory: {
    title: "Binary path is a directory",
    meaning: "The manifest points at a directory, but an executable file is required.",
    remediation: [
      "Update the manifest to point to the executable file.",
      "Re-export the bundle after rebuilding the service."
    ]
  },
  asset_missing: {
    title: "Asset not found",
    meaning: "The manifest references an asset that doesn't exist in the bundle root.",
    remediation: [
      "Rebuild UI/assets (e.g., UI dist output) and re-export the bundle.",
      "Confirm the manifest path matches the staged asset location."
    ]
  },
  asset_is_directory: {
    title: "Asset path is a directory",
    meaning: "The manifest expects a file, but a directory exists at that path.",
    remediation: [
      "Update the manifest to point to a file (e.g., index.html).",
      "Re-export the bundle after rebuilding assets."
    ]
  },
  asset_unreadable: {
    title: "Asset unreadable",
    meaning: "The runtime cannot read the asset on disk.",
    remediation: [
      "Check permissions/ownership of the asset file.",
      "Re-export the bundle to refresh asset staging."
    ]
  },
  checksum_mismatch: {
    title: "Checksum mismatch",
    meaning: "The asset contents do not match the SHA256 checksum recorded in the manifest.",
    remediation: [
      "Re-export the bundle so checksums match rebuilt assets.",
      "Avoid editing assets after bundle export."
    ]
  },
  asset_size_exceeded: {
    title: "Asset exceeds size budget",
    meaning: "The asset is larger than the manifest's expected size + slack.",
    remediation: [
      "Re-export the bundle to update size metadata.",
      "Confirm the build output hasn't unintentionally grown."
    ]
  },
  asset_size_suspicious: {
    title: "Asset size is suspiciously small",
    meaning: "The asset is much smaller than expected, which often indicates a failed build.",
    remediation: [
      "Rebuild the asset and re-export the bundle.",
      "Verify the build output is complete (not a stub file)."
    ]
  },
  asset_size_warning: {
    title: "Asset size warning",
    meaning: "The asset is larger than expected but within the slack budget.",
    remediation: [
      "Re-export the bundle to update size metadata if this is expected.",
      "Review build artifacts for unexpected growth."
    ]
  }
};

interface BundledPreflightSectionProps {
  bundleManifestPath: string;
  preflightResult: BundlePreflightResponse | null;
  preflightPending: boolean;
  preflightError: string | null;
  missingSecrets: BundlePreflightSecret[];
  secretInputs: Record<string, string>;
  preflightOk: boolean;
  preflightOverride: boolean;
  preflightStartServices: boolean;
  preflightLogTails?: BundlePreflightLogTail[];
  deploymentManagerUrl?: string | null;
  onReexportBundle?: () => void;
  onOverrideChange: (value: boolean) => void;
  onStartServicesChange: (value: boolean) => void;
  onSecretChange: (id: string, value: string) => void;
  onRun: (secretsOverride?: Record<string, string>) => void;
}

function BundledPreflightSection({
  bundleManifestPath,
  preflightResult,
  preflightPending,
  preflightError,
  missingSecrets,
  secretInputs,
  preflightOk,
  preflightOverride,
  preflightStartServices,
  preflightLogTails,
  deploymentManagerUrl,
  onReexportBundle,
  onOverrideChange,
  onStartServicesChange,
  onSecretChange,
  onRun
}: BundledPreflightSectionProps) {
  const [viewMode, setViewMode] = useState<"summary" | "json">("summary");
  const [copyStatus, setCopyStatus] = useState<"idle" | "copied" | "error">("idle");
  const validation = preflightResult?.validation;
  const readiness = preflightResult?.ready;
  const ports = preflightResult?.ports;
  const telemetry = preflightResult?.telemetry;
  const logTails = preflightLogTails ?? preflightResult?.log_tails;
  const bundleRootPreview = bundleManifestPath.trim()
    ? bundleManifestPath.trim().replace(/[/\\][^/\\]+$/, "")
    : "";
  const readinessDetails = readiness?.details ? Object.entries(readiness.details) : [];
  const hasMissingArtifacts = Boolean(
    validation
      && !validation.valid
      && ((validation.missing_assets?.length ?? 0) > 0 || (validation.missing_binaries?.length ?? 0) > 0)
  );
  const likelyRootMismatch = Boolean(
    hasMissingArtifacts
      && bundleManifestPath.trim()
      && !bundleManifestPath.includes("scenario-to-desktop/data/staging")
  );
  const preflightPayload = useMemo(
    () => ({
      bundle_manifest_path: bundleManifestPath,
      start_services: preflightStartServices,
      result: preflightResult,
      error: preflightError || undefined,
      missing_secrets: missingSecrets
    }),
    [bundleManifestPath, preflightStartServices, preflightResult, preflightError, missingSecrets]
  );
  const hasRun = Boolean(preflightResult || preflightError);
  const portSummary = ports
    ? Object.entries(ports)
        .map(([svc, portMap]) => {
          const pairs = Object.entries(portMap)
            .map(([name, port]) => `${name}:${port}`)
            .join(", ");
          return `${svc}(${pairs})`;
        })
        .join(" · ")
    : "";
  const diagnosticsAvailable = Boolean(
    portSummary || telemetry?.path || (logTails && logTails.length > 0)
  );

  const stepValidationStatus: PreflightStepStatus = (() => {
    if (preflightPending) {
      return { state: "testing", label: "Testing" };
    }
    if (preflightError) {
      return { state: "fail", label: "Failed" };
    }
    if (!hasRun) {
      return { state: "pending", label: "Pending" };
    }
    if (validation?.valid === true) {
      return { state: "pass", label: "Pass" };
    }
    if (validation?.valid === false) {
      return { state: "fail", label: "Fail" };
    }
    if (preflightError) {
      return { state: "fail", label: "Fail" };
    }
    return { state: "warning", label: "Review" };
  })();

  const stepSecretsStatus: PreflightStepStatus = (() => {
    if (preflightPending) {
      return { state: "testing", label: "Checking" };
    }
    if (preflightError) {
      return { state: "fail", label: "Failed" };
    }
    if (!hasRun) {
      return { state: "pending", label: "Pending" };
    }
    if (missingSecrets.length > 0) {
      return { state: "warning", label: "Missing" };
    }
    return { state: "pass", label: "Ready" };
  })();

  const stepRuntimeStatus: PreflightStepStatus = (() => {
    if (preflightPending) {
      return { state: "testing", label: "Starting" };
    }
    if (preflightError) {
      return { state: "fail", label: "Failed" };
    }
    if (preflightResult) {
      return { state: "pass", label: "Running" };
    }
    if (!hasRun) {
      return { state: "pending", label: "Pending" };
    }
    return { state: "warning", label: "Unknown" };
  })();

  const stepServicesStatus: PreflightStepStatus = (() => {
    if (!preflightStartServices) {
      return { state: "skipped", label: "Skipped" };
    }
    if (preflightPending) {
      return { state: "testing", label: "Starting" };
    }
    if (preflightError) {
      return { state: "fail", label: "Failed" };
    }
    if (!hasRun) {
      return { state: "pending", label: "Pending" };
    }
    if (readiness?.ready === true) {
      return { state: "pass", label: "Ready" };
    }
    if (readiness?.ready === false) {
      return { state: "warning", label: "Waiting" };
    }
    if (preflightError) {
      return { state: "fail", label: "Failed" };
    }
    return { state: "warning", label: "Unknown" };
  })();

  const stepDiagnosticsStatus: PreflightStepStatus = (() => {
    if (preflightPending) {
      return { state: "testing", label: "Collecting" };
    }
    if (preflightError) {
      return { state: "fail", label: "Failed" };
    }
    if (!hasRun) {
      return { state: "pending", label: "Pending" };
    }
    if (diagnosticsAvailable) {
      return { state: "pass", label: "Available" };
    }
    return { state: "warning", label: "Empty" };
  })();

  const copyJson = async () => {
    if (!preflightResult && !preflightError) {
      return;
    }
    try {
      await navigator.clipboard.writeText(JSON.stringify(preflightPayload, null, 2));
      setCopyStatus("copied");
      setTimeout(() => setCopyStatus("idle"), 1500);
    } catch (error) {
      console.warn("Failed to copy preflight JSON", error);
      setCopyStatus("error");
      setTimeout(() => setCopyStatus("idle"), 2000);
    }
  };

  return (
    <div className="rounded-lg border border-slate-800 bg-slate-950/40 p-4 space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="text-sm font-semibold text-slate-100">Preflight validation</p>
          <p className="text-xs text-slate-400">
            Validates the bundle manifest + staged assets by running the bundled runtime (no desktop wrapper needed).
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <div className="flex items-center gap-1 rounded-full border border-slate-800/70 bg-slate-950/60 p-1 text-[11px]">
            <Button
              type="button"
              size="sm"
              variant={viewMode === "summary" ? "default" : "ghost"}
              onClick={() => setViewMode("summary")}
              aria-label="Show preflight summary"
              className="h-10 w-10 p-0"
            >
              <LayoutList className="h-4 w-4" />
            </Button>
            <Button
              type="button"
              size="sm"
              variant={viewMode === "json" ? "default" : "ghost"}
              onClick={() => setViewMode("json")}
              aria-label="Show preflight JSON"
              className="h-10 w-10 p-0"
            >
              <Braces className="h-4 w-4" />
            </Button>
          </div>
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => onRun()}
            disabled={preflightPending || !bundleManifestPath.trim()}
            className="gap-2"
          >
            {preflightPending ? "Running..." : "Run preflight"}
          </Button>
        </div>
      </div>
      <div className="flex items-center gap-2 text-xs text-slate-400">
        <Checkbox
          checked={preflightStartServices}
          onChange={(e) => onStartServicesChange(e.target.checked)}
          label="Start services to capture log tails + readiness (slower)"
        />
      </div>

      {!bundleManifestPath.trim() && (
        <p className="text-xs text-amber-200">Add a bundle_manifest_path to enable preflight checks.</p>
      )}

      {preflightError && (
        <div className="rounded-md border border-red-800 bg-red-950/40 p-3 text-sm text-red-200">
          {preflightError}
        </div>
      )}

      {(preflightResult || preflightError) && viewMode === "summary" && (
        <div className="space-y-4">
          <div className="rounded-md border border-slate-800 bg-slate-950/40 p-3 text-xs text-slate-200 space-y-2">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <p className="font-semibold text-slate-100">Bundle context</p>
              <div className="flex flex-wrap items-center gap-2">
                {onReexportBundle && (
                  <Button type="button" size="xs" variant="outline" onClick={onReexportBundle}>
                    Re-export bundle.json
                  </Button>
                )}
                {deploymentManagerUrl && (
                  <Button type="button" size="xs" variant="outline" asChild>
                    <a href={deploymentManagerUrl} target="_blank" rel="noreferrer">
                      Open deployment-manager
                    </a>
                  </Button>
                )}
              </div>
            </div>
            <div className="space-y-1 text-[11px] text-slate-300">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="text-slate-400">Manifest</span>
                <span className="text-slate-200">{bundleManifestPath.trim() || "Not set"}</span>
              </div>
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="text-slate-400">Bundle root</span>
                <span className="text-slate-200">{bundleRootPreview || "Unknown"}</span>
              </div>
            </div>
            {likelyRootMismatch && (
              <p className="text-[11px] text-amber-200">
                Missing artifacts detected. If your bundle assets live elsewhere, re-export the bundle so the manifest
                and staged files sit in the same directory.
              </p>
            )}
          </div>

          <div className="space-y-3">
            <div className="rounded-md border border-slate-800 bg-slate-950/40 p-3 space-y-3">
              <PreflightStepHeader
                index={1}
                title="Load bundle + validate"
                subtitle="Manifest structure and staged binaries/assets"
                status={stepValidationStatus}
              />
              <p className="text-[11px] text-slate-400">
                Confirms the manifest is valid and staged files exist with matching checksums.
              </p>
              {validation?.valid && !preflightError && (
                <p className="text-[11px] text-slate-300">No validation issues detected.</p>
              )}
              {preflightError && !validation && (
                <p className="text-[11px] text-red-200">
                  Validation did not complete. Review the error above and re-run preflight.
                </p>
              )}
              {validation && !validation.valid && (
                <details className="rounded-md border border-red-900/50 bg-red-950/20 p-3 text-[11px] text-red-200" open>
                  <summary className="cursor-pointer text-xs font-semibold text-red-100">Validation issues</summary>
                  <div className="mt-2 space-y-3">
                    {validation.errors && validation.errors.length > 0 && (
                      <div className="space-y-2">
                        {validation.errors.map((err, idx) => {
                          const guidance = PREFLIGHT_ISSUE_GUIDANCE[err.code];
                          return (
                            <div key={`${err.code}-${idx}`} className="rounded-md border border-red-900/50 bg-red-950/40 p-2 space-y-1">
                              <p className="font-semibold text-red-100">
                                {guidance?.title || err.code}
                              </p>
                              <p>{err.message}</p>
                              {(err.service || err.path) && (
                                <p className="text-[11px] text-red-200/80">
                                  {err.service ? `Service: ${err.service}` : ""}{err.service && err.path ? " · " : ""}{err.path ? `Path: ${err.path}` : ""}
                                </p>
                              )}
                              {guidance && (
                                <>
                                  <p className="text-[11px] text-red-200/80">{guidance.meaning}</p>
                                  <ul className="space-y-1 text-[11px] text-red-100">
                                    {guidance.remediation.map((step, stepIdx) => (
                                      <li key={`${err.code}-remedy-${stepIdx}`}>• {step}</li>
                                    ))}
                                  </ul>
                                </>
                              )}
                            </div>
                          );
                        })}
                      </div>
                    )}
                    {validation.missing_binaries && validation.missing_binaries.length > 0 && (
                      <div className="space-y-1 text-[11px] text-red-100">
                        <p className="font-semibold text-red-100">Missing binaries</p>
                        <ul className="space-y-1">
                          {validation.missing_binaries.map((item, idx) => (
                            <li key={`${item.service_id}-${item.path}-${idx}`}>
                              {item.service_id}: {item.path} ({item.platform})
                            </li>
                          ))}
                        </ul>
                        <p className="text-red-200/80">
                          Rebuild the service binaries and re-export the bundle to update manifest paths.
                        </p>
                      </div>
                    )}
                    {validation.missing_assets && validation.missing_assets.length > 0 && (
                      <div className="space-y-1 text-[11px] text-red-100">
                        <p className="font-semibold text-red-100">Missing assets</p>
                        <ul className="space-y-1">
                          {validation.missing_assets.map((item, idx) => (
                            <li key={`${item.service_id}-${item.path}-${idx}`}>
                              {item.service_id}: {item.path}
                            </li>
                          ))}
                        </ul>
                        <p className="text-red-200/80">
                          Rebuild UI/assets and re-export the bundle so the staged paths exist.
                        </p>
                      </div>
                    )}
                    {validation.invalid_checksums && validation.invalid_checksums.length > 0 && (
                      <div className="space-y-1 text-[11px] text-red-100">
                        <p className="font-semibold text-red-100">Invalid checksums</p>
                        <ul className="space-y-1">
                          {validation.invalid_checksums.map((item, idx) => (
                            <li key={`${item.service_id}-${item.path}-${idx}`}>
                              {item.service_id}: {item.path}
                            </li>
                          ))}
                        </ul>
                        <p className="text-red-200/80">
                          Re-export the bundle after rebuilding assets so checksums match.
                        </p>
                      </div>
                    )}
                  </div>
                </details>
              )}
              {validation?.warnings && validation.warnings.length > 0 && (
                <details className="rounded-md border border-amber-800/50 bg-amber-950/20 p-3 text-[11px] text-amber-200">
                  <summary className="cursor-pointer text-xs font-semibold">Warnings</summary>
                  <ul className="mt-2 space-y-1">
                    {validation.warnings.map((warn, idx) => (
                      <li key={`${warn.code}-${idx}`}>{warn.message}</li>
                    ))}
                  </ul>
                </details>
              )}
            </div>

            <div className="rounded-md border border-slate-800 bg-slate-950/40 p-3 space-y-3">
              <PreflightStepHeader
                index={2}
                title="Apply secrets"
                subtitle="Required secrets must be present for readiness"
                status={stepSecretsStatus}
              />
              {!hasRun && (
                <p className="text-[11px] text-slate-400">
                  Run preflight to detect required secrets.
                </p>
              )}
              {preflightError && (
                <p className="text-[11px] text-red-200">
                  Secrets were not checked because preflight failed.
                </p>
              )}
              {hasRun && !preflightError && missingSecrets.length === 0 && (
                <p className="text-[11px] text-slate-300">All required secrets are present for this run.</p>
              )}
              {missingSecrets.length > 0 && (
                <div className="rounded-md border border-amber-800/60 bg-amber-950/20 p-3 text-xs text-amber-100 space-y-3">
                  <p className="font-semibold text-amber-100">Missing required secrets</p>
                  <p className="text-amber-100/80">
                    Enter temporary values to validate readiness. Values are used only for this preflight run.
                  </p>
                  <div className="space-y-2">
                    {missingSecrets.map((secret) => (
                      <div key={secret.id} className="space-y-1">
                        <Label htmlFor={`preflight-${secret.id}`}>
                          {secret.prompt?.label || secret.id}
                        </Label>
                        <Input
                          id={`preflight-${secret.id}`}
                          type="password"
                          value={secretInputs[secret.id] || ""}
                          onChange={(e) => onSecretChange(secret.id, e.target.value)}
                          placeholder={secret.prompt?.hint || "Enter value"}
                        />
                        {secret.class && (
                          <p className="text-[11px] text-amber-200/80">
                            {secret.class.replace(/_/g, " ")}
                          </p>
                        )}
                      </div>
                    ))}
                  </div>
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    onClick={() => onRun(secretInputs)}
                    disabled={preflightPending}
                  >
                    Apply secrets and re-run
                  </Button>
                </div>
              )}
            </div>

            <div className="rounded-md border border-slate-800 bg-slate-950/40 p-3 space-y-3">
              <PreflightStepHeader
                index={3}
                title="Start runtime control API"
                subtitle="IPC auth token + control API readiness"
                status={stepRuntimeStatus}
              />
              {preflightPending && (
                <p className="text-[11px] text-slate-400">
                  Starting the runtime supervisor and waiting for the control API.
                </p>
              )}
              {!preflightPending && preflightResult && (
                <p className="text-[11px] text-slate-300">
                  Control API is responding. Runtime supervisor initialized.
                </p>
              )}
              {!preflightPending && !preflightResult && !preflightError && (
                <p className="text-[11px] text-slate-400">
                  Run preflight to boot the runtime supervisor.
                </p>
              )}
              {preflightError && (
                <p className="text-[11px] text-red-200">
                  Control API failed to start. Review the error above.
                </p>
              )}
            </div>

            <div className="rounded-md border border-slate-800 bg-slate-950/40 p-3 space-y-3">
              <PreflightStepHeader
                index={4}
                title="Services ready"
                subtitle="Optional service startup + readiness checks"
                status={stepServicesStatus}
              />
              {!preflightStartServices && (
                <p className="text-[11px] text-slate-400">
                  Skipped in dry run. Enable "Start services" to test readiness and capture log tails.
                </p>
              )}
              {preflightStartServices && readiness && readinessDetails.length > 0 && (
                <details className="rounded-md border border-slate-800/70 bg-slate-950/70 p-3 text-xs text-slate-200" open={!readiness.ready}>
                  <summary className="cursor-pointer text-xs font-semibold text-slate-100">
                    Readiness details
                  </summary>
                  <div className="mt-2 grid gap-2 sm:grid-cols-2">
                    {readinessDetails.map(([serviceId, status]) => (
                      <div key={serviceId} className="rounded-md border border-slate-800/70 bg-slate-950/80 p-2 space-y-1">
                        <p className="text-[11px] uppercase tracking-wide text-slate-400">{serviceId}</p>
                        <p className={`text-xs ${status.ready ? "text-emerald-200" : "text-amber-200"}`}>
                          {status.ready ? "Ready" : "Waiting"}
                        </p>
                        {status.message && (
                          <p className="text-[11px] text-slate-300">{status.message}</p>
                        )}
                        {typeof status.exit_code === "number" && (
                          <p className="text-[11px] text-slate-400">Exit code: {status.exit_code}</p>
                        )}
                      </div>
                    ))}
                  </div>
                  {!readiness.ready && (
                    <p className="mt-2 text-[11px] text-slate-300">
                      If readiness stays waiting, review log tails and confirm services can start with supplied secrets.
                    </p>
                  )}
                </details>
              )}
              {preflightStartServices && (!readiness || readinessDetails.length === 0) && (
                <p className="text-[11px] text-slate-400">
                  Run preflight to fetch service readiness details.
                </p>
              )}
            </div>

            <div className="rounded-md border border-slate-800 bg-slate-950/40 p-3 space-y-3">
              <PreflightStepHeader
                index={5}
                title="Diagnostics"
                subtitle="Ports, telemetry, and optional log tails"
                status={stepDiagnosticsStatus}
              />
              {!hasRun && (
                <p className="text-[11px] text-slate-400">
                  Run preflight to collect diagnostics.
                </p>
              )}
              {hasRun && !diagnosticsAvailable && (
                <p className="text-[11px] text-slate-400">
                  No diagnostics reported yet.
                </p>
              )}
              {(portSummary || telemetry?.path) && (
                <div className="rounded-md border border-slate-800/70 bg-slate-950/70 p-3 text-xs text-slate-200 space-y-2">
                  {portSummary && <p>Ports: {portSummary}</p>}
                  {telemetry?.path && <p>Telemetry: {telemetry.path}</p>}
                </div>
              )}
              {logTails && logTails.length > 0 && (
                <details className="rounded-md border border-slate-800/70 bg-slate-950/70 p-3 text-xs text-slate-200">
                  <summary className="cursor-pointer text-xs font-semibold text-slate-100">
                    Service log tails
                  </summary>
                  <div className="mt-2 space-y-2">
                    {logTails.map((tail, idx) => (
                      <div key={`${tail.service_id}-${idx}`} className="rounded-md border border-slate-800/70 bg-slate-950/80 p-2 space-y-1">
                        <p className="text-[11px] uppercase tracking-wide text-slate-400">
                          {tail.service_id} · {tail.lines} lines
                        </p>
                        {tail.error ? (
                          <p className="text-amber-200">{tail.error}</p>
                        ) : (
                          <pre className="max-h-40 overflow-auto whitespace-pre-wrap text-[11px] text-slate-200">
                            {tail.content || "No log output yet."}
                          </pre>
                        )}
                      </div>
                    ))}
                  </div>
                </details>
              )}
            </div>
          </div>

          <details className="rounded-md border border-slate-800 bg-slate-950/40 p-3 text-xs text-slate-200">
            <summary className="cursor-pointer text-xs font-semibold text-slate-100">Coverage map</summary>
            <p className="mt-2 text-[11px] text-slate-400">
              Dry run starts the runtime supervisor, validates bundle files, and checks secrets without starting services.
              Start services adds service startup, readiness checks, and log tails. The Electron UI is not started during
              preflight.
            </p>
          </details>

          {!preflightOk && (
            <div className="flex flex-wrap items-center justify-between gap-3 rounded-md border border-amber-800 bg-amber-950/20 p-3 text-xs text-amber-100">
              <span>Override preflight and allow generation anyway.</span>
              <Checkbox
                checked={preflightOverride}
                onChange={(e) => onOverrideChange(e.target.checked)}
                label="Override"
              />
            </div>
          )}
        </div>
      )}

      {viewMode === "json" && (
        <div className="rounded-md border border-slate-800 bg-slate-950/60 p-3 space-y-2">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <p className="text-xs font-semibold text-slate-200">Preflight JSON</p>
            <div className="flex items-center gap-2">
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={copyJson}
                disabled={!preflightResult && !preflightError}
                aria-label="Copy preflight JSON"
                className="h-10 w-10 p-0"
              >
                <Copy className="h-4 w-4" />
              </Button>
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => {
                  const blob = new Blob([JSON.stringify(preflightPayload, null, 2)], { type: "application/json" });
                  const url = URL.createObjectURL(blob);
                  const link = document.createElement("a");
                  link.href = url;
                  link.download = "preflight.json";
                  link.click();
                  URL.revokeObjectURL(url);
                }}
                disabled={!preflightResult && !preflightError}
                aria-label="Download preflight JSON"
                className="h-10 w-10 p-0"
              >
                <Download className="h-4 w-4" />
              </Button>
            </div>
          </div>
          <pre className="max-h-96 overflow-auto whitespace-pre-wrap rounded-md border border-slate-800/70 bg-slate-950/80 p-3 text-[11px] text-slate-200">
            {JSON.stringify(preflightPayload, null, 2)}
          </pre>
          {copyStatus !== "idle" && (
            <p className="text-[11px] text-slate-400">
              {copyStatus === "copied" ? "Copied to clipboard." : "Copy failed."}
            </p>
          )}
          <p className="text-[11px] text-slate-400">
            Use this view to share the full preflight snapshot with an agent or teammate.
          </p>
        </div>
      )}
    </div>
  );
}

type DeploymentManagerBundleHelperHandle = {
  exportBundle: () => void;
};

type DeploymentManagerBundleHelperProps = {
  scenarioName: string;
  onBundleManifestChange: (value: string) => void;
  onBundleExported?: (manifestPath: string) => void;
  onDeploymentManagerUrlChange?: (url: string | null) => void;
};

const DeploymentManagerBundleHelper = forwardRef<DeploymentManagerBundleHelperHandle, DeploymentManagerBundleHelperProps>(
  ({ scenarioName, onBundleManifestChange, onBundleExported, onDeploymentManagerUrlChange }, ref) => {
  const [tier, setTier] = useState("tier-2-desktop");
  const [generationPhase, setGenerationPhase] = useState<"idle" | "building" | "exporting">("idle");
  const [exportError, setExportError] = useState<string | null>(null);
  const [buildError, setBuildError] = useState<string | null>(null);
  const [downloadUrl, setDownloadUrl] = useState<string | null>(null);
  const [downloadName, setDownloadName] = useState<string | null>(null);
  const [exportMeta, setExportMeta] = useState<{ checksum?: string; generated_at?: string } | null>(null);
  const [deploymentManagerUrl, setDeploymentManagerUrl] = useState<string | null>(null);
  const [manifestPath, setManifestPath] = useState<string | null>(null);
  const [buildJobId, setBuildJobId] = useState<string | null>(null);
  const [buildStatus, setBuildStatus] = useState<AutoBuildStatusResponse | null>(null);
  const exportTriggeredRef = useRef(false);

  useImperativeHandle(ref, () => ({
    exportBundle: () => {
      handleExport();
    }
  }));

  useEffect(() => {
    setGenerationPhase("idle");
    setExportError(null);
    setBuildError(null);
    setBuildJobId(null);
    setBuildStatus(null);
    exportTriggeredRef.current = false;
  }, [scenarioName]);

  const resetGenerationState = () => {
    setExportError(null);
    setBuildError(null);
    setExportMeta(null);
    setDownloadUrl(null);
    setDownloadName(null);
    setManifestPath(null);
    setDeploymentManagerUrl(null);
    setBuildJobId(null);
    setBuildStatus(null);
    exportTriggeredRef.current = false;
    if (onDeploymentManagerUrlChange) {
      onDeploymentManagerUrlChange(null);
    }
  };

  const runExport = useCallback(async () => {
    setGenerationPhase("exporting");
    try {
      const response = await exportBundleFromDeploymentManager({
        scenario: scenarioName,
        tier
      });
      const blob = new Blob([JSON.stringify(response.manifest, null, 2)], { type: "application/json" });
      const url = URL.createObjectURL(blob);
      const filename = `${scenarioName}-${tier}-bundle.json`;
      setDownloadUrl(url);
      setDownloadName(filename);
      setExportMeta({ checksum: response.checksum, generated_at: response.generated_at });
      setDeploymentManagerUrl(response.deployment_manager_url ?? null);
      if (onDeploymentManagerUrlChange) {
        onDeploymentManagerUrlChange(response.deployment_manager_url ?? null);
      }
      if (response.manifest_path) {
        setManifestPath(response.manifest_path);
        onBundleManifestChange(response.manifest_path);
        onBundleExported?.(response.manifest_path);
      }
    } catch (error) {
      setExportError((error as Error).message);
    } finally {
      setGenerationPhase("idle");
    }
  }, [scenarioName, tier, onBundleManifestChange, onBundleExported, onDeploymentManagerUrlChange]);

  const handleExport = async () => {
    resetGenerationState();
    if (!scenarioName.trim()) {
      setExportError("Enter a scenario name above before exporting the bundle.");
      return;
    }
    setGenerationPhase("building");
    try {
      const response = await startDeploymentManagerAutoBuild({
        scenario: scenarioName
      });
      setBuildStatus(response);
      if (response.build_id) {
        setBuildJobId(response.build_id);
      }
      if (response.status === "skipped") {
        exportTriggeredRef.current = true;
        await runExport();
      }
    } catch (error) {
      setBuildError((error as Error).message);
      setGenerationPhase("idle");
    }
  };

  const buildProgress = useMemo(() => {
    const targets = buildStatus?.targets ?? [];
    let total = 0;
    let completed = 0;
    for (const target of targets) {
      for (const platform of target.platforms || []) {
        total += 1;
        if (["success", "failed", "skipped"].includes(platform.status)) {
          completed += 1;
        }
      }
    }
    if (total === 0) {
      return 0;
    }
    return Math.round((completed / total) * 100);
  }, [buildStatus]);

  const buildTargets = buildStatus?.targets ?? [];
  const buildLog = buildStatus?.build_log ?? [];
  const buildErrors = buildStatus?.error_log ?? [];

  useEffect(() => {
    if (!buildJobId) {
      return;
    }
    let active = true;
    let timeoutId: number | undefined;

    const poll = async () => {
      try {
        const status = await fetchDeploymentManagerAutoBuildStatus(buildJobId);
        if (!active) return;
        setBuildStatus(status);
        const doneStatuses = new Set(["success", "partial", "failed", "skipped", "dry_run"]);
        if (doneStatuses.has(status.status)) {
          if ((status.status === "success" || status.status === "skipped") && !exportTriggeredRef.current) {
            exportTriggeredRef.current = true;
            await runExport();
          } else if (status.status !== "success" && status.status !== "skipped") {
            setGenerationPhase("idle");
            setBuildError(`Auto build finished with status "${status.status}".`);
          }
          return;
        }
        const delay = status.poll_after_ms && status.poll_after_ms > 0 ? status.poll_after_ms : 2000;
        timeoutId = window.setTimeout(poll, delay);
      } catch (error) {
        if (!active) return;
        setBuildError((error as Error).message);
        timeoutId = window.setTimeout(poll, 4000);
      }
    };

    poll();

    return () => {
      active = false;
      if (timeoutId) {
        window.clearTimeout(timeoutId);
      }
    };
  }, [buildJobId, runExport]);

  const isBusy = generationPhase !== "idle";
  const buttonLabel =
    generationPhase === "building"
      ? "Building bundle..."
      : generationPhase === "exporting"
        ? "Exporting bundle..."
        : "Generate bundle";
  const buildStatusTone = (status?: string) => {
    switch (status) {
      case "success":
        return "bg-emerald-500/10 text-emerald-200 border-emerald-500/40";
      case "partial":
        return "bg-amber-500/10 text-amber-200 border-amber-500/40";
      case "failed":
        return "bg-rose-500/10 text-rose-200 border-rose-500/40";
      case "building":
        return "bg-blue-500/10 text-blue-200 border-blue-500/40";
      default:
        return "bg-slate-500/10 text-slate-200 border-slate-500/40";
    }
  };

  return (
    <div className="rounded border border-slate-800 bg-black/20 p-3 space-y-3 text-xs text-slate-200">
      <div className="flex items-center justify-between gap-2">
        <div>
          <p className="text-xs uppercase tracking-wide text-slate-400">Bundle helper</p>
          <p className="text-sm font-semibold text-slate-100">Get bundle.json from deployment-manager</p>
        </div>
      </div>
      <p className="text-xs text-slate-400">
        Generate a bundle manifest and stage binaries/assets for offline use.
      </p>
      <div className="space-y-1">
        <Label className="text-[11px] uppercase tracking-wide text-slate-400">Tier</Label>
        <Select value={tier} onChange={(e) => setTier(e.target.value)} className="mt-0.5">
          <option value="tier-2-desktop">tier-2-desktop</option>
          <option value="tier-3-mobile" disabled>
            tier-3-mobile (desktop export only)
          </option>
        </Select>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <Button size="sm" variant="outline" onClick={handleExport} disabled={isBusy}>
          {buttonLabel}
        </Button>
        {downloadUrl && downloadName && (
          <Button
            size="sm"
            variant="outline"
            onClick={() => {
              const link = document.createElement("a");
              link.href = downloadUrl;
              link.download = downloadName;
              link.click();
            }}
          >
            Download bundle
          </Button>
        )}
      </div>
      {generationPhase === "building" && !buildStatus && (
        <p className="text-[11px] text-slate-400">Starting auto build…</p>
      )}
      {buildStatus && (
        <div className="rounded-lg border border-slate-800 bg-slate-950/50 p-3 space-y-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div>
              <p className="text-[11px] uppercase tracking-wide text-slate-400">Auto build</p>
              <p className="text-sm font-semibold text-slate-100">Bundle binaries & assets</p>
            </div>
            <span className={`rounded-full border px-2 py-0.5 text-[11px] ${buildStatusTone(buildStatus.status)}`}>
              {buildStatus.status}
            </span>
          </div>
          <div className="text-[11px] text-slate-400">
            Build ID: <span className="font-mono text-slate-300">{buildStatus.build_id}</span>
          </div>
          <div className="space-y-1">
            <div className="flex justify-between text-[11px] text-slate-400">
              <span>Progress</span>
              <span className="text-slate-300">{buildProgress}%</span>
            </div>
            <Progress value={buildProgress} />
          </div>
          {buildTargets.length > 0 && (
            <div className="space-y-2">
              <p className="text-[11px] uppercase tracking-wide text-slate-500">Targets</p>
              {buildTargets.map((target) => (
                <div key={target.id} className="rounded-md border border-slate-800/70 bg-slate-950/70 p-2 space-y-1">
                  <div className="flex items-center justify-between gap-2">
                    <p className="text-xs font-semibold text-slate-200">{target.id}</p>
                    <span className="text-[11px] text-slate-500">{target.folder}</span>
                  </div>
                  <div className="space-y-1">
                    {(target.platforms || []).map((platform) => {
                      const isBuilding = platform.status === "building";
                      const isSuccess = platform.status === "success";
                      const isFailed = platform.status === "failed";
                      return (
                        <div key={platform.name} className="flex flex-wrap items-center justify-between gap-2 text-[11px]">
                          <span className="flex items-center gap-2 text-slate-300">
                            {isSuccess ? (
                              <CheckCircle className="h-4 w-4 text-emerald-400" />
                            ) : isFailed ? (
                              <AlertTriangle className="h-4 w-4 text-rose-400" />
                            ) : isBuilding ? (
                              <Loader2 className="h-4 w-4 text-blue-400 animate-spin" />
                            ) : (
                              <span className="h-4 w-4 rounded-full border border-slate-600" />
                            )}
                            {platform.name}
                          </span>
                          <span className="max-w-[240px] truncate text-slate-500">{platform.output_path}</span>
                        </div>
                      );
                    })}
                  </div>
                </div>
              ))}
            </div>
          )}
          {buildLog.length > 0 && (
            <div>
              <p className="text-[11px] uppercase tracking-wide text-slate-500 mb-1">Build log</p>
              <div className="max-h-40 overflow-auto rounded-md border border-slate-800/70 bg-slate-950/80 p-2 font-mono text-[11px] text-slate-300">
                {buildLog.map((line, idx) => (
                  <div key={idx}>{line}</div>
                ))}
              </div>
            </div>
          )}
          {buildErrors.length > 0 && (
            <div>
              <p className="text-[11px] uppercase tracking-wide text-rose-300 mb-1">Build errors</p>
              <div className="max-h-40 overflow-auto rounded-md border border-rose-500/40 bg-rose-950/40 p-2 font-mono text-[11px] text-rose-200">
                {buildErrors.map((line, idx) => (
                  <div key={idx}>{line}</div>
                ))}
              </div>
            </div>
          )}
          {buildStatus.check_command && (
            <p className="text-[11px] text-slate-400">
              Check progress: <span className="font-mono text-slate-300">{buildStatus.check_command}</span>
            </p>
          )}
        </div>
      )}
      {buildError && (
        <p className="rounded border border-amber-500/40 bg-amber-500/10 px-2 py-1 text-amber-100">
          {buildError}
        </p>
      )}
      {exportMeta && (
          <p className="text-[11px] text-slate-300">
            Generated {exportMeta.generated_at || "just now"}
            {exportMeta.checksum ? ` · checksum ${exportMeta.checksum.slice(0, 8)}…` : ""}
          </p>
      )}
      {manifestPath && (
        <p className="text-[11px] text-slate-300">
          Saved to {manifestPath}
        </p>
      )}
      {exportError && (
        <p className="rounded border border-amber-500/40 bg-amber-500/10 px-2 py-1 text-amber-100">
          {exportError}
        </p>
      )}
      {!exportError && !downloadUrl && (
        <p className="text-[11px] text-slate-400">
          Start deployment-manager (`vrooli scenario start deployment-manager`), then click Generate. We’ll auto-build
          scenario binaries first, then export the manifest + staged files into the scenario bundle folder.
        </p>
      )}
    </div>
  );
  }
);

interface ConnectionTester {
  isPending: boolean;
  mutate: () => void;
}

interface ExternalServerSectionProps {
  proxyUrl: string;
  onProxyUrlChange: (value: string) => void;
  scenarioName: string;
  proxyHints?: ProxyHintsResponse | null;
  connectionTester: ConnectionTester;
  connectionResult: ProbeResponse | null;
  connectionError: string | null;
  autoManageTier1: boolean;
  onAutoManageTier1Change: (value: boolean) => void;
  vrooliBinaryPath: string;
  onVrooliBinaryPathChange: (value: string) => void;
}

function ExternalServerSection({
  proxyUrl,
  onProxyUrlChange,
  scenarioName,
  proxyHints,
  connectionTester,
  connectionResult,
  connectionError,
  autoManageTier1,
  onAutoManageTier1Change,
  vrooliBinaryPath,
  onVrooliBinaryPathChange
}: ExternalServerSectionProps) {
  return (
    <div className="rounded-lg border border-slate-700 bg-slate-900/40 p-4 space-y-3">
      <div>
        <Label htmlFor="proxyUrl">Proxy URL</Label>
        <p className="text-xs text-slate-400 mb-2">
          Paste the exact URL you open in your browser (for example <code>https://app-monitor.yourdomain.com/apps/{scenarioName || "scenario"}/proxy/</code>). This keeps all traffic inside the secure tunnel.
        </p>
      </div>
      <Input
        id="proxyUrl"
        value={proxyUrl}
        onChange={(e) => onProxyUrlChange(e.target.value)}
        placeholder="https://app-monitor.example.dev/apps/picker-wheel/proxy/"
      />
      <p className="text-xs text-slate-400 space-x-1">
        <span>Desktop apps simply load this URL. Use the Cloudflare/app-monitor address if you want remote access.</span>
      </p>

      {proxyHints?.hints && proxyHints.hints.length > 0 && (
        <div className="rounded border border-slate-800 bg-black/20 p-3 space-y-2">
          <p className="text-xs uppercase tracking-wide text-slate-400">Detected URLs</p>
          <div className="space-y-2">
            {proxyHints.hints.map((hint) => (
              <button
                key={hint.url}
                type="button"
                onClick={() => onProxyUrlChange(hint.url)}
                className="w-full rounded border border-slate-700 bg-slate-950/30 px-3 py-2 text-left text-sm hover:border-blue-500"
              >
                <div className="font-medium text-slate-200">{hint.url}</div>
                <div className="text-xs text-slate-400">
                  {hint.message} · Source: {hint.source}
                </div>
              </button>
            ))}
          </div>
        </div>
      )}

      <div className="flex items-center gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => connectionTester.mutate()}
          disabled={connectionTester.isPending || !proxyUrl}
        >
          {connectionTester.isPending ? "Testing..." : "Test connection"}
        </Button>
        {connectionResult && connectionResult.server.status === "ok" && connectionResult.api.status === "ok" && (
          <span className="text-xs text-green-300">Both URLs responded ✔</span>
        )}
        {connectionError && <span className="text-xs text-red-300">{connectionError}</span>}
      </div>
      {(connectionResult?.server || connectionResult?.api) && (
        <div className="rounded border border-slate-800 bg-black/20 p-3 text-xs text-slate-300 space-y-1">
          <p className="font-semibold text-slate-200">Connectivity snapshot</p>
          <p>
            UI URL: {connectionResult?.server.status === "ok" ? "reachable" : connectionResult?.server.message || "no response"}
          </p>
          <p>
            API URL: {connectionResult?.api.status === "ok" ? "reachable" : connectionResult?.api.message || "no response"}
          </p>
        </div>
      )}

      <Checkbox
        checked={autoManageTier1}
        onChange={(e) => onAutoManageTier1Change(e.target.checked)}
        label="Automatically run the scenario locally with the vrooli CLI (advanced)"
      />
      <p className="text-xs text-slate-400">
        If enabled, the desktop app will look for the `vrooli` binary, run `vrooli setup`, and start/stop the scenario on the user's machine. Enable only when the end user expects to host the full stack locally.
      </p>

      <Label htmlFor="vrooliBinary" className={autoManageTier1 ? undefined : "text-slate-500"}>
        vrooli CLI path
      </Label>
      <Input
        id="vrooliBinary"
        value={vrooliBinaryPath}
        onChange={(e) => onVrooliBinaryPathChange(e.target.value)}
        disabled={!autoManageTier1}
        placeholder="vrooli"
      />
    </div>
  );
}

interface EmbeddedServerSectionProps {
  serverPort: number;
  onServerPortChange: (port: number) => void;
  localServerPath: string;
  onLocalServerPathChange: (path: string) => void;
  localApiEndpoint: string;
  onLocalApiEndpointChange: (endpoint: string) => void;
}

function EmbeddedServerSection({
  serverPort,
  onServerPortChange,
  localServerPath,
  onLocalServerPathChange,
  localApiEndpoint,
  onLocalApiEndpointChange
}: EmbeddedServerSectionProps) {
  return (
    <div className="rounded-lg border border-yellow-800 bg-yellow-950/10 p-4 space-y-3">
      <p className="text-sm text-yellow-200">
        Embedded servers require more manual work. Make sure the scenario's API can run within the wrapper (Node script or executable) and that resource usage fits the target machine.
      </p>
      <div className="grid gap-3 sm:grid-cols-2">
        <div>
          <Label htmlFor="serverPort">Server Port</Label>
          <Input
            id="serverPort"
            type="number"
            value={serverPort}
            onChange={(e) => onServerPortChange(Number(e.target.value))}
            min={1}
          />
        </div>
        <div>
          <Label htmlFor="localServerPath">Server Entry</Label>
          <Input
            id="localServerPath"
            value={localServerPath}
            onChange={(e) => onLocalServerPathChange(e.target.value)}
            placeholder="ui/server.js"
          />
        </div>
        <div className="sm:col-span-2">
          <Label htmlFor="localApiEndpoint">API Endpoint</Label>
          <Input
            id="localApiEndpoint"
            value={localApiEndpoint}
            onChange={(e) => onLocalApiEndpointChange(e.target.value)}
          />
        </div>
      </div>
    </div>
  );
}

interface PlatformSelectorProps {
  platforms: PlatformSelection;
  onPlatformChange: (platform: string, checked: boolean) => void;
}

function PlatformSelector({ platforms, onPlatformChange }: PlatformSelectorProps) {
  return (
    <div>
      <Label>Target Platforms</Label>
      <div className="mt-2 flex flex-wrap gap-4">
        <Checkbox
          id="platformWin"
          checked={platforms.win}
          onChange={(e) => onPlatformChange("win", e.target.checked)}
          label="Windows"
        />
        <Checkbox
          id="platformMac"
          checked={platforms.mac}
          onChange={(e) => onPlatformChange("mac", e.target.checked)}
          label="macOS"
        />
        <Checkbox
          id="platformLinux"
          checked={platforms.linux}
          onChange={(e) => onPlatformChange("linux", e.target.checked)}
          label="Linux"
        />
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
}

export function GeneratorForm({
  selectedTemplate,
  onTemplateChange,
  onBuildStart,
  scenarioName,
  onScenarioNameChange,
  selectionSource,
  onOpenSigningTab
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
  const [preflightResult, setPreflightResult] = useState<BundlePreflightResponse | null>(null);
  const [preflightError, setPreflightError] = useState<string | null>(null);
  const [preflightPending, setPreflightPending] = useState(false);
  const [preflightOverride, setPreflightOverride] = useState(false);
  const [preflightSecrets, setPreflightSecrets] = useState<Record<string, string>>({});
  const [preflightStartServices, setPreflightStartServices] = useState(false);
  const [deploymentManagerUrl, setDeploymentManagerUrl] = useState<string | null>(null);
  const [lastLoadedScenario, setLastLoadedScenario] = useState<string | null>(null);
  const [signingEnabledForBuild, setSigningEnabledForBuild] = useState(false);
  const isUpdateMode = selectionSource === "inventory";
  const bundleHelperRef = useRef<DeploymentManagerBundleHelperHandle>(null);

  const connectionDecision = useMemo(
    () => decideConnection(deploymentMode, serverType),
    [deploymentMode, serverType]
  );
  const isBundled = connectionDecision.kind === "bundled-runtime";
  const requiresRemoteConfig = connectionDecision.requiresProxyUrl;
  const standardOutputPath = useMemo(
    () => (scenarioName ? `scenarios/${scenarioName}/platforms/electron` : "scenarios/<scenario>/platforms/electron"),
    [scenarioName]
  );
  const stagingPreviewPath = useMemo(
    () =>
      scenarioName
        ? `scenarios/scenario-to-desktop/data/staging/${scenarioName}/<build-id>`
        : "scenarios/scenario-to-desktop/data/staging/<scenario>/<build-id>",
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
  const missingPreflightSecrets = useMemo(() => {
    if (!preflightResult?.secrets) {
      return [];
    }
    return preflightResult.secrets.filter((secret) => secret.required && !secret.has_value);
  }, [preflightResult]);
  const preflightValidationOk = Boolean(preflightResult?.validation?.valid);
  const preflightReady = Boolean(preflightResult?.ready?.ready);
  const preflightSecretsReady = missingPreflightSecrets.length === 0;
  const preflightOk = Boolean(preflightResult) && preflightValidationOk && preflightReady && preflightSecretsReady;

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
    if (!isBundled) {
      setPreflightResult(null);
      setPreflightError(null);
      setPreflightOverride(false);
      setPreflightSecrets({});
      return;
    }
    setPreflightResult(null);
    setPreflightError(null);
    setPreflightOverride(false);
    setPreflightSecrets({});
  }, [bundleManifestPath, isBundled]);

  useEffect(() => {
    setScenarioLocked(selectionSource === "inventory");
  }, [selectionSource]);

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

  const runPreflight = async (overrideSecrets?: Record<string, string>, manifestPathOverride?: string) => {
    const manifestPath = (manifestPathOverride ?? bundleManifestPath).trim();
    if (!manifestPath) {
      setPreflightError("Provide bundle_manifest_path before running preflight.");
      return;
    }
    setPreflightPending(true);
    setPreflightError(null);
    try {
      const baseSecrets = overrideSecrets ?? preflightSecrets;
      const filteredSecrets = Object.entries(baseSecrets)
        .filter(([, value]) => value.trim())
        .reduce<Record<string, string>>((acc, [key, value]) => {
          acc[key] = value;
          return acc;
        }, {});
      const result = await runBundlePreflight({
        bundle_manifest_path: manifestPath,
        secrets: Object.keys(filteredSecrets).length > 0 ? filteredSecrets : undefined,
        start_services: preflightStartServices,
        log_tail_lines: preflightStartServices ? 80 : undefined
      });
      setPreflightResult(result);
      setPreflightOverride(false);
    } catch (error) {
      setPreflightResult(null);
      setPreflightError((error as Error).message);
    } finally {
      setPreflightPending(false);
    }
  };

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

  const { data: signingConfigResp, isFetching: loadingSigningConfig, refetch: refetchSigningConfig } = useQuery({
    queryKey: ["signing-config-inline", scenarioName],
    queryFn: () => fetchSigningConfig(scenarioName),
    enabled: Boolean(scenarioName)
  });

  const { data: signingReadiness, isFetching: loadingSigningReadiness, refetch: refetchSigningReadiness } = useQuery<SigningReadinessResponse>({
    queryKey: ["signing-readiness-inline", scenarioName],
    queryFn: () => checkSigningReadiness(scenarioName),
    enabled: Boolean(scenarioName)
  });

  useEffect(() => {
    if (signingConfigResp?.config) {
      setSigningEnabledForBuild(Boolean(signingConfigResp.config.enabled));
    } else if (scenarioName) {
      setSigningEnabledForBuild(false);
    }
  }, [scenarioName, signingConfigResp?.config]);

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
    applySavedConnection(selectedScenario?.connection_config);
    setLastLoadedScenario(configKey);
  }, [scenarioName, selectedScenario?.connection_config?.updated_at, lastLoadedScenario, selectedScenario?.connection_config]);

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

    const signingConfig = signingConfigResp?.config;
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
      codeSigning: signingEnabledForBuild ? signingConfig : { enabled: false }
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

  return (
    <Card>
      <CardHeader>
      <CardTitle className="flex items-center gap-2">
        <Rocket className="h-5 w-5" />
        Generate Desktop App
      </CardTitle>
    </CardHeader>
    <CardContent>
      <form onSubmit={handleSubmit} className="space-y-4">
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
              preflightResult={preflightResult}
              preflightPending={preflightPending}
              preflightError={preflightError}
              missingSecrets={missingPreflightSecrets}
              secretInputs={preflightSecrets}
              preflightOk={preflightOk}
              preflightOverride={preflightOverride}
              preflightStartServices={preflightStartServices}
              preflightLogTails={preflightResult?.log_tails}
              deploymentManagerUrl={deploymentManagerUrl}
              onReexportBundle={() => bundleHelperRef.current?.exportBundle()}
              onOverrideChange={setPreflightOverride}
              onStartServicesChange={setPreflightStartServices}
              onSecretChange={(id, value) => {
                setPreflightSecrets((prev) => ({ ...prev, [id]: value }));
              }}
              onRun={runPreflight}
            />
          )}

          <PlatformSelector platforms={platforms} onPlatformChange={handlePlatformChange} />

          <SigningInlineSection
            scenarioName={scenarioName}
            signingEnabled={signingEnabledForBuild}
            signingConfig={signingConfigResp?.config || null}
            readiness={signingReadiness}
            loading={loadingSigningConfig || loadingSigningReadiness}
            onToggleSigning={setSigningEnabledForBuild}
            onOpenSigning={() => onOpenSigningTab(scenarioName)}
            onRefresh={() => {
              if (scenarioName) {
                refetchSigningConfig();
                refetchSigningReadiness();
              }
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
            onFrameworkChange(nextFramework);
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
      </CardContent>
    </Card>
  );
}
