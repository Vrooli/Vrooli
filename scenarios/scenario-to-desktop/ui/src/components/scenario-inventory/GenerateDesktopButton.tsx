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
import { BuildDesktopButton } from "./BuildDesktopButton";
import { TelemetryUploadCard } from "./TelemetryUploadCard";

const API_BASE = resolveApiBase({ appendSuffix: true });
const buildUrl = (path: string) => buildApiUrl(path, { baseUrl: API_BASE });

interface GenerateDesktopButtonProps {
  scenario: ScenarioDesktopStatus;
}

export function GenerateDesktopButton({ scenario }: GenerateDesktopButtonProps) {
  const queryClient = useQueryClient();
  const [buildId, setBuildId] = useState<string | null>(null);
  const [showConfigurator, setShowConfigurator] = useState(false);
  const saved = scenario.connection_config;
  const [deploymentMode, setDeploymentMode] = useState(saved?.deployment_mode ?? "external-server");
  const [proxyUrl, setProxyUrl] = useState(saved?.proxy_url ?? saved?.server_url ?? "");
  const [autoManageVrooli, setAutoManageVrooli] = useState(saved?.auto_manage_vrooli ?? false);
  const [vrooliBinaryPath, setVrooliBinaryPath] = useState(saved?.vrooli_binary_path ?? "vrooli");
  const [connectionResult, setConnectionResult] = useState<ProbeResponse | null>(null);
  const [connectionError, setConnectionError] = useState<string | null>(null);
  const selectedDeployment = useMemo(
    () => DEPLOYMENT_OPTIONS.find((option) => option.value === deploymentMode) ?? DEPLOYMENT_OPTIONS[0],
    [deploymentMode]
  );

  const generateMutation = useMutation({
    mutationFn: async () => {
      if (deploymentMode !== "external-server") {
        throw new Error("Only the 'Connect to existing Vrooli instance' option works right now.");
      }
      if (!proxyUrl) {
        throw new Error("Provide the proxy URL you use in the browser.");
      }
      const res = await fetch(buildUrl('/desktop/generate/quick'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          scenario_name: scenario.name,
          template_type: 'universal',
          deployment_mode: deploymentMode,
          proxy_url: proxyUrl,
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

        // Add status code context
        if (res.status === 404) {
          throw new Error(`Scenario '${scenario.name}' not found. Check that the scenario exists in the scenarios directory.`);
        } else if (res.status === 400) {
          throw new Error(`Invalid request: ${errorMessage}`);
        } else if (res.status === 500) {
          throw new Error(`Server error: ${errorMessage}. Check API logs for details.`);
        } else {
          throw new Error(`${errorMessage} (HTTP ${res.status})`);
        }
      }
      return res.json();
    },
    onSuccess: (data) => {
      setBuildId(data.build_id);
      // Refresh scenarios list to show the new desktop version
      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: ['scenarios-desktop-status'] });
      }, 3000);
    }
  });

  useEffect(() => {
    if (!scenario.connection_config) {
      return;
    }
    const cfg = scenario.connection_config;
    setDeploymentMode(cfg.deployment_mode ?? "external-server");
    setProxyUrl(cfg.proxy_url ?? cfg.server_url ?? "");
    setAutoManageVrooli(cfg.auto_manage_vrooli ?? false);
    setVrooliBinaryPath(cfg.vrooli_binary_path ?? "vrooli");
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

  // Poll build status if we have a buildId
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
      // Stop polling if build is complete or failed
      if (data?.status === 'ready' || data?.status === 'failed') {
        return false;
      }
      return 2000; // Poll every 2 seconds while building
    }
  });

  const isBuilding = generateMutation.isPending || buildStatus?.status === 'building';
  const isComplete = buildStatus?.status === 'ready';
  const isFailed = buildStatus?.status === 'failed' || generateMutation.isError;

  if (isComplete) {
    return (
      <div className="ml-4 w-full max-w-xl space-y-4 text-slate-200">
        <div className="rounded-lg border border-slate-700 bg-slate-900/40 p-4 space-y-2">
          <div className="flex items-center gap-2">
            <Badge variant="success" className="gap-1">
              <CheckCircle className="h-3 w-3" />
              Desktop wrapper ready
            </Badge>
            <span className="text-xs text-slate-400">Files live at {buildStatus.output_path}</span>
          </div>
          <p className="text-xs text-slate-400">
            Use the controls below to build installers and upload telemetry—no terminal required.
          </p>
        </div>
        <div className="rounded-lg border border-slate-700 bg-slate-900/40 p-4 space-y-3">
          <p className="text-sm font-semibold text-slate-100">Build installers</p>
          <p className="text-xs text-slate-400">
            We'll run <code>npm install</code>, <code>npm run build</code>, and <code>npm run dist</code> for the platforms you select.
          </p>
          <BuildDesktopButton scenarioName={scenario.name} />
        </div>
        <TelemetryUploadCard scenarioName={scenario.name} />
      </div>
    );
  }

  if (isFailed) {
    const errorMessage = buildStatus?.error_log?.join('\n\n') || generateMutation.error?.message || 'Unknown error';

    // Parse error to provide actionable suggestions
    const getSuggestion = (error: string): string | null => {
      if (error.includes('not found') || error.includes('404')) {
        return `💡 Ensure the scenario '${scenario.name}' exists in /scenarios/ directory`;
      }
      if (error.includes('ui/dist') || error.includes('UI not built')) {
        return `💡 Build the scenario UI first: cd scenarios/${scenario.name}/ui && npm run build`;
      }
      if (error.includes('permission') || error.includes('EACCES')) {
        return '💡 Check file permissions in the scenarios directory';
      }
      if (error.includes('ENOSPC') || error.includes('no space')) {
        return '💡 Free up disk space and try again';
      }
      if (error.includes('port') || error.includes('EADDRINUSE')) {
        return '💡 Another process is using the required port. Stop it or change ports.';
      }
      return null;
    };

    const suggestion = getSuggestion(errorMessage);

    const copyError = () => {
      navigator.clipboard.writeText(errorMessage);
    };

    return (
      <div className="ml-4 flex flex-col items-end gap-2 max-w-md">
        <Badge variant="destructive" className="gap-1">
          <XCircle className="h-3 w-3" />
          Failed
        </Badge>
        <div className="w-full rounded border border-red-900 bg-red-950/20 p-2">
          <p className="text-xs text-red-300 font-mono whitespace-pre-wrap max-h-32 overflow-y-auto">
            {errorMessage}
          </p>
          {suggestion && (
            <p className="mt-2 text-xs text-yellow-300 border-t border-red-800 pt-2">
              {suggestion}
            </p>
          )}
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={copyError}
            className="gap-1"
          >
            <svg className="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
            </svg>
            Copy Error
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              setBuildId(null);
              generateMutation.reset();
            }}
          >
            Retry
          </Button>
        </div>
      </div>
    );
  }

  if (isBuilding) {
    return (
      <div className="ml-4 flex items-center gap-2">
        <Loader2 className="h-4 w-4 animate-spin text-blue-400" />
        <span className="text-sm text-slate-400">Generating...</span>
      </div>
    );
  }

  return (
    <div className="ml-4 w-full max-w-xl">
      {!showConfigurator ? (
        <Button
          variant="outline"
          size="sm"
          className="gap-2"
          onClick={() => setShowConfigurator(true)}
        >
          <Zap className="h-4 w-4" />
          {scenario.has_desktop ? "Update Desktop Config" : "Configure Desktop Build"}
        </Button>
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
              Connect this desktop wrapper to the same Vrooli instance you already open in the browser (local machine or Cloudflare/app-monitor link). Paste that URL so the desktop shell can reuse it.
            </p>
            <Button variant="outline" type="button" size="sm" onClick={() => setShowConfigurator(false)}>
              Close
            </Button>
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
                <option key={option.value} value={option.value} disabled={option.value !== "external-server"}>
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

          <div>
            <Label htmlFor={`proxyUrl-${scenario.name}`}>Proxy URL</Label>
            <p className="text-xs text-slate-400 mb-1">
              Paste the Cloudflare/app-monitor link you already use (for example <code>https://app-monitor.example.com/apps/{scenario.name}/proxy/</code>). Desktop apps simply open this URL.
            </p>
            <Input
              id={`proxyUrl-${scenario.name}`}
              value={proxyUrl}
              onChange={(e) => setProxyUrl(e.target.value)}
              placeholder="https://app-monitor.example.dev/apps/picker-wheel/proxy/"
              className="mt-1"
            />

            <div className="flex items-center gap-2 mt-2">
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
              <div className="mt-2 rounded border border-slate-800 bg-black/20 p-2 text-xs text-slate-200 space-y-1">
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

          <div className="flex items-center gap-3">
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
    label: "Offline bundle (coming soon)",
    description: "Future option: ship APIs/resources next to the UI so everything runs on the desktop.",
    docs: "https://github.com/vrooli/vrooli/blob/main/docs/deployment/tiers/tier-2-desktop.md"
  }
];
