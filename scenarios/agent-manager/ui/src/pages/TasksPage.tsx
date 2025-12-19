import { useState } from "react";
import {
  AlertCircle,
  ClipboardList,
  FolderOpen,
  Play,
  Plus,
  RefreshCw,
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
import { Textarea } from "../components/ui/textarea";
import { formatRelativeTime } from "../lib/utils";
import type { AgentProfile, CreateRunRequest, CreateTaskRequest, Run, Task } from "../types";

interface TasksPageProps {
  tasks: Task[];
  profiles: AgentProfile[];
  loading: boolean;
  error: string | null;
  onCreateTask: (task: CreateTaskRequest) => Promise<Task>;
  onCancelTask: (id: string) => Promise<void>;
  onCreateRun: (run: CreateRunRequest) => Promise<Run>;
  onRefresh: () => void;
}

export function TasksPage({
  tasks,
  profiles,
  loading,
  error,
  onCreateTask,
  onCancelTask,
  onCreateRun,
  onRefresh,
}: TasksPageProps) {
  const [showForm, setShowForm] = useState(false);
  const [showRunDialog, setShowRunDialog] = useState<Task | null>(null);
  const [formData, setFormData] = useState<CreateTaskRequest>({
    title: "",
    description: "",
    scopePath: ".",
    projectRoot: "",
  });
  const [selectedProfileId, setSelectedProfileId] = useState("");
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
    if (!showRunDialog || !selectedProfileId) return;
    setSubmitting(true);
    try {
      await onCreateRun({
        taskId: showRunDialog.id,
        agentProfileId: selectedProfileId,
      });
      setShowRunDialog(null);
      setSelectedProfileId("");
    } catch (err) {
      console.error("Failed to start run:", err);
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
          }
        }}
      >
        <DialogContent>
          <DialogHeader onClose={() => setShowRunDialog(null)}>
            <DialogTitle>Start Run</DialogTitle>
            <DialogDescription>
              Select an agent profile to execute: {showRunDialog?.title}
            </DialogDescription>
          </DialogHeader>
          <DialogBody className="space-y-4">
            {profiles.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                No agent profiles available. Create a profile first.
              </p>
            ) : (
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
            )}
          </DialogBody>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowRunDialog(null)}
            >
              Cancel
            </Button>
            {profiles.length > 0 && (
              <Button
                onClick={handleStartRun}
                disabled={!selectedProfileId || submitting}
                className="gap-2"
              >
                <Play className="h-4 w-4" />
                {submitting ? "Starting..." : "Start Run"}
              </Button>
            )}
          </DialogFooter>
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
