import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";
import { Button } from "../ui/button";
import { Badge } from "../ui/badge";
import { Input } from "../ui/input";
import { Label } from "../ui/label";
import { Select } from "../ui/select";
import { Checkbox } from "../ui/checkbox";
import { Loader2, Zap, CheckCircle, XCircle } from "lucide-react";
import type { ProbeResponse } from "../../lib/api";
import { probeEndpoints } from "../../lib/api";
import type { ScenarioDesktopStatus } from "./types";

const API_BASE = resolveApiBase({ appendSuffix: true });
const buildUrl = (path: string) => buildApiUrl(path, { baseUrl: API_BASE });

interface GenerateDesktopButtonProps {
  scenario: ScenarioDesktopStatus;
}

export function GenerateDesktopButton({ scenario }: GenerateDesktopButtonProps) {
  const queryClient = useQueryClient();
  const [buildId, setBuildId] = useState<string | null>(null);
  const [showConfigurator, setShowConfigurator] = useState(!scenario.has_desktop);
  const saved = scenario.connection_config;
  const [deploymentMode, setDeploymentMode] = useState(saved?.deployment_mode ?? "external-server");
  const [proxyUrl, setProxyUrl] = useState(saved?.proxy_url ?? saved?.server_url ?? "");
  const [autoManageVrooli, setAutoManageVrooli] = useState(saved?.auto_manage_vrooli ?? false);
  const [vrooliBinaryPath, setVrooliBinaryPath] = useState(saved?.vrooli_binary_path ?? "vrooli");
  const [bundleManifestPath, setBundleManifestPath] = useState(saved?.bundle_manifest_path ?? "");
  const [connectionResult, setConnectionResult] = useState<ProbeResponse | null>(null);
  const [connectionError, setConnectionError] = useState<string | null>(null);
  const selectedDeployment = useMemo(
    () => DEPLOYMENT_OPTIONS.find((option) => option.value === deploymentMode) ?? DEPLOYMENT_OPTIONS[0],
    [deploymentMode]
  );

  const generateMutation = useMutation({
    mutationFn: async () => {
      if (deploymentMode === "external-server" && !proxyUrl) {
        throw new Error("Provide the proxy URL you use in the browser.");
      }
      if (deploymentMode === "bundled" && !bundleManifestPath) {
        throw new Error("Provide the bundle_manifest_path exported by deployment-manager.");
      }
      const res = await fetch(buildUrl('/desktop/generate/quick'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          scenario_name: scenario.name,
          template_type: 'universal',
          deployment_mode: deploymentMode,
          proxy_url: proxyUrl,
          bundle_manifest_path: bundleManifestPath || undefined,
          auto_manage_vrooli: autoManageVrooli,
          vrooli_binary_path: vrooliBinaryPath
        })
      });
      if (!res.ok) {
        const contentType = res.headers.get('content-type');
        let errorMessage = 'Failed to generate desktop app';

        if (contentType?.includes('application/json')) {
          const errorData = await res.json();
          errorMessage = errorData.message || errorData.error || errorMessage;
        } else {
          const textError = await res.text();
          errorMessage = textError || errorMessage;
        }

        if (res.status === 404) {
          throw new Error(`Scenario '${scenario.name}' not found. Check that the scenario exists in the scenarios directory.`);
        } else if (res.status === 400) {
          throw new Error(`Invalid request: ${errorMessage}`);
        } else if (res.status === 500) {
          throw new Error(`Server error: ${errorMessage}. Check API logs for details.`);
        }
        throw new Error(`${errorMessage} (HTTP ${res.status})`);
      }
      return res.json();
    },
    onSuccess: (data) => {
      setBuildId(data.build_id);
      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: ['scenarios-desktop-status'] });
      }, 3000);
    }
  });

  useEffect(() => {
    if (scenario.has_desktop) {
      setShowConfigurator(false);
    } else {
      setShowConfigurator(true);
    }
  }, [scenario.has_desktop]);

  useEffect(() => {
    if (!scenario.connection_config) {
      return;
    }
    const cfg = scenario.connection_config;
    setDeploymentMode(cfg.deployment_mode ?? "external-server");
    setProxyUrl(cfg.proxy_url ?? cfg.server_url ?? "");
    setAutoManageVrooli(cfg.auto_manage_vrooli ?? false);
    setVrooliBinaryPath(cfg.vrooli_binary_path ?? "vrooli");
    setBundleManifestPath(cfg.bundle_manifest_path ?? "");
  }, [scenario.connection_config?.updated_at, scenario.connection_config, scenario.name]);

  const connectionMutation = useMutation({
    mutationFn: async () => {
      if (!proxyUrl) {
        throw new Error("Enter the proxy URL first");
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

  const { data: buildStatus } = useQuery({
    queryKey: ['build-status', buildId],
    queryFn: async () => {
      if (!buildId) return null;
      const res = await fetch(buildUrl(`/desktop/status/${buildId}`));
      if (!res.ok) throw new Error('Failed to fetch build status');
      return res.json();
    },
    enabled: !!buildId,
    refetchInterval: (data) => {
      if (data?.status === 'ready' || data?.status === 'failed') {
        return false;
      }
      return 2000;
    }
  });

  const isBuilding = generateMutation.isPending || buildStatus?.status === 'building';
  const isComplete = buildStatus?.status === 'ready';
  const isFailed = buildStatus?.status === 'failed' || generateMutation.isError;

  const defaultSummary = scenario.connection_config;

  return (
    <div className="space-y-3">
      {isComplete && (
        <div className="rounded-lg border border-green-900 bg-green-950/30 p-3 text-xs text-green-200">
          <div className="flex items-center gap-2">
            <Badge variant="success" className="gap-1">
              <CheckCircle className="h-3 w-3" /> Ready
            </Badge>
            <span>Files live at {buildStatus?.output_path || scenario.desktop_path}</span>
          </div>
          <p className="mt-1 text-[11px]">We’ll keep polling so the Scenario Inventory stays up to date.</p>
        </div>
      )}

      {isBuilding && (
        <div className="flex items-center gap-2 text-sm text-blue-300">
          <Loader2 className="h-4 w-4 animate-spin" /> Generating desktop wrapper...
        </div>
      )}

      {isFailed && (
        <ErrorCallout
          scenarioName={scenario.name}
          errorMessage={buildStatus?.error_log?.join('\n\n') || generateMutation.error?.message || 'Unknown error'}
          onRetry={() => {
            setBuildId(null);
            generateMutation.reset();
          }}
        />
      )}

      {!showConfigurator && scenario.has_desktop && defaultSummary ? (
        <div className="rounded-lg border border-slate-700 bg-slate-900/40 p-4 text-sm text-slate-200 space-y-2">
          <p className="font-semibold">Currently targeting</p>
          <p className="text-xs text-slate-400">
            {defaultSummary.proxy_url || defaultSummary.server_url || 'No proxy URL saved yet. Click edit to add it.'}
          </p>
          {defaultSummary.deployment_mode && (
            <p className="text-xs text-slate-500">Mode: {defaultSummary.deployment_mode}</p>
          )}
          <div className="flex flex-wrap gap-2 pt-2">
            <Button
              size="sm"
              className="gap-2"
              disabled={isBuilding}
              onClick={() => generateMutation.mutate()}
            >
              <Zap className="h-4 w-4" /> Regenerate wrapper
            </Button>
            <Button variant="outline" size="sm" onClick={() => setShowConfigurator(true)}>
              Edit connection
            </Button>
          </div>
        </div>
      ) : (
        <form
          className="space-y-3 rounded-lg border border-slate-700 bg-slate-900/40 p-4"
          onSubmit={(e) => {
            e.preventDefault();
            generateMutation.mutate();
          }}
        >
          <div className="flex items-center justify-between">
            <p className="text-sm text-slate-300">
              Connect this desktop wrapper to the same Vrooli instance you already open in the browser (local machine or Cloudflare/app-monitor link).
            </p>
            {scenario.has_desktop && (
              <Button variant="outline" type="button" size="sm" onClick={() => setShowConfigurator(false)}>
                Close
              </Button>
            )}
          </div>

          <div>
            <Label htmlFor={`deploymentMode-${scenario.name}`}>Deployment intent</Label>
            <Select
              id={`deploymentMode-${scenario.name}`}
              value={deploymentMode}
              onChange={(e) => setDeploymentMode(e.target.value)}
              className="mt-1"
            >
              {DEPLOYMENT_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </Select>
            <p className="mt-1 text-xs text-slate-400">
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

          {deploymentMode === "bundled" ? (
            <div className="space-y-2">
              <Label htmlFor={`bundleManifest-${scenario.name}`}>bundle_manifest_path</Label>
              <Input
                id={`bundleManifest-${scenario.name}`}
                value={bundleManifestPath}
                onChange={(e) => setBundleManifestPath(e.target.value)}
                placeholder="/home/you/Vrooli/docs/deployment/examples/manifests/desktop-happy.json"
              />
              <p className="text-xs text-emerald-200/80">
                Stages the manifest + bundled binaries into the desktop app so the packaged runtime can start offline.
              </p>
            </div>
          ) : (
            <>
              <div>
                <Label htmlFor={`proxyUrl-${scenario.name}`}>Proxy URL</Label>
                <p className="mb-1 text-xs text-slate-400">
                  Paste the Cloudflare/app-monitor link you already use (for example <code>https://app-monitor.example.com/apps/{scenario.name}/proxy/</code>).
                </p>
                <Input
                  id={`proxyUrl-${scenario.name}`}
                  value={proxyUrl}
                  onChange={(e) => setProxyUrl(e.target.value)}
                  placeholder="https://app-monitor.example.dev/apps/picker-wheel/proxy/"
                  className="mt-1"
                />

                <div className="mt-2 flex items-center gap-2">
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
                    <span className="text-xs text-green-300">Proxy responded ✔</span>
                  )}
                  {connectionError && <span className="text-xs text-red-300">{connectionError}</span>}
                </div>
                {(connectionResult?.server || connectionResult?.api) && (
                  <div className="mt-2 space-y-1 rounded border border-slate-800 bg-black/20 p-2 text-xs text-slate-200">
                    <p className="font-semibold text-slate-100">Connectivity snapshot</p>
                    <p>UI URL: {connectionResult?.server.status === "ok" ? "reachable" : connectionResult?.server.message || "no response"}</p>
                    <p>API URL: {connectionResult?.api.status === "ok" ? "reachable" : connectionResult?.api.message || "no response"}</p>
                  </div>
                )}
              </div>

              <div className="space-y-2">
                <Checkbox
                  checked={autoManageVrooli}
                  onChange={(e) => setAutoManageVrooli(e.target.checked)}
                  label="Let the desktop build run the scenario locally (vrooli setup/start)"
                />
                <Input
                  value={vrooliBinaryPath}
                  onChange={(e) => setVrooliBinaryPath(e.target.value)}
                  disabled={!autoManageVrooli}
                  placeholder="vrooli"
                />
                <p className="text-xs text-slate-400">
                  This runs `vrooli setup/start/stop` on the user's machine. Enable only when they expect to host the scenario locally.
                </p>
              </div>
            </>
          )}

          <div className="flex flex-wrap items-center gap-3">
            <Button type="submit" className="gap-2" disabled={generateMutation.isPending}>
              {generateMutation.isPending ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" /> Generating...
                </>
              ) : (
                <>
                  <Zap className="h-4 w-4" /> Generate Desktop Wrapper
                </>
              )}
            </Button>
            <Button
              variant="outline"
              type="button"
              onClick={() => {
                setProxyUrl("");
                setDeploymentMode("external-server");
                setAutoManageVrooli(false);
                setConnectionResult(null);
                setConnectionError(null);
                setVrooliBinaryPath("vrooli");
                setBundleManifestPath("");
              }}
            >
              Reset
            </Button>
          </div>
        </form>
      )}
    </div>
  );
}

const DEPLOYMENT_OPTIONS = [
  {
    value: "external-server",
    label: "Connect to existing Vrooli instance (UI-only)",
    description:
      "Reuse the URL you already open in the browser (local or Cloudflare/app-monitor). The desktop shell streams against that server.",
    docs: "https://github.com/vrooli/vrooli/blob/main/docs/deployment/tiers/tier-2-desktop.md"
  },
  {
    value: "cloud-api",
    label: "Package remote API (coming soon)",
    description: "Future option: hand deployment-manager a cloud target and connect this desktop app to it.",
    docs: "https://github.com/vrooli/vrooli/blob/main/docs/deployment/tiers/tier-4-saas.md"
  },
  {
    value: "bundled",
    label: "Offline bundle (bundle.json)",
    description: "Ship APIs/resources next to the UI using a bundle.json manifest so everything runs locally.",
    docs: "https://github.com/vrooli/vrooli/blob/main/docs/deployment/tiers/tier-2-desktop.md"
  }
];

function ErrorCallout({
  scenarioName,
  errorMessage,
  onRetry
}: {
  scenarioName: string;
  errorMessage: string;
  onRetry: () => void;
}) {
  const suggestion = (() => {
    if (errorMessage.includes('not found') || errorMessage.includes('404')) {
      return `Ensure the scenario '${scenarioName}' exists in /scenarios/ first.`;
    }
    if (errorMessage.includes('ui/dist') || errorMessage.includes('UI not built')) {
      return `Build the scenario UI first: cd scenarios/${scenarioName}/ui && npm run build.`;
    }
    if (errorMessage.includes('permission') || errorMessage.includes('EACCES')) {
      return 'Check file permissions in the scenarios directory.';
    }
    if (errorMessage.includes('ENOSPC') || errorMessage.includes('no space')) {
      return 'Free up disk space and try again.';
    }
    if (errorMessage.includes('port') || errorMessage.includes('EADDRINUSE')) {
      return 'Another process is using the required port. Stop it or change ports.';
    }
    return null;
  })();

  return (
    <div className="space-y-2 rounded-lg border border-red-900 bg-red-950/20 p-3 text-xs text-red-200">
      <div className="flex items-center gap-2 text-red-300">
        <XCircle className="h-3 w-3" /> Unable to generate desktop wrapper
      </div>
      <pre className="max-h-32 overflow-y-auto whitespace-pre-wrap font-mono text-[11px] text-red-200/80">
        {errorMessage}
      </pre>
      {suggestion && <p className="text-yellow-200">{suggestion}</p>}
      <div className="flex gap-2">
        <Button
          variant="outline"
          size="sm"
          onClick={() => navigator.clipboard.writeText(errorMessage)}
          className="gap-1"
        >
          Copy error
        </Button>
        <Button variant="outline" size="sm" onClick={onRetry}>
          Retry
        </Button>
      </div>
    </div>
  );
}
