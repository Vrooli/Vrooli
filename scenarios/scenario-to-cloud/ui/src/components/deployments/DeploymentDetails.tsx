import { useCallback, useEffect, useMemo, useState } from "react";
import {
  Activity,
  AlertCircle,
  ArrowLeft,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  Clock,
  FolderOpen,
  GitCompare,
  History,
  LayoutDashboard,
  Loader2,
  Play,
  RefreshCw,
  Search,
  Square,
  Terminal,
  XCircle,
  Key,
} from "lucide-react";
import {
  useDeployment,
  useInspectDeployment,
  useStopDeployment,
  useStartDeployment,
  useExecuteDeployment,
  getStatusInfo,
} from "../../hooks/useDeployments";
import { useDeploymentInvestigation } from "../../hooks/useInvestigation";
import { useHealthObservation, useLiveState } from "../../hooks/useLiveState";
import { useDeploymentUrl } from "../../hooks/useDeploymentUrl";
import { cn } from "../../lib/utils";
import type { Deployment } from "../../lib/api";
import type { DeploymentTab } from "../../types/url";
import type { DurableOperationPointer } from "../../types/console";
import { LiveStateTab, FilesTab, DriftTab, SecretsTab, HistoryTab, InvestigationsTab, TerminalTab, HealthObservationBadge } from "./tabs";
import { DeploymentConsole, IdentityHeader, ActionButton, DestructiveActionDialog } from "./console";
import { targetKey } from "../../lib/consoleActions";
import { CodeBlock } from "../ui/code-block";
import { Alert } from "../ui/alert";
import { SpawnAgentButton } from "../wizard/SpawnAgentButton";
import { InvestigationProgress } from "../wizard/InvestigationProgress";
import { InvestigationReport } from "../wizard/InvestigationReport";

interface DeploymentDetailsProps {
  deploymentId: string;
  onBack: () => void;
}

const ADVANCED_TABS: { id: DeploymentTab; label: string; Icon: typeof Activity }[] = [
  { id: "live-state", label: "Live State", Icon: Activity },
  { id: "files", label: "Files", Icon: FolderOpen },
  { id: "drift", label: "Drift", Icon: GitCompare },
  { id: "secrets", label: "Secrets", Icon: Key },
  { id: "history", label: "History", Icon: History },
  { id: "investigations", label: "Investigations", Icon: Search },
  { id: "terminal", label: "Terminal", Icon: Terminal },
];

/**
 * DeploymentDetails is the deployment page: a persistent identity header,
 * the console (release, health, operation, recovery) and, behind one
 * disclosure, the advanced surfaces (live state, files, drift, secrets,
 * history, investigations, terminal, raw results and the legacy pipeline).
 */
