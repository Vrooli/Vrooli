// Stats API client - fetch functions for React Query

import { timestampFromDate, type Timestamp } from "@bufbuild/protobuf/wkt";
import type {
  InvocationFilter,
  RunBreakdownRow,
  TerminalRunTrendRow,
  ToolUsageRow,
} from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import type { TimeWindow } from "@vrooli/proto-types/measures/v1/measures_pb";
import { measuresClient } from "./measuresClient";
import type {
  CompareModelsRequest,
  CompareModelsResponse,
  StatsFilter,
} from "./types";

// =============================================================================
// Helpers
// =============================================================================

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

// =============================================================================
// Typed durable measures (Connect JSON)
// =============================================================================

export interface MeasureWindow {
  window: {
    custom: {
      from: string;
      to: string;
    };
  };
}

interface MeasureResponse {
  executedQuery: string;
}

export interface ExternalToolShareMeasure extends MeasureResponse {
  share: number;
  externalCalls: number;
  resolvedCalls: number;
  unknownCalls: number;
}
export interface RateMeasure extends MeasureResponse {
  rate: number;
}
export interface FileRereadRateMeasure extends RateMeasure {
  filesReadMoreThanOnce: number;
  readCalls: number;
}
export interface FindingRecurrenceRateMeasure extends RateMeasure {
  recurringFindings: number;
  totalFindings: number;
  recurringFingerprints: number;
}
export interface ErrorPatternMeasureRow {
  errorCode: string;
  count: number;
  lastSeen: string;
  sampleRunId: string;
}
export interface ErrorPatternsMeasure extends MeasureResponse {
  rows: ErrorPatternMeasureRow[];
}

function numberValue(value: bigint): number {
  return Number(value);
}

function timestamp(value: string): Timestamp {
  const date = new Date(value);
  if (Number.isNaN(date.valueOf())) {
    throw new Error(`Invalid measure timestamp: ${value}`);
  }
  return timestampFromDate(date);
}

function customWindow(from: string, to: string): TimeWindow {
  return { window: { case: "custom", value: { from: timestamp(from), to: timestamp(to) } } } as TimeWindow;
}

function windowFromMeasureWindow(input: MeasureWindow): TimeWindow {
  return customWindow(input.window.custom.from, input.window.custom.to);
}

function presetHours(preset: StatsFilter["preset"]): number {
  switch (preset) {
    case "6h": return 6;
    case "12h": return 12;
    case "7d": return 24 * 7;
    case "30d": return 24 * 30;
    default: return 24;
  }
}

function measureRequest(filter: StatsFilter): { window: TimeWindow; filter: InvocationFilter } {
  const now = new Date();
  const from = filter.start ?? new Date(now.getTime() - presetHours(filter.preset) * 60 * 60 * 1000).toISOString();
  const to = filter.end ?? now.toISOString();
  return {
    window: customWindow(from, to),
    filter: {
      ownership: "", outcome: "", executable: "", fingerprint: "",
      profileId: filter.profileId ?? "", runnerType: filter.runnerType ?? "", model: filter.model ?? "",
      tagPrefix: filter.tagPrefix ?? "", runStatus: "",
    } as InvocationFilter,
  };
}

function measureRequestForTool(filter: StatsFilter, toolName: string): { window: TimeWindow; filter: InvocationFilter } {
  const request = measureRequest(filter);
  return { ...request, filter: { ...request.filter, toolName } };
}

