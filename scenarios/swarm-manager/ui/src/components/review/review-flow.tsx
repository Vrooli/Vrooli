/**
 * ReviewFlow — Unified composition root for post-execution review display.
 *
 * Used identically by both BacklogDetailsPage (Output tab) and
 * ExecutionDetailsPage (Review tab). Presents a linear narrative:
 * status header → finalization progress → scenario results → evidence.
 */

import { useQuery } from "@tanstack/react-query";
import { ReviewStatusHeader } from "./review-status-header";
import { ReviewLaunchSheet } from "./review-launch-sheet";
import { ScenarioResultCards } from "./scenario-result-cards";
import { useReviewActions } from "./use-review-actions";
import { PostRunStatusBadge } from "../execution/post-run-status-badge";
import { EvidencePanel } from "../backlog/evidence-panel";
import { resolvePostRunExecution } from "../../lib/finalization";
import { defaultQueryOptions } from "../../lib";
import { settingsService } from "../../services/settings-service";
import { selectors } from "../../consts/selectors";
import type { ExecutionRecord } from "../../types";
import type { Settings } from "../../types/settings";
import type { ReviewRound } from "../../services/review-service";

export interface ReviewFlowProps {
  execution: ExecutionRecord | undefined;
  reviewRounds: ReviewRound[];
  isGatheringEvidence: boolean;
  isActive: boolean;
  backlogKind: string;
  backlogName: string;
  onFollowUp: (exec: ExecutionRecord) => void;
  onVerifyEvidence: (round: number, evidenceId: string, verified: boolean) => void;
  onRequestMoreEvidence: (round: number, evidenceId?: string) => void;
}

export function ReviewFlow({
  execution,
  reviewRounds,
  isGatheringEvidence,
  isActive,
  backlogKind,
  backlogName,
  onFollowUp,
  onVerifyEvidence,
  onRequestMoreEvidence,
}: ReviewFlowProps) {
  const {
    triggerReview,
    triggerEvidenceOnly,
    cancelReview,
    isTriggering,
    isTriggeringEvidence,
    isCancelling,
    triggerError,
    isLaunchSheetOpen,
    openLaunchSheet,
    closeLaunchSheet,
  } = useReviewActions(execution?.executionId, backlogKind, backlogName);
  const { data: settings } = useQuery<Settings, Error>({
    queryKey: ["settings"],
    queryFn: () => settingsService.get(),
    ...defaultQueryOptions,
  });
  const reviewAgentEnabled = settings?.reviewAgentEnabled ?? true;
  const resolved = execution ? resolvePostRunExecution(execution) : null;
  const hasExistingFinalization = Boolean(resolved?.finalization);
  const finalizationTerminal = resolved?.finalization?.status === "completed"
    || resolved?.finalization?.status === "failed"
    || resolved?.finalization?.status === "skipped";

  // Nothing to show when no execution and not active
  if (!execution && !isActive) return null;

  return (
    <div className="space-y-0" data-testid={selectors.review.flow}>
      <ReviewStatusHeader
        execution={execution}
        isActive={isActive}
        isTriggering={isTriggering}
        isTriggeringEvidence={isTriggeringEvidence}
        isCancelling={isCancelling}
        onOpenLaunchSheet={openLaunchSheet}
        onCancelReview={cancelReview}
        onFollowUp={onFollowUp}
      />

      {/* Finalization progress stepper — shown during active finalization */}
      {resolved && (
        <div className="py-2">
          <PostRunStatusBadge execution={resolved} />
        </div>
      )}

      {/* Per-scenario results — shown after finalization completes */}
      {execution && (
        <div className="py-2">
          <ScenarioResultCards execution={execution} />
        </div>
      )}

      {/* Evidence panel — always shown when finalization is terminal so empty state is visible */}
      {(reviewRounds.length > 0 || isGatheringEvidence || finalizationTerminal) && (
        <EvidencePanel
          rounds={reviewRounds}
          backlogKind={backlogKind}
          backlogName={backlogName}
          isGathering={isGatheringEvidence}
          onVerify={onVerifyEvidence}
          onRequestMore={onRequestMoreEvidence}
        />
      )}

      {/* Review launch sheet */}
      <ReviewLaunchSheet
        isOpen={isLaunchSheetOpen}
        onClose={closeLaunchSheet}
        onFullReview={triggerReview}
        onGatherEvidence={triggerEvidenceOnly}
        isTriggering={isTriggering}
        isTriggeringEvidence={isTriggeringEvidence}
        hasExistingFinalization={hasExistingFinalization}
        reviewAgentEnabled={reviewAgentEnabled}
        triggerError={triggerError}
      />
    </div>
  );
}
