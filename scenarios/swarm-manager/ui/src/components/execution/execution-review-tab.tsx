/**
 * ExecutionReviewTab — Post-run review results for a single execution.
 * Delegates to ReviewFlow for the shared review display.
 */

import { ClipboardList } from "lucide-react";
import { ReviewFlow } from "../review/review-flow";
import { DetailSection } from "../detail/DetailSection";
import { selectors } from "../../consts/selectors";
import type { ExecutionRecord } from "../../types";
import type { ReviewRound } from "../../services/review-service";

export interface ExecutionReviewTabProps {
  execution: ExecutionRecord;
  reviewRounds: ReviewRound[];
  isGatheringEvidence: boolean;
  isActive: boolean;
  onFollowUp: (exec: ExecutionRecord) => void;
  onVerifyEvidence: (round: number, evidenceId: string, verified: boolean) => void;
  onRequestMoreEvidence: (round: number, evidenceId?: string) => void;
}

export function ExecutionReviewTab({
  execution,
  reviewRounds,
  isGatheringEvidence,
  isActive,
  onFollowUp,
  onVerifyEvidence,
  onRequestMoreEvidence,
}: ExecutionReviewTabProps) {
  if (isActive) {
    return (
      <DetailSection title="Review" hideDivider>
        <div className="py-6 text-center" data-testid={selectors.executionDetails.reviewEmpty}>
          <p className="text-sm text-slate-400">Review will be available after the execution completes.</p>
        </div>
      </DetailSection>
    );
  }

  const hasContent = execution.finalization || reviewRounds.length > 0 || isGatheringEvidence;

  if (!hasContent) {
    return (
      <DetailSection title="Review" hideDivider>
        <div className="py-6 text-center" data-testid={selectors.executionDetails.reviewEmpty}>
          <ClipboardList className="mx-auto mb-2 h-8 w-8 text-slate-600" />
          <p className="text-sm text-slate-400">No review data available yet.</p>
          <p className="mt-1 text-xs text-slate-500">
            Run post-run checks to generate review results.
          </p>
        </div>
      </DetailSection>
    );
  }

  return (
    <div data-testid={selectors.executionDetails.reviewSection}>
      <ReviewFlow
        execution={execution}
        reviewRounds={reviewRounds}
        isGatheringEvidence={isGatheringEvidence}
        isActive={false}
        backlogKind={execution.backlogKind}
        backlogName={execution.backlogName}
        onFollowUp={onFollowUp}
        onVerifyEvidence={onVerifyEvidence}
        onRequestMoreEvidence={onRequestMoreEvidence}
      />
    </div>
  );
}
