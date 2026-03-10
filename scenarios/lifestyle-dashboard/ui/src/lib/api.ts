import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

// =============================================================================
// Error Types (matching API error semantics)
// =============================================================================

/** Error categories for recovery path selection */
export type ErrorCategory = "validation" | "not_found" | "conflict" | "internal" | "unavailable";

/** Structured API error from the backend */
export interface APIErrorResponse {
  error: boolean;
  category: ErrorCategory;
  code: string;
  message: string;
  details?: Record<string, unknown>;
  recovery?: string;
}

/** Custom error class for structured API errors */
export class APIError extends Error {
  category: ErrorCategory;
  code: string;
  details?: Record<string, unknown>;
  recovery?: string;
  status: number;

  constructor(response: APIErrorResponse, status: number) {
    super(response.message);
    this.name = "APIError";
    this.category = response.category;
    this.code = response.code;
    this.details = response.details;
    this.recovery = response.recovery;
    this.status = status;
  }

  /** Whether the error is recoverable by retrying */
  get isRetryable(): boolean {
    return this.category === "internal" || this.category === "unavailable";
  }

  /** Whether the error is due to invalid user input */
  get isValidation(): boolean {
    return this.category === "validation";
  }

  /** Whether the resource was not found */
  get isNotFound(): boolean {
    return this.category === "not_found";
  }
}

/**
 * Handles API response errors with structured error extraction.
 * Tries to parse the response body for structured error info.
 */
async function handleApiError(res: Response, fallbackMessage: string): Promise<never> {
  try {
    const body = await res.json();
    if (body && body.error && body.category) {
      throw new APIError(body as APIErrorResponse, res.status);
    }
    // Legacy error format
    throw new Error(body?.message || fallbackMessage);
  } catch (e) {
    if (e instanceof APIError) throw e;
    throw new Error(fallbackMessage);
  }
}

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
  dependencies?: Record<string, { connected: boolean; latency_ms?: number; error?: string; database?: string }>;
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
    await handleApiError(res, `API health check failed: ${res.status}`);
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
    await handleApiError(res, `Failed to fetch events: ${res.status}`);
  }

  return res.json();
}

export async function fetchEvent(id: string): Promise<Event> {
  const url = buildApiUrl(`/events/${id}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
  });

  if (!res.ok) {
    await handleApiError(res, `Failed to fetch event: ${res.status}`);
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
    await handleApiError(res, `Failed to create event: ${res.status}`);
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
    await handleApiError(res, `Failed to fetch domains: ${res.status}`);
  }

  return res.json();
}

export async function fetchDomain(name: string): Promise<Domain> {
  const url = buildApiUrl(`/domains/${name}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
  });

  if (!res.ok) {
    await handleApiError(res, `Failed to fetch domain: ${res.status}`);
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
    await handleApiError(res, `Failed to register domain: ${res.status}`);
  }

  return res.json();
}

export async function fetchDomainHealth(name: string): Promise<{ domain: string; status: string; last_check: string; message?: string }> {
  const url = buildApiUrl(`/domains/${name}/health`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
  });

  if (!res.ok) {
    await handleApiError(res, `Failed to check domain health: ${res.status}`);
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
    await handleApiError(res, `Failed to fetch timeline: ${res.status}`);
  }

  return res.json();
}

export async function fetchSummary(): Promise<Summary> {
  const url = buildApiUrl("/stats/summary", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
  });

  if (!res.ok) {
    await handleApiError(res, `Failed to fetch summary: ${res.status}`);
  }

  return res.json();
}

// =============================================================================
// Lifestyle Score API
// =============================================================================

export interface DomainScore {
  domain: string;
  display_name: string;
  score: number;
  weight: number;
  event_count: number;
}

export interface LifestyleScore {
  score: number;
  date: string;
  domain_scores: DomainScore[];
  trend: "up" | "down" | "stable";
  change_from_yesterday: number;
  data_quality: "good" | "limited" | "insufficient";
  message: string;
}

export interface ScoreHistoryEntry {
  date: string;
  score: number;
}

export interface ScoreResponse {
  current: LifestyleScore;
  history: ScoreHistoryEntry[];
}

export async function fetchScore(historyDays = 7): Promise<ScoreResponse> {
  const url = buildApiUrl(`/stats/score?history_days=${historyDays}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
  });

  if (!res.ok) {
    await handleApiError(res, `Failed to fetch score: ${res.status}`);
  }

  return res.json();
}

// =============================================================================
// Storage API (P0-006)
// =============================================================================

export interface DomainStorageInfo {
  domain: string;
  display_name: string;
  event_count: number;
}

export interface StorageInfo {
  database_size_bytes: number;
  total_events: number;
  total_domains: number;
  events_by_domain: DomainStorageInfo[];
  oldest_event?: string;
  newest_event?: string;
}

export interface CleanupRequest {
  domains?: string[];
  before?: string;
}

export interface CleanupResponse {
  deleted_events: number;
  domains_cleared: string[];
  message: string;
}

/** [REQ:LD-UI-STORAGE] Fetch storage information for settings page */
export async function fetchStorageInfo(): Promise<StorageInfo> {
  const url = buildApiUrl("/storage", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
  });

  if (!res.ok) {
    await handleApiError(res, `Failed to fetch storage info: ${res.status}`);
  }

  return res.json();
}

/** [REQ:LD-UI-STORAGE] Clean up events matching the request criteria */
export async function cleanupEvents(params: CleanupRequest = {}): Promise<CleanupResponse> {
  const url = buildApiUrl("/storage/events", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(params),
  });

  if (!res.ok) {
    await handleApiError(res, `Failed to cleanup events: ${res.status}`);
  }

  return res.json();
}

// =============================================================================
// Brief Types (P0-005)
// =============================================================================

export interface BriefSection {
  domain: string;
  display_name: string;
  priority: number;
  items: string[];
  event_count: number;
}

export interface Brief {
  type: "morning" | "evening";
  generated_at: string;
  date: string;
  summary: string;
  sections: BriefSection[];
  score?: number;
  score_trend?: "up" | "down" | "stable";
}

export interface BriefConfig {
  morning_hour: number;
  evening_hour: number;
}

export interface BriefResponse {
  brief: Brief;
  config: BriefConfig;
}

// =============================================================================
// Briefs API (P0-005)
// =============================================================================

/** [REQ:LD-BRIEF-MORNING] [REQ:LD-BRIEF-EVENING] Fetch current brief based on time of day */
export async function fetchCurrentBrief(): Promise<BriefResponse> {
  const url = buildApiUrl("/briefs/current", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
  });

  if (!res.ok) {
    await handleApiError(res, `Failed to fetch current brief: ${res.status}`);
  }

  return res.json();
}

/** [REQ:LD-BRIEF-MORNING] Fetch morning brief for a specific date */
export async function fetchMorningBrief(date?: string): Promise<BriefResponse> {
  const params = date ? `?date=${date}` : "";
  const url = buildApiUrl(`/briefs/morning${params}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
  });

  if (!res.ok) {
    await handleApiError(res, `Failed to fetch morning brief: ${res.status}`);
  }

  return res.json();
}

/** [REQ:LD-BRIEF-EVENING] Fetch evening brief for a specific date */
export async function fetchEveningBrief(date?: string): Promise<BriefResponse> {
  const params = date ? `?date=${date}` : "";
  const url = buildApiUrl(`/briefs/evening${params}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
  });

  if (!res.ok) {
    await handleApiError(res, `Failed to fetch evening brief: ${res.status}`);
  }

  return res.json();
}
