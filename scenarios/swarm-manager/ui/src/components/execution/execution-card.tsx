import { useState } from "react";
import { ArrowUpRight, ChevronDown, ChevronUp, ExternalLink, Loader2, MessageSquare, RefreshCw } from "lucide-react";
import { Button } from "../ui/button";
import { cn, formatRelativeTime, canFollowUpExecution } from "../../lib";
import {
  BACKLOG_KIND_LABELS,
  EXECUTION_STATUS_COLORS,
  formatExecutionMode,
  formatExecutionStatus,
  type BacklogKind,
  type ExecutionRecord,
  type PromptTrace,
} from "../../types";
import { ReviewStatusBadge, ReviewValidatingIndicator, ReviewSkipIndicator } from "./review-status-badge";

// ============================================================================
// ExecutionCard
//
// Layout zones (top → bottom):
//   1. Header row:  status dot + label  |  operation badge + mode badge
//   2. Title:       clickable — navigates to the backlog item detail page
//   3. Meta row:    started-by  |  timestamps (updated / created)
//   4. Failure:     only when failureReason is present
//   5. Review:      ReviewStatusBadge / validating / skip indicator
//   6. Actions:     primary (Start/Cancel/Retry/Follow Up) + secondary (Run link, Trace, IDs)
// ============================================================================

