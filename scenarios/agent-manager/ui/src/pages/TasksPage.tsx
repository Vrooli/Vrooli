import { useMemo, useState } from "react";
import {
  AlertCircle,
  ClipboardList,
  FolderOpen,
  Play,
  Plus,
  RefreshCw,
  Search,
  Settings2,
  Trash2,
  XCircle,
} from "lucide-react";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { ModelSelector } from "../components/ModelSelector";
import { ScrollArea } from "../components/ui/scroll-area";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import { Textarea } from "../components/ui/textarea";
import { formatRelativeTime } from "../lib/utils";
import type { AgentProfile, CreateProfileRequest, CreateRunRequest, CreateTaskRequest, Run, RunnerStatus, RunnerType, Task } from "../types";

const RUNNER_TYPES: RunnerType[] = ["claude-code", "codex", "opencode"];

interface TasksPageProps {
  tasks: Task[];
  profiles: AgentProfile[];
  loading: boolean;
  error: string | null;
  onCreateTask: (task: CreateTaskRequest) => Promise<Task>;
  onCancelTask: (id: string) => Promise<void>;
  onDeleteTask: (id: string) => Promise<void>;
  onCreateRun: (run: CreateRunRequest) => Promise<Run>;
  onCreateProfile: (profile: CreateProfileRequest) => Promise<AgentProfile>;
  onRefresh: () => void;
  runners?: Record<string, RunnerStatus>;
}

// Default models for each runner type
const DEFAULT_MODELS: Record<RunnerType, string> = {
  "claude-code": "sonnet",
  "codex": "o4-mini",
  "opencode": "anthropic/claude-sonnet-4-5",
};

interface InlineRunConfig {
  runnerType: RunnerType;
  model: string;
  maxTurns: number;
  timeoutMinutes: number;
  runMode: "sandboxed" | "in_place";
  skipPermissionPrompt: boolean;
}

// Convert minutes to nanoseconds for Go's time.Duration
const minutesToNanoseconds = (minutes: number): number => minutes * 60 * 1_000_000_000;

