/**
 * OutputTab
 *
 * Composition root for the Output tab on BacklogDetailsPage.
 * Presents a linear narrative: active run indicator → review flow → timeline.
 *
 * All data flows in via props — no direct hook calls, keeping this
 * component testable and the data flow explicit.
 */

import { LatestExecutionSummary } from "./latest-execution-summary";
import { ReviewFlow } from "../review/review-flow";
import { ActivityTimeline } from "../detail/ActivityTimeline";
import { selectors } from "../../consts/selectors";
import type { ExecutionRecord } from "../../types";
import type { ReviewRound } from "../../services/review-service";
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
  /** Review evidence rounds. */
  reviewRounds: ReviewRound[];
  /** Whether the review agent is currently gathering evidence. */
  isGatheringEvidence: boolean;
  /** Backlog item kind (for evidence API calls). */
  backlogKind: string;
  /** Backlog item name (for evidence API calls). */
  backlogName: string;
  // Callbacks
  onStopRun: (runId: string) => void;
  onFollowUp: (exec: ExecutionRecord) => void;
  onViewExecution: (exec: ExecutionRecord) => void;
  onSelectScenario: (name: string) => void;
  onVerifyEvidence: (round: number, evidenceId: string, verified: boolean) => void;
  onRequestMoreEvidence: (round: number, evidenceId?: string) => void;
}

export function OutputTab({
  executionHistory,
  timeline,
  targetScenarios,
  agentRunIsActive,
  latestAgentActivity,
  agentManagerUiUrl,
  reviewRounds,
  isGatheringEvidence,
  backlogKind,
  backlogName,
  onStopRun,
  onFollowUp,
  onViewExecution,
  onSelectScenario,
  onVerifyEvidence,
  onRequestMoreEvidence,
}: OutputTabProps) {
  const latestExecution = executionHistory?.[0];

  return (
    <div className="space-y-0" data-testid={selectors.backlogDetails.outputTab}>
      <LatestExecutionSummary
        latestExecution={latestExecution}
        agentRunIsActive={agentRunIsActive}
        latestAgentActivity={latestAgentActivity}
        onStopRun={onStopRun}
      />

      <ReviewFlow
        execution={latestExecution}
        targetScenarios={targetScenarios}
        reviewRounds={reviewRounds}
        isGatheringEvidence={isGatheringEvidence}
        isActive={agentRunIsActive}
        backlogKind={backlogKind}
        backlogName={backlogName}
        onFollowUp={onFollowUp}
        onSelectScenario={onSelectScenario}
        onVerifyEvidence={onVerifyEvidence}
        onRequestMoreEvidence={onRequestMoreEvidence}
      />

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
