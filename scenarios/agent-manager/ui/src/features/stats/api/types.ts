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
  workloadKey?: string;
  errorCode?: string;
}

// =============================================================================
// Response Types
// =============================================================================

export interface MeasureMetadata {
  executedQuery: string;
  validity: {
    state: "available" | "unreliable" | "unavailable";
    reason: string;
    sampleSize: number;
    largestFingerprintShare: number;
  };
  definitionId: string;
}

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
  totalTokens?: number;
  totalChargeMicroUsd?: number;
}

export interface ProfileBreakdown {
  profileId: string;
  profileName: string;
  runCount: number;
  successCount: number;
  failedCount: number;
  totalCostUsd: number; // Note: backend sends totalCostUsd (lowercase 'd')
  totalTokens?: number;
  totalChargeMicroUsd?: number;
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

export interface ToolUsageModelBreakdown {
  model: string;
  runCount: number;
  callCount?: number;
  successCount: number;
  failedCount: number;
}

export interface ModelUsageRun {
  runId: string;
  taskTitle?: string;
  profileName?: string;
  status?: string;
  createdAt?: string;
  model?: string;
  totalTokens: number;
  totalChargeMicroUsd?: number;
  chargeBasis?: string;
}

export interface ToolUsageRun {
  runId: string;
  taskTitle?: string;
  profileName?: string;
  status?: string;
  createdAt?: string;
  model?: string;
  callCount?: number;
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
  measure?: MeasureMetadata;
}

export interface SuccessRateResponse {
  successRate: number;
  measure?: MeasureMetadata;
}

export interface DurationResponse {
  duration: DurationStats;
  measure?: MeasureMetadata;
}

export interface CostResponse {
  cost: CostStats;
  measure?: MeasureMetadata;
}

export interface RunnerBreakdownResponse {
  runners: RunnerBreakdown[];
  measure?: MeasureMetadata;
}

export interface ProfileBreakdownResponse {
  profiles: ProfileBreakdown[];
  measure?: MeasureMetadata;
}

export interface ModelBreakdownResponse {
  models: ModelBreakdown[];
  measure?: MeasureMetadata;
}

export interface ToolUsageResponse {
  tools: ToolUsageStats[];
  measure?: MeasureMetadata;
}
export interface ToolCommandBreakdownResponse {
  rows: Array<{ executable: string; commandPath: string; callCount: number; successCount: number; failedCount: number; runCount: number; truncated: boolean }>;
  measure?: MeasureMetadata;
}

export interface ModelUsageRunsResponse {
  runs: ModelUsageRun[];
  measure?: MeasureMetadata;
}

export interface ToolUsageRunsResponse {
  runs: ToolUsageRun[];
  measure?: MeasureMetadata;
}

export interface ToolUsageModelsResponse {
  models: ToolUsageModelBreakdown[];
  measure?: MeasureMetadata;
}

export interface ErrorPatternsResponse {
  errors: ErrorPattern[];
  measure?: MeasureMetadata;
}

export interface TimeSeriesResponse {
  buckets: TimeSeriesBucket[];
  bucketDuration: string;
  measure?: MeasureMetadata;
}

// =============================================================================
// Model Cost Comparison Types
// =============================================================================

export interface ModelCostComparison {
  model: string;
  canonicalModel?: string;
  estimatedCostUsd: number;
  differenceUsd: number;
  differencePercent: number;
  isActualModel: boolean;
  priceSource?: string;
}

export interface CompareModelsRequest {
  inputTokens: number;
  outputTokens: number;
  cacheReadTokens: number;
  cacheCreationTokens: number;
  webSearchRequests: number;
  serverToolUseCount: number;
  actualCostUsd: number;
  actualModel: string;
  modelList: "popular" | "recent";
}

export interface CompareModelsResponse {
  comparisons: ModelCostComparison[];
}
