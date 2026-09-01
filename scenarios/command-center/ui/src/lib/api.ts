import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

// API base with /api/v1 suffix appended.
const API_BASE = resolveApiBase({ appendSuffix: true });

export type DataSourceStatus = "live" | "partial" | "gap";
export type Coverage = "NOW" | "IN-REACH" | "MISSING" | "UNREGISTERED";
export type Trust = "VALID" | "CACHED" | "UNAVAILABLE" | "UNTRUSTED";
export type Empirical = "NONE" | "PENDING" | "HIT" | "MISS" | "UNMEASURABLE";
export type UpstreamSource = "swarm" | "vrooli" | "lpbs" | "none";

export interface MetricEntry {
  id: string;
  label: string;
  dataSource: DataSourceStatus;
  upstreamSource: UpstreamSource;
  description?: string;
	whatIsNeeded?: string | null;
	unit?: string;
	format?: string;
	value?: number | null;
	observedAt?: string | null;
	ttlSeconds?: number;
	coverage?: Coverage;
	trust?: Trust;
	empirical?: Empirical;
	source?: { team?: string; binding?: string; instrumentStatus?: string; instrumentArchetype?: string };
	owner?: string;
	sample?: { value: number; series: number[]; basis: string } | null;
	firstObservedMissing?: string | null;
	gapOpenDays?: number | null;
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

export interface BoardRoom { id: string; title: string; theme?: string; composition?: string; metricIds?: string[] }
export interface BoardResponse { schemaVersion: string; generatedAt: string; rooms: BoardRoom[]; denominator: { outcomeCategories: number; confidence: string; rationale: string }; sources: Array<Record<string, unknown>> }
export interface RoomResponse { room: BoardRoom; readings: MetricEntry[]; sources: Record<string, SourceMetadata> }
export interface FocusEntry { kind: string; owner: string; reason: string; metricId?: string; rankReason: string }
export interface FocusResponse { generatedAt: string; entries: FocusEntry[] }
export interface OpenLoopResponse { generatedAt: string; missing: MetricEntry[]; unregistered: MetricEntry[]; self: Array<Record<string, unknown>> }

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
export function fetchBoard(): Promise<BoardResponse> { return getJSON<BoardResponse>("/board"); }
export function fetchRoom(id: string, samples = "mark"): Promise<RoomResponse> { return getJSON<RoomResponse>(`/rooms/${id}?samples=${samples}`); }
export function fetchFocus(): Promise<FocusResponse> { return getJSON<FocusResponse>("/focus"); }
export function fetchOpenLoop(): Promise<OpenLoopResponse> { return getJSON<OpenLoopResponse>("/open-loop"); }
