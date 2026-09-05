import { useCallback, useEffect, useRef, useState } from "react";
import { create, fromJson, toJson, type DescMessage, type MessageShape, type JsonValue } from "@bufbuild/protobuf";
import { durationFromMs, ValueSchema } from "@bufbuild/protobuf/wkt";
import { getApiBaseUrl, jsonObjectToPlain, runnerTypeToSlug } from "../lib/utils";
import type {
  AgentProfile,
  ApproveFormData,
  ApproveResult,
  HealthResponse,
  InvestigationContextFlags,
  InvestigationDepth,
  InvestigationFindings,
  InvestigationSettings,
  InvestigationTagRule,
  ProfileFormData,
  ProbeResult,
  RejectFormData,
  Run,
  RunDiff,
  RunEvent,
  RunFormData,
  RunnerStatus,
  RunnerType,
  Task,
  TaskFormData,
} from "../types";
import { StructuredResultStatus } from "../types";
import {
  AgentProfileSchema,
  RunConfigOverridesSchema,
} from "@vrooli/proto-types/agent-manager/v1/domain/profile_pb";
import { TaskSchema } from "@vrooli/proto-types/agent-manager/v1/domain/task_pb";
import {
  ApproveRunRequestSchema,
  ApproveRunResponseSchema,
  CreateProfileRequestSchema,
  CreateProfileResponseSchema,
  CreateRunRequestSchema,
  CreateRunResponseSchema,
  CreateTaskRequestSchema,
  CreateTaskResponseSchema,
  EnsureProfileResponseSchema,
  UpdateTaskRequestSchema,
  UpdateTaskResponseSchema,
  GetRunDiffResponseSchema,
  GetRunEventsResponseSchema,
  GetPermissionPolicyCatalogResponseSchema,
  GetPermissionPolicyStatusResponseSchema,
  GetRunResponseSchema,
  GetRolePolicyCatalogResponseSchema,
  GetRunnerStatusResponseSchema,
  GetTaskResponseSchema,
  ListProfilesResponseSchema,
  ListRunsResponseSchema,
  ListTasksResponseSchema,
  PartialApproveRunRequestSchema,
  PartialApproveRunResponseSchema,
  PurgeDataRequestSchema,
  PurgeDataResponseSchema,
  PurgeTarget,
  ProbeRunnerResponseSchema,
  RejectRunRequestSchema,
  DoctorPermissionPolicyResponseSchema,
  PlanPermissionPolicyResponseSchema,
  ReconcilePermissionPolicyRequestSchema,
  ReconcilePermissionPolicyResponseSchema,
  ReloadPermissionPolicyCatalogResponseSchema,
  ValidatePermissionPolicyCatalogResponseSchema,
  UpdateProfileRequestSchema,
  UpdateProfileResponseSchema,
  GetWorkflowExecutionTraceResponseSchema,
  ListWorkflowExecutionsResponseSchema,
  SignalWorkflowExecutionRequestSchema,
  WorkflowExecutionOperationRequestSchema,
  WorkflowExecutionOperationResponseSchema,
} from "@vrooli/proto-types/agent-manager/v1/api/service_pb";
import type { WorkflowExecution, WorkflowJournalEntry, WorkflowNodeAttempt } from "@vrooli/proto-types/agent-manager/v1/domain/workflow_pb";
import type { CohortWatch, InspectCohortWatchResponse } from "@vrooli/proto-types/agent-manager/v1/domain/watch_pb";
import {
  InspectCohortWatchResponseSchema,
  ListCohortWatchesResponseSchema,
} from "@vrooli/proto-types/agent-manager/v1/domain/watch_pb";
import {
  ErrorResponseSchema,
  HealthResponseSchema,
} from "@vrooli/proto-types/common/v1/types_pb";
import {
  ExtraFlagListSchema,
  FeatureFlagsSchema,
  NetworkAccess,
  SandboxConfigSchema,
  SandboxMode,
} from "@vrooli/proto-types/agent-manager/v1/domain/types_pb";

// sandboxModeFromForm parses the UI form-string to the proto enum.
// Empty/unknown maps to UNSPECIFIED so agent-manager applies its
// DefaultSandboxConfig.
function sandboxModeFromForm(s?: "off" | "tracking" | "protected"): SandboxMode {
  switch (s) {
    case "off":
      return SandboxMode.OFF;
    case "tracking":
      return SandboxMode.TRACKING;
    case "protected":
      return SandboxMode.PROTECTED;
    default:
      return SandboxMode.UNSPECIFIED;
  }
}

function networkAccessToProto(na: "none" | "localhost" | "full"): NetworkAccess {
  switch (na) {
    case "none":
      return NetworkAccess.NONE;
    case "localhost":
      return NetworkAccess.LOCALHOST;
    case "full":
      return NetworkAccess.FULL;
    default:
      return NetworkAccess.LOCALHOST;
  }
}

interface ApiState<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
}

export interface RunStatusCounts {
  pending: number;
  running: number;
  complete: number;
  failed: number;
  cancelled: number;
  needsReview: number;
  total: number;
}

export interface RunReportView {
  run_id: string;
  status: string;
  exit_code?: number;
  error?: string;
  duration_ms?: string;
  heartbeat_gap_ms?: string;
  turns: number;
  tokens: number;
  cost_usd: number;
  result: { selection_status: string; selection_rule?: string; candidate_count: number; structured_status?: string; structured_method?: string; diagnostic_codes?: string[] };
  event_counts: Record<string, number>;
  tools: Array<{ name: string; calls: number; successes: number; failures: number; unresolved?: number }>;
  project_owned_tool_calls: number;
  external_tool_calls: number;
  requested_model?: string;
  actual_model?: string;
  fallback_count: number;
  repeated_tool_calls: number;
  files_read_more_than_once: number;
  longest_event_gap_ms: string;
  diff: { files: number; bytes: number; available: { state: string; reason?: string } };
  events_availability: { state: string; reason?: string };
  receipts_availability: { state: string; reason?: string };
  receipt_count: number;
}