export async function fetchExternalToolShare(window: MeasureWindow): Promise<ExternalToolShareMeasure> {
  const response = await measuresClient.externalToolShare({ window: windowFromMeasureWindow(window) });
  return { share: response.share, externalCalls: numberValue(response.externalCalls), resolvedCalls: numberValue(response.resolvedCalls), unknownCalls: numberValue(response.unknownCalls), executedQuery: response.executedQuery };
}
export async function fetchRetryRate(window: MeasureWindow): Promise<RateMeasure> {
  const response = await measuresClient.retryRate({ window: windowFromMeasureWindow(window) });
  return { rate: response.rate, executedQuery: response.executedQuery };
}
export async function fetchHelpRecoveryRate(window: MeasureWindow): Promise<RateMeasure> {
  const response = await measuresClient.helpRecoveryRate({ window: windowFromMeasureWindow(window) });
  return { rate: response.rate, executedQuery: response.executedQuery };
}
export async function fetchRepeatedWorkRate(window: MeasureWindow): Promise<RateMeasure> {
  const response = await measuresClient.repeatedWorkRate({ window: windowFromMeasureWindow(window) });
  return { rate: response.rate, executedQuery: response.executedQuery };
}
export async function fetchFileRereadRate(window: MeasureWindow): Promise<FileRereadRateMeasure> {
  const response = await measuresClient.fileRereadRate({ window: windowFromMeasureWindow(window) });
  return { rate: response.rate, filesReadMoreThanOnce: numberValue(response.filesReadMoreThanOnce), readCalls: numberValue(response.readCalls), executedQuery: response.executedQuery };
}
export async function fetchFindingRecurrenceRate(window: MeasureWindow): Promise<FindingRecurrenceRateMeasure> {
  const response = await measuresClient.findingRecurrenceRate({ window: windowFromMeasureWindow(window) });
  return { rate: response.rate, recurringFindings: numberValue(response.recurringFindings), totalFindings: numberValue(response.totalFindings), recurringFingerprints: numberValue(response.recurringFingerprints), executedQuery: response.executedQuery };
}
export async function fetchDurableErrorPatterns(window: MeasureWindow): Promise<ErrorPatternsMeasure> {
  const response = await measuresClient.errorPatterns({ window: windowFromMeasureWindow(window) });
  return { rows: response.rows.map((row) => ({ errorCode: row.errorCode, count: numberValue(row.count), lastSeen: row.lastSeen, sampleRunId: row.sampleRunId })), executedQuery: response.executedQuery };
}

