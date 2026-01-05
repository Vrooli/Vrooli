import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

// Simple! Just specify if you want the /api/v1 suffix
const API_BASE = resolveApiBase({ appendSuffix: true });

export async function fetchHealth() {
  const url = buildApiUrl("/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`API health check failed: ${res.status}`);
  }

  return res.json() as Promise<{ status: string; service: string; timestamp: string }>;
}

export type ValidationIssue = {
  path: string;
  message: string;
  hint?: string;
  severity: "error" | "warn";
};

export type ManifestValidateResponse = {
  valid: boolean;
  issues?: ValidationIssue[];
  /** The normalized/auto-fixed manifest returned by the backend */
  manifest: unknown;
  timestamp: string;
  schema_hint?: string;
};

export type BundleArtifact = { path: string; sha256: string; size_bytes: number };
export type BundleBuildResponse = { artifact: BundleArtifact; timestamp: string };

export type BundleInfo = {
  path: string;
  filename: string;
  scenario_id: string;
  sha256: string;
  size_bytes: number;
  created_at: string;
};
export type ListBundlesResponse = { bundles: BundleInfo[]; timestamp: string };

export type ScenarioStats = {
  count: number;
  size_bytes: number;
};

export type BundleStats = {
  total_count: number;
  total_size_bytes: number;
  oldest_created_at?: string;
  newest_created_at?: string;
  by_scenario: Record<string, ScenarioStats>;
};

export type BundleStatsResponse = {
  stats: BundleStats;
  timestamp: string;
};

export type BundleCleanupRequest = {
  scenario_id?: string;
  keep_latest?: number;
  clean_vps?: boolean;
  host?: string;
  port?: number;
  user?: string;
  key_path?: string;
  workdir?: string;
};

export type BundleCleanupResponse = {
  ok: boolean;
  local_deleted?: BundleInfo[];
  local_freed_bytes: number;
  vps_deleted?: number;
  vps_freed_bytes?: number;
  vps_error?: string;
  message: string;
  timestamp: string;
};

export type PortConfig = {
  env_var?: string;
  description?: string;
  range?: string;
  port?: number;
};

export type ScenarioInfo = {
  id: string;
  displayName?: string;
  description?: string;
  ports?: Record<string, PortConfig>;
};

export type ScenariosResponse = {
  scenarios: ScenarioInfo[];
  timestamp: string;
};

export type ScenarioPortsResponse = {
  scenario_id: string;
  ports: Record<string, PortConfig>;
  timestamp: string;
};

export type ScenarioDependenciesResponse = {
  scenario_id: string;
  resources: string[];
  scenarios: string[];
  analyzer_available: boolean;
  source: "analyzer" | "service.json";
  timestamp: string;
};

export type ReachabilityResult = {
  target: string;
  type: "host" | "domain";
  reachable: boolean;
  message?: string;
  hint?: string;
};

export type ReachabilityResponse = {
  results: ReachabilityResult[];
  timestamp: string;
};

export async function validateManifest(manifest: unknown) {
  const url = buildApiUrl("/manifest/validate", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(manifest)
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Manifest validation failed: ${res.status} ${text}`);
  }
  return res.json() as Promise<ManifestValidateResponse>;
}

export async function buildBundle(manifest: unknown) {
  const url = buildApiUrl("/bundle/build", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(manifest)
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Bundle build failed: ${res.status} ${text}`);
  }
  return res.json() as Promise<BundleBuildResponse>;
}

export async function listBundles() {
  const url = buildApiUrl("/bundles", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to list bundles: ${res.status} ${text}`);
  }
  return res.json() as Promise<ListBundlesResponse>;
}

export async function getBundleStats() {
  const url = buildApiUrl("/bundles/stats", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to get bundle stats: ${res.status} ${text}`);
  }
  return res.json() as Promise<BundleStatsResponse>;
}

export async function cleanupBundles(request: BundleCleanupRequest) {
  const url = buildApiUrl("/bundles/cleanup", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to cleanup bundles: ${res.status} ${text}`);
  }
  return res.json() as Promise<BundleCleanupResponse>;
}

export type BundleDeleteResponse = {
  ok: boolean;
  freed_bytes: number;
  message: string;
  timestamp: string;
};

export async function deleteBundle(sha256: string): Promise<BundleDeleteResponse> {
  const url = buildApiUrl(`/bundles/${encodeURIComponent(sha256)}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" },
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to delete bundle: ${res.status} ${text}`);
  }
  return res.json() as Promise<BundleDeleteResponse>;
}

export type VPSBundleListRequest = {
  host: string;
  port?: number;
  user?: string;
  key_path: string;
  workdir: string;
};

export type VPSBundleInfo = {
  filename: string;
  scenario_id: string;
  sha256: string;
  size_bytes: number;
  mod_time: string;
};

export type VPSBundleListResponse = {
  ok: boolean;
  bundles: VPSBundleInfo[];
  total_size_bytes: number;
  error?: string;
  timestamp: string;
};

export async function listVPSBundles(request: VPSBundleListRequest): Promise<VPSBundleListResponse> {
  const url = buildApiUrl("/bundles/vps/list", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to list VPS bundles: ${res.status} ${text}`);
  }
  return res.json() as Promise<VPSBundleListResponse>;
}