interface ExecutionCardProps {
  item: ExecutionRecord;
  isBusy: boolean;
  canStart: boolean;
  canCancel: boolean;
  canRetry: boolean;
  onStart: (executionId: string) => void;
  onCancel: (executionId: string) => void;
  onRetry: (executionId: string) => void;
  onViewTrace: (executionId: string) => void;
  onViewBacklog: (backlogKind: string, backlogName: string) => void;
  onFollowUp?: (executionId: string) => void;
  onOpenReviewSandbox?: (executionId: string) => void;
  onTriggerReview?: (executionId: string) => void;
  trace?: PromptTrace;
  traceLoading?: boolean;
  agentManagerUiUrl?: string | null;
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
  onViewTrace,
  onViewBacklog,
  onFollowUp,
  onOpenReviewSandbox,
  onTriggerReview,
  trace,
  traceLoading = false,
  agentManagerUiUrl,
  testId,
}: ExecutionCardProps) {
  const [showDetails, setShowDetails] = useState(false);
  const backlogKindLabel = BACKLOG_KIND_LABELS[(item.backlogKind as BacklogKind)] ?? item.backlogKind;
  const canFollowUp = canFollowUpExecution(item.status) && onFollowUp;

  const hasPrimaryActions = canStart || canCancel || canRetry || canFollowUp;
  const hasReviewTrigger = !item.reviewResult && !item.reviewSkipReason && item.status !== "validating"
    && onTriggerReview && (item.status === "completed" || item.status === "needs_fixup");

  return (
    <article className="group block space-y-2.5" data-testid={testId}>
      {/* ── Zone 1: Header — status + badges ── */}
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <span
            className={`inline-block h-2 w-2 rounded-full ${EXECUTION_STATUS_COLORS[item.status] ?? "bg-slate-500"}`}
          />
          <span className="text-xs uppercase tracking-wider text-slate-400">
            {formatExecutionStatus(item.status)}
          </span>
        </div>
        <div className="flex items-center gap-1.5">
          {item.operation ? (
            <span className="rounded bg-slate-700/60 px-1.5 py-0.5 text-[11px] text-slate-400">
              {item.operation}
            </span>
          ) : null}
          <span className="rounded bg-slate-700/50 px-1.5 py-0.5 text-[11px] text-slate-400">
            {formatExecutionMode(item.mode)}
          </span>
        </div>
      </div>

      {/* ── Zone 2: Clickable title — navigates to backlog item ── */}
      <button
        type="button"
        className={cn(
          "w-full text-left text-base font-medium text-slate-100",
          "hover:text-cyan-300 transition-colors cursor-pointer",
          "flex items-center gap-1.5",
        )}
        onClick={(e) => {
          e.preventDefault();
          e.stopPropagation();
          onViewBacklog(item.backlogKind, item.backlogName);
        }}
        data-testid="execution-backlog-link"
      >
        <span className="truncate">
          {backlogKindLabel}: {item.backlogName}
        </span>
        <ArrowUpRight className="h-3.5 w-3.5 shrink-0 opacity-0 transition-opacity group-hover:opacity-100 text-cyan-400" />
      </button>

      {/* ── Zone 3: Meta row — source + timestamps ── */}
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-500">
        {item.startedBy ? <span>by {item.startedBy}</span> : null}
        <span className="ml-auto flex items-center gap-3">
          <span title={new Date(item.updatedAt).toLocaleString()}>Updated {formatRelativeTime(item.updatedAt)}</span>
          <span title={new Date(item.createdAt).toLocaleString()}>Created {formatRelativeTime(item.createdAt)}</span>
        </span>
      </div>

      {/* ── Zone 4: Failure reason ── */}
      {item.failureReason ? (
        <p className="rounded-md border border-red-500/30 bg-red-500/10 px-2 py-1 text-xs text-red-300">
          {item.failureReason}
        </p>
      ) : null}

      {/* ── Zone 5: Review status ── */}
      {item.reviewResult ? (
        <ReviewStatusBadge
          result={item.reviewResult}
          onOpenSandbox={onOpenReviewSandbox ? () => onOpenReviewSandbox(item.executionId) : undefined}
          onTriggerReview={onTriggerReview ? () => onTriggerReview(item.executionId) : undefined}
        />
      ) : item.status === "validating" ? (
        <ReviewValidatingIndicator />
      ) : item.reviewSkipReason ? (
        <ReviewSkipIndicator
          reason={item.reviewSkipReason}
          onTriggerReview={onTriggerReview ? () => onTriggerReview(item.executionId) : undefined}
        />
      ) : null}

      {/* Run Review button for terminal executions without a review */}
      {hasReviewTrigger && (
        <Button
          size="sm"
          variant="outline"
          onClick={(e) => {
            e.stopPropagation();
            onTriggerReview!(item.executionId);
          }}
          data-testid="review-trigger-button"
        >
          <RefreshCw className="mr-1.5 h-3 w-3" />
          Run Review
        </Button>
      )}

      {/* ── Zone 6: Actions ── */}
      {/* Primary actions: Start / Cancel / Retry / Follow Up */}
      {hasPrimaryActions && (
        <div className="flex flex-wrap gap-2">
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

          {canFollowUp && (
            <Button
              size="sm"
              variant="outline"
              onClick={(event) => {
                event.preventDefault();
                event.stopPropagation();
                onFollowUp(item.executionId);
              }}
              data-testid="follow-up-button"
            >
              <MessageSquare className="mr-1.5 h-3 w-3" />
              Follow Up
            </Button>
          )}
        </div>
      )}

      {/* Secondary actions: external links + trace + IDs */}
      <div className="flex flex-wrap items-center gap-2 border-t border-white/5 pt-2">
        {item.runId && agentManagerUiUrl ? (
          <a
            href={`${agentManagerUiUrl}/runs/${item.runId}`}
            target="_blank"
            rel="noopener noreferrer"
            onClick={(event) => event.stopPropagation()}
          >
            <Button size="sm" variant="outline" className="border-slate-600/40 text-slate-400 hover:text-slate-200">
              <ExternalLink className="mr-1.5 h-3 w-3" />
              Run
            </Button>
          </a>
        ) : null}
        <Button
          size="sm"
          variant="outline"
          className="border-slate-600/40 text-slate-400 hover:text-slate-200"
          onClick={() => onViewTrace(item.executionId)}
        >
          {traceLoading ? <Loader2 className="mr-1.5 h-3 w-3 animate-spin" /> : null}
          Trace
        </Button>

        {/* Details toggle for IDs */}
        <button
          type="button"
          onClick={() => setShowDetails(!showDetails)}
          className="ml-auto flex items-center gap-1 text-[11px] text-slate-500 hover:text-slate-300"
        >
          IDs
          {showDetails ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
        </button>
      </div>

      {/* Collapsible ID details */}
      {showDetails && (
        <div className="space-y-1 rounded-md bg-slate-800/50 px-2.5 py-2 font-mono text-[11px] text-slate-500">
          <p>exec {item.executionId}</p>
          {item.runId ? <p>run {item.runId}</p> : null}
          {item.taskId ? <p>task {item.taskId}</p> : null}
          {item.parentExecutionId ? <p>parent {item.parentExecutionId}</p> : null}
        </div>
      )}

      {/* Prompt trace */}
      {trace ? (
        <div className="rounded-md border border-cyan-500/30 bg-cyan-500/10 p-2 text-xs" data-testid="execution-prompt-trace">
          <p className="font-mono text-cyan-300">{trace.purpose}</p>
          <p className="mt-1 text-slate-300">Captured {formatRelativeTime(trace.captured_at)}</p>
          <pre className="mt-2 max-h-40 overflow-auto whitespace-pre-wrap break-words font-mono text-[11px] text-slate-200">
            {trace.prompt}
          </pre>
        </div>
      ) : null}
    </article>
  );
}