export interface DurableRunBreakdown extends MeasureResponse {
  rows: Array<{ key: string; value: string; runCount: number; successCount: number; failedCount: number; totalCostUsd: number; totalTokens: number; averageDurationMs: number }>;
}
export interface DurableTerminalTrend extends MeasureResponse {
  rows: Array<{ bucket: string; terminalRuns: number; completedRuns: number; failedRuns: number; cancelledRuns: number; totalCostUsd: number; averageDurationMs: number }>;
}
export interface DurableToolUsage extends MeasureResponse {
  rows: Array<{ toolName: string; callCount: number; successCount: number; failedCount: number }>;
}
export interface DurableCohort extends MeasureResponse {
  runIds: string[];
  truncated: boolean;
}
export interface DurableRunCost extends MeasureResponse {
  totalCostUsd: number; averageCostUsd: number; totalRuns: number; totalTokens: number;
  inputTokens: number; outputTokens: number; cacheReadTokens: number; cacheCreationTokens: number;
  inputCostUsd: number; outputCostUsd: number; cacheReadCostUsd: number; cacheCreationCostUsd: number;
  authoritativeCostUsd: number; estimatedCostUsd: number; unknownCostUsd: number;
}
export interface DurableRunDurationStatistics extends MeasureResponse {
  averageDurationMs: number; p50DurationMs: number; p95DurationMs: number; p99DurationMs: number;
  minDurationMs: number; maxDurationMs: number; count: number;
}
export interface DurableRunStatusDistribution extends MeasureResponse {
  rows: Array<{ status: string; count: number }>;
}
function breakdownRows(rows: readonly RunBreakdownRow[]): DurableRunBreakdown["rows"] {
  return rows.map((row) => ({ key: row.key, value: row.value, runCount: numberValue(row.runCount), successCount: numberValue(row.successCount), failedCount: numberValue(row.failedCount), totalCostUsd: row.totalCostUsd, totalTokens: numberValue(row.totalTokens), averageDurationMs: row.averageDurationMs }));
}
function trendRows(rows: readonly TerminalRunTrendRow[]): DurableTerminalTrend["rows"] {
  return rows.map((row) => ({ bucket: row.bucket, terminalRuns: numberValue(row.terminalRuns), completedRuns: numberValue(row.completedRuns), failedRuns: numberValue(row.failedRuns), cancelledRuns: numberValue(row.cancelledRuns), totalCostUsd: row.totalCostUsd, averageDurationMs: row.averageDurationMs }));
}
function toolRows(rows: readonly ToolUsageRow[]): DurableToolUsage["rows"] {
  return rows.map((row) => ({ toolName: row.toolName, callCount: numberValue(row.callCount), successCount: numberValue(row.successCount), failedCount: numberValue(row.failedCount) }));
}
export async function fetchDurableRunnerBreakdown(filter: StatsFilter): Promise<DurableRunBreakdown> {
  const response = await measuresClient.runnerBreakdown(measureRequest(filter));
  return { rows: breakdownRows(response.rows), executedQuery: response.executedQuery };
}
export async function fetchDurableProfileBreakdown(filter: StatsFilter): Promise<DurableRunBreakdown> {
  const response = await measuresClient.profileBreakdown(measureRequest(filter));
  return { rows: breakdownRows(response.rows), executedQuery: response.executedQuery };
}
export async function fetchDurableModelBreakdown(filter: StatsFilter): Promise<DurableRunBreakdown> {
  const response = await measuresClient.modelBreakdown(measureRequest(filter));
  return { rows: breakdownRows(response.rows), executedQuery: response.executedQuery };
}
export async function fetchDurableTerminalTrend(filter: StatsFilter): Promise<DurableTerminalTrend> {
  const response = await measuresClient.terminalRunTrend(measureRequest(filter));
  return { rows: trendRows(response.rows), executedQuery: response.executedQuery };
}
export async function fetchDurableToolUsage(filter: StatsFilter): Promise<DurableToolUsage> {
  const response = await measuresClient.toolUsage(measureRequest(filter));
  return { rows: toolRows(response.rows), executedQuery: response.executedQuery };
}
export async function fetchDurableRunCost(filter: StatsFilter): Promise<DurableRunCost> {
  const r = await measuresClient.runCost(measureRequest(filter));
  return { totalCostUsd: r.totalCostUsd, averageCostUsd: r.averageCostUsd, totalRuns: numberValue(r.totalRuns), totalTokens: numberValue(r.totalTokens), inputTokens: numberValue(r.inputTokens), outputTokens: numberValue(r.outputTokens), cacheReadTokens: numberValue(r.cacheReadTokens), cacheCreationTokens: numberValue(r.cacheCreationTokens), inputCostUsd: r.inputCostUsd, outputCostUsd: r.outputCostUsd, cacheReadCostUsd: r.cacheReadCostUsd, cacheCreationCostUsd: r.cacheCreationCostUsd, authoritativeCostUsd: r.authoritativeCostUsd, estimatedCostUsd: r.estimatedCostUsd, unknownCostUsd: r.unknownCostUsd, executedQuery: r.executedQuery };
}
export async function fetchDurableRunSuccess(filter: StatsFilter): Promise<RateMeasure> {
  const r = await measuresClient.runSuccessRate(measureRequest(filter)); return { rate: r.rate, executedQuery: r.executedQuery };
}
export async function fetchDurableRunCycleTime(filter: StatsFilter): Promise<RateMeasure> {
  const r = await measuresClient.runCycleTime(measureRequest(filter)); return { rate: r.averageDurationMs, executedQuery: r.executedQuery };
}
export async function fetchDurableRunDurationStatistics(filter: StatsFilter): Promise<DurableRunDurationStatistics> {
  const r = await measuresClient.runDurationStatistics(measureRequest(filter));
  return { averageDurationMs: r.averageDurationMs, p50DurationMs: r.p50DurationMs, p95DurationMs: r.p95DurationMs, p99DurationMs: r.p99DurationMs, minDurationMs: numberValue(r.minDurationMs), maxDurationMs: numberValue(r.maxDurationMs), count: numberValue(r.count), executedQuery: r.executedQuery };
}
export async function fetchDurableRunVolume(filter: StatsFilter): Promise<{ totalRuns: number; executedQuery: string }> {
  const r = await measuresClient.runVolume(measureRequest(filter)); return { totalRuns: numberValue(r.totalRuns), executedQuery: r.executedQuery };
}
export async function fetchDurableRunStatusDistribution(filter: StatsFilter): Promise<DurableRunStatusDistribution> {
  const r = await measuresClient.runStatusDistribution(measureRequest(filter));
  return { rows: r.rows.map((row) => ({ status: row.status, count: numberValue(row.count) })), executedQuery: r.executedQuery };
}
export async function fetchDurableToolCohort(filter: StatsFilter, toolName: string, limit = 25): Promise<DurableCohort> {
  const response = await measuresClient.selectCohort({ ...measureRequestForTool(filter, toolName), limit });
  return { runIds: response.runIds, truncated: response.truncated, executedQuery: response.executedQuery };
}
export async function fetchDurableModelCohort(filter: StatsFilter, model: string, limit = 25): Promise<DurableCohort> {
  const response = await measuresClient.selectCohort({ ...measureRequest({ ...filter, model }), limit });
  return { runIds: response.runIds, truncated: response.truncated, executedQuery: response.executedQuery };
}
export async function fetchDurableToolModels(filter: StatsFilter, toolName: string): Promise<DurableRunBreakdown> {
  const response = await measuresClient.modelBreakdown(measureRequestForTool(filter, toolName));
  return { rows: breakdownRows(response.rows), executedQuery: response.executedQuery };
}
