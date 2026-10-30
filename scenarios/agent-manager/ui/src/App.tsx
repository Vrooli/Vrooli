import { lazy, Profiler, Suspense, useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { Routes, Route, useNavigate, useLocation, useSearchParams, Navigate } from "react-router-dom";
import { Play, Settings2 } from "lucide-react";
import { AppShell, type AppShellNavItem } from "@vrooli/react-component-library/AppShell/2";
import { useHealth, useRolePolicyCatalog, useProfiles, useRuns, useTasks, useDashboardRunCounts, type RunOrigin } from "./hooks/useApi";
import { useWebSocket, type WebSocketMessage } from "./hooks/useWebSocket";
import { useRunEventStore } from "./hooks/useRunEventStore";
import type { Run, RunEvent } from "./types";
import { QueryProvider } from "./providers/QueryProvider";
import { NAV_DESTINATIONS, sectionForPath } from "./components/layout/navigation";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Badge } from "./components/ui/badge";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { jsonValueToPlain } from "./lib/utils";
import { onProfilerRender } from "./lib/profiler";
import { healthView } from "./lib/healthState";

const DashboardPage = lazy(async () => ({ default: (await import("./pages/DashboardPage")).DashboardPage }));
const ProfilesPage = lazy(async () => ({ default: (await import("./pages/ProfilesPage")).ProfilesPage }));
const TasksPage = lazy(async () => ({ default: (await import("./pages/TasksPage")).TasksPage }));
const RunsPage = lazy(async () => ({ default: (await import("./pages/RunsPage")).RunsPage }));
const WorkflowsPage = lazy(async () => ({ default: (await import("./pages/WorkflowsPage")).WorkflowsPage }));
const WatchesPage = lazy(async () => ({ default: (await import("./pages/WatchesPage")).WatchesPage }));
const EffortsPage = lazy(async () => ({ default: (await import("./pages/EffortsPage")).EffortsPage }));
const UsagePage = lazy(async () => ({ default: (await import("./features/usage")).UsagePage }));
const HealthPage = lazy(async () => ({ default: (await import("./features/health")).HealthPage }));
const InvestigationsPage = lazy(async () => ({ default: (await import("./pages/InvestigationsPage")).InvestigationsPage }));
const StatusDialog = lazy(async () => ({ default: (await import("./components/dialogs/StatusDialog")).StatusDialog }));
const SettingsDialog = lazy(async () => ({ default: (await import("./components/dialogs/SettingsDialog")).SettingsDialog }));
const QuickRunDialog = lazy(async () => ({ default: (await import("./components/QuickRunDialog")).QuickRunDialog }));

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
  // Runs show their profile's name, so the (small) profile list loads there too.
  const needsProfileData = path.startsWith("/profiles") || path.startsWith("/runs") || quickRunOpen;
  // Master-detail pages own their panes edge to edge; the shell adds no gutter.
  const mainMode = ["/runs", "/tasks", "/profiles"].some((prefix) => path.startsWith(prefix)) ? "fill" : "scroll";
  const needsRunnerData = path.startsWith("/profiles") || settingsOpen || quickRunOpen;
  const needsTaskData = path.startsWith("/tasks") || path.startsWith("/runs");
  const runsLimit = isDashboardRoute ? 40 : undefined;

  const health = useHealth();
  const healthState = healthView(health.data, health.error, health.reason);
  const profiles = useProfiles({ enabled: needsProfileData });
  const tasks = useTasks({ enabled: needsTaskData });
  // The runs list is filtered by who started the run, server-side. The origin is held here, not in the URL, because
  // selecting a run navigates to /runs/<id> and would drop the query; a link such as /runs?origin=agent_manager sets it.
  const [searchParams] = useSearchParams();
  const linkedOrigin = searchParams.get("origin") as RunOrigin | null;
  const [runsOrigin, setRunsOrigin] = useState<RunOrigin>(linkedOrigin ?? "");
  useEffect(() => {
    if (linkedOrigin !== null) setRunsOrigin(linkedOrigin);
  }, [linkedOrigin]);
  // The dashboard shows what was started through Agent Manager.
  const listOrigin: RunOrigin = isDashboardRoute ? "agent_manager" : path.startsWith("/runs") ? runsOrigin : "";
  const runs = useRuns({ limit: runsLimit, origin: listOrigin });
  const dashboardCounts = useDashboardRunCounts({ enabled: isDashboardRoute });
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

  const activeSection = sectionForPath(location.pathname);
  const shellItems: AppShellNavItem[] = NAV_DESTINATIONS.map(({ id, label, shortLabel, href, icon, phoneTab }) => ({
    id, label, shortLabel, href, icon,
    mobile: phoneTab,
    current: id === activeSection,
    testId: `agent-manager-nav-${id}`,
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
            dashboardCounts.refetch();
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
    [isDashboardRoute, needsTaskData, runEventStore.actions, dashboardCounts, runs, tasks]
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

  const handleNavigate = useCallback((item: AppShellNavItem) => navigate(item.href), [navigate]);

  const handlePurgeComplete = useCallback(() => {
    profiles.refetch();
    if (needsTaskData) {
      tasks.refetch();
    }
    runs.refetch();
    if (isDashboardRoute) {
      dashboardCounts.refetch();
    }
  }, [isDashboardRoute, needsTaskData, profiles, dashboardCounts, tasks, runs]);

  const pageFallback = (
    <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
      Loading...
    </div>
  );

  return (
    <QueryProvider>
      <AppShell
        brand="Mizar"
        brandMark={<img src="/public/logo.svg" alt="" aria-hidden="true" className="h-5 w-5" />}
        items={shellItems}
        density="sidebar"
        mobileNav="menu-tabs"
        mainMode={mainMode}
        onNavigate={handleNavigate}
        renderLink={(item, props) => <a {...props} href={item.href} />}
        header={
          <div className="flex min-w-0 flex-1 items-center justify-between gap-3">
            <div className="flex min-w-0 items-center gap-2">
              <span className="truncate text-sm font-semibold md:hidden">Mizar</span>
              <Badge
                variant={healthState.kind === "unreachable" || ws.status === "error" || ws.status === "disconnected" ? "destructive" : healthState.kind === "degraded" ? "warning" : healthState.kind === "checking" || ws.status === "connecting" ? "secondary" : "success"}
                className="min-h-11 cursor-pointer gap-1 sm:min-h-8 whitespace-nowrap text-xs"
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
                {healthState.label} · {ws.status === "connected" ? "Live" : ws.status === "connecting" ? "Connecting" : ws.status === "error" ? "Error" : "Offline"}
              </Badge>
            </div>
            <Button type="button" size="sm" className="min-h-11 min-w-11 shrink-0" aria-label="Quick Run" onClick={() => setQuickRunOpen(true)}>
              <Play aria-hidden="true" className="h-4 w-4" />
              <span className="hidden sm:inline">Quick Run</span>
            </Button>
          </div>
        }
        utility={
          <Button type="button" variant="ghost" size="sm" className="min-h-11 min-w-11 shrink-0 md:w-full md:justify-start" aria-label="Settings" onClick={() => setSettingsOpen(true)}>
            <Settings2 aria-hidden="true" className="h-4 w-4" />
            <span className="hidden md:inline">Settings</span>
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
              healthReason={health.reason}
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
                  dashboardCounts.refetch();
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
                        runsLoading={runs.loading}
                        runsError={runs.error}
                        healthError={health.error}
                        healthReason={health.reason}
                        counts={dashboardCounts.data}
                        onRefresh={() => {
                          health.refetch();
                          runs.refetch();
                          dashboardCounts.refetch();
                        }}
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
                        origin={runsOrigin}
                        onOriginChange={setRunsOrigin}
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
              path="/efforts"
              element={
                <Suspense fallback={pageFallback}>
                  <ErrorBoundary section="Efforts">
                    <ProfiledPage id="EffortsPage"><EffortsPage /></ProfiledPage>
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
              path="/usage"
              element={
                <Suspense fallback={pageFallback}>
                  <ErrorBoundary section="Usage">
                    <ProfiledPage id="UsagePage">
                      <UsagePage />
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
            <Route path="/investigations" element={<Suspense fallback={pageFallback}><ErrorBoundary section="Investigations"><ProfiledPage id="InvestigationsPage"><InvestigationsPage /></ProfiledPage></ErrorBoundary></Suspense>} />
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
