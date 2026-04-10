/**
 * LatestExecutionSummary
 *
 * Persistent section at the top of the Output tab showing the active
 * run state or the empty state. Completed/failed execution status and
 * actions are now handled by ReviewStatusHeader in ReviewFlow.
 */

import { Square } from "lucide-react";
import { Button } from "../ui/button";
import { formatRelativeTime } from "../../lib";
import { selectors } from "../../consts/selectors";
import type { ExecutionRecord } from "../../types";
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
}

export function LatestExecutionSummary({
  latestExecution,
  agentRunIsActive,
  latestAgentActivity,
  onStopRun,
}: LatestExecutionSummaryProps) {
  const testId = `${selectors.backlogDetails.outputTab}-latest-exec`;

  // Active run state — agent is currently running
  if (agentRunIsActive && latestAgentActivity) {
    return (
      <div className="rounded-lg border border-cyan-500/30 bg-cyan-500/10 p-3" data-testid={testId}>
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
      </div>
    );
  }

  // Empty state — no executions yet
  if (!latestExecution) {
    return (
      <div className="py-3" data-testid={testId}>
        <p className="text-sm text-slate-400">
          No executions yet. Queue or run the agent to see results here.
        </p>
      </div>
    );
  }

  // Completed/failed — status is shown by ReviewStatusHeader
  return null;
}
