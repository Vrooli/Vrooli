import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchHealth,
  fetchRepoStatus,
  fetchRepoHistory,
  fetchDiff,
  fetchSyncStatus,
  fetchApprovedChanges,
  fetchApprovedChangesPreview,
  stageFiles,
  unstageFiles,
  createCommit,
  discardFiles,
  ignoreFile,
  pushToRemote,
  pullFromRemote,
  type RepoHistoryResponse,
  type StageRequest,
  type UnstageRequest,
  type CommitRequest,
  type DiscardRequest,
  type IgnoreRequest,
  type PushRequest,
  type PullRequest,
  type ApprovedChangesPreviewRequest,
  type ViewMode
} from "./api";

export const queryKeys = {
  health: ["health"] as const,
  repoStatus: ["repo", "status"] as const,
  repoHistory: (limit?: number, includeFiles?: boolean) =>
    ["repo", "history", limit, includeFiles] as const,
  syncStatus: ["repo", "sync-status"] as const,
  diff: (path?: string, staged?: boolean, untracked?: boolean, commit?: string, mode?: ViewMode) =>
    ["repo", "diff", path, staged, untracked, commit, mode] as const,
  approvedChanges: ["repo", "approved-changes"] as const
};

export function useHealth() {
  return useQuery({
    queryKey: queryKeys.health,
    queryFn: fetchHealth,
    refetchInterval: 30000
  });
}

export function useRepoStatus() {
  return useQuery({
    queryKey: queryKeys.repoStatus,
    queryFn: fetchRepoStatus,
    refetchInterval: 5000
  });
}

export function useRepoHistory(limit = 30, includeFiles = false) {
  return useQuery<RepoHistoryResponse, Error>({
    queryKey: queryKeys.repoHistory(limit, includeFiles),
    queryFn: () => fetchRepoHistory(limit, includeFiles),
    refetchInterval: 30000
  });
}

export function useDiff(path?: string, staged = false, untracked = false, commit?: string, mode: ViewMode = "diff") {
  return useQuery({
    queryKey: queryKeys.diff(path, staged, untracked, commit, mode),
    queryFn: () => fetchDiff(path, staged, untracked, commit, mode),
    enabled: true
  });
}

export function useStageFiles() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: StageRequest) => stageFiles(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus });
    }
  });
}

export function useUnstageFiles() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: UnstageRequest) => unstageFiles(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus });
    }
  });
}

export function useSyncStatus() {
  return useQuery({
    queryKey: queryKeys.syncStatus,
    queryFn: () => fetchSyncStatus(false),
    refetchInterval: 30000
  });
}

export function useCommit() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: CommitRequest) => createCommit(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus });
      queryClient.invalidateQueries({ queryKey: queryKeys.syncStatus });
      queryClient.invalidateQueries({ queryKey: queryKeys.approvedChanges });
    }
  });
}

export function useDiscardFiles() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: DiscardRequest) => discardFiles(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus });
    }
  });
}

export function useIgnoreFile() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: IgnoreRequest) => ignoreFile(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus });
    }
  });
}

export function usePush() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: PushRequest = {}) => pushToRemote(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.syncStatus });
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus });
    }
  });
}

export function usePull() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: PullRequest = {}) => pullFromRemote(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.syncStatus });
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus });
    }
  });
}

export function useApprovedChanges() {
  return useQuery({
    queryKey: queryKeys.approvedChanges,
    queryFn: fetchApprovedChanges,
    refetchInterval: 5000
  });
}

export function useApprovedChangesPreview() {
  return useMutation({
    mutationFn: (request: ApprovedChangesPreviewRequest) => fetchApprovedChangesPreview(request)
  });
}
