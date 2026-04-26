import { lazy, Suspense, useCallback, useEffect, useRef, useState } from "react";
import { Routes, Route, useNavigate, useLocation, Navigate } from "react-router-dom";
import { useHealth, useProfiles, useRuns, useRunners, useModelRegistry, useTasks, useRunStatusCounts } from "./hooks/useApi";
import { useWebSocket, type WebSocketMessage } from "./hooks/useWebSocket";
import { RunStatus } from "./types";
import type { Run } from "./types";
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

  // WebSocket connection for real-time updates
  // Use a ref to break the circular dependency: handleWebSocketMessage → ws → handleWebSocketMessage
  const subscribeAllRef = useRef<() => void>(() => {});

  // Track runs that reached terminal state to skip redundant refetches
  const terminalRunIdsRef = useRef<Set<string>>(new Set());

  // Debounce run refetches to coalesce rapid-fire WS updates (300ms)
  const refetchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const debouncedRunRefetch = useCallback(() => {
    if (refetchTimerRef.current !== null) {
      clearTimeout(refetchTimerRef.current);
    }
    refetchTimerRef.current = setTimeout(() => {
      refetchTimerRef.current = null;
      runs.refetch();
    }, 300);
  }, [runs]);

  // Clean up debounce timer on unmount
  useEffect(() => {
    return () => {
      if (refetchTimerRef.current !== null) {
        clearTimeout(refetchTimerRef.current);
      }
    };
  }, []);

  const TERMINAL_STATUSES = useRef([RunStatus.COMPLETE, RunStatus.FAILED, RunStatus.CANCELLED]);

  const handleWebSocketMessage = useCallback(
    (message: WebSocketMessage) => {
      if (import.meta.env.DEV) {
        console.log("[WS] Received:", message.type);
      }

      switch (message.type) {
        case "run_status": {
          const statusUpdate = message.payload as Partial<Run>;
          const isTerminal = statusUpdate.status !== undefined &&
            TERMINAL_STATUSES.current.includes(statusUpdate.status);
          if (isTerminal && message.runId) {
            terminalRunIdsRef.current.add(message.runId);
          }
          debouncedRunRefetch();
          // Refetch tasks if this run references a task we don't have yet
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
          // Skip refetch if run already reached terminal state
          if (message.runId && terminalRunIdsRef.current.has(message.runId)) {
            break;
          }
          debouncedRunRefetch();
          break;
        case "task_status":
          if (needsTaskData) {
            tasks.refetch();
          }
          break;
        case "connected":
          // Clear terminal cache on reconnect (state may have changed)
          terminalRunIdsRef.current.clear();
          subscribeAllRef.current();
          break;
      }
    },
    [debouncedRunRefetch, needsTaskData, tasks]
  );

  const ws = useWebSocket({
    enabled: true,
    onMessage: handleWebSocketMessage,
  });
  subscribeAllRef.current = ws.subscribeAll;

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
                      runs={runs.data || []}
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
                      runs={runs.data || []}
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
                      wsSubscribe={ws.subscribe}
                      wsUnsubscribe={ws.unsubscribe}
                      wsAddMessageHandler={ws.addMessageHandler}
                      wsRemoveMessageHandler={ws.removeMessageHandler}
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
