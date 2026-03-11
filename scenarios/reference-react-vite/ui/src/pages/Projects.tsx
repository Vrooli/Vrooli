import { useState, useCallback } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, Trash2, FolderKanban, AlertCircle, Pause, CheckCircle, Archive } from "lucide-react";
import { fetchProjects, createProject, updateProject, deleteProject, type Project } from "../lib/api";
import { Button } from "../components/ui/button";
import { ErrorBoundary } from "../components/ErrorBoundary";
import { ConfirmDialog } from "../components/ConfirmDialog";

const statusIcons: Record<Project["status"], React.ComponentType<{ className?: string }>> = {
  active: FolderKanban,
  paused: Pause,
  complete: CheckCircle,
  archived: Archive
};

const statusColors: Record<Project["status"], string> = {
  active: "text-green-400",
  paused: "text-yellow-400",
  complete: "text-blue-400",
  archived: "text-slate-600"
};

const defaultColors = ["#3B82F6", "#10B981", "#F59E0B", "#EF4444", "#8B5CF6", "#EC4899"];

function ProjectForm({ onSubmit, isSubmitting }: { onSubmit: (name: string) => void; isSubmitting: boolean }) {
  const [name, setName] = useState("");

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (name.trim()) {
      onSubmit(name.trim());
      setName("");
    }
  };

  return (
    <form onSubmit={handleSubmit} className="flex gap-2" data-testid="project-form">
      <input
        type="text"
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder="Enter a new project name..."
        className="flex-1 rounded-lg border border-white/10 bg-white/5 px-4 py-2 text-sm text-white placeholder:text-slate-500 focus:border-white/20 focus:outline-none"
        data-testid="project-input"
        disabled={isSubmitting}
      />
      <Button type="submit" disabled={!name.trim() || isSubmitting} data-testid="project-submit">
        <Plus className="mr-2 h-4 w-4" />
        Add Project
      </Button>
    </form>
  );
}

function ProjectCard({
  project,
  onStatusChange,
  onDelete,
  isUpdating
}: {
  project: Project;
  onStatusChange: (status: Project["status"]) => void;
  onDelete: () => void;
  isUpdating: boolean;
}) {
  const StatusIcon = statusIcons[project.status];
  const borderColor = project.color ?? defaultColors[0];

  const cycleStatus = () => {
    const statusOrder: Project["status"][] = ["active", "paused", "complete"];
    const currentIndex = statusOrder.indexOf(project.status);
    const nextIndex = (currentIndex + 1) % statusOrder.length;
    const nextStatus = statusOrder[nextIndex];
    if (nextStatus) {
      onStatusChange(nextStatus);
    }
  };

  return (
    <div
      data-testid={`project-card-${project.id}`}
      className={`rounded-xl border border-white/10 bg-white/5 p-5 transition-opacity ${
        isUpdating ? "opacity-50" : ""
      }`}
      style={{ borderLeftColor: borderColor, borderLeftWidth: "4px" }}
    >
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <h3 className="font-medium">{project.name}</h3>
          {project.description && (
            <p className="mt-1 text-sm text-slate-400">{project.description}</p>
          )}
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={cycleStatus}
            disabled={isUpdating}
            className="rounded p-1 hover:bg-white/10 focus:outline-none focus:ring-2 focus:ring-white/20"
            data-testid={`project-status-toggle-${project.id}`}
            title={`Status: ${project.status}`}
          >
            <StatusIcon className={`h-5 w-5 ${statusColors[project.status]}`} />
          </button>
          <button
            onClick={onDelete}
            disabled={isUpdating}
            className="rounded p-1 text-slate-500 hover:bg-white/10 hover:text-red-400 focus:outline-none focus:ring-2 focus:ring-white/20"
            data-testid={`project-delete-${project.id}`}
            title="Delete project"
          >
            <Trash2 className="h-4 w-4" />
          </button>
        </div>
      </div>
      <div className="mt-4 flex items-center justify-between text-xs text-slate-500">
        <span className="capitalize">{project.status}</span>
        <span>Created {new Date(project.created_at).toLocaleDateString()}</span>
      </div>
    </div>
  );
}

