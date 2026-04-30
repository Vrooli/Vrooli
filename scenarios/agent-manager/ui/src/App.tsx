import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Routes, Route, useNavigate, useLocation, Navigate } from "react-router-dom";
import { useHealth, useProfiles, useRuns, useRunners, useModelRegistry, useTasks, useRunStatusCounts } from "./hooks/useApi";
import { useWebSocket, type WebSocketMessage } from "./hooks/useWebSocket";
import { useRunEventStore } from "./hooks/useRunEventStore";
import type { Run, RunEvent } from "./types";
import { useIsMobile } from "./hooks/useViewportSize";
import { QueryProvider } from "./providers/QueryProvider";
import { AppHeader } from "./components/layout/AppHeader";
import { MobileNav, type NavSection } from "./components/layout/MobileNav";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { jsonValueToPlain } from "./lib/utils";

const DashboardPage = lazy(async () => ({ default: (await import("./pages/DashboardPage")).DashboardPage }));
const ProfilesPage = lazy(async () => ({ default: (await import("./pages/ProfilesPage")).ProfilesPage }));
const TasksPage = lazy(async () => ({ default: (await import("./pages/TasksPage")).TasksPage }));
const RunsPage = lazy(async () => ({ default: (await import("./pages/RunsPage")).RunsPage }));
const StatsPage = lazy(async () => ({ default: (await import("./features/stats")).StatsPage }));
const StatusDialog = lazy(async () => ({ default: (await import("./components/dialogs/StatusDialog")).StatusDialog }));
const SettingsDialog = lazy(async () => ({ default: (await import("./components/dialogs/SettingsDialog")).SettingsDialog }));
const QuickRunDialog = lazy(async () => ({ default: (await import("./components/QuickRunDialog")).QuickRunDialog }));

export default function App() {
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
  const runners = useRunners({ enabled: needsRunnerData });
  const modelRegistry = useModelRegistry({ enabled: needsRunnerData });
  const isMobile = useIsMobile();
  const runEventStore = useRunEventStore();
  const reconciliationInFlightRef = useRef<Set<string>>(new Set());

  useEffect(() => {
    runEventStore.actions.runsSnapshotLoaded(runs.data || []);
  }, [runs.data, runEventStore.actions]);

  const mergedRuns = useMemo(() => {
    const snapshots = runEventStore.state.runsById;
    return (runs.data || []).map((run) => ({ ...run, ...(snapshots[run.id] ?? {}) } as Run));
  }, [runs.data, runEventStore.state.runsById]);

  // Derive active section from current path
  const getActiveSection = useCallback((): NavSection => {
    const path = location.pathname;
    if (path.startsWith("/profiles")) return "profiles";
    if (path.startsWith("/tasks")) return "tasks";
    if (path.startsWith("/runs")) return "runs";
    if (path.startsWith("/stats")) return "stats";
    return "dashboard";
  }, [location.pathname]);

  const activeSection = getActiveSection();

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
      navigate(`/${section === "dashboard" ? "" : section}`);
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
      <div className="h-screen bg-transparent text-foreground flex flex-col overflow-hidden">
        <AppHeader
          health={health.data}
          wsStatus={ws.status}
          activeSection={activeSection}
          isMobile={isMobile}
          onSectionChange={handleSectionChange}
          onStatusClick={() => setStatusOpen(true)}
          onSettingsClick={() => setSettingsOpen(true)}
          onQuickRunClick={() => setQuickRunOpen(true)}
        />

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
                runners={runners.data ?? undefined}
                modelRegistry={modelRegistry.data ?? undefined}
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
        <main
          className={`flex-1 min-h-0 overflow-hidden ${isMobile ? "pb-16" : ""}`}
        >
          <ErrorBoundary section="Application">
            <Routes>
            <Route
              path="/"
              element={
                <ErrorBoundary section="Dashboard">
                  <Suspense fallback={pageFallback}>
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
                  </Suspense>
                </ErrorBoundary>
              }
            />
            <Route
              path="/profiles"
              element={
                <Suspense fallback={pageFallback}>
                  <ErrorBoundary section="Profiles">
                    <ProfilesPage
                      profiles={profiles.data || []}
                      loading={profiles.loading}
                      error={profiles.error}
                      onCreateProfile={profiles.createProfile}
                      onUpdateProfile={profiles.updateProfile}
                      onDeleteProfile={profiles.deleteProfile}
                      onRefresh={profiles.refetch}
                      runners={runners.data ?? undefined}
                      modelRegistry={modelRegistry.data ?? undefined}
                    />
                  </ErrorBoundary>
                </Suspense>
              }
            />
            <Route
              path="/tasks"
              element={
                <Suspense fallback={pageFallback}>
                  <ErrorBoundary section="Tasks">
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
                      runners={runners.data ?? undefined}
                      modelRegistry={modelRegistry.data ?? undefined}
                    />
                  </ErrorBoundary>
                </Suspense>
              }
            />
            <Route
              path="/runs/:runId?"
              element={
                <Suspense fallback={pageFallback}>
                  <ErrorBoundary section="Runs">
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
                      onInvestigateRuns={runs.investigateRuns}
                      onApplyInvestigation={runs.applyInvestigation}
                      onResumeFromFailedRun={runs.resumeFromFailedRun}
                      onContinueRun={runs.continueRun}
                      onDeleteRunMessage={runs.deleteRunMessage}
                      onRefresh={runs.refetch}
                      runEventStore={runEventStore}
                      wsSubscribe={ws.subscribe}
                      wsUnsubscribe={ws.unsubscribe}
                    />
                  </ErrorBoundary>
                </Suspense>
              }
            />
            <Route
              path="/stats"
              element={
                <Suspense fallback={pageFallback}>
                  <ErrorBoundary section="Stats">
                    <StatsPage />
                  </ErrorBoundary>
                </Suspense>
              }
            />
            {/* Redirect unknown paths to dashboard */}
            <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </ErrorBoundary>
        </main>

        {/* Mobile bottom navigation */}
        {isMobile && (
          <MobileNav
            activeSection={activeSection}
            onSectionChange={handleSectionChange}
          />
        )}
      </div>
    </QueryProvider>
  );
}
