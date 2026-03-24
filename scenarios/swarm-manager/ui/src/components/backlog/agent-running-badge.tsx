/**
 * Badge shown on backlog list cards when an agent is actively running for that item.
 * Uses the agent runs store to check for active runs (pending/starting/running/needs_review).
 */
import { useAgentRunsStore, selectLatestRunForBacklog } from "../../stores/agent-runs-store";
import type { BacklogKind } from "../../types";

const ACTIVE_STATUSES = new Set(["pending", "starting", "running", "needs_review"]);

interface AgentRunningBadgeProps {
  backlogKind: BacklogKind;
  backlogName: string;
}

export function AgentRunningBadge({ backlogKind, backlogName }: AgentRunningBadgeProps) {
  const run = useAgentRunsStore((state) => selectLatestRunForBacklog(state, backlogKind, backlogName));

  if (!run || !ACTIVE_STATUSES.has(run.status)) return null;

  const label = run.status === "needs_review" ? "Needs review" : "Agent running";

  return (
    <span className="inline-flex items-center gap-1.5 rounded-full bg-cyan-500/15 px-2 py-0.5 text-xs font-medium text-cyan-300">
      <span className="relative flex h-1.5 w-1.5 shrink-0">
        <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-cyan-400 opacity-75" />
        <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-cyan-500" />
      </span>
      {label}
    </span>
  );
}
