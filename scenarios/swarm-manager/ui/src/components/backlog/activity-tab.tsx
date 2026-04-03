/**
 * ActivityTab
 *
 * Composition wrapper for the Activity tab on BacklogDetailsPage.
 * Renders the full activity timeline (executions + agent activities)
 * for the backlog item's lifecycle.
 *
 * All data flows in via props — no direct hook calls.
 */

import { ActivityTimeline } from "../detail/ActivityTimeline";
import { selectors } from "../../consts/selectors";
import type { ExecutionRecord } from "../../types";
import type { TimelineEntry } from "../../hooks/useActivityTimeline";
import type { AgentActivityRecord } from "../../stores/agent-activities-store";

export interface ActivityTabProps {
  /** Timeline data (from useActivityTimeline). */
  timeline: {
    entries: TimelineEntry[];
    isLoading: boolean;
    error: Error | null;
  };
  /** Whether an agent run is active. */
  agentRunIsActive: boolean;
  /** Latest agent activity from global store. */
  latestAgentActivity: AgentActivityRecord | null;
  /** Agent manager UI URL for external links. */
  agentManagerUiUrl: string | null;
  // Callbacks
  onStopRun: (runId: string) => void;
  onFollowUp: (exec: ExecutionRecord) => void;
  onViewExecution: (exec: ExecutionRecord) => void;
}

export function ActivityTab({
  timeline,
  agentRunIsActive,
  latestAgentActivity,
  agentManagerUiUrl,
  onStopRun,
  onFollowUp,
  onViewExecution,
}: ActivityTabProps) {
  return (
    <div data-testid={selectors.backlogDetails.activityTab}>
      <ActivityTimeline
        entries={timeline.entries}
        isLoading={timeline.isLoading}
        error={timeline.error}
        onViewExecution={onViewExecution}
        onStopRun={onStopRun}
        onFollowUp={onFollowUp}
        latestAgentActivity={latestAgentActivity ?? undefined}
        agentRunIsActive={agentRunIsActive}
        agentManagerUiUrl={agentManagerUiUrl ?? undefined}
      />
    </div>
  );
}
