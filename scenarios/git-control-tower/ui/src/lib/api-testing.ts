// ============================================================================
// Test Execution & Tidiness Manager — Types + API Functions
// ============================================================================

import { API_BASE, buildRepoHeaders, handleResponse, buildApiUrl } from "./api-internals";

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

// ── Test Execution API Functions ───────────────────────────────────────

export async function triggerTestExecution(
  request: TestExecutionRequest,
  repoId?: string
): Promise<TestExecutionResult> {
  const url = buildApiUrl("/repo/test-execution", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request),
  });
  return handleResponse<TestExecutionResult>(res);
}

export async function fetchTestExecutions(
  scenarioName: string,
  limit = 10,
  repoId?: string
): Promise<TestExecutionListResponse> {
  const params = new URLSearchParams({ scenarioName, limit: String(limit) });
  const url = buildApiUrl(`/repo/test-executions?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store",
  });
  return handleResponse<TestExecutionListResponse>(res);
}

export async function fetchTestExecution(
  id: string,
  repoId?: string
): Promise<TestExecutionResult> {
  const url = buildApiUrl(`/repo/test-executions/${encodeURIComponent(id)}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store",
  });
  return handleResponse<TestExecutionResult>(res);
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
  repoId?: string
): Promise<TidinessScoreResponse> {
  const params = new URLSearchParams({ scenarioName });
  const url = buildApiUrl(`/repo/tidiness-score?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store",
  });
  return handleResponse<TidinessScoreResponse>(res);
}

export async function fetchTidinessIssues(
  scenarioName: string,
  file?: string,
  category?: string,
  severity?: string,
  limit?: number,
  repoId?: string
): Promise<TidinessIssue[]> {
  const params = new URLSearchParams({ scenarioName });
  if (file) params.set("file", file);
  if (category) params.set("category", category);
  if (severity) params.set("severity", severity);
  if (limit) params.set("limit", String(limit));
  const url = buildApiUrl(`/repo/tidiness-issues?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store",
  });
  return handleResponse<TidinessIssue[]>(res);
}

export async function fetchTidinessStaleness(
  scenarioName: string,
  repoId?: string
): Promise<TidinessStalenessInfo> {
  const params = new URLSearchParams({ scenarioName });
  const url = buildApiUrl(`/repo/tidiness-staleness?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store",
  });
  return handleResponse<TidinessStalenessInfo>(res);
}

export async function triggerTidinessLightScan(
  request: TidinessLightScanRequest,
  repoId?: string
): Promise<TidinessLightScanResult> {
  const url = buildApiUrl("/repo/tidiness-scan", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request),
  });
  return handleResponse<TidinessLightScanResult>(res);
}

export async function fetchTidinessScenarioDetail(
  scenarioName: string,
  repoId?: string
): Promise<TidinessScenarioDetail> {
  const params = new URLSearchParams({ scenarioName });
  const url = buildApiUrl(`/repo/tidiness-scenario?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store",
  });
  return handleResponse<TidinessScenarioDetail>(res);
}
