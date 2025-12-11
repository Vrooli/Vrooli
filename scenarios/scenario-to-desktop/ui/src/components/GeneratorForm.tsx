import { useEffect, useMemo, useState, type FormEvent } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import {
  exportBundleFromDeploymentManager,
  fetchProxyHints,
  fetchScenarioDesktopStatus,
  generateDesktop,
  fetchScenarioPort,
  fetchSigningConfig,
  checkSigningReadiness,
  probeEndpoints,
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
import { AlertTriangle, ExternalLink, RefreshCw, Rocket, ShieldCheck } from "lucide-react";
import type { DesktopConnectionConfig, ScenarioDesktopStatus, ScenariosResponse } from "./scenario-inventory/types";
import {
  DEFAULT_DEPLOYMENT_MODE,
  DEFAULT_SERVER_TYPE,
  DEPLOYMENT_OPTIONS,
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
    app_display_name: options.appDisplayName || `${options.scenarioName} Desktop`,
    app_description: options.appDescription || `Desktop application for ${options.scenarioName} scenario`,
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
  useDropdown: boolean;
  loadingScenarios: boolean;
  scenariosData?: ScenariosResponse;
  selectedScenario?: ScenarioDesktopStatus;
  onScenarioChange: (name: string) => void;
  onToggleInput: () => void;
  onLoadSaved?: () => void;
  locked?: boolean;
  onUnlock?: () => void;
}

function ScenarioSelector({
  scenarioName,
  useDropdown,
  loadingScenarios,
  scenariosData,
  selectedScenario,
  onScenarioChange,
  onToggleInput,
  onLoadSaved,
  locked = false,
  onUnlock
}: ScenarioSelectorProps) {
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
            onClick={onToggleInput}
            className="text-xs text-blue-400 hover:text-blue-300"
          >
            {useDropdown ? "Enter manually" : "Select from list"}
          </button>
        </div>
      </div>

      {useDropdown ? (
        <Select
          id="scenarioName"
          value={scenarioName}
          onChange={(e) => onScenarioChange(e.target.value)}
          className="mt-1.5"
          disabled={loadingScenarios}
        >
          <option value="">
            {loadingScenarios ? "Loading scenarios..." : "Select a scenario..."}
          </option>
          {scenariosData?.scenarios
            .sort((a, b) => a.name.localeCompare(b.name))
            .map((scenario) => (
              <option key={scenario.name} value={scenario.name}>
                {scenario.name}
                {scenario.display_name ? ` (${scenario.display_name})` : ""}
                {scenario.has_desktop ? " — Desktop ready" : ""}
              </option>
            ))}
        </Select>
      ) : (
        <Input
          id="scenarioName"
          value={scenarioName}
          onChange={(e) => onScenarioChange(e.target.value)}
          placeholder="e.g., picker-wheel"
          required
          className="mt-1.5"
        />
      )}
      <p className="mt-1.5 text-xs text-slate-400">
        {useDropdown
          ? "Select from available scenarios that don't have desktop versions yet"
          : "Enter scenario name manually (must exist in scenarios directory)"}
      </p>
    </div>
  );
}

interface FrameworkTemplateSectionProps {
  framework: string;
  onFrameworkChange: (framework: string) => void;
  selectedTemplate: string;
  onTemplateChange: (template: string) => void;
}

function FrameworkTemplateSection({
  framework,
  onFrameworkChange,
  selectedTemplate,
  onTemplateChange
}: FrameworkTemplateSectionProps) {
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      <div>
        <Label htmlFor="framework">Framework</Label>
        <Select
          id="framework"
          value={framework}
          onChange={(e) => onFrameworkChange(e.target.value)}
          className="mt-1.5"
        >
          <option value="electron">Electron</option>
          <option value="tauri">Tauri</option>
          <option value="neutralino">Neutralino</option>
        </Select>
      </div>

      <div>
        <Label htmlFor="template">Template</Label>
        <Select
          id="template"
          value={selectedTemplate}
          onChange={(e) => onTemplateChange(e.target.value)}
          className="mt-1.5"
        >
          <option value="basic">Basic</option>
          <option value="advanced">Advanced</option>
          <option value="multi_window">Multi-Window</option>
          <option value="kiosk">Kiosk Mode</option>
        </Select>
        <p className="mt-1.5 text-xs text-slate-400">
          All templates share the same codebase. If you change your mind later, switch templates here or from the
          Generated Apps tab - your scenario stays intact.
        </p>
      </div>
    </div>
  );
}

