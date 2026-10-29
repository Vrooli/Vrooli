import { useState } from "react";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { create } from "@bufbuild/protobuf";
import { timestampMs } from "@bufbuild/protobuf/wkt";
import {
  Activity,
  AlertCircle,
  CheckCircle2,
  ChevronDown,
  Clock,
  Copy,
  Check,
  Loader2,
  RefreshCw,
  Server,
  Shield,
  Zap,
  XCircle,
} from "lucide-react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { DataState } from "../components/feedback/DataState";
import { ErrorNotice } from "../components/feedback/ErrorNotice";
import { probeRunner } from "../hooks/useApi";
import type { DashboardRunCounts } from "../hooks/useApi";
import { useClipboardFeedback } from "../hooks/useClipboardFeedback";
import { useCollapsiblePanel } from "../hooks/useCollapsiblePanel";
import { formatHyphenatedLabel, statusTone } from "../lib/display";
import { jsonValueToPlain, runnerTypeFromSlug, runnerTypeLabel } from "../lib/utils";
import type { ProbeResult, Run, RunnerType, HealthResponse, JsonValue } from "../types";
import { RunStatus } from "../types";
import { ProbeResultSchema } from "@vrooli/proto-types/agent-manager/v1/domain/run_pb";
import { formatStandardRelativeTime } from "../lib/dateTime";
import { reviewFileCount } from "../lib/reviewFileCount";
import { runHeading } from "../lib/runPresentation";
import { DashboardUsageStrip } from "../features/usage/DashboardUsageStrip";
import { healthView } from "../lib/healthState";

interface DashboardPageProps {
  health: HealthResponse | null;
  healthError?: string | null;
  /** Why a reachable API reports itself degraded. */
  healthReason?: string;
  runs: Run[];
  runsLoading?: boolean;
  runsError?: string | null;
  /** API-counted live and windowed totals; absent while loading or unavailable. */
  counts?: DashboardRunCounts | null;
  onRefresh: () => void;
  onNavigateToRun?: (runId: string, tab?: string) => void;
}

