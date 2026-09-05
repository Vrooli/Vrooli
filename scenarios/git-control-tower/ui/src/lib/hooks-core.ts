// ============================================================================
// Core Git/Repo Hooks — health, status, diff, stage, commit, branches, files
// ============================================================================

import { useCallback, useEffect, useRef, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "./hooks-query-keys";
import {
  fetchHealth,
  fetchRepoStatus,
  fetchRepoHistory,
  fetchDiff,
  fetchSyncStatus,
  fetchApprovedChanges,
  fetchApprovedChangesPreview,
  fetchProvenance,
  stageFiles,
  unstageFiles,
  createCommit,
  fetchPrecommitConfig,
  savePrecommitConfig,
  runPrecommit,
  runPrecommitStream,
  discardFiles,
  ignoreFile,
  pushToRemote,
  pullFromRemote,
  fetchBranches,
  createBranch,
  switchBranch,
  publishBranch,
  fetchFiles,
  fetchRelatedFiles,
  fetchDirectoryContents,
  deletePath,
  saveFileContent,
  searchContent,
} from "./api";
import type {
  DiffStats,
  RepoStatus,
  RepoFilesStatus,
  RepoStatusSummary,
  RepoHistoryResponse,
  ViewMode,
  StageRequest,
  UnstageRequest,
  CommitRequest,
  DiscardRequest,
  IgnoreRequest,
  PushRequest,
  PullRequest,
  CreateBranchRequest,
  SwitchBranchRequest,
  PublishBranchRequest,
  ApprovedChangesPreviewRequest,
  FileTreeResponse,
  RelatedFilesResponse,
  DirListResponse,
  DeletePathRequest,
  SaveFileContentRequest,
  SaveFileContentResponse,
  ContentSearchRequest,
  ContentSearchResponse,
  PrecommitConfig,
  PrecommitRunRequest,
  PrecommitStreamEvent,
} from "./api";

export function useHealth() {
  return useQuery({
    queryKey: queryKeys.health,
    queryFn: fetchHealth,
    refetchInterval: 30000
  });
}

export function useRepoStatus(repoId?: string | null) {
  return useQuery({
    queryKey: queryKeys.repoStatus(repoId),
    queryFn: () => fetchRepoStatus(repoId ?? undefined),
    refetchInterval: 15_000,
    staleTime: 5_000,
  });
}

export function useRepoHistory(limit = 30, includeFiles = false, repoId?: string | null, grep?: string, includeChecks = false) {
  return useQuery<RepoHistoryResponse, Error>({
    queryKey: queryKeys.repoHistory(limit, includeFiles, repoId, grep, includeChecks),
    queryFn: () => fetchRepoHistory(limit, includeFiles, repoId ?? undefined, grep, includeChecks),
    refetchInterval: grep ? false : 30000,
  });
}

export function useDiff(
  path?: string,
  staged = false,
  untracked = false,
  commit?: string,
  mode: ViewMode = "diff",
  any = false,
  repoId?: string | null
) {
  return useQuery({
    queryKey: queryKeys.diff(path, staged, untracked, commit, mode, any, repoId),
    queryFn: () => fetchDiff(path, staged, untracked, commit, mode, any, repoId ?? undefined),
    enabled: Boolean(path)
  });
}

export function useDiffStats(
  path?: string,
  staged = false,
  untracked = false,
  enabled = false,
  repoId?: string | null,
) {
  const query = useQuery({
    queryKey: queryKeys.diff(path, staged, untracked, undefined, "diff", false, repoId),
    queryFn: () => fetchDiff(path, staged, untracked, undefined, "diff", false, repoId ?? undefined),
    enabled: enabled && Boolean(path) && !untracked,
    staleTime: 30_000,
  });
  return {
    stats: query.data?.stats as DiffStats | undefined,
    isLoading: query.isLoading && enabled && Boolean(path) && !untracked,
  };
}

// ----------------------------------------------------------------------------
// Optimistic status transitions
//
// Pure reducers over the cached RepoStatus so stage/unstage/discard move files
// between sections instantly. They are best-effort: the mutation's onSettled
// refetch reconciles any drift (e.g. mixed hunk state), so these only need to
// match the common whole-file case. Kept pure and exported for unit tests.
// ----------------------------------------------------------------------------

// The server omits empty bucket arrays from the status payload, so the cached
// RepoFilesStatus can have undefined staged/unstaged/untracked/conflicts at
// runtime despite the non-optional TS types. Always read buckets through this.
function bucket(arr: string[] | undefined): string[] {
  return arr ?? [];
}

function recomputeSummary(files: RepoFilesStatus, prev: RepoStatusSummary): RepoStatusSummary {
  return {
    ...prev,
    staged: bucket(files.staged).length,
    unstaged: bucket(files.unstaged).length,
    untracked: bucket(files.untracked).length,
    conflicts: bucket(files.conflicts).length,
  };
}

function withFiles(status: RepoStatus, files: RepoFilesStatus): RepoStatus {
  return { ...status, files, summary: recomputeSummary(files, status.summary) };
}

/** Move paths into the staged bucket, removing them from unstaged/untracked/conflicts. */
export function applyStageOptimistic(status: RepoStatus, paths: string[]): RepoStatus {
  const moving = new Set(paths);
  const files = status.files;
  const staged = [...bucket(files.staged)];
  for (const path of paths) {
    if (!staged.includes(path)) staged.push(path);
  }
  return withFiles(status, {
    ...files,
    staged,
    unstaged: bucket(files.unstaged).filter((p) => !moving.has(p)),
    untracked: bucket(files.untracked).filter((p) => !moving.has(p)),
    conflicts: bucket(files.conflicts).filter((p) => !moving.has(p)),
  });
}

/**
 * Move paths out of the staged bucket. A staged-new file (index status "A")
 * returns to untracked; everything else returns to unstaged.
 */
export function applyUnstageOptimistic(status: RepoStatus, paths: string[]): RepoStatus {
  const moving = new Set(paths);
  const files = status.files;
  const statuses = files.statuses ?? {};
  const unstaged = [...bucket(files.unstaged)];
  const untracked = [...bucket(files.untracked)];
  for (const path of paths) {
    const wasAdded = (statuses[path] ?? "").charAt(0) === "A";
    if (wasAdded) {
      if (!untracked.includes(path)) untracked.push(path);
    } else if (!unstaged.includes(path)) {
      unstaged.push(path);
    }
  }
  return withFiles(status, {
    ...files,
    staged: bucket(files.staged).filter((p) => !moving.has(p)),
    unstaged,
    untracked,
  });
}

/** Remove discarded paths from the worktree buckets (unstaged/untracked/conflicts). */
export function applyDiscardOptimistic(status: RepoStatus, paths: string[]): RepoStatus {
  const removing = new Set(paths);
  const files = status.files;
  return withFiles(status, {
    ...files,
    unstaged: bucket(files.unstaged).filter((p) => !removing.has(p)),
    untracked: bucket(files.untracked).filter((p) => !removing.has(p)),
    conflicts: bucket(files.conflicts).filter((p) => !removing.has(p)),
  });
}

export function useStageFiles(repoId?: string | null) {
  const queryClient = useQueryClient();
  const statusKey = queryKeys.repoStatus(repoId);
  return useMutation({
    mutationFn: (request: StageRequest) => stageFiles(request, repoId ?? undefined),
    onMutate: async (request: StageRequest) => {
      await queryClient.cancelQueries({ queryKey: statusKey });
      const previous = queryClient.getQueryData<RepoStatus>(statusKey);
      if (previous) {
        queryClient.setQueryData<RepoStatus>(statusKey, applyStageOptimistic(previous, request.paths));
      }
      return { previous };
    },
    onError: (_err, _request, context) => {
      if (context?.previous) {
        queryClient.setQueryData(statusKey, context.previous);
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: statusKey });
    },
  });
}

