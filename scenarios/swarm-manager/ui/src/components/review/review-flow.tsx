/**
 * ReviewFlow — Unified composition root for post-execution review display.
 *
 * Used identically by both BacklogDetailsPage (Output tab) and
 * ExecutionDetailsPage (Review tab). Presents a linear narrative:
 * status header → scenario chips → finalization details → evidence.
 */

import { ReviewStatusHeader } from "./review-status-header";
import { ScenarioChips } from "./scenario-chips";
import { useReviewActions } from "./use-review-actions";
import { PostRunStatusBadge } from "../execution/post-run-status-badge";
import { EvidencePanel } from "../backlog/evidence-panel";
import { resolvePostRunExecution } from "../../lib/finalization";
import { selectors } from "../../consts/selectors";
import type { ExecutionRecord } from "../../types";
import type { ReviewRound } from "../../services/review-service";

export interface ReviewFlowProps {
  execution: ExecutionRecord | undefined;
  targetScenarios: string[];
  reviewRounds: ReviewRound[];
  isGatheringEvidence: boolean;
  isActive: boolean;
  backlogKind: string;
  backlogName: string;
  onFollowUp: (exec: ExecutionRecord) => void;
  onSelectScenario: (name: string) => void;
  onVerifyEvidence: (round: number, evidenceId: string, verified: boolean) => void;
  onRequestMoreEvidence: (round: number, evidenceId?: string) => void;
}

export function ReviewFlow({
  execution,
  targetScenarios,
  reviewRounds,
  isGatheringEvidence,
  isActive,
  backlogKind,
  backlogName,
  onFollowUp,
  onSelectScenario,
  onVerifyEvidence,
  onRequestMoreEvidence,
}: ReviewFlowProps) {
  const { triggerReview, isTriggering } = useReviewActions(execution?.executionId);
  const resolved = execution ? resolvePostRunExecution(execution) : null;

  // Nothing to show when no execution and not active
  if (!execution && !isActive) return null;

  return (
    <div className="space-y-0" data-testid={selectors.review.flow}>
      <ReviewStatusHeader
        execution={execution}
        isActive={isActive}
        isTriggering={isTriggering}
        onTriggerReview={triggerReview}
        onFollowUp={onFollowUp}
      />

      {targetScenarios.length > 0 && (
        <div className="py-2">
          <ScenarioChips scenarios={targetScenarios} onSelect={onSelectScenario} />
        </div>
      )}

      {resolved && (
        <div className="py-2">
          <PostRunStatusBadge execution={resolved} />
        </div>
      )}

      {(reviewRounds.length > 0 || isGatheringEvidence) && (
        <EvidencePanel
          rounds={reviewRounds}
          backlogKind={backlogKind}
          backlogName={backlogName}
          isGathering={isGatheringEvidence}
          onVerify={onVerifyEvidence}
          onRequestMore={onRequestMoreEvidence}
        />
      )}
    </div>
  );
}
