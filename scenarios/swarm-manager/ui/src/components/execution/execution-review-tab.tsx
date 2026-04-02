/**
 * ExecutionReviewTab — Post-run review results, scenario reviews, and
 * evidence for a single execution. Composes existing review components.
 */

import { ClipboardList } from "lucide-react";
import { PostRunStatusBadge } from "./post-run-status-badge";
import { ScenarioReviewResults } from "../backlog/scenario-review-results";
import { EvidencePanel } from "../backlog/evidence-panel";
import { DetailSection } from "../detail/DetailSection";
import { selectors } from "../../consts/selectors";
import type { ExecutionRecord } from "../../types";
import type { ReviewRound } from "../../services/review-service";

export interface ExecutionReviewTabProps {
  execution: ExecutionRecord;
  reviewRounds: ReviewRound[];
  isGatheringEvidence: boolean;
  targetScenarios: string[];
  postRunBadgeExecution: ExecutionRecord | null;
  isActive: boolean;
  onSelectScenario: (name: string) => void;
  onVerifyEvidence: (round: number, evidenceId: string, verified: boolean) => void;
  onRequestMoreEvidence: (round: number, evidenceId?: string) => void;
  onRunPostRunChecks: () => void;
}

export function ExecutionReviewTab({
  execution,
  reviewRounds,
  isGatheringEvidence,
  targetScenarios,
  postRunBadgeExecution,
  isActive,
  onSelectScenario,
  onVerifyEvidence,
  onRequestMoreEvidence,
  onRunPostRunChecks,
}: ExecutionReviewTabProps) {
  const hasReviewContent = postRunBadgeExecution || targetScenarios.length > 0 || reviewRounds.length > 0 || isGatheringEvidence;

  if (isActive) {
    return (
      <DetailSection title="Review" hideDivider>
        <div className="py-6 text-center" data-testid={selectors.executionDetails.reviewEmpty}>
          <p className="text-sm text-slate-400">Review will be available after the execution completes.</p>
        </div>
      </DetailSection>
    );
  }

  if (!hasReviewContent) {
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
    <div className="space-y-0" data-testid={selectors.executionDetails.reviewSection}>
      {/* Post-run status badge */}
      {postRunBadgeExecution && (
        <DetailSection title="Post-Run Checks" hideDivider>
          <PostRunStatusBadge
            execution={postRunBadgeExecution}
            onRunChecks={onRunPostRunChecks}
          />
        </DetailSection>
      )}

      {/* Scenario reviews */}
      {targetScenarios.length > 0 && (
        <ScenarioReviewResults
          latestExecution={execution}
          targetScenarios={targetScenarios}
          onSelectScenario={onSelectScenario}
        />
      )}

      {/* Evidence panel */}
      {(reviewRounds.length > 0 || isGatheringEvidence) && (
        <EvidencePanel
          rounds={reviewRounds}
          backlogKind={execution.backlogKind}
          backlogName={execution.backlogName}
          isGathering={isGatheringEvidence}
          onVerify={onVerifyEvidence}
          onRequestMore={onRequestMoreEvidence}
        />
      )}
    </div>
  );
}
