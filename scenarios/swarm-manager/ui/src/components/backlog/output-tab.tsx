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

import { ReviewFlow } from "../review/review-flow";
import { selectors } from "../../consts/selectors";
import type { ExecutionRecord } from "../../types";
import type { ReviewRound } from "../../services/review-service";
import type { AgentActivityRecord } from "../../stores/agent-activities-store";

export interface OutputTabProps {
  /** Full execution history (from useBacklogDetailData). */
  executionHistory: ExecutionRecord[] | undefined;
  /** Whether an agent run is actively executing. */
  agentRunIsBusy: boolean;
  /** Latest agent activity from global store. */
  latestAgentActivity: AgentActivityRecord | null;
  /** Agent manager UI URL for external run links. */
  agentManagerUiUrl: string | null;
  /** Review evidence rounds. */
  reviewRounds: ReviewRound[];
  /** Whether the review agent is currently gathering evidence. */
  isGatheringEvidence: boolean;
  /** Whether the review agent is blocked in manual review/approval. */
  isAwaitingManualReview: boolean;
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
  agentRunIsBusy,
  agentManagerUiUrl,
  reviewRounds,
  isGatheringEvidence,
  isAwaitingManualReview,
  backlogKind,
  backlogName,
  onFollowUp,
  onArchive,
  onVerifyEvidence,
  onRequestMoreEvidence,
}: OutputTabProps) {
  const latestExecution = executionHistory?.[0];

  return (
    <div className="space-y-0" data-testid={selectors.backlogDetails.outputTab}>
      <ReviewFlow
        execution={latestExecution}
        reviewRounds={reviewRounds}
        isGatheringEvidence={isGatheringEvidence}
        isAwaitingManualReview={isAwaitingManualReview}
        isActive={agentRunIsBusy}
        agentManagerUiUrl={agentManagerUiUrl}
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