export interface RecurringFindingView {
  id: string;
  runId: string;
  investigationRunId: string;
  category: string;
  severity: string;
  recommendation: string;
  evidence?: string;
  targetPath?: string;
  fingerprint: string;
  decision?: string;
  createdAt: string;
  occurrences: number;
}

export function useRecurringFindings() {
  const { data, loading, error, setData, setLoading, setError } = useApiState<RecurringFindingView[]>(null);
  const refetch = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await apiRequest<{ findings: RecurringFindingView[] }>("/findings");
      setData(response.findings.sort((a, b) => b.occurrences - a.occurrences || b.createdAt.localeCompare(a.createdAt)));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load investigation findings");
    } finally {
      setLoading(false);
    }
  }, [setData, setError, setLoading]);
  useEffect(() => { void refetch(); }, [refetch]);
  return { data, loading, error, refetch };
}

export function useRunReport(runId: string) {
  const { data, loading, error, setData, setLoading, setError } = useApiState<RunReportView>(null);
  const refetch = useCallback(async () => {
    if (!runId) return;
    setLoading(true);
    setError(null);
    try {
      setData(await apiRequest<RunReportView>(`/runs/${encodeURIComponent(runId)}/report`));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load run report");
    } finally {
      setLoading(false);
    }
  }, [runId, setData, setError, setLoading]);
  useEffect(() => { void refetch(); }, [refetch]);
  return { data, loading, error, refetch };
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

const protoReadOptions = { ignoreUnknownFields: true, protoFieldName: true };
const protoWriteOptions = { useProtoFieldName: true };

type PurgeCounts = {
  profiles?: number;
  tasks?: number;
  runs?: number;
};

const jsonValueKeys = [
  { snake: "bool_value", camel: "boolValue" },
  { snake: "int_value", camel: "intValue" },
  { snake: "double_value", camel: "doubleValue" },
  { snake: "string_value", camel: "stringValue" },
  { snake: "object_value", camel: "objectValue" },
  { snake: "list_value", camel: "listValue" },
  { snake: "null_value", camel: "nullValue" },
  { snake: "bytes_value", camel: "bytesValue" },
];

function parseProto<Desc extends DescMessage>(schema: Desc, raw: unknown): MessageShape<Desc> {
  return fromJson(schema, raw as JsonValue, protoReadOptions);
}

function toProtoJson<Desc extends DescMessage>(schema: Desc, message: MessageShape<Desc>): Record<string, unknown> {
  return toJson(schema, message, protoWriteOptions) as Record<string, unknown>;
}

function normalizeJsonValueInput(value: unknown): Record<string, unknown> {
  if (value === null) {
    return { null_value: "NULL_VALUE" };
  }
  if (typeof value === "boolean") {
    return { bool_value: value };
  }
  if (typeof value === "number") {
    if (Number.isInteger(value)) {
      return { int_value: value };
    }
    return { double_value: value };
  }
  if (typeof value === "string") {
    return { string_value: value };
  }
  if (Array.isArray(value)) {
    return { list_value: { values: value.map(normalizeJsonValueInput) } };
  }
  if (typeof value === "object" && value !== null) {
    const obj = value as Record<string, unknown>;
    for (const key of jsonValueKeys) {
      if (key.snake in obj || key.camel in obj) {
        const raw = (key.snake in obj ? obj[key.snake] : obj[key.camel]) as unknown;
        if (key.snake === "object_value") {
          const rawObj = raw as Record<string, unknown> | undefined;
          const rawFields = rawObj && typeof rawObj === "object" ? (rawObj.fields as Record<string, unknown> | undefined) : undefined;
          const fieldsSource = rawFields ?? (rawObj && !Array.isArray(rawObj) ? rawObj : {});
          const fields: Record<string, unknown> = {};
          for (const [fieldKey, fieldValue] of Object.entries(fieldsSource ?? {})) {
            fields[fieldKey] = normalizeJsonValueInput(fieldValue);
          }
          return { object_value: { fields } };
        }
        if (key.snake === "list_value") {
          const rawList = Array.isArray(raw) ? raw : (raw as Record<string, unknown>)?.values;
          const values = Array.isArray(rawList) ? rawList.map(normalizeJsonValueInput) : [];
          return { list_value: { values } };
        }
        if (key.snake === "null_value") {
          return { null_value: "NULL_VALUE" };
        }
        return { [key.snake]: raw };
      }
    }

    const fields: Record<string, unknown> = {};
    for (const [fieldKey, fieldValue] of Object.entries(obj)) {
      fields[fieldKey] = normalizeJsonValueInput(fieldValue);
    }
    return { object_value: { fields } };
  }

  return { string_value: String(value) };
}

function normalizeJsonValueMap(value: unknown): Record<string, unknown> | undefined {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return undefined;
  }
  const normalized: Record<string, unknown> = {};
  for (const [key, entry] of Object.entries(value as Record<string, unknown>)) {
    normalized[key] = normalizeJsonValueInput(entry);
  }
  return normalized;
}

function normalizeHealthResponseJson(raw: unknown): unknown {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    return raw;
  }
  const obj = raw as Record<string, unknown>;
  const normalized: Record<string, unknown> = { ...obj };
  const dependencies = normalizeJsonValueMap(obj.dependencies);
  const metrics = normalizeJsonValueMap(obj.metrics);
  if (dependencies) {
    normalized.dependencies = dependencies;
  }
  if (metrics) {
    normalized.metrics = metrics;
  }
  return normalized;
}

function extractErrorMessage(raw: unknown, fallback: string): string {
  try {
    const parsed = parseProto(ErrorResponseSchema, raw);
    const details = jsonObjectToPlain(parsed.details);
    const userMessage = details?.user_message;
    if (typeof userMessage === "string" && userMessage.trim() !== "") {
      return userMessage;
    }
    if (parsed.message) {
      return parsed.message;
    }
  } catch {
    // ignore
  }
  return fallback;
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
    const errorData: unknown = await response.json().catch(() => ({}));
    throw new Error(extractErrorMessage(errorData, "Request failed: " + response.status));
  }

  if (response.status === 204) {
    return {} as T;
  }

  const json: unknown = await response.json();
  return json as T;
}

