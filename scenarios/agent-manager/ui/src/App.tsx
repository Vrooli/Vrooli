import { useCallback, useState } from "react";
import { Routes, Route, useNavigate, useLocation, Navigate } from "react-router-dom";
import {
  Activity,
  AlertCircle,
  BarChart3,
  ClipboardList,
  Play,
  Search,
  Settings2,
} from "lucide-react";
import { Card, CardContent } from "./components/ui/card";
import { Tabs, TabsList, TabsTrigger } from "./components/ui/tabs";
import { useHealth, useProfiles, useRuns, useRunners, useModelRegistry, useTasks } from "./hooks/useApi";
import { useWebSocket, type WebSocketMessage } from "./hooks/useWebSocket";
import { QueryProvider } from "./providers/QueryProvider";
import { AppHeader } from "./components/layout/AppHeader";
import { StatusDialog } from "./components/dialogs/StatusDialog";
import { SettingsDialog } from "./components/dialogs/SettingsDialog";
import { DashboardPage } from "./pages/DashboardPage";
import { ProfilesPage } from "./pages/ProfilesPage";
import { TasksPage } from "./pages/TasksPage";
import { RunsPage } from "./pages/RunsPage";
import { InvestigationsPage } from "./pages/InvestigationsPage";
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

  const [statusOpen, setStatusOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);

  // Derive active tab from current path
  const getActiveTab = useCallback(() => {
    const path = location.pathname;
    if (path.startsWith("/profiles")) return "profiles";
    if (path.startsWith("/tasks")) return "tasks";
    if (path.startsWith("/runs")) return "runs";
    if (path.startsWith("/investigations")) return "investigations";
    if (path.startsWith("/stats")) return "stats";
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

  const handleTabChange = useCallback((value: string) => {
    navigate(`/${value === "dashboard" ? "" : value}`);
  }, [navigate]);

  const handlePurgeComplete = useCallback(() => {
    profiles.refetch();
    tasks.refetch();
    runs.refetch();
  }, [profiles, tasks, runs]);

  return (
    <QueryProvider>
      <div className="min-h-screen bg-transparent text-foreground">
        <AppHeader
          health={health.data}
          wsStatus={ws.status}
          onStatusClick={() => setStatusOpen(true)}
          onSettingsClick={() => setSettingsOpen(true)}
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
            <TabsList className="mb-6 grid w-full max-w-[900px] grid-cols-6">
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
              <TabsTrigger value="investigations" className="gap-2">
                <Search className="h-4 w-4" />
                Investigations
              </TabsTrigger>
              <TabsTrigger value="stats" className="gap-2">
                <BarChart3 className="h-4 w-4" />
                Stats
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
                    modelRegistry={modelRegistry.data ?? undefined}
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
              <Route
                path="/investigations/:investigationId?"
                element={
                  <InvestigationsPage
                    onViewRun={(runId) => navigate(`/runs/${runId}`)}
                  />
                }
              />
              <Route path="/stats" element={<StatsPage />} />
              {/* Redirect unknown paths to dashboard */}
              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </Tabs>
        </main>
      </div>
    </QueryProvider>
  );
}
