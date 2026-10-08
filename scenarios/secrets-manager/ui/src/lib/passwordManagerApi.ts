import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";

export type VaultItem = {
  id: string;
  vault_id: string;
  type: string;
  name: string;
  username?: string;
  uri?: string;
  folder?: string;
  tags: string[];
  favorite: boolean;
  trashed: boolean;
  revision: number;
  created_at: string;
  updated_at: string;
};

export type ItemHistoryRecord = {
  id: string;
  item_id: string;
  version: number;
  changed_by: string;
  changed_at: string;
};

export type VaultStatus = {
  vault_id: string;
  workspace_id: string;
  status: "locked" | "unlocked";
  key_available: boolean;
  supports_recovery: boolean;
};

export type RecoveryStatus = {
  key_available: boolean;
  backup_status: string;
  restore_status: string;
  recovery_epoch: number;
  activation_status?: string;
  stale_capabilities?: string;
  evidence_owner: string;
  recovery_ready: boolean;
  remediation?: string;
  evidence_updated?: string;
  evidence?: Array<{
    kind: string;
    artifact_identity: string;
    source_generation?: string;
    checksum?: string;
    observed_at: string;
    verified: boolean;
    remediation?: string;
  }>;
};

export type ImportPreview = {
  version: number;
  format: string;
  item_count: number;
  duplicate_candidates: string[];
  unsupported_items: string[];
  requires_duplicate_policy: boolean;
};

export type EnrollmentStatus = {
  workspace_id: string;
  bootstrap_configured: boolean;
  enrolled: boolean;
  role_model: string[];
};

export type Grant = {
  id: string;
  vault_id: string;
  item_id?: string;
  principal_type: string;
  principal_id: string;
  selector_mode: "current_snapshot" | "dynamic";
  members: string[];
  operations: string[];
  target?: string;
  expires_at: string;
  status: string;
};

export type GrantAccessExplanation = {
  grant_id: string;
  workspace_id: string;
  actor: string;
  item_id: string;
  operation: string;
  decision: "allow" | "deny";
  reason: string;
  selector_mode: Grant["selector_mode"];
  future_members: boolean;
  raw_read_permitted: boolean;
};

export type AccessRequest = {
  id: string;
  grant_id: string;
  item_id: string;
  operation: string;
  request_digest: string;
  status: string;
  requested_by: string;
  expires_at: string;
  created_at: string;
  decided_by?: string;
  decision_reason?: string;
  decided_at?: string;
};

export type AuditEvent = {
  id: string;
  workspace_id?: string;
  actor_id: string;
  action: string;
  item_id?: string;
  request_id?: string;
  correlation_id?: string;
  event_key?: string;
  outcome: string;
  decision?: string;
  destination?: string;
  detail?: string;
  integrity_status?: "verified" | "unknown" | "tampered";
  created_at: string;
};

export type ExportHandle = {
  download_handle: string;
  download_token: string;
  expires_at: string;
  format: "native" | "plaintext";
  secret_bearing: boolean;
};

export type Source = {
  id: string;
  workspace_id: string;
  kind: string;
  label: string;
  endpoint?: string;
  status: string;
  last_error?: string;
  capabilities: Record<string, boolean>;
  created_at: string;
  updated_at: string;
};

let ownerToken = "";
let refreshToken = "";

export function setOwnerToken(token: string) {
  ownerToken = token.trim();
  refreshToken = "";
}

export function loginWithAuthenticator(email: string, password: string) {
  return request<{ access_token: string; refresh_token?: string; email?: string; user_id?: string }>("/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password })
  }).then((result) => {
    ownerToken = result.access_token;
    refreshToken = result.refresh_token || "";
    return result;
  });
}

function baseUrl() {
  const configured = typeof window !== "undefined" ? window.__SECRETS_MANAGER_CONFIG__?.apiBaseUrl : undefined;
  return configured && configured.trim() ? configured : resolveApiBase({ appendSuffix: true });
}

async function refreshOwnerSession() {
  if (!refreshToken) return false;
  const headers = new Headers({ "Content-Type": "application/json", "X-Workspace-ID": "local", "X-Actor-ID": "local-owner" });
  const response = await fetch(buildApiUrl("/auth/refresh", { baseUrl: baseUrl() }), {
    method: "POST", headers, body: JSON.stringify({ refresh_token: refreshToken }), cache: "no-store"
  });
  if (!response.ok) {
    ownerToken = "";
    refreshToken = "";
    return false;
  }
  const result = (await response.json()) as { access_token?: string; refresh_token?: string };
  if (!result.access_token) return false;
  ownerToken = result.access_token;
  refreshToken = result.refresh_token || refreshToken;
  return true;
}

