// Fetch helpers for the persisted health audit endpoints.
//
// All endpoints live under /api/v1/health/* and are proxied by the UI
// dev server. See API: scenarios/agent-manager/api/internal/handlers/health_audit.go.

import type {
  HealthAuditFilters,
  HealthAuditResponse,
  ModelHealthListResponse,
	RunnerHealthListResponse,
	ModelPolicyDriftSnapshot,
} from "./types";

const BASE = "/api/v1/health";

async function fetchJson<T>(url: string): Promise<T> {
  const res = await fetch(url, { headers: { "Content-Type": "application/json" }, cache: "no-store" });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`Health API error ${res.status}: ${body}`);
  }
  return (await res.json()) as T;
}

export async function fetchModelHealth(): Promise<ModelHealthListResponse> {
  return fetchJson<ModelHealthListResponse>(`${BASE}/models`);
}

export async function fetchRunnerHealth(): Promise<RunnerHealthListResponse> {
  return fetchJson<RunnerHealthListResponse>(`${BASE}/runners`);
}

export async function fetchModelPolicyDrift(): Promise<ModelPolicyDriftSnapshot> {
	return fetchJson<ModelPolicyDriftSnapshot>(`${BASE}/model-policy-drift`);
}

export async function fetchHealthAudit(filters: HealthAuditFilters): Promise<HealthAuditResponse> {
  const params = new URLSearchParams();
  params.set("scope", filters.scope);
  if (filters.runner) params.set("runner", filters.runner);
  if (filters.model) params.set("model", filters.model);
  if (filters.status) params.set("status", filters.status);
  if (filters.since) params.set("since", filters.since);
  if (filters.until) params.set("until", filters.until);
  if (filters.limit) params.set("limit", String(filters.limit));
  return fetchJson<HealthAuditResponse>(`${BASE}/audit?${params.toString()}`);
}

export const healthQueryKeys = {
  all: ["health"] as const,
  models: () => [...healthQueryKeys.all, "models"] as const,
  runners: () => [...healthQueryKeys.all, "runners"] as const,
  modelPolicyDrift: () => [...healthQueryKeys.all, "model-policy-drift"] as const,
  audit: (filters: HealthAuditFilters) => [...healthQueryKeys.all, "audit", filters] as const,
};
