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

export interface PassRate {
  passing: number;
  total: number;
  rate: number;
  points: number;
}

export interface QualityScore {
  score: number;
  max: number;
  disabled?: boolean;
  requirement_pass_rate: PassRate;
  target_pass_rate: PassRate;
  test_pass_rate: PassRate;
}

export interface CoverageRatio {
  ratio: number;
  points: number;
}

export interface DepthScoreDetail {
  avg_depth: number;
  points: number;
}

export interface CoverageScore {
  score: number;
  max: number;
  disabled?: boolean;
  test_coverage_ratio: CoverageRatio;
  depth_score: DepthScoreDetail;
}

export interface QuantityMetric {
  count: number;
  threshold: string;
  points: number;
}

export interface QuantityScore {
  score: number;
  max: number;
  disabled?: boolean;
  requirements: QuantityMetric;
  targets: QuantityMetric;
  tests: QuantityMetric;
}

export interface TemplateCheckResult {
  is_template: boolean;
  penalty: number;
  points: number;
}

export interface ComponentComplexity {
  file_count: number;
  component_count?: number;
  page_count?: number;
  threshold: string;
  points: number;
}

export interface APIIntegration {
  endpoint_count: number;
  total_endpoints?: number;
  points: number;
}

export interface RoutingScore {
  has_routing: boolean;
  route_count: number;
  points: number;
}

export interface CodeVolume {
  total_loc: number;
  points: number;
}

export interface UIScore {
  score: number;
  max: number;
  disabled?: boolean;
  template_check: TemplateCheckResult;
  component_complexity: ComponentComplexity;
  api_integration: APIIntegration;
  routing: RoutingScore;
  code_volume: CodeVolume;
}

export interface ScoreBreakdown {
  base_score: number;
  validation_penalty: number;
  score: number;
  classification: string;
  classification_description: string;
  quality: QualityScore;
  coverage: CoverageScore;
  quantity: QuantityScore;
  ui: UIScore;
}

export interface Recommendation {
  priority: number;
  description: string;
  impact?: number;
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

export interface ValidationQualityAnalysis {
  has_issues: boolean;
  issue_count: number;
  total_penalty: number;
  overall_severity: string;
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
  validation_analysis?: ValidationQualityAnalysis;
  recalculated?: boolean;
  snapshot_id?: number;
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

// Bulk Operations
export interface BulkRefreshResult {
  result: {
    total: number;
    successful: number;
    failed: number;
    scenarios: Array<{
      scenario: string;
      category: string;
      score: number;
      classification: string;
      delta: number;
      success: boolean;
      error?: string;
    }>;
  };
}

export async function refreshAllScores(): Promise<BulkRefreshResult> {
  return apiCall<BulkRefreshResult>("/scores/refresh-all", { method: "POST" });
}
