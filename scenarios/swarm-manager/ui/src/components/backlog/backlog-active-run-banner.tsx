/**
 * BacklogActiveRunBanner
 *
 * Shows a status banner when an agent run is actively processing a backlog item.
 * Displays the run status, purpose, timing, and stop/view controls.
 *
 * Extracted from BacklogDetailsPage to reduce file size and improve modularity.
 */

import { ArrowUpRight, Square } from "lucide-react";
import { Button } from "../ui/button";
import { formatRelativeTime } from "../../lib";
import { selectors } from "../../consts/selectors";
import type { AgentActivityRecord } from "../../stores/agent-activities-store";

export interface BacklogActiveRunBannerProps {
  /** Whether the agent run is currently active. */
  agentRunIsActive: boolean;
  /** The latest agent activity record. */
  latestAgentActivity: AgentActivityRecord | null;
  /** Callback to stop the current run. */
  onStopRun: (runId: string) => void;
  /** Callback to open the execution timeline / close detail. */
  onOpenTimeline: () => void;
}

export function BacklogActiveRunBanner({
  agentRunIsActive,
  latestAgentActivity,
  onStopRun,
  onOpenTimeline,
}: BacklogActiveRunBannerProps) {
  if (!agentRunIsActive || !latestAgentActivity) return null;

  return (
    <div
      className="flex items-center gap-2 rounded-lg border border-cyan-500/30 bg-cyan-500/10 px-3 py-1.5"
      data-testid={selectors.backlogDetails.activeRunBanner}
    >
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
        >
          <Square className="mr-1 h-3 w-3" />
          {latestAgentActivity.isStopping ? "Stopping..." : "Stop"}
        </Button>
        <Button
          variant="outline"
          size="sm"
          className="h-7 w-7 p-0"
          onClick={onOpenTimeline}
          aria-label="View execution"
        >
          <ArrowUpRight className="h-3.5 w-3.5" />
        </Button>
      </div>
    </div>
  );
}
