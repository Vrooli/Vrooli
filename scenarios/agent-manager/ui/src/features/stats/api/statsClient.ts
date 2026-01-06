// Stats API client - fetch functions for React Query

import { STATS_ENDPOINTS } from "./endpoints";
import type {
  CompareModelsRequest,
  CompareModelsResponse,
  CostResponse,
  DurationResponse,
  ErrorPatternsResponse,
  ModelBreakdownResponse,
  ModelUsageRunsResponse,
  ProfileBreakdownResponse,
  RunnerBreakdownResponse,
  StatsFilter,
  StatusDistributionResponse,
  SuccessRateResponse,
  SummaryResponse,
  TimeSeriesResponse,
  ToolUsageResponse,
  ToolUsageModelsResponse,
  ToolUsageRunsResponse,
} from "./types";

// =============================================================================
// Helpers
// =============================================================================

function buildQueryParams(filter: StatsFilter): URLSearchParams {
  const params = new URLSearchParams();

  if (filter.preset) {
    params.set("preset", filter.preset);
  }
  if (filter.start) {
    params.set("start", filter.start);
  }
  if (filter.end) {
    params.set("end", filter.end);
  }
  if (filter.runnerType) {
    params.set("runner_type", filter.runnerType);
  }
  if (filter.profileId) {
    params.set("profile_id", filter.profileId);
  }
  if (filter.model) {
    params.set("model", filter.model);
  }
  if (filter.tagPrefix) {
    params.set("tag_prefix", filter.tagPrefix);
  }

  return params;
}

async function fetchJson<T>(url: string): Promise<T> {
  const response = await fetch(url);
  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`API error ${response.status}: ${errorText}`);
  }
  return response.json() as Promise<T>;
}

function buildUrl(endpoint: string, filter: StatsFilter, extra?: Record<string, string>): string {
  const params = buildQueryParams(filter);
  if (extra) {
    for (const [key, value] of Object.entries(extra)) {
      params.set(key, value);
    }
  }
  const queryString = params.toString();
  return queryString ? `${endpoint}?${queryString}` : endpoint;
}

// =============================================================================
// API Functions
// =============================================================================

export async function fetchStatsSummary(filter: StatsFilter): Promise<SummaryResponse> {
  const url = buildUrl(STATS_ENDPOINTS.SUMMARY, filter);
  return fetchJson<SummaryResponse>(url);
}

export async function fetchStatusDistribution(filter: StatsFilter): Promise<StatusDistributionResponse> {
  const url = buildUrl(STATS_ENDPOINTS.STATUS_DISTRIBUTION, filter);
  return fetchJson<StatusDistributionResponse>(url);
}

export async function fetchSuccessRate(filter: StatsFilter): Promise<SuccessRateResponse> {
  const url = buildUrl(STATS_ENDPOINTS.SUCCESS_RATE, filter);
  return fetchJson<SuccessRateResponse>(url);
}

export async function fetchDurationStats(filter: StatsFilter): Promise<DurationResponse> {
  const url = buildUrl(STATS_ENDPOINTS.DURATION, filter);
  return fetchJson<DurationResponse>(url);
}

export async function fetchCostStats(filter: StatsFilter): Promise<CostResponse> {
  const url = buildUrl(STATS_ENDPOINTS.COST, filter);
  return fetchJson<CostResponse>(url);
}

export async function fetchRunnerBreakdown(filter: StatsFilter): Promise<RunnerBreakdownResponse> {
  const url = buildUrl(STATS_ENDPOINTS.RUNNERS, filter);
  return fetchJson<RunnerBreakdownResponse>(url);
}

export async function fetchProfileBreakdown(filter: StatsFilter, limit = 10): Promise<ProfileBreakdownResponse> {
  const url = buildUrl(STATS_ENDPOINTS.PROFILES, filter, { limit: String(limit) });
  return fetchJson<ProfileBreakdownResponse>(url);
}

export async function fetchModelBreakdown(filter: StatsFilter, limit = 10): Promise<ModelBreakdownResponse> {
  const url = buildUrl(STATS_ENDPOINTS.MODELS, filter, { limit: String(limit) });
  return fetchJson<ModelBreakdownResponse>(url);
}