export function DashboardPage({
  health,
  healthError,
  healthReason,
  runs,
  runsLoading = false,
  runsError,
  counts,
  onRefresh,
  onNavigateToRun,
}: DashboardPageProps) {
  const pendingReview = runs.filter((r) => r.status === RunStatus.NEEDS_REVIEW);
  const recentRuns = [...runs]
    .filter((r) => r.status !== RunStatus.NEEDS_REVIEW)
    .sort((a, b) => {
      const aTime = a.createdAt ? timestampMs(a.createdAt) : 0;
      const bTime = b.createdAt ? timestampMs(b.createdAt) : 0;
      return bTime - aTime;
    })
    .slice(0, 5);
  // Counts come from the API (true totals, not the loaded page); an unavailable
  // count renders as unknown rather than as a misleading page length.
  const activeCount = counts?.active;
  const reviewCount = counts?.review;
  const totalRuns = counts?.windowTotal;

  const healthPanel = useCollapsiblePanel({ storageKey: "dashboard.health" });
  const healthState = healthView(health, healthError, healthReason);

  // Compute health summary
  const dependencies = health?.dependencies ?? {};
  const sandboxDep = parseDependency(dependencies["sandbox"]);
  const runnerDeps = Object.entries(dependencies)
    .filter(([name]) => name.startsWith("runner_"))
    .map(([name, value]) => {
      const runnerKey = name.replace("runner_", "");
      return {
        name: formatRunnerName(runnerKey),
        status: parseDependency(value),
        runnerType: runnerTypeFromSlug(runnerKey),
      };
    });
  // A degraded start-up answer carries no dependency map; rows are shown only when reported.
  const hasDependencies = Object.keys(dependencies).length > 0;
  const allHealthy =
    health !== null &&
    sandboxDep?.status === "healthy" &&
    runnerDeps.every((r) => r.status?.status === "healthy");
  const unhealthyCount =
    (sandboxDep && sandboxDep.status !== "healthy" ? 1 : 0) +
    runnerDeps.filter((r) => r.status?.status !== "healthy").length;

  return (
    <div className="h-full overflow-y-auto overflow-x-hidden space-y-3 sm:space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Dashboard</h2>
        <Button variant="ghost" size="icon" onClick={onRefresh} title="Refresh">
          <RefreshCw className="h-4 w-4" />
        </Button>
      </div>

      <DashboardUsageStrip />

      {/* Compact Stats */}
      <div className="grid grid-cols-3 gap-2">
        <CompactStatCard
          title="Active"
          value={activeCount ?? "—"}
          icon={<Activity className="h-4 w-4" />}
          variant={(activeCount ?? 0) > 0 ? "primary" : "muted"}
        />
        <CompactStatCard
          title="Review"
          value={reviewCount ?? "—"}
          icon={<Clock className="h-4 w-4" />}
          variant={(reviewCount ?? 0) > 0 ? "warning" : "muted"}
        />
        <CompactStatCard
          title="Runs · 7d"
          value={totalRuns ?? "—"}
          icon={<Server className="h-4 w-4" />}
          variant="muted"
        />
      </div>

      {/* Runs Awaiting Review — promoted to top */}
      {pendingReview.length > 0 && (
        <Card className="border-l-2 border-l-warning overflow-hidden">
          <CardHeader className="px-4 py-3">
            <CardTitle className="flex items-center gap-2 text-base text-warning">
              <Clock className="h-4 w-4" />
              Awaiting Review ({pendingReview.length})
            </CardTitle>
            <CardDescription>
              Completed runs needing approval before changes are applied
            </CardDescription>
          </CardHeader>
          <CardContent className="p-0">
            <div>
              {pendingReview.map((run) => {
                const fileCount = reviewFileCount(run);
                return (
                  <div
                    key={run.id}
                    className="flex items-center justify-between px-4 py-2.5 border-b border-border last:border-b-0 cursor-pointer hover:bg-muted/50 transition-colors"
                    onClick={() => onNavigateToRun?.(run.id, "diff")}
                    role="button"
                    tabIndex={0}
                    onKeyDown={(e) => e.key === "Enter" && onNavigateToRun?.(run.id, "diff")}
                  >
                    <div className="min-w-0 flex-1 mr-3">
                      <p className="font-medium text-sm truncate">{runHeading(run).title}</p>
                      <p className="text-xs text-muted-foreground">
                        {fileCount.available ? `${fileCount.count} files changed` : "File count unavailable"} | {formatStandardRelativeTime(run.endedAt)}
                      </p>
                    </div>
                    <StatusBadge tone="warning" className="flex-shrink-0">Needs Review</StatusBadge>
                  </div>
                );
              })}
            </div>
          </CardContent>
        </Card>
      )}

      {/* System Health & Recent Activity */}
      <div className="grid gap-3 sm:gap-4 lg:grid-cols-2 min-w-0">
        {/* System Health — collapsible */}
        <Card className="overflow-hidden min-w-0">
          <CardHeader
            className="px-4 py-3 cursor-pointer select-none"
            onClick={healthPanel.toggle}
          >
            <CardTitle className="flex items-center justify-between text-base">
              <span className="flex items-center gap-2">
                <Shield className="h-4 w-4" />
                System Health
              </span>
              <ChevronDown
                className={`h-4 w-4 text-muted-foreground transition-transform duration-200 ${
                  healthPanel.isCollapsed ? "" : "rotate-180"
                }`}
              />
            </CardTitle>
          </CardHeader>

          {/* Collapsed summary */}
          {healthPanel.isCollapsed && health && hasDependencies && (
            <div className="px-4 pb-3">
              {allHealthy ? (
                <div className="flex items-center gap-2 text-sm text-success">
                  <CheckCircle2 className="h-4 w-4" />
                  All systems operational
                </div>
              ) : (
                <div className="flex items-center gap-2 text-sm text-destructive">
                  <AlertCircle className="h-4 w-4" />
                  {unhealthyCount} {unhealthyCount === 1 ? "issue" : "issues"} detected
                </div>
              )}
            </div>
          )}

          {/* Expanded health items */}
          <div
            className={`overflow-hidden transition-all duration-200 ${
              healthPanel.isCollapsed ? "max-h-0" : "max-h-[500px]"
            }`}
          >
            {health && hasDependencies && <CardContent className="p-0">
              <HealthItem
                name="Workspace Sandbox"
                available={sandboxDep?.status === "healthy"}
                message={sandboxDep?.error}
              />
              {runnerDeps.map((runner) => (
                <HealthItem
                  key={runner.name}
                  name={runner.name}
                  available={runner.status?.status === "healthy"}
                  message={runner.status?.error}
                  runnerType={runner.runnerType}
                />
              ))}
            </CardContent>}
          </div>

          {healthState.kind !== "healthy" && (
            <div className="px-4 pb-3">
              {healthState.kind === "unreachable" && healthError ? (
                <ErrorNotice compact error={healthError} subject="Health" onRetry={onRefresh} />
              ) : healthState.kind === "degraded" ? (
                <p role="status" data-testid="health-degraded" className="flex min-w-0 items-start gap-2 rounded-md border border-warning/40 bg-warning/5 px-3 py-2 text-sm">
                  <AlertCircle aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0 text-warning" />
                  <span className="min-w-0 flex-1 break-words">
                    <span className="font-medium">Agent Manager is degraded.</span>{" "}
                    <span className="text-muted-foreground">{healthState.reason}</span>
                  </span>
                </p>
              ) : (
                <p role="status" className="text-sm text-muted-foreground">Loading health status…</p>
              )}
            </div>
          )}
        </Card>

        {/* Recent Activity */}
        <Card className="overflow-hidden min-w-0">
          <CardHeader className="px-4 py-3">
            <CardTitle className="flex items-center gap-2 text-base">
              <Activity className="h-4 w-4" />
              Recent Activity
            </CardTitle>
            <CardDescription>Latest run executions</CardDescription>
          </CardHeader>
          <CardContent className="p-0">
            <div className="max-h-[260px] overflow-y-auto">
              <DataState
                compact
                subject="Recent runs"
                loading={runsLoading}
                error={runsError}
                isEmpty={recentRuns.length === 0}
                onRetry={onRefresh}
                empty={<EmptyState density="compact" title="No runs yet" description="Start one with Quick Run or from a task." />}
              >
                {recentRuns.map((run) => (
                  <RunActivityItem key={run.id} run={run} onClick={() => onNavigateToRun?.(run.id)} />
                ))}
              </DataState>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function CompactStatCard({
  title,
  value,
  icon,
  variant,
}: {
  title: string;
  value: number | string;
  icon: React.ReactNode;
  variant: "primary" | "warning" | "muted";
}) {
  const accentColor = {
    primary: "border-l-primary",
    warning: "border-l-warning",
    muted: "border-l-border",
  }[variant];

  return (
    <div
      className={`flex items-center gap-2.5 rounded-md border border-border bg-card px-3 py-2 border-l-2 ${accentColor}`}
    >
      <div className="text-muted-foreground">{icon}</div>
      <div className="min-w-0">
        <p className="text-xl font-bold leading-none">{value}</p>
        <p className="text-[11px] text-muted-foreground truncate">{title}</p>
      </div>
    </div>
  );
}

function parseDependency(value?: JsonValue): { status: string; error?: string } | null {
  const parsed = jsonValueToPlain(value) as Record<string, unknown> | undefined;
  if (!parsed) return null;
  const status = typeof parsed.status === "string" ? parsed.status : "unknown";
  const error = typeof parsed.error === "string" ? parsed.error : undefined;
  return { status, error };
}

function formatRunnerName(name: string): string {
  const runnerType = runnerTypeFromSlug(name);
  if (runnerType !== undefined) {
    return runnerTypeLabel(runnerType);
  }
  return formatHyphenatedLabel(name);
}

function runStatusLabel(status: RunStatus): string {
  switch (status) {
    case RunStatus.PENDING:
      return "pending";
    case RunStatus.STARTING:
      return "starting";
    case RunStatus.RUNNING:
      return "running";
    case RunStatus.NEEDS_REVIEW:
      return "needs_review";
    case RunStatus.COMPLETE:
      return "complete";
    case RunStatus.FAILED:
      return "failed";
    case RunStatus.CANCELLED:
      return "cancelled";
    case RunStatus.PARKED:
      return "parked";
    default:
      return "pending";
  }
}

function HealthItem({
  name,
  available,
  message,
  runnerType,
}: {
  name: string;
  available: boolean;
  message?: string;
  runnerType?: RunnerType;
}) {
  const { status: messageCopyStatus, copyText: copyMessage } = useClipboardFeedback();
  const { status: probeCopyStatus, copyText: copyProbe } = useClipboardFeedback();
  const [probing, setProbing] = useState(false);
  const [probeResult, setProbeResult] = useState<ProbeResult | null>(null);
  const probeDetails = probeResult?.details ?? {};
  const probeMessage =
    probeResult?.error ||
    (probeResult?.success ? "Probe succeeded" : "Probe failed");
  const probeLatencyMs =
    typeof probeResult?.latencyMs === "bigint"
      ? Number(probeResult.latencyMs)
      : Number(probeResult?.latencyMs ?? 0);

  const handleCopy = async () => {
    if (!message) return;
    const result = await copyMessage(message);
    if (!result.ok) console.error("Failed to copy:", result.error);
  };

  const handleCopyProbeResult = async () => {
    if (!probeResult) return;
    const copyText = [
      `Runner: ${runnerType ? runnerTypeLabel(runnerType) : "Unknown"}`,
      `Status: ${probeResult.success ? "Success" : "Failed"}`,
      `Message: ${probeMessage}`,
      `Duration: ${probeLatencyMs}ms`,
      typeof probeDetails.response === "string" ? `Response: ${probeDetails.response}` : null,
    ]
      .filter(Boolean)
      .join("\n");

    const result = await copyProbe(copyText);
    if (!result.ok) console.error("Failed to copy probe result:", result.error);
  };

  const handleProbe = async () => {
    if (!runnerType || probing) return;
    setProbing(true);
    setProbeResult(null);
    try {
      const result = await probeRunner(runnerType);
      setProbeResult(result);
    } catch (err) {
      setProbeResult(create(ProbeResultSchema, {
        success: false,
        latencyMs: 0n,
        error: (err as Error).message,
        details: {},
      }));
    } finally {
      setProbing(false);
    }
  };

  const handleDismissProbe = () => {
    setProbeResult(null);
  };

  return (
    <div className="flex flex-col gap-2 px-4 py-2.5 border-b border-border last:border-b-0">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3 min-w-0 flex-1">
          {available ? (
            <CheckCircle2 className="h-4 w-4 text-success flex-shrink-0" />
          ) : (
            <XCircle className="h-4 w-4 text-destructive flex-shrink-0" />
          )}
          <div className="min-w-0 flex-1">
            <p className="font-medium text-sm">{name}</p>
            {message && (
              <div className="flex items-center gap-2">
                <p className="text-xs text-muted-foreground max-w-md truncate" title={message}>
                  {message}
                </p>
                <button
                  onClick={handleCopy}
                  className="flex-shrink-0 p-1 rounded hover:bg-muted transition-colors"
                  title="Copy full error message"
                  aria-label="Copy error message"
                >
                  {messageCopyStatus === "copied" ? (
                    <Check className="h-3 w-3 text-success" />
                  ) : (
                    <Copy className="h-3 w-3 text-muted-foreground hover:text-foreground" />
                  )}
                </button>
              </div>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2 flex-shrink-0 ml-2">
          {runnerType && (
            <Button
              variant="secondary"
              size="sm"
              onClick={handleProbe}
              disabled={probing}
              className="h-7 px-2 text-xs gap-1"
              title="Send a test request to verify the agent responds"
            >
              {probing ? (
                <Loader2 className="h-3 w-3 animate-spin" />
              ) : (
                <Zap className="h-3 w-3" />
              )}
              Probe
            </Button>
          )}
          <StatusBadge tone={available ? "success" : "danger"}>
            {available ? "Available" : "Unavailable"}
          </StatusBadge>
        </div>
      </div>
      {probeResult && (
        <div
          className={`text-xs p-2 rounded border ${
            probeResult.success
              ? "bg-success/10 border-success/20 text-success"
              : "bg-destructive/10 border-destructive/20 text-destructive"
          }`}
        >
          <div className="flex items-center justify-between gap-2">
            <span className="font-medium">
              {probeResult.success ? "✓ Probe successful" : "✗ Probe failed"}
            </span>
            <div className="flex items-center gap-1">
              <span className="text-muted-foreground">{probeLatencyMs}ms</span>
              <button
                onClick={handleCopyProbeResult}
                className="p-1 rounded hover:bg-black/10 transition-colors"
                title="Copy probe result"
                aria-label="Copy probe result"
              >
                {probeCopyStatus === "copied" ? (
                  <Check className="h-3 w-3" />
                ) : (
                  <Copy className="h-3 w-3 opacity-60 hover:opacity-100" />
                )}
              </button>
              <button
                onClick={handleDismissProbe}
                className="p-1 rounded hover:bg-black/10 transition-colors"
                title="Dismiss"
                aria-label="Dismiss probe result"
              >
                <XCircle className="h-3 w-3 opacity-60 hover:opacity-100" />
              </button>
            </div>
          </div>
          {typeof probeDetails.response === "string" && (
            <p className="mt-1 font-mono text-[10px] opacity-80 break-all whitespace-pre-wrap max-h-24 overflow-y-auto" title={probeDetails.response}>
              {probeDetails.response}
            </p>
          )}
          {!probeResult.success && probeMessage && (
            <p className="mt-1 opacity-80">{probeMessage}</p>
          )}
        </div>
      )}
    </div>
  );
}

function RunActivityItem({ run, onClick }: { run: Run; onClick?: () => void }) {
  const { title, reference } = runHeading(run);
  return (
    <div
      className="flex items-center justify-between px-4 py-2.5 border-b border-border last:border-b-0 cursor-pointer hover:bg-muted/50 transition-colors"
      onClick={onClick}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => e.key === "Enter" && onClick?.()}
    >
      <div className="flex items-center gap-3 min-w-0 flex-1 mr-3">
        <RunStatusIcon status={run.status} />
        <div className="min-w-0">
          <p className="font-medium text-sm truncate" title={title}>{title}</p>
          <p className="text-xs text-muted-foreground">
            {formatStandardRelativeTime(run.createdAt)}
            <span className="ms-2 font-mono text-[11px] opacity-70">{reference}</span>
          </p>
        </div>
      </div>
      <StatusBadge className="flex-shrink-0" tone={statusTone(runStatusLabel(run.status))}>
        {runStatusLabel(run.status).replace("_", " ").replace(/\b\w/g, (c) => c.toUpperCase())}
      </StatusBadge>
    </div>
  );
}

function RunStatusIcon({ status }: { status: RunStatus }) {
  switch (status) {
    case RunStatus.COMPLETE:
      return <CheckCircle2 className="h-4 w-4 text-success flex-shrink-0" />;
    case RunStatus.FAILED:
      return <XCircle className="h-4 w-4 text-destructive flex-shrink-0" />;
    case RunStatus.RUNNING:
    case RunStatus.STARTING:
      return <Activity className="h-4 w-4 text-primary animate-pulse flex-shrink-0" />;
    case RunStatus.NEEDS_REVIEW:
      return <Clock className="h-4 w-4 text-warning flex-shrink-0" />;
    default:
      return <Clock className="h-4 w-4 text-muted-foreground flex-shrink-0" />;
  }
}
