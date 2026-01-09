import { useEffect, useMemo, useState } from "react";
import { timestampMs } from "@bufbuild/protobuf/wkt";
import {
  AlertCircle,
  ClipboardList,
  Edit,
  FolderOpen,
  Play,
  Plus,
  RefreshCw,
  Settings2,
  Trash2,
  XCircle,
} from "lucide-react";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent } from "../components/ui/card";
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
import { ModelConfigSelector, type ModelSelectionMode } from "../components/ModelConfigSelector";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import { Textarea } from "../components/ui/textarea";
import { formatRelativeTime, runnerTypeLabel, runnerTypeToSlug } from "../lib/utils";
import type { AgentProfile, ModelRegistry, ProfileFormData, Run, RunFormData, RunnerStatus, RunnerType, Task, TaskFormData } from "../types";
import { ModelPreset, RunMode, RunnerType as RunnerTypeEnum, TaskStatus } from "../types";

import { MasterDetailLayout, ListPanel, DetailPanel } from "../components/patterns/MasterDetail";
import { SearchToolbar, type FilterConfig, type SortOption } from "../components/patterns/SearchToolbar";
import { ListItem, ListItemTitle, ListItemSubtitle } from "../components/patterns/ListItem";
import { TaskDetail } from "../components/TaskDetail";
import { ContextAttachmentEditor } from "../components/ContextAttachmentEditor";
import { useViewportSize } from "../hooks/useViewportSize";

const RUNNER_TYPES: RunnerType[] = [
  RunnerTypeEnum.CLAUDE_CODE,
  RunnerTypeEnum.CODEX,
  RunnerTypeEnum.OPENCODE,
];

interface TasksPageProps {
  tasks: Task[];
  profiles: AgentProfile[];
  loading: boolean;
  error: string | null;
  onCreateTask: (task: TaskFormData) => Promise<Task>;
  onUpdateTask: (id: string, task: TaskFormData) => Promise<Task>;
  onCancelTask: (id: string) => Promise<void>;
  onDeleteTask: (id: string) => Promise<void>;
  onCreateRun: (run: RunFormData) => Promise<Run>;
  onCreateProfile: (profile: ProfileFormData) => Promise<AgentProfile>;
  onRefresh: () => void;
  runners?: Record<string, RunnerStatus>;
  modelRegistry?: ModelRegistry;
}

const taskStatusLabel = (status: TaskStatus): string => {
  switch (status) {
    case TaskStatus.QUEUED:
      return "queued";
    case TaskStatus.RUNNING:
      return "running";
    case TaskStatus.NEEDS_REVIEW:
      return "needs_review";
    case TaskStatus.APPROVED:
      return "approved";
    case TaskStatus.REJECTED:
      return "rejected";
    case TaskStatus.FAILED:
      return "failed";
    case TaskStatus.CANCELLED:
      return "cancelled";
    default:
      return "queued";
  }
};

const getModelId = (model: string | { id: string }): string => {
  return typeof model === "string" ? model : model.id;
};

interface InlineRunConfig {
  runnerType: RunnerType;
  model: string;
  modelPreset: ModelPreset;
  modelMode: ModelSelectionMode;
  maxTurns: number;
  timeoutMinutes: number;
  runMode: RunMode;
  skipPermissionPrompt: boolean;
  fallbackRunnerTypes: RunnerType[];
}

type ProfileFormState = ProfileFormData & {
  modelMode: ModelSelectionMode;
};

const STATUS_FILTER_OPTIONS = [
  { value: String(TaskStatus.QUEUED), label: "Queued" },
  { value: String(TaskStatus.RUNNING), label: "Running" },
  { value: String(TaskStatus.NEEDS_REVIEW), label: "Needs Review" },
  { value: String(TaskStatus.APPROVED), label: "Approved" },
  { value: String(TaskStatus.REJECTED), label: "Rejected" },
  { value: String(TaskStatus.FAILED), label: "Failed" },
  { value: String(TaskStatus.CANCELLED), label: "Cancelled" },
];