export function TasksPage({
  tasks,
  profiles,
  loading,
  error,
  onCreateTask,
  onCancelTask,
  onDeleteTask,
  onCreateRun,
  onCreateProfile,
  onRefresh,
  runners,
}: TasksPageProps) {
  // Helper to get models for a runner type from capabilities
  const getModelsForRunner = (runnerType: RunnerType): string[] => {
    const runner = runners?.[runnerType];
    return runner?.capabilities?.SupportedModels ?? [];
  };

  // Helper to get default model for a runner type
  const getDefaultModelForRunner = (runnerType: RunnerType): string => {
    return DEFAULT_MODELS[runnerType] ?? "";
  };
  const [showForm, setShowForm] = useState(false);
  const [showRunDialog, setShowRunDialog] = useState<Task | null>(null);
  const [showProfileDialog, setShowProfileDialog] = useState(false);
  const [formData, setFormData] = useState<CreateTaskRequest>({
    title: "",
    description: "",
    scopePath: ".",
    projectRoot: "",
  });
  const [selectedProfileId, setSelectedProfileId] = useState("");
  const [runConfigMode, setRunConfigMode] = useState<"profile" | "custom">("profile");
  const [inlineConfig, setInlineConfig] = useState<InlineRunConfig>({
    runnerType: "claude-code",
    model: "sonnet",
    maxTurns: 100,
    timeoutMinutes: 30,
    runMode: "sandboxed",
    skipPermissionPrompt: true,
  });
  const [profileFormData, setProfileFormData] = useState<CreateProfileRequest>({
    name: "",
    description: "",
    runnerType: "claude-code",
    model: "sonnet",
    maxTurns: 100,
    requiresSandbox: true,
    requiresApproval: true,
  });
  const [profileTimeoutMinutes, setProfileTimeoutMinutes] = useState(30);
  const [submitting, setSubmitting] = useState(false);

  // Filter/sort/search state
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [sortBy, setSortBy] = useState<"newest" | "oldest" | "title">("newest");

  const resetForm = () => {
    setFormData({
      title: "",
      description: "",
      scopePath: ".",
      projectRoot: "",
    });
    setShowForm(false);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      await onCreateTask(formData);
      resetForm();
    } catch (err) {
      console.error("Failed to create task:", err);
    } finally {
      setSubmitting(false);
    }
  };

  const handleStartRun = async () => {
    if (!showRunDialog) return;

    // Validate based on mode
    if (runConfigMode === "profile" && !selectedProfileId) return;

    setSubmitting(true);
    try {
      const request: CreateRunRequest = {
        taskId: showRunDialog.id,
      };

      if (runConfigMode === "profile") {
        request.agentProfileId = selectedProfileId;
      } else {
        // Use inline config
        request.runnerType = inlineConfig.runnerType;
        request.model = inlineConfig.model;
        request.maxTurns = inlineConfig.maxTurns;
        request.timeout = minutesToNanoseconds(inlineConfig.timeoutMinutes);
        request.runMode = inlineConfig.runMode;
        request.skipPermissionPrompt = inlineConfig.skipPermissionPrompt;
      }

      await onCreateRun(request);
      setShowRunDialog(null);
      setSelectedProfileId("");
      setRunConfigMode("profile");
    } catch (err) {
      console.error("Failed to start run:", err);
    } finally {
      setSubmitting(false);
    }
  };

  const resetProfileForm = () => {
    setProfileFormData({
      name: "",
      description: "",
      runnerType: "claude-code",
      model: "claude-sonnet-4-20250514",
      maxTurns: 100,
      requiresSandbox: true,
      requiresApproval: true,
    });
    setProfileTimeoutMinutes(30);
    setShowProfileDialog(false);
  };

  const handleCreateProfile = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      const profileWithTimeout: CreateProfileRequest = {
        ...profileFormData,
        timeout: minutesToNanoseconds(profileTimeoutMinutes),
      };
      const newProfile = await onCreateProfile(profileWithTimeout);
      // Auto-select the new profile and switch to profile mode
      setSelectedProfileId(newProfile.id);
      setRunConfigMode("profile");
      resetProfileForm();
    } catch (err) {
      console.error("Failed to create profile:", err);
    } finally {
      setSubmitting(false);
    }
  };

  const handleCancel = async (taskId: string) => {
    if (!confirm("Are you sure you want to cancel this task?")) return;
    try {
      await onCancelTask(taskId);
    } catch (err) {
      console.error("Failed to cancel task:", err);
    }
  };

  const handleDelete = async (taskId: string) => {
    if (!confirm("Permanently delete this task? This cannot be undone.")) return;
    try {
      await onDeleteTask(taskId);
    } catch (err) {
      console.error("Failed to delete task:", err);
    }
  };

  const filteredAndSortedTasks = useMemo(() => {
    let result = [...tasks];

    // Filter by status
    if (statusFilter !== "all") {
      result = result.filter((t) => t.status === statusFilter);
    }

    // Filter by search query
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      result = result.filter(
        (t) =>
          t.title.toLowerCase().includes(query) ||
          t.description?.toLowerCase().includes(query)
      );
    }

    // Sort
    result.sort((a, b) => {
      if (sortBy === "newest") {
        return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime();
      } else if (sortBy === "oldest") {
        return new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime();
      } else {
        return a.title.localeCompare(b.title);
      }
    });

    return result;
  }, [tasks, statusFilter, searchQuery, sortBy]);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold">Tasks</h2>
          <p className="text-sm text-muted-foreground">
            Define what needs to be done
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={onRefresh} className="gap-2">
            <RefreshCw className="h-4 w-4" />
            Refresh
          </Button>
          <Button size="sm" onClick={() => setShowForm(true)} className="gap-2">
            <Plus className="h-4 w-4" />
            New Task
          </Button>
        </div>
      </div>

      {error && (
        <Card className="border-destructive/50 bg-destructive/10">
          <CardContent className="flex items-center gap-3 py-4">
            <AlertCircle className="h-4 w-4 text-destructive" />
            <p className="text-sm text-destructive">{error}</p>
          </CardContent>
        </Card>
      )}

      {/* Filter/Sort/Search */}
      <div className="flex flex-wrap gap-3 items-center">
        <div className="relative flex-1 min-w-[200px]">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search tasks..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="h-9 pl-9"
          />
        </div>
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="h-9 rounded-md border border-input bg-background px-3 text-sm"
        >
          <option value="all">All Status</option>
          <option value="queued">Queued</option>
          <option value="running">Running</option>
          <option value="needs_review">Needs Review</option>
          <option value="approved">Approved</option>
          <option value="rejected">Rejected</option>
          <option value="failed">Failed</option>
          <option value="cancelled">Cancelled</option>
        </select>
        <select
          value={sortBy}
          onChange={(e) => setSortBy(e.target.value as "newest" | "oldest" | "title")}
          className="h-9 rounded-md border border-input bg-background px-3 text-sm"
        >
          <option value="newest">Newest First</option>
          <option value="oldest">Oldest First</option>
          <option value="title">Title A-Z</option>
        </select>
      </div>

      {/* Create Task Modal */}
      <Dialog open={showForm} onOpenChange={(open) => !open && resetForm()}>
        <DialogContent>
          <DialogHeader onClose={resetForm}>
            <DialogTitle>Create New Task</DialogTitle>
            <DialogDescription>
              Define the work that an agent should perform
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleSubmit}>
            <DialogBody className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="title">Title *</Label>
                <Input
                  id="title"
                  value={formData.title}
                  onChange={(e) =>
                    setFormData({ ...formData, title: e.target.value })
                  }
                  placeholder="e.g., Fix authentication bug"
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  value={formData.description}
                  onChange={(e) =>
                    setFormData({ ...formData, description: e.target.value })
                  }
                  placeholder="Detailed instructions for the agent..."
                  rows={4}
                />
              </div>

              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="scopePath">Scope Path *</Label>
                  <Input
                    id="scopePath"
                    value={formData.scopePath}
                    onChange={(e) =>
                      setFormData({ ...formData, scopePath: e.target.value })
                    }
                    placeholder="e.g., src/auth"
                    required
                  />
                  <p className="text-xs text-muted-foreground">
                    The directory scope where the agent can operate
                  </p>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="projectRoot">Project Root</Label>
                  <Input
                    id="projectRoot"
                    value={formData.projectRoot}
                    onChange={(e) =>
                      setFormData({ ...formData, projectRoot: e.target.value })
                    }
                    placeholder="Optional: /path/to/project"
                  />
                </div>
              </div>
            </DialogBody>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={resetForm}>
                Cancel
              </Button>
              <Button type="submit" disabled={submitting}>
                {submitting ? "Creating..." : "Create Task"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Start Run Modal */}
      <Dialog
        open={showRunDialog !== null}
        onOpenChange={(open) => {
          if (!open) {
            setShowRunDialog(null);
            setSelectedProfileId("");
            setRunConfigMode("profile");
          }
        }}
      >
        <DialogContent className="max-w-lg">
          <DialogHeader onClose={() => setShowRunDialog(null)}>
            <DialogTitle>Start Run</DialogTitle>
            <DialogDescription>
              Configure how to execute: {showRunDialog?.title}
            </DialogDescription>
          </DialogHeader>
          <DialogBody className="space-y-4">
            <Tabs value={runConfigMode} onValueChange={(v) => setRunConfigMode(v as "profile" | "custom")}>
              <TabsList className="grid w-full grid-cols-2">
                <TabsTrigger value="profile" className="gap-2">
                  <Settings2 className="h-4 w-4" />
                  Use Profile
                </TabsTrigger>
                <TabsTrigger value="custom" className="gap-2">
                  <Play className="h-4 w-4" />
                  Quick Run
                </TabsTrigger>
              </TabsList>

              <TabsContent value="profile" className="space-y-4 mt-4">
                {profiles.length === 0 ? (
                  <div className="text-center py-4 space-y-3">
                    <p className="text-sm text-muted-foreground">
                      No agent profiles available.
                    </p>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setShowProfileDialog(true)}
                      className="gap-2"
                    >
                      <Plus className="h-4 w-4" />
                      Create Profile
                    </Button>
                  </div>
                ) : (
                  <>
                    <div className="space-y-2">
                      <Label htmlFor="profile">Agent Profile *</Label>
                      <select
                        id="profile"
                        value={selectedProfileId}
                        onChange={(e) => setSelectedProfileId(e.target.value)}
                        className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                      >
                        <option value="">Select a profile...</option>
                        {profiles.map((profile) => (
                          <option key={profile.id} value={profile.id}>
                            {profile.name} ({profile.runnerType})
                          </option>
                        ))}
                      </select>
                    </div>
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => setShowProfileDialog(true)}
                      className="gap-2"
                    >
                      <Plus className="h-4 w-4" />
                      Create New Profile
                    </Button>
                  </>
                )}
              </TabsContent>

              <TabsContent value="custom" className="space-y-4 mt-4">
                <p className="text-xs text-muted-foreground">
                  Run with custom settings without saving a profile.
                </p>
                <div className="space-y-2">
                  <Label htmlFor="runnerType">Runner Type *</Label>
                  <select
                    id="runnerType"
                    value={inlineConfig.runnerType}
                    onChange={(e) => {
                      const newRunnerType = e.target.value as RunnerType;
                      setInlineConfig({
                        ...inlineConfig,
                        runnerType: newRunnerType,
                        model: getDefaultModelForRunner(newRunnerType),
                      });
                    }}
                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                  >
                    {RUNNER_TYPES.map((type) => (
                      <option key={type} value={type}>
                        {type}
                      </option>
                    ))}
                  </select>
                </div>

                <ModelSelector
                  value={inlineConfig.model}
                  onChange={(model) => setInlineConfig({ ...inlineConfig, model })}
                  models={getModelsForRunner(inlineConfig.runnerType)}
                  label="Model"
                  placeholder="Enter custom model..."
                />

                <div className="grid gap-4 grid-cols-2">
                  <div className="space-y-2">
                    <Label htmlFor="maxTurns">Max Turns</Label>
                    <Input
                      id="maxTurns"
                      type="number"
                      value={inlineConfig.maxTurns}
                      onChange={(e) =>
                        setInlineConfig({
                          ...inlineConfig,
                          maxTurns: parseInt(e.target.value) || 100,
                        })
                      }
                      min={1}
                      max={1000}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="timeoutMinutes">Timeout (minutes)</Label>
                    <Input
                      id="timeoutMinutes"
                      type="number"
                      value={inlineConfig.timeoutMinutes}
                      onChange={(e) =>
                        setInlineConfig({
                          ...inlineConfig,
                          timeoutMinutes: parseInt(e.target.value) || 30,
                        })
                      }
                      min={1}
                      max={1440}
                    />
                  </div>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="runMode">Run Mode *</Label>
                  <select
                    id="runMode"
                    value={inlineConfig.runMode}
                    onChange={(e) =>
                      setInlineConfig({
                        ...inlineConfig,
                        runMode: e.target.value as "sandboxed" | "in_place",
                      })
                    }
                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                  >
                    <option value="sandboxed">Sandboxed (isolated copy)</option>
                    <option value="in_place">In-place (direct changes)</option>
                  </select>
                  <p className="text-xs text-muted-foreground">
                    Sandboxed runs in an isolated copy; in-place modifies files directly.
                  </p>
                </div>

                <div className="flex gap-6">
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={inlineConfig.skipPermissionPrompt}
                      onChange={(e) =>
                        setInlineConfig({
                          ...inlineConfig,
                          skipPermissionPrompt: e.target.checked,
                        })
                      }
                      className="h-4 w-4 rounded border-input"
                    />
                    <span className="text-sm">Skip Permission Prompts</span>
                  </label>
                </div>
              </TabsContent>
            </Tabs>
          </DialogBody>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowRunDialog(null)}
            >
              Cancel
            </Button>
            <Button
              onClick={handleStartRun}
              disabled={
                submitting ||
                (runConfigMode === "profile" && !selectedProfileId)
              }
              className="gap-2"
            >
              <Play className="h-4 w-4" />
              {submitting ? "Starting..." : "Start Run"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Create Profile Modal (from Run dialog) */}
      <Dialog open={showProfileDialog} onOpenChange={(open) => !open && resetProfileForm()}>
        <DialogContent>
          <DialogHeader onClose={resetProfileForm}>
            <DialogTitle>Create New Profile</DialogTitle>
            <DialogDescription>
              Create a reusable agent profile for running tasks
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleCreateProfile}>
            <DialogBody className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="profileName">Name *</Label>
                  <Input
                    id="profileName"
                    value={profileFormData.name}
                    onChange={(e) =>
                      setProfileFormData({ ...profileFormData, name: e.target.value })
                    }
                    placeholder="e.g., Claude Code Default"
                    required
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="profileRunnerType">Runner Type *</Label>
                  <select
                    id="profileRunnerType"
                    value={profileFormData.runnerType}
                    onChange={(e) => {
                      const newRunnerType = e.target.value as RunnerType;
                      setProfileFormData({
                        ...profileFormData,
                        runnerType: newRunnerType,
                        model: getDefaultModelForRunner(newRunnerType),
                      });
                    }}
                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                  >
                    {RUNNER_TYPES.map((type) => (
                      <option key={type} value={type}>
                        {type}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="profileDescription">Description</Label>
                <Textarea
                  id="profileDescription"
                  value={profileFormData.description}
                  onChange={(e) =>
                    setProfileFormData({ ...profileFormData, description: e.target.value })
                  }
                  placeholder="Describe what this profile is for..."
                  rows={2}
                />
              </div>

              <ModelSelector
                value={profileFormData.model}
                onChange={(model) => setProfileFormData({ ...profileFormData, model })}
                models={getModelsForRunner(profileFormData.runnerType)}
                label="Model"
                placeholder="Enter custom model..."
              />

              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="profileMaxTurns">Max Turns</Label>
                  <Input
                    id="profileMaxTurns"
                    type="number"
                    value={profileFormData.maxTurns}
                    onChange={(e) =>
                      setProfileFormData({
                        ...profileFormData,
                        maxTurns: parseInt(e.target.value) || 100,
                      })
                    }
                    min={1}
                    max={1000}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="profileTimeout">Timeout (minutes)</Label>
                  <Input
                    id="profileTimeout"
                    type="number"
                    value={profileTimeoutMinutes}
                    onChange={(e) =>
                      setProfileTimeoutMinutes(parseInt(e.target.value) || 30)
                    }
                    min={1}
                    max={1440}
                  />
                </div>
              </div>

              <div className="flex gap-6">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={profileFormData.requiresSandbox}
                    onChange={(e) =>
                      setProfileFormData({ ...profileFormData, requiresSandbox: e.target.checked })
                    }
                    className="h-4 w-4 rounded border-input"
                  />
                  <span className="text-sm">Require Sandbox</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={profileFormData.requiresApproval}
                    onChange={(e) =>
                      setProfileFormData({ ...profileFormData, requiresApproval: e.target.checked })
                    }
                    className="h-4 w-4 rounded border-input"
                  />
                  <span className="text-sm">Require Approval</span>
                </label>
              </div>
            </DialogBody>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={resetProfileForm}>
                Cancel
              </Button>
              <Button type="submit" disabled={submitting}>
                {submitting ? "Creating..." : "Create & Select"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Tasks List */}
      {loading ? (
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">
            Loading tasks...
          </CardContent>
        </Card>
      ) : filteredAndSortedTasks.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12 text-muted-foreground">
            <ClipboardList className="h-16 w-16 mb-4 opacity-50" />
            <h3 className="text-lg font-semibold mb-1">
              {tasks.length === 0 ? "No Tasks" : "No Matching Tasks"}
            </h3>
            <p className="text-sm mb-4">
              {tasks.length === 0
                ? "Create your first task to get started"
                : "Try adjusting your filters"}
            </p>
            {tasks.length === 0 && (
              <Button onClick={() => setShowForm(true)} className="gap-2">
                <Plus className="h-4 w-4" />
                Create Task
              </Button>
            )}
          </CardContent>
        </Card>
      ) : (
        <ScrollArea className="h-[600px]">
          <div className="space-y-4">
            {filteredAndSortedTasks.map((task) => (
              <Card key={task.id}>
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between">
                    <div className="space-y-1">
                      <div className="flex items-center gap-2">
                        <CardTitle className="text-base">{task.title}</CardTitle>
                        <Badge variant={task.status as "queued" | "running" | "failed"}>
                          {task.status.replace("_", " ")}
                        </Badge>
                      </div>
                      <CardDescription className="line-clamp-2">
                        {task.description || "No description provided"}
                      </CardDescription>
                    </div>
                    <div className="flex gap-2">
                      {task.status === "queued" && (
                        <>
                          <Button
                            variant="default"
                            size="sm"
                            onClick={() => setShowRunDialog(task)}
                            className="gap-1"
                          >
                            <Play className="h-3 w-3" />
                            Run
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleCancel(task.id)}
                            className="text-destructive hover:text-destructive"
                          >
                            <XCircle className="h-4 w-4" />
                          </Button>
                        </>
                      )}
                      {task.status === "cancelled" && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleDelete(task.id)}
                          className="text-destructive hover:text-destructive gap-1"
                        >
                          <Trash2 className="h-4 w-4" />
                          Delete
                        </Button>
                      )}
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="flex items-center gap-4 text-sm text-muted-foreground">
                    <div className="flex items-center gap-1">
                      <FolderOpen className="h-4 w-4" />
                      <code className="text-xs bg-muted px-1 py-0.5 rounded">
                        {task.scopePath}
                      </code>
                    </div>
                    <span>Created {formatRelativeTime(task.createdAt)}</span>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </ScrollArea>
      )}
    </div>
  );
}
