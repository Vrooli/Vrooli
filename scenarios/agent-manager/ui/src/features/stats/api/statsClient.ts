// Stats API client - fetch functions for React Query

import { timestampFromDate, type Timestamp } from "@bufbuild/protobuf/wkt";
import type {
  CohortRun,
  InvocationFilter,
  ChargeByBasis,
  MeasureProvenance,
  MeasureValidity,
  RunBreakdownRow,
  TerminalRunTrendRow,
  ToolUsageRow,
  ToolCommandBreakdownRow,
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
  workload: (filter: StatsFilter, limit: number) => [...statsQueryKeys.all, "workload", filter, limit] as const,
  toolCommands: (filter: StatsFilter, toolName: string, limit: number) => [...statsQueryKeys.all, "toolCommands", filter, toolName, limit] as const,
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

export interface MeasureValidityView {
  state: "available" | "unreliable" | "unavailable";
  reason: string;
  sampleSize: number;
  largestFingerprintShare: number;
}

export interface MeasureProvenanceView {
  sourceTable: string;
  windowStart: string;
  windowEnd: string;
  rowCount: number;
  appliedFilters: Array<{ field: string; value: string }>;
}

export interface MeasureDefinitionView {
  id: string;
  counts: string;
  numerator: string;
  denominator: string;
  sourceTable: string;
  limitation: string;
}

export interface MeasureResponse {
  executedQuery: string;
  validity: MeasureValidityView;
  provenance?: MeasureProvenanceView;
  definitionId: string;
}

export interface ChargeByBasisView {
  basis: string;
  runCount: number;
  chargeMicroUsd: number;
  tokenCount: number;
  chargeReason: string;
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

function validity(value?: MeasureValidity): MeasureValidityView {
  if (!value) {
    return { state: "unavailable", reason: "the measure did not provide a validity assessment", sampleSize: 0, largestFingerprintShare: 0 };
  }
  const state = value.state === "available" || value.state === "unreliable" || value.state === "unavailable" ? value.state : "unavailable";
  return { state, reason: value.reason, sampleSize: numberValue(value.sampleSize), largestFingerprintShare: value.largestFingerprintShare };
}

function provenance(value?: MeasureProvenance): MeasureProvenanceView | undefined {
  if (!value) return undefined;
  return { sourceTable: value.sourceTable, windowStart: value.windowStart, windowEnd: value.windowEnd, rowCount: numberValue(value.rowCount), appliedFilters: value.appliedFilters.map((filter) => ({ field: filter.field, value: filter.value })) };
}

function metadata(response: { executedQuery: string; validity?: MeasureValidity; provenance?: MeasureProvenance; definitionId: string }): Pick<MeasureResponse, "executedQuery" | "validity" | "provenance" | "definitionId"> {
  return { executedQuery: response.executedQuery, validity: validity(response.validity), provenance: provenance(response.provenance), definitionId: response.definitionId };
}

function cohortRow(row: CohortRun): DurableCohortRun {
  return {
    runId: row.runId,
    taskTitle: row.taskTitle || undefined,
    profileId: row.profileId || undefined,
    profileName: row.profileName || undefined,
    status: row.status || undefined,
    createdAt: row.createdAt || undefined,
    model: row.model || undefined,
    runnerType: row.runnerType || undefined,
    workloadKey: row.workloadKey || undefined,
    totalTokens: numberValue(row.totalTokens),
    totalChargeMicroUsd: row.totalChargeMicroUsd === undefined ? undefined : numberValue(row.totalChargeMicroUsd),
    chargeBasis: row.chargeBasis,
    toolCallCount: row.toolCallCount === undefined ? undefined : numberValue(row.toolCallCount),
  };
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
      tagPrefix: filter.tagPrefix ?? "", runStatus: "", workloadKey: filter.workloadKey ?? "", errorCode: filter.errorCode ?? "",
    } as InvocationFilter,
  };
}

function measureRequestForTool(filter: StatsFilter, toolName: string): { window: TimeWindow; filter: InvocationFilter } {
  const request = measureRequest(filter);
  return { ...request, filter: { ...request.filter, toolName } };
}

