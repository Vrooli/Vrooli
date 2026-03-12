import { useState, useCallback } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Plus, Trash2, CheckCircle2, Circle, Clock, Archive, AlertCircle, Edit2, Save, X } from "lucide-react";
import { fetchTask, updateTask, deleteTask, fetchNotes, createNote, deleteNote, type Task, type Note } from "../lib/api";
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

const statusLabels: Record<Task["status"], string> = {
  pending: "Pending",
  in_progress: "In Progress",
  completed: "Completed",
  archived: "Archived"
};

const priorityLabels: Record<number, { label: string; color: string }> = {
  1: { label: "Low", color: "bg-slate-600" },
  2: { label: "Medium", color: "bg-yellow-600" },
  3: { label: "High", color: "bg-orange-600" },
  4: { label: "Urgent", color: "bg-red-600" },
  5: { label: "Critical", color: "bg-red-700" }
};

function NoteItem({
  note,
  onDelete,
  isDeleting
}: {
  note: Note;
  onDelete: () => void;
  isDeleting: boolean;
}) {
  return (
    <div
      data-testid={`note-item-${note.id}`}
      className={`rounded-lg border border-white/10 bg-white/5 p-4 transition-opacity ${
        isDeleting ? "opacity-50" : ""
      }`}
    >
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1 min-w-0">
          <p className="whitespace-pre-wrap text-sm">{note.content}</p>
          <div className="mt-2 flex items-center gap-2 text-xs text-slate-500">
            {note.author && <span>by {note.author}</span>}
            <span>{new Date(note.created_at).toLocaleString()}</span>
          </div>
        </div>
        <button
          onClick={onDelete}
          disabled={isDeleting}
          className="rounded p-1 text-slate-500 hover:bg-white/10 hover:text-red-400 focus:outline-none focus:ring-2 focus:ring-white/20"
          data-testid={`note-delete-${note.id}`}
          title="Delete note"
        >
          <Trash2 className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
}

function NoteForm({ onSubmit, isSubmitting }: { onSubmit: (content: string) => void; isSubmitting: boolean }) {
  const [content, setContent] = useState("");

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (content.trim()) {
      onSubmit(content.trim());
      setContent("");
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-3" data-testid="note-form">
      <textarea
        value={content}
        onChange={(e) => setContent(e.target.value)}
        placeholder="Add a note..."
        rows={3}
        className="w-full rounded-lg border border-white/10 bg-white/5 px-4 py-2 text-sm text-white placeholder:text-slate-500 focus:border-white/20 focus:outline-none resize-none"
        data-testid="note-input"
        disabled={isSubmitting}
      />
      <Button type="submit" disabled={!content.trim() || isSubmitting} data-testid="note-submit">
        <Plus className="mr-2 h-4 w-4" />
        Add Note
      </Button>
    </form>
  );
}

function TaskDetailContent() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [isEditing, setIsEditing] = useState(false);
  const [editTitle, setEditTitle] = useState("");
  const [editDescription, setEditDescription] = useState("");
  const [deleteTarget, setDeleteTarget] = useState<"task" | Note | null>(null);
  const [deletingNoteIds, setDeletingNoteIds] = useState<Set<string>>(new Set());

  const { data: task, isLoading: taskLoading, error: taskError } = useQuery({
    queryKey: ["task", id],
    queryFn: () => fetchTask(id!),
    enabled: !!id
  });

  const { data: notesData, isLoading: notesLoading, error: notesError } = useQuery({
    queryKey: ["notes", id],
    queryFn: () => fetchNotes(id!, { limit: 100 }),
    enabled: !!id
  });

  const updateMutation = useMutation({
    mutationFn: (updates: Parameters<typeof updateTask>[1]) => updateTask(id!, updates),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["task", id] });
      void queryClient.invalidateQueries({ queryKey: ["tasks"] });
      setIsEditing(false);
    }
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteTask(id!),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["tasks"] });
      navigate("/tasks");
    }
  });

  const createNoteMutation = useMutation({
    mutationFn: (content: string) => createNote(id!, { content }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["notes", id] });
    }
  });

  const deleteNoteMutation = useMutation({
    mutationFn: deleteNote,
    onMutate: (noteId) => {
      setDeletingNoteIds((prev) => new Set(prev).add(noteId));
    },
    onSettled: (_, __, noteId) => {
      setDeletingNoteIds((prev) => {
        const next = new Set(prev);
        next.delete(noteId);
        return next;
      });
      setDeleteTarget(null);
      void queryClient.invalidateQueries({ queryKey: ["notes", id] });
    }
  });

  const handleStartEdit = useCallback(() => {
    if (task) {
      setEditTitle(task.title);
      setEditDescription(task.description ?? "");
      setIsEditing(true);
    }
  }, [task]);

  const handleSaveEdit = useCallback(() => {
    updateMutation.mutate({
      title: editTitle.trim() || undefined,
      description: editDescription.trim() || undefined
    });
  }, [editTitle, editDescription, updateMutation]);

  const handleCancelEdit = useCallback(() => {
    setIsEditing(false);
  }, []);

  const cycleStatus = useCallback(() => {
    if (!task) return;
    const statusOrder: Task["status"][] = ["pending", "in_progress", "completed"];
    const currentIndex = statusOrder.indexOf(task.status);
    const nextIndex = (currentIndex + 1) % statusOrder.length;
    const nextStatus = statusOrder[nextIndex];
    if (nextStatus) {
      updateMutation.mutate({ status: nextStatus });
    }
  }, [task, updateMutation]);

  const handleDeleteTaskClick = useCallback(() => {
    setDeleteTarget("task");
  }, []);

  const handleDeleteNoteClick = useCallback((note: Note) => {
    setDeleteTarget(note);
  }, []);

  const handleDeleteConfirm = useCallback(() => {
    if (deleteTarget === "task") {
      deleteMutation.mutate();
    } else if (deleteTarget && typeof deleteTarget === "object") {
      deleteNoteMutation.mutate(deleteTarget.id);
    }
  }, [deleteTarget, deleteMutation, deleteNoteMutation]);

  const handleDeleteCancel = useCallback(() => {
    setDeleteTarget(null);
  }, []);

  const notes = notesData?.data ?? [];

  if (taskLoading) {
    return (
      <div className="space-y-6" data-testid="task-detail-loading">
        <div className="h-8 w-24 animate-pulse rounded bg-white/5" />
        <div className="h-40 animate-pulse rounded-lg bg-white/5" />
        <div className="h-32 animate-pulse rounded-lg bg-white/5" />
      </div>
    );
  }

  if (taskError || !task) {
    return (
      <div data-testid="task-detail-error" className="space-y-6">
        <Link
          to="/tasks"
          className="inline-flex items-center gap-2 text-sm text-slate-400 hover:text-white"
          data-testid="back-to-tasks"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Tasks
        </Link>
        <div className="rounded-lg border border-red-500/20 bg-red-500/10 p-6 text-center text-red-400">
          <AlertCircle className="mx-auto h-8 w-8" />
          <p className="mt-2">Task not found</p>
          <p className="mt-1 text-sm text-red-400/80">The task may have been deleted</p>
        </div>
      </div>
    );
  }

  const StatusIcon = statusIcons[task.status];
  const defaultPriority = { label: "Medium", color: "bg-yellow-600" };
  const priority = priorityLabels[task.priority] ?? defaultPriority;

  return (
    <div className="space-y-6" data-testid="task-detail-page">
      {/* Delete confirmation dialogs */}
      <ConfirmDialog
        isOpen={deleteTarget === "task"}
        title="Delete Task"
        message={`Are you sure you want to delete "${task.title}"? This will also delete all notes. This action cannot be undone.`}
        confirmLabel="Delete"
        onConfirm={handleDeleteConfirm}
        onCancel={handleDeleteCancel}
        variant="danger"
      />
      <ConfirmDialog
        isOpen={deleteTarget !== null && deleteTarget !== "task"}
        title="Delete Note"
        message="Are you sure you want to delete this note? This action cannot be undone."
        confirmLabel="Delete"
        onConfirm={handleDeleteConfirm}
        onCancel={handleDeleteCancel}
        variant="danger"
      />

      {/* Header with back link */}
      <div className="flex items-center justify-between">
        <Link
          to="/tasks"
          className="inline-flex items-center gap-2 text-sm text-slate-400 hover:text-white"
          data-testid="back-to-tasks"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Tasks
        </Link>
        <div className="flex items-center gap-2">
          {!isEditing && (
            <Button variant="outline" size="sm" onClick={handleStartEdit} data-testid="edit-task">
              <Edit2 className="mr-2 h-4 w-4" />
              Edit
            </Button>
          )}
          <Button variant="destructive" size="sm" onClick={handleDeleteTaskClick} data-testid="delete-task">
            <Trash2 className="mr-2 h-4 w-4" />
            Delete
          </Button>
        </div>
      </div>

      {/* Task details card */}
      <div className="rounded-lg border border-white/10 bg-white/5 p-6" data-testid="task-detail-card">
        {isEditing ? (
          <div className="space-y-4">
            <input
              type="text"
              value={editTitle}
              onChange={(e) => setEditTitle(e.target.value)}
              className="w-full rounded-lg border border-white/10 bg-white/5 px-4 py-2 text-lg font-semibold text-white focus:border-white/20 focus:outline-none"
              data-testid="edit-title-input"
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
                  data-testid="task-detail-status-toggle"
                  title={`Status: ${statusLabels[task.status]}`}
                >
                  <StatusIcon className={`h-6 w-6 ${statusColors[task.status]}`} />
                </button>
                <div>
                  <h1 className={`text-2xl font-semibold ${task.status === "completed" ? "text-slate-500 line-through" : ""}`}>
                    {task.title}
                  </h1>
                  {task.description && (
                    <p className="mt-2 text-slate-400">{task.description}</p>
                  )}
                </div>
              </div>
              <span className={`rounded px-2 py-0.5 text-xs ${priority.color}`} data-testid="task-priority">
                {priority.label}
              </span>
            </div>

            <div className="mt-4 flex flex-wrap gap-4 text-sm text-slate-500">
              <div data-testid="task-status">
                <span className="text-slate-600">Status:</span>{" "}
                <span className="capitalize">{statusLabels[task.status]}</span>
              </div>
              {task.due_date && (
                <div data-testid="task-due-date">
                  <span className="text-slate-600">Due:</span>{" "}
                  {new Date(task.due_date).toLocaleDateString()}
                </div>
              )}
              <div data-testid="task-created">
                <span className="text-slate-600">Created:</span>{" "}
                {new Date(task.created_at).toLocaleDateString()}
              </div>
            </div>
          </>
        )}
      </div>

      {/* Notes section */}
      <div className="space-y-4" data-testid="notes-section">
        <h2 className="text-xl font-semibold">Notes</h2>

        <NoteForm
          onSubmit={(content) => createNoteMutation.mutate(content)}
          isSubmitting={createNoteMutation.isPending}
        />

        {createNoteMutation.isError && (
          <div
            data-testid="note-create-error"
            className="rounded-lg border border-red-500/20 bg-red-500/10 p-3 text-sm text-red-400"
          >
            <div className="flex items-center gap-2">
              <AlertCircle className="h-4 w-4" />
              <span>Failed to create note. Please try again.</span>
            </div>
          </div>
        )}

        {notesLoading ? (
          <div className="space-y-3" data-testid="notes-loading">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="h-20 animate-pulse rounded-lg bg-white/5" />
            ))}
          </div>
        ) : notesError ? (
          <div
            data-testid="notes-error"
            className="rounded-lg border border-red-500/20 bg-red-500/10 p-6 text-center text-red-400"
          >
            <AlertCircle className="mx-auto h-8 w-8" />
            <p className="mt-2">Failed to load notes</p>
          </div>
        ) : notes.length === 0 ? (
          <div
            data-testid="notes-empty"
            className="rounded-lg border border-white/10 bg-white/5 p-6 text-center"
          >
            <p className="text-slate-400">No notes yet</p>
            <p className="mt-1 text-sm text-slate-500">Add a note above to track progress</p>
          </div>
        ) : (
          <div className="space-y-2" data-testid="notes-list">
            {notes.map((note) => (
              <NoteItem
                key={note.id}
                note={note}
                onDelete={() => handleDeleteNoteClick(note)}
                isDeleting={deletingNoteIds.has(note.id)}
              />
            ))}
          </div>
        )}

        {notes.length > 0 && (
          <div className="text-sm text-slate-500" data-testid="notes-count">
            {notes.length} note{notes.length !== 1 ? "s" : ""}
          </div>
        )}
      </div>
    </div>
  );
}

/**
 * Task detail page component
 *
 * Shows full task information with edit capability and notes management.
 *
 * [REQ:P1-001a] Task detail view with full entity display
 * [REQ:P1-006b] API integration for task and notes operations
 */
export function TaskDetail() {
  return (
    <ErrorBoundary>
      <TaskDetailContent />
    </ErrorBoundary>
  );
}
