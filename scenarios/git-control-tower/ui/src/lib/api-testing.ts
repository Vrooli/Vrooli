// ============================================================================
// Test Execution & Tidiness Manager — Types + API Functions
// ============================================================================

import { create } from "@bufbuild/protobuf";
import { GetScenarioDetailRequestSchema, GetScoreRequestSchema, GetStalenessRequestSchema, ListIssuesRequestSchema, TriggerLightScanRequestSchema } from "@vrooli/proto-types/git-control-tower/v1/tidiness/tidiness_pb";
import { tidinessClient } from "./connect";

// ── Test Execution Types ───────────────────────────────────────────────

export interface TestExecutionRequest {
  scenarioName: string;
  preset?: string;
  phases?: string[];
  skip?: string[];
  failFast?: boolean;
}

export interface TestExecutionResult {
  executionId: string;
  scenarioName: string;
  success: boolean;
  startedAt: string;
  completedAt?: string;
  preset?: string;
  phases: TestPhaseResult[];
  phaseSummary: TestPhaseSummary;
  warnings?: string[];
}

export interface TestPhaseResult {
  name: string;
  status: "passed" | "failed";
  durationSeconds: number;
  logPath?: string;
  error?: string;
  classification?: string;
  remediation?: string;
  observations?: TestObservation[];
}

export interface TestPhaseSummary {
  total: number;
  passed: number;
  failed: number;
  durationSeconds: number;
  observationCount: number;
}

export interface TestObservation {
  icon?: string;
  prefix?: string;
  section?: string;
  text: string;
}

export interface TestExecutionListResponse {
  items: TestExecutionResult[];
  count: number;
}

// ── Tidiness Manager Types ─────────────────────────────────────────────

export interface TidinessBreakdown {
  lint_issues: number;
  type_issues: number;
  long_files: number;
  complex_functions: number;
  tech_debt_markers: number;
  duplication_issues: number;
}

export interface TidinessMetricsSummary {
  total_files: number;
  total_lines: number;
  avg_file_length: number;
  max_complexity: number;
  avg_complexity: number;
  duplication_pct: number;
}

export interface TidinessScoreResponse {
  scenario: string;
  score: number;
  violations: number;
  last_scan?: string;
  breakdown?: TidinessBreakdown;
  metrics?: TidinessMetricsSummary;
}

export interface TidinessIssue {
  id: number;
  scenario: string;
  file_path: string;
  category: string;
  severity: string;
  title: string;
  description: string;
  line_number?: number;
  column_number?: number;
  agent_notes?: string;
  remediation_steps?: string;
  status: string;
  created_at: string;
}

export interface TidinessStalenessInfo {
  last_scan_at?: string;
  is_stale: boolean;
  modified_files?: number;
  stale_reason?: string;
  rescan_command?: string;
}

export interface TidinessLightScanRequest {
  scenario_name: string;
  timeout_sec?: number;
  incremental?: boolean;
}

export interface TidinessFileMetric {
  path: string;
  lines: number;
  extension: string;
}

export interface TidinessLongFile {
  path: string;
  lines: number;
  threshold: number;
}

export interface TidinessLightScanResult {
  scenario: string;
  started_at: string;
  completed_at: string;
  duration_ms: number;
  file_metrics: TidinessFileMetric[];
  long_files: TidinessLongFile[];
  total_files: number;
  total_lines: number;
  lint_issues: number;
  type_issues: number;
  long_files_count: number;
}

export interface TidinessScenarioFileInfo {
  path: string;
  lines: number;
  totalIssues: number;
  visitCount: number;
}

export interface TidinessScenarioDetail {
  scenario: string;
  lightIssues: number;
  aiIssues: number;
  longFiles: number;
  files: TidinessScenarioFileInfo[];
}

// ── Tidiness Manager API Functions ─────────────────────────────────────

