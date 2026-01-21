import { buildUrl, throwIfNotOk } from "./client";

// ==================== Distribution Types ====================

export interface DistributionRetryConfig {
  max_attempts?: number;
  initial_backoff_ms?: number;
  max_backoff_ms?: number;
  backoff_multiplier?: number;
}

export interface DistributionTarget {
  name: string;
  enabled: boolean;
  provider: "s3" | "r2" | "s3-compatible";
  bucket: string;
  endpoint?: string;
  region?: string;
  path_prefix?: string;
  access_key_id_env: string;
  secret_access_key_env: string;
  acl?: string;
  cdn_url?: string;
  retry?: DistributionRetryConfig;
  created_at?: string;
  updated_at?: string;
}

export interface DistributionConfig {
  schema_version: string;
  targets: Record<string, DistributionTarget>;
  created_at?: string;
  updated_at?: string;
}

export interface DistributionConfigResponse {
  config: DistributionConfig | null;
  config_path: string;
}

export interface DistributionTargetResponse {
  target: DistributionTarget;
}

export interface DistributionTargetsResponse {
  targets: DistributionTarget[];
  count: number;
}

export interface DistributionTargetValidation {
  target_name: string;
  valid: boolean;
  errors: string[];
  warnings: string[];
}

export interface DistributionValidationResult {
  valid: boolean;
  targets: Record<string, DistributionTargetValidation>;
  error?: string;
}

export interface DistributionTestResult {
  target_name: string;
  success: boolean;
  message?: string;
  error?: string;
}

export interface DistributeRequest {
  scenario_name: string;
  version?: string;
  artifacts: Record<string, string>;
  target_names?: string[];
  parallel?: boolean;
  /** Inline credentials for targets. Keys are env var names, values are secrets. */
  inline_credentials?: Record<string, string>;
}

export interface CheckCredentialsRequest {
  target_names?: string[];
}

export interface TargetCredentialStatus {
  target_name: string;
  all_present: boolean;
  missing_credentials: string[];
  required_credentials: string[];
}

export interface CheckCredentialsResponse {
  all_present: boolean;
  targets: Record<string, TargetCredentialStatus>;
}

export interface DistributeResponse {
  distribution_id: string;
  status_url: string;
}

export interface UploadStatus {
  platform: string;
  status: "pending" | "uploading" | "completed" | "failed";
  url?: string;
  error?: string;
  progress?: number;
  started_at?: string;
  completed_at?: string;
}

export interface TargetDistribution {
  target_name: string;
  status: "pending" | "running" | "completed" | "partial" | "failed";
  uploads: Record<string, UploadStatus>;
  error?: string;
  started_at?: string;
  completed_at?: string;
}

export interface DistributionStatus {
  distribution_id: string;
  scenario_name: string;
  version?: string;
  status: "pending" | "running" | "completed" | "partial" | "failed" | "cancelled";
  targets: Record<string, TargetDistribution>;
  error?: string;
  started_at?: string;
  completed_at?: string;
}

// ==================== Distribution Functions ====================

export async function fetchDistributionConfig(): Promise<DistributionConfigResponse> {
  const response = await fetch(buildUrl("/distribution/config"));
  await throwIfNotOk(response);
  return response.json();
}

export async function fetchDistributionConfigPath(): Promise<{ path: string }> {
  const response = await fetch(buildUrl("/distribution/config-path"));
  await throwIfNotOk(response);
  return response.json();
}

export async function fetchDistributionTargets(): Promise<DistributionTargetsResponse> {
  const response = await fetch(buildUrl("/distribution/targets"));
  await throwIfNotOk(response);
  return response.json();
}

export async function fetchDistributionTarget(name: string): Promise<DistributionTargetResponse> {
  const response = await fetch(buildUrl(`/distribution/targets/${encodeURIComponent(name)}`));
  await throwIfNotOk(response);
  return response.json();
}

export async function createDistributionTarget(target: DistributionTarget): Promise<DistributionTargetResponse> {
  const response = await fetch(buildUrl("/distribution/targets"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(target),
  });
  await throwIfNotOk(response);
  return response.json();
}

export async function updateDistributionTarget(
  name: string,
  target: Partial<DistributionTarget>
): Promise<DistributionTargetResponse> {
  const response = await fetch(buildUrl(`/distribution/targets/${encodeURIComponent(name)}`), {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(target),
  });
  await throwIfNotOk(response);
  return response.json();
}

export async function deleteDistributionTarget(name: string): Promise<{ status: string }> {
  const response = await fetch(buildUrl(`/distribution/targets/${encodeURIComponent(name)}`), {
    method: "DELETE",
  });
  await throwIfNotOk(response);
  return response.json();
}

export async function testDistributionTarget(name: string): Promise<DistributionTestResult> {
  const response = await fetch(buildUrl(`/distribution/targets/${encodeURIComponent(name)}/test`), {
    method: "POST",
  });
  await throwIfNotOk(response);
  return response.json();
}

export async function validateDistributionTargets(
  targetNames?: string[]
): Promise<DistributionValidationResult> {
  const response = await fetch(buildUrl("/distribution/validate"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ target_names: targetNames }),
  });
  await throwIfNotOk(response);
  return response.json();
}

export async function startDistribution(request: DistributeRequest): Promise<DistributeResponse> {
  const response = await fetch(buildUrl("/distribution/distribute"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  await throwIfNotOk(response);
  return response.json();
}

export async function fetchDistributionStatus(distributionId: string): Promise<DistributionStatus> {
  const response = await fetch(buildUrl(`/distribution/status/${encodeURIComponent(distributionId)}`));
  await throwIfNotOk(response);
  return response.json();
}

export async function cancelDistribution(distributionId: string): Promise<{ status: string }> {
  const response = await fetch(buildUrl(`/distribution/cancel/${encodeURIComponent(distributionId)}`), {
    method: "POST",
  });
  await throwIfNotOk(response);
  return response.json();
}

export async function checkDistributionCredentials(
  request?: CheckCredentialsRequest
): Promise<CheckCredentialsResponse> {
  const response = await fetch(buildUrl("/distribution/check-credentials"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request || {}),
  });
  await throwIfNotOk(response);
  return response.json();
}
