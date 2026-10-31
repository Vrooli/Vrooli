import { buildApiUrl } from "@vrooli/api-base";

import { API_BASE } from "./client";

export type AuthPolicy = { max_attempts: number; attempt_limit: number; settle: number };
export type AuthProfile = {
  id: string;
  device_id: string;
  method: string;
  credential_identity: string;
  credential_field: string;
  verification: string;
  policy: AuthPolicy;
  status: string;
  last_outcome?: string;
  revoked_at?: string;
  created_at: string;
  updated_at: string;
};
export type ProviderStatus = { provider?: string; provider_state: string; configured: boolean; provider_detail?: string };
export type UnlockResult = { profile_id: string; device_id: string; method: string; outcome: string; next_action: string; attempts: number; provider_state?: string; before_lock_state: string; after_lock_state?: string };

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  headers.set("Content-Type", "application/json");
  const response = await fetch(buildApiUrl(path, { baseUrl: API_BASE }), { cache: "no-store", ...init, headers });
  const body: unknown = await response.json().catch(() => ({}));
  if (!response.ok) {
    const error = typeof body === "object" && body !== null ? body as { message?: unknown; code?: unknown } : {};
    throw new Error(typeof error.message === "string" ? error.message : typeof error.code === "string" ? error.code : `Request failed (${response.status})`);
  }
  return body as T;
}

export const listAuthProfiles = () => request<{ profiles: AuthProfile[] }>("/api/v1/auth/profiles");
export const getAuthProfile = (id: string) => request<{ profile: AuthProfile; provider: ProviderStatus }>(`/api/v1/auth/profiles/${encodeURIComponent(id)}`);
export const updateAuthProfile = (id: string, profile: Partial<AuthProfile>) => request<{ profile: AuthProfile }>(`/api/v1/auth/profiles/${encodeURIComponent(id)}`, { method: "PUT", body: JSON.stringify({ profile }) });
export const testAuthProfile = (id: string) => request<{ profile: AuthProfile; provider: ProviderStatus; outcome: string }>(`/api/v1/auth/profiles/${encodeURIComponent(id)}/test`, { method: "POST" });
export const unlockDevice = (profile_id: string, device_id: string, actor: string, lease_token: string) => request<{ profile_id: string; device_id: string; outcome: string; next_action: string; attempts: number; before_lock_state: string; after_lock_state?: string }>("/api/v1/auth/unlock", { method: "POST", body: JSON.stringify({ profile_id, device_id, actor, lease_token }) });