export type VPSBundleDeleteRequest = {
  host: string;
  port?: number;
  user?: string;
  key_path: string;
  workdir: string;
  filename: string;
};

export type VPSBundleDeleteResponse = {
  ok: boolean;
  freed_bytes: number;
  message: string;
  error?: string;
  timestamp: string;
};

export async function deleteVPSBundle(request: VPSBundleDeleteRequest): Promise<VPSBundleDeleteResponse> {
  const url = buildApiUrl("/bundles/vps/delete", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to delete VPS bundle: ${res.status} ${text}`);
  }
  return res.json() as Promise<VPSBundleDeleteResponse>;
}

export async function listScenarios() {
  const url = buildApiUrl("/scenarios", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to list scenarios: ${res.status} ${text}`);
  }
  return res.json() as Promise<ScenariosResponse>;
}

export async function getScenarioPorts(scenarioId: string) {
  const url = buildApiUrl(`/scenarios/${encodeURIComponent(scenarioId)}/ports`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to get scenario ports: ${res.status} ${text}`);
  }
  return res.json() as Promise<ScenarioPortsResponse>;
}

/**
 * Get scenario dependencies (resources and scenarios) from dependency analyzer or service.json fallback.
 */
export async function getScenarioDependencies(scenarioId: string) {
  const url = buildApiUrl(`/scenarios/${encodeURIComponent(scenarioId)}/dependencies`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to get scenario dependencies: ${res.status} ${text}`);
  }
  return res.json() as Promise<ScenarioDependenciesResponse>;
}

export async function checkReachability(host?: string, domain?: string) {
  const url = buildApiUrl("/validate/reachability", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ host, domain })
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Reachability check failed: ${res.status} ${text}`);
  }
  return res.json() as Promise<ReachabilityResponse>;
}

export type PreflightCheckStatus = "pass" | "warn" | "fail";

export type PreflightCheck = {
  id: string;
  title: string;
  status: PreflightCheckStatus;
  details?: string;
  hint?: string;
  data?: Record<string, string>;
};

export type PreflightResponse = {
  ok: boolean;
  checks: PreflightCheck[];
  issues?: ValidationIssue[];
  timestamp: string;
};

export async function runPreflight(manifest: unknown) {
  const url = buildApiUrl("/preflight", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(manifest)
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Preflight check failed: ${res.status} ${text}`);
  }
  return res.json() as Promise<PreflightResponse>;
}

// ============================================================================
// Secrets Management Types & Functions
// ============================================================================

import type { SecretsManifest, ProvidedSecrets } from "../types/secrets";

export type SecretsResponse = {
  secrets: SecretsManifest;
};

/**
 * Fetch secrets requirements for a scenario from secrets-manager.
 * This returns what secrets are needed and how they will be handled during deployment.
 */
