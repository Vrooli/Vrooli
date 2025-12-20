import { useCallback, useEffect, useRef, useState } from "react";
import { getApiBaseUrl } from "../lib/utils";
import type {
  AgentProfile,
  ApprovalResult,
  ApproveRequest,
  CreateProfileRequest,
  CreateRunRequest,
  CreateTaskRequest,
  DiffResult,
  DiffFile,
  DiffStats,
  ErrorDetails,
  GetRunDiffResponse,
  GetRunEventsResponse,
  GetRunnerStatusResponse,
  HealthStatus,
  ListProfilesResponse,
  ListRunsResponse,
  ListTasksResponse,
  ProbeResult,
  ProbeRunnerResponse,
  RejectRequest,
  Run,
  RunEvent,
  RunEventData,
  RunSummary,
  RunnerStatus,
  RunnerType,
  Task,
} from "../types";

interface ApiState<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
}

function useApiState<T>(initialData: T | null = null): ApiState<T> & {
  setData: (data: T | null) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
} {
  const [data, setData] = useState<T | null>(initialData);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  return { data, loading, error, setData, setLoading, setError };
}

async function apiRequest<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const baseUrl = getApiBaseUrl();
  const url = endpoint.startsWith("http") ? endpoint : baseUrl + endpoint;

  const response = await fetch(url, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options.headers,
    },
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(
      errorData.message || errorData.userMessage || "Request failed: " + response.status
    );
  }

  if (response.status === 204) {
    return {} as T;
  }

  return response.json();
}

function normalizeList<T>(data: unknown, key: string): T[] {
  if (Array.isArray(data)) {
    return data as T[];
  }
  if (data && typeof data === "object") {
    const record = data as Record<string, unknown>;
    const value = record[key];
    if (Array.isArray(value)) {
      return value as T[];
    }
  }
  return [];
}

function unwrapField<T>(data: unknown, key: string): T {
  if (data && typeof data === "object") {
    const record = data as Record<string, unknown>;
    if (key in record) {
      return record[key] as T;
    }
  }
  return data as T;
}

function asRecord(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }
  return value as Record<string, unknown>;
}

function getField<T>(record: Record<string, unknown> | null, ...keys: string[]): T | undefined {
  if (!record) return undefined;
  for (const key of keys) {
    if (key in record) {
      return record[key] as T;
    }
  }
  return undefined;
}

function toNumber(value: unknown, fallback = 0): number {
  if (typeof value === "number") return value;
  if (typeof value === "string") {
    const parsed = Number(value);
    return Number.isNaN(parsed) ? fallback : parsed;
  }
  return fallback;
}

function parseDurationToNanoseconds(value: unknown): number | undefined {
  if (typeof value === "number") return value;
  if (typeof value === "string") {
    const match = value.match(/^(-?\d+(?:\.\d+)?)s$/);
    if (!match) return undefined;
    const seconds = Number(match[1]);
    if (Number.isNaN(seconds)) return undefined;
    return Math.round(seconds * 1_000_000_000);
  }
  const record = asRecord(value);
  if (record) {
    const seconds = toNumber(getField(record, "seconds"), 0);
    const nanos = toNumber(getField(record, "nanos"), 0);
    if (seconds || nanos) {
      return Math.round(seconds * 1_000_000_000 + nanos);
    }
  }
  return undefined;
}

function normalizeEnumValue(value: unknown, prefix: string): string | undefined {
  if (typeof value !== "string") return undefined;
  if (value.startsWith(prefix)) {
    return value.slice(prefix.length).toLowerCase();
  }
  if (value.includes(prefix)) {
    const parts = value.split(prefix);
    return parts[parts.length - 1]?.toLowerCase();
  }
  return value.toLowerCase();
}

function normalizeRunnerType(value: unknown): RunnerType | undefined {
  const normalized = normalizeEnumValue(value, "RUNNER_TYPE_");
  if (!normalized) return undefined;
  return normalized.replace(/_/g, "-") as RunnerType;
}

function mapRunSummary(value: unknown): RunSummary | undefined {
  const record = asRecord(value);
  if (!record) return value as RunSummary | undefined;
  return {
    description: getField(record, "description"),
    filesModified: getField(record, "filesModified", "files_modified"),
    filesCreated: getField(record, "filesCreated", "files_created"),
    filesDeleted: getField(record, "filesDeleted", "files_deleted"),
    tokensUsed: toNumber(getField(record, "tokensUsed", "tokens_used"), 0),
    turnsUsed: toNumber(getField(record, "turnsUsed", "turns_used"), 0),
    costEstimate: toNumber(getField(record, "costEstimate", "cost_estimate"), 0),
  };
}

