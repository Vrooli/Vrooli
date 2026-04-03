/**
 * useReviewActions — Manages review trigger, evidence-only, cancel, and
 * launch sheet state.
 *
 * Provides loading/error state and optimistic React Query cache updates
 * for immediate UI feedback after triggering actions.
 */

import { useCallback, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { executionService } from "../../services";
import { reviewService } from "../../services/review-service";
import type { ExecutionRecord } from "../../types";

export function useReviewActions(
  executionId: string | undefined,
  backlogKind?: string,
  backlogName?: string,
) {
  const queryClient = useQueryClient();
  const [isTriggering, setIsTriggering] = useState(false);
  const [isTriggeringEvidence, setIsTriggeringEvidence] = useState(false);
  const [isCancelling, setIsCancelling] = useState(false);
  const [triggerError, setTriggerError] = useState<string | null>(null);
  const [isLaunchSheetOpen, setIsLaunchSheetOpen] = useState(false);

  const updateCache = useCallback(
    (updated: ExecutionRecord) => {
      if (backlogKind && backlogName) {
        queryClient.setQueryData<ExecutionRecord[]>(
          ["executions", backlogKind, backlogName],
          (old) => old?.map((e) => (e.executionId === updated.executionId ? updated : e)),
        );
      }
    },
    [backlogKind, backlogName, queryClient],
  );

  const openLaunchSheet = useCallback(() => setIsLaunchSheetOpen(true), []);
  const closeLaunchSheet = useCallback(() => setIsLaunchSheetOpen(false), []);

  const triggerReview = useCallback(async () => {
    if (!executionId) return;
    setIsTriggering(true);
    setTriggerError(null);
    try {
      const updated = await executionService.triggerReview(executionId);
      updateCache(updated);
      closeLaunchSheet();
    } catch (err) {
      setTriggerError(err instanceof Error ? err.message : "Failed to trigger review");
    } finally {
      setIsTriggering(false);
    }
  }, [executionId, updateCache, closeLaunchSheet]);

  const triggerEvidenceOnly = useCallback(async () => {
    if (!executionId || !backlogKind || !backlogName) return;
    setIsTriggeringEvidence(true);
    setTriggerError(null);
    try {
      await reviewService.triggerReviewAgent(executionId, backlogKind, backlogName);
      // Refresh review rounds so the new "gathering" round appears immediately.
      await queryClient.invalidateQueries({ queryKey: ["review-rounds", backlogKind, backlogName] });
      closeLaunchSheet();
    } catch (err) {
      setTriggerError(err instanceof Error ? err.message : "Failed to trigger evidence gathering");
    } finally {
      setIsTriggeringEvidence(false);
    }
  }, [executionId, backlogKind, backlogName, queryClient, closeLaunchSheet]);

  const cancelReview = useCallback(async () => {
    if (!executionId) return;
    setIsCancelling(true);
    setTriggerError(null);
    try {
      const updated = await executionService.cancel(executionId);
      updateCache(updated);
    } catch (err) {
      setTriggerError(err instanceof Error ? err.message : "Failed to cancel review");
    } finally {
      setIsCancelling(false);
    }
  }, [executionId, updateCache]);

  return {
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
  };
}
