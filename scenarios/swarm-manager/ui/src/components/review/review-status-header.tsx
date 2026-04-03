/**
 * ReviewStatusHeader — Single source of truth for post-execution status
 * display and primary action.
 *
 * Consolidates status rendering that was previously duplicated across
 * LatestExecutionSummary, PostRunStatusBadge (buttons), and
 * ScenarioReviewResults (inline service calls).
 *
 * The primary action button is a pure function of execution state.
 */

import { Loader2, Play, RefreshCw, Wrench, MessageSquare } from "lucide-react";
import { Button } from "../ui/button";
import { cn, formatRelativeTime, canFollowUpExecution } from "../../lib";
import { resolvePostRunExecution } from "../../lib/finalization";
import { EXECUTION_STATUS_COLORS, formatExecutionStatus } from "../../types";
import { selectors } from "../../consts/selectors";
import type { ExecutionRecord, ExecutionStatus } from "../../types";

export interface ReviewStatusHeaderProps {
  execution: ExecutionRecord | undefined;
  isActive: boolean;
  isTriggering: boolean;
  onTriggerReview: () => void;
  onFollowUp: (exec: ExecutionRecord) => void;
}

type PrimaryAction =
  | { kind: "none" }
  | { kind: "run-review" }
  | { kind: "triggering" }
  | { kind: "fix-issues"; exec: ExecutionRecord }
  | { kind: "follow-up"; exec: ExecutionRecord };

function resolvePrimaryAction(
  execution: ExecutionRecord | undefined,
  isActive: boolean,
  isTriggering: boolean,
): PrimaryAction {
  if (!execution || isActive) return { kind: "none" };
  if (isTriggering) return { kind: "triggering" };

  const resolved = resolvePostRunExecution(execution);
  const isTerminal = canFollowUpExecution(execution.status as ExecutionStatus);

  // Terminal execution with no finalization data — offer to run review
  if (isTerminal && !resolved) return { kind: "run-review" };

  // Finalization still running — no action (PostRunStatusBadge shows progress)
  if (resolved?.finalization?.status === "running" || resolved?.finalization?.status === "pending") {
    return { kind: "none" };
  }

  // Finalization complete with issues — offer fix
  if (resolved?.finalization?.aggregateClassification === "needs_work") {
    return { kind: "fix-issues", exec: execution };
  }

  // Finalization complete — offer general follow-up
  if (isTerminal) {
    return { kind: "follow-up", exec: execution };
  }

  return { kind: "none" };
}

export function ReviewStatusHeader({
  execution,
  isActive,
  isTriggering,
  onTriggerReview,
  onFollowUp,
}: ReviewStatusHeaderProps) {
  if (!execution) return null;
  if (isActive) return null; // Active run display is handled by LatestExecutionSummary

  const statusColor = EXECUTION_STATUS_COLORS[execution.status as ExecutionStatus] ?? "bg-slate-500";
  const action = resolvePrimaryAction(execution, isActive, isTriggering);
  const resolved = resolvePostRunExecution(execution);
  const hasFinalization = Boolean(resolved?.finalization);

  return (
    <div className="py-3" data-testid={selectors.review.statusHeader}>
      <div className="flex items-center gap-2">
        <span className={cn("inline-block h-2 w-2 shrink-0 rounded-full", statusColor)} />
        <span className="text-xs font-medium capitalize text-slate-200">
          {formatExecutionStatus(execution.status as ExecutionStatus)}
        </span>
        {execution.operation && (
          <span className="rounded bg-slate-700/60 px-1.5 py-0.5 text-[11px] font-medium text-slate-300">
            {execution.operation.replace("_", " ")}
          </span>
        )}
        <span className="text-xs text-slate-400">
          {formatRelativeTime(execution.createdAt)}
        </span>

        <div className="ml-auto flex items-center gap-2">
          {/* Secondary: Re-run Checks (when finalization exists) */}
          {hasFinalization && (
            <button
              type="button"
              className="text-[11px] text-slate-500 hover:text-slate-300 transition-colors"
              onClick={onTriggerReview}
              disabled={isTriggering}
              data-testid={selectors.review.rerunAction}
            >
              <RefreshCw className={cn("mr-1 inline h-3 w-3", isTriggering && "animate-spin")} />
              Re-run
            </button>
          )}

          {/* Primary action */}
          {action.kind === "run-review" && (
            <Button
              size="sm"
              variant="outline"
              className="h-7 px-2 text-xs"
              onClick={onTriggerReview}
              data-testid={selectors.review.primaryAction}
            >
              <Play className="mr-1 h-3 w-3" />
              Run Review
            </Button>
          )}
          {action.kind === "triggering" && (
            <Button
              size="sm"
              variant="outline"
              className="h-7 px-2 text-xs"
              disabled
              data-testid={selectors.review.primaryAction}
            >
              <Loader2 className="mr-1 h-3 w-3 animate-spin" />
              Running...
            </Button>
          )}
          {action.kind === "fix-issues" && (
            <Button
              size="sm"
              variant="outline"
              className="h-7 border-red-500/30 px-2 text-xs text-red-300 hover:bg-red-500/10"
              onClick={() => onFollowUp(action.exec)}
              data-testid={selectors.review.primaryAction}
            >
              <Wrench className="mr-1 h-3 w-3" />
              Fix Issues
            </Button>
          )}
          {action.kind === "follow-up" && (
            <Button
              size="sm"
              variant="outline"
              className="h-7 px-2 text-xs"
              onClick={() => onFollowUp(action.exec)}
              data-testid={selectors.review.primaryAction}
            >
              <MessageSquare className="mr-1 h-3 w-3" />
              Follow Up
            </Button>
          )}
        </div>
      </div>

      {execution.failureReason && (
        <div className="mt-2 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-300">
          {execution.failureReason}
        </div>
      )}
    </div>
  );
}
