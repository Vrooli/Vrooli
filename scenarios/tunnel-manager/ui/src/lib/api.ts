import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

function apiUrl(path: string): string {
  return buildApiUrl(path, { baseUrl: API_BASE });
}

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(apiUrl(path), {
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

// --- API Functions ---

export function fetchHealth(): Promise<HealthResponse> {
  return apiFetch<HealthResponse>("/health");
}

export function fetchRoutes(): Promise<Route[]> {
  return apiFetch<Route[]>("/routes");
}

export function fetchTunnelHealth(): Promise<TunnelStatus> {
  return apiFetch<TunnelStatus>("/tunnel/health");
}

export function runProbes(): Promise<ProbeResponse> {
  return apiFetch<ProbeResponse>("/probes", { method: "POST" });
}

export function fetchAudit(): Promise<AuditResponse> {
  return apiFetch<AuditResponse>("/audit/ports");
}
