import { Edit, FolderOpen, Play, Trash2, XCircle } from "lucide-react";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { MarkdownRenderer } from "./markdown";
import { formatDate, formatRelativeTime } from "../lib/utils";
import type { Task } from "../types";
import { TaskStatus } from "../types";

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

interface TaskDetailProps {
  task: Task;
  onEdit: (task: Task) => void;
  onRun: (task: Task) => void;
  onCancel: (taskId: string) => void;
  onDelete: (taskId: string) => void;
}

export function TaskDetail({
  task,
  onEdit,
  onRun,
  onCancel,
  onDelete,
}: TaskDetailProps) {
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="space-y-2">
        <div className="flex items-center gap-2 flex-wrap">
          <h3 className="text-lg font-semibold">{task.title}</h3>
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
        </div>
        {task.description ? (
          <div className="text-sm text-muted-foreground">
            <MarkdownRenderer content={task.description} />
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">No description provided</p>
        )}
      </div>

      {/* Actions */}
      <div className="flex flex-wrap gap-2">
        <Button
          variant="outline"
          size="sm"
          onClick={() => onEdit(task)}
          className="gap-1"
        >
          <Edit className="h-4 w-4" />
          Edit
        </Button>
        {task.status === TaskStatus.QUEUED && (
          <>
            <Button
              variant="default"
              size="sm"
              onClick={() => onRun(task)}
              className="gap-1"
            >
              <Play className="h-3 w-3" />
              Run
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => onCancel(task.id)}
              className="text-destructive hover:text-destructive gap-1"
            >
              <XCircle className="h-4 w-4" />
              Cancel
            </Button>
          </>
        )}
        {task.status === TaskStatus.CANCELLED && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => onDelete(task.id)}
            className="text-destructive hover:text-destructive gap-1"
          >
            <Trash2 className="h-4 w-4" />
            Delete
          </Button>
        )}
      </div>

      {/* Details */}
      <div className="space-y-4">
        <div className="space-y-2">
          <h4 className="text-sm font-medium">Scope Path</h4>
          <div className="flex items-center gap-2">
            <FolderOpen className="h-4 w-4 text-muted-foreground" />
            <code className="text-sm bg-muted px-2 py-1 rounded">
              {task.scopePath}
            </code>
          </div>
        </div>

        {task.projectRoot && (
          <div className="space-y-2">
            <h4 className="text-sm font-medium">Project Root</h4>
            <code className="text-sm bg-muted px-2 py-1 rounded">
              {task.projectRoot}
            </code>
          </div>
        )}

        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <span className="text-muted-foreground">Created</span>
            <p className="font-medium">{formatRelativeTime(task.createdAt)}</p>
          </div>
          <div>
            <span className="text-muted-foreground">Last Updated</span>
            <p className="font-medium">{formatDate(task.updatedAt)}</p>
          </div>
        </div>
      </div>
    </div>
  );
}
