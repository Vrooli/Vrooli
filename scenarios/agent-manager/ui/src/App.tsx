import { useCallback, useEffect, useState } from "react";
import { Routes, Route, useNavigate, useLocation, Navigate } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  Activity,
  AlertCircle,
  BarChart3,
  Bot,
  CheckCircle2,
  ClipboardList,
  Play,
  Search,
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "./components/ui/tabs";
import { useHealth, useMaintenance, useModelRegistry, useProfiles, useRuns, useRunners, useTasks } from "./hooks/useApi";
import { useWebSocket, type WebSocketMessage } from "./hooks/useWebSocket";
import { DashboardPage } from "./pages/DashboardPage";
import { ProfilesPage } from "./pages/ProfilesPage";
import { TasksPage } from "./pages/TasksPage";
import { RunsPage } from "./pages/RunsPage";
import { InvestigationsPage } from "./pages/InvestigationsPage";
import { StatsPage } from "./features/stats";

// Create a QueryClient instance for React Query
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000, // 30 seconds
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});
import type { ModelOption, ModelRegistry } from "./types";
import { HealthStatus, RunStatus } from "./types";
import { PurgeTarget } from "@vrooli/proto-types/agent-manager/v1/api/service_pb";
import { formatDate, jsonValueToPlain, runnerTypeFromSlug, runnerTypeLabel } from "./lib/utils";

