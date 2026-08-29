import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";
import type {
  Resource,
  ResourceHealthResponse,
  GlossaryResponse,
  OperatorState,
  OperatorStatePatch,
  V2ScenarioResponse,
  V2Recommendation,
  V2CoreSetResponse,
  V2ReadinessResponse,
  V2HostRequirementsResponse,
  V2ClosureResponse,
  V2ResourceResponse,
  V2ApplyResponse,
  V2ApplyPlanResponse,
  V2SessionResponse,
  V2StepsResponse,
  OperatorInputQueue,
  CapabilityStatusResponse,
  CapabilityPreview,
  CapabilityResult,
} from "../types";

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
  const data = (await res.json()) as unknown;
  return data as T;
}

export function fetchHealth() {
  const url = buildApiUrl("/health", { baseUrl: API_BASE });
  return typedFetch<{ status: string; service: string; timestamp: string }>(
    url,
    { cache: "no-store" },
  );
}

// V2 onboarding reads selected scenarios and their derived resources from
// manifests. It never constructs service.json in the browser.
export function fetchV2Scenarios() {
  const url = buildApiUrl("/v2/scenarios", {
    baseUrl: API_BASE.replace(/\/v1$/, ""),
  });
  return typedFetch<V2ScenarioResponse>(url, { cache: "no-store" });
}

export function fetchV2Recommendation() {
  const url = buildApiUrl("/v2/recommendation", {
    baseUrl: API_BASE.replace(/\/v1$/, ""),
  });
  return typedFetch<V2Recommendation>(url, { cache: "no-store" });
}

export function fetchV2CoreSet(seed?: Iterable<string>) {
  const url = new URL(
    buildApiUrl("/v2/core-set", { baseUrl: API_BASE.replace(/\/v1$/, "") }),
  );
  if (seed) {
    for (const name of Array.from(seed).sort()) url.searchParams.append("seed", name);
  }
  return typedFetch<V2CoreSetResponse>(url.toString(), { cache: "no-store" });
}

export function fetchV2Readiness() {
  const url = buildApiUrl("/v2/readiness", {
    baseUrl: API_BASE.replace(/\/v1$/, ""),
  });
  return typedFetch<V2ReadinessResponse>(url, { cache: "no-store" });
}
export function fetchV2HostRequirements() {
  const url = buildApiUrl("/v2/host-requirements", {
    baseUrl: API_BASE.replace(/\/v1$/, ""),
  });
  return typedFetch<V2HostRequirementsResponse>(url, { cache: "no-store" });
}

export function fetchV2Closure() {
  const url = buildApiUrl("/v2/closure", {
    baseUrl: API_BASE.replace(/\/v1$/, ""),
  });
  return typedFetch<V2ClosureResponse>(url, { cache: "no-store" });
}

export function fetchV2Resources() {
  const url = buildApiUrl("/v2/resources", {
    baseUrl: API_BASE.replace(/\/v1$/, ""),
  });
  return typedFetch<V2ResourceResponse>(url, { cache: "no-store" });
}

export function provisionCredential(input: {
  logical_id: string;
  field: string;
  value: string;
}) {
  const url = buildApiUrl("/v2/credentials/provision", {
    baseUrl: API_BASE.replace(/\/v1$/, ""),
  });
  return typedFetch<{
    status: "provisioned";
    logical_id: string;
    field: string;
  }>(url, { method: "POST", body: JSON.stringify(input) });
}

/**
 * Records the operator's acceptance of the exact degraded set readiness reports
 * now. The digest identifies that set, so accepting one gap cannot authorise
 * completion over a different one later, and a reload cannot discard it.
 */
export function acknowledgeDegraded(readiness_digest: string) {
  const url = buildApiUrl("/v2/readiness/degraded-acknowledgement", {
    baseUrl: API_BASE.replace(/\/v1$/, ""),
  });
  return typedFetch<{ status: "acknowledged"; readiness_digest: string }>(url, {
    method: "POST",
    body: JSON.stringify({ readiness_digest }),
  });
}

export function applyOnboarding(signal?: AbortSignal) {
  const url = buildApiUrl("/v2/apply", {
    baseUrl: API_BASE.replace(/\/v1$/, ""),
  });
  return typedFetch<V2ApplyResponse>(url, {
    method: "POST",
    body: "{}",
    signal,
  });
}