export function DeploymentDetails({ deploymentId, onBack }: DeploymentDetailsProps) {
  const { data: deploymentRecord, isLoading, error } = useDeployment(deploymentId);
  const inspectMutation = useInspectDeployment();
  const stopMutation = useStopDeployment();
  const startMutation = useStartDeployment();
  const executeMutation = useExecuteDeployment();
  const { state: urlState, setTab, openModal, closeModal } = useDeploymentUrl();
  const activeTab = urlState.tab;
  const showInvestigationReport = urlState.modal === "investigation-report";
  const [advancedOpen, setAdvancedOpen] = useState(activeTab !== "overview");
  const [showManifest, setShowManifest] = useState(false);
  const [showSetupResult, setShowSetupResult] = useState(false);
  const [showDeployResult, setShowDeployResult] = useState(false);
  const [showLogs, setShowLogs] = useState(false);
  const [pipelineOptions, setPipelineOptions] = useState({ forceBundleBuild: false, runPreflight: false });
  const [confirmPipeline, setConfirmPipeline] = useState(false);
  const [attachRequest, setAttachRequest] = useState<DurableOperationPointer | null>(null);

  useEffect(() => {
    if (activeTab !== "overview") setAdvancedOpen(true);
  }, [activeTab]);

  const investigation = useDeploymentInvestigation(deploymentId);
  const { data: liveState } = useLiveState(deploymentId);
  const { data: healthObservation } = useHealthObservation(deploymentId);

  const manifest = useMemo(
    () =>
      (deploymentRecord?.manifest ?? {}) as {
        scenario?: { id: string };
        edge?: { domain: string };
        target?: { vps?: { host: string } };
        dependencies?: { resources?: string[]; scenarios?: string[] };
      },
    [deploymentRecord?.manifest],
  );

  const isInvestigationOutdated = useCallback(
    (inv?: { created_at: string } | null) => {
      if (!inv || !deploymentRecord?.last_deployed_at) return false;
      return new Date(inv.created_at).getTime() < new Date(deploymentRecord.last_deployed_at).getTime();
    },
    [deploymentRecord?.last_deployed_at],
  );

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12" role="status" aria-label="Loading deployment">
        <Loader2 className="h-8 w-8 motion-safe:animate-spin text-blue-300" aria-hidden="true" />
      </div>
    );
  }

  if (error || !deploymentRecord) {
    return (
      <div className="space-y-4">
        <button type="button" onClick={onBack} className="flex items-center gap-2 text-slate-300 hover:text-white rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400">
          <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          Back to Deployments
        </button>
        <div role="alert" className="p-4 rounded-lg border border-red-500/30 bg-red-500/10 text-red-200">
          <div className="flex items-center gap-2">
            <AlertCircle className="h-5 w-5" aria-hidden="true" />
            <span>{error?.message || "Deployment not found"}</span>
          </div>
        </div>
      </div>
    );
  }

  const mutationError = [inspectMutation, stopMutation, startMutation, executeMutation].map((m) => m.error).find((e): e is Error => e instanceof Error);

  return (
    <div className="space-y-6" data-testid="deployment-details">
      <IdentityHeader
        deployment={deploymentRecord}
        domain={manifest.edge?.domain ?? null}
        onBack={onBack}
        badges={
          <>
            <StatusBadgeLarge status={deploymentRecord.status} />
            <SSHKeyAuthBadge liveState={liveState} />
            <HealthObservationBadge observation={healthObservation} />
          </>
        }
        actions={
          <>
            <button
              type="button"
              onClick={() => inspectMutation.mutate(deploymentId)}
              disabled={inspectMutation.isPending}
              data-testid="console-inspect"
              className={cn(
                "flex items-center gap-2 px-4 py-2 rounded-lg border border-white/15 text-sm font-medium text-slate-100 hover:bg-white/5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400",
                inspectMutation.isPending && "opacity-60 cursor-not-allowed",
              )}
            >
              {inspectMutation.isPending ? <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden="true" /> : <RefreshCw className="h-4 w-4" aria-hidden="true" />}
              Inspect
            </button>
            <SpawnAgentButton deploymentId={deploymentId} onTaskStarted={() => {}} />
          </>
        }
      />

      {mutationError && (
        <Alert variant="error" title="Action refused">
          {mutationError.message}
        </Alert>
      )}

      {investigation.activeInvestigation && (
        <InvestigationProgress
          investigation={investigation.activeInvestigation}
          isRunning={investigation.isRunning}
          onStop={investigation.stop}
          isStopping={investigation.isStopping}
          isOutdated={isInvestigationOutdated(investigation.activeInvestigation)}
          onViewReport={(invId) => {
            investigation.viewReport(invId);
            openModal("investigation-report", { invId });
          }}
        />
      )}

      {showInvestigationReport && investigation.activeInvestigation && (
        <InvestigationReport
          investigation={investigation.activeInvestigation}
          onClose={closeModal}
          isOutdated={isInvestigationOutdated(investigation.activeInvestigation)}
          onApplyFixes={async (invId, options) => {
            await investigation.applyFixes(invId, options);
            closeModal();
          }}
          isApplyingFixes={investigation.isApplyingFixes}
        />
      )}

      {deploymentRecord.error_message && (
        <div role="alert" className="p-4 rounded-lg border border-red-500/30 bg-red-500/10" data-testid="deployment-record-error">
          <div className="flex items-start gap-2">
            <XCircle className="h-5 w-5 text-red-300 mt-0.5" aria-hidden="true" />
            <div>
              <p className="font-medium text-red-200">Last recorded failure at: {deploymentRecord.error_step || "unknown step"}</p>
              <p className="text-red-100 mt-1">{deploymentRecord.error_message}</p>
            </div>
          </div>
        </div>
      )}

      <DeploymentConsole deployment={deploymentRecord} attachRequest={attachRequest} />

      {/* Advanced surfaces: raw process, files, terminal and the legacy pipeline stay behind one disclosure. */}
      <section className="border border-white/10 rounded-lg bg-slate-900/40" aria-labelledby="console-advanced-heading">
        <h2 id="console-advanced-heading" className="sr-only">
          Advanced surfaces
        </h2>
        <button
          type="button"
          onClick={() => setAdvancedOpen((open) => !open)}
          aria-expanded={advancedOpen}
          aria-controls="console-advanced-region"
          data-testid="console-advanced-toggle"
          className="w-full flex items-center justify-between p-4 text-left hover:bg-white/5 rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400"
        >
          <span className="text-sm font-medium text-slate-100">Advanced surfaces</span>
          <span className="flex items-center gap-2 text-xs text-slate-300">
            live state, files, drift, secrets, history, investigations, terminal, raw results
            {advancedOpen ? <ChevronDown className="h-4 w-4" aria-hidden="true" /> : <ChevronRight className="h-4 w-4" aria-hidden="true" />}
          </span>
        </button>
        {advancedOpen && (
          <div id="console-advanced-region" className="p-4 pt-0 space-y-4">
            <div className="rounded-lg border border-white/10 bg-slate-900/60 p-4 space-y-3" data-testid="console-pipeline">
              <p className="text-xs uppercase tracking-wide text-slate-400">Workload controls</p>
              <p className="text-xs text-slate-300">These call the API directly; a refusal is shown with its reason. The reviewed plan above is the preferred path.</p>
              <div className="flex flex-wrap gap-2">
                <ActionButton available={!stopMutation.isPending} tone="secondary" testId="console-stop" onClick={() => stopMutation.mutate(deploymentId)}>
                  <Square className="h-4 w-4" aria-hidden="true" />
                  Stop
                </ActionButton>
                <ActionButton available={!startMutation.isPending} tone="secondary" testId="console-start" onClick={() => startMutation.mutate(deploymentId)}>
                  <Play className="h-4 w-4" aria-hidden="true" />
                  Start
                </ActionButton>
              </div>
              <div className="flex flex-wrap items-center gap-4">
                <SwitchRow label="Build a new bundle" checked={pipelineOptions.forceBundleBuild} onChange={() => setPipelineOptions((o) => ({ ...o, forceBundleBuild: !o.forceBundleBuild }))} />
                <SwitchRow label="Run preflight checks" checked={pipelineOptions.runPreflight} onChange={() => setPipelineOptions((o) => ({ ...o, runPreflight: !o.runPreflight }))} />
                <ActionButton available={!executeMutation.isPending} testId="console-run-pipeline" onClick={() => setConfirmPipeline(true)}>
                  Run pipeline with options
                </ActionButton>
              </div>
            </div>

            <div className="flex gap-1 border-b border-white/10 pb-px overflow-x-auto" role="tablist" aria-label="Advanced surfaces">
              <TabButton id="overview" label="Overview" Icon={LayoutDashboard} active={activeTab === "overview"} onClick={() => setTab("overview")} />
              {ADVANCED_TABS.map((tab) => (
                <TabButton key={tab.id} id={tab.id} label={tab.label} Icon={tab.Icon} active={activeTab === tab.id} onClick={() => setTab(tab.id)} />
              ))}
            </div>

            <div role="tabpanel" id={`console-tabpanel-${activeTab}`} aria-labelledby={`console-tab-${activeTab}`}>
              {activeTab === "live-state" && <LiveStateTab deploymentId={deploymentId} deploymentName={deploymentRecord.name} />}
              {activeTab === "files" && <FilesTab deploymentId={deploymentId} />}
              {activeTab === "drift" && <DriftTab deploymentId={deploymentId} />}
              {activeTab === "secrets" && <SecretsTab deploymentId={deploymentId} />}
              {activeTab === "history" && <HistoryTab deploymentId={deploymentId} />}
              {activeTab === "investigations" && (
                <InvestigationsTab
                  deploymentId={deploymentId}
                  lastDeployedAt={deploymentRecord.last_deployed_at}
                  onViewReport={(inv) => {
                    investigation.viewReport(inv.id);
                    openModal("investigation-report", { invId: inv.id });
                  }}
                />
              )}
              {activeTab === "terminal" && <TerminalTab deploymentId={deploymentId} />}
              {activeTab === "overview" && (
                <div className="space-y-4">
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div className="border border-white/10 rounded-lg bg-slate-900/50 p-4">
                      <h3 className="text-sm font-medium text-slate-300 mb-3">Record</h3>
                      <dl className="space-y-2 text-sm">
                        <div className="flex items-center gap-2">
                          <dt className="text-slate-300">Scenario:</dt>
                          <dd className="text-white">{manifest.scenario?.id || deploymentRecord.scenario_id}</dd>
                        </div>
                        {manifest.target?.vps?.host && (
                          <div className="flex items-center gap-2">
                            <dt className="text-slate-300">Manifest host:</dt>
                            <dd className="text-white">{manifest.target.vps.host}</dd>
                          </div>
                        )}
                        <div className="flex items-center gap-2">
                          <dt className="text-slate-300">Target key:</dt>
                          <dd className="text-white font-mono text-xs">{targetKey(deploymentRecord.target)}</dd>
                        </div>
                      </dl>
                    </div>
                    <div className="border border-white/10 rounded-lg bg-slate-900/50 p-4">
                      <h3 className="text-sm font-medium text-slate-300 mb-3">Timeline</h3>
                      <dl className="space-y-2 text-sm">
                        <div className="flex items-center gap-2">
                          <Clock className="h-4 w-4 text-slate-400" aria-hidden="true" />
                          <dt className="text-slate-300">Created:</dt>
                          <dd className="text-white">{new Date(deploymentRecord.created_at).toLocaleString()}</dd>
                        </div>
                        {deploymentRecord.last_deployed_at && (
                          <div className="flex items-center gap-2">
                            <CheckCircle2 className="h-4 w-4 text-emerald-300" aria-hidden="true" />
                            <dt className="text-slate-300">Last deployed:</dt>
                            <dd className="text-white">{new Date(deploymentRecord.last_deployed_at).toLocaleString()}</dd>
                          </div>
                        )}
                        {deploymentRecord.last_inspected_at && (
                          <div className="flex items-center gap-2">
                            <RefreshCw className="h-4 w-4 text-blue-300" aria-hidden="true" />
                            <dt className="text-slate-300">Last inspected:</dt>
                            <dd className="text-white">{new Date(deploymentRecord.last_inspected_at).toLocaleString()}</dd>
                          </div>
                        )}
                      </dl>
                    </div>
                  </div>

                  {(manifest.dependencies?.resources?.length || manifest.dependencies?.scenarios?.length) ? (
                    <div className="border border-white/10 rounded-lg bg-slate-900/50 p-4">
                      <h3 className="text-sm font-medium text-slate-300 mb-3">Dependencies</h3>
                      <ul className="flex flex-wrap gap-2">
                        {manifest.dependencies?.resources?.map((res) => (
                          <li key={res} className="px-2 py-1 rounded bg-purple-500/20 text-purple-200 text-xs">
                            {res}
                          </li>
                        ))}
                        {manifest.dependencies?.scenarios?.map((scen) => (
                          <li key={scen} className="px-2 py-1 rounded bg-blue-500/20 text-blue-200 text-xs">
                            {scen}
                          </li>
                        ))}
                      </ul>
                    </div>
                  ) : null}

                  {deploymentRecord.manifest ? (
                    <CollapsibleSection title="Deployment Manifest" isOpen={showManifest} onToggle={() => setShowManifest(!showManifest)}>
                      <CodeBlock code={JSON.stringify(deploymentRecord.manifest, null, 2)} language="json" maxHeight="500px" showLineNumbers showHeader />
                    </CollapsibleSection>
                  ) : null}
                  {deploymentRecord.setup_result ? (
                    <CollapsibleSection title="Setup Result" isOpen={showSetupResult} onToggle={() => setShowSetupResult(!showSetupResult)}>
                      <CodeBlock code={JSON.stringify(deploymentRecord.setup_result, null, 2)} language="json" maxHeight="400px" showLineNumbers showHeader />
                    </CollapsibleSection>
                  ) : null}
                  {deploymentRecord.deploy_result ? (
                    <CollapsibleSection title="Deploy Result" isOpen={showDeployResult} onToggle={() => setShowDeployResult(!showDeployResult)}>
                      <CodeBlock code={JSON.stringify(deploymentRecord.deploy_result, null, 2)} language="json" maxHeight="400px" showLineNumbers showHeader />
                    </CollapsibleSection>
                  ) : null}
                  {deploymentRecord.last_inspect_result && (
                    <CollapsibleSection title="Logs" isOpen={showLogs} onToggle={() => setShowLogs(!showLogs)}>
                      <LogsSection inspectResult={deploymentRecord.last_inspect_result} />
                    </CollapsibleSection>
                  )}
                </div>
              )}
            </div>
          </div>
        )}
      </section>

      {confirmPipeline && (
        <DestructiveActionDialog
          title="Run the deployment pipeline?"
          description={`${pipelineOptions.runPreflight ? "Preflight checks, then " : ""}${pipelineOptions.forceBundleBuild ? "a fresh bundle build, then " : ""}setup and deploy will run against the target as a durable operation.`}
          target={targetKey(deploymentRecord.target)}
          affectedData={[]}
          details={[{ label: "Options", value: `${pipelineOptions.forceBundleBuild ? "build new bundle" : "reuse bundle"}, ${pipelineOptions.runPreflight ? "run preflight" : "skip preflight"}` }]}
          confirmText={deploymentId.slice(0, 8)}
          confirmLabel="Run pipeline"
          isPending={executeMutation.isPending}
          error={executeMutation.error instanceof Error ? executeMutation.error.message : null}
          onConfirm={async () => {
            const result = await executeMutation.mutateAsync({ id: deploymentId, options: pipelineOptions });
            if (result.operation_id) {
              setAttachRequest({ deployment_id: deploymentId, operation_id: result.operation_id, plan_digest: result.plan_digest });
            }
            setConfirmPipeline(false);
          }}
          onCancel={() => setConfirmPipeline(false)}
        />
      )}
    </div>
  );
}

