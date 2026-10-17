/**
 * Badge shown on backlog list cards when an agent is actively running for that item.
 * Uses the agent activity store to check for active tracked agent work.
 */
import { memo } from "react";
import { useAgentActivitiesStore, selectLatestActivityForBacklog } from "../../stores/agent-activities-store";
import { isAgentActivityBlocking, isAgentActivityExecuting } from "../../lib/agent-activity-utils";
import type { BacklogKind } from "../../types";

interface AgentRunningBadgeProps {
  backlogKind: BacklogKind;
  backlogName: string;
}

export const AgentRunningBadge = memo(function AgentRunningBadge({ backlogKind, backlogName }: AgentRunningBadgeProps) {
  const activity = useAgentActivitiesStore((state) =>
    selectLatestActivityForBacklog(state, backlogKind, backlogName)
  );

  if (!activity || !isAgentActivityBlocking(activity.status)) return null;

  const label = activity.status === "needs_review"
    ? "Needs review"
    : isAgentActivityExecuting(activity.status)
      ? "Agent running"
      : null;

  if (!label) return null;
  const pulse = isAgentActivityExecuting(activity.status);

  return (
    <span className="inline-flex items-center gap-1.5 rounded-full bg-cyan-500/15 px-2 py-0.5 text-xs font-medium text-cyan-300">
      <span className="relative flex h-1.5 w-1.5 shrink-0">
        {pulse && <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-cyan-400 opacity-75" />}
        <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-cyan-500" />
      </span>
      {label}
    </span>
  );
});
