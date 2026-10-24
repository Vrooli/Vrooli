// ============================================================================
// Capabilities, Credentials, SSH, Repo Registry, Grouping & Gitignore
// Types + API Functions
// ============================================================================

import { API_BASE, handleResponse, buildApiUrl } from "./api-internals";
import { create } from "@bufbuild/protobuf";
import {
	CloneRepositoryRequestSchema,
	DeleteCredentialRequestSchema,
	DeleteSSHKeyRequestSchema,
	GenerateSSHKeyRequestSchema,
	GetSSHPublicKeyRequestSchema,
	GetGitignoreHealthRequestSchema,
	GetGroupingRulesRequestSchema,
	GetActiveRepositoryRequestSchema,
	GetRepoGroupsRequestSchema,
	GetTrackedBinariesRequestSchema,
	ListCredentialsRequestSchema,
	ListRepositoriesRequestSchema,
	ListSSHKeysRequestSchema,
	MoveGitignoreEntryRequestSchema,
	OpenRepositoryRequestSchema,
	GroupingRuleSchema,
	RemoveRepositoryRequestSchema,
	SaveCredentialRequestSchema,
	SaveGroupingRulesRequestSchema,
	SetActiveRepositoryRequestSchema,
	TestCredentialRequestSchema,
	TestSSHConnectionRequestSchema,
	UntrackBinaryRequestSchema,
	UpdateRemoteURLRequestSchema,
} from "@vrooli/proto-types/git-control-tower/v1/repo/repo_pb";
import { repoClient } from "./connect";
import { issueMutationIntentForOperation } from "./api-core";
import type { RepoListResponse, RepoActiveResponse, RepoRecord, RepoOpenRequest, RepoCloneRequest, RepoActiveRequest, RepoMutationResponse, RepoRemoveResponse } from "./api-types-repo";
import type { GroupingRulesConfig, RepoGroupsResponse, GitignoreHealthResponse, GitignoreMoveRequest, GitignoreMoveResponse, TrackedBinariesResponse, UntrackBinaryRequest, UntrackBinaryResponse } from "./api-types-operations";

// ── Capabilities Types ─────────────────────────────────────────────────

export type DependencyKind = "scenario" | "resource";
export type CapabilityStatus = "available" | "unavailable" | "unknown";

export interface CapabilityState {
  id: string;
  name: string;
  description: string;
  dependencyKind: DependencyKind;
  dependencySlug: string;
  features: string[];
  status: CapabilityStatus;
  message?: string;
  checkedAt?: string;
}

export interface CapabilitiesResponse {
  capabilities: CapabilityState[];
  timestamp: string;
}

// ── Credentials Types ──────────────────────────────────────────────────

export type CredentialType = "https" | "ssh";

export interface Credential {
  id: string;
  remote: string;
  url: string;
  type: CredentialType;
  username?: string;
  token_masked?: string;
  ssh_key_path?: string;
  is_configured: boolean;
  created_at: string;
  updated_at: string;
}

export interface CredentialsListResponse {
  credentials: Credential[];
  timestamp: string;
}

export interface CredentialSaveRequest {
  remote: string;
  url?: string;
  username?: string;
  token?: string;
  ssh_key_path?: string;
}

export interface CredentialSaveResponse {
  success: boolean;
  credential?: Credential;
  error?: string;
  timestamp: string;
}

export interface CredentialDeleteResponse {
  success: boolean;
  error?: string;
  timestamp: string;
}

export interface CredentialTestRequest {
  remote: string;
  use_stored?: boolean;
}

export interface CredentialTestResponse {
  success: boolean;
  reachable: boolean;
  authorized: boolean;
  error?: string;
  timestamp: string;
}

export interface RemoteURLUpdateRequest {
  remote: string;
  url: string;
}

export interface RemoteURLUpdateResponse {
  success: boolean;
  old_url?: string;
  new_url?: string;
  error?: string;
  timestamp: string;
}