async function request<T>(path: string, init: RequestInit = {}, retry = true) {
  const headers = new Headers(init.headers);
  headers.set("Content-Type", "application/json");
  headers.set("X-Workspace-ID", "local");
  headers.set("X-Actor-ID", "local-owner");
  const configuredToken = typeof window !== "undefined" ? window.__SECRETS_MANAGER_CONFIG__?.ownerToken : undefined;
  const token = ownerToken || configuredToken?.trim();
  if (token) headers.set("Authorization", `Bearer ${token}`);
  const response = await fetch(buildApiUrl(path, { baseUrl: baseUrl() }), { ...init, headers, cache: "no-store" });
  if (response.status === 401 && retry && refreshToken && path !== "/auth/refresh" && await refreshOwnerSession()) {
    return request<T>(path, init, false);
  }
  const body = (await response.json().catch(() => ({}))) as T & { message?: string };
  if (!response.ok) throw new Error(body.message || `Request failed (${response.status})`);
  return body;
}

export function getVaultStatus(vault = "personal") {
  return request<VaultStatus>(`/vaults/${encodeURIComponent(vault)}/status`);
}

export function getEnrollmentStatus() {
  return request<EnrollmentStatus>("/enrollment/status");
}

export function completeEnrollment(bootstrapToken: string) {
  return request<{ workspace_id: string; principal_id: string; role: string; owner_token: string; token_once: boolean }>("/enrollment/complete", {
    method: "POST",
    body: JSON.stringify({ bootstrap_token: bootstrapToken })
  });
}

export function unlockVault(vault = "personal") {
  return request<VaultStatus>(`/vaults/${encodeURIComponent(vault)}/unlock`, { method: "POST", body: "{}" });
}

export function lockVault(vault = "personal") {
  return request<VaultStatus>(`/vaults/${encodeURIComponent(vault)}/lock`, { method: "POST", body: "{}" });
}

export function listVaultItems(query = "", includeTrashed = false, vault = "personal") {
  const params = new URLSearchParams();
  if (query) params.set("q", query);
  if (includeTrashed) params.set("include_trashed", "true");
  return request<{ items: VaultItem[] }>(`/vaults/${encodeURIComponent(vault)}/items?${params.toString()}`);
}

export function createVaultItem(input: Record<string, unknown>, vault = "personal") {
  return request<VaultItem>(`/vaults/${encodeURIComponent(vault)}/items`, {
    method: "POST",
    headers: { "Idempotency-Key": crypto.randomUUID() },
    body: JSON.stringify(input)
  });
}

export function updateVaultItem(id: string, revision: number, input: Record<string, unknown>, vault = "personal") {
  return request<VaultItem>(`/vaults/${encodeURIComponent(vault)}/items/${encodeURIComponent(id)}`, {
    method: "PUT",
    headers: { "If-Match": String(revision) },
    body: JSON.stringify(input)
  });
}

