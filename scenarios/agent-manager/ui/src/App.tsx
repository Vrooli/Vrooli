import { useCallback, useState } from "react";
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
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "./components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "./components/ui/tabs";
import { useHealth, useProfiles, useRuns, useTasks } from "./hooks/useApi";
import { useWebSocket, WebSocketMessage } from "./hooks/useWebSocket";
import { DashboardPage } from "./pages/DashboardPage";
import { ProfilesPage } from "./pages/ProfilesPage";
import { TasksPage } from "./pages/TasksPage";
import { RunsPage } from "./pages/RunsPage";

export default function App() {
  const [activeTab, setActiveTab] = useState("dashboard");
  const health = useHealth();
  const profiles = useProfiles();
  const tasks = useTasks();
  const runs = useRuns();

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
    setActiveTab(value);
  }, []);

  const isHealthy = health.data?.status === "healthy";

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
                  variant={isHealthy ? "success" : "destructive"}
                  className="gap-1"
                >
                  {isHealthy ? (
                    <CheckCircle2 className="h-3 w-3" />
                  ) : (
                    <AlertCircle className="h-3 w-3" />
                  )}
                  {isHealthy ? "Healthy" : "Degraded"}
                </Badge>
                <Badge
                  variant={ws.status === "connected" ? "success" : ws.status === "connecting" ? "secondary" : "outline"}
                  className="gap-1 cursor-pointer"
                  onClick={() => ws.status !== "connected" && ws.reconnect()}
                  title={ws.status === "connected" ? "WebSocket connected" : "Click to reconnect"}
                >
                  {ws.status === "connected" ? (
                    <Wifi className="h-3 w-3" />
                  ) : (
                    <WifiOff className="h-3 w-3" />
                  )}
                  {ws.status === "connected" ? "Live" : ws.status === "connecting" ? "Connecting..." : "Offline"}
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
            <div className="flex gap-4">
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
                    (r) => r.status === "running" || r.status === "starting"
                  ).length ?? 0
                }
              />
            </div>
          </div>
        </header>
      </div>

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

          <TabsContent value="dashboard">
            <DashboardPage
              health={health.data}
              profiles={profiles.data || []}
              tasks={tasks.data || []}
              runs={runs.data || []}
              onRefresh={() => {
                health.refetch();
                profiles.refetch();
                tasks.refetch();
                runs.refetch();
              }}
            />
          </TabsContent>

          <TabsContent value="profiles">
            <ProfilesPage
              profiles={profiles.data || []}
              loading={profiles.loading}
              error={profiles.error}
              onCreateProfile={profiles.createProfile}
              onUpdateProfile={profiles.updateProfile}
              onDeleteProfile={profiles.deleteProfile}
              onRefresh={profiles.refetch}
            />
          </TabsContent>

          <TabsContent value="tasks">
            <TasksPage
              tasks={tasks.data || []}
              profiles={profiles.data || []}
              loading={tasks.loading}
              error={tasks.error}
              onCreateTask={tasks.createTask}
              onCancelTask={tasks.cancelTask}
              onCreateRun={runs.createRun}
              onCreateProfile={profiles.createProfile}
              onRefresh={tasks.refetch}
            />
          </TabsContent>

          <TabsContent value="runs">
            <RunsPage
              runs={runs.data || []}
              tasks={tasks.data || []}
              profiles={profiles.data || []}
              loading={runs.loading}
              error={runs.error}
              onStopRun={runs.stopRun}
              onGetEvents={runs.getRunEvents}
              onGetDiff={runs.getRunDiff}
              onApproveRun={runs.approveRun}
              onRejectRun={runs.rejectRun}
              onRefresh={runs.refetch}
            />
          </TabsContent>
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
