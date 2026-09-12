import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { defaultQueryOptions } from "../lib";
import { backlogService, executionService } from "../services";
import { reviewService } from "../services/review-service";
import type { ReviewRound } from "../services/review-service";
import type { BacklogKind } from "../types";
import { useBacklogStore } from "../stores";

const AGENT_RUN_REFRESH_MS = 6000;

export interface UseBacklogQueriesOptions {
  backlogKind: BacklogKind | null;
  name: string | undefined;
  agentRunIsBlocking: boolean;
}

export function useBacklogQueries({
  backlogKind,
  name,
  agentRunIsBlocking,
}: UseBacklogQueriesOptions) {
  const allBacklogItems = useBacklogStore((state) => state.items);

  const cachedItem = useMemo(
    () => allBacklogItems.find((i) => i.kind === backlogKind && i.name === name),
    [allBacklogItems, backlogKind, name],
  );

  const {
    data: item,
    isLoading: isLoadingItem,
    error: itemError,
    refetch: refetchItem,
  } = useQuery({
    queryKey: ["backlog", backlogKind, name],
    queryFn: () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.get(backlogKind, name);
    },
    enabled: !!backlogKind && !!name,
    placeholderData: cachedItem,
    ...defaultQueryOptions,
  });

  const spawnRef = item ? `${item.kind}/${item.name}` : "";
  const { data: spawnedItems } = useQuery({
    queryKey: ["backlog", "spawned-from", spawnRef],
    queryFn: () => backlogService.listBySpawnedFrom(spawnRef),
    enabled: !!spawnRef,
  });

  const {
    data: files,
    isLoading: isLoadingFiles,
    error: filesQueryError,
    refetch: refetchFiles,
  } = useQuery({
    queryKey: ["backlog", backlogKind, name, "files"],
    queryFn: () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.getFiles(backlogKind, name);
    },
    enabled: !!backlogKind && !!name,
    refetchInterval: agentRunIsBlocking ? AGENT_RUN_REFRESH_MS : false,
    ...defaultQueryOptions,
  });

  const { data: executionHistory } = useQuery({
    queryKey: ["executions", backlogKind, name],
    queryFn: () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return executionService.list({ backlogKind: backlogKind, backlogName: name });
    },
    enabled: !!backlogKind && !!name,
    refetchInterval: (query) => {
      const records = query.state.data;
      const latest = records?.[0];
      // Poll faster during active finalization for responsive progress updates
      return latest?.status === "validating" ? 3_000 : 10_000;
    },
  });

  const { data: nextAction } = useQuery({
    queryKey: ["backlog", backlogKind, name, "next-action"],
    queryFn: () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.getNextAction(backlogKind, name);
    },
    enabled: !!backlogKind && !!name,
    ...defaultQueryOptions,
  });

  const isValidating = executionHistory?.[0]?.status === "validating";


  const { data: reviewRounds } = useQuery({
    queryKey: ["review-rounds", backlogKind, name],
    queryFn: () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return reviewService.listRounds(backlogKind, name);
    },
    enabled: !!backlogKind && !!name,
    refetchInterval: (agentRunIsBlocking || isValidating) ? AGENT_RUN_REFRESH_MS : false,
  });

  const isGatheringEvidence = useMemo(
    () =>
      (reviewRounds ?? []).some(
        (r: ReviewRound) => r.status === "gathering" && r.current_run_status !== "needs_review",
      ),
    [reviewRounds],
  );

  const isAwaitingManualReview = useMemo(
    () =>
      (reviewRounds ?? []).some(
        (r: ReviewRound) => r.status === "gathering" && r.current_run_status === "needs_review",
      ),
    [reviewRounds],
  );

  const { data: archiveTargets } = useQuery({
    queryKey: ["backlog", backlogKind, name, "archive-targets"],
    queryFn: () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.getArchiveTargets(backlogKind, name);
    },
    enabled: !!backlogKind && !!name,
    ...defaultQueryOptions,
  });

  const filesError = filesQueryError instanceof Error ? filesQueryError : null;

  return {
    item,
    isLoadingItem,
    itemError: itemError,
    refetchItem,

    spawnedItems,

    files,
    isLoadingFiles,
    filesError,
    refetchFiles,

    executionHistory,
    nextAction,

    reviewRounds,
    isGatheringEvidence,
    isAwaitingManualReview,

    archiveTargets,
  };
}
