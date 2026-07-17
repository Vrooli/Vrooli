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
import {
  isReviewOperation,
  provenanceByAttempt,
  provenanceForRun,
} from "../../lib/agent-ops-utils";
import type { ExecutionRecord } from "../../types";
import type {
  WorkflowExecutionSummary,
  WorkflowProjection,
} from "../../types/agent-operations";
import type { ReviewRound } from "../../services/review-service";
import type { AgentActivityRecord } from "../../stores/agent-activities-store";

export interface OutputTabProps {
  /** Full execution history (from useBacklogDetailData). */
  executionHistory: ExecutionRecord[] | undefined;
  /** Canonical workflow projection (found=false → legacy-only rendering). */
  workflowProjection?: WorkflowProjection;
  /** Canonical execution provenance history, newest first. */
  canonicalExecutionHistory?: WorkflowExecutionSummary[];
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
  workflowProjection,
  canonicalExecutionHistory,
  agentRunIsBusy,
  latestAgentActivity,
  agentManagerUiUrl,
  reviewRounds,
  isGatheringEvidence,
  isAwaitingManualReview,
  backlogKind,
  backlogName,
  onStopRun,
  onFollowUp,
  onArchive,
  onVerifyEvidence,
  onRequestMoreEvidence,
}: OutputTabProps) {
  const latestExecution = executionHistory?.[0];

  // Canonical inspectability: match the live run / review rounds back to
  // their projected operation records (server data, presentation-only index).
  const runProvenance = provenanceForRun(
    workflowProjection,
    latestAgentActivity?.runId ?? latestExecution?.runId,
  );
  const reviewProvenanceByRound = provenanceByAttempt(workflowProjection, isReviewOperation);

  return (
    <div className="space-y-0" data-testid={selectors.backlogDetails.outputTab}>
      <LatestExecutionSummary
        latestExecution={latestExecution}
        agentRunIsBusy={agentRunIsBusy}
        latestAgentActivity={latestAgentActivity}
        agentManagerUiUrl={agentManagerUiUrl}
        onStopRun={onStopRun}
        runProvenance={runProvenance}
        canonicalHistory={workflowProjection?.found ? canonicalExecutionHistory : undefined}
      />

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
        reviewProvenanceByRound={reviewProvenanceByRound}
      />
    </div>
  );
}
