import { Link } from "react-router-dom";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import { timestampMs } from "@bufbuild/protobuf/wkt";
import { Activity, AlertCircle, CheckCircle2, Clock, RefreshCw, XCircle } from "lucide-react";
import { DataState } from "../components/feedback/DataState";
import { ErrorNotice } from "../components/feedback/ErrorNotice";
import type { DashboardRunCounts } from "../hooks/useApi";
import { statusTone } from "../lib/display";
import { formatCode, formatCount, MISSING } from "../lib/format";
import { healthView } from "../lib/healthState";
import { jsonValueToPlain } from "../lib/utils";
import { formatStandardRelativeTime } from "../lib/dateTime";
import { reviewFileCount } from "../lib/reviewFileCount";
import { runHeading } from "../lib/runPresentation";
import { DashboardUsageStrip } from "../features/usage/DashboardUsageStrip";
import { UsageLeverLine } from "./dashboard/UsageLeverLine";
import type { HealthResponse, Run } from "../types";
import { RunStatus } from "../types";

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

const RUN_STATUS_WORDS: Partial<Record<RunStatus, string>> = {
  [RunStatus.PENDING]: "pending",
  [RunStatus.STARTING]: "starting",
  [RunStatus.RUNNING]: "running",
  [RunStatus.NEEDS_REVIEW]: "needs_review",
  [RunStatus.COMPLETE]: "complete",
  [RunStatus.FAILED]: "failed",
  [RunStatus.CANCELLED]: "cancelled",
  [RunStatus.PARKED]: "parked",
};

const runStatusLabel = (status: RunStatus) => RUN_STATUS_WORDS[status] ?? "pending";

/** Counts the runner and sandbox dependencies a health document reports as not healthy. */
function unhealthyDependencies(health: HealthResponse | null): number | undefined {
  const entries = Object.entries(health?.dependencies ?? {}).filter(([name]) => name === "sandbox" || name.startsWith("runner_"));
  if (entries.length === 0) return undefined;
  return entries.filter(([, value]) => (jsonValueToPlain(value) as { status?: unknown } | undefined)?.status !== "healthy").length;
}

function HealthSummary({ health, healthError, healthReason, onRetry }: Pick<DashboardPageProps, "health" | "healthError" | "healthReason"> & { onRetry: () => void }) {
  const state = healthView(health, healthError, healthReason);
  const unhealthy = state.kind === "healthy" ? unhealthyDependencies(health) : undefined;
  const tone = state.kind === "unreachable" || unhealthy ? "text-destructive" : state.kind === "degraded" ? "text-warning" : state.kind === "checking" ? "text-muted-foreground" : "text-success";
  const text = state.kind === "healthy"
    ? unhealthy ? `${formatCount(unhealthy)} ${unhealthy === 1 ? "check needs" : "checks need"} attention` : "All systems operational"
    : state.kind === "checking" ? "Checking health…" : state.kind === "degraded" ? `Agent Manager is degraded. ${state.reason}` : state.reason || "Agent Manager is unreachable.";
  if (state.kind === "unreachable" && healthError) {
    return <ErrorNotice compact error={healthError} subject="Health" onRetry={onRetry} />;
  }
  return (
    <p role="status" data-testid="dashboard-health-summary" className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1 px-3 text-sm">
      <span className={`flex min-w-0 items-center gap-2 ${tone}`}>
        {state.kind === "healthy" && !unhealthy ? <CheckCircle2 aria-hidden="true" className="h-4 w-4 shrink-0" /> : <AlertCircle aria-hidden="true" className="h-4 w-4 shrink-0" />}
        <span className="min-w-0 break-words">{text}</span>
      </span>
      <Link to="/observability" className="inline-flex min-h-11 items-center font-medium text-primary hover:underline sm:min-h-8">Health →</Link>
    </p>
  );
}

function CountTile({ title, value, to, linkLabel, note }: { title: string; value?: number; to: string; linkLabel: string; note?: string }) {
  return (
    <Card className="flex flex-col gap-1 p-4" data-testid="dashboard-count-tile">
      <span className="text-xs text-muted-foreground">{title}</span>
      <b className="text-2xl font-bold leading-none tabular-nums">{value === undefined ? MISSING : formatCount(value)}</b>
      {note ? <span className="text-xs text-muted-foreground">{note}</span> : null}
      <Link to={to} className="inline-flex min-h-11 items-center text-xs font-medium text-primary hover:underline sm:min-h-6">{linkLabel}</Link>
    </Card>
  );
}

function RunStatusIcon({ status }: { status: RunStatus }) {
  switch (status) {
    case RunStatus.COMPLETE:
      return <CheckCircle2 aria-hidden="true" className="h-4 w-4 shrink-0 text-success" />;
    case RunStatus.FAILED:
      return <XCircle aria-hidden="true" className="h-4 w-4 shrink-0 text-destructive" />;
    case RunStatus.RUNNING:
    case RunStatus.STARTING:
      return <Activity aria-hidden="true" className="h-4 w-4 shrink-0 animate-pulse text-primary" />;
    case RunStatus.NEEDS_REVIEW:
      return <Clock aria-hidden="true" className="h-4 w-4 shrink-0 text-warning" />;
    default:
      return <Clock aria-hidden="true" className="h-4 w-4 shrink-0 text-muted-foreground" />;
  }
}

