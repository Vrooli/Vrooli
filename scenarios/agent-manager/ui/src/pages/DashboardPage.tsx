import { useState } from "react";
import {
  Activity,
  AlertCircle,
  Bot,
  CheckCircle2,
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
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { ScrollArea } from "../components/ui/scroll-area";
import { probeRunner } from "../hooks/useApi";
import { formatRelativeTime } from "../lib/utils";
import type { AgentProfile, HealthStatus, ProbeResult, Run, RunnerType, Task } from "../types";

interface DashboardPageProps {
  health: HealthStatus | null;
  profiles: AgentProfile[];
  tasks: Task[];
  runs: Run[];
  onRefresh: () => void;
}

export function DashboardPage({
  health,
  profiles,
  tasks,
  runs,
  onRefresh,
}: DashboardPageProps) {
  const activeRuns = runs.filter(
    (r) => r.status === "running" || r.status === "starting"
  );
  const pendingReview = runs.filter((r) => r.status === "needs_review");
  const recentRuns = [...runs]
    .sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())
    .slice(0, 5);

  return (
    <div className="space-y-6">
      {/* Header with refresh */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold">Dashboard</h2>
          <p className="text-sm text-muted-foreground">
            Overview of agent orchestration system status
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={onRefresh} className="gap-2">
          <RefreshCw className="h-4 w-4" />
          Refresh
        </Button>
      </div>

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
          <CardContent className="space-y-4">
            {/* Sandbox Status */}
            <HealthItem
              name="Workspace Sandbox"
              available={health?.dependencies?.sandbox?.connected ?? false}
              message={health?.dependencies?.sandbox?.error}
            />

            {/* Runner Status */}
            {health?.dependencies?.runners &&
              Object.entries(health.dependencies.runners).map(([name, runner]) => (
                <HealthItem
                  key={name}
                  name={formatRunnerName(name)}
                  available={runner.connected}
                  message={runner.error}
                  runnerType={name as RunnerType}
                />
              ))}

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
          <CardContent>
            <ScrollArea className="h-[280px]">
              {recentRuns.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                  <Bot className="h-12 w-12 mb-2 opacity-50" />
                  <p className="text-sm">No runs yet</p>
                  <p className="text-xs">Create a task and start a run to see activity</p>
                </div>
              ) : (
                <div className="space-y-3">
                  {recentRuns.map((run) => (
                    <RunActivityItem key={run.id} run={run} tasks={tasks} />
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
          <CardContent>
            <div className="space-y-2">
              {pendingReview.map((run) => {
                const task = tasks.find((t) => t.id === run.taskId);
                return (
                  <div
                    key={run.id}
                    className="flex items-center justify-between rounded-lg border border-border bg-card/50 p-3"
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
function formatRunnerName(name: string): string {
  return name
    .split("-")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
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
      `Runner: ${probeResult.runnerType}`,
      `Status: ${probeResult.success ? "Success" : "Failed"}`,
      `Message: ${probeResult.message}`,
      `Duration: ${probeResult.durationMs}ms`,
      probeResult.response ? `Response: ${probeResult.response}` : null,
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
      setProbeResult({
        runnerType,
        success: false,
        message: (err as Error).message,
        durationMs: 0,
      });
    } finally {
      setProbing(false);
    }
  };

  const handleDismissProbe = () => {
    setProbeResult(null);
  };

  return (
    <div className="flex flex-col gap-2 rounded-lg border border-border bg-card/50 p-3">
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
              <span className="text-muted-foreground">{probeResult.durationMs}ms</span>
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
          {probeResult.response && (
            <p className="mt-1 font-mono text-[10px] opacity-80 break-all whitespace-pre-wrap max-h-24 overflow-y-auto" title={probeResult.response}>
              {probeResult.response}
            </p>
          )}
          {!probeResult.success && probeResult.message && (
            <p className="mt-1 opacity-80">{probeResult.message}</p>
          )}
        </div>
      )}
    </div>
  );
}

function RunActivityItem({ run, tasks }: { run: Run; tasks: Task[] }) {
  const task = tasks.find((t) => t.id === run.taskId);

  return (
    <div className="flex items-center justify-between rounded-lg border border-border bg-card/50 p-3">
      <div className="flex items-center gap-3">
        <RunStatusIcon status={run.status} />
        <div>
          <p className="font-medium text-sm">{task?.title || "Unknown Task"}</p>
          <p className="text-xs text-muted-foreground">
            {formatRelativeTime(run.createdAt)}
          </p>
        </div>
      </div>
      <Badge variant={run.status as "running" | "complete" | "failed" | "pending"}>
        {run.status.replace("_", " ")}
      </Badge>
    </div>
  );
}

function RunStatusIcon({ status }: { status: string }) {
  switch (status) {
    case "complete":
    case "approved":
      return <CheckCircle2 className="h-5 w-5 text-success" />;
    case "failed":
    case "rejected":
      return <XCircle className="h-5 w-5 text-destructive" />;
    case "running":
    case "starting":
      return <Activity className="h-5 w-5 text-primary animate-pulse" />;
    case "needs_review":
      return <Clock className="h-5 w-5 text-warning" />;
    default:
      return <Clock className="h-5 w-5 text-muted-foreground" />;
  }
}
