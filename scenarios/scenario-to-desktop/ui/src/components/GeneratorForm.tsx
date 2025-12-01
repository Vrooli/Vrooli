import { useEffect, useMemo, useState, type FormEvent } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";
import { fetchProxyHints, generateDesktop, probeEndpoints, type DesktopConfig, type ProbeResponse, type ProxyHintsResponse } from "../lib/api";
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card";
import { Input } from "./ui/input";
import { Label } from "./ui/label";
import { Select } from "./ui/select";
import { Checkbox } from "./ui/checkbox";
import { Button } from "./ui/button";
import { Rocket } from "lucide-react";
import type { DesktopConnectionConfig, ScenariosResponse } from "./scenario-inventory/types";

const API_BASE = resolveApiBase({ appendSuffix: true });
const buildUrl = (path: string) => buildApiUrl(path, { baseUrl: API_BASE });

interface GeneratorFormProps {
  selectedTemplate: string;
  onTemplateChange: (template: string) => void;
  onBuildStart: (buildId: string) => void;
}

export function GeneratorForm({ selectedTemplate, onTemplateChange, onBuildStart }: GeneratorFormProps) {
  const [scenarioName, setScenarioName] = useState("");
  const [useDropdown, setUseDropdown] = useState(true);
  const [framework, setFramework] = useState("electron");
  const [serverType, setServerType] = useState("external");
  const [deploymentMode, setDeploymentMode] = useState("external-server");
  const [platforms, setPlatforms] = useState({
    win: true,
    mac: true,
    linux: true
  });
  const [outputPath, setOutputPath] = useState("./desktop-app");
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

  const selectedDeployment = useMemo(
    () => DEPLOYMENT_OPTIONS.find((option) => option.value === deploymentMode) ?? DEPLOYMENT_OPTIONS[0],
    [deploymentMode]
  );
  const selectedServerType = useMemo(
    () => SERVER_TYPE_OPTIONS.find((option) => option.value === serverType) ?? SERVER_TYPE_OPTIONS[0],
    [serverType]
  );

  // Fetch available scenarios
  const { data: scenariosData, isLoading: loadingScenarios } = useQuery<ScenariosResponse>({
    queryKey: ['scenarios-desktop-status'],
    queryFn: async () => {
      const res = await fetch(buildUrl('/scenarios/desktop-status'));
      if (!res.ok) throw new Error('Failed to fetch scenarios');
      return res.json();
    },
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

  const applySavedConnection = (config?: DesktopConnectionConfig | null) => {
    if (!config) return;
    setDeploymentMode(config.deployment_mode ?? "external-server");
    setProxyUrl(config.proxy_url ?? config.server_url ?? "");
    setAutoManageTier1(config.auto_manage_vrooli ?? false);
    setVrooliBinaryPath(config.vrooli_binary_path ?? "vrooli");
    setBundleManifestPath(config.bundle_manifest_path ?? "");
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
    if (deploymentMode === "bundled") {
      setServerType("external");
      setAutoManageTier1(false);
    }
  }, [deploymentMode]);

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();

    const selectedPlatforms = Object.entries(platforms)
      .filter(([, enabled]) => enabled)
      .map(([platform]) => platform);

    if (selectedPlatforms.length === 0) {
      alert("Please select at least one target platform");
      return;
    }

    const isBundled = deploymentMode === "bundled";
    if (isBundled && !bundleManifestPath) {
      alert("Provide bundle_manifest_path from deployment-manager before generating a bundled build.");
      return;
    }

    const requiresRemoteConfig = serverType === "external" && !isBundled;
    if (requiresRemoteConfig && !proxyUrl) {
      alert("Provide the proxy URL you use in the browser (for example https://app-monitor.example.com/apps/<scenario>/proxy/).");
      return;
    }

    const resolvedServerPath = isBundled ? "http://127.0.0.1" : (requiresRemoteConfig ? proxyUrl : localServerPath);
    const resolvedApiEndpoint = isBundled ? "http://127.0.0.1" : (requiresRemoteConfig ? proxyUrl : localApiEndpoint);

    const serverTypeForConfig = isBundled ? "external" : serverType;

    const config: DesktopConfig = {
      app_name: scenarioName,
      app_display_name: `${scenarioName} Desktop`,
      app_description: `Desktop application for ${scenarioName} scenario`,
      version: "1.0.0",
      author: "Vrooli Platform",
      license: "MIT",
      app_id: `com.vrooli.${scenarioName.replace(/-/g, ".")}`,
      server_type: serverTypeForConfig,
      server_port: serverPort,
      server_path: resolvedServerPath,
      api_endpoint: resolvedApiEndpoint,
      framework,
      template_type: selectedTemplate,
      platforms: selectedPlatforms,
      output_path: outputPath,
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
      deployment_mode: deploymentMode,
      auto_manage_vrooli: autoManageTier1,
      vrooli_binary_path: vrooliBinaryPath,
      proxy_url: requiresRemoteConfig ? proxyUrl : undefined,
      external_server_url: requiresRemoteConfig ? proxyUrl : undefined,
      external_api_url: requiresRemoteConfig ? undefined : undefined,
      bundle_manifest_path: isBundled ? bundleManifestPath : undefined
    };

    if (!requiresRemoteConfig && !isBundled) {
      config.external_api_url = localApiEndpoint;
    }

    generateMutation.mutate(config);
  };

  const handlePlatformChange = (platform: string, checked: boolean) => {
    setPlatforms((prev) => ({ ...prev, [platform]: checked }));
  };

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
          <div>
            <div className="flex items-center justify-between mb-1.5 gap-3">
              <Label htmlFor="scenarioName">Scenario Name</Label>
              <div className="flex items-center gap-2">
                {selectedScenario?.connection_config && (
                  <button
                    type="button"
                    onClick={() => applySavedConnection(selectedScenario.connection_config)}
                    className="text-xs text-blue-300 hover:text-blue-200"
                  >
                    Load saved URLs
                  </button>
                )}
                <button
                  type="button"
                  onClick={() => setUseDropdown(!useDropdown)}
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
                onChange={(e) => setScenarioName(e.target.value)}
                required
                className="mt-1.5"
                disabled={loadingScenarios}
              >
                <option value="">
                  {loadingScenarios ? "Loading scenarios..." : "Select a scenario..."}
                </option>
                {scenariosData?.scenarios
                  .sort((a, b) => a.name.localeCompare(b.name))
                  .map(scenario => (
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
                onChange={(e) => setScenarioName(e.target.value)}
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

          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <Label htmlFor="framework">Framework</Label>
              <Select
                id="framework"
                value={framework}
                onChange={(e) => setFramework(e.target.value)}
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
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <Label htmlFor="deploymentMode">Deployment intent</Label>
              <Select
                id="deploymentMode"
                value={deploymentMode}
                onChange={(e) => setDeploymentMode(e.target.value)}
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
                onChange={(e) => setServerType(e.target.value)}
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

          {deploymentMode === "bundled" ? (
            <div className="rounded-lg border border-emerald-900 bg-emerald-950/10 p-4 space-y-3">
              <Label htmlFor="bundleManifest">bundle_manifest_path</Label>
              <Input
                id="bundleManifest"
                value={bundleManifestPath}
                onChange={(e) => setBundleManifestPath(e.target.value)}
                placeholder="/home/you/Vrooli/docs/deployment/examples/manifests/desktop-happy.json"
              />
              <p className="text-xs text-emerald-200/80">
                Stages the manifest and bundled binaries so the packaged runtime can launch the scenario offline.
              </p>
            </div>
          ) : serverType === "external" ? (
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
                onChange={(e) => setProxyUrl(e.target.value)}
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
                        onClick={() => setProxyUrl(hint.url)}
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
                  onClick={() => connectionMutation.mutate()}
                  disabled={connectionMutation.isPending || !proxyUrl}
                >
                  {connectionMutation.isPending ? "Testing..." : "Test connection"}
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
                onChange={(e) => setAutoManageTier1(e.target.checked)}
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
                onChange={(e) => setVrooliBinaryPath(e.target.value)}
                disabled={!autoManageTier1}
                placeholder="vrooli"
              />
            </div>
          ) : (
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
                    onChange={(e) => setServerPort(Number(e.target.value))}
                    min={1}
                  />
                </div>
                <div>
                  <Label htmlFor="localServerPath">Server Entry</Label>
                  <Input
                    id="localServerPath"
                    value={localServerPath}
                    onChange={(e) => setLocalServerPath(e.target.value)}
                    placeholder="ui/server.js"
                  />
                </div>
                <div className="sm:col-span-2">
                  <Label htmlFor="localApiEndpoint">API Endpoint</Label>
                  <Input
                    id="localApiEndpoint"
                    value={localApiEndpoint}
                    onChange={(e) => setLocalApiEndpoint(e.target.value)}
                  />
                </div>
              </div>
            </div>
          )}

          <div>
            <Label>Target Platforms</Label>
            <div className="mt-2 flex flex-wrap gap-4">
              <Checkbox
                id="platformWin"
                checked={platforms.win}
                onChange={(e) => handlePlatformChange("win", e.target.checked)}
                label="Windows"
              />
              <Checkbox
                id="platformMac"
                checked={platforms.mac}
                onChange={(e) => handlePlatformChange("mac", e.target.checked)}
                label="macOS"
              />
              <Checkbox
                id="platformLinux"
                checked={platforms.linux}
                onChange={(e) => handlePlatformChange("linux", e.target.checked)}
                label="Linux"
              />
            </div>
          </div>

          <div>
            <Label htmlFor="outputPath">Output Directory</Label>
            <Input
              id="outputPath"
              value={outputPath}
              onChange={(e) => setOutputPath(e.target.value)}
              placeholder="./desktop-app"
              className="mt-1.5"
            />
            <p className="mt-1 text-xs text-slate-400">
              scenario-to-desktop will place the Electron wrapper here. For analyzer-generated builds we still recommend using the scenario's `platforms/electron` folder.
            </p>
          </div>

          <Button
            type="submit"
            className="w-full"
            disabled={generateMutation.isPending}
          >
            {generateMutation.isPending ? "Generating..." : "Generate Desktop Application"}
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
const DEPLOYMENT_OPTIONS = [
  {
    value: "external-server",
    label: "Thin client (connect to your Vrooli server)",
    description:
      "Connects to the scenario you already started via `vrooli scenario start`. APIs/resources stay on that machine; the desktop build just hosts the UI shell.",
    docs: "https://github.com/vrooli/vrooli/blob/main/docs/deployment/tiers/tier-2-desktop.md"
  },
  {
    value: "cloud-api",
    label: "Cloud API bundle (coming soon)",
    description:
      "Planned: package a cloud-friendly API target generated by deployment-manager. Disabled until bundle manifests exist.",
    docs: "https://github.com/vrooli/vrooli/blob/main/docs/deployment/tiers/tier-4-saas.md"
  },
  {
    value: "bundled",
    label: "Fully bundled/offline",
    description:
      "Ships the API/resources inside the desktop installer using a bundle.json manifest exported by deployment-manager.",
    docs: "https://github.com/vrooli/vrooli/blob/main/docs/deployment/tiers/tier-2-desktop.md"
  }
];

const SERVER_TYPE_OPTIONS = [
  {
    value: "external",
    label: "External (connect to your Vrooli server)",
    description:
      "Recommended. Desktop loads UI from the browser URL you already use (LAN or Cloudflare) and calls the remote API.",
    docs: "https://github.com/vrooli/vrooli/blob/main/docs/deployment/tiers/tier-1-local-dev.md"
  },
  {
    value: "static",
    label: "Static files (UI only)",
    description:
      "Ships only the built UI. Use when the scenario has no API or when you plan to wire every API request to a hosted endpoint manually.",
    docs: "https://github.com/vrooli/vrooli/blob/main/docs/deployment/tiers/tier-2-desktop.md"
  },
  {
    value: "node",
    label: "Embedded Node server",
    description:
      "Experimental. Runs a Node script bundled with the app (e.g., lightweight APIs) and proxies requests locally.",
    docs: "https://github.com/vrooli/vrooli/blob/main/docs/deployment/scenarios/scenario-to-desktop.md"
  },
  {
    value: "executable",
    label: "Executable / background binary",
    description:
      "Launches a binary packaged with the desktop app. Useful for custom services but requires careful resource planning.",
    docs: "https://github.com/vrooli/vrooli/blob/main/docs/deployment/scenarios/scenario-to-desktop.md"
  }
];