export default function App() {
  const navigate = useNavigate();
  const location = useLocation();
  const health = useHealth();
  const profiles = useProfiles();
  const tasks = useTasks();
  const runs = useRuns();
  const runners = useRunners();
  const modelRegistry = useModelRegistry();
  const maintenance = useMaintenance();
  const [settingsTab, setSettingsTab] = useState("models");
  const [modelRegistryDraft, setModelRegistryDraft] = useState<ModelRegistry | null>(null);
  const [modelRegistryError, setModelRegistryError] = useState<string | null>(null);
  const [modelRegistrySaving, setModelRegistrySaving] = useState(false);
  const [newRunnerKey, setNewRunnerKey] = useState("");

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

  const updateModelRegistryDraft = useCallback(
    (updater: (draft: ModelRegistry) => ModelRegistry) => {
      setModelRegistryDraft((prev) => (prev ? updater(prev) : prev));
    },
    []
  );

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

  useEffect(() => {
    if (!settingsOpen) return;
    if (modelRegistry.data) {
      setModelRegistryDraft(JSON.parse(JSON.stringify(modelRegistry.data)) as ModelRegistry);
      setModelRegistryError(null);
    }
  }, [settingsOpen, modelRegistry.data]);
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

  const handleModelRegistrySave = useCallback(async () => {
    if (!modelRegistryDraft) return;
    setModelRegistrySaving(true);
    setModelRegistryError(null);
    try {
      const sanitized = sanitizeModelRegistry(modelRegistryDraft);
      const updated = await modelRegistry.updateRegistry(sanitized);
      setModelRegistryDraft(JSON.parse(JSON.stringify(updated)) as ModelRegistry);
    } catch (err) {
      setModelRegistryError((err as Error).message);
    } finally {
      setModelRegistrySaving(false);
    }
  }, [modelRegistry.updateRegistry, modelRegistryDraft]);

  const handleModelRegistryReset = useCallback(() => {
    if (!modelRegistry.data) return;
    setModelRegistryDraft(JSON.parse(JSON.stringify(modelRegistry.data)) as ModelRegistry);
    setModelRegistryError(null);
  }, [modelRegistry.data]);

  const handleAddRunner = useCallback(() => {
    const trimmedKey = newRunnerKey.trim();
    if (!trimmedKey) {
      setModelRegistryError("Runner key is required.");
      return;
    }
    updateModelRegistryDraft((draft) => {
      if (draft.runners[trimmedKey]) {
        setModelRegistryError("Runner already exists.");
        return draft;
      }
      return {
        ...draft,
        runners: {
          ...draft.runners,
          [trimmedKey]: {
            models: [],
            presets: {},
          },
        },
      };
    });
    setModelRegistryError(null);
    setNewRunnerKey("");
  }, [newRunnerKey, updateModelRegistryDraft]);

  const handleAddFallbackRunner = useCallback(() => {
    updateModelRegistryDraft((draft) => {
      const fallback = draft.fallbackRunnerTypes ? [...draft.fallbackRunnerTypes] : [];
      fallback.push("");
      return { ...draft, fallbackRunnerTypes: fallback };
    });
  }, [updateModelRegistryDraft]);

  const handleFallbackRunnerChange = useCallback(
    (index: number, value: string) => {
      updateModelRegistryDraft((draft) => {
        const fallback = draft.fallbackRunnerTypes ? [...draft.fallbackRunnerTypes] : [];
        fallback[index] = value;
        return { ...draft, fallbackRunnerTypes: fallback };
      });
    },
    [updateModelRegistryDraft]
  );

  const handleRemoveFallbackRunner = useCallback(
    (index: number) => {
      updateModelRegistryDraft((draft) => {
        const fallback = draft.fallbackRunnerTypes ? [...draft.fallbackRunnerTypes] : [];
        fallback.splice(index, 1);
        return { ...draft, fallbackRunnerTypes: fallback };
      });
    },
    [updateModelRegistryDraft]
  );

  const handleRemoveRunner = useCallback(
    (runnerKey: string) => {
      updateModelRegistryDraft((draft) => {
        const { [runnerKey]: _, ...rest } = draft.runners;
        return { ...draft, runners: rest };
      });
    },
    [updateModelRegistryDraft]
  );

  const handleAddModel = useCallback(
    (runnerKey: string) => {
      updateModelRegistryDraft((draft) => {
        const runner = draft.runners[runnerKey];
        if (!runner) return draft;
        const models = normalizeModelOptions(runner.models);
        return {
          ...draft,
          runners: {
            ...draft.runners,
            [runnerKey]: {
              ...runner,
              models: [...models, { id: "", description: "" }],
            },
          },
        };
      });
    },
    [updateModelRegistryDraft]
  );

  const handleRemoveModel = useCallback(
    (runnerKey: string, index: number) => {
      updateModelRegistryDraft((draft) => {
        const runner = draft.runners[runnerKey];
        if (!runner) return draft;
        const models = normalizeModelOptions(runner.models).filter((_, idx) => idx !== index);
        return {
          ...draft,
          runners: {
            ...draft.runners,
            [runnerKey]: {
              ...runner,
              models,
            },
          },
        };
      });
    },
    [updateModelRegistryDraft]
  );

  const handleUpdateModel = useCallback(
    (runnerKey: string, index: number, field: "id" | "description", value: string) => {
      updateModelRegistryDraft((draft) => {
        const runner = draft.runners[runnerKey];
        if (!runner) return draft;
        const models = normalizeModelOptions(runner.models);
        const next = models.map((model, idx) =>
          idx === index ? { ...model, [field]: value } : model
        );
        return {
          ...draft,
          runners: {
            ...draft.runners,
            [runnerKey]: {
              ...runner,
              models: next,
            },
          },
        };
      });
    },
    [updateModelRegistryDraft]
  );

  const handlePresetChange = useCallback(
    (runnerKey: string, presetKey: string, value: string) => {
      updateModelRegistryDraft((draft) => {
        const runner = draft.runners[runnerKey];
        if (!runner) return draft;
        const nextPresets = { ...(runner.presets ?? {}) };
        if (value) {
          nextPresets[presetKey] = value;
        } else {
          delete nextPresets[presetKey];
        }
        return {
          ...draft,
          runners: {
            ...draft.runners,
            [runnerKey]: {
              ...runner,
              presets: nextPresets,
            },
          },
        };
      });
    },
    [updateModelRegistryDraft]
  );

  return (
    <QueryClientProvider client={queryClient}>
    <div className="min-h-screen bg-transparent text-foreground">
      {/* Header */}
      <header className="flex items-center justify-between gap-4 border-b border-border/50 px-6 py-3 sm:px-10">
        <div className="flex items-center gap-3">
          <Bot className="h-6 w-6 text-primary" />
          <span className="text-lg font-semibold">Agent Manager</span>
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
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setSettingsOpen(true)}
          className="gap-2"
        >
          <Settings2 className="h-4 w-4" />
          Settings
        </Button>
      </header>

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
                    const storage = plain?.storage as string | undefined;
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
                        {storage && (
                          <div className="flex items-center justify-between">
                            <span>Storage</span>
                            <span>{storage}</span>
                          </div>
                        )}
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

      <Dialog open={settingsOpen} onOpenChange={setSettingsOpen} contentClassName="max-w-5xl">
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Settings</DialogTitle>
            <DialogDescription>Manage model registry configuration and maintenance actions.</DialogDescription>
          </DialogHeader>
          <DialogBody className="space-y-6">
            <Tabs value={settingsTab} onValueChange={setSettingsTab}>
              <TabsList className="grid w-full grid-cols-2">
                <TabsTrigger value="models">Model Registry</TabsTrigger>
                <TabsTrigger value="maintenance">Maintenance</TabsTrigger>
              </TabsList>
              <TabsContent value="models" className="mt-4 space-y-6">
                <div className="flex flex-wrap items-start justify-between gap-4">
                  <div className="space-y-1">
                    <h3 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">Model Registry</h3>
                    <p className="text-sm text-muted-foreground">
                      Configure per-runner model lists and map Fast/Cheap/Smart presets to model IDs. Updates apply across all scenarios that use agent-manager.
                    </p>
                  </div>
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <Badge variant="secondary">FAST</Badge>
                    <Badge variant="secondary">CHEAP</Badge>
                    <Badge variant="secondary">SMART</Badge>
                  </div>
                </div>

                {!modelRegistryDraft && modelRegistry.loading && (
                  <p className="text-sm text-muted-foreground">Loading model registry...</p>
                )}
                {!modelRegistryDraft && modelRegistry.error && (
                  <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                    {modelRegistry.error}
                  </div>
                )}

                {modelRegistryDraft && (
                  <div className="space-y-5">
                    <Card className="border-border/60 bg-card/40">
                      <CardContent className="space-y-4 py-5">
                        <div className="space-y-1">
                          <h3 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">Global Fallbacks</h3>
                          <p className="text-xs text-muted-foreground">
                            Ordered runner list to try when the primary runner is unavailable. Empty disables fallback.
                          </p>
                        </div>
                        <div className="space-y-2">
                          {(modelRegistryDraft.fallbackRunnerTypes ?? []).length === 0 ? (
                            <p className="text-xs text-muted-foreground">No global fallbacks configured.</p>
                          ) : (
                            (modelRegistryDraft.fallbackRunnerTypes ?? []).map((runnerKey, index) => {
                              const resolvedRunner = runnerTypeFromSlug(runnerKey);
                              const label = resolvedRunner ? runnerTypeLabel(resolvedRunner) : runnerKey;
                              return (
                                <div key={`fallback-${index}`} className="flex flex-wrap items-center gap-2">
                                  <select
                                    value={runnerKey}
                                    onChange={(event) => handleFallbackRunnerChange(index, event.target.value)}
                                    className="flex h-10 min-w-[200px] flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                                  >
                                    <option value="">Select runner...</option>
                                    {Array.from(
                                      new Set([
                                        ...Object.keys(modelRegistryDraft.runners),
                                        "claude-code",
                                        "codex",
                                        "opencode",
                                      ])
                                    )
                                      .sort((a, b) => a.localeCompare(b))
                                      .map((option) => {
                                        const resolvedOption = runnerTypeFromSlug(option);
                                        const optionLabel = resolvedOption ? runnerTypeLabel(resolvedOption) : option;
                                        return (
                                          <option key={option} value={option}>
                                            {optionLabel}
                                          </option>
                                        );
                                      })}
                                  </select>
                                  <Badge variant="secondary">{label}</Badge>
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => handleRemoveFallbackRunner(index)}
                                  >
                                    Remove
                                  </Button>
                                </div>
                              );
                            })
                          )}
                          <div>
                            <Button variant="outline" size="sm" onClick={handleAddFallbackRunner}>
                              Add Fallback
                            </Button>
                          </div>
                        </div>
                      </CardContent>
                    </Card>

                    {Object.entries(modelRegistryDraft.runners).length === 0 && (
                      <Card className="border-border/60 bg-card/40">
                        <CardContent className="py-6 text-sm text-muted-foreground">
                          No runners are configured yet. Add your first runner to build a model list.
                        </CardContent>
                      </Card>
                    )}
                    {Object.entries(modelRegistryDraft.runners)
                      .sort(([a], [b]) => a.localeCompare(b))
                      .map(([runnerKey, runnerConfig]) => {
                        const models = normalizeModelOptions(runnerConfig.models);
                        const modelIds = models.map((model) => model.id).filter((id) => id);
                        const presets = runnerConfig.presets ?? {};
                        return (
                          <Card key={runnerKey} className="border-border/60 bg-card/40">
                            <CardContent className="space-y-4 py-5">
                              <div className="flex flex-wrap items-start justify-between gap-3">
                                <div className="space-y-1">
                                  <div className="flex items-center gap-2">
                                    <Badge variant="secondary">{runnerKey}</Badge>
                                    <span className="text-xs text-muted-foreground">
                                      {models.length} model{models.length === 1 ? "" : "s"}
                                    </span>
                                  </div>
                                  <p className="text-xs text-muted-foreground">Configure models and presets for this runner.</p>
                                </div>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => handleRemoveRunner(runnerKey)}
                                >
                                  Remove Runner
                                </Button>
                              </div>

                              <div className="space-y-3 rounded-md border border-border/70 bg-background/60 p-3">
                                <div className="grid gap-2 text-[11px] font-semibold uppercase text-muted-foreground sm:grid-cols-[1fr_1.5fr_auto]">
                                  <span>Model ID</span>
                                  <span>Description</span>
                                  <span className="sr-only">Actions</span>
                                </div>
                                {models.length === 0 ? (
                                  <p className="text-xs text-muted-foreground">No models yet. Add one to populate the list.</p>
                                ) : (
                                  models.map((model, index) => (
                                    <div
                                      key={`${runnerKey}-${index}`}
                                      className="grid gap-2 sm:grid-cols-[1fr_1.5fr_auto] sm:items-end"
                                    >
                                      <Input
                                        value={model.id}
                                        onChange={(event) =>
                                          handleUpdateModel(runnerKey, index, "id", event.target.value)
                                        }
                                        placeholder="gpt-5.1-codex"
                                      />
                                      <Input
                                        value={model.description}
                                        onChange={(event) =>
                                          handleUpdateModel(
                                            runnerKey,
                                            index,
                                            "description",
                                            event.target.value
                                          )
                                        }
                                        placeholder="Short description"
                                      />
                                      <Button
                                        variant="ghost"
                                        size="sm"
                                        onClick={() => handleRemoveModel(runnerKey, index)}
                                      >
                                        Remove
                                      </Button>
                                    </div>
                                  ))
                                )}
                                <div>
                                  <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={() => handleAddModel(runnerKey)}
                                  >
                                    Add Model
                                  </Button>
                                </div>
                              </div>

                              <div className="space-y-3 rounded-md border border-border/70 bg-background/60 p-3">
                                <div className="flex flex-wrap items-center justify-between gap-2">
                                  <Label>Preset Mappings</Label>
                                  <p className="text-xs text-muted-foreground">Map presets to models in this list.</p>
                                </div>
                                <div className="grid gap-2 sm:grid-cols-3">
                                  {[
                                    { key: "FAST", label: "Fast" },
                                    { key: "CHEAP", label: "Cheap" },
                                    { key: "SMART", label: "Smart" },
                                  ].map((preset) => (
                                    <div key={`${runnerKey}-${preset.key}`} className="space-y-1">
                                      <Label>{preset.label}</Label>
                                      <select
                                        value={presets[preset.key] ?? ""}
                                        onChange={(event) =>
                                          handlePresetChange(runnerKey, preset.key, event.target.value)
                                        }
                                        className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                                      >
                                        <option value="">Unassigned</option>
                                        {modelIds.map((modelId) => (
                                          <option key={modelId} value={modelId}>
                                            {modelId}
                                          </option>
                                        ))}
                                      </select>
                                    </div>
                                  ))}
                                </div>
                              </div>
                            </CardContent>
                          </Card>
                        );
                      })}

                    <Card className="border-dashed border-border/70 bg-card/30">
                      <CardContent className="space-y-3 py-5">
                        <Label>Add Runner</Label>
                        <div className="flex flex-wrap gap-2">
                          <Input
                            value={newRunnerKey}
                            onChange={(event) => setNewRunnerKey(event.target.value)}
                            placeholder="runner-key"
                            className="flex-1 min-w-[200px]"
                          />
                          <Button variant="outline" onClick={handleAddRunner}>
                            Add Runner
                          </Button>
                        </div>
                        {runners.data && Object.keys(runners.data).length > 0 && (
                          <p className="text-xs text-muted-foreground">
                            Known runners: {Object.keys(runners.data).join(", ")}
                          </p>
                        )}
                      </CardContent>
                    </Card>

                    {modelRegistryError && (
                      <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                        {modelRegistryError}
                      </div>
                    )}
                  </div>
                )}
              </TabsContent>
              <TabsContent value="maintenance" className="mt-4 space-y-6">
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
              </TabsContent>
            </Tabs>
          </DialogBody>
          <DialogFooter>
            {settingsTab === "models" ? (
              <>
                <Button variant="outline" onClick={handleModelRegistryReset} disabled={!modelRegistryDraft}>
                  Reset
                </Button>
                <Button
                  onClick={handleModelRegistrySave}
                  disabled={!modelRegistryDraft || modelRegistrySaving}
                >
                  {modelRegistrySaving ? "Saving..." : "Save"}
                </Button>
                <Button variant="outline" onClick={() => setSettingsOpen(false)}>
                  Close
                </Button>
              </>
            ) : (
              <Button variant="outline" onClick={() => setSettingsOpen(false)}>
                Close
              </Button>
            )}
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
    </QueryClientProvider>
  );
}


