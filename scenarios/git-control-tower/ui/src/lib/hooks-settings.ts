// ============================================================================
// Settings Hooks — Capabilities, Credentials, Repos, SSH, Grouping, Gitignore
// ============================================================================

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "./hooks-query-keys";
import { invalidateCredentials, invalidateRepositoryRegistry } from "./hooks-query-invalidations";
import {
  fetchCapabilities,
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
  fetchGroupingRules,
  fetchRepoGroups,
  saveGroupingRules,
  fetchGitignoreHealth,
  moveGitignoreEntry,
  fetchTrackedBinaries,
  untrackBinary,
} from "./api";
import type {
  CapabilitiesResponse,
  CredentialsListResponse,
  CredentialSaveRequest,
  CredentialTestRequest,
  RemoteURLUpdateRequest,
  RepoOpenRequest,
  RepoCloneRequest,
  RepoActiveRequest,
  SSHListKeysResponse,
  SSHGenerateKeyRequest,
  SSHGetPublicKeyRequest,
  SSHTestConnectionRequest,
  SSHDeleteKeyRequest,
  GroupingRulesConfig,
  RepoGroupsResponse,
  GitignoreHealthResponse,
  TrackedBinariesResponse,
  UntrackBinaryRequest,
  GitignoreMoveRequest,
} from "./api";

// ── Capabilities ───────────────────────────────────────────────────────

export function useCapabilities() {
  return useQuery<CapabilitiesResponse, Error>({
    queryKey: queryKeys.capabilities,
    queryFn: fetchCapabilities,
    refetchInterval: 30_000,
  });
}

// ── Credentials ────────────────────────────────────────────────────────

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
      invalidateCredentials(queryClient, repoId);
    }
  });
}

export function useDeleteCredential(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deleteCredential(id, repoId ?? undefined),
    onSuccess: () => {
      invalidateCredentials(queryClient, repoId);
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
      invalidateCredentials(queryClient, repoId);
      queryClient.invalidateQueries({ queryKey: queryKeys.syncStatus(repoId) });
    }
  });
}

// ── Repo Registry ──────────────────────────────────────────────────────

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
      invalidateRepositoryRegistry(queryClient);
    }
  });
}

export function useCloneRepo() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (request: RepoCloneRequest) => cloneRepo(request),
    onSuccess: () => {
      invalidateRepositoryRegistry(queryClient);
    }
  });
}

export function useSetActiveRepo() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (request: RepoActiveRequest) => setActiveRepo(request),
    onSuccess: () => {
      invalidateRepositoryRegistry(queryClient);
    }
  });
}

export function useRemoveRepo() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => removeRepo(id),
    onSuccess: () => {
      invalidateRepositoryRegistry(queryClient);
    }
  });
}

// ── SSH Key Management ─────────────────────────────────────────────────

export function useSSHKeys() {
  return useQuery<SSHListKeysResponse, Error>({
    queryKey: queryKeys.sshKeys,
    queryFn: fetchSSHKeys
  });
}

export function useGenerateSSHKey(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (request: SSHGenerateKeyRequest) => generateSSHKey(request, repoId ?? undefined),
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

export function useDeleteSSHKey(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (request: SSHDeleteKeyRequest) => deleteSSHKey(request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.sshKeys });
    }
  });
}

// ── Grouping Rules ─────────────────────────────────────────────────────

export function useGroupingRules(repoId?: string | null) {
  return useQuery<GroupingRulesConfig, Error>({
    queryKey: queryKeys.groupingRules(repoId),
    queryFn: () => fetchGroupingRules(repoId ?? undefined),
    enabled: Boolean(repoId),
  });
}

export function useRepoGroups(repoId?: string | null, statusSnapshotId?: string) {
  return useQuery<RepoGroupsResponse, Error>({
    queryKey: [...queryKeys.repoGroups(repoId), statusSnapshotId ?? "pending"],
    queryFn: ({ signal }) => fetchRepoGroups(repoId ?? undefined, statusSnapshotId, signal),
    // Resolve against the exact status snapshot already used by Changes. This
    // avoids a second Git status read and keeps group membership consistent
    // with the file rows while the worktree is changing.
    enabled: Boolean(repoId && statusSnapshotId),
    staleTime: Number.POSITIVE_INFINITY,
    // The status snapshot is authoritative for deciding when groups need a
    // new result. Do not also refetch the old snapshot on focus; that starts a
    // redundant Git status read just before the new snapshot key is selected.
    refetchOnWindowFocus: false,
    // Keep current group rows visible while resolving membership for the new
    // status snapshot. The new result atomically replaces them when ready.
    placeholderData: (previousData, previousQuery) =>
      previousQuery?.queryKey[2] === (repoId ?? "default") ? previousData : undefined,
  });
}

export function useSaveGroupingRules(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (config: GroupingRulesConfig) => saveGroupingRules(config, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.groupingRules(repoId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.repoGroups(repoId) });
    },
  });
}

// ── Gitignore Health ───────────────────────────────────────────────────

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

// ── Tracked Binaries ───────────────────────────────────────────────────

export function useTrackedBinaries(repoId?: string | null) {
  return useQuery<TrackedBinariesResponse, Error>({
    queryKey: queryKeys.trackedBinaries(repoId),
    queryFn: () => fetchTrackedBinaries(repoId ?? undefined),
    enabled: Boolean(repoId),
    // Scanning every tracked file is not free, and committed binaries do not
    // appear mid-session. Refetching on every settings open would be waste.
    staleTime: 5 * 60 * 1000,
  });
}

export function useUntrackBinary(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (request: UntrackBinaryRequest) => untrackBinary(request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.trackedBinaries(repoId) });
      // Untracking stages a deletion, so the working-tree status changes too.
      queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus(repoId) });
    },
  });
}

// ── Health Notification Count ──────────────────────────────────────────

/**
 * Number of health findings that have a remediation the user can act on.
 *
 * Counts ONLY actionable items. Informational findings are excluded on purpose:
 * cross-group .gitignore patterns are permanently non-empty in a real repo, so
 * including them would pin the badge to a constant non-zero value and train the
 * user to ignore it. A badge that is always lit carries no signal.
 */
export function useHealthIssueCount(repoId?: string | null): number {
  const gitignore = useGitignoreHealth(repoId);
  const binaries = useTrackedBinaries(repoId);

  const movable = (gitignore.data?.suggestions ?? []).filter(s => s.type === "single_group").length;
  const trackedBinaries = binaries.data?.binaries?.length ?? 0;
  return movable + trackedBinaries;
}
