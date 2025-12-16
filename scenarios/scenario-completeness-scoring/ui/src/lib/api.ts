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
  // Partial result info (when some collectors failed)
  partial?: boolean;
  confidence?: number;
  missing_collectors?: string[];
}

// Degradation info when scenarios are skipped or have partial data
export interface DegradationInfo {
  skipped: Array<{
    scenario: string;
    error: string;
    reason: string;
  }>;
  skipped_count: number;
  partial_count: number;
  is_complete: boolean;
  message: string;
}

export interface ScoresListResponse {
  scenarios: ScenarioScore[];
  total: number;
  calculated_at: string;
  // Optional degradation info when there are issues
  degradation?: DegradationInfo;
}

export interface ValidationQualityAnalysis {
  has_issues: boolean;
  issue_count: number;
  total_penalty: number;
  overall_severity: string;
}

// Partial result info for score detail responses
export interface PartialResultInfo {
  is_complete: boolean;
  confidence: number;
  available: Record<string, boolean>;
  missing: string[];
  collector_errors?: Array<{
    collector: string;
    message: string;
    severity: string;
  }>;
  message: string;
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
  // Partial result info when some collectors failed
  partial_result?: PartialResultInfo;
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

export interface ScoringConfig {
  version: string;
  components: {
    quality: {
      enabled: boolean;
      requirement_pass_rate: boolean;
      target_pass_rate: boolean;
      test_pass_rate: boolean;
    };
    coverage: {
      enabled: boolean;
      test_coverage_ratio: boolean;
      requirement_depth: boolean;
    };
    quantity: {
      enabled: boolean;
      requirements: boolean;
      targets: boolean;
      tests: boolean;
    };
    ui: {
      enabled: boolean;
      template_detection: boolean;
      component_complexity: boolean;
      api_integration: boolean;
      routing: boolean;
      code_volume: boolean;
    };
  };
  penalties: {
    enabled: boolean;
    insufficient_test_coverage: boolean;
    invalid_test_location: boolean;
    monolithic_test_files: boolean;
    single_layer_validation: boolean;
    target_mapping_ratio: boolean;
    superficial_test_implementation: boolean;
    manual_validations: boolean;
  };
  circuit_breaker: {
    enabled: boolean;
    fail_threshold: number;
    retry_interval_seconds: number;
    auto_disable: boolean;
  };
  weights: {
    quality: number;
    coverage: number;
    quantity: number;
    ui: number;
  };
}

export interface ConfigResponse {
  config: ScoringConfig;
  effective_weights: ScoringConfig["weights"];
}

export interface ConfigSchema {
  version: string;
  sections: Array<{
    key: string;
    title: string;
    description: string;
    enabled_path?: string;
    weight_path?: string;
    fields: Array<{
      key: string;
      path: string;
      type: "boolean" | "integer";
      title: string;
      description: string;
      min?: number;
      max?: number;
    }>;
  }>;
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

/**
 * Retry configuration for transient failures
 * [REQ:SCS-CORE-003] Graceful degradation with automatic retry
 */
interface RetryConfig {
  maxRetries: number;
  baseDelay: number; // ms
  maxDelay: number;  // ms
  retryableStatuses: number[];
}

const DEFAULT_RETRY_CONFIG: RetryConfig = {
  maxRetries: 3,
  baseDelay: 500,
  maxDelay: 5000,
  retryableStatuses: [408, 429, 500, 502, 503, 504], // Timeout, Rate limit, Server errors
};

/**
 * Sleep utility for retry delays
 */
function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Calculate exponential backoff delay with jitter
 */
function getRetryDelay(attempt: number, config: RetryConfig): number {
  const exponentialDelay = Math.min(
    config.baseDelay * Math.pow(2, attempt),
    config.maxDelay
  );
  // Add jitter (±25%) to prevent thundering herd
  const jitter = exponentialDelay * 0.25 * (Math.random() * 2 - 1);
  return Math.round(exponentialDelay + jitter);
}

/**
 * Core API call function with automatic retry for transient failures
 * [REQ:SCS-CORE-003] Graceful degradation - retries transient network failures
 */
async function apiCall<T>(
  path: string,
  options?: RequestInit,
  retryConfig: Partial<RetryConfig> = {}
): Promise<T> {
  const config = { ...DEFAULT_RETRY_CONFIG, ...retryConfig };
  const url = buildApiUrl(path, { baseUrl: API_BASE });

  let lastError: Error | null = null;

  for (let attempt = 0; attempt <= config.maxRetries; attempt++) {
    try {
      const res = await fetch(url, {
        headers: { "Content-Type": "application/json" },
        ...options,
      });

      if (!res.ok) {
        // Check if this is a retryable status
        if (config.retryableStatuses.includes(res.status) && attempt < config.maxRetries) {
          const delay = getRetryDelay(attempt, config);
          console.warn(`API request failed with ${res.status}, retrying in ${delay}ms (attempt ${attempt + 1}/${config.maxRetries})`);
          await sleep(delay);
          continue;
        }

        // Non-retryable error or max retries exceeded
        const text = await res.text().catch(() => "");
        throw new Error(`API error ${res.status}: ${text || res.statusText}`);
      }

      return res.json() as Promise<T>;
    } catch (error) {
      lastError = error instanceof Error ? error : new Error(String(error));

      // Only retry network errors, not response errors
      if (error instanceof TypeError && error.message.includes('fetch') && attempt < config.maxRetries) {
        const delay = getRetryDelay(attempt, config);
        console.warn(`Network error, retrying in ${delay}ms (attempt ${attempt + 1}/${config.maxRetries}):`, error.message);
        await sleep(delay);
        continue;
      }

      throw lastError;
    }
  }

  throw lastError || new Error('Max retries exceeded');
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
export async function fetchConfig(): Promise<ConfigResponse> {
  return apiCall<ConfigResponse>("/config");
}

export async function updateConfig(config: ScoringConfig): Promise<{ success: boolean; message: string; config: ScoringConfig }> {
  return apiCall<{ success: boolean; message: string; config: ScoringConfig }>("/config", {
    method: "PUT",
    body: JSON.stringify(config),
  });
}

export async function resetConfig(): Promise<{ success: boolean; message: string; config: ScoringConfig }> {
  return apiCall<{ success: boolean; message: string; config: ScoringConfig }>("/config/reset", {
    method: "POST",
  });
}

export async function fetchConfigSchema(): Promise<{ schema: ConfigSchema }> {
  return apiCall<{ schema: ConfigSchema }>("/config/schema");
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
