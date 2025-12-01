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
