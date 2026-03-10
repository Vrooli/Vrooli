import { useCallback, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import type { DiffStats } from "./api";
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
  saveFileContent,
  searchContent,
  fetchCredentials,
  saveCredential,
  deleteCredential,
  testCredential,
  updateRemoteURL,
  fetchRepos,
  fetchActiveRepo,
  openRepo,
  cloneRepo,
  setActiveRepo,
  removeRepo,
  fetchSSHKeys,
  generateSSHKey,
  getSSHPublicKey,
  testSSHConnection,
  deleteSSHKey,
  fetchCapabilities,
  fetchGroupingRules,
  saveGroupingRules,
  fetchGitignoreHealth,
  moveGitignoreEntry,
  type CapabilitiesResponse,
  type GroupingRulesConfig,
  type GitignoreMoveRequest,
  type GitignoreHealthResponse,
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
  type SaveFileContentRequest,
  type SaveFileContentResponse,
  type ContentSearchRequest,
  type ContentSearchResponse,
  type CredentialsListResponse,
  type CredentialSaveRequest,
  type CredentialTestRequest,
  type RemoteURLUpdateRequest,
  type RepoOpenRequest,
  type RepoCloneRequest,
  type RepoActiveRequest,
  type SSHListKeysResponse,
  type SSHGenerateKeyRequest,
  type SSHGetPublicKeyRequest,
  type SSHTestConnectionRequest,
  type SSHDeleteKeyRequest
} from "./api";

export const queryKeys = {
  health: ["health"] as const,
  repoStatus: (repoId?: string | null) => ["repo", "status", repoId ?? "default"] as const,
  repoHistory: (limit?: number, includeFiles?: boolean, repoId?: string | null) =>
    ["repo", "history", repoId ?? "default", limit, includeFiles] as const,
  syncStatus: (repoId?: string | null) => ["repo", "sync-status", repoId ?? "default"] as const,
  branches: (repoId?: string | null) => ["repo", "branches", repoId ?? "default"] as const,
  diff: (
    path?: string,
    staged?: boolean,
    untracked?: boolean,
    commit?: string,
    mode?: ViewMode,
    any?: boolean,
    repoId?: string | null
  ) => ["repo", "diff", repoId ?? "default", path, staged, untracked, commit, mode, any] as const,
  approvedChanges: (repoId?: string | null) =>
    ["repo", "approved-changes", repoId ?? "default"] as const,
  files: (pattern?: string, deep?: boolean, repoId?: string | null) =>
    ["repo", "files", repoId ?? "default", pattern, deep] as const,
  relatedFiles: (path: string, repoId?: string | null) =>
    ["repo", "related", repoId ?? "default", path] as const,
  directoryContents: (path: string, repoId?: string | null) =>
    ["repo", "dir", repoId ?? "default", path] as const,
  contentSearch: (query: string, opts?: Partial<ContentSearchRequest>, repoId?: string | null) =>
    ["repo", "search", "content", repoId ?? "default", query, opts] as const,
  credentials: (repoId?: string | null) => ["credentials", repoId ?? "default"] as const,
  groupingRules: (repoId?: string | null) => ["repo", "grouping-rules", repoId ?? "default"] as const,
  gitignoreHealth: (repoId?: string | null) => ["repo", "gitignore", "health", repoId ?? "default"] as const,
  capabilities: ["capabilities"] as const,
  sshKeys: ["ssh", "keys"] as const,
  repos: ["repos"] as const,
  activeRepo: ["repos", "active"] as const
};

const REPO_STORAGE_KEY = "gct.activeRepoId";

function readStoredRepoId(): string | null {
  if (typeof window === "undefined") return null;
  try {
    const value = window.localStorage.getItem(REPO_STORAGE_KEY);
    return value && value.trim().length > 0 ? value : null;
  } catch {
    return null;
  }
}

function persistRepoId(repoId: string | null) {
  if (typeof window === "undefined") return;
  try {
    if (repoId && repoId.trim().length > 0) {
      window.localStorage.setItem(REPO_STORAGE_KEY, repoId);
    } else {
      window.localStorage.removeItem(REPO_STORAGE_KEY);
    }
  } catch {
    // Ignore storage errors; repo selection still works in-memory.
  }
}

export function useRepoSelection() {
  const [repoId, setRepoIdState] = useState<string | null>(() => readStoredRepoId());

  const setRepoId = useCallback((next: string | null) => {
    const normalized = next && next.trim().length > 0 ? next : null;
    setRepoIdState(normalized);
    persistRepoId(normalized);
  }, []);

  return { repoId, setRepoId };
}

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
    refetchInterval: 5000
  });
}

export function useRepoHistory(limit = 30, includeFiles = false, repoId?: string | null) {
  return useQuery<RepoHistoryResponse, Error>({
    queryKey: queryKeys.repoHistory(limit, includeFiles, repoId),
    queryFn: () => fetchRepoHistory(limit, includeFiles, repoId ?? undefined),
    refetchInterval: 30000
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
    // Only enable when we have a valid path, especially important for "any" file viewing
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

export function useStageFiles(repoId?: string | null) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: StageRequest) => stageFiles(request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus(repoId) });
    }
  });
}

