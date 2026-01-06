import { useState } from "react";
import {
  ArrowLeft,
  Server,
  Globe,
  Clock,
  ExternalLink,
  RefreshCw,
  Play,
  Square,
  ChevronDown,
  ChevronRight,
  Terminal,
  CheckCircle2,
  XCircle,
  Loader2,
  AlertCircle,
  Activity,
  LayoutDashboard,
  FolderOpen,
  GitCompare,
  History,
  Search,
} from "lucide-react";
import {
  useDeployment,
  useInspectDeployment,
  useStopDeployment,
  useExecuteDeployment,
  getStatusInfo,
} from "../../hooks/useDeployments";
import { useDeploymentInvestigation } from "../../hooks/useInvestigation";
import { cn } from "../../lib/utils";
import type { Deployment } from "../../lib/api";
import { LiveStateTab, FilesTab, DriftTab, HistoryTab, InvestigationsTab, TerminalTab } from "./tabs";
import { CodeBlock } from "../ui/code-block";
import { InvestigateButton } from "../wizard/InvestigateButton";
import { InvestigationProgress } from "../wizard/InvestigationProgress";
import { InvestigationReport } from "../wizard/InvestigationReport";

interface DeploymentDetailsProps {
  deploymentId: string;
  onBack: () => void;
}

