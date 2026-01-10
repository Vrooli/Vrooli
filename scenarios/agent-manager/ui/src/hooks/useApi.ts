import { useCallback, useEffect, useRef, useState } from "react";
import { create, fromJson, toJson } from "@bufbuild/protobuf";
import { durationFromMs } from "@bufbuild/protobuf/wkt";
import { getApiBaseUrl, jsonObjectToPlain, runnerTypeToSlug } from "../lib/utils";
import type {
  AgentProfile,
  ApproveFormData,
  ApproveResult,
  ExtractionResult,
  HealthResponse,
  InvestigationContextFlags,
  InvestigationDepth,
  InvestigationSettings,
  InvestigationTagRule,
  ModelRegistry,
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
import { ModelPreset } from "../types";
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
  GetRunResponseSchema,
  GetRunnerStatusResponseSchema,
  GetTaskResponseSchema,
  ListProfilesResponseSchema,
  ListRunsResponseSchema,
  ListTasksResponseSchema,
  PurgeDataRequestSchema,
  PurgeDataResponseSchema,
  PurgeTarget,
  ProbeRunnerResponseSchema,
  RejectRunRequestSchema,
  UpdateProfileRequestSchema,
  UpdateProfileResponseSchema,
} from "@vrooli/proto-types/agent-manager/v1/api/service_pb";
import {
  ErrorResponseSchema,
  HealthResponseSchema,
} from "@vrooli/proto-types/common/v1/types_pb";

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

function parseProto<T>(schema: any, raw: unknown): T {
  return fromJson(schema, raw as any, protoReadOptions) as T;
}

function toProtoJson(schema: any, message: any): Record<string, unknown> {
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
    const parsed = parseProto<any>(ErrorResponseSchema, raw);
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
    const errorData = await response.json().catch(() => ({}));
    throw new Error(extractErrorMessage(errorData, "Request failed: " + response.status));
  }

  if (response.status === 204) {
    return {} as T;
  }

  return response.json();
}

function durationFromMinutes(minutes?: number) {
  if (typeof minutes !== "number" || Number.isNaN(minutes) || minutes <= 0) {
    return undefined;
  }
  return durationFromMs(minutes * 60_000);
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
  const model = profile.model?.trim() ?? "";
  return create(AgentProfileSchema, {
    name: profile.name,
    profileKey: resolveProfileKey(profile),
    description: profile.description ?? "",
    runnerType: profile.runnerType,
    model,
    modelPreset: profile.modelPreset ?? ModelPreset.UNSPECIFIED,
    maxTurns: profile.maxTurns ?? 0,
    timeout: durationFromMinutes(profile.timeoutMinutes),
    fallbackRunnerTypes: profile.fallbackRunnerTypes ?? [],
    allowedTools: profile.allowedTools ?? [],
    deniedTools: profile.deniedTools ?? [],
    skipPermissionPrompt: profile.skipPermissionPrompt ?? false,
    requiresSandbox: profile.requiresSandbox ?? true,
    requiresApproval: profile.requiresApproval ?? true,
    allowedPaths: profile.allowedPaths ?? [],
    deniedPaths: profile.deniedPaths ?? [],
  });
}

function buildTask(task: TaskFormData): Task {
  return create(TaskSchema, {
    title: task.title,
    description: task.description ?? "",
    scopePath: task.scopePath,
    projectRoot: task.projectRoot ?? "",
    contextAttachments: task.contextAttachments ?? [],
  });
}