function durationFromMinutes(minutes?: number) {
  if (typeof minutes !== "number" || Number.isNaN(minutes) || minutes <= 0) {
    return undefined;
  }
  return durationFromMs(minutes * 60_000);
}

export interface WorkflowTraceView {
  execution?: WorkflowExecution;
  attempts: WorkflowNodeAttempt[];
  journal: WorkflowJournalEntry[];
}

export function useWorkflowExecutions() {
  const { data, loading, error, setData, setLoading, setError } = useApiState<WorkflowExecution[]>([]);

  const refetch = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const raw = await apiRequest<unknown>("/workflow-executions?limit=100");
      const response = parseProto(ListWorkflowExecutionsResponseSchema, raw);
      setData(response.executions);
    } catch (error) {
      setError(error instanceof Error ? error.message : "Failed to load workflow executions");
    } finally {
      setLoading(false);
    }
  }, [setData, setError, setLoading]);

  useEffect(() => {
    void refetch();
  }, [refetch]);

  useEffect(() => {
    const refresh = () => void refetch();
    window.addEventListener("agent-manager:workflow-lifecycle", refresh);
    return () => window.removeEventListener("agent-manager:workflow-lifecycle", refresh);
  }, [refetch]);

  const getTrace = useCallback(async (executionId: string): Promise<WorkflowTraceView> => {
    const raw = await apiRequest<unknown>(`/workflow-executions/${encodeURIComponent(executionId)}/trace?limit=500`);
    const response = parseProto(GetWorkflowExecutionTraceResponseSchema, raw);
    return { execution: response.execution, attempts: response.attempts, journal: response.journal };
  }, []);

  const control = useCallback(async (execution: WorkflowExecution, operation: "cancel" | "retry" | "resume") => {
    const request = create(WorkflowExecutionOperationRequestSchema, {
      executionId: execution.id,
      idempotencyKey: `ui-${operation}-${execution.id}-${execution.version.toString()}`,
      expectedVersion: execution.version,
      reason: "Operator action from Agent Manager workflow console",
    });
    const raw = await apiRequest<unknown>(`/workflow-executions/${encodeURIComponent(execution.id)}/${operation}`, {
      method: "POST",
      body: JSON.stringify(toProtoJson(WorkflowExecutionOperationRequestSchema, request)),
    });
    const response = parseProto(WorkflowExecutionOperationResponseSchema, raw);
    await refetch();
    return response.execution;
  }, [refetch]);

  const signal = useCallback(async (execution: WorkflowExecution, name: string, payload: unknown) => {
    const request = create(SignalWorkflowExecutionRequestSchema, {
      executionId: execution.id,
      signal: name,
      payload: fromJson(ValueSchema, normalizeJsonValueInput(payload) as JsonValue, protoReadOptions),
      idempotencyKey: `ui-signal-${execution.id}-${execution.version.toString()}-${name}`,
      expectedVersion: execution.version,
    });
    const raw = await apiRequest<unknown>(`/workflow-executions/${encodeURIComponent(execution.id)}/signals`, {
      method: "POST",
      body: JSON.stringify(toProtoJson(SignalWorkflowExecutionRequestSchema, request)),
    });
    const response = parseProto(WorkflowExecutionOperationResponseSchema, raw);
    await refetch();
    return response.execution;
  }, [refetch]);

  return { data, loading, error, refetch, getTrace, control, signal };
}

export interface CohortWatchInspection {
  inspection: InspectCohortWatchResponse;
  actions: CohortWatchActionView[];
}

export interface CohortWatchActionView {
  actionId: string;
  kind: number;
  targetRunId: string;
  state: number;
  status: string;
  rejectionReason: string;
}

function normalizeCohortWatchActions(raw: unknown): CohortWatchActionView[] {
  if (typeof raw !== "object" || raw === null || !("actions" in raw) || !Array.isArray(raw.actions)) return [];
  return raw.actions.flatMap((value): CohortWatchActionView[] => {
    if (typeof value !== "object" || value === null) return [];
    const action = value as Record<string, unknown>;
    const text = (camel: string, snake: string) => typeof action[camel] === "string" ? action[camel] as string : typeof action[snake] === "string" ? action[snake] as string : "";
    const numeric = (key: string) => typeof action[key] === "number" ? action[key] as number : 0;
    return [{
      actionId: text("actionId", "action_id"),
      kind: numeric("kind"),
      targetRunId: text("targetRunId", "target_run_id"),
      state: numeric("state"),
      status: text("status", "status"),
      rejectionReason: text("rejectionReason", "rejection_reason"),
    }];
  });
}

export function useCohortWatches() {
  const { data, loading, error, setData, setLoading, setError } = useApiState<CohortWatch[]>([]);

  const refetch = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const raw = await apiRequest<unknown>("/cohort-watches?page_size=100");
      const response = parseProto(ListCohortWatchesResponseSchema, raw);
      setData(response.watches);
    } catch (error) {
      setError(error instanceof Error ? error.message : "Failed to load cohort watches");
    } finally {
      setLoading(false);
    }
  }, [setData, setError, setLoading]);

  useEffect(() => {
    void refetch();
  }, [refetch]);

  const inspect = useCallback(async (watchId: string): Promise<CohortWatchInspection> => {
    const encodedID = encodeURIComponent(watchId);
    const [inspectionRaw, actionsRaw] = await Promise.all([
      apiRequest<unknown>(`/cohort-watches/${encodedID}/inspect?event_limit=100`),
      apiRequest<unknown>(`/cohort-watches/${encodedID}/actions?limit=100`),
    ]);
    return {
      inspection: parseProto(InspectCohortWatchResponseSchema, inspectionRaw),
      actions: normalizeCohortWatchActions(actionsRaw),
    };
  }, []);

  return { data, loading, error, refetch, inspect };
}

