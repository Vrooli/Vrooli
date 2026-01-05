import { useState } from "react";
import { create } from "@bufbuild/protobuf";
import { timestampMs } from "@bufbuild/protobuf/wkt";
import {
  Activity,
  AlertCircle,
  Bot,
  CheckCircle2,
  Clock,
  Copy,
  Check,
  Loader2,
  Play,
  RefreshCw,
  Server,
  Shield,
  Zap,
  XCircle,
} from "lucide-react";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { ScrollArea } from "../components/ui/scroll-area";
import { QuickRunDialog } from "../components/QuickRunDialog";
import { probeRunner } from "../hooks/useApi";
import { formatRelativeTime, jsonValueToPlain, runnerTypeFromSlug, runnerTypeLabel } from "../lib/utils";
import type { AgentProfile, ModelRegistry, ProbeResult, Run, RunnerStatus, RunnerType, Task, TaskFormData, RunFormData, HealthResponse, JsonValue } from "../types";
import { RunStatus } from "../types";
import { ProbeResultSchema } from "@vrooli/proto-types/agent-manager/v1/domain/run_pb";

interface DashboardPageProps {
  health: HealthResponse | null;
  profiles: AgentProfile[];
  tasks: Task[];
  runs: Run[];
  runners?: Record<string, RunnerStatus>;
  modelRegistry?: ModelRegistry;
  onRefresh: () => void;
  onCreateTask: (task: TaskFormData) => Promise<Task>;
  onCreateRun: (run: RunFormData) => Promise<Run>;
  onRunCreated?: (run: Run) => void;
  onNavigateToRun?: (runId: string) => void;
}