export function useUnstageFiles(repoId?: string | null) {
  const queryClient = useQueryClient();
  const statusKey = queryKeys.repoStatus(repoId);
  return useMutation({
    mutationFn: (request: UnstageRequest) => unstageFiles(request, repoId ?? undefined),
    onMutate: async (request: UnstageRequest) => {
      await queryClient.cancelQueries({ queryKey: statusKey });
      const previous = queryClient.getQueryData<RepoStatus>(statusKey);
      if (previous) {
        queryClient.setQueryData<RepoStatus>(statusKey, applyUnstageOptimistic(previous, request.paths));
      }
      return { previous };
    },
    onError: (_err, _request, context) => {
      if (context?.previous) {
        queryClient.setQueryData(statusKey, context.previous);
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: statusKey });
    },
  });
}

export function useSyncStatus(repoId?: string | null) {
  return useQuery({
    queryKey: queryKeys.syncStatus(repoId),
    queryFn: () => fetchSyncStatus(false, repoId ?? undefined),
    refetchInterval: 15_000,
    staleTime: 5_000,
  });
}

export function useCommit(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (request: CommitRequest) => createCommit(request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus(repoId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.syncStatus(repoId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.approvedChanges(repoId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.provenance(repoId) });
    }
  });
}

export function usePrecommitConfig(repoId?: string | null) {
  return useQuery({
    queryKey: ["repo", "precommit", repoId ?? "default"],
    queryFn: () => fetchPrecommitConfig(repoId ?? undefined),
    staleTime: 10_000,
  });
}

