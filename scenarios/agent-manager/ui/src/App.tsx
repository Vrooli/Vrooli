import { lazy, Profiler, Suspense, useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { Routes, Route, useNavigate, useLocation, Navigate } from "react-router-dom";
import { Activity, BarChart3, Binoculars, ClipboardList, GitBranch, HeartPulse, Play, Search, Settings2, Upload } from "lucide-react";
import { AppShell, type AppShellNavItem } from "@vrooli/react-component-library/AppShell/2";
import { useHealth, useRolePolicyCatalog, useProfiles, useRuns, useTasks, useRunStatusCounts } from "./hooks/useApi";
import { useWebSocket, type WebSocketMessage } from "./hooks/useWebSocket";
import { useRunEventStore } from "./hooks/useRunEventStore";
import { HealthStatus, type Run, type RunEvent } from "./types";
import { QueryProvider } from "./providers/QueryProvider";
import type { NavSection } from "./components/layout/SideNav";
import { Button } from "./components/ui/button";
import { Badge } from "./components/ui/badge";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { jsonValueToPlain } from "./lib/utils";
import { onProfilerRender } from "./lib/profiler";

const DashboardPage = lazy(async () => ({ default: (await import("./pages/DashboardPage")).DashboardPage }));
const ProfilesPage = lazy(async () => ({ default: (await import("./pages/ProfilesPage")).ProfilesPage }));
const TasksPage = lazy(async () => ({ default: (await import("./pages/TasksPage")).TasksPage }));
const RunsPage = lazy(async () => ({ default: (await import("./pages/RunsPage")).RunsPage }));
const WorkflowsPage = lazy(async () => ({ default: (await import("./pages/WorkflowsPage")).WorkflowsPage }));
const WatchesPage = lazy(async () => ({ default: (await import("./pages/WatchesPage")).WatchesPage }));
const StatsPage = lazy(async () => ({ default: (await import("./features/stats")).StatsPage }));
const HealthPage = lazy(async () => ({ default: (await import("./features/health")).HealthPage }));
const FindingsPage = lazy(async () => ({ default: (await import("./pages/FindingsPage")).FindingsPage }));
const InvestigationsPage = lazy(async () => ({ default: (await import("./pages/InvestigationsPage")).InvestigationsPage }));
const ImportPage = lazy(async () => ({ default: (await import("./pages/ImportPage")).ImportPage }));
const StatusDialog = lazy(async () => ({ default: (await import("./components/dialogs/StatusDialog")).StatusDialog }));
const SettingsDialog = lazy(async () => ({ default: (await import("./components/dialogs/SettingsDialog")).SettingsDialog }));
const QuickRunDialog = lazy(async () => ({ default: (await import("./components/QuickRunDialog")).QuickRunDialog }));

const shellNavigation: Array<{ id: NavSection; label: string; href: string; icon: ReactNode }> = [
  { id: "dashboard", label: "Dashboard", href: "/", icon: <Activity aria-hidden="true" /> },
  { id: "tasks", label: "Tasks", href: "/tasks", icon: <ClipboardList aria-hidden="true" /> },
  { id: "runs", label: "Runs", href: "/runs", icon: <Play aria-hidden="true" /> },
  { id: "workflows", label: "Flows", href: "/workflows", icon: <GitBranch aria-hidden="true" /> },
  { id: "watches", label: "Watches", href: "/watches", icon: <Binoculars aria-hidden="true" /> },
  { id: "investigations", label: "Investigations", href: "/investigations", icon: <Search aria-hidden="true" /> },
  { id: "findings", label: "Findings", href: "/findings", icon: <ClipboardList aria-hidden="true" /> },
  { id: "stats", label: "Stats", href: "/stats", icon: <BarChart3 aria-hidden="true" /> },
  { id: "profiles", label: "Profiles", href: "/profiles", icon: <Settings2 aria-hidden="true" /> },
  { id: "health", label: "Health", href: "/observability", icon: <HeartPulse aria-hidden="true" /> },
  { id: "import", label: "Import", href: "/import", icon: <Upload aria-hidden="true" /> },
];

// AI_CHECK: AGENT_MANAGER_RENDER_PERF=2 | LAST: 2026-05-04
function ProfiledPage({ id, children }: { id: string; children: ReactNode }) {
  return (
    <Profiler id={id} onRender={onProfilerRender}>
      {children}
    </Profiler>
  );
}

export function App() {
  const navigate = useNavigate();
  const location = useLocation();
  const [statusOpen, setStatusOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [quickRunOpen, setQuickRunOpen] = useState(false);

  const path = location.pathname;
  const isDashboardRoute = path === "/";
  const needsProfileData = path.startsWith("/profiles") || quickRunOpen;
  const needsRunnerData = path.startsWith("/profiles") || settingsOpen || quickRunOpen;
  const needsTaskData = path.startsWith("/tasks") || path.startsWith("/runs");
  const runsLimit = isDashboardRoute ? 40 : undefined;

  const health = useHealth();
  const profiles = useProfiles({ enabled: needsProfileData });
  const tasks = useTasks({ enabled: needsTaskData });
  const runs = useRuns({ limit: runsLimit });
  const runStatusCounts = useRunStatusCounts({ enabled: isDashboardRoute });
	const modelPolicy = useRolePolicyCatalog({ enabled: needsRunnerData });
  const runEventStore = useRunEventStore();
  const reconciliationInFlightRef = useRef<Set<string>>(new Set());

  useEffect(() => {
    runEventStore.actions.runsSnapshotLoaded(runs.data || []);
  }, [runs.data, runEventStore.actions]);

  const mergedRuns = useMemo(() => {
    const snapshots = runEventStore.state.runsById;
    return (runs.data || []).map((run) => {
      const snapshot = snapshots[run.id];
      return snapshot ? ({ ...run, ...snapshot } as Run) : run;
    });
  }, [runs.data, runEventStore.state.runsById]);

  // Derive active section from current path
  const getActiveSection = useCallback((): NavSection => {
    const path = location.pathname;
    if (path.startsWith("/profiles")) return "profiles";
    if (path.startsWith("/tasks")) return "tasks";
    if (path.startsWith("/runs")) return "runs";
    if (path.startsWith("/workflows")) return "workflows";
    if (path.startsWith("/watches")) return "watches";
    if (path.startsWith("/stats")) return "stats";
    if (path.startsWith("/observability")) return "health";
    if (path.startsWith("/investigations")) return "investigations";
    if (path.startsWith("/findings")) return "findings";
    if (path.startsWith("/import")) return "import";
    return "dashboard";
  }, [location.pathname]);

  const activeSection = getActiveSection();

  const shellItems: AppShellNavItem[] = shellNavigation.map((item) => ({
    ...item,
    current: item.id === activeSection,
    testId: `agent-manager-nav-${item.id}`,
  }));

  const handleWebSocketMessage = useCallback(
    (message: WebSocketMessage) => {
      if (import.meta.env.DEV) {
        console.log("[WS] Received:", message.type);
      }

      switch (message.type) {
        case "run_status": {
          const statusUpdate = message.payload as Partial<Run>;
          if (statusUpdate.id) {
            runEventStore.actions.runStatusReceived({ ...statusUpdate, id: statusUpdate.id });
            void runs.getRun(statusUpdate.id)
              .then((run) => {
                runEventStore.actions.runSnapshotLoaded(run);
              })
              .catch((err) => {
                console.error(`Failed to hydrate run status update for ${statusUpdate.id}:`, err);
              });
          }
          if (isDashboardRoute) {
            runStatusCounts.refetch();
          }
          if (
            needsTaskData &&
            statusUpdate.taskId &&
            tasks.data &&
            !tasks.data.some((t) => t.id === statusUpdate.taskId)
          ) {
            tasks.refetch();
          }
          break;
        }
        case "run_event":
          runEventStore.actions.runEventReceived(message.payload as RunEvent);
          break;
        case "task_status":
          runEventStore.actions.taskStatusReceived(message.payload as { id: string });
          if (needsTaskData) {
            tasks.refetch();
          }
          break;
        case "workflow_lifecycle":
          window.dispatchEvent(new CustomEvent("agent-manager:workflow-lifecycle", { detail: message.payload }));
          break;
      }
    },
    [isDashboardRoute, needsTaskData, runEventStore.actions, runStatusCounts, runs, tasks]
  );

  const ws = useWebSocket({
    enabled: true,
    onMessage: handleWebSocketMessage,
    onStatusChange: (status) => {
      if (status === "connected") {
        runEventStore.actions.connected();
        return;
      }
      if (status === "disconnected" || status === "error") {
        runEventStore.actions.disconnected();
      }
    },
  });
  const getRunEvents = runs.getRunEvents;

  useEffect(() => {
    for (const intent of runEventStore.reconciliationIntents) {
      if (reconciliationInFlightRef.current.has(intent.runId)) {
        continue;
      }
      reconciliationInFlightRef.current.add(intent.runId);
      void (async () => {
        try {
          const events = await getRunEvents(intent.runId, {
            afterSequence: intent.afterSequence,
          });
          runEventStore.actions.eventsGapFilled(intent.runId, events);
        } catch (err) {
          console.error(`Failed to reconcile run events for ${intent.runId}:`, err);
          runEventStore.actions.clearReconciliationIntent(intent.runId);
        } finally {
          reconciliationInFlightRef.current.delete(intent.runId);
        }
      })();
    }
  }, [runEventStore.reconciliationIntents, runEventStore.actions, getRunEvents]);

  const handleSectionChange = useCallback(
    (section: NavSection) => {
      if (section === "dashboard") {
        navigate("/");
        return;
      }
      // The api-base UI server reserves `/health` for its own status JSON
      // (see packages/api-base/src/server/health.ts), so the user-facing
      // health page lives at `/observability`. The "health" NavSection
      // label stays — the route name is the only thing that differs.
      if (section === "health") {
        navigate("/observability");
        return;
      }
      navigate(`/${section}`);
    },
    [navigate]
  );

  const handlePurgeComplete = useCallback(() => {
    profiles.refetch();
    if (needsTaskData) {
      tasks.refetch();
    }
    runs.refetch();
    if (isDashboardRoute) {
      runStatusCounts.refetch();
    }
  }, [isDashboardRoute, needsTaskData, profiles, runStatusCounts, tasks, runs]);

  const pageFallback = (
    <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
      Loading...
    </div>
  );

  return (
    <QueryProvider>
      <AppShell
        brand="Agent Manager"
        brandMark={<Activity aria-hidden="true" />}
        items={shellItems}
        density="sidebar"
        mobileNav="tabs"
        mainMode="scroll"
        onNavigate={(item) => handleSectionChange(item.id as NavSection)}
        renderLink={(item, props) => <a {...props} href={item.href} />}
        header={
          <div className="flex min-h-12 items-center justify-between gap-4 border-b border-border bg-background/95 px-4 py-2 sm:px-6">
            <div className="flex min-w-0 items-center gap-3">
              <span className="truncate text-sm font-semibold sm:text-base">Agent Manager</span>
              <Badge
                variant={!health.data || health.data.status !== HealthStatus.HEALTHY || ws.status === "error" || ws.status === "disconnected" ? "destructive" : ws.status === "connecting" ? "secondary" : "success"}
                className="cursor-pointer gap-1 text-xs"
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
                {health.data?.status === HealthStatus.HEALTHY ? "Healthy" : "Degraded"} · {ws.status === "connected" ? "Live" : ws.status === "connecting" ? "Connecting" : ws.status === "error" ? "Error" : "Offline"}
              </Badge>
            </div>
            <Button type="button" size="sm" onClick={() => setQuickRunOpen(true)}>
              <Play aria-hidden="true" className="h-4 w-4" />
              <span className="hidden sm:inline">Quick Run</span>
            </Button>
          </div>
        }
        utility={
          <Button type="button" variant="ghost" size="sm" className="w-full justify-start" onClick={() => setSettingsOpen(true)}>
            <Settings2 aria-hidden="true" className="h-4 w-4" />
            Settings
          </Button>
        }
        testId="agent-manager-shell"
      >

        {statusOpen ? (
          <Suspense fallback={null}>
            <StatusDialog
              open={statusOpen}
              onOpenChange={setStatusOpen}
              health={health.data}
              healthError={health.error}
              wsStatus={ws.status}
            />
          </Suspense>
        ) : null}

        {settingsOpen ? (
          <Suspense fallback={null}>
            <SettingsDialog
              open={settingsOpen}
              onOpenChange={setSettingsOpen}
              onPurgeComplete={handlePurgeComplete}
            />
          </Suspense>
        ) : null}

        {quickRunOpen ? (
          <ErrorBoundary section="Quick Run">
            <Suspense fallback={null}>
              <QuickRunDialog
                open={quickRunOpen}
                onOpenChange={setQuickRunOpen}
                profiles={profiles.data || []}
                rolePolicyCatalog={modelPolicy.data?.catalog}
                defaultProjectRoot={(() => {
                  const raw = health.data?.metrics?.default_project_root;
                  if (!raw) return undefined;
                  const plain = jsonValueToPlain(raw);
                  return typeof plain === "string" ? plain : undefined;
                })()}
                onCreateTask={tasks.createTask}
                onCreateRun={runs.createRun}
                onRunCreated={(run) => {
                  runs.refetch();
                  if (needsTaskData) {
                    tasks.refetch();
                  }
                  runStatusCounts.refetch();
                  navigate(`/runs/${run.id}`);
                }}
              />
            </Suspense>
          </ErrorBoundary>
        ) : null}

        {/* Main Content */}
        <div className="min-h-0 h-full overflow-hidden">
          <ErrorBoundary section="Application">
            <Routes>
            <Route
              path="/"
              element={
                <ErrorBoundary section="Dashboard">
                  <Suspense fallback={pageFallback}>
                    <ProfiledPage id="DashboardPage">
                      <DashboardPage
                        health={health.data}
                        runs={mergedRuns}
                        statusCounts={runStatusCounts.data}
                        onRefresh={() => {
                          health.refetch();
                          runs.refetch();
                          runStatusCounts.refetch();
                        }}
                        onGetTask={tasks.getTask}
                        onNavigateToRun={(runId, tab) => navigate(`/runs/${runId}${tab ? `?tab=${tab}` : ""}`)}
                      />
                    </ProfiledPage>
                  </Suspense>
                </ErrorBoundary>
              }
            />
            <Route
              path="/profiles"
              element={
                <Suspense fallback={pageFallback}>
                  <ErrorBoundary section="Profiles">
                    <ProfiledPage id="ProfilesPage">
                      <ProfilesPage
                        profiles={profiles.data || []}
                        loading={profiles.loading}
                        error={profiles.error}
                        onCreateProfile={profiles.createProfile}
                        onUpdateProfile={profiles.updateProfile}
                        onDeleteProfile={profiles.deleteProfile}
                        onRefresh={profiles.refetch}
                        rolePolicyCatalog={modelPolicy.data?.catalog}
                      />
                    </ProfiledPage>
                  </ErrorBoundary>
                </Suspense>
              }
            />
            <Route
              path="/tasks"
              element={
                <Suspense fallback={pageFallback}>
                  <ErrorBoundary section="Tasks">
                    <ProfiledPage id="TasksPage">
                      <TasksPage
                        tasks={tasks.data || []}
                        profiles={profiles.data || []}
                        loading={tasks.loading}
                        error={tasks.error}
                        onCreateTask={tasks.createTask}
                        onUpdateTask={tasks.updateTask}
                        onCancelTask={tasks.cancelTask}
                        onDeleteTask={tasks.deleteTask}
                        onCreateRun={runs.createRun}
                        onCreateProfile={profiles.createProfile}
                        onRefresh={tasks.refetch}
                        rolePolicyCatalog={modelPolicy.data?.catalog}
                      />
                    </ProfiledPage>
                  </ErrorBoundary>
                </Suspense>
              }
            />
            <Route
              path="/runs/:runId?"
              element={
                <Suspense fallback={pageFallback}>
                  <ErrorBoundary section="Runs">
                    <ProfiledPage id="RunsPage">
                      <RunsPage
                        runs={mergedRuns}
                        tasks={tasks.data || []}
                        profiles={profiles.data || []}
                        loading={runs.loading}
                        error={runs.error}
                        onStopRun={runs.stopRun}
                        onDeleteRun={runs.deleteRun}
                        onRetryRun={runs.retryRun}
                        onGetRun={runs.getRun}
                        onGetEvents={runs.getRunEvents}
                        onGetDiff={runs.getRunDiff}
                        onGetTask={tasks.getTask}
                        onApproveRun={runs.approveRun}
                        onRejectRun={runs.rejectRun}
                        onPartialApproveRun={runs.partialApproveRun}
                        onStartInvestigation={runs.startTypedInvestigation}
                        onApplyInvestigation={runs.applyInvestigation}
                        onResumeFromFailedRun={runs.resumeFromFailedRun}
                        onContinueRun={runs.continueRun}
                        onDeleteRunMessage={runs.deleteRunMessage}
                        onRefresh={runs.refetch}
                        runEventStore={runEventStore}
                        wsSubscribe={ws.subscribe}
                        wsUnsubscribe={ws.unsubscribe}
                      />
                    </ProfiledPage>
                  </ErrorBoundary>
                </Suspense>
              }
            />
            <Route
              path="/workflows"
              element={
                <Suspense fallback={pageFallback}>
                  <ErrorBoundary section="Workflows">
                    <ProfiledPage id="WorkflowsPage"><WorkflowsPage /></ProfiledPage>
                  </ErrorBoundary>
                </Suspense>
              }
            />
            <Route
              path="/watches"
              element={
                <Suspense fallback={pageFallback}>
                  <ErrorBoundary section="Watches">
                    <ProfiledPage id="WatchesPage"><WatchesPage /></ProfiledPage>
                  </ErrorBoundary>
                </Suspense>
              }
            />
            <Route
              path="/stats"
              element={
                <Suspense fallback={pageFallback}>
                  <ErrorBoundary section="Stats">
                    <ProfiledPage id="StatsPage">
                      <StatsPage />
                    </ProfiledPage>
                  </ErrorBoundary>
                </Suspense>
              }
            />
            <Route
              path="/observability"
              element={
                <Suspense fallback={pageFallback}>
                  <ErrorBoundary section="Health">
                    <ProfiledPage id="HealthPage">
                      <HealthPage />
                    </ProfiledPage>
                  </ErrorBoundary>
                </Suspense>
              }
            />
            <Route path="/findings" element={<Suspense fallback={pageFallback}><ErrorBoundary section="Findings"><ProfiledPage id="FindingsPage"><FindingsPage /></ProfiledPage></ErrorBoundary></Suspense>} />
            <Route path="/investigations" element={<Suspense fallback={pageFallback}><ErrorBoundary section="Investigations"><ProfiledPage id="InvestigationsPage"><InvestigationsPage /></ProfiledPage></ErrorBoundary></Suspense>} />
            <Route path="/import" element={<Suspense fallback={pageFallback}><ErrorBoundary section="Import"><ProfiledPage id="ImportPage"><ImportPage /></ProfiledPage></ErrorBoundary></Suspense>} />
            {/* Redirect unknown paths to dashboard */}
            <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </ErrorBoundary>
        </div>
      </AppShell>
    </QueryProvider>
  );
}

export default App;
