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
import { apiRequest } from "../../../hooks/useApiTransport";

const fetchJson = <T>(path: string) => apiRequest<T>(path, { cache: "no-store" });

export async function fetchModelHealth(): Promise<ModelHealthListResponse> {
  return fetchJson<ModelHealthListResponse>(`/health/models`);
}

export async function fetchRunnerHealth(): Promise<RunnerHealthListResponse> {
  return fetchJson<RunnerHealthListResponse>(`/health/runners`);
}

export async function fetchModelPolicyDrift(): Promise<ModelPolicyDriftSnapshot> {
	return fetchJson<ModelPolicyDriftSnapshot>(`/health/model-policy-drift`);
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
  return fetchJson<HealthAuditResponse>(`/health/audit?${params.toString()}`);
}

export const healthQueryKeys = {
  all: ["health"] as const,
  models: () => [...healthQueryKeys.all, "models"] as const,
  runners: () => [...healthQueryKeys.all, "runners"] as const,
  modelPolicyDrift: () => [...healthQueryKeys.all, "model-policy-drift"] as const,
  audit: (filters: HealthAuditFilters) => [...healthQueryKeys.all, "audit", filters] as const,
};
