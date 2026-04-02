/**
 * useExecutionDetailData — Data hook for ExecutionDetailsPage.
 *
 * Encapsulates all queries (execution, prompt trace, review rounds, timeline)
 * and mutations (cancel, retry, triggerReview) for a single execution.
 * Follows the same boundary pattern as useBacklogDetailData.
 */

import { useState, useMemo, useCallback } from "react";
import { useQuery } from "@tanstack/react-query";
import { executionService, promptService } from "../services";
import { reviewService } from "../services/review-service";
import {
  resolvePostRunExecution,
  defaultQueryOptions,
} from "../lib";
import { isExecutionActive } from "../lib/execution-utils";
import { useActivityTimeline } from "./useActivityTimeline";
import type { ExecutionRecord, PromptTrace } from "../types";
import type { ReviewRound } from "../services/review-service";
import type { TimelineEntry } from "./useActivityTimeline";

export interface UseExecutionDetailDataOptions {
  executionId: string | undefined;
}

export interface UseExecutionDetailDataResult {
  // Data
  execution: ExecutionRecord | undefined;
  trace: PromptTrace | null | undefined;
  isTraceLoading: boolean;
  reviewRounds: ReviewRound[];
  isGatheringEvidence: boolean;
  timeline: { entries: TimelineEntry[]; isLoading: boolean; error: Error | null };

  // Loading/error
  isLoading: boolean;
  error: Error | undefined;

  // Computed
  isActive: boolean;
  isTerminal: boolean;
  postRunBadgeExecution: ExecutionRecord | null;
  targetScenarios: string[];

  // Mutations
  cancel: () => Promise<void>;
  retry: () => Promise<void>;
  triggerReview: () => Promise<void>;
  refetch: () => void;
  actionBusy: boolean;
}

const TERMINAL_STATUSES = new Set(["completed", "failed", "canceled"]);

export function useExecutionDetailData({
  executionId,
}: UseExecutionDetailDataOptions): UseExecutionDetailDataResult {
  const [actionBusy, setActionBusy] = useState(false);

  // --- Primary execution query ---
  const {
    data: execution,
    error: execError,
    isLoading: execLoading,
    refetch: refetchExec,
  } = useQuery({
    queryKey: ["execution", executionId],
    queryFn: () => executionService.get(executionId!),
    enabled: !!executionId,
    ...defaultQueryOptions,
  });

  // --- Prompt trace query ---
  const { data: trace, isLoading: isTraceLoading } = useQuery({
    queryKey: ["execution", executionId, "prompt-trace"],
    queryFn: () => promptService.getExecutionPromptTrace(executionId!).catch(() => null),
    enabled: !!executionId,
    ...defaultQueryOptions,
  });

  // --- Review rounds query (keyed by execution's backlog) ---
  const backlogKind = execution?.backlogKind;
  const backlogName = execution?.backlogName;
  const { data: reviewRounds } = useQuery({
    queryKey: ["review-rounds", backlogKind, backlogName],
    queryFn: () => reviewService.listRounds(backlogKind!, backlogName!),
    enabled: !!backlogKind && !!backlogName,
    ...defaultQueryOptions,
  });

  // --- Activity timeline ---
  const isActive = execution ? isExecutionActive(execution) : false;
  const timeline = useActivityTimeline({
    backlogKind,
    backlogName,
    enabled: !!execution,
    agentRunIsActive: isActive,
  });

  // --- Computed values ---
  const isTerminal = execution ? TERMINAL_STATUSES.has(execution.status) : false;

  const postRunBadgeExecution = useMemo(
    () => (execution ? resolvePostRunExecution(execution) : null),
    [execution],
  );

  const targetScenarios = useMemo(
    () => execution?.finalization?.affectedScenarios ?? [],
    [execution],
  );

  const isGatheringEvidence = useMemo(
    () => (reviewRounds ?? []).some((r) => r.status === "gathering"),
    [reviewRounds],
  );

  // --- Mutations ---
  const doAction = useCallback(
    async (fn: () => Promise<ExecutionRecord>) => {
      setActionBusy(true);
      try {
        await fn();
        void refetchExec();
      } finally {
        setActionBusy(false);
      }
    },
    [refetchExec],
  );

  const cancel = useCallback(
    () => doAction(() => executionService.cancel(executionId!)),
    [doAction, executionId],
  );

  const retry = useCallback(
    () => doAction(() => executionService.retry(executionId!)),
    [doAction, executionId],
  );

  const triggerReviewAction = useCallback(
    () => doAction(() => executionService.triggerReview(executionId!)),
    [doAction, executionId],
  );

  const refetch = useCallback(() => {
    void refetchExec();
  }, [refetchExec]);

  return {
    execution,
    trace: trace ?? null,
    isTraceLoading,
    reviewRounds: reviewRounds ?? [],
    isGatheringEvidence,
    timeline,
    isLoading: execLoading,
    error: execError as Error | undefined,
    isActive,
    isTerminal,
    postRunBadgeExecution,
    targetScenarios,
    cancel,
    retry,
    triggerReview: triggerReviewAction,
    refetch,
    actionBusy,
  };
}
