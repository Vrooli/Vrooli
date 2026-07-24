// ============================================================================
// Capabilities, Credentials, SSH, Repo Registry, Grouping & Gitignore
// Types + API Functions
// ============================================================================

import { API_BASE, buildRepoHeaders, handleResponse, buildApiUrl } from "./api-internals";
import type { RepoListResponse, RepoActiveResponse, RepoOpenRequest, RepoCloneRequest, RepoActiveRequest, RepoMutationResponse, RepoRemoveResponse } from "./api-types-repo";
import type { GroupingRulesConfig, GitignoreHealthResponse, GitignoreMoveRequest, GitignoreMoveResponse, TrackedBinariesResponse, UntrackBinaryRequest, UntrackBinaryResponse } from "./api-types-operations";

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
  const url = buildApiUrl("/repo/grouping-rules", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "GET",
    headers: buildRepoHeaders(repoId),
    cache: "no-store",
  });
  return handleResponse<GroupingRulesConfig>(res);
}

export async function saveGroupingRules(config: GroupingRulesConfig, repoId?: string): Promise<GroupingRulesConfig> {
  const url = buildApiUrl("/repo/grouping-rules", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "PUT",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(config),
  });
  return handleResponse<GroupingRulesConfig>(res);
}

// ── Gitignore Health API ───────────────────────────────────────────────

export async function fetchGitignoreHealth(repoId?: string): Promise<GitignoreHealthResponse> {
  const url = buildApiUrl("/repo/gitignore/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "GET",
    headers: buildRepoHeaders(repoId),
  });
  return handleResponse<GitignoreHealthResponse>(res);
}

export async function moveGitignoreEntry(request: GitignoreMoveRequest, repoId?: string): Promise<GitignoreMoveResponse> {
  const url = buildApiUrl("/repo/gitignore/move", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request),
  });
  return handleResponse<GitignoreMoveResponse>(res);
}

// ── Tracked Binaries API ───────────────────────────────────────────────

export async function fetchTrackedBinaries(repoId?: string): Promise<TrackedBinariesResponse> {
  const url = buildApiUrl("/repo/tracked-binaries", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "GET",
    headers: buildRepoHeaders(repoId),
  });
  return handleResponse<TrackedBinariesResponse>(res);
}

export async function untrackBinary(request: UntrackBinaryRequest, repoId?: string): Promise<UntrackBinaryResponse> {
  const url = buildApiUrl("/repo/tracked-binaries/untrack", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request),
  });
  return handleResponse<UntrackBinaryResponse>(res);
}

// ── Credentials API ────────────────────────────────────────────────────

export async function fetchCredentials(repoId?: string): Promise<CredentialsListResponse> {
  const url = buildApiUrl("/credentials", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store"
  });
  return handleResponse<CredentialsListResponse>(res);
}

export async function saveCredential(request: CredentialSaveRequest, repoId?: string): Promise<CredentialSaveResponse> {
  const url = buildApiUrl("/credentials", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request)
  });
  return handleResponse<CredentialSaveResponse>(res);
}

export async function deleteCredential(
  id: string,
  repoId?: string
): Promise<CredentialDeleteResponse> {
  const url = buildApiUrl(`/credentials/${encodeURIComponent(id)}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "DELETE",
    headers: buildRepoHeaders(repoId)
  });
  return handleResponse<CredentialDeleteResponse>(res);
}

export async function testCredential(request: CredentialTestRequest, repoId?: string): Promise<CredentialTestResponse> {
  const url = buildApiUrl("/credentials/test", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request)
  });
  return handleResponse<CredentialTestResponse>(res);
}

export async function updateRemoteURL(
  request: RemoteURLUpdateRequest,
  repoId?: string
): Promise<RemoteURLUpdateResponse> {
  const url = buildApiUrl("/repo/remote/url", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request)
  });
  return handleResponse<RemoteURLUpdateResponse>(res);
}

// ── Repo Registry API ──────────────────────────────────────────────────

export async function fetchRepos(): Promise<RepoListResponse> {
  const url = buildApiUrl("/repos", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  return handleResponse<RepoListResponse>(res);
}

export async function fetchActiveRepo(): Promise<RepoActiveResponse> {
  const url = buildApiUrl("/repos/active", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  return handleResponse<RepoActiveResponse>(res);
}

export async function openRepo(request: RepoOpenRequest): Promise<RepoMutationResponse> {
  const url = buildApiUrl("/repos/open", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request)
  });
  return handleResponse<RepoMutationResponse>(res);
}

export async function cloneRepo(request: RepoCloneRequest): Promise<RepoMutationResponse> {
  const url = buildApiUrl("/repos/clone", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request)
  });
  return handleResponse<RepoMutationResponse>(res);
}

export async function setActiveRepo(request: RepoActiveRequest): Promise<RepoMutationResponse> {
  const url = buildApiUrl("/repos/active", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request)
  });
  return handleResponse<RepoMutationResponse>(res);
}

export async function removeRepo(id: number): Promise<RepoRemoveResponse> {
  const url = buildApiUrl(`/repos/${encodeURIComponent(String(id))}`, {
    baseUrl: API_BASE
  });
  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" }
  });
  return handleResponse<RepoRemoveResponse>(res);
}

// ── SSH Key Management API ─────────────────────────────────────────────

export async function fetchSSHKeys(): Promise<SSHListKeysResponse> {
  const url = buildApiUrl("/ssh/keys", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  return handleResponse<SSHListKeysResponse>(res);
}

export async function generateSSHKey(request: SSHGenerateKeyRequest): Promise<SSHGenerateKeyResponse> {
  const url = buildApiUrl("/ssh/keys/generate", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request)
  });
  return handleResponse<SSHGenerateKeyResponse>(res);
}

export async function getSSHPublicKey(request: SSHGetPublicKeyRequest): Promise<SSHGetPublicKeyResponse> {
  const url = buildApiUrl("/ssh/keys/public", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request)
  });
  return handleResponse<SSHGetPublicKeyResponse>(res);
}

export async function testSSHConnection(request: SSHTestConnectionRequest): Promise<SSHTestConnectionResponse> {
  const url = buildApiUrl("/ssh/keys/test", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request)
  });
  return handleResponse<SSHTestConnectionResponse>(res);
}

export async function deleteSSHKey(request: SSHDeleteKeyRequest): Promise<SSHDeleteKeyResponse> {
  const url = buildApiUrl("/ssh/keys", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request)
  });
  return handleResponse<SSHDeleteKeyResponse>(res);
}
