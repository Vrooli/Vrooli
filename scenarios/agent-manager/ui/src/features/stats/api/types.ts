// Stats API response types matching the backend handlers

// =============================================================================
// Filter Types
// =============================================================================

export type TimePreset = "6h" | "12h" | "24h" | "7d" | "30d";

export interface TimeWindow {
  start: string; // RFC3339
  end: string; // RFC3339
}

export interface StatsFilter {
  preset?: TimePreset;
  start?: string;
  end?: string;
  runnerType?: string;
  profileId?: string;
  model?: string;
  tagPrefix?: string;
}

// =============================================================================
// Response Types
// =============================================================================

export interface RunStatusCounts {
  pending: number;
  running: number;
  complete: number;
  failed: number;
  cancelled: number;
  needsReview: number;
  total: number;
}

export interface DurationStats {
  avgMs: number;
  p50Ms: number;
  p95Ms: number;
  p99Ms: number;
  minMs: number;
  maxMs: number;
  count: number;
}

export interface CostStats {
  totalCostUsd: number;
  avgCostUsd: number;
  inputTokens: number;
  outputTokens: number;
  cacheReadTokens: number;
  totalTokens: number;
}

export interface RunnerBreakdown {
  runnerType: string;
  runCount: number;
  successCount: number;
  failedCount: number;
  totalCostUsd: number;
  avgDurationMs: number;
}

export interface ProfileBreakdown {
  profileId: string;
  profileName: string;
  runCount: number;
  successCount: number;
  failedCount: number;
  totalCostUsd: number; // Note: backend sends totalCostUsd (lowercase 'd')
}

export interface ModelBreakdown {
  model: string;
  runCount: number;
  successCount: number;
  totalCostUsd: number; // Note: backend sends totalCostUsd (lowercase 'd')
  totalTokens: number;
}

export interface ToolUsageStats {
  toolName: string;
  callCount: number;
  successCount: number;
  failedCount: number;
}

export interface ErrorPattern {
  errorCode: string;
  count: number;
  lastSeen: string;
  sampleRunId: string;
}

export interface TimeSeriesBucket {
  timestamp: string;
  runsStarted: number;
  runsCompleted: number;
  runsFailed: number;
  totalCostUsd: number;
  avgDurationMs: number;
}

// =============================================================================
// API Response Wrappers
// =============================================================================

export interface StatsSummary {
  statusCounts: RunStatusCounts;
  successRate: number;
  duration: DurationStats;
  cost: CostStats;
  runnerBreakdown: RunnerBreakdown[];
}

export interface SummaryResponse {
  summary: StatsSummary;
}

export interface StatusDistributionResponse {
  statusCounts: RunStatusCounts;
}

export interface SuccessRateResponse {
  successRate: number;
}

export interface DurationResponse {
  duration: DurationStats;
}

export interface CostResponse {
  cost: CostStats;
}

export interface RunnerBreakdownResponse {
  runners: RunnerBreakdown[];
}

export interface ProfileBreakdownResponse {
  profiles: ProfileBreakdown[];
}

export interface ModelBreakdownResponse {
  models: ModelBreakdown[];
}

export interface ToolUsageResponse {
  tools: ToolUsageStats[];
}

export interface ErrorPatternsResponse {
  errors: ErrorPattern[];
}

export interface TimeSeriesResponse {
  buckets: TimeSeriesBucket[];
  bucketDuration: string;
}
