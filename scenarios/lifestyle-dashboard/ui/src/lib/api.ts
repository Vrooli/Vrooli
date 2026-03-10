import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

// =============================================================================
// Types
// =============================================================================

export interface Event {
  id: string;
  timestamp: string;
  domain: string;
  event_type: string;
  payload: Record<string, unknown>;
  is_intervention: boolean;
  hypothesis_id?: string;
  created_at: string;
}

export interface Domain {
  name: string;
  display_name: string;
  description?: string;
  capabilities?: string[];
  status: "active" | "inactive" | "unhealthy";
  health_url?: string;
  last_health_at?: string;
  registered_at: string;
  updated_at: string;
}

export interface TimelineEntry {
  day: string;
  domain: string;
  count: number;
}

export interface Summary {
  total_events: number;
  active_domains: number;
  events_by_domain: Array<{ domain: string; count: number }>;
  last_event_at?: string;
}

export interface HealthResponse {
  status: string;
  service: string;
  timestamp: string;
  readiness: boolean;
  version?: string;
  uptime_seconds?: number;
  dependencies?: Record<string, { connected: boolean; latency_ms?: number; error?: string }>;
  metrics?: Record<string, number>;
}

// =============================================================================
// Health API
// =============================================================================

export async function fetchHealth(): Promise<HealthResponse> {
  const url = buildApiUrl("/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`API health check failed: ${res.status}`);
  }

  return res.json();
}

// =============================================================================
// Events API
// =============================================================================

export interface QueryEventsParams {
  domain?: string;
  event_type?: string;
  start?: string;
  end?: string;
  limit?: number;
}

export async function fetchEvents(params: QueryEventsParams = {}): Promise<{ events: Event[]; count: number }> {
  const searchParams = new URLSearchParams();
  if (params.domain) searchParams.set("domain", params.domain);
  if (params.event_type) searchParams.set("event_type", params.event_type);
  if (params.start) searchParams.set("start", params.start);
  if (params.end) searchParams.set("end", params.end);
  if (params.limit) searchParams.set("limit", String(params.limit));

  const url = buildApiUrl(`/events?${searchParams}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch events: ${res.status}`);
  }

  return res.json();
}

export async function fetchEvent(id: string): Promise<Event> {
  const url = buildApiUrl(`/events/${id}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch event: ${res.status}`);
  }

  return res.json();
}

export interface CreateEventParams {
  domain: string;
  event_type: string;
  payload?: Record<string, unknown>;
  timestamp?: string;
  is_intervention?: boolean;
  hypothesis_id?: string;
}

export async function createEvent(params: CreateEventParams): Promise<Event> {
  const url = buildApiUrl("/events", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(params),
  });

  if (!res.ok) {
    throw new Error(`Failed to create event: ${res.status}`);
  }

  return res.json();
}

// =============================================================================
// Domains API
// =============================================================================

export async function fetchDomains(): Promise<{ domains: Domain[]; count: number }> {
  const url = buildApiUrl("/domains", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch domains: ${res.status}`);
  }

  return res.json();
}

export async function fetchDomain(name: string): Promise<Domain> {
  const url = buildApiUrl(`/domains/${name}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch domain: ${res.status}`);
  }

  return res.json();
}

export interface RegisterDomainParams {
  name: string;
  display_name: string;
  description?: string;
  capabilities?: string[];
  health_url?: string;
}

export async function registerDomain(params: RegisterDomainParams): Promise<Domain> {
  const url = buildApiUrl("/domains", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(params),
  });

  if (!res.ok) {
    throw new Error(`Failed to register domain: ${res.status}`);
  }

  return res.json();
}

export async function fetchDomainHealth(name: string): Promise<{ domain: string; status: string; last_check: string; message?: string }> {
  const url = buildApiUrl(`/domains/${name}/health`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
  });

  if (!res.ok) {
    throw new Error(`Failed to check domain health: ${res.status}`);
  }

  return res.json();
}

// Alias for backward compatibility
export const checkDomainHealth = fetchDomainHealth;

// =============================================================================
// Statistics API
// =============================================================================

export async function fetchTimeline(days = 7): Promise<{ timeline: TimelineEntry[]; days: string }> {
  const url = buildApiUrl(`/stats/timeline?days=${days}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch timeline: ${res.status}`);
  }

  return res.json();
}

export async function fetchSummary(): Promise<Summary> {
  const url = buildApiUrl("/stats/summary", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch summary: ${res.status}`);
  }

  return res.json();
}
