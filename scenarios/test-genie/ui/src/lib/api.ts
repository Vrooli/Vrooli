import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";
import {
  getSelfHealth as getSelfHealthConnect,
  listRunArtifacts,
  type ArtifactRef,
  type CatalogSummary,
  type ListRunArtifactsResponse,
  type PhaseReliability,
  type ProviderConformance,
  type ReliabilityLedger,
  type SelfHealth,
  type SelfHealthTrendPoint,
  type TrendDelta,
} from "../api/runs";

export type {
  CatalogSummary,
  PhaseReliability,
  ProviderConformance,
  ReliabilityLedger,
  SelfHealth,
  SelfHealthTrendPoint,
  TrendDelta,
};

// All scenario APIs hang off /api/v1.
export const API_BASE = resolveApiBase({ appendSuffix: true });

export function buildTestGenieApiUrl(path: string): string {
  return buildApiUrl(path, { baseUrl: API_BASE });
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function parseResponse<T>(res: Response): Promise<T>;
async function parseResponse(res: Response): Promise<unknown> {
  let payload: unknown;
  try {
    payload = await res.json();
  } catch {
    payload = null;
  }

  if (!res.ok) {
    const message =
      isRecord(payload) && typeof payload.error === "string"
        ? payload.error
        : `Request failed with status ${res.status}`;
    throw new Error(message);
  }

  return payload;
}

export interface PhaseSummary {
  total: number;
  passed: number;
  failed: number;
  durationSeconds: number;
  observationCount: number;
}

export interface PhaseExecutionResult {
  name: string;
  status: string;
  durationSeconds: number;
  logPath: string;
  error?: string;
  classification?: string;
  remediation?: string;
  runnabilityVerdict?: string;
  runnabilityReason?: string;
  findingSource?: string;
  findings?: Finding[];
  observations?: Array<{ type?: string; message?: string }>;
}

export interface PhaseFindingEvidence {
  kind?: string;
  summary?: string;
  locator?: string;
  /** Protobuf bytes are JSON encoded as base64. Producers may put safe metadata here. */
  payload?: string;
}

export interface Finding {
  stableId: string;
  code: string;
  source?: string;
  severity: string;
  class: string;
  locations?: string[];
  domains?: string[];
  message?: string;
  suggestion?: string;
  effort?: string;
  phase: string;
  gating: boolean;
  evidence?: PhaseFindingEvidence[];
  occurrences?: Array<{ phase: string; locations?: string[] }>;
}

export interface RemediationPhase {
  name: string;
  displayName?: string;
  provider?: string;
  docsPath?: string;
  status: string;
  runnabilityVerdict?: string;
  runnabilityReason?: string;
  remediation?: string;
  phasePresentation?: string;
  resultGating?: string;
}

export interface RemediationBundle {
  id: string;
  reason: string;
  findingIds: string[];
  phaseNames: string[];
  rank: number;
  gating: boolean;
}

export interface RemediationPlan {
  sourceExecutionId: string;
  sourceRunId: string;
  scenario: string;
  createdAt: string;
  phases: RemediationPhase[];
  findings: Finding[];
  bundles: RemediationBundle[];
	  requirements?: Array<{ id: string; title: string; description?: string; status?: string; liveStatus?: string; criticality?: string; validations?: string[] }>;
  degraded: boolean;
  degradedReasons?: string[];
}

export interface RemediationJob {
  id: string;
  scenario: string;
  status: "created" | "launch_pending" | "running" | "agent_completed" | "verification_running" | "verified" | "failed" | "cancelled" | "degraded";
  source: RemediationPlan;
  sourceHash: string;
  selectedFindingIds: string[];
  selectedRequirementIds?: string[];
  selectionHash: string;
  additionalContext?: string;
  attribution?: { taskId?: string; runId?: string; roleRef?: string; resolvedProfile?: string; outputReference?: string };
  verification?: { executionId?: string; runId?: string; completedAt?: string; degraded?: string; delta?: { resolved?: string[]; remaining?: string[]; new?: string[]; changedSeverity?: string[]; skipped?: string[]; unverifiable?: string[] }; requirementDelta?: { resolved?: string[]; remaining?: string[]; skipped?: string[]; unverifiable?: string[] } };
  failure?: string;
  attempts?: Array<{ id: string; kind: string; state: string; idempotencyKey: string; roleRef?: string; taskId?: string; runId?: string; detail?: string; createdAt: string }>;
}

export interface PhaseDescriptor {
  name: string;
  optional: boolean;
  description?: string;
  source?: string;
  defaultTimeoutSeconds?: number;
}

export interface PhaseToggle {
  disabled: boolean;
  reason?: string;
  owner?: string;
  addedAt?: string;
}

export interface PhaseSettingsResponse {
  items: PhaseDescriptor[];
  count: number;
  toggles?: Record<string, PhaseToggle>;
}

export interface ExecutionPlanPhase {
  name: string;
  description?: string;
  optional: boolean;
  estimatedDurationSeconds: number;
  timeoutSeconds: number;
  estimateSource: string;
  estimateConfidence: string;
  estimateSampleSize: number;
}

export interface ExecutionPlanSummary {
  phaseCount: number;
  estimatedDurationSeconds: number;
  timeoutSeconds: number;
}

export interface ExecutionPlanPreview {
  scenarioName: string;
  targetKind?: string;
  targetId?: string;
  presetUsed?: string;
  phases: ExecutionPlanPhase[];
  summary: ExecutionPlanSummary;
  warnings?: string[];
}

export interface SuiteExecutionResult {
	  executionId?: string;
	  runId?: string;
  scenarioName: string;
  startedAt: string;
  completedAt: string;
  success: boolean;
  preset?: string;
  phases: PhaseExecutionResult[];
  phaseSummary: PhaseSummary;
}

export interface ApiHealthResponse {
  status: string;
  service: string;
  timestamp: string;
  operations?: {
    lastExecution?: {
      executionId?: string;
      scenario: string;
      success: boolean;
      completedAt: string;
      startedAt: string;
      preset?: string;
      phaseSummary: PhaseSummary;
    };
  };
}

export interface ExecuteSuiteInput {
  scenarioName: string;
  /** Optional kind:id identity; scenarioName remains the legacy display alias. */
  target?: string;
  preset?: string;
  phases?: string[];
  skip?: string[];
  failFast?: boolean;
}

export interface ScenarioSummary {
  scenarioName: string;
  scenarioDescription?: string;
  scenarioStatus?: string;
  scenarioTags?: string[];
  totalExecutions: number;
  lastExecutionAt?: string;
  lastExecutionId?: string;
  lastRunId?: string;
  lastExecutionPreset?: string;
  lastExecutionSuccess?: boolean;
  lastExecutionPhases?: PhaseExecutionResult[];
  lastExecutionPhaseSummary?: PhaseSummary;
  lastFailureAt?: string;
  lastExperienceStatus?: string;
  lastExperienceAt?: string;
  lastExperienceRunId?: string;
  lastUIChangeAt?: string;
  /** Server-owned freshness classification for the stored experience verdict. */
  experienceFreshness?: "never_run" | "older_than_last_ui_change" | "evaluated_after_last_ui_change" | "ui_surface_unavailable" | "ui_change_time_unavailable";
}

export interface ValidationTarget {
  kind: string;
  id: string;
  root: string;
  target: string;
}

export async function fetchValidationTargets(): Promise<ValidationTarget[]> {
  const res = await fetch(buildApiUrl("/targets", { baseUrl: API_BASE }), {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  const payload = await parseResponse<{ items: ValidationTarget[]; count: number }>(res);
  return payload.items ?? [];
}

export async function fetchHealth(): Promise<ApiHealthResponse> {
  const url = buildTestGenieApiUrl("/health");
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  return parseResponse<ApiHealthResponse>(res);
}

export type RunArtifactRef = ArtifactRef;
export type RunArtifactCatalog = ListRunArtifactsResponse;

export interface GetSelfHealthOptions {
  windowDays?: number;
  skipConformance?: boolean;
  includeTrend?: boolean;
}

// Construct the owner transfer route from opaque catalog identifiers. The
// catalog itself stays path-free; route shape is kept in this API module.
export function buildOpaqueArtifactUrl(scenario: string, runId: string, artifactId: string): string {
  return `${API_BASE}/scenarios/${encodeURIComponent(scenario)}/runs/${encodeURIComponent(runId)}/artifacts/${encodeURIComponent(artifactId)}`;
}

// RunsService returns generated artifact descriptors. Artifact bytes remain on
// the owning opaque transfer route; the UI never derives storage locations.
export async function fetchRunArtifacts(scenario: string, runId: string): Promise<RunArtifactCatalog> {
  return listRunArtifacts(scenario, runId);
}

export async function getSelfHealth(options: GetSelfHealthOptions = {}): Promise<SelfHealth> {
  return getSelfHealthConnect(options);
}

export async function fetchExecutionHistory(params?: {
  scenario?: string;
  limit?: number;
  offset?: number;
}): Promise<SuiteExecutionResult[]> {
  const query = new URLSearchParams();
  if (params?.scenario) {
    query.set("scenario", params.scenario);
  }
  if (params?.limit) {
    query.set("limit", String(params.limit));
  }
  if (typeof params?.offset === "number" && params.offset > 0) {
    query.set("offset", String(params.offset));
  }
  const queryString = query.toString();
  const baseUrl = buildTestGenieApiUrl("/executions");
  const url = queryString ? `${baseUrl}?${queryString}` : baseUrl;

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  const payload = await parseResponse<{ items: SuiteExecutionResult[]; count: number }>(res);
  return payload.items ?? [];
}

export async function triggerSuiteExecution(input: ExecuteSuiteInput): Promise<SuiteExecutionResult> {
  const url = buildTestGenieApiUrl("/executions");
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input)
  });
  return parseResponse<SuiteExecutionResult>(res);
}