function ProjectsContent() {
  const queryClient = useQueryClient();
  const [updatingIds, setUpdatingIds] = useState<Set<string>>(new Set());
  const [deleteTarget, setDeleteTarget] = useState<Project | null>(null);

  const { data, isLoading, error } = useQuery({
    queryKey: ["projects", {}],
    queryFn: () => fetchProjects({ limit: 100 })
  });

  const createMutation = useMutation({
    mutationFn: (name: string) => {
      const colorIndex = Math.floor(Math.random() * defaultColors.length);
      const color = defaultColors[colorIndex];
      return createProject({ name, color });
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["projects"] });
    }
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: Project["status"] }) =>
      updateProject(id, { status }),
    onMutate: ({ id }) => {
      setUpdatingIds((prev) => new Set(prev).add(id));
    },
    onSettled: (_, __, { id }) => {
      setUpdatingIds((prev) => {
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
      void queryClient.invalidateQueries({ queryKey: ["projects"] });
    }
  });

  const deleteMutation = useMutation({
    mutationFn: deleteProject,
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
      void queryClient.invalidateQueries({ queryKey: ["projects"] });
    }
  });

  const handleDeleteClick = useCallback((project: Project) => {
    setDeleteTarget(project);
  }, []);

  const handleDeleteConfirm = useCallback(() => {
    if (deleteTarget) {
      deleteMutation.mutate(deleteTarget.id);
    }
  }, [deleteTarget, deleteMutation]);

  const handleDeleteCancel = useCallback(() => {
    setDeleteTarget(null);
  }, []);

  const projects = data?.data ?? [];

  return (
    <div className="space-y-6" data-testid="projects-page">
      {/* Delete confirmation dialog for replay safety */}
      <ConfirmDialog
        isOpen={deleteTarget !== null}
        title="Delete Project"
        message={`Are you sure you want to delete "${deleteTarget?.name ?? ""}"? This action cannot be undone.`}
        confirmLabel="Delete"
        onConfirm={handleDeleteConfirm}
        onCancel={handleDeleteCancel}
        variant="danger"
      />

      <div>
        <h2 className="text-2xl font-semibold">Projects</h2>
        <p className="mt-1 text-slate-400">Organize your tasks into projects</p>
      </div>

      <ProjectForm
        onSubmit={(name) => createMutation.mutate(name)}
        isSubmitting={createMutation.isPending}
      />

      {createMutation.isError && (
        <div
          data-testid="project-create-error"
          className="rounded-lg border border-red-500/20 bg-red-500/10 p-3 text-sm text-red-400"
        >
          <div className="flex items-center gap-2">
            <AlertCircle className="h-4 w-4" />
            <span>Failed to create project. Please try again.</span>
          </div>
        </div>
      )}

      {isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3" data-testid="projects-loading">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="h-32 animate-pulse rounded-xl bg-white/5" />
          ))}
        </div>
      ) : error ? (
        <div
          data-testid="projects-error"
          className="rounded-lg border border-red-500/20 bg-red-500/10 p-6 text-center text-red-400"
        >
          <AlertCircle className="mx-auto h-8 w-8" />
          <p className="mt-2">Failed to load projects</p>
          <p className="mt-1 text-sm text-red-400/80">Make sure the API is running</p>
        </div>
      ) : projects.length === 0 ? (
        <div
          data-testid="projects-empty"
          className="rounded-lg border border-white/10 bg-white/5 p-8 text-center"
        >
          <FolderKanban className="mx-auto h-12 w-12 text-slate-600" />
          <p className="mt-3 text-lg text-slate-400">No projects yet</p>
          <p className="mt-1 text-sm text-slate-500">Create your first project above to get started</p>
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3" data-testid="projects-grid">
          {projects.map((project) => (
            <ProjectCard
              key={project.id}
              project={project}
              onStatusChange={(status) => updateMutation.mutate({ id: project.id, status })}
              onDelete={() => handleDeleteClick(project)}
              isUpdating={updatingIds.has(project.id)}
            />
          ))}
        </div>
      )}

      {projects.length > 0 && (
        <div className="text-sm text-slate-500" data-testid="projects-count">
          Showing {projects.length} project{projects.length !== 1 ? "s" : ""}
        </div>
      )}
    </div>
  );
}

export function Projects() {
  return (
    <ErrorBoundary>
      <ProjectsContent />
    </ErrorBoundary>
  );
}
