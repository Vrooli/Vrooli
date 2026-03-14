import { useCallback, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import type { CapturePreset, DiffStats } from "./api";
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
  triggerVisualCapture,
  fetchVisualCaptures,
  fetchVisualCaptureDetail,
  fetchCaptureStorageStats,
  deleteVisualCapture,
  clearAllCaptureStorage,
  triggerWorkflowCapture,
  fetchWorkflowCaptures,
  triggerTestExecution,
  fetchTestExecutions,
  fetchTestExecution,
  fetchTidinessScore,
  fetchTidinessIssues,
  fetchTidinessStaleness,
  triggerTidinessLightScan,
  fetchTidinessScenarioDetail,
  fetchScenarios,
  fetchAgentProfiles,
  createAgentRun,
  fetchAgentRuns,
  fetchAgentRun,
  fetchAgentRunEvents,
  fetchAgentRunDiff,
  continueAgentRun,
  approveAgentRun,
  rejectAgentRun,
  stopAgentRun,
  startAuditorCheck,
  pollAuditorJob,
  fetchAuditorRules,
  applyAuditorFix,
  fetchAuditorViolations,
  type AuditorCheckJobResponse,
  type AuditorJobStatus,
  type AuditorRulesListResponse,
  type AuditorFixRequest,
  type AuditorFixResponse,
  type AuditorViolation,
  type CapabilitiesResponse,
  type TestExecutionRequest,
  type TestExecutionResult,
  type TestExecutionListResponse,
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
  type SSHDeleteKeyRequest,
  type VisualCaptureListResponse,
  type SnapshotSetMeta,
  type SnapshotSetDetail,
  type CaptureStorageStats,
  type WorkflowCaptureResult,
  type WorkflowCaptureListResponse,
  type ExecutionMode,
  type TidinessScoreResponse,
  type TidinessIssue,
  type TidinessStalenessInfo,
  type TidinessLightScanResult,
  type TidinessScenarioDetail,
  type AgentProfileListResponse,
  type AgentRunRequest,
  type AgentRunCreateResponse,
  type AgentRun,
  type AgentRunListResponse,
  type AgentRunEventsResponse,
  type AgentRunDiffResponse,
  type AgentContinueRequest,
  type AgentContinueResponse,
  type AgentApproveRequest,
  type AgentApproveResponse,
  type AgentRejectRequest,
  type AgentRejectResponse,
  type AgentStopResponse,
  type ScenarioInfo,
  ACTIVE_STATUSES,
  RUN_STATUS,
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
  activeRepo: ["repos", "active"] as const,
  visualCaptures: (slug: string, repoId?: string | null) =>
    ["repo", "visual-captures", repoId ?? "default", slug] as const,
  visualCaptureDetail: (id: string, slug: string, repoId?: string | null) =>
    ["repo", "visual-captures", repoId ?? "default", "detail", id, slug] as const,
  captureStorage: (repoId?: string | null) =>
    ["repo", "visual-capture-storage", repoId ?? "default"] as const,
  workflowCaptures: (slug: string, repoId?: string | null) =>
    ["repo", "workflow-captures", repoId ?? "default", slug] as const,
  testExecutions: (scenarioName: string, repoId?: string | null) =>
    ["repo", "test-executions", repoId ?? "default", scenarioName] as const,
  testExecution: (id: string, repoId?: string | null) =>
    ["repo", "test-executions", repoId ?? "default", "detail", id] as const,
  tidinessScore: (scenarioName: string, repoId?: string | null) =>
    ["repo", "tidiness-score", repoId ?? "default", scenarioName] as const,
  tidinessIssues: (scenarioName: string, file?: string, repoId?: string | null) =>
    ["repo", "tidiness-issues", repoId ?? "default", scenarioName, file] as const,
  tidinessStaleness: (scenarioName: string, repoId?: string | null) =>
    ["repo", "tidiness-staleness", repoId ?? "default", scenarioName] as const,
  tidinessScenarioDetail: (scenarioName: string, repoId?: string | null) =>
    ["repo", "tidiness-scenario", repoId ?? "default", scenarioName] as const,
  scenarios: ["scenarios"] as const,
  agentProfiles: ["agent", "profiles"] as const,
  agentRuns: (slug: string, repoId?: string | null) =>
    ["agent", "runs", repoId ?? "default", slug] as const,
  agentRun: (runId: string, repoId?: string | null) =>
    ["agent", "runs", repoId ?? "default", "detail", runId] as const,
  agentRunEvents: (runId: string, repoId?: string | null) =>
    ["agent", "runs", repoId ?? "default", "events", runId] as const,
  agentRunDiff: (runId: string, repoId?: string | null) =>
    ["agent", "runs", repoId ?? "default", "diff", runId] as const,
  rulesRun: (scenarioName: string, repoId?: string | null) =>
    ["repo", "rules-run", repoId ?? "default", scenarioName] as const,
  rulesJob: (jobId: string, repoId?: string | null) =>
    ["repo", "rules-job", repoId ?? "default", jobId] as const,
  rulesList: (repoId?: string | null) =>
    ["repo", "rules-list", repoId ?? "default"] as const,
  rulesViolations: (scenarioName: string, repoId?: string | null) =>
    ["repo", "rules-violations", repoId ?? "default", scenarioName] as const,
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

// ============================================================================
// Visual Capture Hooks
// ============================================================================

export function useVisualCaptures(slug: string, enabled = true, repoId?: string | null) {
  return useQuery<VisualCaptureListResponse, Error>({
    queryKey: queryKeys.visualCaptures(slug, repoId),
    queryFn: () => fetchVisualCaptures(slug, repoId ?? undefined),
    enabled: enabled && Boolean(slug),
    refetchInterval: 10_000,
  });
}

export function useVisualCaptureDetail(id: string, slug: string, enabled = true, repoId?: string | null) {
  return useQuery<SnapshotSetDetail, Error>({
    queryKey: queryKeys.visualCaptureDetail(id, slug, repoId),
    queryFn: () => fetchVisualCaptureDetail(id, slug, repoId ?? undefined),
    enabled: enabled && Boolean(id) && Boolean(slug),
    staleTime: 60_000,
  });
}

export function useTriggerVisualCapture(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<SnapshotSetMeta, Error, { scenarioSlug: string; mode: "baseline" | "capture"; presets: CapturePreset[] }>({
    mutationFn: async ({ scenarioSlug, mode, presets }) => {
      const meta = await triggerVisualCapture(scenarioSlug, mode, repoId ?? undefined, presets);
      if (meta.status === "failed") {
        throw new Error(meta.error || "Capture failed — no screenshots were captured");
      }
      return meta;
    },
    onSuccess: (_data, { scenarioSlug }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.visualCaptures(scenarioSlug, repoId) });
    },
  });
}