type EditableModelOption = { id: string; description: string };

function normalizeModelOption(option: ModelOption): EditableModelOption {
  if (typeof option === "string") {
    return { id: option, description: "" };
  }
  return {
    id: option.id,
    description: option.description ?? "",
  };
}

function normalizeModelOptions(options: ModelOption[]): EditableModelOption[] {
  return options.map(normalizeModelOption);
}

function sanitizeModelRegistry(registry: ModelRegistry): ModelRegistry {
  const runners: ModelRegistry["runners"] = {};
  const fallbackRunnerTypes = Array.from(
    new Set(
      (registry.fallbackRunnerTypes ?? [])
        .map((runner) => runner.trim())
        .filter((runner) => runner.length > 0)
    )
  );
  for (const [runnerKey, runner] of Object.entries(registry.runners)) {
    const normalizedModels = normalizeModelOptions(runner.models)
      .map((model) => ({
        id: model.id.trim(),
        description: model.description.trim(),
      }))
      .filter((model) => model.id.length > 0);
    const models: ModelOption[] = normalizedModels.map((model) =>
      model.description ? { id: model.id, description: model.description } : model.id
    );
    const presets: Record<string, string> = {};
    for (const [presetKey, modelId] of Object.entries(runner.presets ?? {})) {
      const trimmedModelId = modelId.trim();
      if (trimmedModelId) {
        presets[presetKey] = trimmedModelId;
      }
    }
    runners[runnerKey] = {
      models,
      presets,
    };
  }
  return {
    ...registry,
    fallbackRunnerTypes,
    runners,
  };
}