export function useSavePrecommitConfig(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (config: PrecommitConfig) => savePrecommitConfig(config, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["repo", "precommit", repoId ?? "default"] });
    },
  });
}

export function useRunPrecommit(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (request: PrecommitRunRequest = {}) => runPrecommit(request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["repo", "precommit", repoId ?? "default"] });
    },
  });
}

export interface PrecommitStreamState {
  running: boolean;
  command?: string;
  elapsedMs: number;
  tail: string[];
  finished?: PrecommitStreamEvent;
  error?: string;
}

export function useStreamPrecommit(repoId?: string | null) {
  const queryClient = useQueryClient();
  const [state, setState] = useState<PrecommitStreamState>({ running: false, elapsedMs: 0, tail: [] });
  const startedAtRef = useRef<number | null>(null);
  const abortRef = useRef<AbortController | null>(null);
  const tickRef = useRef<number | null>(null);

  useEffect(() => {
    return () => {
      if (tickRef.current !== null) window.clearInterval(tickRef.current);
      abortRef.current?.abort();
    };
  }, []);

  const cancel = useCallback(() => {
    abortRef.current?.abort();
  }, []);

  const reset = useCallback(() => {
    setState({ running: false, elapsedMs: 0, tail: [] });
  }, []);

  const run = useCallback(
    async (request: PrecommitRunRequest = {}): Promise<PrecommitStreamEvent> => {
      abortRef.current?.abort();
      const controller = new AbortController();
      abortRef.current = controller;
      startedAtRef.current = Date.now();
      setState({ running: true, elapsedMs: 0, tail: [], command: request.command });
      if (tickRef.current !== null) window.clearInterval(tickRef.current);
      tickRef.current = window.setInterval(() => {
        const startedAt = startedAtRef.current;
        if (startedAt !== null) {
          setState((prev) => (prev.running ? { ...prev, elapsedMs: Date.now() - startedAt } : prev));
        }
      }, 250);

      try {
        const final = await runPrecommitStream(request, repoId ?? undefined, {
          signal: controller.signal,
          onEvent: (event) => {
            setState((prev) => {
              const next: PrecommitStreamState = { ...prev };
              if (event.command) next.command = event.command;
              if (event.tail) next.tail = event.tail;
              next.elapsedMs = event.elapsed_ms;
              return next;
            });
          },
        });
        setState((prev) => ({ ...prev, running: false, finished: final, error: final.error }));
        queryClient.invalidateQueries({ queryKey: ["repo", "precommit", repoId ?? "default"] });
        return final;
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        setState((prev) => ({ ...prev, running: false, error: message }));
        throw err;
      } finally {
        if (tickRef.current !== null) {
          window.clearInterval(tickRef.current);
          tickRef.current = null;
        }
        abortRef.current = null;
        startedAtRef.current = null;
      }
    },
    [queryClient, repoId]
  );

  return { state, run, cancel, reset };
}

export function useDiscardFiles(repoId?: string | null) {
  const queryClient = useQueryClient();
  const statusKey = queryKeys.repoStatus(repoId);
  return useMutation({
    mutationFn: (request: DiscardRequest) => discardFiles(request, repoId ?? undefined),
    onMutate: async (request: DiscardRequest) => {
      await queryClient.cancelQueries({ queryKey: statusKey });
      const previous = queryClient.getQueryData<RepoStatus>(statusKey);
      if (previous) {
        queryClient.setQueryData<RepoStatus>(statusKey, applyDiscardOptimistic(previous, request.paths));
      }
      return { previous };
    },
    onError: (_err, _request, context) => {
      if (context?.previous) {
        queryClient.setQueryData(statusKey, context.previous);
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: statusKey });
    },
  });
}

