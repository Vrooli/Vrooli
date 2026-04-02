/**
 * OutputTab
 *
 * Composition root for the Output tab on BacklogDetailsPage.
 * Consolidates execution status, scenario review results, and the
 * activity timeline into a single unified view.
 *
 * All data flows in via props — no direct hook calls, keeping this
 * component testable and the data flow explicit.
 */

import { LatestExecutionSummary } from "./latest-execution-summary";
import { ScenarioReviewResults } from "./scenario-review-results";
import { ActivityTimeline } from "../detail/ActivityTimeline";
import { selectors } from "../../consts/selectors";
import type { ExecutionRecord } from "../../types";
import type { TimelineEntry } from "../../hooks/useActivityTimeline";
import type { AgentActivityRecord } from "../../stores/agent-activities-store";

export interface OutputTabProps {
  /** Full execution history (from useBacklogDetailData). */
  executionHistory: ExecutionRecord[] | undefined;
  /** Timeline data (from useActivityTimeline). */
  timeline: {
    entries: TimelineEntry[];
    isLoading: boolean;
    error: Error | null;
  };
  /** Target scenario names. */
  targetScenarios: string[];
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
  onSelectScenario: (name: string) => void;
}

export function OutputTab({
  executionHistory,
  timeline,
  targetScenarios,
  agentRunIsActive,
  latestAgentActivity,
  agentManagerUiUrl,
  onStopRun,
  onFollowUp,
  onViewExecution,
  onSelectScenario,
}: OutputTabProps) {
  const latestExecution = executionHistory?.[0];

  return (
    <div className="space-y-0" data-testid={selectors.backlogDetails.outputTab}>
      <LatestExecutionSummary
        latestExecution={latestExecution}
        agentRunIsActive={agentRunIsActive}
        latestAgentActivity={latestAgentActivity}
        onStopRun={onStopRun}
        onFollowUp={onFollowUp}
      />

      {targetScenarios.length > 0 && (
        <ScenarioReviewResults
          latestExecution={latestExecution}
          targetScenarios={targetScenarios}
          onSelectScenario={onSelectScenario}
        />
      )}

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
