import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

// All scenario APIs hang off /api/v1.
const API_BASE = resolveApiBase({ appendSuffix: true });

async function parseResponse<T>(res: Response): Promise<T> {
  let payload: unknown;
  try {
    payload = await res.json();
  } catch {
    payload = null;
  }

  if (!res.ok) {
    const message =
      payload && typeof payload === "object" && payload !== null && "error" in payload
        ? String((payload as Record<string, unknown>).error)
        : `Request failed with status ${res.status}`;
    throw new Error(message);
  }

  return payload as T;
}

export interface QueueSnapshot {
  total: number;
  queued: number;
  delegated: number;
  running: number;
  completed: number;
  failed: number;
  pending: number;
  timestamp: string;
  oldestQueuedAt?: string;
  oldestQueuedAgeSeconds?: number;
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
  observations?: string[];
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

export interface SuiteExecutionResult {
  executionId?: string;
  suiteRequestId?: string;
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
    queue?: QueueSnapshot;
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

export interface SuiteRequest {
  id: string;
  scenarioName: string;
  requestedTypes: string[];
  coverageTarget: number;
  priority: string;
  status: string;
  notes?: string;
  delegationIssueId?: string;
  createdAt: string;
  updatedAt: string;
  estimatedQueueTimeSeconds?: number;
}

export interface QueueSuiteRequestInput {
  scenarioName: string;
  requestedTypes: string[];
  coverageTarget: number;
  priority: string;
  notes?: string;
}

export interface ExecuteSuiteInput {
  scenarioName: string;
  preset?: string;
  phases?: string[];
  skip?: string[];
  failFast?: boolean;
  suiteRequestId?: string;
}

export interface ScenarioSummary {
  scenarioName: string;
  scenarioDescription?: string;
  scenarioStatus?: string;
  scenarioTags?: string[];
  pendingRequests: number;
  totalRequests: number;
  lastRequestAt?: string;
  lastRequestPriority?: string;
  lastRequestStatus?: string;
  lastRequestNotes?: string;
  lastRequestCoverageTarget?: number;
  lastRequestTypes?: string[];
  totalExecutions: number;
  lastExecutionAt?: string;
  lastExecutionId?: string;
  lastExecutionPreset?: string;
  lastExecutionSuccess?: boolean;
  lastExecutionPhases?: PhaseExecutionResult[];
  lastExecutionPhaseSummary?: PhaseSummary;
  lastFailureAt?: string;
}

export async function fetchHealth(): Promise<ApiHealthResponse> {
  const url = buildApiUrl("/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  return parseResponse<ApiHealthResponse>(res);
}

export async function fetchSuiteRequests(): Promise<SuiteRequest[]> {
  const url = buildApiUrl("/suite-requests", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  const payload = await parseResponse<{ items: SuiteRequest[]; count: number }>(res);
  return payload.items ?? [];
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
  const baseUrl = buildApiUrl("/executions", { baseUrl: API_BASE });
  const url = queryString ? `${baseUrl}?${queryString}` : baseUrl;

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  const payload = await parseResponse<{ items: SuiteExecutionResult[]; count: number }>(res);
  return payload.items ?? [];
}

export async function queueSuiteRequest(input: QueueSuiteRequestInput): Promise<SuiteRequest> {
  const url = buildApiUrl("/suite-requests", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input)
  });
  return parseResponse<SuiteRequest>(res);
}

export async function triggerSuiteExecution(input: ExecuteSuiteInput): Promise<SuiteExecutionResult> {
  const url = buildApiUrl("/executions", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input)
  });
  return parseResponse<SuiteExecutionResult>(res);
}

