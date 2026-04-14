import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";
import type { Resource, OnboardingProgress, ResourceHealthResponse, GlossaryResponse, SetupOrderResponse } from "../types";

// Simple! Just specify if you want the /api/v1 suffix
const API_BASE = resolveApiBase({ appendSuffix: true });

/** Type-safe fetch wrapper that handles error checking and JSON parsing */
async function typedFetch<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    ...init,
  });
  if (!res.ok) {
    throw new Error(`API request failed: ${res.status}`);
  }
  return res.json();
}

export function fetchHealth() {
  const url = buildApiUrl("/health", { baseUrl: API_BASE });
  return typedFetch<{ status: string; service: string; timestamp: string }>(url, { cache: "no-store" });
}

export async function fetchResources() {
  const url = buildApiUrl("/resources", { baseUrl: API_BASE });
  const data = await typedFetch<{ resources: Resource[] } | Resource[]>(url, { cache: "no-store" });
  // API wraps resources in {count, resources: [...]}, unwrap if needed
  return Array.isArray(data) ? data : data.resources;
}

export function fetchProgress(userId?: string) {
  const params = new URLSearchParams();
  if (userId) params.set("user_id", userId);
  const url = buildApiUrl(`/progress${params.toString() ? `?${params}` : ""}`, { baseUrl: API_BASE });
  return typedFetch<OnboardingProgress>(url, { cache: "no-store" });
}

export function updateProgress(data: {
  current_step: number;
  completed_steps: number[];
  config_data: Record<string, unknown>;
}) {
  const url = buildApiUrl("/progress", { baseUrl: API_BASE });
  return typedFetch<OnboardingProgress>(url, {
    method: "PUT",
    body: JSON.stringify({ user_id: "default", ...data }),
  });
}

export function completeOnboarding() {
  const url = buildApiUrl("/complete", { baseUrl: API_BASE });
  return typedFetch<{ status: string; user_id: string; completed_at: string; config_path: string }>(url, {
    method: "POST",
    body: JSON.stringify({ user_id: "default" }),
  });
}

export function generateConfig(resources: string[]) {
  const url = buildApiUrl("/config/generate", { baseUrl: API_BASE });
  return typedFetch<Record<string, unknown>>(url, {
    method: "POST",
    body: JSON.stringify({ resources }),
  });
}

export function validateConfig(resources: Record<string, { enabled: boolean; name: string }>) {
  const url = buildApiUrl("/config/validate", { baseUrl: API_BASE });
  return typedFetch<{ valid: boolean; errors?: string[]; warnings?: string[] }>(url, {
    method: "POST",
    body: JSON.stringify({ resources }),
  });
}

export function fetchResourceHealth() {
  const url = buildApiUrl("/resources/health", { baseUrl: API_BASE });
  return typedFetch<ResourceHealthResponse>(url, { cache: "no-store" });
}

export function fetchGlossary(query?: string) {
  const params = query ? `?q=${encodeURIComponent(query)}` : "";
  const url = buildApiUrl(`/glossary${params}`, { baseUrl: API_BASE });
  return typedFetch<GlossaryResponse>(url, { cache: "no-store" });
}

export function fetchSetupOrder() {
  const url = buildApiUrl("/setup-order", { baseUrl: API_BASE });
  return typedFetch<SetupOrderResponse>(url, { cache: "no-store" });
}
