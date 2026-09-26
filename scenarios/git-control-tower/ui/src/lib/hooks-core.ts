// ============================================================================
// Core Git/Repo Hooks — health, status, diff, stage, commit, branches, files
// ============================================================================

import { useCallback, useEffect, useRef, useState, useMemo } from "react";
import { useQuery, useMutation, useQueryClient, useIsMutating, useMutationState, type QueryClient } from "@tanstack/react-query";
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
  fetchBlame,
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
  CommitRequest,
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
  const pending = useIsMutating({ mutationKey: workspaceMutationKey(repoId) });
  return useQuery({
    queryKey: queryKeys.repoStatus(repoId),
    queryFn: () => fetchRepoStatus(repoId ?? undefined),
    enabled: pending === 0,
    refetchInterval: pending ? false : 15_000,
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
  const staged = [...new Set([...bucket(files.staged), ...paths])];
  const statuses = { ...files.statuses };
  const untracked = new Set(bucket(files.untracked));
  for (const path of paths) {
    if (untracked.has(path)) statuses[path] = "A ";
  }
  return withFiles(status, {
    ...files,
    staged,
    statuses,
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

const workspaceMutationKey = (repoId?: string | null) => ["repo", "workspace-mutation", repoId ?? "default"];
type WorkspaceRequest = { paths: string[] };
type OptimisticOperation = { apply: (status: RepoStatus) => RepoStatus; failed: boolean; settled: boolean };
type OptimisticBatch = { base?: RepoStatus; operations: OptimisticOperation[] };
const optimisticBatches = new WeakMap<QueryClient, Map<string, OptimisticBatch>>();

// Each request is visible immediately, but its authorization + Git write is
// serialized per repository. Replaying the batch preserves later clicks when
// an earlier request fails, and reconciles only after the whole queue drains.
function useWorkspaceMutation<Request extends WorkspaceRequest, Response extends { success: boolean; errors?: string[] }>(
  repoId: string | null | undefined,
  operation: string,
  run: (request: Request, repoId?: string) => Promise<Response>,
  apply: (status: RepoStatus, paths: string[]) => RepoStatus,
) {
  const queryClient = useQueryClient();
  const [failure, setFailure] = useState<Error | null>(null);
  const statusKey = queryKeys.repoStatus(repoId);
  const id = repoId ?? "default";
  const project = (batch: OptimisticBatch) => {
    if (!batch.base) return;
    const status = batch.operations.reduce((status, item) => item.failed ? status : item.apply(status), batch.base);
    queryClient.setQueryData(statusKey, status);
  };
  const mutation = useMutation({
    mutationKey: [...workspaceMutationKey(repoId), operation],
    scope: { id: JSON.stringify(workspaceMutationKey(repoId)) },
    mutationFn: async (request: Request) => {
      const response = await run(request, repoId ?? undefined);
      if (!response.success) throw new Error(response.errors?.join("; ") || `${operation} failed`);
      return response;
    },
    onMutate: async (request: Request) => {
      await queryClient.cancelQueries({ queryKey: statusKey });
      let batches = optimisticBatches.get(queryClient);
      if (!batches) { batches = new Map(); optimisticBatches.set(queryClient, batches); }
      let batch = batches.get(id);
      if (!batch) { batch = { base: queryClient.getQueryData<RepoStatus>(statusKey), operations: [] }; batches.set(id, batch); }
      const item = { apply: (status: RepoStatus) => apply(status, request.paths), failed: false, settled: false };
      batch.operations.push(item);
      project(batch);
      return { batch, item };
    },
    onError: (error, _request, context) => {
      setFailure(error);
      if (context) { context.item.failed = true; project(context.batch); }
    },
    onSettled: (_data, _error, _request, context) => {
      if (context) {
        context.item.settled = true;
        if (context.batch.operations.some(item => !item.settled)) return;
        optimisticBatches.get(queryClient)?.delete(id);
      }
      void queryClient.invalidateQueries({ queryKey: statusKey });
    },
  });
  return { ...mutation, error: failure, reset: () => {
    setFailure(null);
    // An older queued request can fail while the observed request is pending.
    // Dismissing its toast must not detach that pending request's callbacks.
    if (!mutation.isPending) mutation.reset();
  } };
}

export function usePendingWorkspaceChanges(repoId?: string | null) {
  const requests = useMutationState({
    filters: { mutationKey: workspaceMutationKey(repoId), status: "pending" },
    select: mutation => mutation.state.variables as WorkspaceRequest,
  });
  return useMemo(() => ({ count: requests.length, paths: new Set(requests.flatMap(request => request.paths)) }), [requests]);
}

export function useStageFiles(repoId?: string | null) {
  return useWorkspaceMutation(repoId, "stage", stageFiles, applyStageOptimistic);
}

export function useUnstageFiles(repoId?: string | null) {
  return useWorkspaceMutation(repoId, "unstage", unstageFiles, applyUnstageOptimistic);
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
    onSuccess: (result) => {
      if (!result.success) return;
      void queryClient.invalidateQueries({ queryKey: ["repo", "history", repoId ?? "default"] });
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
  return useWorkspaceMutation(repoId, "discard", discardFiles, applyDiscardOptimistic);
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

export function useBlame(path?: string | null, repoId?: string | null) {
  return useQuery({
    queryKey: queryKeys.blame(path ?? "", repoId),
    queryFn: () => fetchBlame([path ?? ""], repoId ?? undefined),
    enabled: Boolean(path),
    staleTime: 30_000,
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