export async function fetchSecretsManifest(
  scenarioId: string,
  tier?: string,
  resources?: string[]
): Promise<SecretsResponse> {
  const params = new URLSearchParams();
  if (tier) params.set("tier", tier);
  if (resources?.length) params.set("resources", resources.join(","));
  const queryString = params.toString();

  const url = buildApiUrl(
    `/secrets/${encodeURIComponent(scenarioId)}${queryString ? `?${queryString}` : ""}`,
    { baseUrl: API_BASE }
  );
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to fetch secrets: ${res.status} ${text}`);
  }
  return res.json() as Promise<SecretsResponse>;
}

// Re-export secrets types for convenience
export type { SecretsManifest, ProvidedSecrets, BundleSecretPlan, SecretsSummary } from "../types/secrets";

// ============================================================================
// Deployment Management Types & Functions
// ============================================================================

export type DeploymentStatus =
  | "pending"
  | "setup_running"
  | "setup_complete"
  | "deploying"
  | "deployed"
  | "failed"
  | "stopped";

export type DeploymentSummary = {
  id: string;
  name: string;
  scenario_id: string;
  status: DeploymentStatus;
  domain?: string;
  host?: string;
  error_message?: string;
  progress_step?: string;
  progress_percent?: number;
  created_at: string;
  last_deployed_at?: string;
};

export type Deployment = {
  id: string;
  name: string;
  scenario_id: string;
  status: DeploymentStatus;
  run_id?: string;
  manifest: unknown;
  bundle_path?: string;
  bundle_sha256?: string;
  bundle_size_bytes?: number;
  setup_result?: unknown;
  deploy_result?: unknown;
  last_inspect_result?: VPSInspectResult;
  error_message?: string;
  error_step?: string;
  progress_step?: string;
  progress_percent?: number;
  created_at: string;
  updated_at: string;
  last_deployed_at?: string;
  last_inspected_at?: string;
};

export type VPSInspectResult = {
  ok: boolean;
  scenario_status?: unknown;
  resource_status?: unknown;
  scenario_logs?: string;
  error?: string;
  timestamp: string;
};

export type ListDeploymentsResponse = {
  deployments: DeploymentSummary[];
  timestamp: string;
};

export type DeploymentResponse = {
  deployment: Deployment;
  created?: boolean;
  updated?: boolean;
  timestamp: string;
};

// Execute deployment is now non-blocking - it returns immediately
// and the actual progress is tracked via SSE on /deployments/{id}/progress
export type ExecuteDeploymentResponse = {
  deployment: Deployment;
  message: string;
  timestamp: string;
};

export type InspectDeploymentResponse = {
  result: VPSInspectResult;
  timestamp: string;
};

export async function listDeployments(): Promise<ListDeploymentsResponse> {
  const url = buildApiUrl("/deployments", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to list deployments: ${res.status} ${text}`);
  }
  return res.json() as Promise<ListDeploymentsResponse>;
}

/**
 * Find an in-progress deployment, optionally filtered by scenario ID.
 * Returns the most recently created in-progress deployment, or null if none found.
 */
export async function findInProgressDeployment(
  scenarioId?: string
): Promise<DeploymentSummary | null> {
  const { deployments } = await listDeployments();

  // Filter to in-progress deployments
  const inProgress = deployments.filter(
    (d) =>
      (d.status === "setup_running" || d.status === "deploying" || d.status === "pending") &&
      (!scenarioId || d.scenario_id === scenarioId)
  );

  if (inProgress.length === 0) {
    return null;
  }

  // Return most recent (list is already sorted by created_at desc from API)
  return inProgress[0];
}

export async function createDeployment(
  manifest: unknown,
  options?: {
    name?: string;
    bundlePath?: string;
    bundleSha256?: string;
    bundleSizeBytes?: number;
    providedSecrets?: ProvidedSecrets;
  }
): Promise<DeploymentResponse> {
  const url = buildApiUrl("/deployments", { baseUrl: API_BASE });
  const body: Record<string, unknown> = { manifest };
  if (options?.name) body.name = options.name;
  if (options?.bundlePath) {
    body.bundle_path = options.bundlePath;
    body.bundle_sha256 = options.bundleSha256;
    body.bundle_size_bytes = options.bundleSizeBytes;
  }
  if (options?.providedSecrets && Object.keys(options.providedSecrets).length > 0) {
    body.provided_secrets = options.providedSecrets;
  }
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body)
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to create deployment: ${res.status} ${text}`);
  }
  return res.json() as Promise<DeploymentResponse>;
}

export async function getDeployment(id: string): Promise<DeploymentResponse> {
  const url = buildApiUrl(`/deployments/${encodeURIComponent(id)}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to get deployment: ${res.status} ${text}`);
  }
  return res.json() as Promise<DeploymentResponse>;
}

export async function executeDeployment(
  id: string,
  options?: { providedSecrets?: ProvidedSecrets }
): Promise<ExecuteDeploymentResponse> {
  const url = buildApiUrl(`/deployments/${encodeURIComponent(id)}/execute`, { baseUrl: API_BASE });
  const body: Record<string, unknown> = {};
  if (options?.providedSecrets && Object.keys(options.providedSecrets).length > 0) {
    body.provided_secrets = options.providedSecrets;
  }
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to execute deployment: ${res.status} ${text}`);
  }
  return res.json() as Promise<ExecuteDeploymentResponse>;
}

export async function inspectDeployment(id: string): Promise<InspectDeploymentResponse> {
  const url = buildApiUrl(`/deployments/${encodeURIComponent(id)}/inspect`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" }
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to inspect deployment: ${res.status} ${text}`);
  }
  return res.json() as Promise<InspectDeploymentResponse>;
}

export async function stopDeployment(id: string): Promise<{ success: boolean; error?: string; timestamp: string }> {
  const url = buildApiUrl(`/deployments/${encodeURIComponent(id)}/stop`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" }
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to stop deployment: ${res.status} ${text}`);
  }
  return res.json();
}

