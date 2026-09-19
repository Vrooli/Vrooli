// ============================================================================
// Auditor — Types + API Functions
// ============================================================================

import { create } from "@bufbuild/protobuf";
import {
  ApplyFixRequestSchema,
  FixRequestSchema,
  GetJobStatusRequestSchema,
  ListRulesRequestSchema,
  ListViolationsRequestSchema,
  StartCheckRequestSchema,
} from "@vrooli/proto-types/git-control-tower/v1/auditor/auditor_pb";
import { auditorClient } from "./connect";
import { issueMutationIntentForOperation } from "./api-core";

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
  const response = await auditorClient.startCheck(create(StartCheckRequestSchema, {
    repositoryId: repoId ?? "",
    scenarioName,
    checkType,
  }));
  return {
    job_id: response.jobId,
    status: auditorJobStatusFromProto(response.status),
  };
}

export async function pollAuditorJob(jobId: string, repoId?: string): Promise<AuditorJobStatus> {
  const response = await auditorClient.getJobStatus(create(GetJobStatusRequestSchema, {
    repositoryId: repoId ?? "",
    jobId,
  }));
  return auditorJobStatusFromProto(response);
}

export async function fetchAuditorRules(repoId?: string): Promise<AuditorRulesListResponse> {
  const response = await auditorClient.listRules(create(ListRulesRequestSchema, {
    repositoryId: repoId ?? "",
  }));
  return {
    rules: Object.fromEntries(response.rules.map((rule) => [rule.id, {
      id: rule.id,
      name: rule.name,
      description: rule.description,
      category: rule.category,
      severity: rule.severity,
      enabled: rule.enabled,
      standard: rule.standard,
      targets: [...rule.targets],
    }])),
    categories: keyValuesToRecord(response.categories),
    count: response.count,
    total: response.total,
  };
}

export async function applyAuditorFix(req: AuditorFixRequest, repoId?: string): Promise<AuditorFixResponse> {
	const scenarioNames = [...new Set(req.scenario_names.map((value) => value.trim()).filter(Boolean))].sort();
	const ruleIds = [...new Set(req.rule_ids.map((value) => value.trim()).filter(Boolean))].sort();
	const subjectContext = JSON.stringify({ scenario_names: scenarioNames, rule_ids: ruleIds });
	if (req.dry_run) {
		return auditorFixResponseFromProto(await auditorClient.previewFix(create(FixRequestSchema, {
			repositoryId: repoId || "",
			scenarioNames,
			ruleIds,
		})));
	}
	const intent = await issueMutationIntentForOperation("repo.auditor.fix", repoId, subjectContext);
	return auditorFixResponseFromProto(await auditorClient.applyFix(create(ApplyFixRequestSchema, {
		repositoryId: intent.repositoryId,
		scenarioNames,
		ruleIds,
		intentId: intent.intentId,
		expectedRevision: intent.expectedRevision,
		subjectDigest: intent.subjectDigest,
	})));
}

function auditorFixResponseFromProto(response: {
	results: Array<{ scenarioName: string; ruleId: string; fixed: boolean; filePath: string; changes: Array<{ type: string; detail: string }>; error: string }>;
	count: number;
	unfixableRules: string[];
	errors: string[];
}): AuditorFixResponse {
	return {
		results: response.results.map((item) => ({
			scenario_name: item.scenarioName,
			rule_id: item.ruleId,
			fixed: item.fixed,
			file_path: item.filePath,
			changes: item.changes.map((change) => ({ type: change.type, detail: change.detail })),
			error: item.error || undefined,
		})),
		count: response.count,
		unfixable_rules: [...response.unfixableRules],
		errors: [...response.errors],
	};
}

export async function fetchAuditorViolations(scenarioName: string, repoId?: string): Promise<AuditorViolation[]> {
  const response = await auditorClient.listViolations(create(ListViolationsRequestSchema, {
    repositoryId: repoId ?? "",
    scenarioName,
  }));
  return response.violations.map(auditorViolationFromProto);
}

