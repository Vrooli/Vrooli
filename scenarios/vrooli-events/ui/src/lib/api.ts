// DOC: docs/reference/api-endpoints.md
// DOC: docs/internal/ASSUMPTIONS.md
import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });
const API_ROOT = resolveApiBase({ appendSuffix: false });

/** Type-safe fetch wrapper that validates response and returns parsed JSON. */
async function typedFetch<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init);
  if (!res.ok) throw new Error(`Request failed: ${res.status}`);
  return await res.json();
}

const JSON_HEADERS = { "Content-Type": "application/json" } as const;
const GET_OPTS = { headers: JSON_HEADERS, cache: "no-store" as const };

export interface HealthResponse {
  status: string;
  service: string;
  timestamp: string;
  readiness: boolean;
  subscribers: number;
  store: {
    totalEvents: number;
    totalPayloadBytes: number;
  };
}

export interface EventEnvelope {
  eventId: string;
  sourceScenario: string;
  targetScenario?: string;
  eventType: string;
  correlationId?: string;
  metadata?: Record<string, string>;
  payload?: unknown;
  createdAt?: string;
}

export async function fetchHealth(): Promise<HealthResponse> {
  const url = buildApiUrl("/health", { baseUrl: API_ROOT });
  return typedFetch<HealthResponse>(url, GET_OPTS);
}

export interface QueryParams {
  type?: string;
  source?: string;
  correlationId?: string;
  since?: number;
  limit?: number;
}

export async function fetchEvents(params: QueryParams = {}): Promise<EventEnvelope[]> {
  const search = new URLSearchParams();
  if (params.type) search.set("type", params.type);
  if (params.source) search.set("source", params.source);
  if (params.correlationId) search.set("correlation_id", params.correlationId);
  if (params.since !== undefined) search.set("since", String(params.since));
  if (params.limit !== undefined) search.set("limit", String(params.limit));

  const qs = search.toString();
  const base = buildApiUrl("/events", { baseUrl: API_BASE });
  const url = qs ? `${base}?${qs}` : base;

  return typedFetch<EventEnvelope[]>(url, GET_OPTS);
}

export interface SSEOptions {
  type?: string;
  source?: string;
  target?: string;
  onEvent: (event: EventEnvelope) => void;
  onHeartbeat?: () => void;
  onError?: (err: Event) => void;
}

// ── Policy types ──────────────────────────────────────────────────────
export interface PolicyRule {
  id: number;
  rule_type: string;
  source_scenario: string;
  target_scenario: string;
  endpoint_pattern?: string;
  effect?: string;
  priority: number;
  enabled: boolean;
  max_requests?: number;
  window_seconds?: number;
  burst_allowance?: number;
  failure_threshold?: number;
  cooldown_seconds?: number;
  success_threshold?: number;
  created_at: string;
  updated_at: string;
}

export interface PolicyViolation {
  id: number;
  timestamp: string;
  source_scenario: string;
  target_scenario: string;
  endpoint: string;
  rule_id: number;
  rule_type: string;
  reason: string;
}