function TabButton({ id, label, Icon, active, onClick }: { id: DeploymentTab; label: string; Icon: typeof Activity; active: boolean; onClick: () => void }) {
  return (
    <button
      type="button"
      role="tab"
      id={`console-tab-${id}`}
      aria-selected={active}
      aria-controls={`console-tabpanel-${id}`}
      onClick={onClick}
      className={cn(
        "flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-t-lg whitespace-nowrap focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400",
        active ? "bg-slate-800 text-white border-b-2 border-blue-400" : "text-slate-300 hover:text-white hover:bg-slate-800/50",
      )}
    >
      <Icon className="h-4 w-4" aria-hidden="true" />
      {label}
    </button>
  );
}

function SSHKeyAuthBadge({ liveState }: { liveState: { system?: { ssh?: { verification_state?: "authorized" | "unauthorized" | "unknown" } } } | null | undefined }) {
  if (!liveState?.system?.ssh) {
    return null;
  }
  const keyState = liveState.system.ssh.verification_state ?? "unknown";
  const config = {
    authorized: { text: "SSH key authorized", className: "bg-emerald-500/15 text-emerald-200 border-emerald-400/30", Icon: CheckCircle2 },
    unauthorized: { text: "SSH key unauthorized", className: "bg-amber-500/15 text-amber-200 border-amber-400/30", Icon: AlertCircle },
    unknown: { text: "SSH key auth unknown", className: "bg-slate-500/15 text-slate-200 border-slate-400/30", Icon: AlertCircle },
  }[keyState];
  return (
    <span className={cn("inline-flex items-center gap-2 px-3 py-1.5 rounded-lg border text-sm font-medium", config.className)}>
      <config.Icon className="h-4 w-4" aria-hidden="true" />
      {config.text}
    </span>
  );
}