export async function fetchModelUsageRuns(filter: StatsFilter, limit = 25): Promise<ModelUsageRunsResponse> {
  const url = buildUrl(STATS_ENDPOINTS.MODEL_RUNS, filter, { limit: String(limit) });
  return fetchJson<ModelUsageRunsResponse>(url);
}

export async function fetchToolUsage(filter: StatsFilter, limit = 20): Promise<ToolUsageResponse> {
  const url = buildUrl(STATS_ENDPOINTS.TOOLS, filter, { limit: String(limit) });
  return fetchJson<ToolUsageResponse>(url);
}

export async function fetchToolUsageModels(filter: StatsFilter, toolName: string, limit = 25): Promise<ToolUsageModelsResponse> {
  const url = buildUrl(STATS_ENDPOINTS.TOOL_MODELS, filter, { limit: String(limit), tool_name: toolName });
  return fetchJson<ToolUsageModelsResponse>(url);
}

export async function fetchToolUsageRuns(filter: StatsFilter, toolName: string, limit = 25): Promise<ToolUsageRunsResponse> {
  const url = buildUrl(STATS_ENDPOINTS.TOOL_RUNS, filter, { limit: String(limit), tool_name: toolName });
  return fetchJson<ToolUsageRunsResponse>(url);
}

export async function fetchErrorPatterns(filter: StatsFilter, limit = 10): Promise<ErrorPatternsResponse> {
  const url = buildUrl(STATS_ENDPOINTS.ERRORS, filter, { limit: String(limit) });
  return fetchJson<ErrorPatternsResponse>(url);
}

export async function fetchTimeSeries(
  filter: StatsFilter,
  bucket?: string
): Promise<TimeSeriesResponse> {
  const extra: Record<string, string> = {};
  if (bucket) {
    extra.bucket = bucket;
  }
  const url = buildUrl(STATS_ENDPOINTS.TIME_SERIES, filter, extra);
  return fetchJson<TimeSeriesResponse>(url);
}

export async function fetchModelCostComparison(
  request: CompareModelsRequest
): Promise<CompareModelsResponse> {
  const response = await fetch("/api/v1/pricing/compare", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`API error ${response.status}: ${errorText}`);
  }
  return response.json() as Promise<CompareModelsResponse>;
}

// =============================================================================
// Query Keys (for React Query)
// =============================================================================

export const statsQueryKeys = {
  all: ["stats"] as const,
  summary: (filter: StatsFilter) => [...statsQueryKeys.all, "summary", filter] as const,
  statusDistribution: (filter: StatsFilter) => [...statsQueryKeys.all, "statusDistribution", filter] as const,
  successRate: (filter: StatsFilter) => [...statsQueryKeys.all, "successRate", filter] as const,
  duration: (filter: StatsFilter) => [...statsQueryKeys.all, "duration", filter] as const,
  cost: (filter: StatsFilter) => [...statsQueryKeys.all, "cost", filter] as const,
  runners: (filter: StatsFilter) => [...statsQueryKeys.all, "runners", filter] as const,
  profiles: (filter: StatsFilter, limit: number) => [...statsQueryKeys.all, "profiles", filter, limit] as const,
  models: (filter: StatsFilter, limit: number) => [...statsQueryKeys.all, "models", filter, limit] as const,
  modelRuns: (filter: StatsFilter, limit: number) => [...statsQueryKeys.all, "modelRuns", filter, limit] as const,
  tools: (filter: StatsFilter, limit: number) => [...statsQueryKeys.all, "tools", filter, limit] as const,
  toolModels: (filter: StatsFilter, toolName: string, limit: number) =>
    [...statsQueryKeys.all, "toolModels", filter, toolName, limit] as const,
  toolRuns: (filter: StatsFilter, toolName: string, limit: number) =>
    [...statsQueryKeys.all, "toolRuns", filter, toolName, limit] as const,
  errors: (filter: StatsFilter, limit: number) => [...statsQueryKeys.all, "errors", filter, limit] as const,
  timeSeries: (filter: StatsFilter, bucket?: string) => [...statsQueryKeys.all, "timeSeries", filter, bucket] as const,
  modelCostComparison: (request: CompareModelsRequest) =>
    [
      "pricing",
      "compare",
      request.modelList,
      request.actualModel,
      request.inputTokens,
      request.outputTokens,
      request.cacheReadTokens,
      request.cacheCreationTokens,
    ] as const,
} as const;