function generateProfileKey(name: string): string {
  const base = name
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
  if (base) {
    return base;
  }
  const rand = Math.random().toString(36).slice(2, 8);
  return `profile-${rand}`;
}

function resolveProfileKey(profile: ProfileFormData): string {
  const provided = profile.profileKey?.trim();
  return provided && provided.length > 0 ? provided : generateProfileKey(profile.name);
}

function buildProfile(profile: ProfileFormData): AgentProfile {
  return create(AgentProfileSchema, {
    name: profile.name,
    profileKey: resolveProfileKey(profile),
    description: profile.description ?? "",
    roleRef: profile.roleRef.trim(),
    maxTurns: profile.maxTurns ?? 0,
    timeout: durationFromMinutes(profile.timeoutMinutes),
    effort: profile.effort ?? "",
    allowedTools: profile.allowedTools ?? [],
    deniedTools: profile.deniedTools ?? [],
    skipPermissionPrompt: profile.skipPermissionPrompt ?? false,
    sandboxConfig: profile.sandboxMode
      ? create(SandboxConfigSchema, { mode: sandboxModeFromForm(profile.sandboxMode) })
      : undefined,
    networkAccess: networkAccessToProto(profile.networkAccess ?? "localhost"),
    allowedPaths: profile.allowedPaths ?? [],
    deniedPaths: profile.deniedPaths ?? [],
    features: profile.features?.enableBrowser
      ? create(FeatureFlagsSchema, { enableBrowser: true })
      : undefined,
    extraFlags: profile.extraFlags
      ? Object.fromEntries(
          Object.entries(profile.extraFlags).map(([rt, flags]) => [
            rt,
            create(ExtraFlagListSchema, { flags }),
          ])
        )
      : undefined,
  });
}

function buildTask(task: TaskFormData): Task {
  return create(TaskSchema, {
    title: task.title,
    description: task.description ?? "",
    scopePath: task.scopePath,
    projectRoot: task.projectRoot ?? "",
    contextAttachments: (task.contextAttachments ?? []).map((att) => ({
      ...att,
      // Map snake_case to camelCase for proto-es
      attachmentId: att.attachment_id ?? "",
    })),
  });
}

function buildRunConfigOverrides(run: RunFormData) {
  const payload: Record<string, unknown> = {};
  if (run.roleRef !== undefined) {
    payload.roleRef = run.roleRef.trim();
  }
  if (run.maxTurns !== undefined) {
    payload.maxTurns = run.maxTurns;
  }
  if (run.timeoutMinutes !== undefined) {
    payload.timeout = durationFromMinutes(run.timeoutMinutes);
  }
  if (run.effort !== undefined) {
    payload.effort = run.effort;
  }
  if (run.allowedTools !== undefined) {
    payload.allowedTools = run.allowedTools;
    if (run.allowedTools.length === 0) {
      payload.clearAllowedTools = true;
    }
  }
  if (run.deniedTools !== undefined) {
    payload.deniedTools = run.deniedTools;
    if (run.deniedTools.length === 0) {
      payload.clearDeniedTools = true;
    }
  }
  if (typeof run.skipPermissionPrompt === "boolean") {
    payload.skipPermissionPrompt = run.skipPermissionPrompt;
  }
  if (run.sandboxMode !== undefined) {
    // SandboxConfig.mode is the single source of truth for sandbox
    // selection — see agent-manager DeriveRunMode. Pass it via
    // sandboxConfig (the request struct's field), letting the
    // orchestrator's resolveSandboxConfig backfill the rest of the
    // contract defaults (auto-apply, manual-review, etc).
    payload.sandboxConfig = create(SandboxConfigSchema, {
      mode: sandboxModeFromForm(run.sandboxMode),
    });
  }
  if (run.networkAccess !== undefined) {
    payload.networkAccess = networkAccessToProto(run.networkAccess);
  }
  if (run.allowedPaths !== undefined) {
    payload.allowedPaths = run.allowedPaths;
    if (run.allowedPaths.length === 0) {
      payload.clearAllowedPaths = true;
    }
  }
  if (run.deniedPaths !== undefined) {
    payload.deniedPaths = run.deniedPaths;
    if (run.deniedPaths.length === 0) {
      payload.clearDeniedPaths = true;
    }
  }
  if (run.features !== undefined) {
    payload.features = create(FeatureFlagsSchema, {
      enableBrowser: run.features.enableBrowser ?? false,
    });
  }
  if (run.extraFlags !== undefined) {
    payload.extraFlags = Object.fromEntries(
      Object.entries(run.extraFlags).map(([rt, flags]) => [
        rt,
        create(ExtraFlagListSchema, { flags }),
      ])
    );
    if (Object.keys(run.extraFlags).length === 0) {
      payload.clearExtraFlags = true;
    }
  }
  return create(RunConfigOverridesSchema, payload);
}

function hasInlineConfig(run: RunFormData): boolean {
  return Boolean(
    run.roleRef !== undefined ||
      run.maxTurns !== undefined ||
      run.timeoutMinutes !== undefined ||
      run.effort !== undefined ||
      run.allowedTools !== undefined ||
      run.deniedTools !== undefined ||
      typeof run.skipPermissionPrompt === "boolean" ||
      run.sandboxMode !== undefined ||
      run.networkAccess !== undefined ||
      run.allowedPaths !== undefined ||
      run.deniedPaths !== undefined ||
      run.features !== undefined ||
      run.extraFlags !== undefined
  );
}