export function useUnstageFiles(repoId?: string | null) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: UnstageRequest) => unstageFiles(request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus(repoId) });
    }
  });
}

export function useSyncStatus(repoId?: string | null) {
  return useQuery({
    queryKey: queryKeys.syncStatus(repoId),
    queryFn: () => fetchSyncStatus(false, repoId ?? undefined),
    refetchInterval: 5000
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
    }
  });
}

export function useDiscardFiles(repoId?: string | null) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: DiscardRequest) => discardFiles(request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus(repoId) });
    }
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
    refetchInterval: 5000
  });
}

export function useApprovedChangesPreview(repoId?: string | null) {
  return useMutation({
    mutationFn: (request: ApprovedChangesPreviewRequest) =>
      fetchApprovedChangesPreview(request, repoId ?? undefined)
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

export function useFileSearch(
  pattern?: string,
  deep = false,
  enabled = true,
  repoId?: string | null
) {
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
    staleTime: 30000 // Cache for 30 seconds
  });
}

export function useDeletePath(repoId?: string | null) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: DeletePathRequest) => deletePath(request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus(repoId) });
      // Invalidate all directory contents caches since structure changed
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
    enabled: enabled && query.length >= 2 // Minimum 2 characters
  });
}

// ============================================================================
// Capabilities Hooks
// ============================================================================

export function useCapabilities() {
  return useQuery<CapabilitiesResponse, Error>({
    queryKey: queryKeys.capabilities,
    queryFn: fetchCapabilities,
    refetchInterval: 30_000,
  });
}

// ============================================================================
// Credentials Hooks
// ============================================================================

export function useCredentials(repoId?: string | null) {
  return useQuery<CredentialsListResponse, Error>({
    queryKey: queryKeys.credentials(repoId),
    queryFn: () => fetchCredentials(repoId ?? undefined)
  });
}

export function useSaveCredential(repoId?: string | null) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: CredentialSaveRequest) => saveCredential(request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.credentials(repoId) });
    }
  });
}

export function useDeleteCredential(repoId?: string | null) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => deleteCredential(id, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.credentials(repoId) });
    }
  });
}

export function useTestCredential(repoId?: string | null) {
  return useMutation({
    mutationFn: (request: CredentialTestRequest) => testCredential(request, repoId ?? undefined)
  });
}

export function useUpdateRemoteURL(repoId?: string | null) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: RemoteURLUpdateRequest) => updateRemoteURL(request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.credentials(repoId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.syncStatus(repoId) });
    }
  });
}

// ============================================================================
// Repo Registry Hooks
// ============================================================================

export function useRepos() {
  return useQuery({
    queryKey: queryKeys.repos,
    queryFn: fetchRepos,
    refetchInterval: 30000
  });
}

export function useActiveRepo() {
  return useQuery({
    queryKey: queryKeys.activeRepo,
    queryFn: fetchActiveRepo,
    refetchInterval: 30000
  });
}

export function useOpenRepo() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (request: RepoOpenRequest) => openRepo(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.repos });
      queryClient.invalidateQueries({ queryKey: queryKeys.activeRepo });
    }
  });
}

export function useCloneRepo() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (request: RepoCloneRequest) => cloneRepo(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.repos });
      queryClient.invalidateQueries({ queryKey: queryKeys.activeRepo });
    }
  });
}

export function useSetActiveRepo() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (request: RepoActiveRequest) => setActiveRepo(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.repos });
      queryClient.invalidateQueries({ queryKey: queryKeys.activeRepo });
    }
  });
}

export function useRemoveRepo() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => removeRepo(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.repos });
      queryClient.invalidateQueries({ queryKey: queryKeys.activeRepo });
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

// ============================================================================
// Grouping Rules Hooks
// ============================================================================

export function useGroupingRules(repoId?: string | null) {
  return useQuery<GroupingRulesConfig, Error>({
    queryKey: queryKeys.groupingRules(repoId),
    queryFn: () => fetchGroupingRules(repoId ?? undefined),
    enabled: Boolean(repoId),
  });
}

export function useSaveGroupingRules(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (config: GroupingRulesConfig) => saveGroupingRules(config, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.groupingRules(repoId) });
    },
  });
}

// ============================================================================
// Gitignore Health Hooks
// ============================================================================

export function useGitignoreHealth(repoId?: string | null) {
  return useQuery<GitignoreHealthResponse, Error>({
    queryKey: queryKeys.gitignoreHealth(repoId),
    queryFn: () => fetchGitignoreHealth(repoId ?? undefined),
    enabled: Boolean(repoId),
  });
}

export function useGitignoreMove(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (request: GitignoreMoveRequest) => moveGitignoreEntry(request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.gitignoreHealth(repoId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus(repoId) });
    },
  });
}