export function fetchV2ApplyPlan() {
  const url = buildApiUrl("/v2/apply/plan", {
    baseUrl: API_BASE.replace(/\/v1$/, ""),
  });
  return typedFetch<V2ApplyPlanResponse>(url, { cache: "no-store" });
}

export function fetchV2ApplyStatus(runID: string, signal?: AbortSignal) {
  const url = buildApiUrl(`/v2/apply/${encodeURIComponent(runID)}`, {
    baseUrl: API_BASE.replace(/\/v1$/, ""),
  });
  return typedFetch<V2ApplyResponse>(url, { cache: "no-store", signal });
}

export function fetchV2Session() {
  const url = buildApiUrl("/v2/session", {
    baseUrl: API_BASE.replace(/\/v1$/, ""),
  });
  return typedFetch<V2SessionResponse>(url, { cache: "no-store" });
}

export function fetchV2Steps() {
  const url = buildApiUrl("/v2/steps", {
    baseUrl: API_BASE.replace(/\/v1$/, ""),
  });
  return typedFetch<V2StepsResponse>(url, { cache: "no-store" });
}

export function fetchOperatorInputs() {
  const url = buildApiUrl("/v2/operator-inputs", {
    baseUrl: API_BASE.replace(/\/v1$/, ""),
  });
  return typedFetch<OperatorInputQueue>(url, { cache: "no-store" });
}

export function fetchCapabilities() {
  const url = buildApiUrl("/v2/capabilities", {
    baseUrl: API_BASE.replace(/\/v1$/, ""),
  });
  return typedFetch<CapabilityStatusResponse>(url, { cache: "no-store" });
}

export interface CapabilityActionRequest {
  capability_id: string;
  idempotency_key?: string;
  confirm: boolean;
  inputs: Record<string, unknown>;
}

export function previewCapability(request: CapabilityActionRequest) {
  const url = buildApiUrl("/v2/capabilities/preview", {
    baseUrl: API_BASE.replace(/\/v1$/, ""),
  });
  return typedFetch<CapabilityPreview>(url, {
    method: "POST",
    body: JSON.stringify(request),
  });
}

export function applyCapability(request: CapabilityActionRequest) {
  const url = buildApiUrl("/v2/capabilities/apply", {
    baseUrl: API_BASE.replace(/\/v1$/, ""),
  });
  return typedFetch<CapabilityResult>(url, {
    method: "POST",
    body: JSON.stringify(request),
  });
}

export function resolveOperatorInputs(
  answers: Array<{ request_id: string; value: string }>,
) {
  const url = buildApiUrl("/v2/operator-inputs/resolve", {
    baseUrl: API_BASE.replace(/\/v1$/, ""),
  });
  return typedFetch<{ status: string; configuration_pending: boolean }>(url, {
    method: "POST",
    body: JSON.stringify(answers),
  });
}

export function saveV2SessionStep(step: number) {
  const url = buildApiUrl("/v2/session/step", {
    baseUrl: API_BASE.replace(/\/v1$/, ""),
  });
  return typedFetch<V2SessionResponse>(url, {
    method: "POST",
    body: JSON.stringify({ step }),
  });
}

export function fetchOperatorState() {
  const url = buildApiUrl("/operator-state", { baseUrl: API_BASE });
  return typedFetch<OperatorState>(url, { cache: "no-store" });
}

export function saveOperatorState(patch: OperatorStatePatch) {
  const url = buildApiUrl("/operator-state", {
    baseUrl: API_BASE.replace(/\/v1$/, "/v2"),
  });
  return typedFetch<OperatorState>(url, {
    method: "PATCH",
    headers: { "Content-Type": "application/merge-patch+json" },
    body: JSON.stringify(patch),
  });
}

export async function fetchResources() {
  const url = buildApiUrl("/resources", { baseUrl: API_BASE });
  const data = await typedFetch<{ resources: Resource[] } | Resource[]>(url, {
    cache: "no-store",
  });
  // API wraps resources in {count, resources: [...]}, unwrap if needed
  return Array.isArray(data) ? data : data.resources;
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