const SORT_OPTIONS: SortOption[] = [
  { value: "newest", label: "Newest First" },
  { value: "oldest", label: "Oldest First" },
  { value: "title", label: "Title A-Z" },
];

export function TasksPage({
  tasks,
  profiles,
  loading,
  error,
  onCreateTask,
  onUpdateTask,
  onCancelTask,
  onDeleteTask,
  onCreateRun,
  onCreateProfile,
  onRefresh,
  runners,
  modelRegistry,
}: TasksPageProps) {
  const { isDesktop } = useViewportSize();
  const getRegistryForRunner = (runnerType: RunnerType) => {
    return modelRegistry?.runners?.[runnerTypeToSlug(runnerType)];
  };

  const getModelsForRunner = (runnerType: RunnerType) => {
    const registry = getRegistryForRunner(runnerType);
    if (registry?.models?.length) {
      return registry.models;
    }
    const runner = runners?.[runnerType];
    return runner?.supportedModels ?? [];
  };

  const getPresetMapForRunner = (runnerType: RunnerType) => {
    return getRegistryForRunner(runnerType)?.presets ?? {};
  };

  // Selection state
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);

  // Modal state
  const [showForm, setShowForm] = useState(false);
  const [editingTask, setEditingTask] = useState<Task | null>(null);
  const [showRunDialog, setShowRunDialog] = useState<Task | null>(null);
  const [showProfileDialog, setShowProfileDialog] = useState(false);

  // Form state
  const [formData, setFormData] = useState<TaskFormData>({
    title: "",
    description: "",
    scopePath: ".",
    projectRoot: "",
    contextAttachments: [],
  });
  const [editFormData, setEditFormData] = useState<TaskFormData>({
    title: "",
    description: "",
    scopePath: ".",
    projectRoot: "",
    contextAttachments: [],
  });
  const [selectedProfileId, setSelectedProfileId] = useState("");
  const [runConfigMode, setRunConfigMode] = useState<"profile" | "custom">("profile");
  const [existingSandboxId, setExistingSandboxId] = useState("");
  const [inlineConfig, setInlineConfig] = useState<InlineRunConfig>({
    runnerType: RunnerTypeEnum.CLAUDE_CODE,
    model: "",
    modelPreset: ModelPreset.UNSPECIFIED,
    modelMode: "default",
    maxTurns: 100,
    timeoutMinutes: 30,
    runMode: RunMode.SANDBOXED,
    skipPermissionPrompt: true,
    fallbackRunnerTypes: [],
  });
  const [profileFormData, setProfileFormData] = useState<ProfileFormState>({
    name: "",
    profileKey: "",
    description: "",
    runnerType: RunnerTypeEnum.CLAUDE_CODE,
    model: "",
    modelPreset: ModelPreset.UNSPECIFIED,
    modelMode: "default",
    maxTurns: 100,
    requiresSandbox: true,
    requiresApproval: true,
    timeoutMinutes: 30,
    fallbackRunnerTypes: [],
  });
  const [submitting, setSubmitting] = useState(false);
  const [profileFormError, setProfileFormError] = useState<string | null>(null);

  // Filter/sort/search state
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [sortBy, setSortBy] = useState<string>("newest");

  const selectedTask = useMemo(
    () => tasks.find((t) => t.id === selectedTaskId) || null,
    [tasks, selectedTaskId]
  );

  const resetForm = () => {
    setFormData({
      title: "",
      description: "",
      scopePath: ".",
      projectRoot: "",
      contextAttachments: [],
    });
    setShowForm(false);
  };

  const resetEditForm = () => {
    setEditFormData({
      title: "",
      description: "",
      scopePath: ".",
      projectRoot: "",
      contextAttachments: [],
    });
    setEditingTask(null);
  };

  const handleEditTask = (task: Task) => {
    setEditingTask(task);
    setEditFormData({
      title: task.title,
      description: task.description ?? "",
      scopePath: task.scopePath,
      projectRoot: task.projectRoot ?? "",
      contextAttachments: task.contextAttachments ?? [],
    });
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

  const handleUpdate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingTask) return;
    setSubmitting(true);
    try {
      await onUpdateTask(editingTask.id, editFormData);
      resetEditForm();
    } catch (err) {
      console.error("Failed to update task:", err);
    } finally {
      setSubmitting(false);
    }
  };

  const handleStartRun = async () => {
    if (!showRunDialog) return;

    if (runConfigMode === "profile" && !selectedProfileId) return;

    setSubmitting(true);
    try {
      const request: RunFormData = {
        taskId: showRunDialog.id,
      };

      if (runConfigMode === "profile") {
        request.agentProfileId = selectedProfileId;
      } else {
        request.runnerType = inlineConfig.runnerType;
        if (inlineConfig.modelMode === "model" && inlineConfig.model.trim() !== "") {
          request.model = inlineConfig.model;
        }
        if (inlineConfig.modelMode === "preset") {
          request.modelPreset = inlineConfig.modelPreset;
        }
        request.maxTurns = inlineConfig.maxTurns;
        request.timeoutMinutes = inlineConfig.timeoutMinutes;
        request.runMode = inlineConfig.runMode;
        request.skipPermissionPrompt = inlineConfig.skipPermissionPrompt;
        if (inlineConfig.fallbackRunnerTypes.length > 0) {
          request.fallbackRunnerTypes = inlineConfig.fallbackRunnerTypes;
        }
      }
      if (existingSandboxId.trim() !== "") {
        request.existingSandboxId = existingSandboxId.trim();
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
      profileKey: "",
      description: "",
      runnerType: RunnerTypeEnum.CLAUDE_CODE,
      model: "",
      modelPreset: ModelPreset.UNSPECIFIED,
      modelMode: "default",
      maxTurns: 100,
      requiresSandbox: true,
      requiresApproval: true,
      timeoutMinutes: 30,
      fallbackRunnerTypes: [],
    });
    setShowProfileDialog(false);
    setProfileFormError(null);
  };

  const handleCreateProfile = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setProfileFormError(null);
    try {
      const newProfile = await onCreateProfile({
        ...profileFormData,
        model:
          profileFormData.modelMode === "model"
            ? profileFormData.model?.trim() ?? ""
            : "",
        modelPreset:
          profileFormData.modelMode === "preset"
            ? profileFormData.modelPreset ?? ModelPreset.FAST
            : ModelPreset.UNSPECIFIED,
        timeoutMinutes: profileFormData.timeoutMinutes ?? 30,
        fallbackRunnerTypes: profileFormData.fallbackRunnerTypes ?? [],
      });
      setSelectedProfileId(newProfile.id);
      setRunConfigMode("profile");
      resetProfileForm();
    } catch (err) {
      setProfileFormError((err as Error).message);
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
      if (selectedTaskId === taskId) {
        setSelectedTaskId(null);
      }
    } catch (err) {
      console.error("Failed to delete task:", err);
    }
  };

  const handleAddInlineFallback = () => {
    setInlineConfig((prev) => ({
      ...prev,
      fallbackRunnerTypes: [...prev.fallbackRunnerTypes, RunnerTypeEnum.CLAUDE_CODE],
    }));
  };

  const handleInlineFallbackChange = (index: number, value: string) => {
    const parsed = Number(value) as RunnerType;
    setInlineConfig((prev) => {
      const fallback = [...prev.fallbackRunnerTypes];
      fallback[index] = parsed;
      return { ...prev, fallbackRunnerTypes: fallback };
    });
  };

  const handleRemoveInlineFallback = (index: number) => {
    setInlineConfig((prev) => {
      const fallback = [...prev.fallbackRunnerTypes];
      fallback.splice(index, 1);
      return { ...prev, fallbackRunnerTypes: fallback };
    });
  };

  const handleAddProfileFallback = () => {
    setProfileFormData((prev) => ({
      ...prev,
      fallbackRunnerTypes: [...(prev.fallbackRunnerTypes ?? []), RunnerTypeEnum.CLAUDE_CODE],
    }));
  };

  const handleProfileFallbackChange = (index: number, value: string) => {
    const parsed = Number(value) as RunnerType;
    setProfileFormData((prev) => {
      const fallback = [...(prev.fallbackRunnerTypes ?? [])];
      fallback[index] = parsed;
      return { ...prev, fallbackRunnerTypes: fallback };
    });
  };

  const handleRemoveProfileFallback = (index: number) => {
    setProfileFormData((prev) => {
      const fallback = [...(prev.fallbackRunnerTypes ?? [])];
      fallback.splice(index, 1);
      return { ...prev, fallbackRunnerTypes: fallback };
    });
  };

  const filteredAndSortedTasks = useMemo(() => {
    let result = [...tasks];

    if (statusFilter !== "all") {
      const statusValue = Number(statusFilter) as TaskStatus;
      result = result.filter((t) => t.status === statusValue);
    }

    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      result = result.filter(
        (t) =>
          t.title.toLowerCase().includes(query) ||
          t.description?.toLowerCase().includes(query)
      );
    }

    result.sort((a, b) => {
      const createdAtA = a.createdAt ? timestampMs(a.createdAt) : 0;
      const createdAtB = b.createdAt ? timestampMs(b.createdAt) : 0;
      if (sortBy === "newest") {
        return createdAtB - createdAtA;
      } else if (sortBy === "oldest") {
        return createdAtA - createdAtB;
      } else {
        return a.title.localeCompare(b.title);
      }
    });

    return result;
  }, [tasks, statusFilter, searchQuery, sortBy]);

  useEffect(() => {
    if (!isDesktop) return;
    if (filteredAndSortedTasks.length === 0) return;

    const hasSelection =
      selectedTaskId !== null &&
      filteredAndSortedTasks.some((task) => task.id === selectedTaskId);

    if (!hasSelection) {
      setSelectedTaskId(filteredAndSortedTasks[0].id);
    }
  }, [filteredAndSortedTasks, isDesktop, selectedTaskId]);

  const filters: FilterConfig[] = [
    {
      id: "status",
      label: "Filter by status",
      value: statusFilter,
      options: STATUS_FILTER_OPTIONS,
      onChange: setStatusFilter,
      allLabel: "All Status",
    },
  ];

  const listPanel = (
    <ListPanel
      title="Tasks"
      count={filteredAndSortedTasks.length}
      loading={loading}
      headerActions={
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={onRefresh}>
            <RefreshCw className="h-4 w-4" />
          </Button>
          <Button size="sm" onClick={() => setShowForm(true)} className="gap-1">
            <Plus className="h-4 w-4" />
            <span className="hidden sm:inline">New</span>
          </Button>
        </div>
      }
      toolbar={
        <SearchToolbar
          searchValue={searchQuery}
          onSearchChange={setSearchQuery}
          searchPlaceholder="Search tasks..."
          filters={filters}
          sortOptions={SORT_OPTIONS}
          currentSort={sortBy}
          onSortChange={setSortBy}
        />
      }
      empty={
        <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
          <ClipboardList className="h-12 w-12 mb-3 opacity-50" />
          <p className="font-medium">
            {tasks.length === 0 ? "No Tasks" : "No Matching Tasks"}
          </p>
          <p className="text-sm text-center mt-1">
            {tasks.length === 0
              ? "Create your first task to get started"
              : "Try adjusting your filters"}
          </p>
          {tasks.length === 0 && (
            <Button
              onClick={() => setShowForm(true)}
              className="gap-2 mt-4"
              size="sm"
            >
              <Plus className="h-4 w-4" />
              Create Task
            </Button>
          )}
        </div>
      }
    >
      {filteredAndSortedTasks.map((task) => (
        <ListItem
          key={task.id}
          selected={selectedTaskId === task.id}
          onClick={() => setSelectedTaskId(task.id)}
          icon={<FolderOpen className="h-4 w-4 text-muted-foreground flex-shrink-0" />}
          actions={
            <Badge
              variant={
                taskStatusLabel(task.status) as
                  | "queued"
                  | "running"
                  | "needs_review"
                  | "approved"
                  | "rejected"
                  | "failed"
                  | "cancelled"
              }
            >
              {taskStatusLabel(task.status).replace("_", " ")}
            </Badge>
          }
        >
          <ListItemTitle>{task.title}</ListItemTitle>
          <ListItemSubtitle>
            {task.scopePath} | {formatRelativeTime(task.createdAt)}
          </ListItemSubtitle>
        </ListItem>
      ))}
    </ListPanel>
  );

  const detailPanel = (
    <DetailPanel
      title="Task Details"
      hasSelection={!!selectedTask}
      empty={
        <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
          <ClipboardList className="h-12 w-12 mb-3 opacity-50" />
          <p className="text-sm">Select a task to view details</p>
        </div>
      }
    >
      {selectedTask && (
        <TaskDetail
          task={selectedTask}
          onEdit={handleEditTask}
          onRun={(task) => setShowRunDialog(task)}
          onCancel={handleCancel}
          onDelete={handleDelete}
        />
      )}
    </DetailPanel>
  );

  // Build header content with error banner
  const headerContent = error ? (
    <Card className="border-destructive/50 bg-destructive/10">
      <CardContent className="flex items-center gap-3 py-4">
        <AlertCircle className="h-4 w-4 text-destructive" />
        <p className="text-sm text-destructive">{error}</p>
      </CardContent>
    </Card>
  ) : null;

  return (
    <>
      <MasterDetailLayout
        storageKey="tasks"
        headerContent={headerContent}
        listPanel={listPanel}
        detailPanel={detailPanel}
        selectedId={selectedTaskId}
        onDeselect={() => setSelectedTaskId(null)}
        detailTitle={selectedTask?.title ?? "Task Details"}
      />

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
              <ContextAttachmentEditor
                attachments={formData.contextAttachments || []}
                onChange={(attachments) =>
                  setFormData({ ...formData, contextAttachments: attachments })
                }
              />
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

      {/* Edit Task Modal */}
      <Dialog open={editingTask !== null} onOpenChange={(open) => !open && resetEditForm()}>
        <DialogContent>
          <DialogHeader onClose={resetEditForm}>
            <DialogTitle>Edit Task</DialogTitle>
            <DialogDescription>
              Update task details and scope
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleUpdate}>
            <DialogBody className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="editTitle">Title *</Label>
                <Input
                  id="editTitle"
                  value={editFormData.title}
                  onChange={(e) =>
                    setEditFormData({ ...editFormData, title: e.target.value })
                  }
                  placeholder="e.g., Fix authentication bug"
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="editDescription">Description</Label>
                <Textarea
                  id="editDescription"
                  value={editFormData.description}
                  onChange={(e) =>
                    setEditFormData({ ...editFormData, description: e.target.value })
                  }
                  placeholder="Detailed instructions for the agent..."
                  rows={4}
                />
              </div>

              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="editScopePath">Scope Path *</Label>
                  <Input
                    id="editScopePath"
                    value={editFormData.scopePath}
                    onChange={(e) =>
                      setEditFormData({ ...editFormData, scopePath: e.target.value })
                    }
                    placeholder="e.g., src/auth"
                    required
                  />
                  <p className="text-xs text-muted-foreground">
                    The directory scope where the agent can operate
                  </p>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="editProjectRoot">Project Root</Label>
                  <Input
                    id="editProjectRoot"
                    value={editFormData.projectRoot}
                    onChange={(e) =>
                      setEditFormData({ ...editFormData, projectRoot: e.target.value })
                    }
                    placeholder="Optional: /path/to/project"
                  />
                </div>
              </div>
              <ContextAttachmentEditor
                attachments={editFormData.contextAttachments || []}
                onChange={(attachments) =>
                  setEditFormData({ ...editFormData, contextAttachments: attachments })
                }
              />
            </DialogBody>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={resetEditForm}>
                Cancel
              </Button>
              <Button type="submit" disabled={submitting}>
                {submitting ? "Saving..." : "Save Changes"}
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
            setExistingSandboxId("");
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
                            {profile.name} ({runnerTypeLabel(profile.runnerType)})
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
                    <div className="space-y-2">
                      <Label htmlFor="existingSandboxIdProfile">Reuse Sandbox ID (optional)</Label>
                      <Input
                        id="existingSandboxIdProfile"
                        value={existingSandboxId}
                        onChange={(e) => setExistingSandboxId(e.target.value)}
                        placeholder="UUID of an existing sandbox to reuse"
                      />
                      <p className="text-xs text-muted-foreground">
                        Only applies to sandboxed runs. The sandbox must match the task scope.
                      </p>
                    </div>
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
                    value={String(inlineConfig.runnerType)}
                    onChange={(e) => {
                      const newRunnerType = Number(e.target.value) as RunnerType;
                      const availableModels = getModelsForRunner(newRunnerType);
                      const firstModel = availableModels.length > 0 ? getModelId(availableModels[0]) : "";
                      setInlineConfig({
                        ...inlineConfig,
                        runnerType: newRunnerType,
                        model:
                          inlineConfig.modelMode === "model"
                            ? firstModel
                            : inlineConfig.model,
                      });
                    }}
                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                  >
                    {RUNNER_TYPES.map((type) => (
                      <option key={type} value={type}>
                        {runnerTypeLabel(type)}
                      </option>
                    ))}
                  </select>
                </div>

                <ModelConfigSelector
                  value={{
                    mode: inlineConfig.modelMode,
                    model: inlineConfig.model,
                    preset: inlineConfig.modelPreset,
                  }}
                  onChange={(selection) =>
                    setInlineConfig({
                      ...inlineConfig,
                      modelMode: selection.mode,
                      model: selection.model,
                      modelPreset: selection.preset,
                    })
                  }
                  models={getModelsForRunner(inlineConfig.runnerType)}
                  presetMap={getPresetMapForRunner(inlineConfig.runnerType)}
                  label="Model Selection"
                />

                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <Label>Fallback Runners</Label>
                    <Button type="button" variant="outline" size="sm" onClick={handleAddInlineFallback}>
                      Add
                    </Button>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    Ordered runners to try if the primary runner is unavailable.
                  </p>
                  {inlineConfig.fallbackRunnerTypes.length === 0 ? (
                    <p className="text-xs text-muted-foreground">No fallback runners configured.</p>
                  ) : (
                    <div className="space-y-2">
                      {inlineConfig.fallbackRunnerTypes.map((runnerType, index) => (
                        <div key={`inline-fallback-${index}`} className="flex items-center gap-2">
                          <select
                            value={String(runnerType)}
                            onChange={(e) => handleInlineFallbackChange(index, e.target.value)}
                            className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                          >
                            {RUNNER_TYPES.map((type) => (
                              <option key={type} value={type}>
                                {runnerTypeLabel(type)}
                              </option>
                            ))}
                          </select>
                          <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            onClick={() => handleRemoveInlineFallback(index)}
                          >
                            Remove
                          </Button>
                        </div>
                      ))}
                    </div>
                  )}
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
                    value={String(inlineConfig.runMode)}
                    onChange={(e) =>
                      setInlineConfig({
                        ...inlineConfig,
                        runMode: Number(e.target.value) as RunMode,
                      })
                    }
                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                  >
                    <option value={RunMode.SANDBOXED}>Sandboxed (isolated copy)</option>
                    <option value={RunMode.IN_PLACE}>In-place (direct changes)</option>
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

                <div className="space-y-2">
                  <Label htmlFor="existingSandboxId">Reuse Sandbox ID (optional)</Label>
                  <Input
                    id="existingSandboxId"
                    value={existingSandboxId}
                    onChange={(e) => setExistingSandboxId(e.target.value)}
                    placeholder="UUID of an existing sandbox to reuse"
                  />
                  <p className="text-xs text-muted-foreground">
                    Only applies to sandboxed runs. The sandbox must match the task scope.
                  </p>
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
              {profileFormError && (
                <Card className="border-destructive/50 bg-destructive/10">
                  <CardContent className="flex items-center gap-3 py-3">
                    <AlertCircle className="h-4 w-4 text-destructive" />
                    <p className="text-sm text-destructive">{profileFormError}</p>
                  </CardContent>
                </Card>
              )}
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
                    value={String(profileFormData.runnerType)}
                    onChange={(e) => {
                      const newRunnerType = Number(e.target.value) as RunnerType;
                      const availableModels = getModelsForRunner(newRunnerType);
                      const firstModel = availableModels.length > 0 ? getModelId(availableModels[0]) : "";
                      setProfileFormData({
                        ...profileFormData,
                        runnerType: newRunnerType,
                        model:
                          profileFormData.modelMode === "model"
                            ? firstModel
                            : profileFormData.model,
                      });
                    }}
                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                  >
                    {RUNNER_TYPES.map((type) => (
                      <option key={type} value={type}>
                        {runnerTypeLabel(type)}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="profileKey">Profile Key</Label>
                <Input
                  id="profileKey"
                  value={profileFormData.profileKey ?? ""}
                  onChange={(e) =>
                    setProfileFormData({ ...profileFormData, profileKey: e.target.value })
                  }
                  placeholder="auto-generated from name if left blank"
                />
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

              <ModelConfigSelector
                value={{
                  mode: profileFormData.modelMode,
                  model: profileFormData.model ?? "",
                  preset: profileFormData.modelPreset ?? ModelPreset.UNSPECIFIED,
                }}
                onChange={(selection) =>
                  setProfileFormData({
                    ...profileFormData,
                    modelMode: selection.mode,
                    model: selection.model,
                    modelPreset: selection.preset,
                  })
                }
                models={getModelsForRunner(profileFormData.runnerType)}
                presetMap={getPresetMapForRunner(profileFormData.runnerType)}
                label="Model Selection"
              />

              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <Label>Fallback Runners</Label>
                  <Button type="button" variant="outline" size="sm" onClick={handleAddProfileFallback}>
                    Add
                  </Button>
                </div>
                <p className="text-xs text-muted-foreground">
                  Ordered runners to try if the primary runner is unavailable.
                </p>
                {(profileFormData.fallbackRunnerTypes ?? []).length === 0 ? (
                  <p className="text-xs text-muted-foreground">No fallback runners configured.</p>
                ) : (
                  <div className="space-y-2">
                    {(profileFormData.fallbackRunnerTypes ?? []).map((runnerType, index) => (
                      <div key={`profile-fallback-${index}`} className="flex items-center gap-2">
                        <select
                          value={String(runnerType)}
                          onChange={(e) => handleProfileFallbackChange(index, e.target.value)}
                          className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                        >
                          {RUNNER_TYPES.map((type) => (
                            <option key={type} value={type}>
                              {runnerTypeLabel(type)}
                            </option>
                          ))}
                        </select>
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          onClick={() => handleRemoveProfileFallback(index)}
                        >
                          Remove
                        </Button>
                      </div>
                    ))}
                  </div>
                )}
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
                    value={profileFormData.timeoutMinutes ?? 30}
                    onChange={(e) =>
                      setProfileFormData({
                        ...profileFormData,
                        timeoutMinutes: parseInt(e.target.value) || 30,
                      })
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
    </>
  );
}