export async function deleteDeployment(
  id: string,
  options: { stopOnVPS?: boolean; cleanupBundles?: boolean } = {},
): Promise<{ deleted: boolean; timestamp: string }> {
  const params = new URLSearchParams();
  if (options.stopOnVPS) params.set("stop", "true");
  if (options.cleanupBundles) params.set("cleanup", "true");
  const queryString = params.toString();
  const url = buildApiUrl(
    `/deployments/${encodeURIComponent(id)}${queryString ? `?${queryString}` : ""}`,
    { baseUrl: API_BASE },
  );
  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" },
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to delete deployment: ${res.status} ${text}`);
  }
  return res.json();
}

// ============================================================================
// SSH Key Management Types & Functions
// ============================================================================

// Re-export types from types/ssh.ts for convenience
export type {
  SSHKeyInfo,
  SSHKeyType,
  SSHConnectionStatus,
  ListSSHKeysResponse,
  GenerateSSHKeyRequest,
  GenerateSSHKeyResponse,
  GetPublicKeyRequest,
  GetPublicKeyResponse,
  TestSSHConnectionRequest,
  TestSSHConnectionResponse,
  CopySSHKeyRequest,
  CopySSHKeyResponse,
  CopySSHKeyStatus,
} from "../types/ssh";

import type {
  ListSSHKeysResponse,
  GenerateSSHKeyRequest,
  GenerateSSHKeyResponse,
  GetPublicKeyResponse,
  TestSSHConnectionRequest,
  TestSSHConnectionResponse,
  CopySSHKeyRequest,
  CopySSHKeyResponse,
  DeleteSSHKeyRequest,
  DeleteSSHKeyResponse,
} from "../types/ssh";

/**
 * List available SSH keys from ~/.ssh/
 */
export async function listSSHKeys(): Promise<ListSSHKeysResponse> {
  const url = buildApiUrl("/ssh/keys", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to list SSH keys: ${res.status} ${text}`);
  }
  return res.json() as Promise<ListSSHKeysResponse>;
}

/**
 * Generate a new SSH key pair
 */
export async function generateSSHKey(
  request: GenerateSSHKeyRequest
): Promise<GenerateSSHKeyResponse> {
  const url = buildApiUrl("/ssh/keys/generate", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to generate SSH key: ${res.status} ${text}`);
  }
  return res.json() as Promise<GenerateSSHKeyResponse>;
}

/**
 * Get public key content for display/copying
 */
export async function getPublicKey(keyPath: string): Promise<GetPublicKeyResponse> {
  const url = buildApiUrl("/ssh/keys/public", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ key_path: keyPath }),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to get public key: ${res.status} ${text}`);
  }
  return res.json() as Promise<GetPublicKeyResponse>;
}

/**
 * Test SSH connection to a host using key authentication
 */
export async function testSSHConnection(
  request: TestSSHConnectionRequest
): Promise<TestSSHConnectionResponse> {
  const url = buildApiUrl("/ssh/test", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to test SSH connection: ${res.status} ${text}`);
  }
  return res.json() as Promise<TestSSHConnectionResponse>;
}

/**
 * Copy SSH public key to remote host (ssh-copy-id equivalent)
 * Requires password authentication to copy the key
 */
export async function copySSHKey(
  request: CopySSHKeyRequest
): Promise<CopySSHKeyResponse> {
  const url = buildApiUrl("/ssh/copy-key", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to copy SSH key: ${res.status} ${text}`);
  }
  return res.json() as Promise<CopySSHKeyResponse>;
}

/**
 * Delete an SSH key pair (private and public key files)
 */
export async function deleteSSHKey(
  request: DeleteSSHKeyRequest
): Promise<DeleteSSHKeyResponse> {
  const url = buildApiUrl("/ssh/keys", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to delete SSH key: ${res.status} ${text}`);
  }
  return res.json() as Promise<DeleteSSHKeyResponse>;
}

// ============================================================================
// Preflight Fix Actions
// ============================================================================

export interface StopPortServicesRequest {
  host: string;
  port?: number;
  user?: string;
  key_path: string;
}

export interface StopPortServicesResponse {
  ok: boolean;
  stopped: string[];
  failed?: string[];
  message: string;
  timestamp: string;
}

export interface DiskUsageRequest {
  host: string;
  port?: number;
  user?: string;
  key_path: string;
}

export interface DiskUsageEntry {
  path: string;
  size: string;
  bytes: number;
}

export interface DiskUsageResponse {
  ok: boolean;
  free_space: string;
  free_bytes: number;
  total_space: string;
  total_bytes: number;
  used_percent: number;
  largest_dirs: DiskUsageEntry[];
  timestamp: string;
}

export interface DiskCleanupRequest {
  host: string;
  port?: number;
  user?: string;
  key_path: string;
  actions?: string[];
}

export interface DiskCleanupResponse {
  ok: boolean;
  space_freed: string;
  space_freed_kb: number;
  message: string;
  actions_run: string[];
  actions_failed?: string[];
  timestamp: string;
}

/**
 * Stop services using ports 80/443
 */
export async function stopPortServices(
  request: StopPortServicesRequest
): Promise<StopPortServicesResponse> {
  const url = buildApiUrl("/preflight/fix/ports", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to stop port services: ${res.status} ${text}`);
  }
  return res.json() as Promise<StopPortServicesResponse>;
}