export async function previewSuiteExecution(input: ExecuteSuiteInput): Promise<ExecutionPlanPreview> {
  const url = buildTestGenieApiUrl("/executions/plan");
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input)
  });
  return parseResponse<ExecutionPlanPreview>(res);
}

export async function fetchPhaseSettings(): Promise<PhaseSettingsResponse> {
  const url = buildTestGenieApiUrl("/phases/settings");
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  return parseResponse<PhaseSettingsResponse>(res);
}

export async function updatePhaseSettings(phases: Record<string, PhaseToggle>): Promise<PhaseSettingsResponse> {
  const url = buildTestGenieApiUrl("/phases/settings");
  const res = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ phases })
  });
  return parseResponse<PhaseSettingsResponse>(res);
}

export async function fetchScenarioSummaries(): Promise<ScenarioSummary[]> {
  const url = buildApiUrl("/scenarios", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  const payload = await parseResponse<{ items: ScenarioSummary[]; count: number }>(res);
  return payload.items ?? [];
}

export async function fetchScenarioSummary(name: string): Promise<ScenarioSummary | null> {
  if (!name.trim()) {
    return null;
  }
  const url = buildApiUrl(`/scenarios/${encodeURIComponent(name.trim())}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  if (res.status === 404) {
    return null;
  }
  return parseResponse<ScenarioSummary>(res);
}

export async function fetchRemediationPlan(name: string, executionId: string): Promise<RemediationPlan> {
  const url = buildApiUrl(`/scenarios/${encodeURIComponent(name)}/remediation/plans/${encodeURIComponent(executionId)}`, { baseUrl: API_BASE });
  const res = await fetch(url, { headers: { "Content-Type": "application/json" }, cache: "no-store" });
  return parseResponse<RemediationPlan>(res);
}

export async function fetchRemediationJobs(name: string): Promise<RemediationJob[]> {
  const url = buildApiUrl(`/scenarios/${encodeURIComponent(name)}/remediation/jobs`, { baseUrl: API_BASE });
  const res = await fetch(url, { headers: { "Content-Type": "application/json" }, cache: "no-store" });
  const payload = await parseResponse<{ items: RemediationJob[] }>(res);
  return payload.items ?? [];
}

export async function createRemediationJob(name: string, input: { sourceExecutionId: string; findingIds: string[]; requirementIds?: string[]; roleRef: string; additionalContext?: string }): Promise<RemediationJob> {
  const url = buildApiUrl(`/scenarios/${encodeURIComponent(name)}/remediation/jobs`, { baseUrl: API_BASE });
  const res = await fetch(url, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(input) });
  return parseResponse<RemediationJob>(res);
}

export async function cancelRemediationJob(name: string, id: string): Promise<RemediationJob> {
  const url = buildApiUrl(`/scenarios/${encodeURIComponent(name)}/remediation/jobs/${encodeURIComponent(id)}/cancel`, { baseUrl: API_BASE });
  const res = await fetch(url, { method: "POST", headers: { "Content-Type": "application/json" } });
  return parseResponse<RemediationJob>(res);
}

export async function recoverRemediationJob(name: string, id: string): Promise<RemediationJob> {
  const url = buildApiUrl(`/scenarios/${encodeURIComponent(name)}/remediation/jobs/${encodeURIComponent(id)}/recover`, { baseUrl: API_BASE });
  const res = await fetch(url, { method: "POST", headers: { "Content-Type": "application/json" } });
  return parseResponse<RemediationJob>(res);
}

export async function retryRemediationJob(name: string, id: string): Promise<RemediationJob> {
  const url = buildApiUrl(`/scenarios/${encodeURIComponent(name)}/remediation/jobs/${encodeURIComponent(id)}/retry`, { baseUrl: API_BASE });
  const res = await fetch(url, { method: "POST", headers: { "Content-Type": "application/json" } });
  return parseResponse<RemediationJob>(res);
}

export async function refreshRemediationAgent(name: string, id: string): Promise<RemediationJob> {
  const url = buildApiUrl(`/scenarios/${encodeURIComponent(name)}/remediation/jobs/${encodeURIComponent(id)}/agent-status`, { baseUrl: API_BASE });
  const res = await fetch(url, { method: "POST", headers: { "Content-Type": "application/json" } });
  return parseResponse<RemediationJob>(res);
}

export async function verifyRemediationJob(name: string, id: string): Promise<RemediationJob> {
  const url = buildApiUrl(`/scenarios/${encodeURIComponent(name)}/remediation/jobs/${encodeURIComponent(id)}/verify`, { baseUrl: API_BASE });
  const res = await fetch(url, { method: "POST", headers: { "Content-Type": "application/json" } });
  return parseResponse<RemediationJob>(res);
}

// Requirements types
export interface RequirementsSummary {
  totalRequirements: number;
  totalValidations: number;
  completionRate: number;
  passRate: number;
  criticalGap: number;
  byLiveStatus: Record<string, number>;
  byDeclaredStatus: Record<string, number>;
}

export interface ModuleSnapshot {
  name: string;
  filePath: string;
  total: number;
  complete: number;
  inProgress: number;
  pending: number;
  completionRate: number;
  requirements?: RequirementItem[];
}

export interface RequirementItem {
  id: string;
  title: string;
  status: string;
  liveStatus: string;
  prdRef?: string;
  criticality?: string;
  description?: string;
  validations?: ValidationItem[];
}

export interface ValidationItem {
  type: string;
  ref: string;
  phase?: string;
  status: string;
  liveStatus: string;
}

export interface SyncStatus {
  enabled: boolean;
  lastSyncedAt?: string;
  filesUpdated: number;
  validationsAdded: number;
  validationsRemoved: number;
  statusesChanged: number;
  errorCount: number;
}

export interface RequirementsSnapshot {
  scenarioName: string;
  generatedAt: string;
  summary: RequirementsSummary;
  modules: ModuleSnapshot[];
  syncStatus?: SyncStatus;
}

export interface SyncPreviewResponse {
  scenarioName: string;
  changes: Array<{
    type: string;
    filePath: string;
    requirementId?: string;
    field?: string;
    oldValue?: string;
    newValue?: string;
  }>;
  summary: {
    filesAffected: number;
    statusesWouldChange: number;
    validationsWouldAdd: number;
    validationsWouldRemove: number;
  };
}

export async function fetchScenarioRequirements(name: string): Promise<RequirementsSnapshot | null> {
  if (!name.trim()) {
    return null;
  }
  const url = buildApiUrl(`/scenarios/${encodeURIComponent(name.trim())}/requirements`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  if (res.status === 404) {
    return null;
  }
  return parseResponse<RequirementsSnapshot>(res);
}

export interface SyncRequirementsInput {
  dryRun?: boolean;
  pruneOrphans?: boolean;
  discoverNew?: boolean;
}

export async function syncScenarioRequirements(
  name: string,
  input?: SyncRequirementsInput
): Promise<{ status: string; snapshot?: RequirementsSnapshot } | SyncPreviewResponse> {
  const url = buildApiUrl(`/scenarios/${encodeURIComponent(name.trim())}/requirements/sync`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input ?? {})
  });
  return parseResponse(res);
}

export interface ScenarioFileNode {
  path: string;
  name: string;
  isDir: boolean;
  coveragePct?: number;
}

export interface ScenarioFileResult {
  items: ScenarioFileNode[];
  hiddenCount: number;
}

export async function fetchScenarioFiles(
  name: string,
  params?: { path?: string; search?: string; limit?: number; includeHidden?: boolean; includeCoverage?: boolean }
): Promise<ScenarioFileResult> {
  const trimmed = name.trim();
  if (!trimmed) return { items: [], hiddenCount: 0 };

  const query = new URLSearchParams();
  if (params?.path) query.set("path", params.path);
  if (params?.search) query.set("search", params.search);
  if (params?.limit) query.set("limit", String(params.limit));
  if (params?.includeHidden) query.set("includeHidden", "1");
  if (params?.includeCoverage) query.set("includeCoverage", "1");

  const baseUrl = buildApiUrl(`/scenarios/${encodeURIComponent(trimmed)}/files`, { baseUrl: API_BASE });
  const url = query.toString() ? `${baseUrl}?${query.toString()}` : baseUrl;

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  const payload = await parseResponse<{ items: ScenarioFileNode[]; count: number; hiddenCount?: number }>(res);
  return { items: payload.items ?? [], hiddenCount: payload.hiddenCount ?? 0 };
}

export interface ScenarioCoverageResult {
  totalFiles: number;
  coveredFiles: number;
  overallCoverage: number;
}

export async function fetchScenarioCoverage(name: string): Promise<ScenarioCoverageResult> {
  const trimmed = name.trim();
  if (!trimmed) return { totalFiles: 0, coveredFiles: 0, overallCoverage: 0 };

  // Try to get coverage from files endpoint with coverage data
  const result = await fetchScenarioFiles(trimmed, { includeCoverage: true, limit: 500 });

  const filesWithCoverage = result.items.filter(
    (f) => !f.isDir && typeof f.coveragePct === "number" && f.coveragePct >= 0
  );

  const totalFiles = result.items.filter((f) => !f.isDir).length;
  const coveredFiles = filesWithCoverage.length;
  const overallCoverage = filesWithCoverage.length > 0
    ? filesWithCoverage.reduce((sum, f) => sum + (f.coveragePct ?? 0), 0) / filesWithCoverage.length
    : 0;

  return { totalFiles, coveredFiles, overallCoverage };
}

export interface AgentRole {
  id: string;
  label?: string;
  intent?: string;
  description?: string;
  source?: string;
}

export async function fetchAgentRoles(): Promise<AgentRole[]> {
  const url = buildApiUrl("/agents/roles", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  const payload = await parseResponse<{ items: AgentRole[]; count: number }>(res);
  return payload.items ?? [];
}

// ========================================
// Config API
// ========================================

export interface AppConfig {
  repoRoot: string;
  testGeniePath: string;
  testGenieCLI: string;
  scenariosPath: string;
  timestamp: string;
}

let cachedConfig: AppConfig | null = null;

export async function fetchAppConfig(): Promise<AppConfig> {
  // Return cached config if available (config doesn't change during session)
  if (cachedConfig) {
    return cachedConfig;
  }

  const url = buildApiUrl("/config", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  cachedConfig = await parseResponse<AppConfig>(res);
  return cachedConfig;
}

// Clear cached config (useful for testing or if server restarts)
export function clearConfigCache(): void {
  cachedConfig = null;
}

// ========================================
// Structured Agent Output
// ========================================

/**
 * Structured output format for agent responses.
 * Agents are instructed to output this JSON format wrapped in a ```json block.
 */
export interface AgentStructuredOutput {
  /** Overall status of the agent's work */
  status: "success" | "partial" | "failed";
  /** Brief description of what was accomplished */
  summary: string;
  /** Files that were created, modified, or deleted */
  filesChanged: Array<{
    path: string;
    action: "created" | "modified" | "deleted";
    rationale: string;
  }>;
  /** Summary of tests added */
  testsAdded: {
    count: number;
    byPhase: Record<string, number>;
  };
  /** Commands that were run and their results */
  commandsRun: Array<{
    command: string;
    result: "passed" | "failed";
    output?: string;
  }>;
  /** Coverage impact if measured */
  coverageImpact?: {
    before: number;
    after: number;
    delta: number;
  };
  /** Any blockers encountered */
  blockers: Array<{
    type: "missing_dependency" | "unclear_requirement" | "test_failure" | "other";
    description: string;
    suggestedResolution?: string;
  }>;
  /** Assumptions made during implementation */
  assumptions: string[];
  /** Suggested follow-up actions */
  nextSteps: string[];
}

function isStructuredOutputRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isStructuredStatus(value: unknown): value is AgentStructuredOutput["status"] {
  return value === "success" || value === "partial" || value === "failed";
}

function isFilesChanged(value: unknown): value is AgentStructuredOutput["filesChanged"] {
  return (
    Array.isArray(value) &&
    value.every(
      (item) =>
        isStructuredOutputRecord(item) &&
        typeof item.path === "string" &&
        (item.action === "created" || item.action === "modified" || item.action === "deleted") &&
        typeof item.rationale === "string"
    )
  );
}

function isTestsAdded(value: unknown): value is AgentStructuredOutput["testsAdded"] {
  if (!isStructuredOutputRecord(value) || typeof value.count !== "number" || !isStructuredOutputRecord(value.byPhase)) {
    return false;
  }

  return Object.values(value.byPhase).every((count) => typeof count === "number");
}

function isCommandsRun(value: unknown): value is AgentStructuredOutput["commandsRun"] {
  return (
    Array.isArray(value) &&
    value.every(
      (item) =>
        isStructuredOutputRecord(item) &&
        typeof item.command === "string" &&
        (item.result === "passed" || item.result === "failed") &&
        (item.output === undefined || typeof item.output === "string")
    )
  );
}

function isCoverageImpact(value: unknown): value is NonNullable<AgentStructuredOutput["coverageImpact"]> {
  return (
    isStructuredOutputRecord(value) &&
    typeof value.before === "number" &&
    typeof value.after === "number" &&
    typeof value.delta === "number"
  );
}

function isBlockers(value: unknown): value is AgentStructuredOutput["blockers"] {
  return (
    Array.isArray(value) &&
    value.every(
      (item) =>
        isStructuredOutputRecord(item) &&
        (item.type === "missing_dependency" ||
          item.type === "unclear_requirement" ||
          item.type === "test_failure" ||
          item.type === "other") &&
        typeof item.description === "string" &&
        (item.suggestedResolution === undefined || typeof item.suggestedResolution === "string")
    )
  );
}

/**
 * Attempt to parse structured JSON output from agent response.
 * Looks for a ```json code block and extracts the JSON.
 * Returns null if no valid JSON block is found.
 */
export function parseAgentStructuredOutput(output: string): AgentStructuredOutput | null {
  if (!output) return null;

  // Look for ```json ... ``` block
  const jsonBlockRegex = /```json\s*([\s\S]*?)```/;
  const match = output.match(jsonBlockRegex);

  if (!match || !match[1]) {
    return null;
  }

  try {
    const parsed: unknown = JSON.parse(match[1].trim());
    if (!isStructuredOutputRecord(parsed)) {
      return null;
    }

    const status = parsed.status;
    const summary = parsed.summary;
    const filesChanged = parsed.filesChanged;

    if (!isStructuredStatus(status) || typeof summary !== "string" || !isFilesChanged(filesChanged)) {
      return null;
    }

    const testsAddedValue = parsed.testsAdded;
    const testsAdded = isTestsAdded(testsAddedValue) ? testsAddedValue : { count: 0, byPhase: {} };

    const commandsRunValue = parsed.commandsRun;
    const commandsRun = isCommandsRun(commandsRunValue) ? commandsRunValue : [];

    const coverageImpactValue = parsed.coverageImpact;
    const coverageImpact = isCoverageImpact(coverageImpactValue) ? coverageImpactValue : undefined;

    const blockersValue = parsed.blockers;
    const blockers = isBlockers(blockersValue) ? blockersValue : [];

    const assumptions = Array.isArray(parsed.assumptions)
      ? parsed.assumptions.filter((item): item is string => typeof item === "string")
      : [];

    const nextSteps = Array.isArray(parsed.nextSteps)
      ? parsed.nextSteps.filter((item): item is string => typeof item === "string")
      : [];

    return {
      status,
      summary,
      filesChanged,
      testsAdded,
      commandsRun,
      coverageImpact,
      blockers,
      assumptions,
      nextSteps,
    };
  } catch {
    // JSON parsing failed
    return null;
  }
}

/**
 * Check if an agent's output contains structured JSON.
 */
export function hasStructuredOutput(output: string): boolean {
  if (!output) return false;
  return /```json\s*\{[\s\S]*\}\s*```/.test(output);
}

// PlaybooksClaim mirrors the API's claimDTO (playbooks_claims_handlers.go):
// the concurrency-guard state for a scenario's playbooks run.
export interface PlaybooksClaim {
  scenario_name: string;
  run_id: string;
  mode: string;
  started_by: string;
  acquired_at: string;
  heartbeat_at: string;
  expires_at: string;
  alive: boolean;
}

// fetchPlaybooksClaim returns the active playbooks claim for a scenario, or
// null when none is held. GET /playbooks/claims/{scenario} → { claim }.
export async function fetchPlaybooksClaim(scenario: string): Promise<PlaybooksClaim | null> {
  const url = buildTestGenieApiUrl(`/playbooks/claims/${encodeURIComponent(scenario)}`);
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  const payload = await parseResponse<{ claim: PlaybooksClaim | null }>(res);
  return payload.claim ?? null;
}

// releasePlaybooksClaim force-breaks the active claim for a scenario and
// returns the released claim. POST /playbooks/claims/{scenario}/release → { released }.
export async function releasePlaybooksClaim(scenario: string): Promise<PlaybooksClaim> {
  const url = buildTestGenieApiUrl(`/playbooks/claims/${encodeURIComponent(scenario)}/release`);
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" }
  });
  const payload = await parseResponse<{ released: PlaybooksClaim }>(res);
  return payload.released;
}
