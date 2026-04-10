import { useState, useCallback } from "react";
import { Link } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, Trash2, CheckCircle2, Circle, Clock, AlertCircle, Archive, ChevronRight } from "lucide-react";
import { fetchTasks, createTask, updateTask, deleteTask, type Task } from "../lib/api";
import { Button } from "../components/ui/button";
import { ErrorBoundary } from "../components/ErrorBoundary";
import { ConfirmDialog } from "../components/ConfirmDialog";

const statusIcons: Record<Task["status"], React.ComponentType<{ className?: string }>> = {
  pending: Circle,
  in_progress: Clock,
  completed: CheckCircle2,
  archived: Archive
};

const statusColors: Record<Task["status"], string> = {
  pending: "text-slate-400",
  in_progress: "text-blue-400",
  completed: "text-green-400",
  archived: "text-slate-600"
};

const priorityLabels: Record<number, { label: string; color: string }> = {
  1: { label: "Low", color: "bg-slate-600" },
  2: { label: "Medium", color: "bg-yellow-600" },
  3: { label: "High", color: "bg-orange-600" },
  4: { label: "Urgent", color: "bg-red-600" },
  5: { label: "Critical", color: "bg-red-700" }
};

function TaskForm({ onSubmit, isSubmitting }: { onSubmit: (title: string) => void; isSubmitting: boolean }) {
  const [title, setTitle] = useState("");

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (title.trim()) {
      onSubmit(title.trim());
      setTitle("");
    }
  };

  return (
    <form onSubmit={handleSubmit} className="flex gap-2" data-testid="task-form">
      <input
        type="text"
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        placeholder="Enter a new task..."
        className="flex-1 rounded-lg border border-white/10 bg-white/5 px-4 py-2 text-sm text-white placeholder:text-slate-500 focus:border-white/20 focus:outline-none"
        data-testid="task-input"
        disabled={isSubmitting}
      />
      <Button type="submit" disabled={!title.trim() || isSubmitting} data-testid="task-submit">
        <Plus className="mr-2 h-4 w-4" />
        Add Task
      </Button>
    </form>
  );
}

function TaskItem({
  task,
  onStatusChange,
  onDelete,
  isUpdating
}: {
  task: Task;
  onStatusChange: (status: Task["status"]) => void;
  onDelete: () => void;
  isUpdating: boolean;
}) {
  const StatusIcon = statusIcons[task.status];
  const defaultPriority = { label: "Medium", color: "bg-yellow-600" };
  const priority = priorityLabels[task.priority] ?? defaultPriority;

  const cycleStatus = () => {
    const statusOrder: Task["status"][] = ["pending", "in_progress", "completed"];
    const currentIndex = statusOrder.indexOf(task.status);
    const nextIndex = (currentIndex + 1) % statusOrder.length;
    const nextStatus = statusOrder[nextIndex];
    if (nextStatus) {
      onStatusChange(nextStatus);
    }
  };

  return (
    <div
      data-testid={`task-row-${task.id}`}
      className={`flex items-center justify-between gap-4 rounded-lg border border-white/10 bg-white/5 p-4 transition-opacity ${
        isUpdating ? "opacity-50" : ""
      }`}
    >
      <div className="flex items-center gap-3">
        <button
          onClick={cycleStatus}
          disabled={isUpdating}
          className="focus:outline-none focus:ring-2 focus:ring-white/20 rounded"
          data-testid={`task-status-toggle-${task.id}`}
          title={`Status: ${task.status.replace("_", " ")}`}
        >
          <StatusIcon className={`h-5 w-5 ${statusColors[task.status]}`} />
        </button>
        <div className="flex-1 min-w-0">
          <Link
            to={`/tasks/${task.id}`}
            className={`hover:underline ${task.status === "completed" ? "text-slate-500 line-through" : ""}`}
            data-testid={`task-link-${task.id}`}
          >
            {task.title}
          </Link>
          {task.description && (
            <p className="mt-0.5 text-sm text-slate-500 truncate">{task.description}</p>
          )}
        </div>
      </div>
      <div className="flex items-center gap-3">
        <span className={`rounded px-2 py-0.5 text-xs ${priority.color}`}>
          {priority.label}
        </span>
        <span className="text-xs text-slate-500 capitalize">
          {task.status.replace("_", " ")}
        </span>
        <button
          onClick={onDelete}
          disabled={isUpdating}
          className="rounded p-1 text-slate-500 hover:bg-white/10 hover:text-red-400 focus:outline-none focus:ring-2 focus:ring-white/20"
          data-testid={`task-delete-${task.id}`}
          title="Delete task"
        >
          <Trash2 className="h-4 w-4" />
        </button>
        <Link
          to={`/tasks/${task.id}`}
          className="rounded p-1 text-slate-500 hover:bg-white/10 hover:text-white focus:outline-none focus:ring-2 focus:ring-white/20"
          data-testid={`task-detail-link-${task.id}`}
          title="View details"
        >
          <ChevronRight className="h-4 w-4" />
        </Link>
      </div>
    </div>
  );
}

