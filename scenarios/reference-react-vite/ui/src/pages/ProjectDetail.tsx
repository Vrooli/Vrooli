import { useState, useCallback } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Trash2, FolderKanban, Pause, CheckCircle, Archive, AlertCircle, Edit2, Save, X, Circle, Clock, CheckCircle2, ChevronRight } from "lucide-react";
import { fetchProject, updateProject, deleteProject, fetchTasks, type Project, type Task } from "../lib/api";
import { Button } from "../components/ui/button";
import { ErrorBoundary } from "../components/ErrorBoundary";
import { ConfirmDialog } from "../components/ConfirmDialog";

const projectStatusIcons: Record<Project["status"], React.ComponentType<{ className?: string }>> = {
  active: FolderKanban,
  paused: Pause,
  complete: CheckCircle,
  archived: Archive
};

const projectStatusColors: Record<Project["status"], string> = {
  active: "text-green-400",
  paused: "text-yellow-400",
  complete: "text-blue-400",
  archived: "text-slate-600"
};

const projectStatusLabels: Record<Project["status"], string> = {
  active: "Active",
  paused: "Paused",
  complete: "Complete",
  archived: "Archived"
};

const taskStatusIcons: Record<Task["status"], React.ComponentType<{ className?: string }>> = {
  pending: Circle,
  in_progress: Clock,
  completed: CheckCircle2,
  archived: Archive
};

const taskStatusColors: Record<Task["status"], string> = {
  pending: "text-slate-400",
  in_progress: "text-blue-400",
  completed: "text-green-400",
  archived: "text-slate-600"
};

function TaskRow({ task }: { task: Task }) {
  const StatusIcon = taskStatusIcons[task.status];

  return (
    <Link
      to={`/tasks/${task.id}`}
      data-testid={`project-task-${task.id}`}
      className="flex items-center justify-between gap-4 rounded-lg border border-white/10 bg-white/5 p-3 hover:bg-white/10 transition-colors"
    >
      <div className="flex items-center gap-3 min-w-0">
        <StatusIcon className={`h-5 w-5 flex-shrink-0 ${taskStatusColors[task.status]}`} />
        <span className={`truncate ${task.status === "completed" ? "text-slate-500 line-through" : ""}`}>
          {task.title}
        </span>
      </div>
      <ChevronRight className="h-4 w-4 text-slate-500 flex-shrink-0" />
    </Link>
  );
}