function auditorJobStatusFromProto(status?: {
  id: string;
  scenario: string;
  scanType: string;
  status: string;
  startedAt: string;
  completedAt: string;
  elapsedSeconds: number;
  totalScenarios: number;
  processedScenarios: number;
  processedFiles: number;
  totalFiles: number;
  currentScenario: string;
  currentFile: string;
  message: string;
  error: string;
  result?: {
    checkId: string;
    status: string;
    scanType: string;
    startedAt: string;
    completedAt: string;
    durationSeconds: number;
    filesScanned: number;
    violations: Array<Parameters<typeof auditorViolationFromProto>[0]>;
    statistics: Array<{ key: string; count: number }>;
    message: string;
    scenarioName: string;
    summary?: {
      total: number;
      bySeverity: Array<{ key: string; count: number }>;
      byRule: Array<{ ruleId: string; count: number }>;
      highestSeverity: string;
      topViolations: Array<{ id: string; severity: string; ruleId: string; title: string; filePath: string }>;
      recommendedSteps: string[];
      generatedAt: string;
    };
  };
}): AuditorJobStatus {
  const result = status?.result;
  return {
    id: status?.id ?? "",
    scenario: status?.scenario ?? "",
    scan_type: status?.scanType ?? "",
    status: status?.status ?? "",
    started_at: status?.startedAt ?? "",
    completed_at: status?.completedAt || undefined,
    elapsed_seconds: status?.elapsedSeconds ?? 0,
    total_scenarios: status?.totalScenarios ?? 0,
    processed_scenarios: status?.processedScenarios ?? 0,
    processed_files: status?.processedFiles ?? 0,
    total_files: status?.totalFiles ?? 0,
    current_scenario: status?.currentScenario || undefined,
    current_file: status?.currentFile || undefined,
    message: status?.message || undefined,
    error: status?.error || undefined,
    result: result ? {
      check_id: result.checkId,
      status: result.status,
      scan_type: result.scanType,
      started_at: result.startedAt,
      completed_at: result.completedAt,
      duration_seconds: result.durationSeconds,
      files_scanned: result.filesScanned,
      violations: result.violations.map(auditorViolationFromProto),
      statistics: Object.fromEntries(result.statistics.map((entry) => [entry.key, entry.count])),
      message: result.message,
      scenario_name: result.scenarioName || undefined,
      summary: result.summary ? {
        total: result.summary.total,
        by_severity: Object.fromEntries(result.summary.bySeverity.map((entry) => [entry.key, entry.count])),
        by_rule: result.summary.byRule.map((entry) => ({ rule_id: entry.ruleId, count: entry.count })),
        highest_severity: result.summary.highestSeverity,
        top_violations: result.summary.topViolations.map((entry) => ({
          id: entry.id, severity: entry.severity, rule_id: entry.ruleId,
          title: entry.title, file_path: entry.filePath,
        })),
        recommended_steps: [...result.summary.recommendedSteps],
        generated_at: result.summary.generatedAt,
      } : undefined,
    } : undefined,
  };
}

function auditorViolationFromProto(violation: {
  id: string;
  scenarioName: string;
  type: string;
  severity: string;
  title: string;
  description: string;
  filePath: string;
  lineNumber: number;
  codeSnippet: string;
  recommendation: string;
  standard: string;
  discoveredAt: string;
  source: string;
  metadata: Array<{ key: string; value: string }>;
}): AuditorViolation {
  return {
    id: violation.id,
    scenario_name: violation.scenarioName,
    type: violation.type,
    severity: violation.severity,
    title: violation.title,
    description: violation.description,
    file_path: violation.filePath,
    line_number: violation.lineNumber,
    code_snippet: violation.codeSnippet || undefined,
    recommendation: violation.recommendation,
    standard: violation.standard,
    discovered_at: violation.discoveredAt,
    source: violation.source || undefined,
    metadata: keyValuesToRecord(violation.metadata),
  };
}

function keyValuesToRecord(values: Array<{ key: string; value: string }>): Record<string, unknown> {
  return Object.fromEntries(values.map(({ key, value }) => {
    try {
      return [key, JSON.parse(value)];
    } catch {
      return [key, value];
    }
  }));
}