function TasksContent() {
  const queryClient = useQueryClient();
  const [updatingIds, setUpdatingIds] = useState<Set<string>>(new Set());
  const [deleteTarget, setDeleteTarget] = useState<Task | null>(null);

  const { data, isLoading, error } = useQuery({
    queryKey: ["tasks", {}],
    queryFn: () => fetchTasks({ limit: 100 })
  });

  const createMutation = useMutation({
    mutationFn: (title: string) => createTask({ title }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["tasks"] });
    }
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: Task["status"] }) =>
      updateTask(id, { status }),
    onMutate: ({ id }) => {
      setUpdatingIds((prev) => new Set(prev).add(id));
    },
    onSettled: (_, __, { id }) => {
      setUpdatingIds((prev) => {
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
      void queryClient.invalidateQueries({ queryKey: ["tasks"] });
    }
  });

  const deleteMutation = useMutation({
    mutationFn: deleteTask,
    onMutate: (id) => {
      setUpdatingIds((prev) => new Set(prev).add(id));
    },
    onSettled: (_, __, id) => {
      setUpdatingIds((prev) => {
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
      setDeleteTarget(null);
      void queryClient.invalidateQueries({ queryKey: ["tasks"] });
    }
  });

  const handleDeleteClick = useCallback((task: Task) => {
    setDeleteTarget(task);
  }, []);

  const handleDeleteConfirm = useCallback(() => {
    if (deleteTarget) {
      deleteMutation.mutate(deleteTarget.id);
    }
  }, [deleteTarget, deleteMutation]);

  const handleDeleteCancel = useCallback(() => {
    setDeleteTarget(null);
  }, []);

  const tasks = data?.data ?? [];

  return (
    <div className="space-y-6" data-testid="tasks-page">
      {/* Delete confirmation dialog for replay safety */}
      <ConfirmDialog
        isOpen={deleteTarget !== null}
        title="Delete Task"
        message={`Are you sure you want to delete "${deleteTarget?.title ?? ""}"? This action cannot be undone.`}
        confirmLabel="Delete"
        onConfirm={handleDeleteConfirm}
        onCancel={handleDeleteCancel}
        variant="danger"
      />

      <div>
        <h2 className="text-2xl font-semibold">Tasks</h2>
        <p className="mt-1 text-slate-400">Manage your tasks and track progress</p>
      </div>

      <TaskForm onSubmit={(title) => createMutation.mutate(title)} isSubmitting={createMutation.isPending} />

      {createMutation.isError && (
        <div
          data-testid="task-create-error"
          className="rounded-lg border border-red-500/20 bg-red-500/10 p-3 text-sm text-red-400"
        >
          <div className="flex items-center gap-2">
            <AlertCircle className="h-4 w-4" />
            <span>Failed to create task. Please try again.</span>
          </div>
        </div>
      )}

      {isLoading ? (
        <div className="space-y-3" data-testid="tasks-loading">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="h-16 animate-pulse rounded-lg bg-white/5" />
          ))}
        </div>
      ) : error ? (
        <div
          data-testid="tasks-error"
          className="rounded-lg border border-red-500/20 bg-red-500/10 p-6 text-center text-red-400"
        >
          <AlertCircle className="mx-auto h-8 w-8" />
          <p className="mt-2">Failed to load tasks</p>
          <p className="mt-1 text-sm text-red-400/80">Make sure the API is running</p>
        </div>
      ) : tasks.length === 0 ? (
        <div
          data-testid="tasks-empty"
          className="rounded-lg border border-white/10 bg-white/5 p-8 text-center"
        >
          <Circle className="mx-auto h-12 w-12 text-slate-600" />
          <p className="mt-3 text-lg text-slate-400">No tasks yet</p>
          <p className="mt-1 text-sm text-slate-500">Create your first task above to get started</p>
        </div>
      ) : (
        <div className="space-y-2" data-testid="tasks-list">
          {tasks.map((task) => (
            <TaskItem
              key={task.id}
              task={task}
              onStatusChange={(status) => updateMutation.mutate({ id: task.id, status })}
              onDelete={() => handleDeleteClick(task)}
              isUpdating={updatingIds.has(task.id)}
            />
          ))}
        </div>
      )}

      {tasks.length > 0 && (
        <div className="text-sm text-slate-500" data-testid="tasks-count">
          Showing {tasks.length} task{tasks.length !== 1 ? "s" : ""}
        </div>
      )}
    </div>
  );
}

export function Tasks() {
  return (
    <ErrorBoundary>
      <TasksContent />
    </ErrorBoundary>
  );
}