function mapErrorDetails(value: unknown): ErrorDetails | undefined {
  const record = asRecord(value);
  if (!record) return value as ErrorDetails | undefined;
  const conflictsRaw = getField<unknown[]>(record, "conflicts") ?? [];
  const conflicts = Array.isArray(conflictsRaw)
    ? conflictsRaw
        .map((conflict) => {
          const conflictRecord = asRecord(conflict);
          if (!conflictRecord) return null;
          return {
            sandboxId: String(getField(conflictRecord, "sandboxId", "sandbox_id") ?? ""),
            scope: String(getField(conflictRecord, "scope") ?? ""),
            conflictType: String(getField(conflictRecord, "conflictType", "conflict_type") ?? ""),
          };
        })
        .filter((conflict) => conflict !== null)
    : [];

  return {
    operation: getField(record, "operation"),
    is_transient: getField(record, "is_transient", "isTransient"),
    can_retry: getField(record, "can_retry", "canRetry"),
    sandbox_id: getField(record, "sandbox_id", "sandboxId"),
    cause: getField(record, "cause"),
    conflicts,
  };
}

function mapRunEventData(value: unknown): RunEventData {
  const record = asRecord(value);
  if (!record) return {} as RunEventData;
  return {
    level: getField(record, "level"),
    message: getField(record, "message"),
    role: getField(record, "role"),
    content: getField(record, "content"),
    toolName: getField(record, "toolName", "tool_name"),
    input: getField(record, "input"),
    toolCallId: getField(record, "toolCallId", "tool_call_id"),
    output: getField(record, "output"),
    error: getField(record, "error"),
    success: getField(record, "success"),
    oldStatus: normalizeEnumValue(getField(record, "oldStatus", "old_status"), "RUN_STATUS_") ??
      getField(record, "oldStatus", "old_status"),
    newStatus: normalizeEnumValue(getField(record, "newStatus", "new_status"), "RUN_STATUS_") ??
      getField(record, "newStatus", "new_status"),
    reason: getField(record, "reason"),
    name: getField(record, "name"),
    value: toNumber(getField(record, "value"), 0),
    unit: getField(record, "unit"),
    tags: getField(record, "tags"),
    type: getField(record, "type"),
    path: getField(record, "path"),
    size: toNumber(getField(record, "size"), 0),
    mimeType: getField(record, "mimeType", "mime_type"),
    code: getField(record, "code"),
    retryable: getField(record, "retryable"),
    recovery: normalizeEnumValue(getField(record, "recovery"), "RECOVERY_ACTION_") ??
      getField(record, "recovery"),
    stackTrace: getField(record, "stackTrace", "stack_trace"),
    details: mapErrorDetails(getField(record, "details")),
  };
}

function pickRunEventPayload(record: Record<string, unknown>): unknown {
  const direct = getField(record, "data");
  if (direct !== undefined) {
    return direct;
  }
  const keys = [
    "log",
    "message",
    "tool_call",
    "toolCall",
    "tool_result",
    "toolResult",
    "status",
    "metric",
    "artifact",
    "error",
    "progress",
    "cost",
    "rate_limit",
    "rateLimit",
  ];
  for (const key of keys) {
    if (key in record) {
      return record[key];
    }
  }
  return {};
}

function mapRunEvent(value: unknown): RunEvent {
  const record = asRecord(value);
  if (!record) return value as RunEvent;
  const payload = pickRunEventPayload(record);
  return {
    id: String(getField(record, "id") ?? ""),
    runId: String(getField(record, "runId", "run_id") ?? ""),
    sequence: toNumber(getField(record, "sequence"), 0),
    eventType: (normalizeEnumValue(getField(record, "eventType", "event_type"), "RUN_EVENT_TYPE_") ??
      getField(record, "eventType", "event_type")) as RunEvent["eventType"],
    timestamp: String(getField(record, "timestamp") ?? ""),
    data: mapRunEventData(payload),
  };
}