// Health hook
export function useHealth() {
  const { data, loading, error, setData, setLoading, setError } = useApiState<HealthResponse>();
  const abortRef = useRef<AbortController | null>(null);

  const fetchHealth = useCallback(async () => {
    if (abortRef.current) {
      abortRef.current.abort();
    }
    const controller = new AbortController();
    abortRef.current = controller;

    setLoading(true);
    setError(null);

    try {
      const data = await apiRequest<unknown>("/health", {
        signal: controller.signal,
      });
      const normalized = normalizeHealthResponseJson(data);
      const message = parseProto(HealthResponseSchema, normalized);
      setData(message as HealthResponse);
    } catch (err) {
      if ((err as Error).name !== "AbortError") {
        setError((err as Error).message);
      }
    } finally {
      setLoading(false);
    }
  }, [setData, setLoading, setError]);

  useEffect(() => {
    let timeoutId: ReturnType<typeof setTimeout>;
    let cancelled = false;

    const poll = async () => {
      await fetchHealth();
      if (!cancelled) {
        timeoutId = setTimeout(() => {
          void poll();
        }, 30000);
      }
    };

    void poll();

    return () => {
      cancelled = true;
      clearTimeout(timeoutId);
      abortRef.current?.abort();
    };
  }, [fetchHealth]);

  return { data, loading, error, refetch: fetchHealth };
}

// Profiles hook
export function useProfiles(options?: { enabled?: boolean }) {
  const enabled = options?.enabled ?? true;
  const { data, loading, error, setData, setLoading, setError } = useApiState<AgentProfile[]>([]);

  const fetchProfiles = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiRequest<unknown>("/profiles");
      const message = parseProto(ListProfilesResponseSchema, data);
      setData(message.profiles ?? []);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [setData, setLoading, setError]);

  const createProfile = useCallback(
    async (profile: ProfileFormData): Promise<AgentProfile> => {
      const request = create(CreateProfileRequestSchema, { profile: buildProfile(profile) });
      const created = await apiRequest<unknown>("/profiles", {
        method: "POST",
        body: JSON.stringify(toProtoJson(CreateProfileRequestSchema, request)),
      });
      const message = parseProto(CreateProfileResponseSchema, created);
      const mapped = message.profile as AgentProfile;
      await fetchProfiles();
      return mapped;
    },
    [fetchProfiles]
  );

  const updateProfile = useCallback(
    async (id: string, profile: ProfileFormData): Promise<AgentProfile> => {
      const payload = create(UpdateProfileRequestSchema, {
        profileId: id,
        profile: { ...buildProfile(profile), id },
      });
      const updated = await apiRequest<unknown>("/profiles/" + id, {
        method: "PUT",
        body: JSON.stringify(toProtoJson(UpdateProfileRequestSchema, payload)),
      });
      const message = parseProto(UpdateProfileResponseSchema, updated);
      const mapped = message.profile as AgentProfile;
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
    if (!enabled) {
      return;
    }
    void fetchProfiles();
  }, [enabled, fetchProfiles]);

  return {
    data, loading, error,
    refetch: fetchProfiles,
    createProfile,
    updateProfile,
    deleteProfile,
  };
}

// Tasks hook
export function useTasks(options?: { enabled?: boolean }) {
  const enabled = options?.enabled ?? true;
  const { data, loading, error, setData, setLoading, setError } = useApiState<Task[]>([]);

  const fetchTasks = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiRequest<unknown>("/tasks");
      const message = parseProto(ListTasksResponseSchema, data);
      setData(message.tasks ?? []);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [setData, setLoading, setError]);

  const createTask = useCallback(
    async (task: TaskFormData): Promise<Task> => {
      const request = create(CreateTaskRequestSchema, { task: buildTask(task) });
      const created = await apiRequest<unknown>("/tasks", {
        method: "POST",
        body: JSON.stringify(toProtoJson(CreateTaskRequestSchema, request)),
      });
      const message = parseProto(CreateTaskResponseSchema, created);
      const mapped = message.task as Task;
      await fetchTasks();
      return mapped;
    },
    [fetchTasks]
  );

  const updateTask = useCallback(
    async (id: string, task: TaskFormData): Promise<Task> => {
      const payload = create(UpdateTaskRequestSchema, {
        taskId: id,
        task: { ...buildTask(task), id },
      });
      const updated = await apiRequest<unknown>("/tasks/" + id, {
        method: "PUT",
        body: JSON.stringify(toProtoJson(UpdateTaskRequestSchema, payload)),
      });
      const message = parseProto(UpdateTaskResponseSchema, updated);
      const mapped = message.task as Task;
      await fetchTasks();
      return mapped;
    },
    [fetchTasks]
  );

  const getTask = useCallback(async (id: string): Promise<Task> => {
    const task = await apiRequest<unknown>("/tasks/" + id);
    const message = parseProto(GetTaskResponseSchema, task);
    return message.task as Task;
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
    if (!enabled) {
      return;
    }
    void fetchTasks();
  }, [enabled, fetchTasks]);

  return {
    data, loading, error,
    refetch: fetchTasks,
    createTask,
    updateTask,
    getTask,
    cancelTask,
    deleteTask,
  };
}

