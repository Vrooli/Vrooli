import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

/** Build a full API URL from a path suffix, e.g. "/routes" → "http://…/api/v1/routes". */
function apiUrl(path: string): string {
  return buildApiUrl(path, { baseUrl: API_BASE });
}

async function apiFetch<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
    ...init,
  });
  if (!res.ok) {
    throw new Error(`API error ${res.status}: ${await res.text()}`);
  }
  return res.json() as Promise<T>;
}

// --- Types ---

export interface HealthResponse {
  status: string;
  service: string;
  timestamp: string;
  version: string;
  readiness: boolean;
}

export interface Route {
  id: number;
  subdomain: string;
  scenario_name: string;
  local_port: number;
  health_path: string;
  public_url: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface TunnelStatus {
  status: "healthy" | "degraded" | "unhealthy";
  systemd: string;
  ready: string;
  ready_latency_ms: number;
  score: number;
  message: string;
  checked_at: string;
}

export interface ProbeResult {
  route_id: number;
  subdomain: string;
  probe_type: "internal" | "external";
  status: "up" | "down" | "timeout" | "error";
  latency_ms: number;
  status_code?: number;
  error_msg?: string;
}

export interface ProbeResponse {
  results: ProbeResult[];
  summary: { total: number; up: number; down: number };
}

export interface AuditResult {
  subdomain: string;
  scenario_name: string;
  expected_port: number;
  actual_port?: number;
  status: "compliant" | "mismatch" | "missing_scenario" | "missing_port";
  detail?: string;
}

export interface AuditResponse {
  results: AuditResult[];
  total: number;
  violations: number;
  compliant: number;
}

export interface RecoveryEvent {
  id: number;
  trigger_type: "ready_failure" | "ha_connection_loss" | "manual";
  action: string;
  outcome: "success" | "failure" | "skipped";
  details?: string;
  created_at: string;
}

export interface RecoveryState {
  status: string;
  circuit_open: boolean;
  consecutive_failures: number;
  last_recovery_at?: string;
  backoff_retries: number;
}

export interface RouteInput {
  subdomain: string;
  scenario_name: string;
  local_port: number;
  health_path?: string;
  public_url?: string;
  enabled?: boolean;
}

// --- API Functions ---

export function fetchHealth(): Promise<HealthResponse> {
  return apiFetch<HealthResponse>(apiUrl("/health"));
}

export function fetchRoutes(): Promise<Route[]> {
  return apiFetch<Route[]>(apiUrl("/routes"));
}

export function fetchTunnelHealth(): Promise<TunnelStatus> {
  return apiFetch<TunnelStatus>(apiUrl("/tunnel/health"));
}

export function runProbes(): Promise<ProbeResponse> {
  return apiFetch<ProbeResponse>(apiUrl("/probes"), { method: "POST" });
}

export function fetchAudit(): Promise<AuditResponse> {
  return apiFetch<AuditResponse>(apiUrl("/audit/ports"));
}

export function fetchRecoveryEvents(): Promise<RecoveryEvent[]> {
  return apiFetch<RecoveryEvent[]>(apiUrl("/recovery/events"));
}

export function fetchRecoveryState(): Promise<RecoveryState> {
  return apiFetch<RecoveryState>(apiUrl("/recovery/state"));
}

export function triggerRecovery(force: boolean): Promise<RecoveryEvent> {
  return apiFetch<RecoveryEvent>(apiUrl("/recovery/trigger"), {
    method: "POST",
    body: JSON.stringify({ force }),
  });
}

export function resetCircuit(): Promise<{ status: string }> {
  return apiFetch<{ status: string }>(apiUrl("/recovery/circuit/reset"), {
    method: "POST",
  });
}

export function fetchProbeHistory(): Promise<ProbeResult[]> {
  return apiFetch<ProbeResult[]>(apiUrl("/probes/history"));
}

export function createRoute(input: RouteInput): Promise<Route> {
  return apiFetch<Route>(apiUrl("/routes"), {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function updateRoute(id: number, input: RouteInput): Promise<Route> {
  return apiFetch<Route>(apiUrl(`/routes/${id}`), {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export function deleteRoute(id: number): Promise<void> {
  return apiFetch<void>(apiUrl(`/routes/${id}`), { method: "DELETE" });
}

export function fetchRoute(id: number): Promise<Route> {
  return apiFetch<Route>(apiUrl(`/routes/${id}`));
}

// --- Detailed Health ---

export interface DetailedHealth {
  status: string;
  tunnel: {
    ready: string;
    systemd: string;
    score: number;
    ready_latency_ms?: number;
  };
  routes: {
    subdomain: string;
    scenario_name: string;
    enabled: boolean;
    internal_up?: boolean;
    external_up?: boolean;
  }[];
  timestamp: string;
}

export function fetchDetailedHealth(): Promise<DetailedHealth> {
  return apiFetch<DetailedHealth>(apiUrl("/health/detailed"));
}

// --- Metrics ---

export interface MetricsRecord {
  id: number;
  ha_connections: number;
  request_errors: number;
  active_streams: number;
  smoothed_rtt_ms: number;
  scraped_at: string;
}

export function fetchMetricsHistory(hours = 24): Promise<MetricsRecord[]> {
  return apiFetch<MetricsRecord[]>(apiUrl(`/metrics/history?hours=${hours}`));
}

export function fetchMetricsLatest(): Promise<MetricsRecord | { status: string }> {
  return apiFetch<MetricsRecord | { status: string }>(apiUrl("/metrics/latest"));
}
