import { useState } from "react";
import { Edit, File, FolderOpen, Link2, Play, StickyNote, Tag, Trash2, XCircle } from "lucide-react";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { MarkdownRenderer } from "./markdown";
import { ContextAttachmentModal } from "./ContextAttachmentModal";
import { formatDate, formatRelativeTime } from "../lib/utils";
import type { ContextAttachmentData, Task } from "../types";
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
  const [selectedAttachment, setSelectedAttachment] = useState<ContextAttachmentData | null>(null);

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

        {task.contextAttachments && task.contextAttachments.length > 0 && (
          <div className="space-y-2">
            <h4 className="text-sm font-medium">Context Attachments</h4>
            <div className="space-y-2">
              {task.contextAttachments.map((att, index) => (
                <div
                  key={index}
                  className="flex items-start gap-2 p-2 bg-muted rounded-md text-sm cursor-pointer hover:bg-muted/70 transition-colors"
                  onClick={() => setSelectedAttachment(att as ContextAttachmentData)}
                >
                  {att.type === "file" && <File className="h-4 w-4 text-muted-foreground shrink-0 mt-0.5" />}
                  {att.type === "link" && <Link2 className="h-4 w-4 text-muted-foreground shrink-0 mt-0.5" />}
                  {att.type === "note" && <StickyNote className="h-4 w-4 text-muted-foreground shrink-0 mt-0.5" />}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      {att.key && (
                        <code className="text-xs bg-background px-1 py-0.5 rounded">
                          {att.key}
                        </code>
                      )}
                      {att.label && <span className="font-medium">{att.label}</span>}
                      {!att.key && !att.label && (
                        <span className="text-muted-foreground capitalize">{att.type}</span>
                      )}
                    </div>
                    {att.path && <p className="text-xs text-muted-foreground truncate">{att.path}</p>}
                    {att.url && <p className="text-xs text-muted-foreground truncate">{att.url}</p>}
                    {att.content && att.type === "note" && (
                      <p className="text-xs text-muted-foreground line-clamp-2 mt-1">{att.content}</p>
                    )}
                    {att.tags && att.tags.length > 0 && (
                      <div className="flex gap-1 mt-1 flex-wrap">
                        {att.tags.map((tag, i) => (
                          <Badge key={i} variant="outline" className="text-xs gap-1 py-0">
                            <Tag className="h-2.5 w-2.5" />
                            {tag}
                          </Badge>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
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

      <ContextAttachmentModal
        attachment={selectedAttachment}
        open={selectedAttachment !== null}
        onOpenChange={(open) => !open && setSelectedAttachment(null)}
      />
    </div>
  );
}
