import { buildApiUrl } from "@vrooli/api-base";
import { deviceIdentity } from "../lib/deviceIdentity";
import { getAccessToken } from "../lib/auth";
import { API_BASE, makeApiError } from "./client";

export interface StagingProfile {
  id: string;
  scenario: string;
  environment: string;
  domain: string;
  provider: string;
  offer: string;
  regions: string[];
  cap_currency: string;
  cap_minor_units: number;
  approval: string;
  health_path: string;
}
export interface StagingOperation {
  operation_id: string;
  release_id: string;
	profile_id: string;
  status: string;
  active_stage?: string;
  error?: string;
  updated_at?: string;
  cloud_spend_quote_idempotency_key?: string;
  cloud_compute_manager_instance_id?: string;
  cloud_step?: { state?: string; reason?: string; deployment_id?: string; owner_operation_id?: string; request_key?: string; request_digest?: string; receipt_json?: string; verification_receipt_json?: string; updated_at?: string };
	resume_available: boolean;
	resume_reason?: string;
  tls: { state: "healthy" | "degraded" | "unhealthy" | "unknown" | "not_applicable" | "ambiguous" | "unavailable"; reason?: string; sampled_at?: string };
  certificate: { state: "valid" | "expired" | "unavailable"; not_after?: string; sampled_at?: string; renewal: "blocked"; renewal_reason?: string; reason?: string };
}
export interface StagingState {
  profile: StagingProfile;
  readiness: { state: "ready" | "blocked"; reasons: string[] };
  deployment_manager_profile_id: string;
}
export interface StagingRecovery {
  state: "awaiting_preview" | "preview_ready" | "pending" | "succeeded" | "failed" | "ambiguous" | "stale_preview" | "blocked" | "unavailable";
  reason?: string;
  operation_id: string;
  deployment_id: string;
  current_release_digest: string;
  current_bundle_sha256: string;
  last_good_release_digest: string;
  last_good_bundle_sha256?: string;
  review_ref: string;
  data_compatibility: string;
  preview_ref?: string;
  route_kind?: string;
  impact_summary?: string;
  recovery_operation_id?: string;
  recovery_status?: string;
  recovery_error?: string;
  recovery_receipt_json?: string;
}

async function fetchJSON<T>(path: string): Promise<T> {
  const headers = new Headers({ "X-Vrooli-Device-Id": deviceIdentity().id });
  const token = await getAccessToken();
  if (token) headers.set("Authorization", `Bearer ${token}`);
  const response = await fetch(buildApiUrl(path, { baseUrl: API_BASE }), { headers, cache: "no-store" });
  if (!response.ok) throw makeApiError("deployment_status_error", `deployment status request failed (${response.status})`, response.status);
  return response.json() as Promise<T>;
}

async function postJSON<T>(path: string, body?: unknown): Promise<T> {
  const headers = new Headers({ "X-Vrooli-Device-Id": deviceIdentity().id });
  if (body !== undefined) headers.set("Content-Type", "application/json");
  const token = await getAccessToken();
  if (token) headers.set("Authorization", `Bearer ${token}`);
  const response = await fetch(buildApiUrl(path, { baseUrl: API_BASE }), { method: "POST", headers, body: body === undefined ? undefined : JSON.stringify(body), cache: "no-store" });
  if (!response.ok) throw makeApiError("deployment_action_error", `deployment action was refused (${response.status})`, response.status);
  return response.json() as Promise<T>;
}

export async function getStagingState(): Promise<StagingState> {
  return fetchJSON<StagingState>("/api/v1/deployments/staging");
}
export async function listStagingOperations(): Promise<StagingOperation[]> {
  const body = await fetchJSON<{ operations?: StagingOperation[] }>("/api/v1/deployments/staging/operations");
  return body.operations ?? [];
}
export async function requestStagingStart(): Promise<{ state: "blocked"; reasons: string[]; deployment_manager_called: false }> {
  return postJSON<{ state: "blocked"; reasons: string[]; deployment_manager_called: false }>("/api/v1/deployments/staging/start");
}
export async function resumeStagingOperation(operationID: string): Promise<{ operation: StagingOperation }> {
  return postJSON<{ operation: StagingOperation }>(`/api/v1/deployments/staging/operations/${encodeURIComponent(operationID)}/resume`);
}
export async function getStagingRecovery(operationID: string): Promise<{ recovery: StagingRecovery }> {
  return fetchJSON<{ recovery: StagingRecovery }>(`/api/v1/deployments/staging/operations/${encodeURIComponent(operationID)}/recovery`);
}
export async function previewStagingRecovery(operationID: string): Promise<{ recovery: StagingRecovery }> {
  return postJSON<{ recovery: StagingRecovery }>(`/api/v1/deployments/staging/operations/${encodeURIComponent(operationID)}/recovery/preview`, { compatibility_confirmed: true });
}
export async function confirmStagingRecovery(operationID: string, previewRef: string): Promise<{ recovery: StagingRecovery }> {
  return postJSON<{ recovery: StagingRecovery }>(`/api/v1/deployments/staging/operations/${encodeURIComponent(operationID)}/recovery/confirm`, { confirmed: true, confirmed_preview_ref: previewRef });
}
