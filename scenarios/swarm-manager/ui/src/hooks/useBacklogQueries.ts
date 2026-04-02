import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { defaultQueryOptions } from "../lib";
import { parseWorkshopRound, WORKSHOP_FILE_PATHS, findBacklogFileByPath } from "../lib/workshop-files";
import { buildReadinessData } from "../lib/maturity";
import type { ReadinessIndicatorData } from "../lib/maturity";
import { backlogService, executionService } from "../services";
import { reviewService } from "../services/review-service";
import type { ReviewRound } from "../services/review-service";
import type { BacklogKind } from "../types";
import type { MaturityItemSummary, WorkshopRound } from "../types/domain";
import { useBacklogStore } from "../stores";

const AGENT_RUN_REFRESH_MS = 6000;

export interface UseBacklogQueriesOptions {
  backlogKind: BacklogKind | null;
  name: string | undefined;
  agentRunIsActive: boolean;
}

export function useBacklogQueries({
  backlogKind,
  name,
  agentRunIsActive,
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
    refetchInterval: agentRunIsActive ? AGENT_RUN_REFRESH_MS : false,
    ...defaultQueryOptions,
  });

  const { data: executionHistory } = useQuery({
    queryKey: ["executions", backlogKind, name],
    queryFn: () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return executionService.list({ backlogKind: backlogKind as BacklogKind, backlogName: name });
    },
    enabled: !!backlogKind && !!name,
    refetchInterval: 10_000,
  });

  const { data: reviewRounds } = useQuery({
    queryKey: ["review-rounds", backlogKind, name],
    queryFn: () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return reviewService.listRounds(backlogKind, name);
    },
    enabled: !!backlogKind && !!name,
    refetchInterval: agentRunIsActive ? AGENT_RUN_REFRESH_MS : false,
  });

  const isGatheringEvidence = useMemo(
    () => (reviewRounds ?? []).some((r: ReviewRound) => r.status === "gathering"),
    [reviewRounds],
  );

  const workshopDir = useMemo(
    () => findBacklogFileByPath(files ?? [], WORKSHOP_FILE_PATHS.workshopDir.replace(/\/$/, "")),
    [files],
  );
  const workshopRoundPaths = useMemo(() => {
    if (!workshopDir?.children) return [];
    return workshopDir.children
      .filter((f) => f.type === "file" && /^round-\d+\.json$/.test(f.name))
      .sort((a, b) => a.name.localeCompare(b.name, undefined, { numeric: true }))
      .map((f) => f.path);
  }, [workshopDir]);

  const {
    data: workshopRoundContents,
    refetch: refetchWorkshopRounds,
  } = useQuery({
    queryKey: ["backlog", backlogKind, name, "workshop-rounds", workshopRoundPaths],
    queryFn: async () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      const contents = await Promise.all(
        workshopRoundPaths.map((p) => backlogService.getFileContent(backlogKind, name, p)),
      );
      return contents;
    },
    enabled: !!backlogKind && !!name && workshopRoundPaths.length > 0,
    refetchInterval: agentRunIsActive ? AGENT_RUN_REFRESH_MS : false,
    ...defaultQueryOptions,
  });

  const workshopRounds = useMemo(() => {
    if (!workshopRoundContents) return [];
    return workshopRoundContents
      .map((content) => parseWorkshopRound(content))
      .filter((r): r is { round: WorkshopRound; error?: string } => r.round !== null)
      .map((r) => r.round);
  }, [workshopRoundContents]);

  const { data: maturitySummaryData } = useQuery({
    queryKey: ["backlog-maturity-summary"],
    queryFn: () => backlogService.getMaturitySummary(),
    refetchInterval: agentRunIsActive ? AGENT_RUN_REFRESH_MS : false,
    ...defaultQueryOptions,
  });

  const readinessData = useMemo<ReadinessIndicatorData | null>(() => {
    if (!maturitySummaryData || !backlogKind || !name) return null;
    const match = (maturitySummaryData.items ?? []).find(
      (i: MaturityItemSummary) => i.kind === backlogKind && i.name === name,
    );
    return match ? buildReadinessData(match) : null;
  }, [maturitySummaryData, backlogKind, name]);

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
    itemError: itemError as Error | null,
    refetchItem,

    spawnedItems,

    files,
    isLoadingFiles,
    filesError,
    refetchFiles,

    executionHistory,

    reviewRounds,
    isGatheringEvidence,

    workshopDir,
    workshopRoundPaths,
    workshopRounds,
    refetchWorkshopRounds,

    maturitySummaryData,
    readinessData,

    archiveTargets,
  };
}
