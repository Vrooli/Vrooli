import { useCallback, useEffect, useMemo, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Button } from "../ui/button";
import { Badge } from "../ui/badge";
import { Input } from "../ui/input";
import { Label } from "../ui/label";
import { Select } from "../ui/select";
import { Checkbox } from "../ui/checkbox";
import { Loader2, Zap, CheckCircle } from "lucide-react";
import {
  probeEndpoints,
  type ProbeResponse,
  type PipelineConfig,
  type GenerateStageDetails,
} from "../../lib/api";
import { usePipelineMutation, usePipelineStatus } from "../../hooks";
import { PipelineErrorDisplay, suggestRecovery } from "../pipeline";
import {
  deploymentModeFromFormValue,
  templateTypeFromFormValue,
} from "../../lib/pipeline-enums";
import { StageName } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";
import type { DesktopConnectionConfig, ScenarioDesktopStatus } from "./types";
import {
  DEFAULT_DEPLOYMENT_MODE,
  DEFAULT_SERVER_TYPE,
  DEPLOYMENT_OPTIONS,
  decideConnection,
  findDeploymentOption,
  type ConnectionDecision,
  type DeploymentMode,
} from "../../domain/deployment";

interface GenerateDesktopButtonProps {
  scenario: ScenarioDesktopStatus;
}

type ConnectionDefaults = {
  deploymentMode: DeploymentMode;
  proxyUrl: string;
  autoManageVrooli: boolean;
  vrooliBinaryPath: string;
  bundleManifestPath: string;
};

function buildConnectionDefaults(
  config?: DesktopConnectionConfig | null,
): ConnectionDefaults {
  return {
    deploymentMode: config
      ? (config.deployment_mode as DeploymentMode)
      : DEFAULT_DEPLOYMENT_MODE,
    proxyUrl: config?.proxy_url ?? config?.server_url ?? "",
    autoManageVrooli: config?.auto_manage_vrooli ?? false,
    vrooliBinaryPath: config?.vrooli_binary_path ?? "vrooli",
    bundleManifestPath: config?.bundle_manifest_path ?? "",
  };
}

function ensureRequiredInputs(
  decision: ConnectionDecision,
  proxyUrl: string,
  bundleManifestPath: string,
  messages: { proxy: string; bundle: string },
) {
  if (decision.requiresProxyUrl && !proxyUrl) {
    throw new Error(messages.proxy);
  }
  if (decision.requiresBundleManifest && !bundleManifestPath) {
    throw new Error(messages.bundle);
  }
}