function RunRow({ run, detail, onOpen, badge }: { run: Run; detail: string; onOpen?: () => void; badge: { tone: Parameters<typeof StatusBadge>[0]["tone"]; label: string } }) {
  const { title, reference } = runHeading(run);
  return (
    <button type="button" onClick={onOpen} className="flex min-h-11 w-full items-center justify-between gap-3 border-b border-border px-4 py-2.5 text-start last:border-b-0 hover:bg-muted/50">
      <span className="flex min-w-0 flex-1 items-center gap-3">
        <RunStatusIcon status={run.status} />
        <span className="min-w-0">
          <span className="block truncate text-sm font-medium" title={title}>{title}</span>
          <span className="block text-xs text-muted-foreground">{detail}<span className="ms-2 font-mono text-[11px] opacity-70">{reference}</span></span>
        </span>
      </span>
      <StatusBadge className="shrink-0" tone={badge.tone}>{badge.label}</StatusBadge>
    </button>
  );
}

export function DashboardPage({ health, healthError, healthReason, runs, runsLoading = false, runsError, counts, onRefresh, onNavigateToRun }: DashboardPageProps) {
  const pendingReview = runs.filter((run) => run.status === RunStatus.NEEDS_REVIEW);
  // `runs` is the API's top-level Agent Manager list, so operator sessions and sub-agents never crowd it out.
  const recentRuns = runs
    .filter((run) => run.status !== RunStatus.NEEDS_REVIEW)
    .sort((a, b) => (b.createdAt ? timestampMs(b.createdAt) : 0) - (a.createdAt ? timestampMs(a.createdAt) : 0))
    .slice(0, 5);

  return (
    <div className="mx-auto flex h-full w-full max-w-6xl flex-col gap-3 overflow-y-auto overflow-x-hidden sm:gap-4">
      <PageHeader
        title="Dashboard"
        actions={
          <Button variant="secondary" size="icon" onClick={onRefresh} aria-label="Refresh" title="Refresh" className="min-h-11 min-w-11 sm:min-h-9 sm:min-w-9">
            <RefreshCw aria-hidden="true" className="h-4 w-4" />
          </Button>
        }
      />

      <DashboardUsageStrip />
      <UsageLeverLine />

      <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
        <CountTile title="Running now" value={counts?.active} to={`/runs?status=${RunStatus.RUNNING}`} linkLabel="View runs →" />
        <CountTile title="Waiting for your approval" value={counts?.review} to={`/runs?status=${RunStatus.NEEDS_REVIEW}`} linkLabel="Review →" />
        <CountTile title="Agent Manager runs · 7 days" value={counts?.windowTotal} to="/runs?origin=agent_manager" linkLabel="All runs →" note={counts ? `+ ${formatCount(counts.operatorSessions)} operator sessions` : undefined} />
      </div>

      <HealthSummary health={health} healthError={healthError} healthReason={healthReason} onRetry={onRefresh} />

      {pendingReview.length > 0 && (
        <Card className="overflow-hidden border-l-2 border-l-warning">
          <CardHeader className="px-4 py-3">
            <CardTitle className="flex items-center gap-2 text-base text-warning"><Clock aria-hidden="true" className="h-4 w-4" />Awaiting review ({formatCount(pendingReview.length)})</CardTitle>
            <CardDescription>Completed runs needing approval before changes are applied</CardDescription>
          </CardHeader>
          <CardContent className="p-0">
            {pendingReview.map((run) => {
              const fileCount = reviewFileCount(run);
              return (
                <RunRow
                  key={run.id}
                  run={run}
                  detail={`${fileCount.available ? `${formatCount(fileCount.count)} files changed` : "File count unavailable"} | ${formatStandardRelativeTime(run.endedAt)}`}
                  onOpen={() => onNavigateToRun?.(run.id, "diff")}
                  badge={{ tone: "warning", label: "Needs review" }}
                />
              );
            })}
          </CardContent>
        </Card>
      )}

      <Card className="min-w-0 overflow-hidden">
        <CardHeader className="px-4 py-3">
          <CardTitle className="flex items-center gap-2 text-base"><Activity aria-hidden="true" className="h-4 w-4" />Recent activity</CardTitle>
          <CardDescription>Latest runs started through Agent Manager</CardDescription>
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
              empty={<EmptyState density="compact" title="No recent Agent Manager runs" description="Start one with Quick Run or from a task. Sessions you run yourself are listed under Runs." />}
            >
              {recentRuns.map((run) => (
                <RunRow
                  key={run.id}
                  run={run}
                  detail={formatStandardRelativeTime(run.createdAt)}
                  onOpen={() => onNavigateToRun?.(run.id)}
                  badge={{ tone: statusTone(runStatusLabel(run.status)), label: formatCode(runStatusLabel(run.status)) }}
                />
              ))}
            </DataState>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
