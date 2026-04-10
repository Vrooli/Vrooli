/**
 * ReviewStatusHeader — Single source of truth for post-execution status
 * display and review trigger action.
 *
 * Presents context-aware actions:
 * - "Review" opens the launch sheet for Full Review / Gather Evidence
 * - "Stop Review" when finalization is in progress
 *
 * Follow-up / Fix Issues / Archive actions are in the ReviewFlow footer
 * (below the evidence panel) so users see them after reviewing evidence.
 */

import { Eye, Loader2, Square } from "lucide-react";
import { Button } from "../ui/button";
import { cn, formatRelativeTime, canRunPostRunChecks } from "../../lib";
import { resolvePostRunExecution } from "../../lib/finalization";
import { EXECUTION_STATUS_COLORS, formatExecutionStatus } from "../../types";
import { selectors } from "../../consts/selectors";
import type { ExecutionRecord, ExecutionStatus } from "../../types";

export interface ReviewStatusHeaderProps {
  execution: ExecutionRecord | undefined;
  isActive: boolean;
  isTriggering: boolean;
  isTriggeringEvidence: boolean;
  isCancelling: boolean;
  onOpenLaunchSheet: () => void;
  onCancelReview: () => void;
}

type PrimaryAction =
  | { kind: "none" }
  | { kind: "review"; label: string }
  | { kind: "triggering" }
  | { kind: "stop-review"; exec: ExecutionRecord };

function resolvePrimaryAction(
  execution: ExecutionRecord | undefined,
  isActive: boolean,
  isTriggering: boolean,
  isTriggeringEvidence: boolean,
): PrimaryAction {
  if (!execution || isActive) return { kind: "none" };
  if (isTriggering || isTriggeringEvidence) return { kind: "triggering" };

  const resolved = resolvePostRunExecution(execution);

  // Finalization in progress — offer to stop
  if (resolved?.finalization?.status === "running" || resolved?.finalization?.status === "pending") {
    return { kind: "stop-review", exec: execution };
  }

  // Terminal execution — offer review (opens launch sheet)
  if (canRunPostRunChecks(execution)) {
    return { kind: "review", label: resolved?.finalization ? "Rerun Checks" : "Review" };
  }

  return { kind: "none" };
}

export function ReviewStatusHeader({
  execution,
  isActive,
  isTriggering,
  isTriggeringEvidence,
  isCancelling,
  onOpenLaunchSheet,
  onCancelReview,
}: ReviewStatusHeaderProps) {
  if (!execution) return null;
  if (isActive) return null; // Active run display is handled by LatestExecutionSummary

  const statusColor = EXECUTION_STATUS_COLORS[execution.status as ExecutionStatus] ?? "bg-slate-500";
  const action = resolvePrimaryAction(execution, isActive, isTriggering, isTriggeringEvidence);

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
          {/* Primary action */}
	          {action.kind === "review" && (
            <Button
              size="sm"
              variant="outline"
              className="h-7 px-2 text-xs"
              onClick={onOpenLaunchSheet}
              data-testid={selectors.review.primaryAction}
            >
              <Eye className="mr-1 h-3 w-3" />
              {action.label}
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
          {action.kind === "stop-review" && (
            <Button
              size="sm"
              variant="outline"
              className="h-7 border-amber-500/30 px-2 text-xs text-amber-300 hover:bg-amber-500/10"
              onClick={onCancelReview}
              disabled={isCancelling}
              data-testid={selectors.review.stopAction}
            >
              <Square className="mr-1 h-3 w-3" />
              {isCancelling ? "Stopping..." : "Stop Review"}
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