// ── SSH Key Management Types ───────────────────────────────────────────

export type SSHKeyType = "ed25519" | "rsa" | "ecdsa" | "dsa" | "unknown";

export interface SSHKeyInfo {
  path: string;
  filename: string;
  type: SSHKeyType;
  bits?: number;
  fingerprint: string;
  comment?: string;
  created_at?: string;
  has_public: boolean;
}

export interface SSHListKeysResponse {
  keys: SSHKeyInfo[];
  ssh_dir: string;
  timestamp: string;
}

export interface SSHGenerateKeyRequest {
  type: "ed25519" | "rsa";
  bits?: number;
  comment?: string;
  filename?: string;
}

export interface SSHGenerateKeyResponse {
  success: boolean;
  key?: SSHKeyInfo;
  public_key?: string;
  error?: string;
  timestamp: string;
}

export interface SSHGetPublicKeyRequest {
  key_path: string;
}

export interface SSHGetPublicKeyResponse {
  success: boolean;
  public_key?: string;
  fingerprint?: string;
  error?: string;
  timestamp: string;
}

export interface SSHTestConnectionRequest {
  key_path: string;
}

export interface SSHTestConnectionResponse {
  success: boolean;
  status: string;
  message?: string;
  hint?: string;
  github_user?: string;
  fingerprint?: string;
  latency_ms?: number;
  timestamp: string;
}

export interface SSHDeleteKeyRequest {
  key_path: string;
}

export interface SSHDeleteKeyResponse {
  success: boolean;
  message?: string;
  error?: string;
  private_deleted: boolean;
  public_deleted: boolean;
  timestamp: string;
}

// ── Capabilities API ───────────────────────────────────────────────────

