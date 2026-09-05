// ============================================================================
// Auditor — Types + API Functions
// ============================================================================

import { API_BASE, buildRepoHeaders, buildApiUrl, handleResponse } from "./api-internals";

// ── Auditor Types ──────────────────────────────────────────────────────

export interface AuditorViolation {
  id: string;
  scenario_name: string;
  type: string;
  severity: string;
  title: string;
  description: string;
  file_path: string;
  line_number: number;
  code_snippet?: string;
  recommendation: string;
  standard: string;
  discovered_at: string;
  source?: string;
  metadata?: Record<string, unknown>;
}

export interface AuditorCheckResult {
  check_id: string;
  status: string;
  scan_type: string;
  started_at: string;
  completed_at: string;
  duration_seconds: number;
  files_scanned: number;
  violations: AuditorViolation[];
  statistics: Record<string, number>;
  message: string;
  scenario_name?: string;
  summary?: AuditorViolationSummary;
}

export interface AuditorViolationSummary {
  total: number;
  by_severity: Record<string, number>;
  by_rule?: { rule_id: string; count: number }[];
  highest_severity: string;
  top_violations?: { id: string; severity: string; rule_id: string; title: string; file_path: string }[];
  recommended_steps?: string[];
  generated_at: string;
}

export interface AuditorJobStatus {
  id: string;
  scenario: string;
  scan_type: string;
  status: string;
  started_at: string;
  completed_at?: string;
  elapsed_seconds: number;
  total_scenarios: number;
  processed_scenarios: number;
  processed_files: number;
  total_files: number;
  current_scenario?: string;
  current_file?: string;
  message?: string;
  error?: string;
  result?: AuditorCheckResult;
}

export interface AuditorCheckJobResponse {
  job_id: string;
  status: AuditorJobStatus;
}

export interface AuditorRule {
  id: string;
  name: string;
  description: string;
  category: string;
  severity: string;
  enabled: boolean;
  standard: string;
  targets: string[];
}

export interface AuditorRulesListResponse {
  rules: Record<string, AuditorRule>;
  categories?: Record<string, unknown>;
  count: number;
  total: number;
}

export interface AuditorFixRequest {
  scenario_names: string[];
  rule_ids: string[];
  dry_run?: boolean;
}

export interface AuditorFixChange {
  type: string;
  detail: string;
}

export interface AuditorFixResult {
  scenario_name: string;
  rule_id: string;
  fixed: boolean;
  file_path: string;
  changes: AuditorFixChange[];
  error?: string;
}

export interface AuditorFixResponse {
  results: AuditorFixResult[];
  count: number;
  unfixable_rules: string[];
  errors: string[];
}

// ── Auditor API Functions ──────────────────────────────────────────────

export async function startAuditorCheck(scenarioName: string, checkType = "full", repoId?: string): Promise<AuditorCheckJobResponse> {
  const res = await fetch(buildApiUrl("/repo/rules-run", { baseUrl: API_BASE }), {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify({ scenario_name: scenarioName, check_type: checkType }),
  });
  return handleResponse<AuditorCheckJobResponse>(res);
}

export async function pollAuditorJob(jobId: string, repoId?: string): Promise<AuditorJobStatus> {
  const res = await fetch(buildApiUrl(`/repo/rules-job/${encodeURIComponent(jobId)}`, { baseUrl: API_BASE }), {
    headers: buildRepoHeaders(repoId),
  });
  return handleResponse<AuditorJobStatus>(res);
}

export async function fetchAuditorRules(repoId?: string): Promise<AuditorRulesListResponse> {
  const res = await fetch(buildApiUrl("/repo/rules", { baseUrl: API_BASE }), {
    headers: buildRepoHeaders(repoId),
  });
  return handleResponse<AuditorRulesListResponse>(res);
}

export async function applyAuditorFix(req: AuditorFixRequest, repoId?: string): Promise<AuditorFixResponse> {
  const res = await fetch(buildApiUrl("/repo/rules-fix", { baseUrl: API_BASE }), {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(req),
  });
  return handleResponse<AuditorFixResponse>(res);
}

export async function fetchAuditorViolations(scenarioName: string, repoId?: string): Promise<AuditorViolation[]> {
  const params = new URLSearchParams({ scenarioName });
  const res = await fetch(buildApiUrl(`/repo/rules-violations?${params}`, { baseUrl: API_BASE }), {
    headers: buildRepoHeaders(repoId),
  });
  const data = await handleResponse<{ violations?: AuditorViolation[] } | AuditorViolation[]>(res);
  return Array.isArray(data) ? data : (data.violations ?? []);
}
