import { useCallback, useState } from "react";
import { Routes, Route, useNavigate, useLocation, Navigate } from "react-router-dom";
import {
  Activity,
  AlertCircle,
  Bot,
  CheckCircle2,
  ClipboardList,
  Play,
  Settings2,
  Wifi,
  WifiOff,
} from "lucide-react";
import { Badge } from "./components/ui/badge";
import { Button } from "./components/ui/button";
import { Card, CardContent } from "./components/ui/card";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "./components/ui/dialog";
import { Input } from "./components/ui/input";
import { Label } from "./components/ui/label";
import { Tabs, TabsList, TabsTrigger } from "./components/ui/tabs";
import { useHealth, useMaintenance, useProfiles, useRuns, useRunners, useTasks } from "./hooks/useApi";
import { useWebSocket, type WebSocketMessage } from "./hooks/useWebSocket";
import { DashboardPage } from "./pages/DashboardPage";
import { ProfilesPage } from "./pages/ProfilesPage";
import { TasksPage } from "./pages/TasksPage";
import { RunsPage } from "./pages/RunsPage";
import { HealthStatus, RunStatus } from "./types";
import { PurgeTarget } from "@vrooli/proto-types/agent-manager/v1/api/service_pb";
import { formatDate, jsonValueToPlain } from "./lib/utils";