function SwitchRow({ label, checked, onChange }: { label: string; checked: boolean; onChange: () => void }) {
  return (
    <label className="inline-flex items-center gap-2 text-sm text-slate-100">
      <input type="checkbox" role="switch" aria-checked={checked} checked={checked} onChange={onChange} className="h-4 w-4 rounded border-white/30 bg-slate-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400" />
      {label}
    </label>
  );
}

function StatusBadgeLarge({ status }: { status: Deployment["status"] }) {
  const info = getStatusInfo(status);
  const colorClasses = {
    slate: "bg-slate-500/15 text-slate-200 border-slate-400/30",
    blue: "bg-blue-500/15 text-blue-200 border-blue-400/30",
    emerald: "bg-emerald-500/15 text-emerald-200 border-emerald-400/30",
    red: "bg-red-500/15 text-red-200 border-red-400/30",
    amber: "bg-amber-500/15 text-amber-200 border-amber-400/30",
  };
  const icons = { clock: Clock, loader: Loader2, check: CheckCircle2, "check-circle": CheckCircle2, "x-circle": XCircle, pause: Square, help: AlertCircle };
  const IconComponent = icons[info.icon as keyof typeof icons] || AlertCircle;
  return (
    <span data-testid="console-status" className={cn("inline-flex items-center gap-2 px-3 py-1.5 rounded-lg border text-sm font-medium", colorClasses[info.color as keyof typeof colorClasses])}>
      <IconComponent className={cn("h-4 w-4", info.icon === "loader" && "motion-safe:animate-spin")} aria-hidden="true" />
      {info.label}
    </span>
  );
}