// Runs hook
export function useRuns(options?: { enabled?: boolean; limit?: number }) {
  const enabled = options?.enabled ?? true;
  const limit = options?.limit;
  const { data, loading, error, setData, setLoading, setError } = useApiState<Run[]>([]);

  const hasFetchedRef = useRef(false);

  const fetchRuns = useCallback(async () => {
    // Only show loading spinner on initial fetch, not on refetches
    if (!hasFetchedRef.current) {
      setLoading(true);
    }
    setError(null);
    try {
      const query = limit !== undefined ? `?limit=${encodeURIComponent(String(limit))}` : "";
      const resp = await apiRequest<unknown>("/runs" + query);
      const message = parseProto(ListRunsResponseSchema, resp);
      setData(message.runs ?? []);
      hasFetchedRef.current = true;
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [limit, setData, setLoading, setError]);

  const createRun = useCallback(
    async (run: RunFormData): Promise<Run> => {
  const inlineConfig = hasInlineConfig(run) ? buildRunConfigOverrides(run) : undefined;
      // Mint a fresh ConversationId per Decision D7 if the caller didn't
      // supply one. Each top-level "run task" click is conceptually a new
      // conversation; multi-turn flows pass an explicit conversationId.
      const conversationId = run.conversationId ?? crypto.randomUUID();
      const request = create(CreateRunRequestSchema, {
        taskId: run.taskId,
        agentProfileId: run.agentProfileId,
        tag: run.tag,
        runMode: run.runMode,
        executionMode: run.executionMode,
        inlineConfig,
        idempotencyKey: run.idempotencyKey,
        prompt: run.prompt,
        existingSandboxId: run.existingSandboxId,
        conversationId,
        parentRunId: run.parentRunId,
      });
      const created = await apiRequest<unknown>("/runs", {
        method: "POST",
        body: JSON.stringify(toProtoJson(CreateRunRequestSchema, request)),
      });
      const message = parseProto(CreateRunResponseSchema, created);
      const mapped = message.run as Run;
      await fetchRuns();
      return mapped;
    },
    [fetchRuns]
  );

  const retryRun = useCallback(
    async (run: Run): Promise<Run> => {
      const request: RunFormData = {
        taskId: run.taskId,
        agentProfileId: run.agentProfileId,
      };
      return createRun(request);
    },
    [createRun]
  );

  const investigateRuns = useCallback(
    async (
      runIds: string[],
      customContext?: string,
      depth?: "quick" | "standard" | "deep",
      projectRoot?: string,
      scopePaths?: string[],
      attachmentIds?: string[],
		overrides?: { roleRef?: string }
    ): Promise<Run> => {
      const created = await apiRequest<unknown>("/runs/investigate", {
        method: "POST",
        body: JSON.stringify({
          runIds,
          customContext,
          depth,
          projectRoot,
          scopePaths,
          attachmentIds,
			roleRef: overrides?.roleRef,
        }),
      });
      const message = parseProto(CreateRunResponseSchema, created);
      const mapped = message.run as Run;
      await fetchRuns();
      return mapped;
    },
    [fetchRuns]
  );

  const applyInvestigation = useCallback(
    async (
      investigationRunId: string,
      selected: string[],
      customContext?: string,
      attachmentIds?: string[],
		overrides?: { roleRef?: string }
    ): Promise<Run> => {
      const created = await apiRequest<unknown>("/runs/investigation-apply", {
        method: "POST",
        body: JSON.stringify({
          investigationRunId,
          decision: "completed",
          selected,
          customContext,
          attachmentIds,
			roleRef: overrides?.roleRef,
        }),
      });
      const message = parseProto(CreateRunResponseSchema, created);
      const mapped = message.run as Run;
      await fetchRuns();
      return mapped;
    },
    [fetchRuns]
  );

  const resumeFromFailedRun = useCallback(
    async (
      runId: string,
      customContext?: string,
      attachmentIds?: string[]
    ): Promise<Run> => {
      const created = await apiRequest<unknown>("/runs/resume-from-failed", {
        method: "POST",
        body: JSON.stringify({ runId, customContext, attachmentIds }),
      });
      const message = parseProto(CreateRunResponseSchema, created);
      const mapped = message.run as Run;
      await fetchRuns();
      return mapped;
    },
    [fetchRuns]
  );

  const getRun = useCallback(async (id: string): Promise<Run> => {
    const run = await apiRequest<unknown>("/runs/" + id);
    const message = parseProto(GetRunResponseSchema, run);
    return message.run as Run;
  }, []);

  const stopRun = useCallback(
    async (id: string): Promise<void> => {
      await apiRequest<void>("/runs/" + id + "/stop", { method: "POST" });
      await fetchRuns();
    },
    [fetchRuns]
  );

  const deleteRun = useCallback(
    async (id: string): Promise<void> => {
      await apiRequest<void>("/runs/" + id, { method: "DELETE" });
      await fetchRuns();
    },
    [fetchRuns]
  );

  const getRunEvents = useCallback(
    async (id: string, options?: { afterSequence?: bigint }): Promise<RunEvent[]> => {
      const params = new URLSearchParams();
      if (options?.afterSequence !== undefined) {
        params.set("after_sequence", options.afterSequence.toString());
      }
      const query = params.size > 0 ? `?${params.toString()}` : "";
      const data = await apiRequest<unknown>("/runs/" + id + "/events" + query);
      const message = parseProto(GetRunEventsResponseSchema, data);
      return message.events ?? [];
    },
    []
  );

  const getRunDiff = useCallback(async (id: string): Promise<RunDiff> => {
    const data = await apiRequest<unknown>("/runs/" + id + "/diff");
    const message = parseProto(GetRunDiffResponseSchema, data);
    return message.diff as RunDiff;
  }, []);

  const approveRun = useCallback(
    async (id: string, req: ApproveFormData): Promise<ApproveResult> => {
      const actor = req.actor?.trim();
      const payload = create(ApproveRunRequestSchema, {
        runId: id,
        actor: actor || undefined,
        commitMsg: req.commitMsg,
        force: req.force ?? false,
      });
      const data = await apiRequest<unknown>("/runs/" + id + "/approve", {
        method: "POST",
        body: JSON.stringify(toProtoJson(ApproveRunRequestSchema, payload)),
      });
      const message = parseProto(ApproveRunResponseSchema, data);
      const result = message.result as ApproveResult;
      await fetchRuns();
      return result;
    },
    [fetchRuns]
  );

  const rejectRun = useCallback(
    async (id: string, req: RejectFormData): Promise<void> => {
      const actor = req.actor?.trim();
      const payload = create(RejectRunRequestSchema, {
        runId: id,
        actor: actor || undefined,
        reason: req.reason,
      });
      await apiRequest<void>("/runs/" + id + "/reject", {
        method: "POST",
        body: JSON.stringify(toProtoJson(RejectRunRequestSchema, payload)),
      });
      await fetchRuns();
    },
    [fetchRuns]
  );

  const partialApproveRun = useCallback(
    async (id: string, fileIds: string[], actor?: string, commitMsg?: string): Promise<ApproveResult> => {
      const payload = create(PartialApproveRunRequestSchema, {
        runId: id,
        fileIds,
        actor: actor?.trim() || undefined,
        commitMsg: commitMsg || undefined,
      });
      const data = await apiRequest<unknown>("/runs/" + id + "/partial-approve", {
        method: "POST",
        body: JSON.stringify(toProtoJson(PartialApproveRunRequestSchema, payload)),
      });
      const message = parseProto(PartialApproveRunResponseSchema, data);
      await fetchRuns();
      return message.result as ApproveResult;
    },
    [fetchRuns]
  );

  const continueRun = useCallback(
    async (id: string, message: string, attachmentIds?: string[]): Promise<Run> => {
      const body: Record<string, unknown> = { message };
      if (attachmentIds && attachmentIds.length > 0) {
        body.attachment_ids = attachmentIds;
      }
      const data = await apiRequest<unknown>("/runs/" + id + "/continue", {
        method: "POST",
        body: JSON.stringify(body),
      });
      const response = data as { success: boolean; run: unknown; error?: string };
      if (!response.success && response.error) {
        throw new Error(response.error);
      }
      await fetchRuns();
      return response.run as Run;
    },
    [fetchRuns]
  );

  const deleteRunMessage = useCallback(
    async (runId: string, eventId: string): Promise<void> => {
      await apiRequest<void>(`/runs/${runId}/messages/${eventId}/delete`, {
        method: "POST",
      });
      await fetchRuns();
    },
    [fetchRuns]
  );

  useEffect(() => {
    if (!enabled) {
      return;
    }
    void fetchRuns();
  }, [enabled, fetchRuns]);

  return {
    data, loading, error,
    refetch: fetchRuns,
    createRun,
    retryRun,
    investigateRuns,
    applyInvestigation,
    resumeFromFailedRun,
    getRun,
    stopRun,
    deleteRun,
    getRunEvents,
    getRunDiff,
    approveRun,
    rejectRun,
    partialApproveRun,
    continueRun,
    deleteRunMessage,
  };
}

export function useRunStatusCounts(options?: { enabled?: boolean }) {
  const enabled = options?.enabled ?? true;
  const { data, loading, error, setData, setLoading, setError } = useApiState<RunStatusCounts>();

  const fetchStatusCounts = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await apiRequest<{ statusCounts?: RunStatusCounts }>("/stats/status-distribution");
      setData(response.statusCounts ?? null);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [setData, setLoading, setError]);

  useEffect(() => {
    if (!enabled) {
      return;
    }
    void fetchStatusCounts();
  }, [enabled, fetchStatusCounts]);

  return { data, loading, error, refetch: fetchStatusCounts };
}

// Runners hook
export function useRunners(options?: { enabled?: boolean }) {
  const enabled = options?.enabled ?? true;
  const { data, loading, error, setData, setLoading, setError } = useApiState<Record<string, RunnerStatus>>({});

  const fetchRunners = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiRequest<unknown>("/runners");
      const message = parseProto(GetRunnerStatusResponseSchema, data);
      const record: Record<string, RunnerStatus> = {};
      for (const runner of message.runners ?? []) {
        record[String(runner.runnerType)] = runner;
      }
      setData(record);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [setData, setLoading, setError]);

  useEffect(() => {
    if (!enabled) {
      return;
    }
    void fetchRunners();
  }, [enabled, fetchRunners]);

  return { data, loading, error, refetch: fetchRunners };
}