export default function App() {
  const navigate = useNavigate();
  const location = useLocation();
  const health = useHealth();
  const profiles = useProfiles();
  const tasks = useTasks();
  const runs = useRuns();
  const runners = useRunners();
  const maintenance = useMaintenance();

  // Derive active tab from current path
  const getActiveTab = useCallback(() => {
    const path = location.pathname;
    if (path.startsWith("/profiles")) return "profiles";
    if (path.startsWith("/tasks")) return "tasks";
    if (path.startsWith("/runs")) return "runs";
    return "dashboard";
  }, [location.pathname]);

  const activeTab = getActiveTab();

  // WebSocket connection for real-time updates
  const handleWebSocketMessage = useCallback(
    (message: WebSocketMessage) => {
      console.log("[WS] Received:", message.type);

      switch (message.type) {
        case "run_event":
        case "run_status":
          // Refresh runs when we receive run updates
          runs.refetch();
          break;
        case "task_status":
          // Refresh tasks when we receive task updates
          tasks.refetch();
          break;
        case "connected":
          // Subscribe to all events on connect
          ws.subscribeAll();
          break;
      }
    },
    [runs, tasks]
  );

  const ws = useWebSocket({
    enabled: true,
    onMessage: handleWebSocketMessage,
  });

  const handleTabChange = useCallback((value: string) => {
    navigate(`/${value === "dashboard" ? "" : value}`);
  }, [navigate]);

  const isHealthy = health.data?.status === HealthStatus.HEALTHY;
  const [statusOpen, setStatusOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [purgePattern, setPurgePattern] = useState("^test-.*");
  const [purgeError, setPurgeError] = useState<string | null>(null);
  const [purgeLoading, setPurgeLoading] = useState(false);
  const [purgeConfirmOpen, setPurgeConfirmOpen] = useState(false);
  const [purgeConfirmText, setPurgeConfirmText] = useState("");
  const [purgePreview, setPurgePreview] = useState<{ profiles: number; tasks: number; runs: number } | null>(null);
  const [purgeTargets, setPurgeTargets] = useState<PurgeTarget[]>([]);
  const [purgeActionLabel, setPurgeActionLabel] = useState("");
  const wsLabel =
    ws.status === "connected" ? "Live" : ws.status === "connecting" ? "Connecting" : "Offline";
  const healthLabel = health.data ? (isHealthy ? "Healthy" : "Degraded") : "Unknown";
  const statusVariant =
    !isHealthy || ws.status === "disconnected"
      ? "destructive"
      : ws.status === "connecting"
      ? "secondary"
      : "success";
  const statusText = `${healthLabel} • ${wsLabel}`;
  const dependencyEntries = Object.entries(health.data?.dependencies ?? {});
  const metricEntries = Object.entries(health.data?.metrics ?? {});

  const handlePurgePreview = useCallback(
    async (targets: PurgeTarget[], label: string) => {
      setPurgeError(null);
      setPurgeLoading(true);
      try {
        const counts = await maintenance.previewPurge(purgePattern, targets);
        setPurgePreview({
          profiles: counts.profiles ?? 0,
          tasks: counts.tasks ?? 0,
          runs: counts.runs ?? 0,
        });
        setPurgeTargets(targets);
        setPurgeActionLabel(label);
        setPurgeConfirmText("");
        setPurgeConfirmOpen(true);
      } catch (err) {
        setPurgeError((err as Error).message);
      } finally {
        setPurgeLoading(false);
      }
    },
    [maintenance, purgePattern]
  );

  const handlePurgeExecute = useCallback(async () => {
    if (purgeConfirmText.trim() !== "DELETE") {
      return;
    }
    setPurgeError(null);
    setPurgeLoading(true);
    try {
      await maintenance.executePurge(purgePattern, purgeTargets);
      setPurgeConfirmOpen(false);
      setSettingsOpen(false);
      profiles.refetch();
      tasks.refetch();
      runs.refetch();
    } catch (err) {
      setPurgeError((err as Error).message);
    } finally {
      setPurgeLoading(false);
    }
  }, [maintenance, purgeConfirmText, purgePattern, purgeTargets, profiles, runs, tasks]);

  return (
    <div className="min-h-screen bg-transparent text-foreground">
      {/* Header */}
      <div className="relative overflow-hidden border-b border-border/50 backdrop-blur-sm">
        <div className="pointer-events-none absolute inset-0 opacity-40">
          <div className="absolute -left-32 top-24 h-72 w-72 rounded-full bg-primary/30 blur-3xl" />
          <div className="absolute -top-20 right-0 h-96 w-96 rounded-full bg-purple-500/20 blur-3xl" />
        </div>
        <header className="z-10 px-6 pb-4 pt-8 sm:px-10">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div className="space-y-2">
              <div className="flex items-center gap-3">
                <Bot className="h-8 w-8 text-primary" />
                <Badge className="uppercase tracking-wide" variant="secondary">
                  Agent Manager
                </Badge>
                <Badge
                  variant={statusVariant}
                  className="gap-1 cursor-pointer"
                  onClick={() => setStatusOpen(true)}
                  role="button"
                  tabIndex={0}
                  aria-label="Open status details"
                  onKeyDown={(event) => {
                    if (event.key === "Enter" || event.key === " ") {
                      event.preventDefault();
                      setStatusOpen(true);
                    }
                  }}
                >
                  {isHealthy ? <CheckCircle2 className="h-3 w-3" /> : <AlertCircle className="h-3 w-3" />}
                  {ws.status === "connected" ? <Wifi className="h-3 w-3" /> : <WifiOff className="h-3 w-3" />}
                  {statusText}
                </Badge>
              </div>
              <h1 className="text-2xl font-semibold text-foreground sm:text-3xl">
                AI Agent Orchestration & Control
              </h1>
              <p className="max-w-2xl text-sm text-muted-foreground">
                Manage agent profiles, create tasks, execute runs in sandboxed environments,
                and review changes with approve/reject workflows.
              </p>
            </div>

            {/* Quick Stats */}
            <div className="flex flex-wrap items-center gap-4">
              <QuickStat
                icon={<Settings2 className="h-4 w-4" />}
                label="Profiles"
                value={profiles.data?.length ?? 0}
              />
              <QuickStat
                icon={<ClipboardList className="h-4 w-4" />}
                label="Tasks"
                value={tasks.data?.length ?? 0}
              />
              <QuickStat
                icon={<Play className="h-4 w-4" />}
                label="Runs"
                value={runs.data?.length ?? 0}
              />
              <QuickStat
                icon={<Activity className="h-4 w-4" />}
                label="Active"
                value={
                  runs.data?.filter(
                    (r) => r.status === RunStatus.RUNNING || r.status === RunStatus.STARTING
                  ).length ?? 0
                }
              />
              <Button
                variant="outline"
                size="sm"
                onClick={() => setSettingsOpen(true)}
                className="gap-2"
              >
                <Settings2 className="h-4 w-4" />
                Settings
              </Button>
            </div>
          </div>
        </header>
      </div>

      <Dialog open={statusOpen} onOpenChange={setStatusOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Status Details</DialogTitle>
          </DialogHeader>
          <DialogBody className="space-y-4">
            <div className="flex flex-wrap gap-2">
              <Badge variant={isHealthy ? "success" : "destructive"} className="gap-1">
                {isHealthy ? <CheckCircle2 className="h-3 w-3" /> : <AlertCircle className="h-3 w-3" />}
                {healthLabel}
              </Badge>
              <Badge
                variant={ws.status === "connected" ? "success" : ws.status === "connecting" ? "secondary" : "outline"}
                className="gap-1"
              >
                {ws.status === "connected" ? <Wifi className="h-3 w-3" /> : <WifiOff className="h-3 w-3" />}
                {wsLabel}
              </Badge>
            </div>

            <div className="grid gap-3 text-sm">
              <div className="flex items-center justify-between text-muted-foreground">
                <span>Service</span>
                <span className="text-foreground">{health.data?.service || "Unknown"}</span>
              </div>
              <div className="flex items-center justify-between text-muted-foreground">
                <span>Readiness</span>
                <span className="text-foreground">{health.data?.readiness ? "Ready" : "Not ready"}</span>
              </div>
              <div className="flex items-center justify-between text-muted-foreground">
                <span>Version</span>
                <span className="text-foreground">{health.data?.version || "Unknown"}</span>
              </div>
              <div className="flex items-center justify-between text-muted-foreground">
                <span>Timestamp</span>
                <span className="text-foreground">{formatDate(health.data?.timestamp)}</span>
              </div>
            </div>

            {dependencyEntries.length > 0 && (
              <div className="space-y-2">
                <p className="text-sm font-semibold">Dependencies</p>
                <div className="space-y-2 text-xs text-muted-foreground">
                  {dependencyEntries.map(([name, value]) => {
                    const plain = jsonValueToPlain(value) as Record<string, unknown> | undefined;
                    const status = plain?.status as string | undefined;
                    const latency = plain?.latency_ms as number | undefined;
                    const error = plain?.error as string | undefined;
                    return (
                      <div key={name} className="flex flex-col gap-1 rounded-md border border-border/60 p-2">
                        <div className="flex items-center justify-between text-foreground">
                          <span className="font-semibold">{name}</span>
                          <span>{status || "unknown"}</span>
                        </div>
                        <div className="flex items-center justify-between">
                          <span>Latency</span>
                          <span>{latency !== undefined ? `${latency}ms` : "n/a"}</span>
                        </div>
                        {error && (
                          <div className="text-destructive">Error: {error}</div>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            )}

            {metricEntries.length > 0 && (
              <div className="space-y-2">
                <p className="text-sm font-semibold">Metrics</p>
                <div className="grid gap-2 text-xs text-muted-foreground sm:grid-cols-2">
                  {metricEntries.map(([name, value]) => (
                    <div key={name} className="flex items-center justify-between rounded-md border border-border/60 p-2">
                      <span>{name}</span>
                      <span className="text-foreground">{String(jsonValueToPlain(value) ?? "n/a")}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {health.error && (
              <Card className="border-destructive/40 bg-destructive/10 text-xs">
                <CardContent className="flex items-start gap-2 py-3">
                  <AlertCircle className="h-4 w-4 text-destructive" aria-hidden="true" />
                  <div>
                    <p className="font-semibold text-destructive">API Error</p>
                    <p className="text-destructive/80">{health.error}</p>
                  </div>
                </CardContent>
              </Card>
            )}
          </DialogBody>
        </DialogContent>
      </Dialog>

      <Dialog open={settingsOpen} onOpenChange={setSettingsOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Settings</DialogTitle>
            <DialogDescription>Manage maintenance actions and future controls.</DialogDescription>
          </DialogHeader>
          <DialogBody className="space-y-6">
            <div className="space-y-3">
              <h3 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">Danger Zone</h3>
              <Card className="border-destructive/40 bg-destructive/5">
                <CardContent className="space-y-4 py-4">
                  <div className="space-y-2">
                    <Label htmlFor="purgePattern">Regex Pattern</Label>
                    <Input
                      id="purgePattern"
                      value={purgePattern}
                      onChange={(event) => setPurgePattern(event.target.value)}
                      placeholder="^test-.*"
                    />
                    <p className="text-xs text-muted-foreground">
                      Matches profile keys (profiles), titles (tasks), and tags (runs; falls back to run ID if empty). Use <code>.*</code> to match everything.
                    </p>
                  </div>

                  <div className="grid gap-2 sm:grid-cols-2">
                    <Button
                      variant="destructive"
                      onClick={() => handlePurgePreview([PurgeTarget.PROFILES], "Delete Profiles")}
                      disabled={purgeLoading}
                    >
                      Delete Profiles
                    </Button>
                    <Button
                      variant="destructive"
                      onClick={() => handlePurgePreview([PurgeTarget.TASKS], "Delete Tasks")}
                      disabled={purgeLoading}
                    >
                      Delete Tasks
                    </Button>
                    <Button
                      variant="destructive"
                      onClick={() => handlePurgePreview([PurgeTarget.RUNS], "Delete Runs")}
                      disabled={purgeLoading}
                    >
                      Delete Runs
                    </Button>
                    <Button
                      variant="destructive"
                      onClick={() =>
                        handlePurgePreview(
                          [
                            PurgeTarget.PROFILES,
                            PurgeTarget.TASKS,
                            PurgeTarget.RUNS,
                          ],
                          "Delete All"
                        )
                      }
                      disabled={purgeLoading}
                    >
                      Delete All
                    </Button>
                  </div>
                  {purgeError && (
                    <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                      {purgeError}
                    </div>
                  )}
                </CardContent>
              </Card>
            </div>

            <div className="space-y-2 text-sm text-muted-foreground">
              <h3 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">Service Controls</h3>
              <p>Future settings like pausing new runs or offline mode will live here.</p>
            </div>
          </DialogBody>
          <DialogFooter>
            <Button variant="outline" onClick={() => setSettingsOpen(false)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={purgeConfirmOpen} onOpenChange={setPurgeConfirmOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{purgeActionLabel}</DialogTitle>
            <DialogDescription>
              This action deletes data that matches the regex pattern. This cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogBody className="space-y-4">
            <div className="space-y-2 text-sm">
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground">Pattern</span>
                <span className="font-mono text-xs">{purgePattern}</span>
              </div>
              <div className="flex items-center justify-between">
                <span>Profiles</span>
                <span>{purgePreview?.profiles ?? 0}</span>
              </div>
              <div className="flex items-center justify-between">
                <span>Tasks</span>
                <span>{purgePreview?.tasks ?? 0}</span>
              </div>
              <div className="flex items-center justify-between">
                <span>Runs</span>
                <span>{purgePreview?.runs ?? 0}</span>
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="purgeConfirm">Type DELETE to confirm</Label>
              <Input
                id="purgeConfirm"
                value={purgeConfirmText}
                onChange={(event) => setPurgeConfirmText(event.target.value)}
                placeholder="DELETE"
              />
            </div>
            {purgeError && (
              <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {purgeError}
              </div>
            )}
          </DialogBody>
          <DialogFooter>
            <Button variant="outline" onClick={() => setPurgeConfirmOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handlePurgeExecute}
              disabled={purgeLoading || purgeConfirmText.trim() !== "DELETE"}
            >
              {purgeLoading ? "Deleting..." : "Confirm Delete"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Main Content */}
      <main className="flex flex-1 flex-col gap-6 px-6 py-6 sm:px-10">
        {health.error && (
          <Card className="border border-destructive/40 bg-destructive/10 text-sm">
            <CardContent className="flex items-center gap-3 py-4">
              <AlertCircle className="h-4 w-4 text-destructive" aria-hidden="true" />
              <div>
                <p className="font-semibold text-destructive">API Connection Error</p>
                <p className="text-xs text-destructive/80">{health.error}</p>
              </div>
            </CardContent>
          </Card>
        )}

        <Tabs value={activeTab} onValueChange={handleTabChange} className="w-full">
          <TabsList className="mb-6 grid w-full max-w-[600px] grid-cols-4">
            <TabsTrigger value="dashboard" className="gap-2">
              <Activity className="h-4 w-4" />
              Dashboard
            </TabsTrigger>
            <TabsTrigger value="profiles" className="gap-2">
              <Settings2 className="h-4 w-4" />
              Profiles
            </TabsTrigger>
            <TabsTrigger value="tasks" className="gap-2">
              <ClipboardList className="h-4 w-4" />
              Tasks
            </TabsTrigger>
            <TabsTrigger value="runs" className="gap-2">
              <Play className="h-4 w-4" />
              Runs
            </TabsTrigger>
          </TabsList>

          <Routes>
            <Route
              path="/"
              element={
                <DashboardPage
                  health={health.data}
                  profiles={profiles.data || []}
                  tasks={tasks.data || []}
                  runs={runs.data || []}
                  runners={runners.data ?? undefined}
                  onRefresh={() => {
                    health.refetch();
                    profiles.refetch();
                    tasks.refetch();
                    runs.refetch();
                  }}
                  onCreateTask={tasks.createTask}
                  onCreateRun={runs.createRun}
                  onRunCreated={(run) => {
                    runs.refetch();
                    tasks.refetch();
                    navigate(`/runs/${run.id}`);
                  }}
                  onNavigateToRun={(runId) => navigate(`/runs/${runId}`)}
                />
              }
            />
            <Route
              path="/profiles"
              element={
                <ProfilesPage
                  profiles={profiles.data || []}
                  loading={profiles.loading}
                  error={profiles.error}
                  onCreateProfile={profiles.createProfile}
                  onUpdateProfile={profiles.updateProfile}
                  onDeleteProfile={profiles.deleteProfile}
                  onRefresh={profiles.refetch}
                  runners={runners.data ?? undefined}
                />
              }
            />
            <Route
              path="/tasks"
              element={
                <TasksPage
                  tasks={tasks.data || []}
                  profiles={profiles.data || []}
                  loading={tasks.loading}
                  error={tasks.error}
                  onCreateTask={tasks.createTask}
                  onCancelTask={tasks.cancelTask}
                  onDeleteTask={tasks.deleteTask}
                  onCreateRun={runs.createRun}
                  onCreateProfile={profiles.createProfile}
                  onRefresh={tasks.refetch}
                  runners={runners.data ?? undefined}
                />
              }
            />
            <Route
              path="/runs/:runId?"
              element={
                <RunsPage
                  runs={runs.data || []}
                  tasks={tasks.data || []}
                  profiles={profiles.data || []}
                  loading={runs.loading}
                  error={runs.error}
                  onStopRun={runs.stopRun}
                  onRetryRun={runs.retryRun}
                  onGetEvents={runs.getRunEvents}
                  onGetDiff={runs.getRunDiff}
                  onApproveRun={runs.approveRun}
                  onRejectRun={runs.rejectRun}
                  onRefresh={runs.refetch}
                  wsSubscribe={ws.subscribe}
                  wsUnsubscribe={ws.unsubscribe}
                  wsAddMessageHandler={ws.addMessageHandler}
                  wsRemoveMessageHandler={ws.removeMessageHandler}
                />
              }
            />
            {/* Redirect unknown paths to dashboard */}
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </Tabs>
      </main>
    </div>
  );
}

function QuickStat({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode;
  label: string;
  value: number;
}) {
  return (
    <div className="flex items-center gap-2 rounded-lg bg-card/50 px-4 py-2 border border-border/50">
      <div className="text-muted-foreground">{icon}</div>
      <div>
        <p className="text-xs text-muted-foreground">{label}</p>
        <p className="text-lg font-semibold">{value}</p>
      </div>
    </div>
  );
}