/**
 * Get detailed disk usage information
 */
export async function getDiskUsage(
  request: DiskUsageRequest
): Promise<DiskUsageResponse> {
  const url = buildApiUrl("/preflight/disk/usage", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to get disk usage: ${res.status} ${text}`);
  }
  return res.json() as Promise<DiskUsageResponse>;
}

/**
 * Run disk cleanup operations
 */
export async function runDiskCleanup(
  request: DiskCleanupRequest
): Promise<DiskCleanupResponse> {
  const url = buildApiUrl("/preflight/disk/cleanup", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to run disk cleanup: ${res.status} ${text}`);
  }
  return res.json() as Promise<DiskCleanupResponse>;
}

export interface StopScenarioProcessesRequest {
  host: string;
  port?: number;
  user?: string;
  key_path: string;
  workdir: string;
  scenario_id?: string; // If empty, stops all vrooli processes
}

export interface StopScenarioProcessesResponse {
  ok: boolean;
  action: string; // "stop_scenario" or "stop_all"
  message: string;
  output?: string;
  timestamp: string;
}

/**
 * Stop scenario processes on VPS (for clearing stale processes before deployment)
 * If scenario_id is provided, stops that specific scenario.
 * If scenario_id is empty/undefined, stops all vrooli processes.
 */