function CollapsibleSection({ title, isOpen, onToggle, children }: { title: string; isOpen: boolean; onToggle: () => void; children: React.ReactNode }) {
  return (
    <div className="border border-white/10 rounded-lg bg-slate-900/50">
      <button type="button" onClick={onToggle} aria-expanded={isOpen} className="w-full flex items-center justify-between p-4 hover:bg-white/5 rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400">
        <h3 className="text-sm font-medium text-white">{title}</h3>
        {isOpen ? <ChevronDown className="h-4 w-4 text-slate-300" aria-hidden="true" /> : <ChevronRight className="h-4 w-4 text-slate-300" aria-hidden="true" />}
      </button>
      {isOpen && <div className="p-4 pt-0 border-t border-white/10">{children}</div>}
    </div>
  );
}

function LogsSection({ inspectResult }: { inspectResult: Deployment["last_inspect_result"] }) {
  const result = inspectResult as { ok?: boolean; scenario_logs?: string; error?: string };
  if (!result) return null;
  if (result.error) {
    return <p className="text-red-200 text-sm">Failed to fetch logs: {result.error}</p>;
  }
  if (!result.scenario_logs) {
    return <p className="text-slate-300 text-sm">No logs available</p>;
  }
  return <pre className="text-xs text-slate-200 bg-slate-950 p-4 rounded-lg overflow-x-auto max-h-96 overflow-y-auto font-mono">{result.scenario_logs}</pre>;
}