export async function fetchCapabilities(): Promise<CapabilitiesResponse> {
  const url = buildApiUrl("/capabilities", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  return handleResponse<CapabilitiesResponse>(res);
}

// ── Grouping Rules API ─────────────────────────────────────────────────

export async function fetchGroupingRules(repoId?: string): Promise<GroupingRulesConfig> {
  const response = await repoClient.getGroupingRules(create(GetGroupingRulesRequestSchema, {
    repositoryId: repoId ?? "",
  }));
  return {
    enabled: response.enabled,
    rules: response.rules.map((rule) => ({ id: rule.id, label: rule.label, prefixes: rule.prefixes, mode: rule.mode })),
  };
}

export async function saveGroupingRules(config: GroupingRulesConfig, repoId?: string): Promise<GroupingRulesConfig> {
  const intent = await issueMutationIntentForOperation("repo.grouping-rules", repoId);
  const response = await repoClient.saveGroupingRules(create(SaveGroupingRulesRequestSchema, {
    repositoryId: intent.repositoryId,
    intentId: intent.intentId,
    enabled: config.enabled,
    rules: config.rules.map((rule) => create(GroupingRuleSchema, {
      id: rule.id, label: rule.label, prefixes: rule.prefixes, mode: rule.mode,
    })),
  }));
  return {
    enabled: response.enabled,
    rules: response.rules.map((rule) => ({ id: rule.id, label: rule.label, prefixes: rule.prefixes, mode: rule.mode })),
  };
}

export async function fetchRepoGroups(repoId?: string): Promise<RepoGroupsResponse> {
  const response = await repoClient.getRepoGroups(create(GetRepoGroupsRequestSchema, {
    repositoryId: repoId ?? "",
  }));
  return {
    groups: response.groups.map((group) => ({
      key: group.key,
      kind: group.kind,
      id: group.id,
      label: group.label,
      root: group.root,
      source: group.source,
      files: group.files,
    })),
  };
}

// ── Gitignore Health API ───────────────────────────────────────────────

export async function fetchGitignoreHealth(repoId?: string): Promise<GitignoreHealthResponse> {
  const response = await repoClient.getGitignoreHealth(create(GetGitignoreHealthRequestSchema, {
    repositoryId: repoId ?? "",
  }));
  return {
    root_entry_count: response.rootEntryCount,
    suggestions: response.suggestions.map((suggestion) => ({
      line: suggestion.line,
      pattern: suggestion.pattern,
      type: suggestion.type as GitignoreHealthResponse["suggestions"][number]["type"],
      group_label: suggestion.groupLabel,
      group_dir: suggestion.groupDir,
      target_pattern: suggestion.targetPattern,
      has_gitignore: suggestion.hasGitignore,
    })),
  };
}

export async function moveGitignoreEntry(request: GitignoreMoveRequest, repoId?: string): Promise<GitignoreMoveResponse> {
  const intent = await issueMutationIntentForOperation("repo.gitignore.move", repoId);
  const response = await repoClient.moveGitignoreEntry(create(MoveGitignoreEntryRequestSchema, {
    repositoryId: intent.repositoryId,
    intentId: intent.intentId,
    line: request.line,
    pattern: request.pattern,
    groupDir: request.group_dir,
    targetPattern: request.target_pattern,
  }));
  return {
    success: response.success,
    removed_from: response.removedFrom || undefined,
    added_to: response.addedTo || undefined,
    error: response.error || undefined,
  };
}

// ── Tracked Binaries API ───────────────────────────────────────────────

export async function fetchTrackedBinaries(repoId?: string): Promise<TrackedBinariesResponse> {
  const response = await repoClient.getTrackedBinaries(create(GetTrackedBinariesRequestSchema, {
    repositoryId: repoId ?? "",
  }));
  return {
    binaries: response.binaries.map((binary) => ({
      path: binary.path,
      bytes: Number(binary.bytes),
      format: binary.format as TrackedBinariesResponse["binaries"][number]["format"],
      owner_dir: binary.ownerDir,
      ignore_pattern: binary.ignorePattern,
      already_ignored: binary.alreadyIgnored,
    })),
    total_bytes: Number(response.totalBytes),
    history_warning: response.historyWarning || undefined,
  };
}

export async function untrackBinary(request: UntrackBinaryRequest, repoId?: string): Promise<UntrackBinaryResponse> {
  const intent = await issueMutationIntentForOperation("repo.tracked-binaries.untrack", repoId);
  const response = await repoClient.untrackBinary(create(UntrackBinaryRequestSchema, {
    repositoryId: intent.repositoryId,
    intentId: intent.intentId,
    path: request.path,
    ownerDir: request.owner_dir,
    ignorePattern: request.ignore_pattern,
  }));
  return {
    success: response.success,
    removed_from_index: response.removedFromIndex,
    ignore_added_to: response.ignoreAddedTo || undefined,
    error: response.error || undefined,
  };
}

// ── Credentials API ────────────────────────────────────────────────────

export async function fetchCredentials(repoId?: string): Promise<CredentialsListResponse> {
  const response = await repoClient.listCredentials(create(ListCredentialsRequestSchema, { repositoryId: repoId ?? "" }));
  return {
    credentials: response.credentials.map((credential) => ({
      id: credential.id,
      remote: credential.remote,
      url: credential.url,
      type: credential.type as CredentialType,
      username: credential.username || undefined,
      token_masked: credential.tokenMasked || undefined,
      ssh_key_path: credential.sshKeyPath || undefined,
      is_configured: credential.isConfigured,
      created_at: credential.createdAt,
      updated_at: credential.updatedAt,
    })),
    timestamp: response.timestamp,
  };
}

export async function saveCredential(request: CredentialSaveRequest, repoId?: string): Promise<CredentialSaveResponse> {
  const intent = await issueMutationIntentForOperation("repo.credentials", repoId);
  const response = await repoClient.saveCredential(create(SaveCredentialRequestSchema, {
    repositoryId: intent.repositoryId,
    intentId: intent.intentId,
    remote: request.remote,
    url: request.url ?? "",
    username: request.username ?? "",
    token: request.token ?? "",
    sshKeyPath: request.ssh_key_path ?? "",
  }));
  return {
    success: response.success,
    credential: response.credential ? {
      id: response.credential.id,
      remote: response.credential.remote,
      url: response.credential.url,
      type: response.credential.type as CredentialType,
      username: response.credential.username || undefined,
      token_masked: response.credential.tokenMasked || undefined,
      ssh_key_path: response.credential.sshKeyPath || undefined,
      is_configured: response.credential.isConfigured,
      created_at: response.credential.createdAt,
      updated_at: response.credential.updatedAt,
    } : undefined,
    error: response.error || undefined,
    timestamp: response.timestamp,
  };
}

export async function deleteCredential(
  id: string,
  repoId?: string
): Promise<CredentialDeleteResponse> {
  const intent = await issueMutationIntentForOperation("repo.credentials", repoId);
  const response = await repoClient.deleteCredential(create(DeleteCredentialRequestSchema, {
    repositoryId: intent.repositoryId,
    intentId: intent.intentId,
    id,
  }));
  return { success: response.success, error: response.error || undefined, timestamp: response.timestamp };
}

export async function testCredential(request: CredentialTestRequest, repoId?: string): Promise<CredentialTestResponse> {
  const response = await repoClient.testCredential(create(TestCredentialRequestSchema, {
    repositoryId: repoId ?? "",
    remote: request.remote,
    useStored: request.use_stored ?? false,
  }));
  return { success: response.success, reachable: response.reachable, authorized: response.authorized, error: response.error || undefined, timestamp: response.timestamp };
}

export async function updateRemoteURL(
  request: RemoteURLUpdateRequest,
  repoId?: string
): Promise<RemoteURLUpdateResponse> {
  const intent = await issueMutationIntentForOperation("repo.remote.url", repoId);
  const response = await repoClient.updateRemoteURL(create(UpdateRemoteURLRequestSchema, {
    repositoryId: intent.repositoryId,
    intentId: intent.intentId,
    remote: request.remote,
    url: request.url,
  }));
  return { success: response.success, old_url: response.oldUrl || undefined, new_url: response.newUrl || undefined, error: response.error || undefined, timestamp: response.timestamp };
}

// ── Repo Registry API ──────────────────────────────────────────────────

export async function fetchRepos(): Promise<RepoListResponse> {
  const response = await repoClient.listRepositories(create(ListRepositoriesRequestSchema, {}));
  return {
    repos: response.repos.map(repoRecordFromProto),
    active_id: Number(response.activeId) || undefined,
    timestamp: response.timestamp,
  };
}

export async function fetchActiveRepo(): Promise<RepoActiveResponse> {
  const response = await repoClient.getActiveRepository(create(GetActiveRepositoryRequestSchema, {}));
  return { repo: response.repo ? repoRecordFromProto(response.repo) : undefined, timestamp: response.timestamp };
}

export async function openRepo(request: RepoOpenRequest): Promise<RepoMutationResponse> {
  const response = await repoClient.openRepository(create(OpenRepositoryRequestSchema, { path: request.path }));
  return { repo: response.repo ? repoRecordFromProto(response.repo) : undefined, timestamp: response.timestamp };
}

export async function cloneRepo(request: RepoCloneRequest): Promise<RepoMutationResponse> {
  const response = await repoClient.cloneRepository(create(CloneRepositoryRequestSchema, { url: request.url, destination: request.destination }));
  return { repo: response.repo ? repoRecordFromProto(response.repo) : undefined, timestamp: response.timestamp };
}

export async function setActiveRepo(request: RepoActiveRequest): Promise<RepoMutationResponse> {
  const response = await repoClient.setActiveRepository(create(SetActiveRepositoryRequestSchema, { repositoryId: BigInt(request.id) }));
  return { repo: response.repo ? repoRecordFromProto(response.repo) : undefined, timestamp: response.timestamp };
}

export async function removeRepo(id: number): Promise<RepoRemoveResponse> {
  const response = await repoClient.removeRepository(create(RemoveRepositoryRequestSchema, { repositoryId: BigInt(id) }));
  return { removed: response.removed, timestamp: response.timestamp };
}

function repoRecordFromProto(repo: {
  id: bigint | number;
  path: string;
  name: string;
  remoteUrl: string;
  addedAt: string;
  lastOpenedAt: string;
  favorite: boolean;
}): RepoRecord {
  return {
    id: Number(repo.id), path: repo.path, name: repo.name, remote_url: repo.remoteUrl,
    added_at: repo.addedAt, last_opened_at: repo.lastOpenedAt || undefined, favorite: repo.favorite,
  };
}

// ── SSH Key Management API ─────────────────────────────────────────────

export async function fetchSSHKeys(): Promise<SSHListKeysResponse> {
  const response = await repoClient.listSSHKeys(create(ListSSHKeysRequestSchema, {}));
  return {
    keys: response.keys.map((key) => ({
      path: key.path, filename: key.filename, type: key.type as SSHKeyType, bits: key.bits || undefined,
      fingerprint: key.fingerprint, comment: key.comment || undefined, created_at: key.createdAt || undefined, has_public: key.hasPublic,
    })),
    ssh_dir: response.sshDir,
    timestamp: response.timestamp,
  };
}

export async function generateSSHKey(request: SSHGenerateKeyRequest, repoId?: string): Promise<SSHGenerateKeyResponse> {
  const intent = await issueMutationIntentForOperation("ssh.key.generate", repoId);
  const response = await repoClient.generateSSHKey(create(GenerateSSHKeyRequestSchema, {
    repositoryId: intent.repositoryId, intentId: intent.intentId, type: request.type,
    bits: request.bits ?? 0, comment: request.comment ?? "", filename: request.filename ?? "",
  }));
  return { success: response.success, key: response.key ? { path: response.key.path, filename: response.key.filename, type: response.key.type as SSHKeyType, bits: response.key.bits || undefined, fingerprint: response.key.fingerprint, comment: response.key.comment || undefined, created_at: response.key.createdAt || undefined, has_public: response.key.hasPublic } : undefined, public_key: response.publicKey || undefined, error: response.error || undefined, timestamp: response.timestamp };
}

export async function getSSHPublicKey(request: SSHGetPublicKeyRequest): Promise<SSHGetPublicKeyResponse> {
  const response = await repoClient.getSSHPublicKey(create(GetSSHPublicKeyRequestSchema, { keyPath: request.key_path }));
  return { success: response.success, public_key: response.publicKey || undefined, fingerprint: response.fingerprint || undefined, error: response.error || undefined, timestamp: response.timestamp };
}

export async function testSSHConnection(request: SSHTestConnectionRequest): Promise<SSHTestConnectionResponse> {
  const response = await repoClient.testSSHConnection(create(TestSSHConnectionRequestSchema, { keyPath: request.key_path }));
  return { success: response.success, status: response.status, message: response.message || undefined, hint: response.hint || undefined, github_user: response.githubUser || undefined, fingerprint: response.fingerprint || undefined, latency_ms: Number(response.latencyMs) || undefined, timestamp: response.timestamp };
}

export async function deleteSSHKey(request: SSHDeleteKeyRequest, repoId?: string): Promise<SSHDeleteKeyResponse> {
  const intent = await issueMutationIntentForOperation("ssh.key.delete", repoId);
  const response = await repoClient.deleteSSHKey(create(DeleteSSHKeyRequestSchema, { repositoryId: intent.repositoryId, intentId: intent.intentId, keyPath: request.key_path }));
  return { success: response.success, message: response.message || undefined, error: response.error || undefined, private_deleted: response.privateDeleted, public_deleted: response.publicDeleted, timestamp: response.timestamp };
}