export function useCaptureStorageStats(repoId?: string | null) {
  return useQuery<CaptureStorageStats, Error>({
    queryKey: queryKeys.captureStorage(repoId),
    queryFn: () => fetchCaptureStorageStats(repoId ?? undefined),
    staleTime: 30_000,
  });
}

export function useDeleteVisualCapture(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<void, Error, { id: string; scenarioSlug: string }>({
    mutationFn: ({ id, scenarioSlug }) => deleteVisualCapture(id, scenarioSlug, repoId ?? undefined),
    onSuccess: (_data, { scenarioSlug }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.captureStorage(repoId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.visualCaptures(scenarioSlug, repoId) });
    },
  });
}

export function useClearAllCaptureStorage(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<void, Error, void>({
    mutationFn: () => clearAllCaptureStorage(repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.captureStorage(repoId) });
    },
  });
}

// ============================================================================
// Workflow Capture Hooks
// ============================================================================

export function useWorkflowCaptures(slug: string, enabled = true, repoId?: string | null) {
  return useQuery<WorkflowCaptureListResponse, Error>({
    queryKey: queryKeys.workflowCaptures(slug, repoId),
    queryFn: () => fetchWorkflowCaptures(slug, repoId ?? undefined),
    enabled: enabled && Boolean(slug),
    refetchInterval: 10_000,
  });
}

export function useTriggerWorkflowCapture(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<WorkflowCaptureResult, Error, { scenarioSlug: string; mode: "baseline" | "capture"; executionModes: ExecutionMode[] }>({
    mutationFn: async ({ scenarioSlug, mode, executionModes }) => {
      const result = await triggerWorkflowCapture(scenarioSlug, mode, executionModes, repoId ?? undefined);
      if (result.status === "failed") {
        throw new Error(result.error || "Workflow capture failed");
      }
      return result;
    },
    onSuccess: (_data, { scenarioSlug }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.workflowCaptures(scenarioSlug, repoId) });
    },
  });
}

// ============================================================================
// Test Execution Hooks
// ============================================================================