function mapDiffStats(value: unknown): DiffStats | undefined {
  const record = asRecord(value);
  if (!record) return value as DiffStats | undefined;
  return {
    filesChanged: toNumber(getField(record, "filesChanged", "files_changed"), 0),
    additions: toNumber(getField(record, "additions"), 0),
    deletions: toNumber(getField(record, "deletions"), 0),
    totalBytes: toNumber(getField(record, "totalBytes", "total_bytes"), 0),
  };
}

function mapDiffFiles(value: unknown): DiffFile[] {
  if (!Array.isArray(value)) return [];
  return value.map((file) => {
    const record = asRecord(file);
    if (!record) return file as DiffFile;
    return {
      path: String(getField(record, "path", "file_path") ?? ""),
      status: (getField(record, "status", "change_type") as DiffFile["status"]) ?? "modified",
      additions: toNumber(getField(record, "additions"), 0),
      deletions: toNumber(getField(record, "deletions"), 0),
    };
  });
}

function mapDiffResult(value: unknown): DiffResult {
  const record = asRecord(value);
  if (!record) return value as DiffResult;
  const stats = mapDiffStats(getField(record, "stats"));
  return {
    unified: String(getField(record, "unified", "content") ?? ""),
    files: mapDiffFiles(getField(record, "files")),
    stats: stats ?? { filesChanged: 0, additions: 0, deletions: 0, totalBytes: 0 },
  };
}

function mapRunnerStatus(value: unknown): RunnerStatus {
  const record = asRecord(value);
  if (!record) return value as RunnerStatus;
  const capabilitiesRecord = asRecord(getField(record, "capabilities"));
  const supportedModels = getField(capabilitiesRecord, "SupportedModels", "supported_models");
  const supportedModelsList = Array.isArray(supportedModels) ? supportedModels : [];
  const capabilities = capabilitiesRecord
    ? {
        SupportsMessages: Boolean(
          getField(capabilitiesRecord, "SupportsMessages", "supports_messages")
        ),
        SupportsToolEvents: Boolean(
          getField(capabilitiesRecord, "SupportsToolEvents", "supports_tool_events")
        ),
        SupportsCostTracking: Boolean(
          getField(capabilitiesRecord, "SupportsCostTracking", "supports_cost_tracking")
        ),
        SupportsStreaming: Boolean(
          getField(capabilitiesRecord, "SupportsStreaming", "supports_streaming")
        ),
        SupportsCancellation: Boolean(
          getField(capabilitiesRecord, "SupportsCancellation", "supports_cancellation")
        ),
        SupportedModels: supportedModelsList,
        MaxTurns: toNumber(getField(capabilitiesRecord, "MaxTurns", "max_turns"), 0),
      }
    : undefined;

  return {
    type: (normalizeRunnerType(getField(record, "type", "runner_type")) ??
      getField(record, "type", "runner_type")) as RunnerType,
    available: Boolean(getField(record, "available")),
    message: getField(record, "message"),
    capabilities,
  };
}

function mapAgentProfile(value: unknown): AgentProfile {
  const record = asRecord(value);
  if (!record) return value as AgentProfile;
  const timeoutValue = parseDurationToNanoseconds(getField(record, "timeout"));
  return {
    id: String(getField(record, "id") ?? ""),
    name: String(getField(record, "name") ?? ""),
    description: getField(record, "description"),
    runnerType: (normalizeRunnerType(getField(record, "runnerType", "runner_type")) ??
      getField(record, "runnerType", "runner_type") ??
      "codex") as RunnerType,
    model: getField(record, "model"),
    maxTurns: toNumber(getField(record, "maxTurns", "max_turns"), 0),
    timeout: timeoutValue ?? (getField(record, "timeout") as number | undefined),
    allowedTools: getField(record, "allowedTools", "allowed_tools"),
    deniedTools: getField(record, "deniedTools", "denied_tools"),
    skipPermissionPrompt: getField(record, "skipPermissionPrompt", "skip_permission_prompt"),
    requiresSandbox: Boolean(getField(record, "requiresSandbox", "requires_sandbox")),
    requiresApproval: Boolean(getField(record, "requiresApproval", "requires_approval")),
    allowedPaths: getField(record, "allowedPaths", "allowed_paths"),
    deniedPaths: getField(record, "deniedPaths", "denied_paths"),
    createdBy: getField(record, "createdBy", "created_by"),
    createdAt: String(getField(record, "createdAt", "created_at") ?? ""),
    updatedAt: String(getField(record, "updatedAt", "updated_at") ?? ""),
  };
}