// Role-policy catalog hook. This is a read-only projection of Git-managed
// declared state; reload and validation remain explicit operator commands.
export function useRolePolicyCatalog(options?: { enabled?: boolean }) {
	const enabled = options?.enabled ?? true;
	const { data, loading, error, setData, setLoading, setError } = useApiState<MessageShape<typeof GetRolePolicyCatalogResponseSchema> | null>(null);

  const fetchCatalog = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
		const payload = await apiRequest<unknown>("/role-policy/catalog");
		setData(parseProto(GetRolePolicyCatalogResponseSchema, payload));
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [setData, setLoading, setError]);

  useEffect(() => {
    if (enabled) void fetchCatalog();
  }, [enabled, fetchCatalog]);

  return { data, loading, error, refetch: fetchCatalog };
}

type PermissionPolicyData = {
  status: MessageShape<typeof GetPermissionPolicyStatusResponseSchema>;
  catalog: MessageShape<typeof GetPermissionPolicyCatalogResponseSchema>;
};

// Permission-policy actions are intentionally whole-document operations. The
// UI never exposes resource-native patterns or individual rule mutation.
export function usePermissionPolicy(options?: { enabled?: boolean }) {
  const enabled = options?.enabled ?? true;
  const { data, loading, error, setData, setLoading, setError } = useApiState<PermissionPolicyData | null>(null);

  const refetch = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [statusPayload, catalogPayload] = await Promise.all([
        apiRequest<unknown>("/permission-policy/status"),
        apiRequest<unknown>("/permission-policy/catalog"),
      ]);
      setData({
        status: parseProto(GetPermissionPolicyStatusResponseSchema, statusPayload),
        catalog: parseProto(GetPermissionPolicyCatalogResponseSchema, catalogPayload),
      });
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [setData, setLoading, setError]);

  useEffect(() => {
    if (enabled) void refetch();
  }, [enabled, refetch]);

  const validate = useCallback(async () => {
    const payload = await apiRequest<unknown>("/permission-policy/validate", { method: "POST" });
    const response = parseProto(ValidatePermissionPolicyCatalogResponseSchema, payload);
    await refetch();
    return response;
  }, [refetch]);

  const reload = useCallback(async () => {
    const payload = await apiRequest<unknown>("/permission-policy/reload", { method: "POST" });
    const response = parseProto(ReloadPermissionPolicyCatalogResponseSchema, payload);
    await refetch();
    return response;
  }, [refetch]);

  const plan = useCallback(async () => {
    const payload = await apiRequest<unknown>("/permission-policy/plan", { method: "POST" });
    return parseProto(PlanPermissionPolicyResponseSchema, payload);
  }, []);

  const doctor = useCallback(async () => {
    const payload = await apiRequest<unknown>("/permission-policy/doctor", { method: "POST" });
    return parseProto(DoctorPermissionPolicyResponseSchema, payload);
  }, []);

  const reconcile = useCallback(async () => {
    const request = create(ReconcilePermissionPolicyRequestSchema, { explicitlyAuthorized: true });
    const payload = await apiRequest<unknown>("/permission-policy/reconcile", {
      method: "POST",
      body: JSON.stringify(toProtoJson(ReconcilePermissionPolicyRequestSchema, request)),
    });
    const response = parseProto(ReconcilePermissionPolicyResponseSchema, payload);
    await refetch();
    return response;
  }, [refetch]);

  return { data, loading, error, refetch, validate, reload, plan, doctor, reconcile };
}