export function GenerateDesktopButton({
  scenario,
}: GenerateDesktopButtonProps) {
  const [showConfigurator, setShowConfigurator] = useState(
    !scenario.has_desktop,
  );
  const saved = scenario.connection_config;
  const defaults = buildConnectionDefaults(saved);
  const [deploymentMode, setDeploymentMode] = useState<DeploymentMode>(
    defaults.deploymentMode,
  );
  const [proxyUrl, setProxyUrl] = useState(defaults.proxyUrl);
  const [autoManageVrooli, setAutoManageVrooli] = useState(
    defaults.autoManageVrooli,
  );
  const [vrooliBinaryPath, setVrooliBinaryPath] = useState(
    defaults.vrooliBinaryPath,
  );
  const [bundleManifestPath, setBundleManifestPath] = useState(
    defaults.bundleManifestPath,
  );
  const [connectionResult, setConnectionResult] =
    useState<ProbeResponse | null>(null);
  const [connectionError, setConnectionError] = useState<string | null>(null);

  const selectedDeployment = useMemo(
    () => findDeploymentOption(deploymentMode),
    [deploymentMode],
  );
  const serverType = DEFAULT_SERVER_TYPE;
  const connectionDecision = useMemo(
    () => decideConnection(deploymentMode, serverType),
    [deploymentMode, serverType],
  );

  const applyConnectionConfig = useCallback(
    (config?: DesktopConnectionConfig | null) => {
      const next = buildConnectionDefaults(config);
      setDeploymentMode(next.deploymentMode);
      setProxyUrl(next.proxyUrl);
      setAutoManageVrooli(next.autoManageVrooli);
      setVrooliBinaryPath(next.vrooliBinaryPath);
      setBundleManifestPath(next.bundleManifestPath);
    },
    [],
  );

  // Use shared pipeline mutation hook
  const {
    state: { buildId, error: mutationError },
    mutation: generateMutation,
    runPipelineWithConfig,
    reset,
  } = usePipelineMutation({
    invalidateOnSuccess: ["scenarios-desktop-status"],
  });

  const handleGenerate = useCallback(() => {
    ensureRequiredInputs(connectionDecision, proxyUrl, bundleManifestPath, {
      proxy: "Provide the proxy URL you use in the browser.",
      bundle:
        "Provide the bundle_manifest_path exported by deployment-manager.",
    });
    const config: PipelineConfig = {
      scenarioName: scenario.name,
      templateType: templateTypeFromFormValue("universal"),
      deploymentMode: deploymentModeFromFormValue(deploymentMode),
      proxyUrl: proxyUrl || undefined,
      bundleManifestPath: bundleManifestPath || undefined,
      stopAfterStage: StageName.GENERATE,
    };
    runPipelineWithConfig(config);
  }, [
    connectionDecision,
    proxyUrl,
    bundleManifestPath,
    scenario.name,
    deploymentMode,
    runPipelineWithConfig,
  ]);

  useEffect(() => {
    setShowConfigurator(!scenario.has_desktop);
  }, [scenario.has_desktop]);

  useEffect(() => {
    if (scenario.connection_config) {
      applyConnectionConfig(scenario.connection_config);
    }
  }, [applyConnectionConfig, scenario.connection_config]);

  const connectionMutation = useMutation({
    mutationFn: async () => {
      ensureRequiredInputs(connectionDecision, proxyUrl, bundleManifestPath, {
        proxy: "Enter the proxy URL first",
        bundle:
          "Provide the bundle_manifest_path exported by deployment-manager.",
      });
      return probeEndpoints({ proxy_url: proxyUrl });
    },
    onSuccess: (result) => {
      setConnectionResult(result);
      setConnectionError(null);
    },
    onError: (error: Error) => {
      setConnectionError(error.message);
      setConnectionResult(null);
    },
  });

  const {
    pipelineStatus,
    isBuilding: statusIsBuilding,
    isComplete,
    isFailed,
  } = usePipelineStatus({
    buildId,
    verbose: true,
    queryKeyPrefix: "generate-status",
  });

  const buildStatus = pipelineStatus
    ? {
        status: isComplete ? "ready" : isFailed ? "failed" : "building",
        output_path: (
          pipelineStatus.stages.generate?.details as
            | GenerateStageDetails
            | undefined
        )?.desktopPath,
        // logs only available on VerboseStageResult, use type guard
        error_log:
          pipelineStatus.stages.generate &&
          "logs" in pipelineStatus.stages.generate
            ? (pipelineStatus.stages.generate as { logs?: string[] }).logs
            : undefined,
      }
    : null;

  const isBuilding = generateMutation.isPending || statusIsBuilding;
  const errorMessage =
    buildStatus?.error_log?.join("\n\n") ||
    mutationError ||
    generateMutation.error?.message;
  const showError = isFailed || generateMutation.isError;

  const defaultSummary = scenario.connection_config;

  return (
    <div className="space-y-3">
      {isComplete && (
        <div className="rounded-lg border border-green-900 bg-green-950/30 p-3 text-xs text-green-200">
          <div className="flex items-center gap-2">
            <Badge variant="success" className="gap-1">
              <CheckCircle className="h-3 w-3" /> Ready
            </Badge>
            <span>
              Files live at {buildStatus?.output_path || scenario.desktop_path}
            </span>
          </div>
          <p className="mt-1 text-[11px]">
            We'll keep polling so the Scenario Inventory stays up to date.
          </p>
        </div>
      )}

      {isBuilding && (
        <div className="flex items-center gap-2 text-sm text-blue-300">
          <Loader2 className="h-4 w-4 animate-spin" /> Generating desktop
          wrapper...
        </div>
      )}

      {showError && errorMessage && (
        <PipelineErrorDisplay
          title="Unable to generate desktop wrapper"
          errorMessage={errorMessage}
          suggestion={suggestRecovery(errorMessage, scenario.name)}
          onRetry={reset}
        />
      )}

      {!showConfigurator && scenario.has_desktop && defaultSummary ? (
        <div className="rounded-lg border border-slate-700 bg-slate-900/40 p-4 text-sm text-slate-200 space-y-2">
          <p className="font-semibold">Currently targeting</p>
          <p className="text-xs text-slate-400">
            {defaultSummary.proxy_url ||
              defaultSummary.server_url ||
              "No proxy URL saved yet. Click edit to add it."}
          </p>
          {defaultSummary.deployment_mode && (
            <p className="text-xs text-slate-500">
              Mode: {defaultSummary.deployment_mode}
            </p>
          )}
          <div className="flex flex-wrap gap-2 pt-2">
            <Button
              size="sm"
              className="gap-2"
              disabled={isBuilding}
              onClick={handleGenerate}
            >
              <Zap className="h-4 w-4" /> Regenerate wrapper
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setShowConfigurator(true);
              }}
            >
              Edit connection
            </Button>
          </div>
        </div>
      ) : (
        <form
          className="space-y-3 rounded-lg border border-slate-700 bg-slate-900/40 p-4"
          onSubmit={(e) => {
            e.preventDefault();
            handleGenerate();
          }}
        >
          <div className="flex items-center justify-between">
            <p className="text-sm text-slate-300">
              Connect this desktop wrapper to the same Vrooli instance you
              already open in the browser (local machine or
              Cloudflare/app-monitor link).
            </p>
            {scenario.has_desktop && (
              <Button
                variant="outline"
                type="button"
                size="sm"
                onClick={() => {
                  setShowConfigurator(false);
                }}
              >
                Close
              </Button>
            )}
          </div>

          <div>
            <Label htmlFor={`deploymentMode-${scenario.name}`}>
              Deployment intent
            </Label>
            <Select
              id={`deploymentMode-${scenario.name}`}
              value={deploymentMode}
              onChange={(e) => {
                setDeploymentMode(e.target.value as DeploymentMode);
              }}
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

          {connectionDecision.kind === "bundled-runtime" ? (
            <BundleManifestField
              scenarioName={scenario.name}
              bundleManifestPath={bundleManifestPath}
              onChange={setBundleManifestPath}
            />
          ) : (
            <RemoteConnectionSection
              scenarioName={scenario.name}
              proxyUrl={proxyUrl}
              onProxyUrlChange={setProxyUrl}
              onTestConnection={() => {
                connectionMutation.mutate();
              }}
              isTesting={connectionMutation.isPending}
              connectionResult={connectionResult}
              connectionError={connectionError}
              autoManageVrooli={autoManageVrooli}
              onToggleAutoManageVrooli={setAutoManageVrooli}
              vrooliBinaryPath={vrooliBinaryPath}
              onBinaryPathChange={setVrooliBinaryPath}
            />
          )}

          <div className="flex flex-wrap items-center gap-3">
            <Button
              type="submit"
              className="gap-2"
              disabled={generateMutation.isPending}
            >
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
                applyConnectionConfig(null);
                setConnectionResult(null);
                setConnectionError(null);
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

type BundleManifestFieldProps = {
  scenarioName: string;
  bundleManifestPath: string;
  onChange: (value: string) => void;
};

function BundleManifestField({
  scenarioName,
  bundleManifestPath,
  onChange,
}: BundleManifestFieldProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor={`bundleManifest-${scenarioName}`}>
        bundle_manifest_path
      </Label>
      <Input
        id={`bundleManifest-${scenarioName}`}
        value={bundleManifestPath}
        onChange={(e) => {
          onChange(e.target.value);
        }}
        placeholder="/home/you/Vrooli/docs/deployment/examples/manifests/desktop-happy.json"
      />
      <p className="text-xs text-emerald-200/80">
        Stages the manifest + bundled binaries into the desktop app so the
        packaged runtime can start offline.
      </p>
    </div>
  );
}

type RemoteConnectionSectionProps = {
  scenarioName: string;
  proxyUrl: string;
  onProxyUrlChange: (value: string) => void;
  onTestConnection: () => void;
  isTesting: boolean;
  connectionResult: ProbeResponse | null;
  connectionError: string | null;
  autoManageVrooli: boolean;
  onToggleAutoManageVrooli: (value: boolean) => void;
  vrooliBinaryPath: string;
  onBinaryPathChange: (value: string) => void;
};

function RemoteConnectionSection({
  scenarioName,
  proxyUrl,
  onProxyUrlChange,
  onTestConnection,
  isTesting,
  connectionResult,
  connectionError,
  autoManageVrooli,
  onToggleAutoManageVrooli,
  vrooliBinaryPath,
  onBinaryPathChange,
}: RemoteConnectionSectionProps) {
  const serverResult = connectionResult?.server;
  const apiResult = connectionResult?.api;
  const bothEndpointsHealthy =
    serverResult?.status === "ok" && apiResult?.status === "ok";

  return (
    <>
      <div>
        <Label htmlFor={`proxyUrl-${scenarioName}`}>Proxy URL</Label>
        <p className="mb-1 text-xs text-slate-400">
          Paste the Cloudflare/app-monitor link you already use (for example{" "}
          <code>
            https://app-monitor.example.com/apps/{scenarioName}/proxy/
          </code>
          ).
        </p>
        <Input
          id={`proxyUrl-${scenarioName}`}
          value={proxyUrl}
          onChange={(e) => {
            onProxyUrlChange(e.target.value);
          }}
          placeholder="https://app-monitor.example.dev/apps/picker-wheel/proxy/"
          className="mt-1"
        />

        <div className="mt-2 flex items-center gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={onTestConnection}
            disabled={isTesting || !proxyUrl}
          >
            {isTesting ? "Testing..." : "Test connection"}
          </Button>
          {bothEndpointsHealthy && (
            <span className="text-xs text-green-300">Proxy responded ✔</span>
          )}
          {connectionError && (
            <span className="text-xs text-red-300">{connectionError}</span>
          )}
        </div>
        {(serverResult || apiResult) && (
          <div className="mt-2 space-y-1 rounded border border-slate-800 bg-black/20 p-2 text-xs text-slate-200">
            <p className="font-semibold text-slate-100">
              Connectivity snapshot
            </p>
            <p>
              UI URL:{" "}
              {serverResult?.status === "ok"
                ? "reachable"
                : serverResult?.message || "no response"}
            </p>
            <p>
              API URL:{" "}
              {apiResult?.status === "ok"
                ? "reachable"
                : apiResult?.message || "no response"}
            </p>
          </div>
        )}
      </div>

      <div className="space-y-2">
        <Checkbox
          checked={autoManageVrooli}
          onChange={(e) => {
            onToggleAutoManageVrooli(e.target.checked);
          }}
          label="Let the desktop build run the scenario locally (vrooli setup/start)"
        />
        <Input
          value={vrooliBinaryPath}
          onChange={(e) => {
            onBinaryPathChange(e.target.value);
          }}
          disabled={!autoManageVrooli}
          placeholder="vrooli"
        />
        <p className="text-xs text-slate-400">
          This runs `vrooli setup/start/stop` on the user's machine. Enable only
          when they expect to host the scenario locally.
        </p>
      </div>
    </>
  );
}