export async function stopScenarioProcesses(
  request: StopScenarioProcessesRequest
): Promise<StopScenarioProcessesResponse> {
  const url = buildApiUrl("/preflight/fix/stop-processes", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to stop scenario processes: ${res.status} ${text}`);
  }
  return res.json() as Promise<StopScenarioProcessesResponse>;
}

// ============================================================================
// Live State Types & Functions (Ground Truth Redesign)
// ============================================================================

export type ProcessPort = {
  name?: string;
  port: number;
  status: string;
  responding?: boolean;
  clients?: number;
};

export type ProcessResources = {
  cpu_percent: number;
  memory_mb: number;
  memory_percent: number;
};

export type ScenarioProcess = {
  id: string;
  status: string;
  pid: number;
  uptime_seconds: number;
  last_restart?: string;
  ports?: ProcessPort[];
  resources: ProcessResources;
  vrooli_status?: unknown;
};

export type ResourceProcess = {
  id: string;
  status: string;
  pid: number;
  port?: number;
  uptime_seconds: number;
  metrics?: unknown;
  vrooli_status?: unknown;
};

export type UnexpectedProcess = {
  pid: number;
  command: string;
  port?: number;
  user: string;
};

export type ProcessState = {
  scenarios: ScenarioProcess[];
  resources: ResourceProcess[];
  unexpected: UnexpectedProcess[];
};

export type PortBinding = {
  port: number;
  process: string;
  type: "system" | "edge" | "scenario" | "resource" | "unexpected";
  matches_manifest?: boolean;
  pid?: number;
  command?: string;
};

export type TLSInfo = {
  valid: boolean;
  issuer?: string;
  expires?: string;
  days_remaining?: number;
  error?: string;
};

export type CaddyRoute = {
  path: string;
  upstream: string;
};

export type CaddyState = {
  running: boolean;
  domain: string;
  tls: TLSInfo;
  routes: CaddyRoute[];
};

export type CPUInfo = {
  cores: number;
  model?: string;
  usage_percent: number;
  load_average: number[];
};

export type MemoryInfo = {
  total_mb: number;
  used_mb: number;
  free_mb: number;
  usage_percent: number;
};

export type DiskInfo = {
  total_gb: number;
  used_gb: number;
  free_gb: number;
  usage_percent: number;
};

export type SwapInfo = {
  total_mb: number;
  used_mb: number;
  usage_percent: number;
};

export type SSHHealth = {
  connected: boolean;
  latency_ms: number;
  key_in_auth: boolean;
  key_path: string;
  error?: string;
};

export type SystemState = {
  cpu: CPUInfo;
  memory: MemoryInfo;
  disk: DiskInfo;
  swap: SwapInfo;
  ssh: SSHHealth;
  uptime_seconds: number;
};

export type ExpectedProcess = {
  id: string;
  type: "scenario" | "resource";
  state: "running" | "stopped" | "needs_setup";
  directory_exists: boolean;
};

export type LiveStateResult = {
  ok: boolean;
  timestamp: string;
  sync_duration_ms: number;
  processes?: ProcessState;
  expected?: ExpectedProcess[];
  ports?: PortBinding[];
  caddy?: CaddyState;
  system?: SystemState;
  error?: string;
};

export type LiveStateResponse = {
  result: LiveStateResult;
  timestamp: string;
};

export type FileEntry = {
  name: string;
  type: "file" | "directory" | "symlink";
  size_bytes: number;
  modified: string;
  permissions: string;
};

export type FilesResponse = {
  ok: boolean;
  path: string;
  entries: FileEntry[];
  timestamp: string;
};

export type FileContentResponse = {
  ok: boolean;
  path: string;
  size_bytes: number;
  content: string;
  truncated: boolean;
  timestamp: string;
};

export type DriftCheck = {
  category: "scenarios" | "resources" | "ports" | "edge";
  name: string;
  status: "pass" | "warning" | "drift";
  expected: string;
  actual: string;
  message?: string;
  actions?: string[];
};

export type DriftSummary = {
  passed: number;
  warnings: number;
  drifts: number;
};

export type DriftReport = {
  ok: boolean;
  timestamp: string;
  summary: DriftSummary;
  checks: DriftCheck[];
};

export type DriftResponse = {
  result: DriftReport;
  timestamp: string;
};

export type KillProcessRequest = {
  pid: number;
  signal?: string;
};

export type KillProcessResponse = {
  ok: boolean;
  pid: number;
  signal: string;
  timestamp: string;
};

export type RestartRequest = {
  type: "scenario" | "resource";
  id: string;
};

export type RestartResponse = {
  ok: boolean;
  type: string;
  id: string;
  output?: string;
  timestamp: string;
};

export type ProcessAction = "start" | "stop" | "restart" | "setup";

export type ProcessControlRequest = {
  action: ProcessAction;
  type: "scenario" | "resource";
  id: string;
};

export type ProcessControlResponse = {
  ok: boolean;
  action: string;
  type: string;
  id: string;
  message: string;
  output?: string;
  timestamp: string;
};

export type VPSAction = "reboot" | "stop_vrooli" | "cleanup";
export type CleanupLevel = 1 | 2 | 3 | 4 | 5;

export type VPSActionRequest = {
  action: VPSAction;
  cleanup_level?: CleanupLevel;
  confirmation: string;
};

export type VPSActionResponse = {
  ok: boolean;
  action: string;
  message: string;
  output?: string;
  timestamp: string;
};

/**
 * Fetch live state from VPS (processes, ports, system info, caddy)
 */
export async function getLiveState(deploymentId: string): Promise<LiveStateResponse> {
  const url = buildApiUrl(`/deployments/${encodeURIComponent(deploymentId)}/live-state`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to get live state: ${res.status} ${text}`);
  }
  return res.json() as Promise<LiveStateResponse>;
}

/**
 * List files in a directory on VPS
 */
export async function getFiles(deploymentId: string, path?: string): Promise<FilesResponse> {
  const params = path ? `?path=${encodeURIComponent(path)}` : "";
  const url = buildApiUrl(`/deployments/${encodeURIComponent(deploymentId)}/files${params}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to list files: ${res.status} ${text}`);
  }
  return res.json() as Promise<FilesResponse>;
}

/**
 * Read file content from VPS
 */
export async function getFileContent(deploymentId: string, path: string): Promise<FileContentResponse> {
  const url = buildApiUrl(`/deployments/${encodeURIComponent(deploymentId)}/files/content?path=${encodeURIComponent(path)}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to read file: ${res.status} ${text}`);
  }
  return res.json() as Promise<FileContentResponse>;
}

/**
 * Get drift report comparing manifest vs actual state
 */
export async function getDrift(deploymentId: string): Promise<DriftResponse> {
  const url = buildApiUrl(`/deployments/${encodeURIComponent(deploymentId)}/drift`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to get drift report: ${res.status} ${text}`);
  }
  return res.json() as Promise<DriftResponse>;
}

/**
 * Kill a process on VPS
 */
export async function killProcess(deploymentId: string, request: KillProcessRequest): Promise<KillProcessResponse> {
  const url = buildApiUrl(`/deployments/${encodeURIComponent(deploymentId)}/actions/kill`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to kill process: ${res.status} ${text}`);
  }
  return res.json() as Promise<KillProcessResponse>;
}

/**
 * Restart a scenario or resource on VPS
 */
