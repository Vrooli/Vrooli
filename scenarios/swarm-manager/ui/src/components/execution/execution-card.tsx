import { useState } from "react";
import { ArrowUpRight, Check, ChevronDown, ChevronUp, ExternalLink, Loader2, MessageSquare, AlertTriangle } from "lucide-react";
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
  type ReviewResult,
} from "../../types";

// ============================================================================
// ReviewStatusBadge
// ============================================================================

const REVIEW_DIMENSION_COLORS: Record<string, string> = {
  green: "bg-emerald-500",
  yellow: "bg-amber-500",
  red: "bg-red-500",
  skipped: "bg-slate-500",
};

function ReviewStatusBadge({
  result,
  onOpenSandbox,
}: {
  result: ReviewResult;
  onOpenSandbox?: () => void;
}) {
  const [showDimensions, setShowDimensions] = useState(false);
  const hasIssues = result.classification === "needs_work";
  const nonGreenDimensions = result.dimensions.filter((d) => d.status !== "green" && d.status !== "skipped");

  return (
    <div className="space-y-1.5" data-testid="review-status-badge">
      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation();
          if (result.dimensions.length > 0) setShowDimensions(!showDimensions);
        }}
        className={cn(
          "flex w-full items-center gap-2 rounded-md border px-2 py-1.5 text-xs transition-colors",
          result.classification === "ready" && "border-emerald-500/30 bg-emerald-500/10 text-emerald-300",
          result.classification === "ready_with_notes" && "border-amber-500/30 bg-amber-500/10 text-amber-300",
          result.classification === "needs_work" && "border-red-500/30 bg-red-500/10 text-red-300",
          result.classification === "not_assessable" && "border-slate-600 bg-slate-800/50 text-slate-400",
          result.dimensions.length > 0 && "cursor-pointer hover:border-white/20",
        )}
      >
        {result.classification === "ready" && <Check className="h-3.5 w-3.5 shrink-0" />}
        {result.classification === "ready_with_notes" && <AlertTriangle className="h-3.5 w-3.5 shrink-0" />}
        {result.classification === "needs_work" && <AlertTriangle className="h-3.5 w-3.5 shrink-0" />}
        <span className="flex-1 text-left">
          {result.classification === "ready" && "Checks passed"}
          {result.classification === "ready_with_notes" && "Passed with notes"}
          {result.classification === "needs_work" && "Issues found"}
          {result.classification === "not_assessable" && "Review inconclusive"}
        </span>
        {result.dimensions.length > 0 && (
          showDimensions
            ? <ChevronUp className="h-3 w-3 shrink-0 text-slate-500" />
            : <ChevronDown className="h-3 w-3 shrink-0 text-slate-500" />
        )}
      </button>

      {showDimensions && (
        <div className="space-y-1 rounded-md bg-slate-800/50 px-2.5 py-2">
          {result.dimensions.map((dim) => (
            <div key={dim.name} className="flex items-center gap-2 text-xs" data-testid={`review-dim-${dim.name}`}>
              <span className={cn("h-1.5 w-1.5 shrink-0 rounded-full", REVIEW_DIMENSION_COLORS[dim.status] ?? "bg-slate-500")} />
              <span className="text-slate-300">{dim.name}</span>
              {dim.details && <span className="text-slate-500">— {dim.details}</span>}
            </div>
          ))}
        </div>
      )}

      {hasIssues && onOpenSandbox && (
        <Button
          size="sm"
          variant="outline"
          className="w-full border-red-500/30 text-red-300 hover:bg-red-500/10 hover:text-red-200"
          onClick={(e) => {
            e.stopPropagation();
            onOpenSandbox();
          }}
          data-testid="review-open-sandbox"
        >
          <ExternalLink className="mr-1.5 h-3 w-3" />
          Review in Sandbox
        </Button>
      )}
    </div>
  );
}

// ============================================================================
// ExecutionCard
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
  trace,
  traceLoading = false,
  agentManagerUiUrl,
  testId,
}: ExecutionCardProps) {
  const [showDetails, setShowDetails] = useState(false);
  const backlogKindLabel = BACKLOG_KIND_LABELS[(item.backlogKind as BacklogKind)] ?? item.backlogKind;
  const canFollowUp = canFollowUpExecution(item.status) && onFollowUp;

  return (
    <article className="group block space-y-3" data-testid={testId}>
      {/* Header: status + mode */}
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <span
            className={`inline-block h-2 w-2 rounded-full ${EXECUTION_STATUS_COLORS[item.status] ?? "bg-slate-500"}`}
          />
          <span className="text-xs uppercase tracking-wider text-slate-400">
            {formatExecutionStatus(item.status)}
          </span>
        </div>
        <span className="rounded bg-slate-700/50 px-2 py-0.5 text-[11px] text-slate-400">
          {formatExecutionMode(item.mode)}
        </span>
      </div>

      {/* Title */}
      <h3 className="text-base font-medium text-slate-100">
        {backlogKindLabel}: {item.backlogName}
      </h3>

      {/* Metadata row: operation + source */}
      <div className="flex flex-wrap items-center gap-2 text-xs text-slate-500">
        {item.operation ? <span className="rounded bg-slate-700/60 px-1.5 py-0.5 text-slate-400">{item.operation}</span> : null}
        {item.startedBy ? <span>by {item.startedBy}</span> : null}
      </div>

      {/* Failure reason */}
      {item.failureReason ? (
        <p className="rounded-md border border-red-500/30 bg-red-500/10 px-2 py-1 text-xs text-red-300">
          {item.failureReason}
        </p>
      ) : null}

      {/* Review status badge */}
      {item.reviewResult ? (
        <ReviewStatusBadge
          result={item.reviewResult}
          onOpenSandbox={onOpenReviewSandbox ? () => onOpenReviewSandbox(item.executionId) : undefined}
        />
      ) : null}

      {/* Timestamps — both labeled */}
      <div className="flex items-center justify-between text-xs text-slate-500">
        <span title={new Date(item.updatedAt).toLocaleString()}>Updated {formatRelativeTime(item.updatedAt)}</span>
        <span title={new Date(item.createdAt).toLocaleString()}>Created {formatRelativeTime(item.createdAt)}</span>
      </div>

      {/* Primary actions: Start / Cancel / Retry / Follow Up */}
      {(canStart || canCancel || canRetry || canFollowUp) && (
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

      {/* Divider between primary and secondary actions */}
      <div className="border-t border-white/5" />

      {/* Secondary actions: navigation + trace */}
      <div className="flex flex-wrap items-center gap-2">
        <Button
          size="sm"
          variant="outline"
          className="border-slate-600/40 text-slate-400 hover:text-slate-200"
          onClick={(event) => {
            event.preventDefault();
            event.stopPropagation();
            onViewBacklog(item.backlogKind, item.backlogName);
          }}
        >
          <ArrowUpRight className="mr-1.5 h-3 w-3" />
          {backlogKindLabel}
        </Button>
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
          <p className="font-mono text-cyan-300">{trace.skill_id}</p>
          <p className="mt-1 text-slate-300">Captured {formatRelativeTime(trace.captured_at)}</p>
          <pre className="mt-2 max-h-40 overflow-auto whitespace-pre-wrap break-words font-mono text-[11px] text-slate-200">
            {trace.prompt}
          </pre>
        </div>
      ) : null}
    </article>
  );
}
