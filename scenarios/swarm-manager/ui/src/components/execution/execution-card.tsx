import { Loader2 } from "lucide-react";
import { Button } from "../ui/button";
import { formatRelativeTime } from "../../lib";
import {
  BACKLOG_KIND_LABELS,
  EXECUTION_STATUS_COLORS,
  formatExecutionMode,
  formatExecutionStatus,
  type ExecutionRecord,
} from "../../types";

interface ExecutionCardProps {
  item: ExecutionRecord;
  isBusy: boolean;
  canStart: boolean;
  canCancel: boolean;
  canRetry: boolean;
  onStart: (executionId: string) => void;
  onCancel: (executionId: string) => void;
  onRetry: (executionId: string) => void;
  testId?: string;
}

export function ExecutionCard({
  item,
  isBusy,
  canStart,
  canCancel,
  canRetry,
  onStart,
  onCancel,
  onRetry,
  testId,
}: ExecutionCardProps) {
  return (
    <article className="group block" data-testid={testId}>
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-center gap-2">
          <span
            className={`inline-block h-2 w-2 rounded-full ${EXECUTION_STATUS_COLORS[item.status] ?? "bg-slate-500"}`}
          />
          <span className="text-xs uppercase tracking-wider text-slate-400">
            {formatExecutionStatus(item.status)}
          </span>
        </div>
        <span className="rounded-full bg-slate-700 px-2 py-0.5 text-xs text-slate-300">
          {formatExecutionMode(item.mode)}
        </span>
      </div>

      <h3 className="mt-3 font-medium text-slate-100">
        {BACKLOG_KIND_LABELS[item.backlogKind]}: {item.backlogName}
      </h3>
      <p className="mt-1 truncate font-mono text-xs text-slate-500" title={item.executionId}>
        {item.executionId}
      </p>

      <div className="mt-2 flex flex-wrap items-center gap-2 text-xs text-slate-400">
        {item.operation ? <span className="rounded bg-slate-700/80 px-1.5 py-0.5">{item.operation}</span> : null}
        {item.startedBy ? <span title={item.startedBy}>by {item.startedBy}</span> : null}
        {item.runId ? <span title={item.runId}>run {item.runId}</span> : null}
        {item.taskId ? <span title={item.taskId}>task {item.taskId}</span> : null}
      </div>

      {item.failureReason ? (
        <p className="mt-3 rounded-md border border-red-500/30 bg-red-500/10 px-2 py-1 text-xs text-red-300">
          {item.failureReason}
        </p>
      ) : null}

      <div className="mt-4 flex items-center justify-between text-xs text-slate-400">
        <span title={new Date(item.updatedAt).toLocaleString()}>{formatRelativeTime(item.updatedAt)}</span>
        <span title={new Date(item.createdAt).toLocaleString()}>Created {formatRelativeTime(item.createdAt)}</span>
      </div>

      {(canStart || canCancel || canRetry) && (
        <div className="mt-3 flex flex-wrap gap-2">
          {canStart && (
            <Button
              size="sm"
              disabled={isBusy}
              onClick={(event) => {
                event.preventDefault();
                event.stopPropagation();
                onStart(item.executionId);
              }}
            >
              {isBusy ? <Loader2 className="mr-2 h-3.5 w-3.5 animate-spin" /> : null}
              Start
            </Button>
          )}

          {canCancel && (
            <Button
              size="sm"
              variant="outline"
              disabled={isBusy}
              onClick={(event) => {
                event.preventDefault();
                event.stopPropagation();
                onCancel(item.executionId);
              }}
            >
              {isBusy ? <Loader2 className="mr-2 h-3.5 w-3.5 animate-spin" /> : null}
              Cancel
            </Button>
          )}

          {canRetry && (
            <Button
              size="sm"
              variant="outline"
              disabled={isBusy}
              onClick={(event) => {
                event.preventDefault();
                event.stopPropagation();
                onRetry(item.executionId);
              }}
            >
              {isBusy ? <Loader2 className="mr-2 h-3.5 w-3.5 animate-spin" /> : null}
              Retry
            </Button>
          )}
        </div>
      )}
    </article>
  );
}