// Probe runner function (standalone for use in components)
export async function probeRunner(runnerType: RunnerType): Promise<ProbeResult> {
  const data = await apiRequest<unknown>(`/runners/${runnerTypeToSlug(runnerType)}/probe`, {
    method: "POST",
  });
  const message = parseProto(ProbeRunnerResponseSchema, data);
  return message.result as ProbeResult;
}

// Investigation Settings hook
export function useInvestigationSettings() {
  const { data, loading, error, setData, setLoading, setError } = useApiState<InvestigationSettings | null>(null);

  const fetchSettings = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiRequest<InvestigationSettings>("/investigation-settings");
      setData(data);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [setData, setLoading, setError]);

  const updateSettings = useCallback(async (settings: Partial<{
    promptTemplate: string;
    applyPromptTemplate: string;
    defaultDepth: InvestigationDepth;
    defaultContext: InvestigationContextFlags;
    investigationTagAllowlist: InvestigationTagRule[];
  }>): Promise<InvestigationSettings> => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiRequest<InvestigationSettings>("/investigation-settings", {
        method: "PUT",
        body: JSON.stringify(settings),
      });
      setData(data);
      return data;
    } catch (err) {
      setError((err as Error).message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [setData, setLoading, setError]);

  const resetSettings = useCallback(async (): Promise<InvestigationSettings> => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiRequest<InvestigationSettings>("/investigation-settings/reset", {
        method: "POST",
      });
      setData(data);
      return data;
    } catch (err) {
      setError((err as Error).message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [setData, setLoading, setError]);

  useEffect(() => {
    fetchSettings();
  }, [fetchSettings]);

  return {
    data, loading, error,
    refetch: fetchSettings,
    updateSettings,
    resetSettings,
  };
}

// Ensure profile exists (standalone function)
// Creates the profile with defaults if it doesn't exist, returns existing profile otherwise
export async function ensureProfile(profileKey: string): Promise<AgentProfile> {
  const data = await apiRequest<unknown>("/profiles/ensure", {
    method: "POST",
    body: JSON.stringify({ profileKey }),
  });
  const message = parseProto(EnsureProfileResponseSchema, data);
  return message.profile as AgentProfile;
}

// Reads the categorized recommendations out of an investigation run's
// structured result (run.result.structured, present once the run's
// `agent-manager/investigate` node completes with a schema-validated
// output). Returns null when the run hasn't produced a successful
// structured result yet (still running, failed, or extraction was not
// SUCCESS) — callers should treat that as "not ready" rather than an error.
export function getInvestigationFindings(run: Run | null | undefined): InvestigationFindings | null {
  const structured = run?.result?.structured;
  if (!structured || structured.status !== StructuredResultStatus.SUCCESS) {
    return null;
  }

  const raw: unknown = structured.value;
  let parsed: unknown;
  if (typeof raw === "string") {
    if (!raw.trim()) return null;
    try {
      parsed = JSON.parse(raw);
    } catch {
      return null;
    }
  } else if (ArrayBuffer.isView(raw)) {
    const bytes = new Uint8Array(raw.buffer, raw.byteOffset, raw.byteLength);
    if (bytes.length === 0) return null;
    try {
      parsed = JSON.parse(new TextDecoder().decode(bytes));
    } catch {
      return null;
    }
  } else if (raw && typeof raw === "object") {
    parsed = raw;
  } else {
    return null;
  }

  if (!parsed || typeof parsed !== "object") return null;
  const findings = parsed as Partial<InvestigationFindings>;
  if (!Array.isArray(findings.categories)) return null;

  return {
    summary: typeof findings.summary === "string" ? findings.summary : "",
    primaryCategory: findings.primaryCategory ?? "Both",
    confidence: findings.confidence,
    categories: findings.categories,
  };
}

// Maintenance hook
export function useMaintenance() {
  const previewPurge = useCallback(async (pattern: string, targets: PurgeTarget[]): Promise<PurgeCounts> => {
    const payload = create(PurgeDataRequestSchema, {
      pattern,
      targets,
      dryRun: true,
    });
    const data = await apiRequest<unknown>("/maintenance/purge", {
      method: "POST",
      body: JSON.stringify(toProtoJson(PurgeDataRequestSchema, payload)),
    });
    const message = parseProto(PurgeDataResponseSchema, data);
    return message.matched ?? {};
  }, []);

  const executePurge = useCallback(async (pattern: string, targets: PurgeTarget[]): Promise<PurgeCounts> => {
    const payload = create(PurgeDataRequestSchema, {
      pattern,
      targets,
      dryRun: false,
    });
    const data = await apiRequest<unknown>("/maintenance/purge", {
      method: "POST",
      body: JSON.stringify(toProtoJson(PurgeDataRequestSchema, payload)),
    });
    const message = parseProto(PurgeDataResponseSchema, data);
    return message.deleted ?? {};
  }, []);

  return {
    previewPurge,
    executePurge,
  };
}
