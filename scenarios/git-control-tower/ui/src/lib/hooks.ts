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
  deletePath,
  searchContent,
  fetchCredentials,
  saveCredential,
  deleteCredential,
  testCredential,
  updateRemoteURL,
  fetchSSHKeys,
  generateSSHKey,
  getSSHPublicKey,
  testSSHConnection,
  deleteSSHKey,
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
  type DirListResponse,
  type DeletePathRequest,
  type ContentSearchRequest,
  type ContentSearchResponse,
  type CredentialsListResponse,
  type CredentialSaveRequest,
  type CredentialTestRequest,
  type RemoteURLUpdateRequest,
  type SSHListKeysResponse,
  type SSHGenerateKeyRequest,
  type SSHGetPublicKeyRequest,
  type SSHTestConnectionRequest,
  type SSHDeleteKeyRequest
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
  directoryContents: (path: string) => ["repo", "dir", path] as const,
  contentSearch: (query: string, opts?: Partial<ContentSearchRequest>) =>
    ["repo", "search", "content", query, opts] as const,
  credentials: ["credentials"] as const,
  sshKeys: ["ssh", "keys"] as const
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

export function useDeletePath() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: DeletePathRequest) => deletePath(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus });
      // Invalidate all directory contents caches since structure changed
      queryClient.invalidateQueries({ queryKey: ["repo", "dir"] });
    }
  });
}

export function useContentSearch(
  query: string,
  options: Omit<ContentSearchRequest, "query"> = {},
  enabled = true
) {
  const request: ContentSearchRequest = { query, ...options };
  return useQuery<ContentSearchResponse, Error>({
    queryKey: queryKeys.contentSearch(query, options),
    queryFn: () => searchContent(request),
    enabled: enabled && query.length >= 2 // Minimum 2 characters
  });
}

// ============================================================================
// Credentials Hooks
// ============================================================================

export function useCredentials() {
  return useQuery<CredentialsListResponse, Error>({
    queryKey: queryKeys.credentials,
    queryFn: fetchCredentials
  });
}

export function useSaveCredential() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: CredentialSaveRequest) => saveCredential(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.credentials });
    }
  });
}

export function useDeleteCredential() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => deleteCredential(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.credentials });
    }
  });
}

export function useTestCredential() {
  return useMutation({
    mutationFn: (request: CredentialTestRequest) => testCredential(request)
  });
}

export function useUpdateRemoteURL() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: RemoteURLUpdateRequest) => updateRemoteURL(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.credentials });
      queryClient.invalidateQueries({ queryKey: queryKeys.syncStatus });
    }
  });
}

// ============================================================================
// SSH Key Management Hooks
// ============================================================================

export function useSSHKeys() {
  return useQuery<SSHListKeysResponse, Error>({
    queryKey: queryKeys.sshKeys,
    queryFn: fetchSSHKeys
  });
}

export function useGenerateSSHKey() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: SSHGenerateKeyRequest) => generateSSHKey(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.sshKeys });
    }
  });
}

export function useGetSSHPublicKey() {
  return useMutation({
    mutationFn: (request: SSHGetPublicKeyRequest) => getSSHPublicKey(request)
  });
}

export function useTestSSHConnection() {
  return useMutation({
    mutationFn: (request: SSHTestConnectionRequest) => testSSHConnection(request)
  });
}

export function useDeleteSSHKey() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: SSHDeleteKeyRequest) => deleteSSHKey(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.sshKeys });
    }
  });
}
