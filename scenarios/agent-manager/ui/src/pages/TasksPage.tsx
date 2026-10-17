import { memo, useCallback, useEffect, useMemo, useState } from "react";
import { timestampMs } from "@bufbuild/protobuf/wkt";
import { useSearchParams } from "react-router-dom";
import {
  AlertCircle,
  ClipboardList,
  FolderOpen,
  Play,
  Plus,
  RefreshCw,
  Settings2,
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
import { RoleSelector } from "../components/RoleSelector";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import { Textarea } from "../components/ui/textarea";
import type { RolePolicyCatalog } from "@vrooli/proto-types/agent-manager/v1/api/service_pb";
import type { AgentProfile, ProfileFormData, Run, RunFormData, Task, TaskFormData } from "../types";
import { RunMode, TaskStatus } from "../types";
import { formatStandardRelativeTime } from "../lib/dateTime";

import { MasterDetailLayout, ListPanel, DetailPanel } from "../components/patterns/MasterDetail";
import { SearchToolbar, type FilterConfig, type SortOption } from "../components/patterns/SearchToolbar";
import { BoundedList, ListItem, ListItemTitle, ListItemSubtitle } from "../components/patterns/ListItem";
import { TaskDetail } from "../components/TaskDetail";
import { ContextAttachmentEditor } from "../components/ContextAttachmentEditor";
import { useViewportSize } from "../hooks/useViewportSize";
import { useTasksRunDialogState } from "../hooks/useTasksRunDialogState";

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
  rolePolicyCatalog?: RolePolicyCatalog;
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

type TaskStatusBadgeVariant =
  | "queued"
  | "running"
  | "needs_review"
  | "approved"
  | "rejected"
  | "failed"
  | "cancelled";

interface TaskListRowProps {
  task: Task;
  selected: boolean;
  onSelect: (taskId: string) => void;
}

const TaskListRow = memo(function TaskListRow({
  task,
  selected,
  onSelect,
}: TaskListRowProps) {
  const statusLabel = taskStatusLabel(task.status);

  return (
    <ListItem
      selected={selected}
      onClick={() => onSelect(task.id)}
      icon={<FolderOpen className="h-4 w-4 text-muted-foreground flex-shrink-0" />}
      actions={
        <Badge variant={statusLabel as TaskStatusBadgeVariant}>
          {statusLabel.replace("_", " ")}
        </Badge>
      }
    >
      <ListItemTitle>{task.title}</ListItemTitle>
      <ListItemSubtitle>
        {task.scopePath} | {formatStandardRelativeTime(task.createdAt)}
      </ListItemSubtitle>
    </ListItem>
  );
});

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
  rolePolicyCatalog,
}: TasksPageProps) {
  const { isDesktop } = useViewportSize();
  const [searchParams] = useSearchParams();
  const taskIdParam = searchParams.get("taskId");

  // Selection state
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);

  // Modal state
  const [showForm, setShowForm] = useState(false);
  const [editingTask, setEditingTask] = useState<Task | null>(null);

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
  const {
    showRunDialog,
    setShowRunDialog,
    showProfileDialog,
    setShowProfileDialog,
    selectedProfileId,
    setSelectedProfileId,
    runConfigMode,
    setRunConfigMode,
    existingSandboxId,
    setExistingSandboxId,
    inlineConfig,
    setInlineConfig,
    profileFormData,
    setProfileFormData,
    profileFormError,
    setProfileFormError,
    resetProfileForm,
    resetRunDialog,
  } = useTasksRunDialogState();

  const [submitting, setSubmitting] = useState(false);

  // Filter/sort/search state
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [sortBy, setSortBy] = useState<string>("newest");

  const selectedTask = useMemo(
    () => tasks.find((t) => t.id === selectedTaskId) || null,
    [tasks, selectedTaskId]
  );

  useEffect(() => {
    if (!taskIdParam) return;
    setSelectedTaskId(taskIdParam);
  }, [taskIdParam]);

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
        request.roleRef = inlineConfig.roleRef;
        request.maxTurns = inlineConfig.maxTurns;
        request.timeoutMinutes = inlineConfig.timeoutMinutes;
		request.effort = inlineConfig.effort;
        request.runMode = inlineConfig.runMode;
        request.skipPermissionPrompt = inlineConfig.skipPermissionPrompt;
      }
      if (existingSandboxId.trim() !== "") {
        request.existingSandboxId = existingSandboxId.trim();
      }

      await onCreateRun(request);
      resetRunDialog();
    } catch (err) {
      console.error("Failed to start run:", err);
    } finally {
      setSubmitting(false);
    }
  };

  const handleCreateProfile = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setProfileFormError(null);
    try {
      const newProfile = await onCreateProfile({
        ...profileFormData,
        roleRef: profileFormData.roleRef.trim(),
        timeoutMinutes: profileFormData.timeoutMinutes ?? 30,
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
    if (taskIdParam) return;
    if (filteredAndSortedTasks.length === 0) return;

    const hasSelection =
      selectedTaskId !== null &&
      filteredAndSortedTasks.some((task) => task.id === selectedTaskId);

    if (!hasSelection) {
      const first = filteredAndSortedTasks[0];
      if (first) setSelectedTaskId(first.id);
    }
  }, [filteredAndSortedTasks, isDesktop, selectedTaskId, taskIdParam]);

  const getTaskKey = useCallback((task: Task) => task.id, []);
  const handleSelectTask = useCallback((taskId: string) => {
    setSelectedTaskId(taskId);
  }, []);
  const renderTaskRow = useCallback(
    (task: Task) => (
      <TaskListRow
        task={task}
        selected={selectedTaskId === task.id}
        onSelect={handleSelectTask}
      />
    ),
    [handleSelectTask, selectedTaskId]
  );

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
      <BoundedList
        items={filteredAndSortedTasks}
        getKey={getTaskKey}
        renderItem={renderTaskRow}
      />
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
      <Dialog open={showForm} onOpenChange={(open) => !open && resetForm()} fullScreenMobile>
        <DialogContent fullScreenMobile>
          <DialogHeader onClose={resetForm}>
            <DialogTitle>Create New Task</DialogTitle>
            <DialogDescription>
              Define the work that an agent should perform
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="flex flex-col flex-1 min-h-0 overflow-hidden">
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
      <Dialog open={editingTask !== null} onOpenChange={(open) => !open && resetEditForm()} fullScreenMobile>
        <DialogContent fullScreenMobile>
          <DialogHeader onClose={resetEditForm}>
            <DialogTitle>Edit Task</DialogTitle>
            <DialogDescription>
              Update task details and scope
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleUpdate} className="flex flex-col flex-1 min-h-0 overflow-hidden">
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
            resetRunDialog();
          }
        }}
      >
        <DialogContent className="max-w-lg">
          <DialogHeader onClose={resetRunDialog}>
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
                            {profile.name} ({profile.roleRef})
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
                <RoleSelector
                  catalog={rolePolicyCatalog}
                  value={inlineConfig.roleRef}
                  onChange={(roleRef) => setInlineConfig({ ...inlineConfig, roleRef })}
                  label="Execution Role"
                  id="inlineRoleRef"
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
                  <Label htmlFor="inlineEffort">Reasoning Effort</Label>
                  <select
                    id="inlineEffort"
                    value={inlineConfig.effort}
                    onChange={(e) => setInlineConfig({ ...inlineConfig, effort: e.target.value as typeof inlineConfig.effort })}
                    className="h-9 w-full rounded-md border border-input bg-background px-2 text-sm"
                  >
                    <option value="">Runner default</option>
                    <option value="low">Low</option>
                    <option value="medium">Medium</option>
                    <option value="high">High</option>
                    <option value="xhigh">Extra high</option>
                    <option value="max">Max</option>
                  </select>
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
              onClick={resetRunDialog}
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
      <Dialog open={showProfileDialog} onOpenChange={(open) => !open && resetProfileForm()} fullScreenMobile>
        <DialogContent fullScreenMobile>
          <DialogHeader onClose={resetProfileForm}>
            <DialogTitle>Create New Profile</DialogTitle>
            <DialogDescription>
              Create a reusable agent profile for running tasks
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleCreateProfile} className="flex flex-col flex-1 min-h-0 overflow-hidden">
            <DialogBody className="space-y-4">
              {profileFormError && (
                <Card className="border-destructive/50 bg-destructive/10">
                  <CardContent className="flex items-center gap-3 py-3">
                    <AlertCircle className="h-4 w-4 text-destructive" />
                    <p className="text-sm text-destructive">{profileFormError}</p>
                  </CardContent>
                </Card>
              )}
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

              <RoleSelector
                catalog={rolePolicyCatalog}
                value={profileFormData.roleRef}
                onChange={(roleRef) => setProfileFormData({ ...profileFormData, roleRef })}
                label="Execution Role"
                id="profileRoleRef"
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

              <div className="space-y-2">
                <Label htmlFor="profileEffort">Reasoning Effort</Label>
                <select
                  id="profileEffort"
                  value={profileFormData.effort ?? ""}
                  onChange={(e) => setProfileFormData({ ...profileFormData, effort: e.target.value as ProfileFormData["effort"] })}
                  className="h-9 w-full rounded-md border border-input bg-background px-2 text-sm"
                >
                  <option value="">Runner default</option>
                  <option value="low">Low</option>
                  <option value="medium">Medium</option>
                  <option value="high">High</option>
                  <option value="xhigh">Extra high</option>
                  <option value="max">Max</option>
                </select>
              </div>

              <div className="flex gap-6">
                <label className="flex items-center gap-2">
                  <span className="text-sm">Sandbox Mode</span>
                  <select
                    value={profileFormData.sandboxMode ?? "protected"}
                    onChange={(e) =>
                      setProfileFormData({
                        ...profileFormData,
                        sandboxMode: e.target.value as "off" | "tracking" | "protected",
                      })
                    }
                    className="h-9 rounded-md border border-input bg-background px-2 text-sm"
                  >
                    <option value="off">Off</option>
                    <option value="tracking">Tracking</option>
                    <option value="protected">Protected</option>
                  </select>
                </label>
                {/* The "Require Approval" toggle was removed in
                    agent-sandbox-audit-foundation Phase 3b — operator-gated
                    apply lives on SandboxConfig.manualReview now. The
                    "Require Sandbox" boolean was removed in agent-manager
                    Phase 1: SandboxConfig.mode is the single source of
                    truth (see DeriveRunMode in domain/decisions.go). */}
                <label className="flex items-center gap-2">
                  <span className="text-sm">Network Access</span>
                  <select
                    value={profileFormData.networkAccess ?? "localhost"}
                    onChange={(e) =>
                      setProfileFormData({ ...profileFormData, networkAccess: e.target.value as "none" | "localhost" | "full" })
                    }
                    className="h-8 rounded border border-input bg-background px-2 text-sm"
                  >
                    <option value="none">None</option>
                    <option value="localhost">Localhost</option>
                    <option value="full">Full</option>
                  </select>
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
