import { useCallback, useState } from "react";
import { Routes, Route, useNavigate, useLocation, Navigate } from "react-router-dom";
import { useHealth, useProfiles, useRuns, useRunners, useModelRegistry, useTasks } from "./hooks/useApi";
import { useWebSocket, type WebSocketMessage } from "./hooks/useWebSocket";
import { useIsMobile } from "./hooks/useViewportSize";
import { QueryProvider } from "./providers/QueryProvider";
import { AppHeader } from "./components/layout/AppHeader";
import { MobileNav, type NavSection } from "./components/layout/MobileNav";
import { StatusDialog } from "./components/dialogs/StatusDialog";
import { SettingsDialog } from "./components/dialogs/SettingsDialog";
import { QuickRunDialog } from "./components/QuickRunDialog";
import { DashboardPage } from "./pages/DashboardPage";
import { ProfilesPage } from "./pages/ProfilesPage";
import { TasksPage } from "./pages/TasksPage";
import { RunsPage } from "./pages/RunsPage";
import { StatsPage } from "./features/stats";

export default function App() {
  const navigate = useNavigate();
  const location = useLocation();
  const health = useHealth();
  const profiles = useProfiles();
  const tasks = useTasks();
  const runs = useRuns();
  const runners = useRunners();
  const modelRegistry = useModelRegistry();
  const isMobile = useIsMobile();

  const [statusOpen, setStatusOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [quickRunOpen, setQuickRunOpen] = useState(false);

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
  const handleWebSocketMessage = useCallback(
    (message: WebSocketMessage) => {
      console.log("[WS] Received:", message.type);

      switch (message.type) {
        case "run_event":
        case "run_status":
          runs.refetch();
          break;
        case "task_status":
          tasks.refetch();
          break;
        case "connected":
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

  const handleSectionChange = useCallback(
    (section: NavSection) => {
      navigate(`/${section === "dashboard" ? "" : section}`);
    },
    [navigate]
  );

  const handlePurgeComplete = useCallback(() => {
    profiles.refetch();
    tasks.refetch();
    runs.refetch();
  }, [profiles, tasks, runs]);

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

        <StatusDialog
          open={statusOpen}
          onOpenChange={setStatusOpen}
          health={health.data}
          healthError={health.error}
          wsStatus={ws.status}
        />

        <SettingsDialog
          open={settingsOpen}
          onOpenChange={setSettingsOpen}
          onPurgeComplete={handlePurgeComplete}
        />

        <QuickRunDialog
          open={quickRunOpen}
          onOpenChange={setQuickRunOpen}
          profiles={profiles.data || []}
          runners={runners.data ?? undefined}
          modelRegistry={modelRegistry.data ?? undefined}
          onCreateTask={tasks.createTask}
          onCreateRun={runs.createRun}
          onRunCreated={(run) => {
            runs.refetch();
            tasks.refetch();
            navigate(`/runs/${run.id}`);
          }}
        />

        {/* Main Content */}
        <main
          className={`flex-1 min-h-0 overflow-hidden ${isMobile ? "pb-16" : ""}`}
        >
          <Routes>
            <Route
              path="/"
              element={
                <DashboardPage
                  health={health.data}
                  tasks={tasks.data || []}
                  runs={runs.data || []}
                  onRefresh={() => {
                    health.refetch();
                    profiles.refetch();
                    tasks.refetch();
                    runs.refetch();
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
                  modelRegistry={modelRegistry.data ?? undefined}
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
                  onUpdateTask={tasks.updateTask}
                  onCancelTask={tasks.cancelTask}
                  onDeleteTask={tasks.deleteTask}
                  onCreateRun={runs.createRun}
                  onCreateProfile={profiles.createProfile}
                  onRefresh={tasks.refetch}
                  runners={runners.data ?? undefined}
                  modelRegistry={modelRegistry.data ?? undefined}
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
                  onDeleteRun={runs.deleteRun}
                  onRetryRun={runs.retryRun}
                  onGetRun={runs.getRun}
                  onGetEvents={runs.getRunEvents}
                  onGetDiff={runs.getRunDiff}
                  onGetTask={tasks.getTask}
                  onApproveRun={runs.approveRun}
                  onRejectRun={runs.rejectRun}
                  onInvestigateRuns={runs.investigateRuns}
                  onApplyInvestigation={runs.applyInvestigation}
                  onContinueRun={runs.continueRun}
                  onRefresh={runs.refetch}
                  wsSubscribe={ws.subscribe}
                  wsUnsubscribe={ws.unsubscribe}
                  wsAddMessageHandler={ws.addMessageHandler}
                  wsRemoveMessageHandler={ws.removeMessageHandler}
                />
              }
            />
            <Route path="/stats" element={<StatsPage />} />
            {/* Redirect unknown paths to dashboard */}
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
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