export async function restartProcess(deploymentId: string, request: RestartRequest): Promise<RestartResponse> {
  const url = buildApiUrl(`/deployments/${encodeURIComponent(deploymentId)}/actions/restart`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to restart: ${res.status} ${text}`);
  }
  return res.json() as Promise<RestartResponse>;
}

/**
 * Control a scenario or resource on VPS (start, stop, restart, setup)
 */
export async function controlProcess(
  deploymentId: string,
  request: ProcessControlRequest
): Promise<ProcessControlResponse> {
  const url = buildApiUrl(`/deployments/${encodeURIComponent(deploymentId)}/actions/process`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to ${request.action} ${request.type}: ${res.status} ${text}`);
  }
  return res.json() as Promise<ProcessControlResponse>;
}

/**
 * Execute VPS management actions (reboot, stop_vrooli, cleanup)
 */
export async function executeVPSAction(
  deploymentId: string,
  request: VPSActionRequest
): Promise<VPSActionResponse> {
  const url = buildApiUrl(`/deployments/${encodeURIComponent(deploymentId)}/actions/vps`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to execute VPS action: ${res.status} ${text}`);
  }
  return res.json() as Promise<VPSActionResponse>;
}

// ============================================================================
// History & Logs Types & Functions (Ground Truth Redesign - Phase 7)
// ============================================================================

export type HistoryEventType =
  | "deployment_created"
  | "bundle_built"
  | "preflight_started"
  | "preflight_completed"
  | "setup_started"
  | "setup_completed"
  | "deploy_started"
  | "deploy_completed"
  | "deploy_failed"
  | "inspection"
  | "stopped"
  | "restarted"
  | "autoheal_triggered";

export type HistoryEvent = {
  type: HistoryEventType;
  timestamp: string;
  message?: string;
  details?: string;
  duration_ms?: number;
  success?: boolean;
  bundle_hash?: string;
  step_name?: string;
  data?: unknown;
};

export type HistoryResponse = {
  ok: boolean;
  history: HistoryEvent[];
  timestamp: string;
};

export type LogEntry = {
  timestamp: string;
  source: string;
  level: string;
  message: string;
};

export type LogsResponse = {
  ok: boolean;
  logs: LogEntry[];
  total: number;
  filtered: number;
  sources: string[];
};

/**
 * Get deployment history timeline
 */
export async function getHistory(deploymentId: string): Promise<HistoryResponse> {
  const url = buildApiUrl(`/deployments/${encodeURIComponent(deploymentId)}/history`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to get history: ${res.status} ${text}`);
  }
  return res.json() as Promise<HistoryResponse>;
}

/**
 * Get aggregated logs from VPS
 */
export async function getLogs(
  deploymentId: string,
  options: { source?: string; level?: string; tail?: number; search?: string } = {}
): Promise<LogsResponse> {
  const params = new URLSearchParams();
  if (options.source) params.set("source", options.source);
  if (options.level) params.set("level", options.level);
  if (options.tail) params.set("tail", options.tail.toString());
  if (options.search) params.set("search", options.search);
  const queryString = params.toString();

  const url = buildApiUrl(
    `/deployments/${encodeURIComponent(deploymentId)}/logs${queryString ? `?${queryString}` : ""}`,
    { baseUrl: API_BASE }
  );
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to get logs: ${res.status} ${text}`);
  }
  return res.json() as Promise<LogsResponse>;
}

// ============================================================================
// Edge/TLS Management Types & Functions (Ground Truth Redesign - Enhancement)
// ============================================================================

export type DNSCheckResponse = {
  ok: boolean;
  domain: string;
  vps_host: string;
  domain_ips?: string[];
  vps_ips?: string[];
  points_to_vps: boolean;
  message: string;
  hint?: string;
  timestamp: string;
};

export type CaddyAction = "start" | "stop" | "restart" | "reload";

export type CaddyControlRequest = {
  action: CaddyAction;
};

export type CaddyControlResponse = {
  ok: boolean;
  action: string;
  message: string;
  output?: string;
  timestamp: string;
};

export type TLSInfoResponse = {
  ok: boolean;
  domain: string;
  valid: boolean;
  issuer?: string;
  subject?: string;
  not_before?: string;
  not_after?: string;
  days_remaining: number;
  serial_number?: string;
  sans?: string[];
  error?: string;
  timestamp: string;
};

export type TLSRenewResponse = {
  ok: boolean;
  domain: string;
  message: string;
  output?: string;
  timestamp: string;
};

/**
 * Check if domain DNS points to VPS
 */
export async function checkDNS(deploymentId: string): Promise<DNSCheckResponse> {
  const url = buildApiUrl(`/deployments/${encodeURIComponent(deploymentId)}/edge/dns-check`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to check DNS: ${res.status} ${text}`);
  }
  return res.json() as Promise<DNSCheckResponse>;
}

