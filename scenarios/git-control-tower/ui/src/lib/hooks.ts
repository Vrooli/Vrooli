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
  fetchBranches,
  createBranch,
  switchBranch,
  publishBranch,
  fetchFiles,
  fetchRelatedFiles,
  fetchDirectoryContents,
  type RepoHistoryResponse,
  type StageRequest,
  type UnstageRequest,
  type CommitRequest,
  type DiscardRequest,
  type IgnoreRequest,
  type PushRequest,
  type PullRequest,
  type CreateBranchRequest,
  type SwitchBranchRequest,
  type PublishBranchRequest,
  type ApprovedChangesPreviewRequest,
  type ViewMode,
  type FileTreeResponse,
  type RelatedFilesResponse,
  type DirListResponse
} from "./api";

export const queryKeys = {
  health: ["health"] as const,
  repoStatus: ["repo", "status"] as const,
  repoHistory: (limit?: number, includeFiles?: boolean) =>
    ["repo", "history", limit, includeFiles] as const,
  syncStatus: ["repo", "sync-status"] as const,
  branches: ["repo", "branches"] as const,
  diff: (path?: string, staged?: boolean, untracked?: boolean, commit?: string, mode?: ViewMode, any?: boolean) =>
    ["repo", "diff", path, staged, untracked, commit, mode, any] as const,
  approvedChanges: ["repo", "approved-changes"] as const,
  files: (pattern?: string, deep?: boolean) => ["repo", "files", pattern, deep] as const,
  relatedFiles: (path: string) => ["repo", "related", path] as const,
  directoryContents: (path: string) => ["repo", "dir", path] as const
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

export function useDiff(path?: string, staged = false, untracked = false, commit?: string, mode: ViewMode = "diff", any = false) {
  return useQuery({
    queryKey: queryKeys.diff(path, staged, untracked, commit, mode, any),
    queryFn: () => fetchDiff(path, staged, untracked, commit, mode, any),
    // Only enable when we have a valid path, especially important for "any" file viewing
    enabled: Boolean(path)
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
    refetchInterval: 5000
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

export function useBranches() {
  return useQuery({
    queryKey: queryKeys.branches,
    queryFn: fetchBranches,
    refetchInterval: 30000
  });
}

export function useCreateBranch() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: CreateBranchRequest) => createBranch(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus });
      queryClient.invalidateQueries({ queryKey: queryKeys.branches });
      queryClient.invalidateQueries({ queryKey: queryKeys.syncStatus });
    }
  });
}

export function useSwitchBranch() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: SwitchBranchRequest) => switchBranch(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus });
      queryClient.invalidateQueries({ queryKey: queryKeys.branches });
      queryClient.invalidateQueries({ queryKey: queryKeys.syncStatus });
    }
  });
}

export function usePublishBranch() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: PublishBranchRequest = {}) => publishBranch(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus });
      queryClient.invalidateQueries({ queryKey: queryKeys.branches });
      queryClient.invalidateQueries({ queryKey: queryKeys.syncStatus });
    }
  });
}

export function useFileSearch(pattern?: string, deep = false, enabled = true) {
  return useQuery<FileTreeResponse, Error>({
    queryKey: queryKeys.files(pattern, deep),
    queryFn: () => fetchFiles(pattern, 1000, deep, 5000),
    enabled
  });
}

export function useRelatedFiles(path: string, enabled = true) {
  return useQuery<RelatedFilesResponse, Error>({
    queryKey: queryKeys.relatedFiles(path),
    queryFn: () => fetchRelatedFiles(path),
    enabled: enabled && Boolean(path)
  });
}

export function useDirectoryContents(path: string, enabled = true) {
  return useQuery<DirListResponse, Error>({
    queryKey: queryKeys.directoryContents(path),
    queryFn: () => fetchDirectoryContents(path),
    enabled,
    staleTime: 30000 // Cache for 30 seconds
  });
}