export async function fetchExternalToolShare(window: MeasureWindow): Promise<ExternalToolShareMeasure> {
  const response = await measuresClient.externalToolShare({ window: windowFromMeasureWindow(window) });
  return { share: response.share, externalCalls: numberValue(response.externalCalls), resolvedCalls: numberValue(response.resolvedCalls), unknownCalls: numberValue(response.unknownCalls), ...metadata(response) };
}
export async function fetchRetryRate(window: MeasureWindow): Promise<RateMeasure> {
  const response = await measuresClient.retryRate({ window: windowFromMeasureWindow(window) });
  return { rate: response.rate, ...metadata(response) };
}
export async function fetchHelpRecoveryRate(window: MeasureWindow): Promise<RateMeasure> {
  const response = await measuresClient.helpRecoveryRate({ window: windowFromMeasureWindow(window) });
  return { rate: response.rate, ...metadata(response) };
}
export async function fetchRepeatedWorkRate(window: MeasureWindow): Promise<RateMeasure> {
  const response = await measuresClient.repeatedWorkRate({ window: windowFromMeasureWindow(window) });
  return { rate: response.rate, ...metadata(response) };
}
export async function fetchFileRereadRate(window: MeasureWindow): Promise<FileRereadRateMeasure> {
  const response = await measuresClient.fileRereadRate({ window: windowFromMeasureWindow(window) });
  return { rate: response.rate, filesReadMoreThanOnce: numberValue(response.filesReadMoreThanOnce), readCalls: numberValue(response.readCalls), ...metadata(response) };
}
export async function fetchFindingRecurrenceRate(window: MeasureWindow): Promise<FindingRecurrenceRateMeasure> {
  const response = await measuresClient.findingRecurrenceRate({ window: windowFromMeasureWindow(window) });
  return { rate: response.rate, recurringFindings: numberValue(response.recurringFindings), totalFindings: numberValue(response.totalFindings), recurringFingerprints: numberValue(response.recurringFingerprints), ...metadata(response) };
}
export async function fetchDurableErrorPatterns(window: MeasureWindow): Promise<ErrorPatternsMeasure> {
  const response = await measuresClient.errorPatterns({ window: windowFromMeasureWindow(window) });
  return { rows: response.rows.map((row) => ({ errorCode: row.errorCode, count: numberValue(row.count), lastSeen: row.lastSeen, sampleRunId: row.sampleRunId })), ...metadata(response) };
}