export async function fetchPhaseSettings(): Promise<PhaseSettingsResponse> {
  const url = buildApiUrl("/phases/settings", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  return parseResponse<PhaseSettingsResponse>(res);
}

export async function updatePhaseSettings(phases: Record<string, PhaseToggle>): Promise<PhaseSettingsResponse> {
  const url = buildApiUrl("/phases/settings", { baseUrl: API_BASE });
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

export interface AgentModel {
  id: string;
  name?: string;
  displayName?: string;
  provider?: string;
  description?: string;
  source?: string;
}

export interface SpawnAgentsRequest {
  prompts: string[];
  preamble?: string;          // Immutable safety preamble (server validates this)
  model: string;
  concurrency?: number;
  maxTurns?: number;
  timeoutSeconds?: number;
  allowedTools?: string[];
  skipPermissions?: boolean;
  scenario?: string;
  scope?: string[];
  phases?: string[];
}

export interface SpawnAgentsResult {
  promptIndex: number;
  agentId?: string;
  status: string;
  sessionId?: string;
  output?: string;
  error?: string;
}

export interface SpawnAgentsResponse {
  items: SpawnAgentsResult[];
  count: number;
  capped?: boolean;
  scopeConflicts?: string[];
  validationError?: string;
}

export interface ActiveAgent {
  id: string;
  sessionId?: string;
  scenario: string;
  scope: string[];
  phases?: string[];
  model: string;
  status: "pending" | "running" | "completed" | "failed" | "timeout" | "stopped";
  startedAt: string;
  completedAt?: string;
  promptHash: string;
  promptIndex: number;
  output?: string;
  error?: string;
}

export interface ScopeLock {
  scenario: string;
  paths: string[];
  agentId: string;
  acquiredAt: string;
  expiresAt: string;
}

export interface ActiveAgentsResponse {
  items: ActiveAgent[];
  count: number;
  activeLocks: ScopeLock[];
}

export interface ConflictDetail {
  path: string;
  lockedBy: {
    path: string;
    agentId: string;
    scenario: string;
    startedAt: string;
    expiresAt: string;
    model?: string;
    phases?: string[];
    status?: string;
    runningSeconds?: number;
  };
}

export interface ScopeConflictResponse {
  hasConflicts: boolean;
  conflicts: ConflictDetail[];
}

export interface BlockedCommandsResponse {
  blockedPatterns: Array<{ pattern: string; description: string }>;
  safeDefaults: string[];
  safeBashPatterns: string[];
}

export async function fetchAgentModels(provider = "openrouter"): Promise<AgentModel[]> {
  const url = buildApiUrl("/agents/models", { baseUrl: API_BASE });
  const finalUrl = `${url}?provider=${encodeURIComponent(provider)}`;
  const res = await fetch(finalUrl, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  const payload = await parseResponse<{ items: AgentModel[]; count: number }>(res);
  return payload.items ?? [];
}

export async function spawnAgents(payload: SpawnAgentsRequest): Promise<SpawnAgentsResponse> {
  const url = buildApiUrl("/agents/spawn", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
  return parseResponse<SpawnAgentsResponse>(res);
}

export async function fetchActiveAgents(includeAll = false): Promise<ActiveAgentsResponse> {
  const url = buildApiUrl("/agents/active", { baseUrl: API_BASE });
  const finalUrl = includeAll ? `${url}?all=true` : url;
  const res = await fetch(finalUrl, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  return parseResponse<ActiveAgentsResponse>(res);
}

export async function fetchAgent(id: string): Promise<ActiveAgent> {
  const url = buildApiUrl(`/agents/${encodeURIComponent(id)}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  return parseResponse<ActiveAgent>(res);
}

export async function stopAgent(id: string): Promise<{ message: string; agent: ActiveAgent }> {
  const url = buildApiUrl(`/agents/${encodeURIComponent(id)}/stop`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" }
  });
  return parseResponse<{ message: string; agent: ActiveAgent }>(res);
}

export interface StopAllAgentsResponse {
  message: string;
  stoppedCount: number;
  stoppedIds: string[];
}

export async function stopAllAgents(): Promise<StopAllAgentsResponse> {
  const url = buildApiUrl("/agents/stop-all", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" }
  });
  return parseResponse<StopAllAgentsResponse>(res);
}

export async function checkScopeConflicts(scenario: string, scope: string[]): Promise<ScopeConflictResponse> {
  const url = buildApiUrl("/agents/check-conflicts", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ scenario, scope })
  });
  return parseResponse<ScopeConflictResponse>(res);
}

export async function fetchBlockedCommands(): Promise<BlockedCommandsResponse> {
  const url = buildApiUrl("/agents/blocked-commands", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  return parseResponse<BlockedCommandsResponse>(res);
}

export async function cleanupAgents(olderThanMinutes = 60): Promise<{ removed: number }> {
  const url = buildApiUrl("/agents/cleanup", { baseUrl: API_BASE });
  const finalUrl = `${url}?olderThanMinutes=${olderThanMinutes}`;
  const res = await fetch(finalUrl, {
    method: "POST",
    headers: { "Content-Type": "application/json" }
  });
  return parseResponse<{ removed: number }>(res);
}

// Path validation for prompts
export interface PathValidationResult {
  path: string;
  description: string;
  exists: boolean;
  isDirectory: boolean;
  required: boolean;
}

export interface ValidatePathsResponse {
  valid: boolean;
  paths: PathValidationResult[];
  warnings: string[];
  errors: string[];
}

export async function validatePromptPaths(scenario: string, phases: string[]): Promise<ValidatePathsResponse> {
  const url = buildApiUrl("/agents/validate-paths", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ scenario, phases })
  });
  return parseResponse<ValidatePathsResponse>(res);
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
  securityModel: string;
  directoryScoping: boolean;
  pathValidation: boolean;
  bashAllowlistOnly: boolean;
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
    const parsed = JSON.parse(match[1].trim());

    // Validate required fields exist
    if (
      typeof parsed.status !== "string" ||
      typeof parsed.summary !== "string" ||
      !Array.isArray(parsed.filesChanged)
    ) {
      return null;
    }

    // Return with defaults for optional fields
    return {
      status: parsed.status,
      summary: parsed.summary,
      filesChanged: parsed.filesChanged || [],
      testsAdded: parsed.testsAdded || { count: 0, byPhase: {} },
      commandsRun: parsed.commandsRun || [],
      coverageImpact: parsed.coverageImpact,
      blockers: parsed.blockers || [],
      assumptions: parsed.assumptions || [],
      nextSteps: parsed.nextSteps || [],
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