export function trashVaultItem(id: string, vault = "personal") {
  return request<{ id: string; trashed: boolean }>(`/vaults/${encodeURIComponent(vault)}/items/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export function restoreVaultItem(id: string, vault = "personal") {
  return request<{ id: string; trashed: boolean }>(`/vaults/${encodeURIComponent(vault)}/items/${encodeURIComponent(id)}/restore`, { method: "POST", body: "{}" });
}

export function listVaultItemHistory(id: string, vault = "personal") {
  return request<{ history: ItemHistoryRecord[] }>(`/vaults/${encodeURIComponent(vault)}/items/${encodeURIComponent(id)}/history`);
}

export function getRecoveryStatus() {
  return request<RecoveryStatus>("/recovery/status");
}

export function previewVaultImport(bundle: string, vault = "personal") {
  return request<ImportPreview>(`/vaults/${encodeURIComponent(vault)}/imports/preview`, {
    method: "POST",
    body: JSON.stringify({ bundle })
  });
}

export function commitVaultImport(bundle: string, duplicatePolicy: "skip" | "rename" | "replace", vault = "personal") {
  return request<{ created: number; skipped: number; renamed: number; source_vault_id: string }>(`/vaults/${encodeURIComponent(vault)}/imports`, {
    method: "POST",
    body: JSON.stringify({ bundle, duplicate_policy: duplicatePolicy })
  });
}

export async function revealVaultItem(id: string, fields: string[], vault = "personal") {
  const requestDigest = `reveal:${id}:${[...fields].sort().join(",")}`;
  const assurance = await request<{ assurance_token: string }>("/assurance", {
    method: "POST",
    body: JSON.stringify({ operation: `reveal:${id}`, request_digest: requestDigest })
  });
  return request<{ fields: Record<string, string> }>(`/vaults/${encodeURIComponent(vault)}/items/${encodeURIComponent(id)}/reveal`, {
    method: "POST",
    body: JSON.stringify({ assurance_token: assurance.assurance_token, request_digest: requestDigest, fields })
  });
}

export function generatePassword(length = 24) {
  return request<{ password: string }>("/password/generate", {
    method: "POST",
    body: JSON.stringify({ length, upper: true, lower: true, digits: true, symbols: true })
  });
}

export function createGrant(input: Record<string, unknown>) {
  return request<Grant>("/grants", { method: "POST", body: JSON.stringify(input) });
}

export function listGrants() {
  return request<{ grants: Grant[] }>("/grants");
}

export function getGrant(id: string) {
  return request<Grant>(`/grants/${encodeURIComponent(id)}`);
}

export function getGrantEffectiveAccess(id: string, itemId?: string, operation = "use") {
  const params = new URLSearchParams({ operation });
  if (itemId) params.set("item_id", itemId);
  return request<GrantAccessExplanation>(`/grants/${encodeURIComponent(id)}/effective-access?${params.toString()}`);
}

export function revokeGrant(id: string) {
  return request<{ grant_id: string; status: string; remote_purge?: string }>(`/grants/${encodeURIComponent(id)}/revoke`, { method: "POST", body: "{}" });
}

export function createAccessRequest(input: Record<string, unknown>) {
  return request<AccessRequest>("/access-requests", { method: "POST", body: JSON.stringify(input) });
}

export function listAccessRequests() {
  return request<{ requests: AccessRequest[] }>("/access-requests");
}

export function getAccessRequest(id: string) {
  return request<AccessRequest>(`/access-requests/${encodeURIComponent(id)}`);
}

export function waitForAccessRequest(id: string, timeoutSeconds = 30) {
  return request<{ request: AccessRequest; timed_out: boolean }>(`/access-requests/${encodeURIComponent(id)}/wait`, {
    method: "POST",
    body: JSON.stringify({ timeout_seconds: timeoutSeconds })
  });
}

export function getAudit() {
  return request<{ events: AuditEvent[]; integrity?: "verified" | "unknown" | "tampered" }>("/audit");
}

export function exportAudit() {
  return request<{ events: AuditEvent[]; integrity: "verified" | "unknown" | "tampered"; payload_policy: "metadata_only" }>("/audit/export");
}

export function issueAssurance(operation: string, requestDigest: string) {
  return request<{ assurance_token: string }>("/assurance", {
    method: "POST",
    body: JSON.stringify({ operation, request_digest: requestDigest })
  });
}

export function approveAccessRequest(id: string, digest: string, assuranceToken?: string) {
  return request<AccessRequest>(`/access-requests/${encodeURIComponent(id)}/approve`, {
    method: "POST",
    body: JSON.stringify({ request_digest: digest, assurance_token: assuranceToken })
  });
}

export function denyAccessRequest(id: string, digest: string, assuranceToken?: string) {
  return request<AccessRequest>(`/access-requests/${encodeURIComponent(id)}/deny`, {
    method: "POST",
    body: JSON.stringify({ request_digest: digest, assurance_token: assuranceToken })
  });
}

export async function createVaultExport(format: "native" | "plaintext", vault = "personal", itemIds: string[] = []) {
  const headers: HeadersInit = {};
  const requestDigest = `export:${vault}:${[...itemIds].sort().join(",")}`;
  if (format === "plaintext") {
    const assurance = await request<{ assurance_token: string }>("/assurance", {
      method: "POST",
      body: JSON.stringify({ operation: `export:${vault}`, request_digest: requestDigest })
    });
    headers["X-Assurance-Token"] = assurance.assurance_token;
  }
  return request<ExportHandle>(`/vaults/${encodeURIComponent(vault)}/export`, {
    method: "POST",
    headers,
    body: JSON.stringify({ format, item_ids: itemIds, request_digest: requestDigest })
  });
}

export function redeemVaultExport(handle: ExportHandle) {
  return request<{ version: number; format: string; vault_id: string; items: Array<Record<string, unknown>> }>(`/exports/${encodeURIComponent(handle.download_handle)}/redeem`, {
    method: "POST",
    headers: { "X-Download-Token": handle.download_token },
    body: "{}"
  });
}

export function setConfiguredOwnerToken(token: string) {
  setOwnerToken(token);
}

export function listSources() {
  return request<{ sources: Source[] }>("/sources");
}

export function createSource(input: { kind: string; label: string; endpoint?: string; bootstrap_ref?: string }) {
  return request<Source>("/sources", { method: "POST", body: JSON.stringify(input) });
}

export function getSourceHealth(id: string) {
  return request<Source>(`/sources/${encodeURIComponent(id)}/health`);
}

export function bindVaultItem(itemId: string, sourceId: string, externalRef: string, vault = "personal") {
  return request<{ item_id: string; source_id: string; binding: string }>(`/vaults/${encodeURIComponent(vault)}/items/${encodeURIComponent(itemId)}/source-binding`, {
    method: "POST",
    body: JSON.stringify({ source_id: sourceId, external_ref: externalRef })
  });
}