interface DeploymentServerSectionProps {
  deploymentMode: DeploymentMode;
  selectedDeployment: { description: string; docs?: string };
  onDeploymentModeChange: (mode: DeploymentMode) => void;
  serverType: ServerType;
  selectedServerType: { description: string; docs?: string };
  onServerTypeChange: (serverType: ServerType) => void;
}

function DeploymentServerSection({
  deploymentMode,
  selectedDeployment,
  onDeploymentModeChange,
  serverType,
  selectedServerType,
  onServerTypeChange
}: DeploymentServerSectionProps) {
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      <div>
        <Label htmlFor="deploymentMode">Deployment intent</Label>
        <Select
          id="deploymentMode"
          value={deploymentMode}
          onChange={(e) => onDeploymentModeChange(e.target.value as DeploymentMode)}
          className="mt-1.5"
        >
          {DEPLOYMENT_OPTIONS.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </Select>
        <p className="mt-1.5 text-xs text-slate-400">
          {selectedDeployment.description}{" "}
          {selectedDeployment.docs && (
            <a
              href={selectedDeployment.docs}
              target="_blank"
              rel="noreferrer"
              className="text-blue-300 underline"
            >
              Learn more
            </a>
          )}
        </p>
      </div>

      <div>
        <Label htmlFor="serverType">Where should this desktop build get its data?</Label>
        <Select
          id="serverType"
          value={serverType}
          onChange={(e) => onServerTypeChange(e.target.value as ServerType)}
          className="mt-1.5"
        >
          {SERVER_TYPE_OPTIONS.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </Select>
        <p className="mt-1.5 text-xs text-slate-400">
          {selectedServerType.description}{" "}
          {selectedServerType.docs && (
            <a
              href={selectedServerType.docs}
              target="_blank"
              rel="noreferrer"
              className="text-blue-300 underline"
            >
              Learn more
            </a>
          )}
        </p>
      </div>
    </div>
  );
}

interface BundledRuntimeSectionProps {
  bundleManifestPath: string;
  onBundleManifestChange: (value: string) => void;
  scenarioName: string;
}

function BundledRuntimeSection({ bundleManifestPath, onBundleManifestChange, scenarioName }: BundledRuntimeSectionProps) {
  return (
    <div className="rounded-lg border border-emerald-900 bg-emerald-950/10 p-4 space-y-3">
      <Label htmlFor="bundleManifest">bundle_manifest_path</Label>
      <Input
        id="bundleManifest"
        value={bundleManifestPath}
        onChange={(e) => onBundleManifestChange(e.target.value)}
        placeholder="/home/you/Vrooli/docs/deployment/examples/manifests/desktop-happy.json"
      />
      <p className="text-xs text-emerald-200/80">
        Stages the manifest and bundled binaries so the packaged runtime can launch the scenario offline.
      </p>
      <DeploymentManagerBundleHelper
        scenarioName={scenarioName}
        onBundleManifestChange={onBundleManifestChange}
      />
    </div>
  );
}

function DeploymentManagerBundleHelper({
  scenarioName,
  onBundleManifestChange
}: {
  scenarioName: string;
  onBundleManifestChange: (value: string) => void;
}) {
  const [deploymentManagerUrl, setDeploymentManagerUrl] = useState("");
  const [tier, setTier] = useState("tier-2-desktop");
  const [exporting, setExporting] = useState(false);
  const [exportError, setExportError] = useState<string | null>(null);
  const [downloadUrl, setDownloadUrl] = useState<string | null>(null);
  const [downloadName, setDownloadName] = useState<string | null>(null);
  const [exportMeta, setExportMeta] = useState<{ checksum?: string; generated_at?: string } | null>(null);
  const [resolvingPort, setResolvingPort] = useState(false);

  const handleExport = async () => {
    setExportError(null);
    setExportMeta(null);
    setDownloadUrl(null);
    setDownloadName(null);
    if (!deploymentManagerUrl.trim()) {
      setExportError("Add the deployment-manager URL first (e.g., http://localhost:32022).");
      return;
    }
    if (!scenarioName.trim()) {
      setExportError("Enter a scenario name above before exporting the bundle.");
      return;
    }
    setExporting(true);
    try {
      const response = await exportBundleFromDeploymentManager({
        baseUrl: deploymentManagerUrl,
        scenario: scenarioName,
        tier
      });
      const blob = new Blob([JSON.stringify(response.manifest, null, 2)], { type: "application/json" });
      const url = URL.createObjectURL(blob);
      const filename = `${scenarioName}-${tier}-bundle.json`;
      setDownloadUrl(url);
      setDownloadName(filename);
      setExportMeta({ checksum: response.checksum, generated_at: response.generated_at });
    } catch (error) {
      setExportError((error as Error).message);
    } finally {
      setExporting(false);
    }
  };

  const handleResolvePort = async () => {
    setExportError(null);
    setResolvingPort(true);
    try {
      if (!scenarioName.trim()) {
        setExportError("Enter a scenario name above before resolving deployment-manager.");
        return;
      }
      const res = await fetchScenarioPort("deployment-manager", "UI_PORT");
      setDeploymentManagerUrl(res.url);
    } catch (error) {
      setExportError((error as Error).message);
    } finally {
      setResolvingPort(false);
    }
  };

  return (
    <div className="rounded-md border border-emerald-800/60 bg-emerald-900/30 p-3 space-y-3 text-xs text-emerald-100">
      <div className="flex items-center justify-between gap-2">
        <p className="font-semibold text-emerald-100">Get bundle.json from deployment-manager</p>
        <Button
          asChild
          size="xs"
          variant="ghost"
        >
          <a href={deploymentManagerUrl || "about:blank"} target="_blank" rel="noreferrer">
            Open UI
          </a>
        </Button>
      </div>
      <div className="grid gap-2 md:grid-cols-[2fr,1fr]">
        <div className="space-y-1">
          <Label className="text-[11px] uppercase tracking-wide text-emerald-200/70">Deployment-manager URL</Label>
          <Input
            value={deploymentManagerUrl}
            onChange={(e) => setDeploymentManagerUrl(e.target.value)}
            placeholder="http://localhost:32022"
          />
        </div>
        <div className="space-y-1">
          <Label className="text-[11px] uppercase tracking-wide text-emerald-200/70">Tier</Label>
          <Select value={tier} onChange={(e) => setTier(e.target.value)} className="mt-0.5">
            <option value="tier-2-desktop">tier-2-desktop</option>
            <option value="tier-3-mobile" disabled>
              tier-3-mobile (desktop export only)
            </option>
          </Select>
        </div>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <Button size="sm" variant="outline" onClick={handleResolvePort} disabled={resolvingPort}>
          {resolvingPort ? "Resolving..." : "Detect URL (vrooli scenario port)"}
        </Button>
        <Button size="sm" variant="secondary" onClick={handleExport} disabled={exporting}>
          {exporting ? "Exporting..." : "Export bundle.json"}
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
      {exportMeta && (
        <p className="text-[11px] text-emerald-100/80">
          Exported {exportMeta.generated_at || "just now"}
          {exportMeta.checksum ? ` · checksum ${exportMeta.checksum.slice(0, 8)}…` : ""}
        </p>
      )}
      {exportError && (
        <p className="rounded border border-amber-500/40 bg-amber-500/10 px-2 py-1 text-amber-100">
          {exportError}
        </p>
      )}
      {!exportError && !downloadUrl && (
        <p className="text-[11px] text-emerald-100/70">
          Start deployment-manager (`vrooli scenario start deployment-manager`), then click Export. Save the downloaded
          file inside this repo and paste its path above so the packager can stage it.
        </p>
      )}
    </div>
  );
}

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

        <div className="flex items-center gap-2">
          <Button type="button" size="sm" variant="outline" onClick={onRefresh} disabled={!scenarioName || loading}>
            <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
            Refresh
          </Button>
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
  const [useDropdown, setUseDropdown] = useState(true);
  const [appDisplayName, setAppDisplayName] = useState(
    scenarioName ? `${scenarioName} Desktop` : ""
  );
  const [appDescription, setAppDescription] = useState(
    scenarioName ? `Desktop application for ${scenarioName} scenario` : ""
  );
  const [iconPath, setIconPath] = useState("");
  const [displayNameEdited, setDisplayNameEdited] = useState(false);
  const [descriptionEdited, setDescriptionEdited] = useState(false);
  const [framework, setFramework] = useState("electron");
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
  const [connectionResult, setConnectionResult] = useState<ProbeResponse | null>(null);
  const [connectionError, setConnectionError] = useState<string | null>(null);
  const [lastLoadedScenario, setLastLoadedScenario] = useState<string | null>(null);
  const [signingEnabledForBuild, setSigningEnabledForBuild] = useState(false);
  const isUpdateMode = selectionSource === "inventory";

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

  const selectedDeployment = useMemo(
    () => findDeploymentOption(deploymentMode),
    [deploymentMode]
  );
  const selectedServerType = useMemo(
    () => findServerTypeOption(serverType),
    [serverType]
  );
  const isCustomLocation = locationMode === "custom";

  useEffect(() => {
    setScenarioLocked(selectionSource === "inventory");
  }, [selectionSource]);

  useEffect(() => {
    if (!scenarioName) {
      return;
    }
    const derivedName = `${scenarioName} Desktop`;
    const derivedDescription = `Desktop application for ${scenarioName} scenario`;

    if (!displayNameEdited) {
      setAppDisplayName(derivedName);
    }

    if (!descriptionEdited) {
      setAppDescription(derivedDescription);
    }
  }, [scenarioName, displayNameEdited, descriptionEdited]);

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

  const selectedScenario = scenariosData?.scenarios.find((s) => s.name === scenarioName);

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
    }
    if (config.server_type) {
      setServerType((config.server_type as ServerType) ?? DEFAULT_SERVER_TYPE);
    }
  };

  useEffect(() => {
    if (!useDropdown || !scenarioName) {
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
  }, [useDropdown, scenarioName, selectedScenario?.connection_config?.updated_at, lastLoadedScenario, selectedScenario?.connection_config]);

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

  const connectionSection =
    connectionDecision.kind === "bundled-runtime" ? (
      <BundledRuntimeSection
        bundleManifestPath={bundleManifestPath}
        onBundleManifestChange={setBundleManifestPath}
        scenarioName={scenarioName}
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
            useDropdown={useDropdown}
            loadingScenarios={loadingScenarios}
            scenariosData={scenariosData}
            selectedScenario={selectedScenario}
            onScenarioChange={onScenarioNameChange}
            onToggleInput={() => setUseDropdown(!useDropdown)}
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
                  onChange={(e) => setIconPath(e.target.value)}
                  placeholder="/home/you/Vrooli/scenarios/picker-wheel/icon.png"
                  className="mt-1.5"
                />
                <p className="mt-1 text-xs text-slate-400">
                  Optional 256px+ PNG; it will be copied into <code>assets/icon.png</code> for the build.
                </p>
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
            selectedTemplate={selectedTemplate}
            onTemplateChange={onTemplateChange}
          />

          <DeploymentServerSection
            deploymentMode={deploymentMode}
            selectedDeployment={selectedDeployment}
            onDeploymentModeChange={setDeploymentMode}
            serverType={serverType}
            selectedServerType={selectedServerType}
            onServerTypeChange={setServerType}
          />

          {connectionSection}

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
      </CardContent>
    </Card>
  );
}