export interface DurableRunBreakdown extends MeasureResponse {
  rows: Array<{ key: string; value: string; runCount: number; successCount: number; failedCount: number; totalCostUsd: number; totalTokens: number; totalChargeMicroUsd: number; averageDurationMs: number; consumptionPerSuccessfulCompletion: number; completionRate: number }>;
}
export interface DurableTerminalTrend extends MeasureResponse {
  rows: Array<{ bucket: string; terminalRuns: number; completedRuns: number; failedRuns: number; cancelledRuns: number; totalCostUsd: number; averageDurationMs: number }>;
}
export interface DurableToolUsage extends MeasureResponse {
  rows: Array<{ toolName: string; callCount: number; successCount: number; failedCount: number }>;
}
export interface DurableToolCommandBreakdown extends MeasureResponse {
  rows: Array<{ executable: string; commandPath: string; callCount: number; successCount: number; failedCount: number; runCount: number; truncated: boolean }>;
}
export interface DurableCohort extends MeasureResponse {
  runIds: string[];
  truncated: boolean;
  rows: DurableCohortRun[];
}
export interface DurableCohortRun {
  runId: string;
  taskTitle?: string;
  profileId?: string;
  profileName?: string;
  status?: string;
  createdAt?: string;
  model?: string;
  runnerType?: string;
  workloadKey?: string;
  totalTokens: number;
  totalChargeMicroUsd?: number;
  chargeBasis?: string;
  toolCallCount?: number;
}
export interface DurableRunCost extends MeasureResponse {
  totalCostUsd: number; averageCostUsd: number; totalRuns: number; totalTokens: number;
  inputTokens: number; outputTokens: number; cacheReadTokens: number; cacheCreationTokens: number;
  inputCostUsd: number; outputCostUsd: number; cacheReadCostUsd: number; cacheCreationCostUsd: number;
  totalChargeMicroUsd: number; unpricedTokenCount: number; chargeByBasis: ChargeByBasisView[];
}
export interface DurableRunDurationStatistics extends MeasureResponse {
  averageDurationMs: number; p50DurationMs: number; p95DurationMs: number; p99DurationMs: number;
  minDurationMs: number; maxDurationMs: number; count: number;
}
export interface DurableRunStatusDistribution extends MeasureResponse {
  rows: Array<{ status: string; count: number }>;
}
function breakdownRows(rows: readonly RunBreakdownRow[]): DurableRunBreakdown["rows"] {
  return rows.map((row) => ({ key: row.key, value: row.value, runCount: numberValue(row.runCount), successCount: numberValue(row.successCount), failedCount: numberValue(row.failedCount), totalCostUsd: row.totalCostUsd, totalTokens: numberValue(row.totalTokens), totalChargeMicroUsd: numberValue(row.totalChargeMicroUsd), averageDurationMs: row.averageDurationMs, consumptionPerSuccessfulCompletion: row.consumptionPerSuccessfulCompletion, completionRate: row.completionRate }));
}
function trendRows(rows: readonly TerminalRunTrendRow[]): DurableTerminalTrend["rows"] {
  return rows.map((row) => ({ bucket: row.bucket, terminalRuns: numberValue(row.terminalRuns), completedRuns: numberValue(row.completedRuns), failedRuns: numberValue(row.failedRuns), cancelledRuns: numberValue(row.cancelledRuns), totalCostUsd: row.totalCostUsd, averageDurationMs: row.averageDurationMs }));
}
function toolRows(rows: readonly ToolUsageRow[]): DurableToolUsage["rows"] {
  return rows.map((row) => ({ toolName: row.toolName, callCount: numberValue(row.callCount), successCount: numberValue(row.successCount), failedCount: numberValue(row.failedCount) }));
}
function chargeRows(rows: readonly ChargeByBasis[] | undefined): ChargeByBasisView[] {
  if (!rows) return [];
  return rows.map((row) => ({ basis: row.basis, runCount: numberValue(row.runCount), chargeMicroUsd: numberValue(row.chargeMicroUsd), tokenCount: numberValue(row.tokenCount), chargeReason: row.chargeReason }));
}
export async function fetchDurableRunnerBreakdown(filter: StatsFilter): Promise<DurableRunBreakdown> {
  const response = await measuresClient.runnerBreakdown(measureRequest(filter));
  return { rows: breakdownRows(response.rows), ...metadata(response) };
}
export async function fetchDurableProfileBreakdown(filter: StatsFilter): Promise<DurableRunBreakdown> {
  const response = await measuresClient.profileBreakdown(measureRequest(filter));
  return { rows: breakdownRows(response.rows), ...metadata(response) };
}
export async function fetchDurableModelBreakdown(filter: StatsFilter): Promise<DurableRunBreakdown> {
  const response = await measuresClient.modelBreakdown(measureRequest(filter));
  return { rows: breakdownRows(response.rows), ...metadata(response) };
}
export async function fetchDurableTerminalTrend(filter: StatsFilter): Promise<DurableTerminalTrend> {
  const response = await measuresClient.terminalRunTrend(measureRequest(filter));
  return { rows: trendRows(response.rows), ...metadata(response) };
}
export async function fetchDurableToolUsage(filter: StatsFilter): Promise<DurableToolUsage> {
  const response = await measuresClient.toolUsage(measureRequest(filter));
  return { rows: toolRows(response.rows), ...metadata(response) };
}
export async function fetchDurableRunCost(filter: StatsFilter): Promise<DurableRunCost> {
  const r = await measuresClient.runCost(measureRequest(filter));
  return { totalCostUsd: r.totalCostUsd, averageCostUsd: r.averageCostUsd, totalRuns: numberValue(r.totalRuns), totalTokens: numberValue(r.totalTokens), inputTokens: numberValue(r.inputTokens), outputTokens: numberValue(r.outputTokens), cacheReadTokens: numberValue(r.cacheReadTokens), cacheCreationTokens: numberValue(r.cacheCreationTokens), inputCostUsd: r.inputCostUsd, outputCostUsd: r.outputCostUsd, cacheReadCostUsd: r.cacheReadCostUsd, cacheCreationCostUsd: r.cacheCreationCostUsd, totalChargeMicroUsd: numberValue(r.totalChargeMicroUsd), unpricedTokenCount: numberValue(r.unpricedTokenCount), chargeByBasis: chargeRows(r.chargeByBasis), ...metadata(r) };
}
export async function fetchDurableRunSuccess(filter: StatsFilter): Promise<RateMeasure> {
  const r = await measuresClient.runSuccessRate(measureRequest(filter)); return { rate: r.rate, ...metadata(r) };
}
export async function fetchDurableRunCycleTime(filter: StatsFilter): Promise<RateMeasure> {
  const r = await measuresClient.runCycleTime(measureRequest(filter)); return { rate: r.averageDurationMs, ...metadata(r) };
}
export async function fetchDurableRunDurationStatistics(filter: StatsFilter): Promise<DurableRunDurationStatistics> {
  const r = await measuresClient.runDurationStatistics(measureRequest(filter));
  return { averageDurationMs: r.averageDurationMs, p50DurationMs: r.p50DurationMs, p95DurationMs: r.p95DurationMs, p99DurationMs: r.p99DurationMs, minDurationMs: numberValue(r.minDurationMs), maxDurationMs: numberValue(r.maxDurationMs), count: numberValue(r.count), ...metadata(r) };
}
export async function fetchDurableRunVolume(filter: StatsFilter): Promise<{ totalRuns: number; historyFloor: string; outsideHistoryRunCount: number } & MeasureResponse> {
  const r = await measuresClient.runVolume(measureRequest(filter)); return { totalRuns: numberValue(r.totalRuns), historyFloor: r.historyFloor, outsideHistoryRunCount: numberValue(r.outsideHistoryRunCount), ...metadata(r) };
}
export async function fetchDurableRunStatusDistribution(filter: StatsFilter): Promise<DurableRunStatusDistribution> {
  const r = await measuresClient.runStatusDistribution(measureRequest(filter));
  return { rows: r.rows.map((row) => ({ status: row.status, count: numberValue(row.count) })), ...metadata(r) };
}
export async function fetchDurableToolCohort(filter: StatsFilter, toolName: string, limit = 25): Promise<DurableCohort> {
  const response = await measuresClient.selectCohort({ ...measureRequestForTool(filter, toolName), limit });
  return { runIds: response.runIds, rows: (response.rows ?? []).map(cohortRow), truncated: response.truncated, ...metadata(response) };
}
export async function fetchDurableModelCohort(filter: StatsFilter, model: string, limit = 25): Promise<DurableCohort> {
  const response = await measuresClient.selectCohort({ ...measureRequest({ ...filter, model }), limit });
  return { runIds: response.runIds, rows: (response.rows ?? []).map(cohortRow), truncated: response.truncated, ...metadata(response) };
}
export async function fetchDurableToolModels(filter: StatsFilter, toolName: string): Promise<DurableRunBreakdown> {
  const response = await measuresClient.modelBreakdown(measureRequestForTool(filter, toolName));
  return { rows: breakdownRows(response.rows), ...metadata(response) };
}

export async function fetchDurableWorkloadBreakdown(filter: StatsFilter): Promise<DurableRunBreakdown> {
  const response = await measuresClient.workloadBreakdown(measureRequest(filter));
  return { rows: breakdownRows(response.rows), ...metadata(response) };
}

export async function fetchDurableToolCommands(filter: StatsFilter, toolName: string, limit = 20): Promise<DurableToolCommandBreakdown> {
  const response = await measuresClient.toolCommandBreakdown({ ...measureRequestForTool(filter, toolName), limit });
  return { rows: response.rows.map((row: ToolCommandBreakdownRow) => ({ executable: row.executable, commandPath: row.commandPath, callCount: numberValue(row.callCount), successCount: numberValue(row.successCount), failedCount: numberValue(row.failedCount), runCount: numberValue(row.runCount), truncated: row.truncated })), ...metadata(response) };
}

export async function fetchMeasureDefinitions(): Promise<MeasureDefinitionView[]> {
  const response = await measuresClient.allMeasureDefinitions({});
  return response.definitions.map((definition) => ({ id: definition.id, counts: definition.counts, numerator: definition.numerator, denominator: definition.denominator, sourceTable: definition.sourceTable, limitation: definition.limitation }));
}