// ── Subscription types ────────────────────────────────────────────────
export interface SubscriptionData {
  id: number;
  name: string;
  owner_scenario: string;
  event_pattern: string;
  source_filter?: string;
  delivery_type: string;
  delivery_target: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface SubscriptionHealth {
  subscription_id: number;
  total_delivered: number;
  total_failed: number;
  consecutive_failures: number;
  last_delivered_at?: string;
  last_failed_at?: string;
  status: string;
}

// ── Policy API ────────────────────────────────────────────────────────
export async function fetchPolicy(id: number): Promise<PolicyRule> {
  return typedFetch<PolicyRule>(buildApiUrl(`/policies/${id}`, { baseUrl: API_BASE }), GET_OPTS);
}

export async function fetchPolicies(): Promise<PolicyRule[]> {
  return typedFetch<PolicyRule[]>(buildApiUrl("/policies", { baseUrl: API_BASE }), GET_OPTS);
}

export async function createPolicy(
  rule: Omit<PolicyRule, "id" | "created_at" | "updated_at">,
): Promise<{ id: number }> {
  return typedFetch<{ id: number }>(buildApiUrl("/policies", { baseUrl: API_BASE }), {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify(rule),
  });
}

export async function updatePolicy(id: number, rule: Partial<PolicyRule>): Promise<void> {
  await typedFetch<void>(buildApiUrl(`/policies/${id}`, { baseUrl: API_BASE }), {
    method: "PUT",
    headers: JSON_HEADERS,
    body: JSON.stringify(rule),
  });
}

export async function deletePolicy(id: number): Promise<void> {
  await typedFetch<void>(buildApiUrl(`/policies/${id}`, { baseUrl: API_BASE }), { method: "DELETE" });
}

export async function fetchViolations(): Promise<PolicyViolation[]> {
  return typedFetch<PolicyViolation[]>(buildApiUrl("/policies/violations", { baseUrl: API_BASE }), GET_OPTS);
}

export async function overrideCircuitBreaker(
  id: number,
  state: string,
  ttl_seconds: number,
): Promise<void> {
  await typedFetch<void>(buildApiUrl(`/policies/${id}/override`, { baseUrl: API_BASE }), {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify({ state, ttl_seconds }),
  });
}

// ── Subscription API ──────────────────────────────────────────────────
export async function fetchSubscription(id: number): Promise<SubscriptionData> {
  return typedFetch<SubscriptionData>(buildApiUrl(`/subscriptions/${id}`, { baseUrl: API_BASE }), GET_OPTS);
}

export async function fetchSubscriptions(): Promise<SubscriptionData[]> {
  return typedFetch<SubscriptionData[]>(buildApiUrl("/subscriptions", { baseUrl: API_BASE }), GET_OPTS);
}

export async function fetchSubscriptionHealth(id: number): Promise<SubscriptionHealth> {
  return typedFetch<SubscriptionHealth>(buildApiUrl(`/subscriptions/${id}/health`, { baseUrl: API_BASE }), GET_OPTS);
}

export async function createSubscription(
  sub: Omit<SubscriptionData, "id" | "created_at" | "updated_at">,
): Promise<{ id: number }> {
  return typedFetch<{ id: number }>(buildApiUrl("/subscriptions", { baseUrl: API_BASE }), {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify(sub),
  });
}

export async function deleteSubscription(id: number): Promise<void> {
  await typedFetch<void>(buildApiUrl(`/subscriptions/${id}`, { baseUrl: API_BASE }), { method: "DELETE" });
}

export function subscribeSSE(opts: SSEOptions): () => void {
  const search = new URLSearchParams();
  if (opts.type) search.set("type", opts.type);
  if (opts.source) search.set("source", opts.source);
  if (opts.target) search.set("target", opts.target);

  const qs = search.toString();
  const base = buildApiUrl("/events/subscribe", { baseUrl: API_BASE });
  const url = qs ? `${base}?${qs}` : base;

  const es = new EventSource(url);

  function handleSSEData(e: MessageEvent): void {
    try {
      const raw: string = typeof e.data === "string" ? e.data : String(e.data);
      const data: EventEnvelope = JSON.parse(raw);
      opts.onEvent(data);
    } catch (err) {
      const preview = typeof e.data === "string" ? e.data.slice(0, 120) : "[non-string data]";
      console.warn("[SSE] Malformed event data:", preview, err);
    }
  }

  // Listen for named SSE event types (server sends `event: <type>`)
  es.addEventListener("message", handleSSEData);

  // Fallback for unnamed SSE events
  es.onmessage = handleSSEData;

  es.onerror = (err) => {
    console.warn("[SSE] Connection error — EventSource will auto-reconnect", err);
    opts.onError?.(err);
  };

  return () => es.close();
}