export async function fetchTidinessScore(
  scenarioName: string,
  repoId?: string,
  signal?: AbortSignal,
): Promise<TidinessScoreResponse> {
  const response = await tidinessClient.getScore(create(GetScoreRequestSchema, { scenarioName, repositoryId: repoId ?? "" }), { signal });
  return {
    scenario: response.scenario,
    score: response.score,
    violations: response.violations,
    last_scan: response.lastScan || undefined,
    breakdown: response.breakdown ? {
      lint_issues: response.breakdown.lintIssues, type_issues: response.breakdown.typeIssues,
      long_files: response.breakdown.longFiles, complex_functions: response.breakdown.complexFunctions,
      tech_debt_markers: response.breakdown.techDebtMarkers, duplication_issues: response.breakdown.duplicationIssues,
    } : undefined,
    metrics: response.metrics ? {
      total_files: response.metrics.totalFiles, total_lines: response.metrics.totalLines,
      avg_file_length: response.metrics.avgFileLength, max_complexity: response.metrics.maxComplexity,
      avg_complexity: response.metrics.avgComplexity, duplication_pct: response.metrics.duplicationPct,
    } : undefined,
  };
}

export async function fetchTidinessIssues(
  scenarioName: string,
  file?: string,
  category?: string,
  severity?: string,
  limit?: number,
  repoId?: string,
  signal?: AbortSignal,
): Promise<TidinessIssue[]> {
  const response = await tidinessClient.listIssues(create(ListIssuesRequestSchema, {
    scenarioName, repositoryId: repoId ?? "", file: file ?? "", category: category ?? "",
    severity: severity ?? "", limit: limit ?? 0,
  }), { signal });
  return response.issues.map((issue) => ({
    id: Number(issue.id), scenario: issue.scenario, file_path: issue.filePath, category: issue.category,
    severity: issue.severity, title: issue.title, description: issue.description,
    line_number: issue.lineNumber, column_number: issue.columnNumber,
    agent_notes: issue.agentNotes || undefined, remediation_steps: issue.remediationSteps || undefined,
    status: issue.status, created_at: issue.createdAt,
  }));
}

export async function fetchTidinessStaleness(
  scenarioName: string,
  repoId?: string,
  signal?: AbortSignal,
): Promise<TidinessStalenessInfo> {
  const response = await tidinessClient.getStaleness(create(GetStalenessRequestSchema, { scenarioName, repositoryId: repoId ?? "" }), { signal });
  return {
    last_scan_at: response.lastScanAt || undefined, is_stale: response.isStale,
    modified_files: response.modifiedFiles || undefined, stale_reason: response.staleReason || undefined,
    rescan_command: response.rescanCommand || undefined,
  };
}

export async function triggerTidinessLightScan(
  request: TidinessLightScanRequest,
  repoId?: string
): Promise<TidinessLightScanResult> {
  const response = await tidinessClient.triggerLightScan(create(TriggerLightScanRequestSchema, {
    repositoryId: repoId ?? "", scenarioName: request.scenario_name,
    timeoutSec: request.timeout_sec ?? 0, incremental: request.incremental ?? false,
  }));
  return {
    scenario: response.scenario, started_at: response.startedAt, completed_at: response.completedAt,
    duration_ms: Number(response.durationMs),
    file_metrics: response.fileMetrics.map((item) => ({ path: item.path, lines: item.lines, extension: item.extension })),
    long_files: response.longFiles.map((item) => ({ path: item.path, lines: item.lines, threshold: item.threshold })),
    total_files: response.totalFiles, total_lines: response.totalLines, lint_issues: response.lintIssues,
    type_issues: response.typeIssues, long_files_count: response.longFilesCount,
  };
}

export async function fetchTidinessScenarioDetail(
  scenarioName: string,
  repoId?: string,
  signal?: AbortSignal,
): Promise<TidinessScenarioDetail> {
  const response = await tidinessClient.getScenarioDetail(create(GetScenarioDetailRequestSchema, { scenarioName, repositoryId: repoId ?? "" }), { signal });
  return {
    scenario: response.scenario, lightIssues: response.lightIssues, aiIssues: response.aiIssues,
    longFiles: response.longFiles,
    files: response.files.map((item) => ({ path: item.path, lines: item.lines, totalIssues: item.totalIssues, visitCount: item.visitCount })),
  };
}
