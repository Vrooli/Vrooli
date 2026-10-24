// Fetch helpers for the Phase 3 operational stats endpoints.
//
// API: scenarios/agent-manager/api/internal/handlers/operational_stats.go

import type { FallbackInsights, HealthSummary } from "./operationalTypes";

const BASE = "/api/v1/stats";

async function fetchJson<T>(url: string): Promise<T> {
  const res = await fetch(url, { headers: { "Content-Type": "application/json" }, cache: "no-store" });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`Operational stats API error ${res.status}: ${body}`);
  }
  return (await res.json()) as T;
}

export async function fetchFallbackInsights(): Promise<FallbackInsights> {
  return fetchJson<FallbackInsights>(`${BASE}/fallback`);
}

export async function fetchHealthSummary(): Promise<HealthSummary> {
  return fetchJson<HealthSummary>(`${BASE}/operational?category=health`);
}

export const operationalQueryKeys = {
  all: ["operational-stats"] as const,
  fallback: () => [...operationalQueryKeys.all, "fallback"] as const,
  health: () => [...operationalQueryKeys.all, "health"] as const,
};
