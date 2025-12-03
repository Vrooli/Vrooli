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
}

export interface ScenarioFileResult {
  items: ScenarioFileNode[];
  hiddenCount: number;
}

export async function fetchScenarioFiles(
  name: string,
  params?: { path?: string; search?: string; limit?: number; includeHidden?: boolean }
): Promise<ScenarioFileResult> {
  const trimmed = name.trim();
  if (!trimmed) return { items: [], hiddenCount: 0 };

  const query = new URLSearchParams();
  if (params?.path) query.set("path", params.path);
  if (params?.search) query.set("search", params.search);
  if (params?.limit) query.set("limit", String(params.limit));
  if (params?.includeHidden) query.set("includeHidden", "1");

  const baseUrl = buildApiUrl(`/scenarios/${encodeURIComponent(trimmed)}/files`, { baseUrl: API_BASE });
  const url = query.toString() ? `${baseUrl}?${query.toString()}` : baseUrl;

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  const payload = await parseResponse<{ items: ScenarioFileNode[]; count: number; hiddenCount?: number }>(res);
  return { items: payload.items ?? [], hiddenCount: payload.hiddenCount ?? 0 };
}