export function useIgnoreFile(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (request: IgnoreRequest) => ignoreFile(request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus(repoId) });
    }
  });
}

export function usePush(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (request: PushRequest = {}) => pushToRemote(request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.syncStatus(repoId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus(repoId) });
    }
  });
}

export function useSaveFileContent(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<SaveFileContentResponse, Error, SaveFileContentRequest>({
    mutationFn: (request: SaveFileContentRequest) => saveFileContent(request, repoId ?? undefined),
    onSuccess: (_result, request) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus(repoId) });
      queryClient.invalidateQueries({
        queryKey: ["repo", "diff", repoId ?? "default", request.path]
      });
    }
  });
}

export function usePull(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (request: PullRequest = {}) => pullFromRemote(request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.syncStatus(repoId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus(repoId) });
    }
  });
}

export function useApprovedChanges(repoId?: string | null) {
  return useQuery({
    queryKey: queryKeys.approvedChanges(repoId),
    queryFn: () => fetchApprovedChanges(repoId ?? undefined),
    refetchInterval: 15_000,
    staleTime: 5_000,
  });
}

export function useApprovedChangesPreview(repoId?: string | null) {
  return useMutation({
    mutationFn: (request: ApprovedChangesPreviewRequest) =>
      fetchApprovedChangesPreview(request, repoId ?? undefined)
  });
}

export function useProvenance(repoId?: string | null) {
  return useQuery({
    queryKey: queryKeys.provenance(repoId),
    queryFn: () => fetchProvenance(repoId ?? undefined),
    refetchInterval: 30_000,
    staleTime: 10_000,
  });
}

export function useBranches(repoId?: string | null) {
  return useQuery({
    queryKey: queryKeys.branches(repoId),
    queryFn: () => fetchBranches(repoId ?? undefined),
    refetchInterval: 30000
  });
}

export function useCreateBranch(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (request: CreateBranchRequest) => createBranch(request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus(repoId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.branches(repoId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.syncStatus(repoId) });
    }
  });
}

export function useSwitchBranch(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (request: SwitchBranchRequest) => switchBranch(request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus(repoId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.branches(repoId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.syncStatus(repoId) });
    }
  });
}

export function usePublishBranch(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (request: PublishBranchRequest = {}) => publishBranch(request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus(repoId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.branches(repoId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.syncStatus(repoId) });
    }
  });
}

export function useFileSearch(pattern?: string, deep = false, enabled = true, repoId?: string | null) {
  return useQuery<FileTreeResponse, Error>({
    queryKey: queryKeys.files(pattern, deep, repoId),
    queryFn: () => fetchFiles(pattern, 1000, deep, 5000, repoId ?? undefined),
    enabled
  });
}

export function useRelatedFiles(path: string, enabled = true, repoId?: string | null) {
  return useQuery<RelatedFilesResponse, Error>({
    queryKey: queryKeys.relatedFiles(path, repoId),
    queryFn: () => fetchRelatedFiles(path, repoId ?? undefined),
    enabled: enabled && Boolean(path)
  });
}

export function useDirectoryContents(path: string, enabled = true, repoId?: string | null) {
  return useQuery<DirListResponse, Error>({
    queryKey: queryKeys.directoryContents(path, repoId),
    queryFn: () => fetchDirectoryContents(path, repoId ?? undefined),
    enabled,
    staleTime: 30000
  });
}

export function useDeletePath(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (request: DeletePathRequest) => deletePath(request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus(repoId) });
      queryClient.invalidateQueries({
        queryKey: ["repo", "dir", repoId ?? "default"]
      });
    }
  });
}

export function useContentSearch(
  query: string,
  options: Omit<ContentSearchRequest, "query"> = {},
  enabled = true,
  repoId?: string | null
) {
  const request: ContentSearchRequest = { query, ...options };
  return useQuery<ContentSearchResponse, Error>({
    queryKey: queryKeys.contentSearch(query, options, repoId),
    queryFn: () => searchContent(request, repoId ?? undefined),
    enabled: enabled && query.length >= 2
  });
}
