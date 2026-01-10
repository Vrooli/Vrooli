import { useCallback, useEffect, useMemo, useRef, useState } from "react";
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
  X,
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
import { useDeploymentProgress } from "../../hooks/useDeploymentProgress";
import { useDeploymentInvestigation } from "../../hooks/useInvestigation";
import { cn } from "../../lib/utils";
import { runPreflight as runPreflightApi, type Deployment, type PreflightCheck, type PreflightResponse } from "../../lib/api";
import type { StepStatus } from "../../types/progress";
import { LiveStateTab, FilesTab, DriftTab, HistoryTab, InvestigationsTab, TerminalTab } from "./tabs";
import { CodeBlock } from "../ui/code-block";
import { Alert } from "../ui/alert";
import { Stepper, type StepperStatus } from "../ui/stepper";
import { InvestigateButton } from "../wizard/InvestigateButton";
import { InvestigationProgress } from "../wizard/InvestigationProgress";
import { InvestigationReport } from "../wizard/InvestigationReport";
import { BuildStatusPanel } from "../wizard/StepBuild";
import {
  buildChecksToDisplay,
  buildReadOnlyChecks,
  DiskUsageModal,
  PortStopModal,
  PreflightChecksPanel,
  usePreflightActions,
  type CheckState,
} from "../wizard/StepPreflight";
import { DeploymentProgressView, getStepStatusFromProgress } from "../wizard/DeploymentProgress";

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
  const [redeployActive, setRedeployActive] = useState(false);
  const [redeployOptionsSnapshot, setRedeployOptionsSnapshot] = useState<{
    runPreflight: boolean;
    forceBundleBuild: boolean;
  } | null>(null);
  const [redeployViewStep, setRedeployViewStep] = useState<number | null>(null);
  const redeployRunId = deployment?.run_id ?? null;
  const [activeRunId, setActiveRunId] = useState<string | null>(null);
  const lastBundleRefreshRunRef = useRef<string | null>(null);
  const [redeployPreflightResult, setRedeployPreflightResult] = useState<PreflightResponse | null>(null);
  const [redeployPreflightError, setRedeployPreflightError] = useState<string | null>(null);
  const [redeployPreflightRunning, setRedeployPreflightRunning] = useState(false);

  const {
    progress: redeployProgress,
    isConnected: redeployConnected,
    connectionError: redeployConnectionError,
    reset: resetRedeployProgress,
  } = useDeploymentProgress(redeployActive ? deploymentId : null, { runId: activeRunId ?? redeployRunId });

  // Investigation state
  const investigation = useDeploymentInvestigation(deploymentId);

  const deploymentRecord = deployment ?? null;
  const statusInfo = getStatusInfo(deploymentRecord?.status ?? "pending");
  const manifest = (deploymentRecord?.manifest ?? {}) as {
    scenario?: { id: string };
    edge?: { domain: string };
    target?: { vps?: { host: string } };
    dependencies?: { resources?: string[]; scenarios?: string[] };
  };

  const isInProgress =
    deploymentRecord?.status === "pending" ||
    deploymentRecord?.status === "setup_running" ||
    deploymentRecord?.status === "deploying";

  const canRedeploy =
    deploymentRecord?.status === "deployed" ||
    deploymentRecord?.status === "failed" ||
    deploymentRecord?.status === "stopped";

  const canStop = deploymentRecord?.status === "deployed";
  const hasExistingBundle = Boolean(deploymentRecord?.bundle_path);
  const canViewRedeployProgress = redeployActive || isInProgress;
  const redeployLocked = executeMutation.isPending || isInProgress;
  const redeployOptions = redeployOptionsSnapshot ?? {
    runPreflight,
    forceBundleBuild: buildNewBundle,
  };
  const redeployPreflightEnabled = redeployOptions.runPreflight;
  const includeBuildStep = redeployOptions.forceBundleBuild;
  const includePreflightStep = redeployPreflightEnabled;
  const buildStepStatus = getStepStatusFromProgress(redeployProgress, "bundle_build");
  const buildStatus: StepStatus =
    redeployActive && executeMutation.isPending && buildStepStatus === "pending"
      ? "running"
      : buildStepStatus;
  const buildError =
    redeployProgress?.error && redeployProgress.currentStep === "bundle_build"
      ? redeployProgress.error
      : null;
  const shouldHideBundleFallback =
    redeployOptions.forceBundleBuild && (!redeployActive || buildStatus !== "completed");
  const bundlePathFallback = shouldHideBundleFallback ? null : deploymentRecord?.bundle_path ?? null;
  const bundleShaFallback = shouldHideBundleFallback ? null : deploymentRecord?.bundle_sha256 ?? null;
  const bundleSizeFallback = shouldHideBundleFallback ? null : deploymentRecord?.bundle_size_bytes ?? null;

  const preflightStepStatus = useMemo<StepStatus>(() => {
    if (!redeployPreflightEnabled) {
      return buildStatus === "completed" && redeployActive ? "completed" : "pending";
    }
    if (!redeployProgress) return "pending";
    if (redeployProgress.error && redeployProgress.currentStep === "preflight") return "failed";
    if (redeployProgress.currentStep === "preflight") return "running";
    if (
      redeployProgress.currentStep &&
      redeployProgress.currentStep !== "bundle_build" &&
      redeployProgress.currentStep !== "preflight"
    ) {
      return "completed";
    }
    if (redeployProgress.isComplete && !redeployProgress.error) return "completed";
    return "pending";
  }, [buildStatus, redeployActive, redeployPreflightEnabled, redeployProgress]);

  const deploymentStepStatus = useMemo<StepStatus>(() => {
    if (!redeployProgress) return "pending";
    const isDeployStep =
      redeployProgress.currentStep !== "" &&
      redeployProgress.currentStep !== "bundle_build" &&
      redeployProgress.currentStep !== "preflight";
    const hasDeploymentProgress = redeployProgress.steps?.some(
      (step) => step.id !== "bundle_build" && step.status !== "pending",
    );
    if (redeployProgress.error && isDeployStep) return "failed";
    if (redeployProgress.isComplete && !redeployProgress.error) return "completed";
    if (hasDeploymentProgress || isDeployStep) return "running";
    return "pending";
  }, [redeployProgress]);

  const effectivePreflightStatus =
    !includeBuildStep && (buildStatus === "running" || buildStatus === "failed")
      ? buildStatus
      : preflightStepStatus;
  const effectiveDeployStatus =
    !includeBuildStep && !includePreflightStep && (buildStatus === "running" || buildStatus === "failed")
      ? buildStatus
      : deploymentStepStatus;
  const redeployStepStatuses: StepStatus[] = [
    ...(includeBuildStep ? [buildStatus] : []),
    ...(includePreflightStep ? [effectivePreflightStatus] : []),
    effectiveDeployStatus,
  ];
  const currentRedeployStepIndex = useMemo(() => {
    const firstActive = redeployStepStatuses.findIndex((status) => status !== "completed");
    return firstActive === -1 ? redeployStepStatuses.length - 1 : firstActive;
  }, [redeployStepStatuses]);

  const preflightCheckState: CheckState =
    preflightStepStatus === "completed"
      ? "pass"
      : preflightStepStatus === "failed"
        ? "fail"
        : preflightStepStatus === "running"
          ? "running"
          : "pending";
  const preflightChecksRaw: PreflightCheck[] | null =
    redeployProgress?.preflightResult?.checks ??
    redeployPreflightResult?.checks ??
    deploymentRecord?.preflight_result?.checks ??
    null;
  const preflightChecks = useMemo(() => {
    if (preflightChecksRaw?.length) {
      return buildChecksToDisplay(preflightChecksRaw, redeployPreflightRunning);
    }
    return buildReadOnlyChecks(preflightCheckState);
  }, [preflightCheckState, preflightChecksRaw, redeployPreflightRunning]);
  const redeploySteps = useMemo(
    () => [
      ...(includeBuildStep ? [{ id: "build", label: "Build" }] : []),
      ...(includePreflightStep ? [{ id: "preflight", label: "Preflight" }] : []),
      { id: "deploy", label: "Deployment" },
    ],
    [includeBuildStep, includePreflightStep],
  );
  const stepperStates = useMemo<StepperStatus[]>(() => {
    const statusById: Record<string, StepperStatus> = {
      build: buildStatus,
      preflight: effectivePreflightStatus,
      deploy: effectiveDeployStatus,
    };
    return redeploySteps.map((step) => statusById[step.id] ?? "pending");
  }, [
    buildStatus,
    effectiveDeployStatus,
    effectivePreflightStatus,
    redeploySteps,
  ]);
  const viewedStepIndex = redeployViewStep ?? currentRedeployStepIndex;
  const displayedStepId = redeploySteps[viewedStepIndex]?.id ?? "deploy";
  const redeployStartError =
    executeMutation.error instanceof Error ? executeMutation.error.message : null;

  const runRedeployPreflight = useCallback(async () => {
    if (!deploymentRecord?.manifest) {
      setRedeployPreflightError("Missing deployment manifest. Please inspect the deployment and try again.");
      return;
    }
    setRedeployPreflightRunning(true);
    setRedeployPreflightError(null);
    try {
      const result = await runPreflightApi(deploymentRecord.manifest);
      setRedeployPreflightResult(result);
    } catch (err) {
      setRedeployPreflightError(err instanceof Error ? err.message : String(err));
    } finally {
      setRedeployPreflightRunning(false);
    }
  }, [deploymentRecord?.manifest]);

  const {
    actionLoading: preflightActionLoading,
    actionError: preflightActionError,
    diskUsage: preflightDiskUsage,
    showDiskModal: showPreflightDiskModal,
    setShowDiskModal: setShowPreflightDiskModal,
    cleanupLoading: preflightCleanupLoading,
    showPortModal: showPreflightPortModal,
    setShowPortModal: setShowPreflightPortModal,
    portSelections: preflightPortSelections,
    portBindings: preflightPortBindings,
    handleAction: handlePreflightAction,
    handlePortStop: handlePreflightPortStop,
    togglePortService: togglePreflightPortService,
    togglePortPID: togglePreflightPortPID,
    handleCleanup: handlePreflightCleanup,
  } = usePreflightActions({
    manifest,
    sshKeyPath: null,
    preflightChecks: preflightChecksRaw,
    onRecheck: runRedeployPreflight,
  });

  const hydrateRedeployOptions = useCallback(() => {
    if (redeployOptionsSnapshot) return;
    try {
      const stored = sessionStorage.getItem(`stc.redeploy.options.${deploymentId}`);
      if (!stored) return;
      const parsed = JSON.parse(stored) as { runPreflight: boolean; forceBundleBuild: boolean };
      if (typeof parsed.runPreflight !== "boolean" || typeof parsed.forceBundleBuild !== "boolean") {
        return;
      }
      setRedeployOptionsSnapshot(parsed);
      setRunPreflight(parsed.runPreflight);
      setBuildNewBundle(parsed.forceBundleBuild);
    } catch {
      // Ignore storage errors
    }
  }, [deploymentId, redeployOptionsSnapshot]);

  useEffect(() => {
    if (redeployViewStep !== null && redeployViewStep >= redeploySteps.length) {
      setRedeployViewStep(null);
    }
  }, [redeploySteps.length, redeployViewStep]);

  useEffect(() => {
    if (redeployActive && !redeployOptionsSnapshot) {
      hydrateRedeployOptions();
    }
  }, [hydrateRedeployOptions, redeployActive, redeployOptionsSnapshot]);

  useEffect(() => {
    if (!includeBuildStep || buildStatus !== "completed") return;
    if (deploymentRecord?.bundle_path) return;
    const runKey = redeployRunId ?? deploymentRecord?.run_id ?? "unknown";
    if (lastBundleRefreshRunRef.current === runKey) return;
    lastBundleRefreshRunRef.current = runKey;
    void refetch();
  }, [
    buildStatus,
    deploymentRecord?.bundle_path,
    deploymentRecord?.run_id,
    includeBuildStep,
    redeployRunId,
    refetch,
  ]);

  const isInvestigationOutdated = (inv?: { deployment_run_id?: string; created_at: string } | null) => {
    if (!inv) return false;
    if (deploymentRecord?.run_id && inv.deployment_run_id) {
      return inv.deployment_run_id !== deploymentRecord.run_id;
    }
    if (deploymentRecord?.last_deployed_at) {
      return new Date(inv.created_at).getTime() < new Date(deploymentRecord.last_deployed_at).getTime();
    }
    return false;
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-blue-400" />
      </div>
    );
  }

  if (error || !deploymentRecord) {
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

  const openRedeployDialog = () => {
    setBuildNewBundle(!hasExistingBundle);
    setRunPreflight(false);
    setRedeployActive(false);
    setActiveRunId(null);
    setRedeployOptionsSnapshot(null);
    setRedeployViewStep(null);
    resetRedeployProgress();
    setShowRedeployDialog(true);
  };

  const openRedeployProgress = () => {
    hydrateRedeployOptions();
    setRedeployActive(true);
    setShowRedeployDialog(true);
  };

  const startRedeploy = () => {
    const snapshot = { runPreflight, forceBundleBuild: buildNewBundle };
    setRedeployOptionsSnapshot(snapshot);
    // Reset state to disconnect any existing SSE and show loading state
    // This handles the case where user clicks Re-deploy again after a previous deployment
    setRedeployActive(false);
    setActiveRunId(null);
    resetRedeployProgress();
    try {
      sessionStorage.setItem(`stc.redeploy.options.${deploymentId}`, JSON.stringify(snapshot));
    } catch {
      // Ignore storage errors
    }
    executeMutation.mutate(
      {
        id: deploymentId,
        options: {
          runPreflight: snapshot.runPreflight,
          forceBundleBuild: snapshot.forceBundleBuild,
        },
      },
      {
        onSuccess: (data) => {
          setActiveRunId(data.run_id);
          setRedeployActive(true);
        },
        onError: () => {
          setRedeployActive(false);
        },
      },
    );
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

            {canViewRedeployProgress && !showRedeployDialog && (
              <button
                onClick={openRedeployProgress}
                className={cn(
                  "flex items-center gap-2 px-4 py-2 rounded-lg border border-white/10",
                  "hover:bg-white/5 transition-colors text-sm font-medium text-slate-200"
                )}
              >
                <Activity className="h-4 w-4" />
                View Progress
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
          <div className="w-full max-w-5xl rounded-lg border border-white/10 bg-slate-900 p-6 shadow-xl max-h-[90vh] flex flex-col">
            <div className="mb-4 flex items-start justify-between gap-4">
              <div>
                <h3 className="text-lg font-semibold text-white">Re-deploy configuration</h3>
                <p className="text-sm text-slate-400 mt-1">
                  Choose what will run. Deployment always executes.
                </p>
              </div>
              <button
                type="button"
                onClick={() => setShowRedeployDialog(false)}
                className="text-slate-400 hover:text-white transition-colors"
                aria-label="Close redeploy dialog"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            {!deploymentRecord && (
              <div className="flex items-center justify-center py-10 text-slate-400">
                <Loader2 className="h-5 w-5 animate-spin mr-2" />
                Loading deployment details...
              </div>
            )}

            {deploymentRecord && (
            <div className="grid gap-6 lg:grid-cols-[1.1fr_1fr] overflow-y-auto pr-1">
              <div className="space-y-4">
                {redeployStartError && (
                  <Alert variant="error" title="Failed to start redeploy">
                    {redeployStartError}
                  </Alert>
                )}

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
                    disabled={!hasExistingBundle || redeployLocked}
                    onChange={() => setBuildNewBundle(!buildNewBundle)}
                  />
                </div>

                <div className="rounded-lg border border-white/10 bg-slate-900/60 p-4">
                  <p className="text-xs uppercase tracking-wide text-slate-500 mb-3">Checks</p>
                  <SwitchRow
                    label="Run preflight checks"
                    description="Validates DNS, SSH, OS, and prerequisites before deployment."
                    checked={runPreflight}
                    disabled={redeployLocked}
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

                <div className="rounded-lg border border-white/10 bg-slate-900/60 p-3">
                  <p className="text-xs uppercase tracking-wide text-slate-500 mb-2">What will happen</p>
                  <p className="text-sm text-slate-300">
                    {runPreflight ? "Run preflight checks, then " : "Skip preflight checks, then "}
                    {buildNewBundle ? "build a new bundle" : "reuse the existing bundle"}, then run VPS setup and deploy.
                  </p>
                </div>

                <div className="flex justify-end gap-3">
                  <button
                    onClick={() => setShowRedeployDialog(false)}
                    className="px-4 py-2 rounded-lg font-medium text-slate-400 hover:text-white hover:bg-white/5 transition-colors"
                  >
                    Close
                  </button>
                  <button
                    onClick={startRedeploy}
                    disabled={redeployLocked}
                    className="px-4 py-2 rounded-lg font-medium bg-blue-500 hover:bg-blue-600 text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                  >
                    {executeMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                    {executeMutation.isPending ? "Starting..." : "Re-deploy"}
                  </button>
                </div>
              </div>

              <div className="space-y-4">
                <div className="rounded-lg border border-white/10 bg-slate-900/60 p-4">
                  <p className="text-xs uppercase tracking-wide text-slate-500 mb-3">Progress</p>
                  <Stepper
                    steps={redeploySteps}
                    currentStep={currentRedeployStepIndex}
                    stepStates={stepperStates}
                    allowFutureClicks
                    displayedStep={viewedStepIndex}
                    onStepClick={(index) => setRedeployViewStep(index)}
                  />
                </div>

                {displayedStepId === "build" && (
                  <div className="rounded-lg border border-white/10 bg-slate-900/60 p-4 space-y-3">
                    <p className="text-xs uppercase tracking-wide text-slate-500">Build</p>
                    <p className="text-xs text-slate-400">
                      {redeployOptions.forceBundleBuild
                        ? "Building a fresh bundle for this run."
                        : "Reusing the most recent bundle unless a new build is required."}
                    </p>
                    <BuildStatusPanel
                      mode="readonly"
                      isBuilding={buildStatus === "running"}
                      bundleArtifact={null}
                      bundleError={buildError}
                      bundlePathFallback={bundlePathFallback}
                      bundleShaFallback={bundleShaFallback}
                      bundleSizeBytesFallback={bundleSizeFallback}
                    />
                  </div>
                )}

                {displayedStepId === "preflight" && (
                  <div className="rounded-lg border border-white/10 bg-slate-900/60 p-4 space-y-3">
                    <p className="text-xs uppercase tracking-wide text-slate-500">Preflight</p>
                    {!redeployPreflightEnabled && (
                      <Alert variant="info" title="Preflight skipped">
                        This redeploy will proceed without preflight checks.
                      </Alert>
                    )}
                    {redeployPreflightError && (
                      <Alert variant="error" title="Preflight failed">
                        {redeployPreflightError}
                      </Alert>
                    )}
                    {preflightActionError && (
                      <Alert variant="error" title="Action failed">
                        {preflightActionError}
                      </Alert>
                    )}
                    {redeployPreflightEnabled && preflightStepStatus === "running" && (
                      <Alert variant="info" title="Preflight running">
                        Running preflight checks against the target VPS.
                      </Alert>
                    )}
                    {redeployPreflightEnabled && preflightStepStatus === "completed" && (
                      <Alert variant="success" title="Preflight complete">
                        Preflight checks passed for this redeploy.
                      </Alert>
                    )}
                    {redeployPreflightEnabled && preflightStepStatus === "failed" && (
                      <Alert variant="error" title="Preflight failed">
                        Preflight checks did not pass. Review the deployment errors for details.
                      </Alert>
                    )}
                    {redeployPreflightEnabled && preflightStepStatus === "pending" && (
                      <Alert variant="info" title="Preflight queued">
                        {includeBuildStep
                          ? "Preflight checks will run after the bundle build completes."
                          : "Preflight checks will run after bundle preparation completes."}
                      </Alert>
                    )}
                    <PreflightChecksPanel
                      checksToDisplay={preflightChecks}
                      actionLoading={preflightActionLoading}
                      onAction={handlePreflightAction}
                      context="redeploy"
                    />

                    {showPreflightDiskModal && (
                      <DiskUsageModal
                        usage={preflightDiskUsage}
                        loading={preflightActionLoading === "disk_free:show_disk"}
                        onClose={() => setShowPreflightDiskModal(false)}
                        onCleanup={handlePreflightCleanup}
                        cleanupLoading={preflightCleanupLoading}
                      />
                    )}

                    {showPreflightPortModal && (
                      <PortStopModal
                        bindings={preflightPortBindings}
                        selections={preflightPortSelections}
                        loading={preflightActionLoading === "ports_80_443:stop_ports"}
                        onToggleService={togglePreflightPortService}
                        onTogglePID={togglePreflightPortPID}
                        onConfirm={handlePreflightPortStop}
                        onClose={() => setShowPreflightPortModal(false)}
                      />
                    )}
                  </div>
                )}

                {displayedStepId === "deploy" && (
                  <div className="rounded-lg border border-white/10 bg-slate-900/60 p-4 space-y-3">
                    <p className="text-xs uppercase tracking-wide text-slate-500">Deployment</p>
                    {redeployActive ? (
                      <DeploymentProgressView
                        progress={redeployProgress}
                        isConnected={redeployConnected}
                        connectionError={redeployConnectionError}
                      />
                    ) : executeMutation.isPending ? (
                      <div className="flex items-center justify-center py-6 text-slate-400">
                        <Loader2 className="h-5 w-5 animate-spin mr-2" />
                        Starting deployment...
                      </div>
                    ) : (
                      <Alert variant="info" title="Ready to deploy">
                        Start the redeploy to stream live deployment progress here.
                      </Alert>
                    )}
                  </div>
                )}
              </div>
            </div>
            )}
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