export function useTestExecutions(scenarioName: string, enabled = true, repoId?: string | null) {
  return useQuery<TestExecutionListResponse, Error>({
    queryKey: queryKeys.testExecutions(scenarioName, repoId),
    queryFn: () => fetchTestExecutions(scenarioName, 10, repoId ?? undefined),
    enabled: enabled && Boolean(scenarioName),
    refetchInterval: 15_000,
  });
}

export function useTestExecution(id: string, enabled = true, repoId?: string | null) {
  return useQuery<TestExecutionResult, Error>({
    queryKey: queryKeys.testExecution(id, repoId),
    queryFn: () => fetchTestExecution(id, repoId ?? undefined),
    enabled: enabled && Boolean(id),
    staleTime: 30_000,
  });
}

export function useTriggerTestExecution(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<TestExecutionResult, Error, TestExecutionRequest>({
    mutationFn: (request: TestExecutionRequest) =>
      triggerTestExecution(request, repoId ?? undefined),
    onSuccess: (_data, request) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.testExecutions(request.scenarioName, repoId),
      });
    },
  });
}

// ============================================================================
// Tidiness Manager Hooks
// ============================================================================

export function useTidinessScore(scenarioName: string, enabled = true, repoId?: string | null) {
  return useQuery<TidinessScoreResponse, Error>({
    queryKey: queryKeys.tidinessScore(scenarioName, repoId),
    queryFn: () => fetchTidinessScore(scenarioName, repoId ?? undefined),
    enabled: enabled && Boolean(scenarioName),
    refetchInterval: 30_000,
  });
}

export function useTidinessIssues(
  scenarioName: string,
  file?: string,
  enabled = true,
  repoId?: string | null
) {
  return useQuery<TidinessIssue[], Error>({
    queryKey: queryKeys.tidinessIssues(scenarioName, file, repoId),
    queryFn: () => fetchTidinessIssues(scenarioName, file, undefined, undefined, repoId ?? undefined),
    enabled: enabled && Boolean(scenarioName),
    staleTime: 30_000,
  });
}

export function useTidinessStaleness(scenarioName: string, enabled = true, repoId?: string | null) {
  return useQuery<TidinessStalenessInfo, Error>({
    queryKey: queryKeys.tidinessStaleness(scenarioName, repoId),
    queryFn: () => fetchTidinessStaleness(scenarioName, repoId ?? undefined),
    enabled: enabled && Boolean(scenarioName),
    staleTime: 60_000,
  });
}

export function useTriggerTidinessScan(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<TidinessLightScanResult, Error, { scenarioName: string; incremental?: boolean }>({
    mutationFn: ({ scenarioName, incremental }) =>
      triggerTidinessLightScan({ scenario_name: scenarioName, incremental }, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["repo", "tidiness-score"] });
      queryClient.invalidateQueries({ queryKey: ["repo", "tidiness-issues"] });
      queryClient.invalidateQueries({ queryKey: ["repo", "tidiness-staleness"] });
      queryClient.invalidateQueries({ queryKey: ["repo", "tidiness-scenario"] });
    },
  });
}

export function useTidinessScenarioDetail(scenarioName: string, enabled = true, repoId?: string | null) {
  return useQuery<TidinessScenarioDetail, Error>({
    queryKey: queryKeys.tidinessScenarioDetail(scenarioName, repoId),
    queryFn: () => fetchTidinessScenarioDetail(scenarioName, repoId ?? undefined),
    enabled: enabled && Boolean(scenarioName),
    refetchInterval: 30_000,
  });
}

// ── Agent Manager hooks ──────────────────────────────────────────────

function agentPollingInterval(status?: string): number | false {
  if (!status) return false;
  if ((ACTIVE_STATUSES as readonly string[]).includes(status)) return 2_000;
  if (status === RUN_STATUS.NEEDS_REVIEW) return 5_000;
  return false; // terminal states: complete, failed, cancelled
}

export function useScenarios(enabled = true) {
  return useQuery<ScenarioInfo[], Error>({
    queryKey: queryKeys.scenarios,
    queryFn: fetchScenarios,
    enabled,
    staleTime: 30_000,
  });
}

export function useAgentProfiles(enabled = true) {
  return useQuery<AgentProfileListResponse, Error>({
    queryKey: queryKeys.agentProfiles,
    queryFn: () => fetchAgentProfiles(),
    enabled,
    staleTime: 60_000,
  });
}

export function useAgentRuns(slug: string, enabled = true, repoId?: string | null) {
  return useQuery<AgentRunListResponse, Error>({
    queryKey: queryKeys.agentRuns(slug, repoId),
    queryFn: () => fetchAgentRuns(slug, 5, repoId ?? undefined),
    enabled: enabled && Boolean(slug),
    refetchInterval: 15_000,
  });
}

