import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

// Simple! Just specify if you want the /api/v1 suffix
const API_BASE = resolveApiBase({ appendSuffix: true });

// === Types ===

export interface HealthResponse {
  status: string;
  service: string;
  timestamp: string;
  version?: string;
  readiness?: boolean;
}

export interface MetricCounts {
  total: number;
  passing: number;
}

export interface UIMetrics {
  is_template: boolean;
  file_count: number;
  route_count: number;
  has_routing: boolean;
  api_beyond_health: number;
  total_loc: number;
}

export interface Metrics {
  scenario: string;
  category: string;
  requirements: MetricCounts;
  targets: MetricCounts;
  tests: MetricCounts;
  ui?: UIMetrics;
}

export interface ScoreBreakdown {
  score: number;
  classification: string;
  quality: number;
  coverage: number;
  quantity: number;
  ui: number;
  penalties: number;
  details: {
    quality: Record<string, number>;
    coverage: Record<string, number>;
    quantity: Record<string, number>;
    ui: Record<string, number>;
    penalties: Record<string, number>;
  };
}

export interface Recommendation {
  area: string;
  action: string;
  impact: number;
  priority: number;
}

export interface ScenarioScore {
  scenario: string;
  category: string;
  score: number;
  classification: string;
}

export interface ScoresListResponse {
  scenarios: ScenarioScore[];
  total: number;
  calculated_at: string;
}

export interface ScoreDetailResponse {
  scenario: string;
  category: string;
  score: number;
  classification: string;
  breakdown: ScoreBreakdown;
  metrics: Metrics;
  recommendations: Recommendation[];
  calculated_at: string;
}

export interface CollectorStatus {
  name: string;
  status: "ok" | "degraded" | "failed";
  last_success?: string;
  last_failure?: string;
  error?: string;
}

export interface HealthCollectorsResponse {
  status: string;
  collectors: Record<string, CollectorStatus>;
  summary: {
    healthy: number;
    degraded: number;
    failed: number;
    total: number;
  };
  checked_at: string;
}

export interface CircuitBreakerStatus {
  name: string;
  state: "closed" | "open" | "half_open";
  failure_count: number;
  last_failure?: string;
  last_success?: string;
}

export interface CircuitBreakersResponse {
  breakers: CircuitBreakerStatus[];
  stats: {
    total: number;
    closed: number;
    open: number;
    half_open: number;
  };
  tripped: string[];
}

export interface PresetInfo {
  name: string;
  description: string;
}

export interface PresetsListResponse {
  presets: PresetInfo[];
  total: number;
}

export interface ScoringConfig {
  components: {
    quality: { enabled: boolean };
    coverage: { enabled: boolean };
    quantity: { enabled: boolean };
    ui: { enabled: boolean };
  };
  weights?: {
    quality: number;
    coverage: number;
    quantity: number;
    ui: number;
  };
}

export interface ScenarioConfigResponse {
  scenario: string;
  effective: ScoringConfig;
  override: {
    scenario: string;
    enabled: boolean;
    preset?: string;
    overrides?: Partial<ScoringConfig>;
  } | null;
}

export interface HistorySnapshot {
  id: number;
  scenario: string;
  score: number;
  classification: string;
  breakdown: ScoreBreakdown;
  created_at: string;
}

export interface HistoryResponse {
  scenario: string;
  snapshots: HistorySnapshot[];
  count: number;
  total: number;
  limit: number;
  fetched_at: string;
}

export interface TrendAnalysis {
  trend: "improving" | "declining" | "stalled" | "stable" | "insufficient_data";
  delta: number;
  change_rate: number;
  snapshots_analyzed: number;
  latest_score: number;
  earliest_score: number;
  is_stalled: boolean;
}

export interface TrendResponse {
  scenario: string;
  analysis: TrendAnalysis;
  analyzed_at: string;
}

// === API Functions ===

async function apiCall<T>(path: string, options?: RequestInit): Promise<T> {
  const url = buildApiUrl(path, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(`API error ${res.status}: ${text || res.statusText}`);
  }

  return res.json() as Promise<T>;
}

// Health
export async function fetchHealth(): Promise<HealthResponse> {
  return apiCall<HealthResponse>("/health");
}

// Scores
export async function fetchScores(): Promise<ScoresListResponse> {
  return apiCall<ScoresListResponse>("/scores");
}

export async function fetchScoreDetail(scenario: string): Promise<ScoreDetailResponse> {
  return apiCall<ScoreDetailResponse>(`/scores/${encodeURIComponent(scenario)}`);
}

export async function calculateScore(scenario: string): Promise<ScoreDetailResponse> {
  return apiCall<ScoreDetailResponse>(`/scores/${encodeURIComponent(scenario)}/calculate`, {
    method: "POST",
  });
}

// Health & Circuit Breakers
export async function fetchCollectorHealth(): Promise<HealthCollectorsResponse> {
  return apiCall<HealthCollectorsResponse>("/health/collectors");
}

export async function fetchCircuitBreakers(): Promise<CircuitBreakersResponse> {
  return apiCall<CircuitBreakersResponse>("/health/circuit-breaker");
}

export async function resetCircuitBreaker(collector?: string): Promise<{ success: boolean; message: string }> {
  const path = collector
    ? `/health/circuit-breaker/${encodeURIComponent(collector)}/reset`
    : "/health/circuit-breaker/reset";
  return apiCall<{ success: boolean; message: string }>(path, { method: "POST" });
}

// Configuration
export async function fetchConfig(): Promise<ScoringConfig> {
  const res = await apiCall<{ scoring: ScoringConfig }>("/config");
  return res.scoring as unknown as ScoringConfig;
}

export async function fetchPresets(): Promise<PresetsListResponse> {
  return apiCall<PresetsListResponse>("/config/presets");
}

export async function applyPreset(name: string): Promise<{ success: boolean; message: string }> {
  return apiCall<{ success: boolean; message: string }>(`/config/presets/${encodeURIComponent(name)}/apply`, {
    method: "POST",
  });
}

export async function fetchScenarioConfig(scenario: string): Promise<ScenarioConfigResponse> {
  return apiCall<ScenarioConfigResponse>(`/config/scenarios/${encodeURIComponent(scenario)}`);
}

// History & Trends
export async function fetchHistory(scenario: string, limit = 30): Promise<HistoryResponse> {
  return apiCall<HistoryResponse>(`/scores/${encodeURIComponent(scenario)}/history?limit=${limit}`);
}

export async function fetchTrends(scenario: string): Promise<TrendResponse> {
  return apiCall<TrendResponse>(`/scores/${encodeURIComponent(scenario)}/trends`);
}

// Recommendations
export async function fetchRecommendations(scenario: string): Promise<{
  scenario: string;
  current_score: number;
  recommendations: Recommendation[];
  total_impact: number;
  potential_score: number;
}> {
  return apiCall(`/recommendations/${encodeURIComponent(scenario)}`);
}
