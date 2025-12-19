import { useState } from "react";
import {
  AlertCircle,
  ClipboardList,
  FolderOpen,
  Play,
  Plus,
  RefreshCw,
  Settings2,
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
import { ScrollArea } from "../components/ui/scroll-area";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import { Textarea } from "../components/ui/textarea";
import { formatRelativeTime } from "../lib/utils";
import type { AgentProfile, CreateProfileRequest, CreateRunRequest, CreateTaskRequest, Run, RunnerType, Task } from "../types";

const RUNNER_TYPES: RunnerType[] = ["claude-code", "codex", "opencode"];

interface TasksPageProps {
  tasks: Task[];
  profiles: AgentProfile[];
  loading: boolean;
  error: string | null;
  onCreateTask: (task: CreateTaskRequest) => Promise<Task>;
  onCancelTask: (id: string) => Promise<void>;
  onCreateRun: (run: CreateRunRequest) => Promise<Run>;
  onCreateProfile: (profile: CreateProfileRequest) => Promise<AgentProfile>;
  onRefresh: () => void;
}

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
  onCreateRun,
  onCreateProfile,
  onRefresh,
}: TasksPageProps) {
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
    model: "claude-sonnet-4-20250514",
    maxTurns: 100,
    timeoutMinutes: 30,
    runMode: "sandboxed",
    skipPermissionPrompt: true,
  });
  const [profileFormData, setProfileFormData] = useState<CreateProfileRequest>({
    name: "",
    description: "",
    runnerType: "claude-code",
    model: "claude-sonnet-4-20250514",
    maxTurns: 100,
    requiresSandbox: true,
    requiresApproval: true,
  });
  const [profileTimeoutMinutes, setProfileTimeoutMinutes] = useState(30);
  const [submitting, setSubmitting] = useState(false);

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

  const sortedTasks = [...tasks].sort(
    (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
  );

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
                    onChange={(e) =>
                      setInlineConfig({
                        ...inlineConfig,
                        runnerType: e.target.value as RunnerType,
                      })
                    }
                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                  >
                    {RUNNER_TYPES.map((type) => (
                      <option key={type} value={type}>
                        {type}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="model">Model</Label>
                  <Input
                    id="model"
                    value={inlineConfig.model}
                    onChange={(e) =>
                      setInlineConfig({ ...inlineConfig, model: e.target.value })
                    }
                    placeholder="e.g., claude-sonnet-4-20250514"
                  />
                </div>

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
                    onChange={(e) =>
                      setProfileFormData({
                        ...profileFormData,
                        runnerType: e.target.value as RunnerType,
                      })
                    }
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

              <div className="space-y-2">
                <Label htmlFor="profileModel">Model</Label>
                <Input
                  id="profileModel"
                  value={profileFormData.model}
                  onChange={(e) =>
                    setProfileFormData({ ...profileFormData, model: e.target.value })
                  }
                  placeholder="e.g., claude-sonnet-4-20250514"
                />
              </div>

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
      ) : sortedTasks.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12 text-muted-foreground">
            <ClipboardList className="h-16 w-16 mb-4 opacity-50" />
            <h3 className="text-lg font-semibold mb-1">No Tasks</h3>
            <p className="text-sm mb-4">Create your first task to get started</p>
            <Button onClick={() => setShowForm(true)} className="gap-2">
              <Plus className="h-4 w-4" />
              Create Task
            </Button>
          </CardContent>
        </Card>
      ) : (
        <ScrollArea className="h-[600px]">
          <div className="space-y-4">
            {sortedTasks.map((task) => (
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