/**
 * Control Caddy service (start, stop, restart, reload)
 */
export async function controlCaddy(
  deploymentId: string,
  action: CaddyAction
): Promise<CaddyControlResponse> {
  const url = buildApiUrl(`/deployments/${encodeURIComponent(deploymentId)}/edge/caddy`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ action }),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to control Caddy: ${res.status} ${text}`);
  }
  return res.json() as Promise<CaddyControlResponse>;
}

/**
 * Get detailed TLS certificate information
 */
export async function getTLSInfo(deploymentId: string): Promise<TLSInfoResponse> {
  const url = buildApiUrl(`/deployments/${encodeURIComponent(deploymentId)}/edge/tls`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to get TLS info: ${res.status} ${text}`);
  }
  return res.json() as Promise<TLSInfoResponse>;
}

/**
 * Force TLS certificate renewal
 */
export async function renewTLS(deploymentId: string): Promise<TLSRenewResponse> {
  const url = buildApiUrl(`/deployments/${encodeURIComponent(deploymentId)}/edge/tls/renew`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to renew TLS: ${res.status} ${text}`);
  }
  return res.json() as Promise<TLSRenewResponse>;
}

// =============================================================================
// Investigation API (Agent-Manager Integration)
// =============================================================================

import type {
  Investigation,
  InvestigationSummary,
  CreateInvestigationRequest,
  ApplyFixesRequest,
  AgentManagerStatus
} from "../types/investigation";

export type TriggerInvestigationResponse = {
  investigation: Investigation;
};

export type ListInvestigationsResponse = {
  investigations: InvestigationSummary[];
};

export type GetInvestigationResponse = {
  investigation: Investigation;
};

/**
 * Trigger a new investigation for a failed deployment.
 * The investigation runs in the background; use getInvestigation to poll status.
 */
export async function triggerInvestigation(
  deploymentId: string,
  options?: CreateInvestigationRequest
): Promise<TriggerInvestigationResponse> {
  const url = buildApiUrl(`/deployments/${encodeURIComponent(deploymentId)}/investigate`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(options || {}),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to trigger investigation: ${res.status} ${text}`);
  }
  return res.json() as Promise<TriggerInvestigationResponse>;
}

/**
 * List all investigations for a deployment.
 */
export async function listInvestigations(
  deploymentId: string,
  limit?: number
): Promise<ListInvestigationsResponse> {
  let url = buildApiUrl(`/deployments/${encodeURIComponent(deploymentId)}/investigations`, { baseUrl: API_BASE });
  if (limit) {
    url += `?limit=${limit}`;
  }
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to list investigations: ${res.status} ${text}`);
  }
  return res.json() as Promise<ListInvestigationsResponse>;
}

/**
 * Get a single investigation by ID.
 */
export async function getInvestigation(
  deploymentId: string,
  investigationId: string
): Promise<GetInvestigationResponse> {
  const url = buildApiUrl(
    `/deployments/${encodeURIComponent(deploymentId)}/investigations/${encodeURIComponent(investigationId)}`,
    { baseUrl: API_BASE }
  );
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to get investigation: ${res.status} ${text}`);
  }
  return res.json() as Promise<GetInvestigationResponse>;
}

/**
 * Stop a running investigation.
 */
export async function stopInvestigation(
  deploymentId: string,
  investigationId: string
): Promise<{ success: boolean; message: string }> {
  const url = buildApiUrl(
    `/deployments/${encodeURIComponent(deploymentId)}/investigations/${encodeURIComponent(investigationId)}/stop`,
    { baseUrl: API_BASE }
  );
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to stop investigation: ${res.status} ${text}`);
  }
  return res.json();
}

export type ApplyFixesResponse = {
  investigation: Investigation;
};

/**
 * Apply selected fixes from a completed investigation.
 * Spawns a new agent to implement the fixes.
 */
export async function applyFixes(
  deploymentId: string,
  investigationId: string,
  options: ApplyFixesRequest
): Promise<ApplyFixesResponse> {
  const url = buildApiUrl(
    `/deployments/${encodeURIComponent(deploymentId)}/investigations/${encodeURIComponent(investigationId)}/apply-fixes`,
    { baseUrl: API_BASE }
  );
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(options),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to apply fixes: ${res.status} ${text}`);
  }
  return res.json() as Promise<ApplyFixesResponse>;
}

/**
 * Check agent-manager availability.
 */
export async function getAgentManagerStatus(): Promise<AgentManagerStatus> {
  const url = buildApiUrl("/agent-manager/status", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to get agent-manager status: ${res.status} ${text}`);
  }
  return res.json() as Promise<AgentManagerStatus>;
}