function mapTask(value: unknown): Task {
  const record = asRecord(value);
  if (!record) return value as Task;
  return {
    id: String(getField(record, "id") ?? ""),
    title: String(getField(record, "title") ?? ""),
    description: getField(record, "description"),
    scopePath: String(getField(record, "scopePath", "scope_path") ?? ""),
    projectRoot: getField(record, "projectRoot", "project_root"),
    phasePromptIds: getField(record, "phasePromptIds", "phase_prompt_ids"),
    contextAttachments: getField(record, "contextAttachments", "context_attachments"),
    status: (normalizeEnumValue(getField(record, "status"), "TASK_STATUS_") ??
      getField(record, "status")) as Task["status"],
    createdBy: getField(record, "createdBy", "created_by"),
    createdAt: String(getField(record, "createdAt", "created_at") ?? ""),
    updatedAt: String(getField(record, "updatedAt", "updated_at") ?? ""),
  };
}

function mapRun(value: unknown): Run {
  const record = asRecord(value);
  if (!record) return value as Run;
  return {
    id: String(getField(record, "id") ?? ""),
    taskId: String(getField(record, "taskId", "task_id") ?? ""),
    agentProfileId: String(getField(record, "agentProfileId", "agent_profile_id") ?? ""),
    sandboxId: getField(record, "sandboxId", "sandbox_id"),
    runMode: (normalizeEnumValue(getField(record, "runMode", "run_mode"), "RUN_MODE_") ??
      getField(record, "runMode", "run_mode")) as Run["runMode"],
    status: (normalizeEnumValue(getField(record, "status"), "RUN_STATUS_") ??
      getField(record, "status")) as Run["status"],
    startedAt: getField(record, "startedAt", "started_at"),
    endedAt: getField(record, "endedAt", "ended_at"),
    phase: (normalizeEnumValue(getField(record, "phase"), "RUN_PHASE_") ??
      getField(record, "phase")) as Run["phase"],
    lastCheckpointId: getField(record, "lastCheckpointId", "last_checkpoint_id"),
    lastHeartbeat: getField(record, "lastHeartbeat", "last_heartbeat"),
    progressPercent: toNumber(getField(record, "progressPercent", "progress_percent"), 0),
    idempotencyKey: getField(record, "idempotencyKey", "idempotency_key"),
    summary: mapRunSummary(getField(record, "summary")),
    errorMsg: getField(record, "errorMsg", "error_msg"),
    exitCode: getField(record, "exitCode", "exit_code"),
    approvalState: (normalizeEnumValue(
      getField(record, "approvalState", "approval_state"),
      "APPROVAL_STATE_"
    ) ??
      getField(record, "approvalState", "approval_state")) as Run["approvalState"],
    approvedBy: getField(record, "approvedBy", "approved_by"),
    approvedAt: getField(record, "approvedAt", "approved_at"),
    diffPath: getField(record, "diffPath", "diff_path"),
    logPath: getField(record, "logPath", "log_path"),
    changedFiles: toNumber(getField(record, "changedFiles", "changed_files"), 0),
    totalSizeBytes: toNumber(getField(record, "totalSizeBytes", "total_size_bytes"), 0),
    createdAt: String(getField(record, "createdAt", "created_at") ?? ""),
    updatedAt: String(getField(record, "updatedAt", "updated_at") ?? ""),
  };
}

function mapProbeResult(value: unknown, runnerTypeFallback?: RunnerType): ProbeResult {
  const record = asRecord(value);
  if (!record) {
    return value as ProbeResult;
  }
  const details = getField<Record<string, string>>(record, "details") ?? {};
  const runnerType = normalizeRunnerType(details.runner_type) ?? runnerTypeFallback;
  const response = details.response;
  const message =
    getField(record, "message") ??
    getField(record, "error") ??
    (getField(record, "success") ? "Probe succeeded" : "Probe failed");
  return {
    runnerType: (runnerType ?? runnerTypeFallback ?? "codex") as RunnerType,
    success: Boolean(getField(record, "success")),
    message: String(message ?? ""),
    response,
    durationMs: toNumber(getField(record, "durationMs", "latency_ms"), 0),
  };
}