function buildRunConfigOverrides(run: RunFormData) {
  const payload: Record<string, unknown> = {};
  if (run.runnerType !== undefined) {
    payload.runnerType = run.runnerType;
  }
  if (run.fallbackRunnerTypes !== undefined) {
    payload.fallbackRunnerTypes = run.fallbackRunnerTypes;
    if (run.fallbackRunnerTypes.length === 0) {
      payload.clearFallbackRunnerTypes = true;
    }
  }
  if (run.modelPreset !== undefined) {
    payload.modelPreset = run.modelPreset;
  }
  if (run.model !== undefined && run.model.trim() !== "") {
    payload.model = run.model;
  }
  if (run.maxTurns !== undefined) {
    payload.maxTurns = run.maxTurns;
  }
  if (run.timeoutMinutes !== undefined) {
    payload.timeout = durationFromMinutes(run.timeoutMinutes);
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
  if (typeof run.requiresSandbox === "boolean") {
    payload.requiresSandbox = run.requiresSandbox;
  }
  if (typeof run.requiresApproval === "boolean") {
    payload.requiresApproval = run.requiresApproval;
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
  return create(RunConfigOverridesSchema, payload);
}

function hasInlineConfig(run: RunFormData): boolean {
  return Boolean(
    run.runnerType !== undefined ||
      run.fallbackRunnerTypes !== undefined ||
      run.model !== undefined ||
      run.modelPreset !== undefined ||
      run.maxTurns !== undefined ||
      run.timeoutMinutes !== undefined ||
      run.allowedTools !== undefined ||
      run.deniedTools !== undefined ||
      typeof run.skipPermissionPrompt === "boolean" ||
      typeof run.requiresSandbox === "boolean" ||
      typeof run.requiresApproval === "boolean" ||
      run.allowedPaths !== undefined ||
      run.deniedPaths !== undefined
  );
}

// Health hook
export function useHealth() {
  const state = useApiState<HealthResponse>();
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
      const data = await apiRequest<unknown>("/health", {
        signal: controller.signal,
      });
      const normalized = normalizeHealthResponseJson(data);
      const message = parseProto<HealthResponse>(HealthResponseSchema, normalized);
      state.setData(message);
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
      const data = await apiRequest<unknown>("/profiles");
      const message = parseProto<any>(ListProfilesResponseSchema, data);
      state.setData(message.profiles ?? []);
    } catch (err) {
      state.setError((err as Error).message);
    } finally {
      state.setLoading(false);
    }
  }, []);

  const createProfile = useCallback(
    async (profile: ProfileFormData): Promise<AgentProfile> => {
      const request = create(CreateProfileRequestSchema, { profile: buildProfile(profile) });
      const created = await apiRequest<unknown>("/profiles", {
        method: "POST",
        body: JSON.stringify(toProtoJson(CreateProfileRequestSchema, request)),
      });
      const message = parseProto<any>(CreateProfileResponseSchema, created);
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
      const message = parseProto<any>(UpdateProfileResponseSchema, updated);
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
      const data = await apiRequest<unknown>("/tasks");
      const message = parseProto<any>(ListTasksResponseSchema, data);
      state.setData(message.tasks ?? []);
    } catch (err) {
      state.setError((err as Error).message);
    } finally {
      state.setLoading(false);
    }
  }, []);

  const createTask = useCallback(
    async (task: TaskFormData): Promise<Task> => {
      const request = create(CreateTaskRequestSchema, { task: buildTask(task) });
      const created = await apiRequest<unknown>("/tasks", {
        method: "POST",
        body: JSON.stringify(toProtoJson(CreateTaskRequestSchema, request)),
      });
      const message = parseProto<any>(CreateTaskResponseSchema, created);
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
      const message = parseProto<any>(UpdateTaskResponseSchema, updated);
      const mapped = message.task as Task;
      await fetchTasks();
      return mapped;
    },
    [fetchTasks]
  );

  const getTask = useCallback(async (id: string): Promise<Task> => {
    const task = await apiRequest<unknown>("/tasks/" + id);
    const message = parseProto<any>(GetTaskResponseSchema, task);
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
    fetchTasks();
  }, [fetchTasks]);

  return {
    ...state,
    refetch: fetchTasks,
    createTask,
    updateTask,
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
      const data = await apiRequest<unknown>("/runs");
      const message = parseProto<any>(ListRunsResponseSchema, data);
      state.setData(message.runs ?? []);
    } catch (err) {
      state.setError((err as Error).message);
    } finally {
      state.setLoading(false);
    }
  }, []);

  const createRun = useCallback(
    async (run: RunFormData): Promise<Run> => {
  const inlineConfig = hasInlineConfig(run) ? buildRunConfigOverrides(run) : undefined;
      const request = create(CreateRunRequestSchema, {
        taskId: run.taskId,
        agentProfileId: run.agentProfileId,
        tag: run.tag,
        runMode: run.runMode,
        inlineConfig,
        idempotencyKey: run.idempotencyKey,
        prompt: run.prompt,
        existingSandboxId: run.existingSandboxId,
      });
      const created = await apiRequest<unknown>("/runs", {
        method: "POST",
        body: JSON.stringify(toProtoJson(CreateRunRequestSchema, request)),
      });
      const message = parseProto<any>(CreateRunResponseSchema, created);
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
      scopePaths?: string[]
    ): Promise<Run> => {
      const created = await apiRequest<unknown>("/runs/investigate", {
        method: "POST",
        body: JSON.stringify({ runIds, customContext, depth, projectRoot, scopePaths }),
      });
      const message = parseProto<any>(CreateRunResponseSchema, created);
      const mapped = message.run as Run;
      await fetchRuns();
      return mapped;
    },
    [fetchRuns]
  );

  const applyInvestigation = useCallback(
    async (investigationRunId: string, customContext?: string): Promise<Run> => {
      const created = await apiRequest<unknown>("/runs/investigation-apply", {
        method: "POST",
        body: JSON.stringify({ investigationRunId, customContext }),
      });
      const message = parseProto<any>(CreateRunResponseSchema, created);
      const mapped = message.run as Run;
      await fetchRuns();
      return mapped;
    },
    [fetchRuns]
  );

  const getRun = useCallback(async (id: string): Promise<Run> => {
    const run = await apiRequest<unknown>("/runs/" + id);
    const message = parseProto<any>(GetRunResponseSchema, run);
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
    async (id: string): Promise<RunEvent[]> => {
      const data = await apiRequest<unknown>("/runs/" + id + "/events");
      const message = parseProto<any>(GetRunEventsResponseSchema, data);
      return message.events ?? [];
    },
    []
  );

  const getRunDiff = useCallback(async (id: string): Promise<RunDiff> => {
    const data = await apiRequest<unknown>("/runs/" + id + "/diff");
    const message = parseProto<any>(GetRunDiffResponseSchema, data);
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
      const message = parseProto<any>(ApproveRunResponseSchema, data);
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

  const continueRun = useCallback(
    async (id: string, message: string): Promise<Run> => {
      const data = await apiRequest<unknown>("/runs/" + id + "/continue", {
        method: "POST",
        body: JSON.stringify({ message }),
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

  useEffect(() => {
    fetchRuns();
  }, [fetchRuns]);

  return {
    ...state,
    refetch: fetchRuns,
    createRun,
    retryRun,
    investigateRuns,
    applyInvestigation,
    getRun,
    stopRun,
    deleteRun,
    getRunEvents,
    getRunDiff,
    approveRun,
    rejectRun,
    continueRun,
  };
}

// Runners hook
export function useRunners() {
  const state = useApiState<Record<string, RunnerStatus>>({});

  const fetchRunners = useCallback(async () => {
    state.setLoading(true);
    state.setError(null);
    try {
      const data = await apiRequest<unknown>("/runners");
      const message = parseProto<any>(GetRunnerStatusResponseSchema, data);
      const record: Record<string, RunnerStatus> = {};
      for (const runner of message.runners ?? []) {
        record[String(runner.runnerType)] = runner;
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

// Model registry hook
export function useModelRegistry() {
  const state = useApiState<ModelRegistry | null>(null);

  const fetchRegistry = useCallback(async () => {
    state.setLoading(true);
    state.setError(null);
    try {
      const data = await apiRequest<ModelRegistry>("/runner-models");
      state.setData(data);
    } catch (err) {
      state.setError((err as Error).message);
    } finally {
      state.setLoading(false);
    }
  }, []);

  const updateRegistry = useCallback(async (registry: ModelRegistry): Promise<ModelRegistry> => {
    state.setLoading(true);
    state.setError(null);
    try {
      const data = await apiRequest<ModelRegistry>("/runner-models", {
        method: "PUT",
        body: JSON.stringify(registry),
      });
      state.setData(data);
      return data;
    } catch (err) {
      state.setError((err as Error).message);
      throw err;
    } finally {
      state.setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchRegistry();
  }, [fetchRegistry]);

  return { ...state, refetch: fetchRegistry, updateRegistry };
}

// Probe runner function (standalone for use in components)
export async function probeRunner(runnerType: RunnerType): Promise<ProbeResult> {
  const data = await apiRequest<unknown>(`/runners/${runnerTypeToSlug(runnerType)}/probe`, {
    method: "POST",
  });
  const message = parseProto<any>(ProbeRunnerResponseSchema, data);
  return message.result as ProbeResult;
}

// Investigation Settings hook
export function useInvestigationSettings() {
  const state = useApiState<InvestigationSettings | null>(null);

  const fetchSettings = useCallback(async () => {
    state.setLoading(true);
    state.setError(null);
    try {
      const data = await apiRequest<InvestigationSettings>("/investigation-settings");
      state.setData(data);
    } catch (err) {
      state.setError((err as Error).message);
    } finally {
      state.setLoading(false);
    }
  }, []);

  const updateSettings = useCallback(async (settings: Partial<{
    promptTemplate: string;
    applyPromptTemplate: string;
    defaultDepth: InvestigationDepth;
    defaultContext: InvestigationContextFlags;
    investigationTagAllowlist: InvestigationTagRule[];
  }>): Promise<InvestigationSettings> => {
    state.setLoading(true);
    state.setError(null);
    try {
      const data = await apiRequest<InvestigationSettings>("/investigation-settings", {
        method: "PUT",
        body: JSON.stringify(settings),
      });
      state.setData(data);
      return data;
    } catch (err) {
      state.setError((err as Error).message);
      throw err;
    } finally {
      state.setLoading(false);
    }
  }, []);

  const resetSettings = useCallback(async (): Promise<InvestigationSettings> => {
    state.setLoading(true);
    state.setError(null);
    try {
      const data = await apiRequest<InvestigationSettings>("/investigation-settings/reset", {
        method: "POST",
      });
      state.setData(data);
      return data;
    } catch (err) {
      state.setError((err as Error).message);
      throw err;
    } finally {
      state.setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSettings();
  }, [fetchSettings]);

  return {
    ...state,
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
  const message = parseProto<any>(EnsureProfileResponseSchema, data);
  return message.profile as AgentProfile;
}

// Extract recommendations from an investigation run (standalone function)
// Uses a longer timeout (120s) since LLM extraction can take time
export async function extractRecommendations(runId: string): Promise<ExtractionResult> {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), 120000); // 120 second timeout

  try {
    return await apiRequest<ExtractionResult>(`/runs/${runId}/extract-recommendations`, {
      method: "POST",
      signal: controller.signal,
    });
  } finally {
    clearTimeout(timeoutId);
  }
}

// Regenerate recommendations for an investigation run (standalone function)
// Resets extraction state and queues for background processing
export async function regenerateRecommendations(runId: string): Promise<void> {
  await apiRequest<{ success: boolean }>(`/runs/${runId}/regenerate-recommendations`, {
    method: "POST",
  });
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
    const message = parseProto<any>(PurgeDataResponseSchema, data);
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
    const message = parseProto<any>(PurgeDataResponseSchema, data);
    return message.deleted ?? {};
  }, []);

  return {
    previewPurge,
    executePurge,
  };
}