export function DeploymentDetails({ deploymentId, onBack }: DeploymentDetailsProps) {
  const { data: deployment, isLoading, error, refetch } = useDeployment(deploymentId);
  const inspectMutation = useInspectDeployment();
  const stopMutation = useStopDeployment();
  const executeMutation = useExecuteDeployment();

  const [showManifest, setShowManifest] = useState(false);
  const [showSetupResult, setShowSetupResult] = useState(false);
  const [showDeployResult, setShowDeployResult] = useState(false);
  const [showLogs, setShowLogs] = useState(true);
  const [activeTab, setActiveTab] = useState<"overview" | "live-state" | "files" | "drift" | "history" | "investigations" | "terminal">("overview");
  const [showInvestigationReport, setShowInvestigationReport] = useState(false);
  const [showRedeployDialog, setShowRedeployDialog] = useState(false);
  const [buildNewBundle, setBuildNewBundle] = useState(false);
  const [runPreflight, setRunPreflight] = useState(false);

  // Investigation state
  const investigation = useDeploymentInvestigation(deploymentId);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-blue-400" />
      </div>
    );
  }

  if (error || !deployment) {
    return (
      <div className="space-y-4">
        <button
          onClick={onBack}
          className="flex items-center gap-2 text-slate-400 hover:text-white transition-colors"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Deployments
        </button>
        <div className="p-4 rounded-lg border border-red-500/30 bg-red-500/10 text-red-400">
          <div className="flex items-center gap-2">
            <AlertCircle className="h-5 w-5" />
            <span>{error?.message || "Deployment not found"}</span>
          </div>
        </div>
      </div>
    );
  }

  const statusInfo = getStatusInfo(deployment.status);
  const manifest = deployment.manifest as {
    scenario?: { id: string };
    edge?: { domain: string };
    target?: { vps?: { host: string } };
    dependencies?: { resources?: string[]; scenarios?: string[] };
  };

  const isInProgress =
    deployment.status === "pending" ||
    deployment.status === "setup_running" ||
    deployment.status === "deploying";

  const canRedeploy =
    deployment.status === "deployed" ||
    deployment.status === "failed" ||
    deployment.status === "stopped";

  const canStop = deployment.status === "deployed";
  const hasExistingBundle = Boolean(deployment.bundle_path);
  const isInvestigationOutdated = (inv?: { deployment_run_id?: string; created_at: string } | null) => {
    if (!inv) return false;
    if (deployment.run_id && inv.deployment_run_id) {
      return inv.deployment_run_id !== deployment.run_id;
    }
    if (deployment.last_deployed_at) {
      return new Date(inv.created_at).getTime() < new Date(deployment.last_deployed_at).getTime();
    }
    return false;
  };

  const openRedeployDialog = () => {
    setBuildNewBundle(!hasExistingBundle);
    setRunPreflight(false);
    setShowRedeployDialog(true);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between gap-4">
        <div>
          <button
            onClick={onBack}
            className="flex items-center gap-2 text-slate-400 hover:text-white transition-colors mb-2"
          >
            <ArrowLeft className="h-4 w-4" />
            Back to Deployments
          </button>
          <h1 className="text-2xl font-bold text-white">{deployment.name}</h1>
          <div className="flex items-center gap-3 mt-2">
            <StatusBadgeLarge status={deployment.status} />
            {manifest.edge?.domain && deployment.status === "deployed" && (
              <a
                href={`https://${manifest.edge.domain}`}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-1.5 text-sm text-blue-400 hover:text-blue-300"
              >
                <ExternalLink className="h-4 w-4" />
                {manifest.edge.domain}
              </a>
            )}
          </div>
        </div>

        {/* Actions */}
        <div className="flex flex-col items-end gap-2">
          <div className="flex items-center gap-2">
            <button
              onClick={() => inspectMutation.mutate(deploymentId)}
              disabled={inspectMutation.isPending}
              className={cn(
                "flex items-center gap-2 px-4 py-2 rounded-lg border border-white/10",
                "hover:bg-white/5 transition-colors text-sm font-medium",
                inspectMutation.isPending && "opacity-50 cursor-not-allowed"
              )}
            >
              {inspectMutation.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <RefreshCw className="h-4 w-4" />
              )}
              Inspect
            </button>

            {canStop && (
              <button
                onClick={() => stopMutation.mutate(deploymentId)}
                disabled={stopMutation.isPending}
                className={cn(
                  "flex items-center gap-2 px-4 py-2 rounded-lg border border-amber-500/30",
                  "hover:bg-amber-500/10 text-amber-400 transition-colors text-sm font-medium",
                  stopMutation.isPending && "opacity-50 cursor-not-allowed"
                )}
              >
                {stopMutation.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Square className="h-4 w-4" />
                )}
                Stop
              </button>
            )}

            {canRedeploy && (
              <button
                onClick={openRedeployDialog}
                disabled={executeMutation.isPending}
                className={cn(
                  "flex items-center gap-2 px-4 py-2 rounded-lg bg-blue-500",
                  "hover:bg-blue-600 text-white transition-colors text-sm font-medium",
                  executeMutation.isPending && "opacity-50 cursor-not-allowed"
                )}
              >
                {executeMutation.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Play className="h-4 w-4" />
                )}
                {deployment.status === "failed" ? "Retry" : "Re-deploy"}
              </button>
            )}

            {/* Investigate button for failed deployments */}
            {deployment.status === "failed" && (
              <InvestigateButton
                deploymentId={deploymentId}
                onInvestigationStarted={() => {}}
              />
            )}
          </div>
        </div>
      </div>

      {showRedeployDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 px-4">
          <div className="w-full max-w-lg rounded-lg border border-white/10 bg-slate-900 p-6 shadow-xl">
            <div className="mb-4">
              <h3 className="text-lg font-semibold text-white">Re-deploy configuration</h3>
              <p className="text-sm text-slate-400 mt-1">
                Choose what will run. Deployment always executes.
              </p>
            </div>

            <div className="space-y-4">
              <div className="rounded-lg border border-white/10 bg-slate-900/60 p-4">
                <p className="text-xs uppercase tracking-wide text-slate-500 mb-3">Bundle</p>
                <SwitchRow
                  label="Build a new bundle"
                  description={
                    hasExistingBundle
                      ? "Creates a fresh bundle before setup."
                      : "No existing bundle found. A new bundle is required."
                  }
                  checked={buildNewBundle}
                  disabled={!hasExistingBundle}
                  onChange={() => setBuildNewBundle(!buildNewBundle)}
                />
              </div>

              <div className="rounded-lg border border-white/10 bg-slate-900/60 p-4">
                <p className="text-xs uppercase tracking-wide text-slate-500 mb-3">Checks</p>
                <SwitchRow
                  label="Run preflight checks"
                  description="Validates DNS, SSH, OS, and prerequisites before deployment."
                  checked={runPreflight}
                  onChange={() => setRunPreflight(!runPreflight)}
                />
              </div>

              <div className="rounded-lg border border-white/10 bg-slate-900/60 p-4">
                <p className="text-xs uppercase tracking-wide text-slate-500 mb-3">Deployment</p>
                <SwitchRow
                  label="Deploy to VPS"
                  description="Always runs. This step cannot be skipped."
                  checked={true}
                  disabled={true}
                  onChange={() => {}}
                />
              </div>
            </div>

            <div className="mt-5 rounded-lg border border-white/10 bg-slate-900/60 p-3">
              <p className="text-xs uppercase tracking-wide text-slate-500 mb-2">What will happen</p>
              <p className="text-sm text-slate-300">
                {runPreflight ? "Run preflight checks, then " : "Skip preflight checks, then "}
                {buildNewBundle ? "build a new bundle" : "reuse the existing bundle"}, then run VPS setup and deploy.
              </p>
            </div>

            <div className="mt-6 flex justify-end gap-3">
              <button
                onClick={() => setShowRedeployDialog(false)}
                disabled={executeMutation.isPending}
                className="px-4 py-2 rounded-lg font-medium text-slate-400 hover:text-white hover:bg-white/5 transition-colors disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                onClick={() =>
                  executeMutation.mutate(
                    {
                      id: deploymentId,
                      options: {
                        runPreflight,
                        forceBundleBuild: buildNewBundle,
                      },
                    },
                    { onSuccess: () => setShowRedeployDialog(false) }
                  )
                }
                disabled={executeMutation.isPending}
                className="px-4 py-2 rounded-lg font-medium bg-blue-500 hover:bg-blue-600 text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
              >
                {executeMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                {executeMutation.isPending ? "Starting..." : "Re-deploy"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Investigation progress - show when there's an active investigation */}
      {investigation.activeInvestigation && deployment.status === "failed" && (
        <InvestigationProgress
          investigation={investigation.activeInvestigation}
          isRunning={investigation.isRunning}
          onStop={investigation.stop}
          isStopping={investigation.isStopping}
          isOutdated={isInvestigationOutdated(investigation.activeInvestigation)}
          onViewReport={(invId) => {
            investigation.viewReport(invId);
            setShowInvestigationReport(true);
          }}
        />
      )}

      {/* Investigation report modal */}
      {showInvestigationReport && investigation.activeInvestigation && (
        <InvestigationReport
          investigation={investigation.activeInvestigation}
          onClose={() => setShowInvestigationReport(false)}
          isOutdated={isInvestigationOutdated(investigation.activeInvestigation)}
          onApplyFixes={async (invId, options) => {
            await investigation.applyFixes(invId, options);
            setShowInvestigationReport(false);
          }}
          isApplyingFixes={investigation.isApplyingFixes}
        />
      )}

      {/* Error message */}
      {deployment.error_message && (
        <div className="p-4 rounded-lg border border-red-500/30 bg-red-500/10">
          <div className="flex items-start gap-2">
            <XCircle className="h-5 w-5 text-red-400 mt-0.5" />
            <div>
              <p className="font-medium text-red-400">
                Failed at: {deployment.error_step || "unknown step"}
              </p>
              <p className="text-red-300 mt-1">{deployment.error_message}</p>
            </div>
          </div>
        </div>
      )}

      {/* Tab Navigation */}
      <div className="flex gap-1 border-b border-white/10 pb-px overflow-x-auto">
        <button
          onClick={() => setActiveTab("overview")}
          className={cn(
            "flex items-center gap-2 px-4 py-2 text-sm font-medium transition-colors rounded-t-lg whitespace-nowrap",
            activeTab === "overview"
              ? "bg-slate-800 text-white border-b-2 border-blue-500"
              : "text-slate-400 hover:text-white hover:bg-slate-800/50"
          )}
        >
          <LayoutDashboard className="h-4 w-4" />
          Overview
        </button>
        <button
          onClick={() => setActiveTab("live-state")}
          className={cn(
            "flex items-center gap-2 px-4 py-2 text-sm font-medium transition-colors rounded-t-lg whitespace-nowrap",
            activeTab === "live-state"
              ? "bg-slate-800 text-white border-b-2 border-blue-500"
              : "text-slate-400 hover:text-white hover:bg-slate-800/50"
          )}
        >
          <Activity className="h-4 w-4" />
          Live State
        </button>
        <button
          onClick={() => setActiveTab("files")}
          className={cn(
            "flex items-center gap-2 px-4 py-2 text-sm font-medium transition-colors rounded-t-lg whitespace-nowrap",
            activeTab === "files"
              ? "bg-slate-800 text-white border-b-2 border-blue-500"
              : "text-slate-400 hover:text-white hover:bg-slate-800/50"
          )}
        >
          <FolderOpen className="h-4 w-4" />
          Files
        </button>
        <button
          onClick={() => setActiveTab("drift")}
          className={cn(
            "flex items-center gap-2 px-4 py-2 text-sm font-medium transition-colors rounded-t-lg whitespace-nowrap",
            activeTab === "drift"
              ? "bg-slate-800 text-white border-b-2 border-blue-500"
              : "text-slate-400 hover:text-white hover:bg-slate-800/50"
          )}
        >
          <GitCompare className="h-4 w-4" />
          Drift
        </button>
        <button
          onClick={() => setActiveTab("history")}
          className={cn(
            "flex items-center gap-2 px-4 py-2 text-sm font-medium transition-colors rounded-t-lg whitespace-nowrap",
            activeTab === "history"
              ? "bg-slate-800 text-white border-b-2 border-blue-500"
              : "text-slate-400 hover:text-white hover:bg-slate-800/50"
          )}
        >
          <History className="h-4 w-4" />
          History
        </button>
        <button
          onClick={() => setActiveTab("investigations")}
          className={cn(
            "flex items-center gap-2 px-4 py-2 text-sm font-medium transition-colors rounded-t-lg whitespace-nowrap",
            activeTab === "investigations"
              ? "bg-slate-800 text-white border-b-2 border-blue-500"
              : "text-slate-400 hover:text-white hover:bg-slate-800/50"
          )}
        >
          <Search className="h-4 w-4" />
          Investigations
        </button>
        <button
          onClick={() => setActiveTab("terminal")}
          className={cn(
            "flex items-center gap-2 px-4 py-2 text-sm font-medium transition-colors rounded-t-lg whitespace-nowrap",
            activeTab === "terminal"
              ? "bg-slate-800 text-white border-b-2 border-blue-500"
              : "text-slate-400 hover:text-white hover:bg-slate-800/50"
          )}
        >
          <Terminal className="h-4 w-4" />
          Terminal
        </button>
      </div>

      {/* Tab Content */}
      {activeTab === "live-state" && (
        <LiveStateTab deploymentId={deploymentId} deploymentName={deployment.name} />
      )}
      {activeTab === "files" && (
        <FilesTab deploymentId={deploymentId} />
      )}
      {activeTab === "drift" && (
        <DriftTab deploymentId={deploymentId} />
      )}
      {activeTab === "history" && (
        <HistoryTab deploymentId={deploymentId} />
      )}
      {activeTab === "investigations" && (
        <InvestigationsTab
          deploymentId={deploymentId}
          deploymentRunId={deployment.run_id}
          lastDeployedAt={deployment.last_deployed_at}
          onViewReport={(inv) => {
            investigation.viewReport(inv.id);
            setShowInvestigationReport(true);
          }}
        />
      )}
      {activeTab === "terminal" && (
        <TerminalTab deploymentId={deploymentId} />
      )}
      {activeTab === "overview" && (
        <>
          {/* Info cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* Manifest summary */}
        <div className="border border-white/10 rounded-lg bg-slate-900/50 p-4">
          <h3 className="text-sm font-medium text-slate-400 mb-3">Deployment Info</h3>
          <div className="space-y-2 text-sm">
            <div className="flex items-center gap-2">
              <Server className="h-4 w-4 text-slate-500" />
              <span className="text-slate-400">Scenario:</span>
              <span className="text-white">{manifest.scenario?.id || deployment.scenario_id}</span>
            </div>
            {manifest.edge?.domain && (
              <div className="flex items-center gap-2">
                <Globe className="h-4 w-4 text-slate-500" />
                <span className="text-slate-400">Domain:</span>
                <span className="text-white">{manifest.edge.domain}</span>
              </div>
            )}
            {manifest.target?.vps?.host && (
              <div className="flex items-center gap-2">
                <Terminal className="h-4 w-4 text-slate-500" />
                <span className="text-slate-400">Host:</span>
                <span className="text-white">{manifest.target.vps.host}</span>
              </div>
            )}
          </div>
        </div>

        {/* Timestamps */}
        <div className="border border-white/10 rounded-lg bg-slate-900/50 p-4">
          <h3 className="text-sm font-medium text-slate-400 mb-3">Timeline</h3>
          <div className="space-y-2 text-sm">
            <div className="flex items-center gap-2">
              <Clock className="h-4 w-4 text-slate-500" />
              <span className="text-slate-400">Created:</span>
              <span className="text-white">
                {new Date(deployment.created_at).toLocaleString()}
              </span>
            </div>
            {deployment.last_deployed_at && (
              <div className="flex items-center gap-2">
                <CheckCircle2 className="h-4 w-4 text-emerald-500" />
                <span className="text-slate-400">Last deployed:</span>
                <span className="text-white">
                  {new Date(deployment.last_deployed_at).toLocaleString()}
                </span>
              </div>
            )}
            {deployment.last_inspected_at && (
              <div className="flex items-center gap-2">
                <RefreshCw className="h-4 w-4 text-blue-500" />
                <span className="text-slate-400">Last inspected:</span>
                <span className="text-white">
                  {new Date(deployment.last_inspected_at).toLocaleString()}
                </span>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Dependencies */}
      {(manifest.dependencies?.resources?.length || manifest.dependencies?.scenarios?.length) && (
        <div className="border border-white/10 rounded-lg bg-slate-900/50 p-4">
          <h3 className="text-sm font-medium text-slate-400 mb-3">Dependencies</h3>
          <div className="flex flex-wrap gap-2">
            {manifest.dependencies?.resources?.map((res) => (
              <span
                key={res}
                className="px-2 py-1 rounded bg-purple-500/20 text-purple-400 text-xs"
              >
                {res}
              </span>
            ))}
            {manifest.dependencies?.scenarios?.map((scen) => (
              <span
                key={scen}
                className="px-2 py-1 rounded bg-blue-500/20 text-blue-400 text-xs"
              >
                {scen}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Deployment Manifest */}
      {deployment.manifest && (
        <CollapsibleSection
          title="Deployment Manifest"
          isOpen={showManifest}
          onToggle={() => setShowManifest(!showManifest)}
        >
          <CodeBlock
            code={JSON.stringify(deployment.manifest, null, 2)}
            language="json"
            maxHeight="500px"
            showLineNumbers={true}
            showHeader={true}
          />
        </CollapsibleSection>
      )}

      {/* Setup Result */}
      {deployment.setup_result && (
        <CollapsibleSection
          title="Setup Result"
          isOpen={showSetupResult}
          onToggle={() => setShowSetupResult(!showSetupResult)}
        >
          <CodeBlock
            code={JSON.stringify(deployment.setup_result, null, 2)}
            language="json"
            maxHeight="400px"
            showLineNumbers={true}
            showHeader={true}
          />
        </CollapsibleSection>
      )}

      {/* Deploy Result */}
      {deployment.deploy_result && (
        <CollapsibleSection
          title="Deploy Result"
          isOpen={showDeployResult}
          onToggle={() => setShowDeployResult(!showDeployResult)}
        >
          <CodeBlock
            code={JSON.stringify(deployment.deploy_result, null, 2)}
            language="json"
            maxHeight="400px"
            showLineNumbers={true}
            showHeader={true}
          />
        </CollapsibleSection>
      )}

      {/* Logs */}
      {deployment.last_inspect_result && (
        <CollapsibleSection
          title="Logs"
          isOpen={showLogs}
          onToggle={() => setShowLogs(!showLogs)}
        >
          <LogsSection inspectResult={deployment.last_inspect_result} />
        </CollapsibleSection>
      )}
        </>
      )}
    </div>
  );
}

interface SwitchRowProps {
  label: string;
  description?: string;
  checked: boolean;
  disabled?: boolean;
  onChange: () => void;
}

function SwitchRow({ label, description, checked, disabled, onChange }: SwitchRowProps) {
  return (
    <div className="flex items-center justify-between gap-4">
      <div className="min-w-0">
        <p className={cn("text-sm font-medium", disabled ? "text-slate-500" : "text-slate-200")}>
          {label}
        </p>
        {description && (
          <p className={cn("text-xs mt-1", disabled ? "text-slate-600" : "text-slate-400")}>
            {description}
          </p>
        )}
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        onClick={() => {
          if (!disabled) onChange();
        }}
        disabled={disabled}
        className={cn(
          "relative inline-flex h-6 w-11 flex-shrink-0 items-center rounded-full border transition-colors",
          checked ? "bg-blue-500 border-blue-500" : "bg-slate-700 border-white/10",
          disabled && "opacity-50 cursor-not-allowed"
        )}
      >
        <span
          className={cn(
            "inline-block h-5 w-5 transform rounded-full bg-white shadow transition-transform",
            checked ? "translate-x-5" : "translate-x-1"
          )}
        />
      </button>
    </div>
  );
}

// Status badge (larger version for detail view)
function StatusBadgeLarge({ status }: { status: Deployment["status"] }) {
  const info = getStatusInfo(status);

  const colorClasses = {
    slate: "bg-slate-500/20 text-slate-400 border-slate-500/30",
    blue: "bg-blue-500/20 text-blue-400 border-blue-500/30",
    emerald: "bg-emerald-500/20 text-emerald-400 border-emerald-500/30",
    red: "bg-red-500/20 text-red-400 border-red-500/30",
    amber: "bg-amber-500/20 text-amber-400 border-amber-500/30",
  };

  const icons = {
    clock: Clock,
    loader: Loader2,
    check: CheckCircle2,
    "check-circle": CheckCircle2,
    "x-circle": XCircle,
    pause: Square,
    help: AlertCircle,
  };

  const IconComponent = icons[info.icon as keyof typeof icons] || AlertCircle;
  const isAnimated = info.icon === "loader";

  return (
    <span
      className={cn(
        "inline-flex items-center gap-2 px-3 py-1.5 rounded-lg border text-sm font-medium",
        colorClasses[info.color as keyof typeof colorClasses]
      )}
    >
      <IconComponent className={cn("h-4 w-4", isAnimated && "animate-spin")} />
      {info.label}
    </span>
  );
}

// Collapsible section component
interface CollapsibleSectionProps {
  title: string;
  isOpen: boolean;
  onToggle: () => void;
  children: React.ReactNode;
}

function CollapsibleSection({ title, isOpen, onToggle, children }: CollapsibleSectionProps) {
  return (
    <div className="border border-white/10 rounded-lg bg-slate-900/50">
      <button
        onClick={onToggle}
        className="w-full flex items-center justify-between p-4 hover:bg-white/5 transition-colors"
      >
        <h3 className="text-sm font-medium text-white">{title}</h3>
        {isOpen ? (
          <ChevronDown className="h-4 w-4 text-slate-400" />
        ) : (
          <ChevronRight className="h-4 w-4 text-slate-400" />
        )}
      </button>
      {isOpen && <div className="p-4 pt-0 border-t border-white/10">{children}</div>}
    </div>
  );
}

// Logs section component
function LogsSection({ inspectResult }: { inspectResult: Deployment["last_inspect_result"] }) {
  const result = inspectResult as {
    ok?: boolean;
    scenario_logs?: string;
    error?: string;
  };

  if (!result) return null;

  if (result.error) {
    return (
      <div className="text-red-400 text-sm">
        <p>Failed to fetch logs: {result.error}</p>
      </div>
    );
  }

  if (!result.scenario_logs) {
    return <p className="text-slate-400 text-sm">No logs available</p>;
  }

  return (
    <pre className="text-xs text-slate-300 bg-slate-950 p-4 rounded-lg overflow-x-auto max-h-96 overflow-y-auto font-mono">
      {result.scenario_logs}
    </pre>
  );
}
