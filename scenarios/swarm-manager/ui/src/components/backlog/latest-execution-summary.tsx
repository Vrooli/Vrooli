/**
 * LatestExecutionSummary
 *
 * Persistent card at the top of the Output tab showing the most recent
 * execution's status. This component always renders — empty state, active
 * run, or completed/failed — providing a persistent execution indicator.
 */

import { Square } from "lucide-react";
import { Card } from "../ui/card";
import { Button } from "../ui/button";
import { formatRelativeTime, canFollowUpExecution } from "../../lib";
import { EXECUTION_STATUS_COLORS, formatExecutionStatus } from "../../types/constants";
import { selectors } from "../../consts/selectors";
import type { ExecutionRecord, ExecutionStatus } from "../../types";
import type { AgentActivityRecord } from "../../stores/agent-activities-store";

export interface LatestExecutionSummaryProps {
  /** Most recent execution, or undefined for empty state. */
  latestExecution: ExecutionRecord | undefined;
  /** Whether an agent run is currently active. */
  agentRunIsActive: boolean;
  /** The latest agent activity record (for live status). */
  latestAgentActivity: AgentActivityRecord | null;
  /** Stop a running agent. */
  onStopRun: (runId: string) => void;
  /** Open follow-up dialog for this execution. */
  onFollowUp: (exec: ExecutionRecord) => void;
}

export function LatestExecutionSummary({
  latestExecution,
  agentRunIsActive,
  latestAgentActivity,
  onStopRun,
  onFollowUp,
}: LatestExecutionSummaryProps) {
  const testId = `${selectors.backlogDetails.outputTab}-latest-exec`;

  // Active run state — agent is currently running
  if (agentRunIsActive && latestAgentActivity) {
    return (
      <Card padding="sm" className="rounded-lg border-cyan-500/30 bg-cyan-500/10" data-testid={testId}>
        <div className="flex items-center gap-2">
          <span className="relative flex h-2 w-2 shrink-0">
            <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-cyan-400 opacity-75" />
            <span className="relative inline-flex h-2 w-2 rounded-full bg-cyan-500" />
          </span>
          <span className="text-xs font-medium capitalize text-cyan-200">
            {latestAgentActivity.status.replace("_", " ")}
          </span>
          {latestAgentActivity.purpose && (
            <span className="rounded bg-cyan-500/20 px-1.5 py-0.5 text-[11px] font-medium text-cyan-300">
              {latestAgentActivity.purpose.replace("_", " ")}
            </span>
          )}
          <span className="text-xs text-slate-400">
            {formatRelativeTime(latestAgentActivity.requestedAt)}
          </span>
          <div className="ml-auto flex items-center gap-1.5">
            <Button
              variant="outline"
              size="sm"
              className="h-7 px-2 text-xs"
              onClick={() => onStopRun(latestAgentActivity.runId ?? "")}
              disabled={latestAgentActivity.isStopping}
              data-testid={`${testId}-stop`}
            >
              <Square className="mr-1 h-3 w-3" />
              {latestAgentActivity.isStopping ? "Stopping..." : "Stop"}
            </Button>
          </div>
        </div>
      </Card>
    );
  }

  // Empty state — no executions yet
  if (!latestExecution) {
    return (
      <Card padding="sm" className="rounded-lg border-slate-700/60 bg-slate-900/45" data-testid={testId}>
        <p className="text-sm text-slate-400">
          No executions yet. Queue or run the agent to see results here.
        </p>
      </Card>
    );
  }

  // Completed/failed execution
  const statusColor = EXECUTION_STATUS_COLORS[latestExecution.status as ExecutionStatus] ?? "bg-slate-500";
  const canFollowUp = canFollowUpExecution(latestExecution.status as ExecutionStatus);

  const duration =
    latestExecution.createdAt && latestExecution.finalization?.scenarios?.[0]?.restart?.finishedAt
      ? undefined // Duration is complex to derive; timeline shows it
      : undefined;

  return (
    <Card padding="sm" className="rounded-lg border-slate-700/60 bg-slate-900/45" data-testid={testId}>
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <span className={`inline-block h-2 w-2 shrink-0 rounded-full ${statusColor}`} />
          <span className="text-xs font-medium capitalize text-slate-200">
            {formatExecutionStatus(latestExecution.status as ExecutionStatus)}
          </span>
          {latestExecution.operation && (
            <span className="rounded bg-slate-700/60 px-1.5 py-0.5 text-[11px] font-medium text-slate-300">
              {latestExecution.operation.replace("_", " ")}
            </span>
          )}
          <span className="text-xs text-slate-400">
            {formatRelativeTime(latestExecution.createdAt)}
          </span>
          {canFollowUp && (
            <Button
              variant="outline"
              size="sm"
              className="ml-auto h-7 px-2 text-xs"
              onClick={() => onFollowUp(latestExecution)}
              data-testid={`${testId}-follow-up`}
            >
              Follow Up
            </Button>
          )}
        </div>
        {latestExecution.failureReason && (
          <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-300">
            {latestExecution.failureReason}
          </div>
        )}
      </div>
    </Card>
  );
}
