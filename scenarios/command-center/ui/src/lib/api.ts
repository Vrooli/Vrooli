import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

// API base with /api/v1 suffix appended.
const API_BASE = resolveApiBase({ appendSuffix: true });

export type DataSourceStatus = "live" | "partial" | "gap";
export type UpstreamSource = "swarm" | "vrooli" | "lpbs" | "none";

export interface MetricEntry {
  id: string;
  label: string;
  dataSource: DataSourceStatus;
  upstreamSource: UpstreamSource;
  description?: string;
  whatIsNeeded?: string | null;
}

export interface SourceMetadata {
  from_cache: boolean;
  staleness_ts: string | null;
}

export interface DashboardResponse {
  dashboard: string;
  generated_at: string;
  metrics: MetricEntry[];
  sources: Record<string, SourceMetadata>;
}

export interface GapsResponse {
  generated_at: string;
  dashboards: Record<string, MetricEntry[]>;
}

export interface HealthResponse {
  status: string;
  service: string;
  timestamp: string;
}

async function getJSON<T>(path: string): Promise<T> {
  const url = buildApiUrl(path, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    throw new Error(`API request failed: ${res.status.toString()} ${res.statusText}`);
  }
  return res.json() as Promise<T>;
}

export function fetchHealth(): Promise<HealthResponse> {
  return getJSON<HealthResponse>("/health");
}

export function fetchDashboard(id: string): Promise<DashboardResponse> {
  return getJSON<DashboardResponse>(`/dashboards/${id}`);
}

export function fetchGaps(): Promise<GapsResponse> {
  return getJSON<GapsResponse>("/gaps");
}