// Health hook
export function useHealth() {
  const state = useApiState<HealthStatus>();
  const abortRef = useRef<AbortController | null>(null);

  const fetchHealth = useCallback(async () => {
    if (abortRef.current) {
      abortRef.current.abort();
    }
    const controller = new AbortController();
    abortRef.current = controller;

    state.setLoading(true);
    state.setError(null);

    try {
      const data = await apiRequest<HealthStatus>("/health", {
        signal: controller.signal,
      });
      state.setData(data);
    } catch (err) {
      if ((err as Error).name !== "AbortError") {
        state.setError((err as Error).message);
      }
    } finally {
      state.setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchHealth();
    const interval = setInterval(fetchHealth, 30000);
    return () => {
      clearInterval(interval);
      abortRef.current?.abort();
    };
  }, [fetchHealth]);

  return { ...state, refetch: fetchHealth };
}

// Profiles hook
export function useProfiles() {
  const state = useApiState<AgentProfile[]>([]);

  const fetchProfiles = useCallback(async () => {
    state.setLoading(true);
    state.setError(null);
    try {
      const data = await apiRequest<AgentProfile[] | ListProfilesResponse>("/profiles");
      const profiles = normalizeList<AgentProfile>(data, "profiles").map(mapAgentProfile);
      state.setData(profiles);
    } catch (err) {
      state.setError((err as Error).message);
    } finally {
      state.setLoading(false);
    }
  }, []);

  const createProfile = useCallback(
    async (profile: CreateProfileRequest): Promise<AgentProfile> => {
      const created = await apiRequest<AgentProfile>("/profiles", {
        method: "POST",
        body: JSON.stringify(profile),
      });
      const mapped = mapAgentProfile(created);
      await fetchProfiles();
      return mapped;
    },
    [fetchProfiles]
  );

  const updateProfile = useCallback(
    async (id: string, profile: Partial<CreateProfileRequest>): Promise<AgentProfile> => {
      const updated = await apiRequest<AgentProfile>("/profiles/" + id, {
        method: "PUT",
        body: JSON.stringify(profile),
      });
      const mapped = mapAgentProfile(updated);
      await fetchProfiles();
      return mapped;
    },
    [fetchProfiles]
  );

  const deleteProfile = useCallback(
    async (id: string): Promise<void> => {
      await apiRequest<void>("/profiles/" + id, { method: "DELETE" });
      await fetchProfiles();
    },
    [fetchProfiles]
  );

  useEffect(() => {
    fetchProfiles();
  }, [fetchProfiles]);

  return {
    ...state,
    refetch: fetchProfiles,
    createProfile,
    updateProfile,
    deleteProfile,
  };
}

// Tasks hook
export function useTasks() {
  const state = useApiState<Task[]>([]);

  const fetchTasks = useCallback(async () => {
    state.setLoading(true);
    state.setError(null);
    try {
      const data = await apiRequest<Task[] | ListTasksResponse>("/tasks");
      const tasks = normalizeList<Task>(data, "tasks").map(mapTask);
      state.setData(tasks);
    } catch (err) {
      state.setError((err as Error).message);
    } finally {
      state.setLoading(false);
    }
  }, []);

  const createTask = useCallback(
    async (task: CreateTaskRequest): Promise<Task> => {
      const created = await apiRequest<Task>("/tasks", {
        method: "POST",
        body: JSON.stringify(task),
      });
      const mapped = mapTask(created);
      await fetchTasks();
      return mapped;
    },
    [fetchTasks]
  );

  const getTask = useCallback(async (id: string): Promise<Task> => {
    const task = await apiRequest<Task>("/tasks/" + id);
    return mapTask(task);
  }, []);

  const cancelTask = useCallback(
    async (id: string): Promise<void> => {
      await apiRequest<void>("/tasks/" + id + "/cancel", { method: "POST" });
      await fetchTasks();
    },
    [fetchTasks]
  );

  const deleteTask = useCallback(
    async (id: string): Promise<void> => {
      await apiRequest<void>("/tasks/" + id, { method: "DELETE" });
      await fetchTasks();
    },
    [fetchTasks]
  );

  useEffect(() => {
    fetchTasks();
  }, [fetchTasks]);

  return {
    ...state,
    refetch: fetchTasks,
    createTask,
    getTask,
    cancelTask,
    deleteTask,
  };
}

// Runs hook
export function useRuns() {
  const state = useApiState<Run[]>([]);

  const fetchRuns = useCallback(async () => {
    state.setLoading(true);
    state.setError(null);
    try {
      const data = await apiRequest<Run[] | ListRunsResponse>("/runs");
      const runs = normalizeList<Run>(data, "runs").map(mapRun);
      state.setData(runs);
    } catch (err) {
      state.setError((err as Error).message);
    } finally {
      state.setLoading(false);
    }
  }, []);

  const createRun = useCallback(
    async (run: CreateRunRequest): Promise<Run> => {
      const created = await apiRequest<Run>("/runs", {
        method: "POST",
        body: JSON.stringify(run),
      });
      const mapped = mapRun(created);
      await fetchRuns();
      return mapped;
    },
    [fetchRuns]
  );

  const retryRun = useCallback(
    async (run: Run): Promise<Run> => {
      const request: CreateRunRequest = {
        taskId: run.taskId,
        agentProfileId: run.agentProfileId,
      };
      return createRun(request);
    },
    [createRun]
  );

  const getRun = useCallback(async (id: string): Promise<Run> => {
    const run = await apiRequest<Run>("/runs/" + id);
    return mapRun(run);
  }, []);

  const stopRun = useCallback(
    async (id: string): Promise<void> => {
      await apiRequest<void>("/runs/" + id + "/stop", { method: "POST" });
      await fetchRuns();
    },
    [fetchRuns]
  );

  const getRunEvents = useCallback(
    async (id: string): Promise<RunEvent[]> => {
      const data = await apiRequest<RunEvent[] | GetRunEventsResponse>("/runs/" + id + "/events");
      return normalizeList<RunEvent>(data, "events").map(mapRunEvent);
    },
    []
  );

  const getRunDiff = useCallback(async (id: string): Promise<DiffResult> => {
    const data = await apiRequest<DiffResult | GetRunDiffResponse>("/runs/" + id + "/diff");
    return mapDiffResult(unwrapField<DiffResult>(data, "diff"));
  }, []);

  const approveRun = useCallback(
    async (id: string, req: ApproveRequest): Promise<ApprovalResult> => {
      const data = await apiRequest<ApprovalResult>("/runs/" + id + "/approve", {
        method: "POST",
        body: JSON.stringify(req),
      });
      const resultRecord = unwrapField<ApprovalResult>(data, "result");
      const resultMap = asRecord(resultRecord);
      const result = resultMap
        ? {
            success: Boolean(getField(resultMap, "success")),
            commitHash: getField<string>(resultMap, "commitHash", "commit_hash"),
            message: getField<string>(resultMap, "message"),
          }
        : resultRecord;
      await fetchRuns();
      return result;
    },
    [fetchRuns]
  );

  const rejectRun = useCallback(
    async (id: string, req: RejectRequest): Promise<void> => {
      await apiRequest<void>("/runs/" + id + "/reject", {
        method: "POST",
        body: JSON.stringify(req),
      });
      await fetchRuns();
    },
    [fetchRuns]
  );

  useEffect(() => {
    fetchRuns();
  }, [fetchRuns]);

  return {
    ...state,
    refetch: fetchRuns,
    createRun,
    retryRun,
    getRun,
    stopRun,
    getRunEvents,
    getRunDiff,
    approveRun,
    rejectRun,
  };
}

// Runners hook
export function useRunners() {
  const state = useApiState<Record<string, RunnerStatus>>({});

  const fetchRunners = useCallback(async () => {
    state.setLoading(true);
    state.setError(null);
    try {
      const data = await apiRequest<RunnerStatus[] | GetRunnerStatusResponse>("/runners");
      const runners = normalizeList<RunnerStatus>(data, "runners").map(mapRunnerStatus);
      const record: Record<string, RunnerStatus> = {};
      for (const runner of runners) {
        record[runner.type] = runner;
      }
      state.setData(record);
    } catch (err) {
      state.setError((err as Error).message);
    } finally {
      state.setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchRunners();
  }, [fetchRunners]);

  return { ...state, refetch: fetchRunners };
}

// Probe runner function (standalone for use in components)
export async function probeRunner(runnerType: RunnerType): Promise<ProbeResult> {
  const data = await apiRequest<ProbeResult | ProbeRunnerResponse>(`/runners/${runnerType}/probe`, {
    method: "POST",
  });
  return mapProbeResult(unwrapField<ProbeResult>(data, "result"), runnerType);
}
