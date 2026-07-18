/**
 * ReviewFlow — Unified composition root for post-execution review display.
 *
 * Used identically by both BacklogDetailsPage (Output tab) and
 * ExecutionDetailsPage (Review tab). Presents a linear narrative:
 * status header → finalization progress → scenario results → evidence →
 * action footer (archive / follow-up).
 */

import { useState } from "react";
import { Archive, MessageSquare, Wrench } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { ReviewStatusHeader } from "./review-status-header";
import { ReviewLaunchSheet } from "./review-launch-sheet";
import { ScenarioResultCards } from "./scenario-result-cards";
import { useReviewActions } from "./use-review-actions";
import { PostRunStatusBadge } from "../execution/post-run-status-badge";
import { EvidencePanel } from "../backlog/evidence-panel";
import { Button } from "../ui/button";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { resolvePostRunExecution } from "../../lib/finalization";
import { defaultQueryOptions, canFollowUpExecution } from "../../lib";
import { settingsService } from "../../services/settings-service";
import { selectors } from "../../consts/selectors";
import type { ExecutionRecord } from "../../types";
import type { Settings } from "../../types/settings";
import type { ReviewRound } from "../../services/review-service";

export interface ReviewFlowProps {
  execution: ExecutionRecord | undefined;
  reviewRounds: ReviewRound[];
  isGatheringEvidence: boolean;
  isAwaitingManualReview: boolean;
  isActive: boolean;
  agentManagerUiUrl: string | null;
  backlogKind: string;
  backlogName: string;
  onFollowUp: (exec: ExecutionRecord) => void;
  onArchive?: () => void;
  onVerifyEvidence: (round: number, evidenceId: string, verified: boolean) => void;
  onRequestMoreEvidence: (round: number, evidenceId?: string) => void;
}

export function ReviewFlow({
  execution,
  reviewRounds,
  isGatheringEvidence,
  isAwaitingManualReview,
  isActive,
  agentManagerUiUrl,
  backlogKind,
  backlogName,
  onFollowUp,
  onArchive,
  onVerifyEvidence,
  onRequestMoreEvidence,
}: ReviewFlowProps) {
  const [archiveConfirmOpen, setArchiveConfirmOpen] = useState(false);
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
  const { data: settings } = useQuery<Settings>({
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

  // Compute review state for footer actions.
  const allEvidence = reviewRounds.flatMap((r) => r.evidence);
  const totalCount = allEvidence.length;
  const reviewedCount = allEvidence.filter((e) => e.verified).length;
  const allReviewed = totalCount > 0 && reviewedCount === totalCount;
  const hasNeedsWork = resolved?.finalization?.aggregateClassification === "needs_work";
  const isTerminal = execution ? canFollowUpExecution(execution.status) : false;
  const showFooter = execution && !isActive && isTerminal && onArchive;

  const handleArchiveClick = () => {
    if (!allReviewed && totalCount > 0) {
      setArchiveConfirmOpen(true);
    } else {
      onArchive?.();
    }
  };

  // Nothing to show when no execution and not active
  if (!execution && !isActive) return null;

  return (
    <div className="space-y-0" data-testid={selectors.review.flow}>
      <ReviewStatusHeader
        execution={execution}
        isActive={isActive}
        agentManagerUiUrl={agentManagerUiUrl}
        isTriggering={isTriggering}
        isTriggeringEvidence={isTriggeringEvidence}
        isCancelling={isCancelling}
        onOpenLaunchSheet={openLaunchSheet}
        onCancelReview={cancelReview}
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
          isAwaitingManualReview={isAwaitingManualReview}
          onVerify={onVerifyEvidence}
          onRequestMore={onRequestMoreEvidence}
        />
      )}

      {/* Action footer — archive and follow-up at the bottom after evidence */}
      {showFooter && (
        <div className="flex items-center gap-2 border-t border-slate-200 px-4 py-3 dark:border-slate-700">
          {hasNeedsWork ? (
            <Button
              size="sm"
              variant="outline"
              className="h-8 border-red-500/30 px-3 text-xs text-red-300 hover:bg-red-500/10"
              onClick={() => onFollowUp(execution)}
            >
              <Wrench className="mr-1.5 h-3.5 w-3.5" />
              Fix Issues
            </Button>
          ) : (
            <Button
              size="sm"
              variant="outline"
              className="h-8 px-3 text-xs"
              onClick={() => onFollowUp(execution)}
            >
              <MessageSquare className="mr-1.5 h-3.5 w-3.5" />
              Follow Up
            </Button>
          )}
          <Button
            size="sm"
            variant="outline"
            className="h-8 px-3 text-xs"
            onClick={handleArchiveClick}
          >
            <Archive className="mr-1.5 h-3.5 w-3.5" />
            Archive
          </Button>
        </div>
      )}

      {/* Archive confirmation when not all evidence is reviewed */}
      <ConfirmDialog
        isOpen={archiveConfirmOpen}
        onClose={() => setArchiveConfirmOpen(false)}
        onConfirm={() => {
          setArchiveConfirmOpen(false);
          onArchive?.();
        }}
        title="Archive without full review?"
        description={`You have ${totalCount - reviewedCount} of ${totalCount} evidence items still unreviewed. Are you sure you want to archive this item?`}
        confirmLabel="Archive anyway"
      />

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