export function DashboardPage({
  health,
  profiles,
  tasks,
  runs,
  runners,
  modelRegistry,
  onRefresh,
  onCreateTask,
  onCreateRun,
  onRunCreated,
  onNavigateToRun,
}: DashboardPageProps) {
  const [showQuickRun, setShowQuickRun] = useState(false);
  const activeRuns = runs.filter(
    (r) => r.status === RunStatus.RUNNING || r.status === RunStatus.STARTING
  );
  const pendingReview = runs.filter((r) => r.status === RunStatus.NEEDS_REVIEW);
  const recentRuns = [...runs]
    .sort((a, b) => {
      const aTime = a.createdAt ? timestampMs(a.createdAt) : 0;
      const bTime = b.createdAt ? timestampMs(b.createdAt) : 0;
      return bTime - aTime;
    })
    .slice(0, 5);

  return (
    <div className="h-full overflow-y-auto px-4 py-4 sm:px-6 lg:px-10 space-y-6">
      {/* Header with refresh */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold">Dashboard</h2>
          <p className="text-sm text-muted-foreground">
            Overview of agent orchestration system status
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={onRefresh} className="gap-2">
            <RefreshCw className="h-4 w-4" />
            Refresh
          </Button>
          <Button size="sm" onClick={() => setShowQuickRun(true)} className="gap-2">
            <Play className="h-4 w-4" />
            Quick Run
          </Button>
        </div>
      </div>

      {/* Quick Run Dialog */}
      <QuickRunDialog
        open={showQuickRun}
        onOpenChange={setShowQuickRun}
        profiles={profiles}
        runners={runners}
        modelRegistry={modelRegistry}
        onCreateTask={onCreateTask}
        onCreateRun={onCreateRun}
        onRunCreated={onRunCreated}
      />

      {/* Status Grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <StatusCard
          title="Active Runs"
          value={activeRuns.length}
          icon={<Activity className="h-5 w-5" />}
          description="Currently executing"
          variant={activeRuns.length > 0 ? "primary" : "muted"}
        />
        <StatusCard
          title="Pending Review"
          value={pendingReview.length}
          icon={<Clock className="h-5 w-5" />}
          description="Awaiting approval"
          variant={pendingReview.length > 0 ? "warning" : "muted"}
        />
        <StatusCard
          title="Agent Profiles"
          value={profiles.length}
          icon={<Bot className="h-5 w-5" />}
          description="Configured agents"
          variant="muted"
        />
        <StatusCard
          title="Total Tasks"
          value={tasks.length}
          icon={<Server className="h-5 w-5" />}
          description="All time"
          variant="muted"
        />
      </div>

      {/* System Health & Recent Activity */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* System Health */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Shield className="h-5 w-5" />
              System Health
            </CardTitle>
            <CardDescription>
              Component status and availability
            </CardDescription>
          </CardHeader>
          <CardContent className="p-0">
            {(() => {
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

              return (
                <>
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
                </>
              );
            })()}

            {!health && (
              <div className="flex items-center gap-2 text-muted-foreground">
                <AlertCircle className="h-4 w-4" />
                <span className="text-sm">Loading health status...</span>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Recent Activity */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Activity className="h-5 w-5" />
              Recent Activity
            </CardTitle>
            <CardDescription>Latest run executions</CardDescription>
          </CardHeader>
          <CardContent className="p-0">
            <ScrollArea className="h-[280px]">
              {recentRuns.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                  <Bot className="h-12 w-12 mb-2 opacity-50" />
                  <p className="text-sm">No runs yet</p>
                  <p className="text-xs">Create a task and start a run to see activity</p>
                </div>
              ) : (
                <div>
                  {recentRuns.map((run) => (
                    <RunActivityItem
                      key={run.id}
                      run={run}
                      tasks={tasks}
                      onClick={() => onNavigateToRun?.(run.id)}
                    />
                  ))}
                </div>
              )}
            </ScrollArea>
          </CardContent>
        </Card>
      </div>

      {/* Pending Review Section */}
      {pendingReview.length > 0 && (
        <Card className="border-warning/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-warning">
              <Clock className="h-5 w-5" />
              Runs Awaiting Review ({pendingReview.length})
            </CardTitle>
            <CardDescription>
              These runs have completed and need your approval before changes are applied
            </CardDescription>
          </CardHeader>
          <CardContent className="p-0">
            <div>
              {pendingReview.map((run) => {
                const task = tasks.find((t) => t.id === run.taskId);
                return (
                  <div
                    key={run.id}
                    className="flex items-center justify-between px-4 py-3 border-b border-border last:border-b-0 cursor-pointer hover:bg-muted/50 transition-colors"
                    onClick={() => onNavigateToRun?.(run.id)}
                    role="button"
                    tabIndex={0}
                    onKeyDown={(e) => e.key === "Enter" && onNavigateToRun?.(run.id)}
                  >
                    <div>
                      <p className="font-medium">{task?.title || "Unknown Task"}</p>
                      <p className="text-xs text-muted-foreground">
                        {run.changedFiles} files changed | {formatRelativeTime(run.endedAt)}
                      </p>
                    </div>
                    <Badge variant="needs_review">Needs Review</Badge>
                  </div>
                );
              })}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

function StatusCard({
  title,
  value,
  icon,
  description,
  variant,
}: {
  title: string;
  value: number;
  icon: React.ReactNode;
  description: string;
  variant: "primary" | "warning" | "muted";
}) {
  const variantStyles = {
    primary: "border-primary/50 bg-primary/5",
    warning: "border-warning/50 bg-warning/5",
    muted: "border-border",
  };

  return (
    <Card className={variantStyles[variant]}>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          {title}
        </CardTitle>
        <div className="text-muted-foreground">{icon}</div>
      </CardHeader>
      <CardContent>
        <div className="text-3xl font-bold">{value}</div>
        <p className="text-xs text-muted-foreground">{description}</p>
      </CardContent>
    </Card>
  );
}

// Format runner name for display (e.g., "claude-code" -> "Claude Code")
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
  return name
    .split("-")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
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
  const [copied, setCopied] = useState(false);
  const [probeCopied, setProbeCopied] = useState(false);
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
    try {
      await navigator.clipboard.writeText(message);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error("Failed to copy:", err);
    }
  };

  const handleCopyProbeResult = async () => {
    if (!probeResult) return;
    // Build a comprehensive copy string with all probe info
    const copyText = [
      `Runner: ${runnerType ? runnerTypeLabel(runnerType) : "Unknown"}`,
      `Status: ${probeResult.success ? "Success" : "Failed"}`,
      `Message: ${probeMessage}`,
      `Duration: ${probeLatencyMs}ms`,
      typeof probeDetails.response === "string" ? `Response: ${probeDetails.response}` : null,
    ]
      .filter(Boolean)
      .join("\n");

    try {
      await navigator.clipboard.writeText(copyText);
      setProbeCopied(true);
      setTimeout(() => setProbeCopied(false), 2000);
    } catch (err) {
      console.error("Failed to copy probe result:", err);
    }
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
    <div className="flex flex-col gap-2 px-4 py-3 border-b border-border last:border-b-0">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3 min-w-0 flex-1">
          {available ? (
            <CheckCircle2 className="h-5 w-5 text-success flex-shrink-0" />
          ) : (
            <XCircle className="h-5 w-5 text-destructive flex-shrink-0" />
          )}
          <div className="min-w-0 flex-1">
            <p className="font-medium">{name}</p>
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
                  {copied ? (
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
              variant="outline"
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
          <Badge variant={available ? "success" : "destructive"}>
            {available ? "Available" : "Unavailable"}
          </Badge>
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
                {probeCopied ? (
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

function RunActivityItem({
  run,
  tasks,
  onClick,
}: {
  run: Run;
  tasks: Task[];
  onClick?: () => void;
}) {
  const task = tasks.find((t) => t.id === run.taskId);

  return (
    <div
      className="flex items-center justify-between px-4 py-3 border-b border-border last:border-b-0 cursor-pointer hover:bg-muted/50 transition-colors"
      onClick={onClick}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => e.key === "Enter" && onClick?.()}
    >
      <div className="flex items-center gap-3">
        <RunStatusIcon status={run.status} />
        <div>
          <p className="font-medium text-sm">{task?.title || "Unknown Task"}</p>
          <p className="text-xs text-muted-foreground">
            {formatRelativeTime(run.createdAt)}
          </p>
        </div>
      </div>
      <Badge
        variant={
          runStatusLabel(run.status) as
            | "pending"
            | "starting"
            | "running"
            | "needs_review"
            | "complete"
            | "failed"
            | "cancelled"
        }
      >
        {runStatusLabel(run.status).replace("_", " ")}
      </Badge>
    </div>
  );
}

function RunStatusIcon({ status }: { status: RunStatus }) {
  switch (status) {
    case RunStatus.COMPLETE:
      return <CheckCircle2 className="h-5 w-5 text-success" />;
    case RunStatus.FAILED:
      return <XCircle className="h-5 w-5 text-destructive" />;
    case RunStatus.RUNNING:
    case RunStatus.STARTING:
      return <Activity className="h-5 w-5 text-primary animate-pulse" />;
    case RunStatus.NEEDS_REVIEW:
      return <Clock className="h-5 w-5 text-warning" />;
    default:
      return <Clock className="h-5 w-5 text-muted-foreground" />;
  }
}