function ProjectDetailContent() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [isEditing, setIsEditing] = useState(false);
  const [editName, setEditName] = useState("");
  const [editDescription, setEditDescription] = useState("");
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  const { data: project, isLoading: projectLoading, error: projectError } = useQuery({
    queryKey: ["project", id],
    queryFn: () => fetchProject(id!),
    enabled: !!id
  });

  const { data: tasksData, isLoading: tasksLoading, error: tasksError } = useQuery({
    queryKey: ["tasks", { project_id: id }],
    queryFn: () => fetchTasks({ project_id: id, limit: 50 }),
    enabled: !!id
  });

  const updateMutation = useMutation({
    mutationFn: (updates: Parameters<typeof updateProject>[1]) => updateProject(id!, updates),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["project", id] });
      void queryClient.invalidateQueries({ queryKey: ["projects"] });
      setIsEditing(false);
    }
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteProject(id!),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["projects"] });
      navigate("/projects");
    }
  });

  const handleStartEdit = useCallback(() => {
    if (project) {
      setEditName(project.name);
      setEditDescription(project.description ?? "");
      setIsEditing(true);
    }
  }, [project]);

  const handleSaveEdit = useCallback(() => {
    updateMutation.mutate({
      name: editName.trim() || undefined,
      description: editDescription.trim() || undefined
    });
  }, [editName, editDescription, updateMutation]);

  const handleCancelEdit = useCallback(() => {
    setIsEditing(false);
  }, []);

  const cycleStatus = useCallback(() => {
    if (!project) return;
    const statusOrder: Project["status"][] = ["active", "paused", "complete"];
    const currentIndex = statusOrder.indexOf(project.status);
    const nextIndex = (currentIndex + 1) % statusOrder.length;
    const nextStatus = statusOrder[nextIndex];
    if (nextStatus) {
      updateMutation.mutate({ status: nextStatus });
    }
  }, [project, updateMutation]);

  const handleDeleteClick = useCallback(() => {
    setShowDeleteConfirm(true);
  }, []);

  const handleDeleteConfirm = useCallback(() => {
    deleteMutation.mutate();
  }, [deleteMutation]);

  const handleDeleteCancel = useCallback(() => {
    setShowDeleteConfirm(false);
  }, []);

  const tasks = tasksData?.data ?? [];

  if (projectLoading) {
    return (
      <div className="space-y-6" data-testid="project-detail-loading">
        <div className="h-8 w-24 animate-pulse rounded bg-white/5" />
        <div className="h-40 animate-pulse rounded-lg bg-white/5" />
        <div className="h-32 animate-pulse rounded-lg bg-white/5" />
      </div>
    );
  }

  if (projectError || !project) {
    return (
      <div data-testid="project-detail-error" className="space-y-6">
        <Link
          to="/projects"
          className="inline-flex items-center gap-2 text-sm text-slate-400 hover:text-white"
          data-testid="back-to-projects"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Projects
        </Link>
        <div className="rounded-lg border border-red-500/20 bg-red-500/10 p-6 text-center text-red-400">
          <AlertCircle className="mx-auto h-8 w-8" />
          <p className="mt-2">Project not found</p>
          <p className="mt-1 text-sm text-red-400/80">The project may have been deleted</p>
        </div>
      </div>
    );
  }

  const StatusIcon = projectStatusIcons[project.status];
  const borderColor = project.color ?? "#3B82F6";

  return (
    <div className="space-y-6" data-testid="project-detail-page">
      {/* Delete confirmation dialog */}
      <ConfirmDialog
        isOpen={showDeleteConfirm}
        title="Delete Project"
        message={`Are you sure you want to delete "${project.name}"? This action cannot be undone. Note: Tasks associated with this project will not be deleted.`}
        confirmLabel="Delete"
        onConfirm={handleDeleteConfirm}
        onCancel={handleDeleteCancel}
        variant="danger"
      />

      {/* Header with back link */}
      <div className="flex items-center justify-between">
        <Link
          to="/projects"
          className="inline-flex items-center gap-2 text-sm text-slate-400 hover:text-white"
          data-testid="back-to-projects"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Projects
        </Link>
        <div className="flex items-center gap-2">
          {!isEditing && (
            <Button variant="outline" size="sm" onClick={handleStartEdit} data-testid="edit-project">
              <Edit2 className="mr-2 h-4 w-4" />
              Edit
            </Button>
          )}
          <Button variant="destructive" size="sm" onClick={handleDeleteClick} data-testid="delete-project">
            <Trash2 className="mr-2 h-4 w-4" />
            Delete
          </Button>
        </div>
      </div>

      {/* Project details card */}
      <div
        className="rounded-lg border border-white/10 bg-white/5 p-6"
        style={{ borderLeftColor: borderColor, borderLeftWidth: "4px" }}
        data-testid="project-detail-card"
      >
        {isEditing ? (
          <div className="space-y-4">
            <input
              type="text"
              value={editName}
              onChange={(e) => setEditName(e.target.value)}
              className="w-full rounded-lg border border-white/10 bg-white/5 px-4 py-2 text-lg font-semibold text-white focus:border-white/20 focus:outline-none"
              data-testid="edit-name-input"
            />
            <textarea
              value={editDescription}
              onChange={(e) => setEditDescription(e.target.value)}
              placeholder="Add a description..."
              rows={3}
              className="w-full rounded-lg border border-white/10 bg-white/5 px-4 py-2 text-sm text-white placeholder:text-slate-500 focus:border-white/20 focus:outline-none resize-none"
              data-testid="edit-description-input"
            />
            <div className="flex gap-2">
              <Button onClick={handleSaveEdit} disabled={updateMutation.isPending} data-testid="save-edit">
                <Save className="mr-2 h-4 w-4" />
                Save
              </Button>
              <Button variant="outline" onClick={handleCancelEdit} data-testid="cancel-edit">
                <X className="mr-2 h-4 w-4" />
                Cancel
              </Button>
            </div>
          </div>
        ) : (
          <>
            <div className="flex items-start justify-between gap-4">
              <div className="flex items-start gap-3">
                <button
                  onClick={cycleStatus}
                  disabled={updateMutation.isPending}
                  className="mt-1 focus:outline-none focus:ring-2 focus:ring-white/20 rounded"
                  data-testid="project-detail-status-toggle"
                  title={`Status: ${projectStatusLabels[project.status]}`}
                >
                  <StatusIcon className={`h-6 w-6 ${projectStatusColors[project.status]}`} />
                </button>
                <div>
                  <h1 className="text-2xl font-semibold">{project.name}</h1>
                  {project.description && (
                    <p className="mt-2 text-slate-400">{project.description}</p>
                  )}
                </div>
              </div>
              {project.color && (
                <div
                  className="h-6 w-6 rounded-full border border-white/20"
                  style={{ backgroundColor: project.color }}
                  data-testid="project-color"
                  title={`Color: ${project.color}`}
                />
              )}
            </div>

            <div className="mt-4 flex flex-wrap gap-4 text-sm text-slate-500">
              <div data-testid="project-status">
                <span className="text-slate-600">Status:</span>{" "}
                <span className="capitalize">{projectStatusLabels[project.status]}</span>
              </div>
              <div data-testid="project-created">
                <span className="text-slate-600">Created:</span>{" "}
                {new Date(project.created_at).toLocaleDateString()}
              </div>
            </div>
          </>
        )}
      </div>

      {/* Tasks section */}
      <div className="space-y-4" data-testid="project-tasks-section">
        <div className="flex items-center justify-between">
          <h2 className="text-xl font-semibold">Tasks</h2>
          <Link to="/tasks">
            <Button variant="outline" size="sm" data-testid="view-all-tasks">
              View All Tasks
            </Button>
          </Link>
        </div>

        {tasksLoading ? (
          <div className="space-y-2" data-testid="project-tasks-loading">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="h-14 animate-pulse rounded-lg bg-white/5" />
            ))}
          </div>
        ) : tasksError ? (
          <div
            data-testid="project-tasks-error"
            className="rounded-lg border border-red-500/20 bg-red-500/10 p-6 text-center text-red-400"
          >
            <AlertCircle className="mx-auto h-8 w-8" />
            <p className="mt-2">Failed to load tasks</p>
          </div>
        ) : tasks.length === 0 ? (
          <div
            data-testid="project-tasks-empty"
            className="rounded-lg border border-white/10 bg-white/5 p-6 text-center"
          >
            <p className="text-slate-400">No tasks in this project</p>
            <p className="mt-1 text-sm text-slate-500">Create tasks and assign them to this project</p>
          </div>
        ) : (
          <div className="space-y-2" data-testid="project-tasks-list">
            {tasks.map((task) => (
              <TaskRow key={task.id} task={task} />
            ))}
          </div>
        )}

        {tasks.length > 0 && (
          <div className="text-sm text-slate-500" data-testid="project-tasks-count">
            {tasks.length} task{tasks.length !== 1 ? "s" : ""} in this project
          </div>
        )}
      </div>
    </div>
  );
}

/**
 * Project detail page component
 *
 * Shows full project information with edit capability and associated tasks.
 *
 * [REQ:P1-001a] Project detail view with full entity display
 * [REQ:P1-006b] API integration for project and tasks operations
 */
export function ProjectDetail() {
  return (
    <ErrorBoundary>
      <ProjectDetailContent />
    </ErrorBoundary>
  );
}
