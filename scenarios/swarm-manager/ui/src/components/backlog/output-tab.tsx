/**
 * OutputTab
 *
 * Composition root for the Output tab on BacklogDetailsPage.
 * Focused on review triage: active run indicator → review flow (status,
 * progress, per-scenario results, evidence).
 *
 * Activity history lives in the separate Activity tab.
 * All data flows in via props — no direct hook calls.
 */

import { LatestExecutionSummary } from "./latest-execution-summary";
import { ReviewFlow } from "../review/review-flow";
import { selectors } from "../../consts/selectors";
import type { ExecutionRecord } from "../../types";
import type { ReviewRound } from "../../services/review-service";
import type { AgentActivityRecord } from "../../stores/agent-activities-store";

export interface OutputTabProps {
  /** Full execution history (from useBacklogDetailData). */
  executionHistory: ExecutionRecord[] | undefined;
  /** Whether an agent run is active. */
  agentRunIsActive: boolean;
  /** Latest agent activity from global store. */
  latestAgentActivity: AgentActivityRecord | null;
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
  onArchive?: () => void;
  onVerifyEvidence: (round: number, evidenceId: string, verified: boolean) => void;
  onRequestMoreEvidence: (round: number, evidenceId?: string) => void;
}

export function OutputTab({
  executionHistory,
  agentRunIsActive,
  latestAgentActivity,
  reviewRounds,
  isGatheringEvidence,
  backlogKind,
  backlogName,
  onStopRun,
  onFollowUp,
  onArchive,
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
        reviewRounds={reviewRounds}
        isGatheringEvidence={isGatheringEvidence}
        isActive={agentRunIsActive}
        backlogKind={backlogKind}
        backlogName={backlogName}
        onFollowUp={onFollowUp}
        onArchive={onArchive}
        onVerifyEvidence={onVerifyEvidence}
        onRequestMoreEvidence={onRequestMoreEvidence}
      />
    </div>
  );
}