export function useAgentRun(runId: string | null, enabled = true, repoId?: string | null) {
  return useQuery<AgentRun, Error>({
    queryKey: queryKeys.agentRun(runId ?? "", repoId),
    queryFn: () => fetchAgentRun(runId as string, repoId ?? undefined),
    enabled: enabled && Boolean(runId),
    refetchInterval: (query) => agentPollingInterval(query.state.data?.status),
  });
}

export function useAgentRunEvents(
  runId: string | null,
  afterSequence: number,
  enabled = true,
  repoId?: string | null
) {
  return useQuery<AgentRunEventsResponse, Error>({
    queryKey: [...queryKeys.agentRunEvents(runId ?? "", repoId), afterSequence],
    queryFn: () => fetchAgentRunEvents(runId as string, afterSequence, repoId ?? undefined),
    enabled: enabled && Boolean(runId),
    refetchInterval: 2_000,
  });
}

export function useAgentRunDiff(runId: string | null, enabled = true, repoId?: string | null) {
  return useQuery<AgentRunDiffResponse, Error>({
    queryKey: queryKeys.agentRunDiff(runId ?? "", repoId),
    queryFn: () => fetchAgentRunDiff(runId as string, repoId ?? undefined),
    enabled: enabled && Boolean(runId),
    staleTime: 10_000,
  });
}

export function useCreateAgentRun(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<AgentRunCreateResponse, Error, AgentRunRequest>({
    mutationFn: (request) => createAgentRun(request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["agent", "runs"] });
    },
  });
}

export function useContinueAgentRun(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<AgentContinueResponse, Error, { runId: string; request: AgentContinueRequest }>({
    mutationFn: ({ runId, request }) => continueAgentRun(runId, request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["agent", "runs"] });
    },
  });
}

export function useApproveAgentRun(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<AgentApproveResponse, Error, { runId: string; request: AgentApproveRequest }>({
    mutationFn: ({ runId, request }) => approveAgentRun(runId, request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["agent", "runs"] });
      queryClient.invalidateQueries({ queryKey: ["repo", "status"] });
    },
  });
}

export function useRejectAgentRun(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<AgentRejectResponse, Error, { runId: string; request: AgentRejectRequest }>({
    mutationFn: ({ runId, request }) => rejectAgentRun(runId, request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["agent", "runs"] });
    },
  });
}

export function useStopAgentRun(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<AgentStopResponse, Error, string>({
    mutationFn: (runId) => stopAgentRun(runId, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["agent", "runs"] });
    },
  });
}

// ============================================================================
// Auditor Hooks
// ============================================================================

export function useAuditorRules(enabled = true, repoId?: string | null) {
  return useQuery<AuditorRulesListResponse, Error>({
    queryKey: queryKeys.rulesList(repoId),
    queryFn: () => fetchAuditorRules(repoId ?? undefined),
    enabled,
    staleTime: 60_000,
  });
}

export function useStartAuditorCheck(repoId?: string | null) {
  return useMutation<AuditorCheckJobResponse, Error, { scenarioName: string; checkType?: string }>({
    mutationFn: ({ scenarioName, checkType }) =>
      startAuditorCheck(scenarioName, checkType, repoId ?? undefined),
  });
}

export function useAuditorJobStatus(jobId: string | null, repoId?: string | null) {
  return useQuery<AuditorJobStatus, Error>({
    queryKey: queryKeys.rulesJob(jobId ?? "", repoId),
    queryFn: () => pollAuditorJob(jobId!, repoId ?? undefined),
    enabled: Boolean(jobId),
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      if (!status || status === "completed" || status === "failed") return false;
      return 2_000;
    },
  });
}

export function useApplyAuditorFix(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<AuditorFixResponse, Error, AuditorFixRequest>({
    mutationFn: (request) => applyAuditorFix(request, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["repo", "rules-run"] });
      queryClient.invalidateQueries({ queryKey: ["repo", "rules-violations"] });
    },
  });
}

export function useAuditorViolations(scenarioName: string, enabled = true, repoId?: string | null) {
  return useQuery<AuditorViolation[], Error>({
    queryKey: queryKeys.rulesViolations(scenarioName, repoId),
    queryFn: () => fetchAuditorViolations(scenarioName, repoId ?? undefined),
    enabled: enabled && Boolean(scenarioName),
    staleTime: 30_000,
  });
}
