/**
 * Proto contract definitions and validation for the Ecosystem Manager.
 *
 * Provides:
 * - Zod validators wrapping protobuf schema validation
 * - Mapping functions from proto types (camelCase) → UI types (snake_case)
 * - Utilities for proto serialization/deserialization
 */

import { z } from "zod";
import {
  create,
  fromJson,
  toJson,
  type JsonValue,
  type Message,
  type MessageInitShape,
} from "@bufbuild/protobuf";
import type { GenMessage } from "@bufbuild/protobuf/codegenv2";
import { createValidator } from "@bufbuild/protovalidate";

// Proto domain schemas
import { TaskSchema } from "@vrooli/proto-types/ecosystem-manager/v1/domain/task_pb";
import { ExecutionRecordSchema, QueueStatusSchema } from "@vrooli/proto-types/ecosystem-manager/v1/domain/queue_pb";
import { SettingsSchema, RecyclerSettingsSchema } from "@vrooli/proto-types/ecosystem-manager/v1/domain/settings_pb";

// Proto domain types
import type {
  Task as ProtoTask,
  ProcessInfo as ProtoProcessInfo,
  ActiveTarget as ProtoActiveTarget,
} from "@vrooli/proto-types/ecosystem-manager/v1/domain/task_pb";
import type {
  ExecutionRecord as ProtoExecutionRecord,
  QueueStatus as ProtoQueueStatus,
} from "@vrooli/proto-types/ecosystem-manager/v1/domain/queue_pb";
import type {
  Settings as ProtoSettings,
} from "@vrooli/proto-types/ecosystem-manager/v1/domain/settings_pb";
// Proto API schemas
import { TaskListResponseSchema, TaskDetailResponseSchema } from "@vrooli/proto-types/ecosystem-manager/v1/api/tasks_pb";
import { SettingsResponseSchema } from "@vrooli/proto-types/ecosystem-manager/v1/api/settings_pb";
import { QueueStatusResponseSchema, ProcessListResponseSchema } from "@vrooli/proto-types/ecosystem-manager/v1/api/queue_pb";

// UI types (existing snake_case interfaces)
import type {
  Task,
  ProcessInfo,
  QueueStatus,
  RunningProcess,
  Settings,
  SettingsConstraints,
  ConstraintRange,
  ExecutionHistory,
  ActiveTarget,
  TaskStatus,
} from "../types/api";

// ---------------------------------------------------------------------------
// Validator & helpers
// ---------------------------------------------------------------------------

const validator = createValidator();

type ProtoSchema<Shape extends Message> = z.ZodType<Shape, z.ZodTypeDef, unknown>;

function isJsonValue(value: unknown): value is JsonValue {
  if (value === null) return true;
  const t = typeof value;
  if (t === "string" || t === "number" || t === "boolean") return true;
  if (Array.isArray(value)) return value.every(isJsonValue);
  if (t === "object") return Object.values(value as Record<string, unknown>).every(isJsonValue);
  return false;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function stringValue(value: unknown): string | undefined {
  return typeof value === "string" ? value : undefined;
}

function numberValue(value: unknown): number | undefined {
  return typeof value === "number" ? value : undefined;
}

function booleanValue(value: unknown): boolean | undefined {
  return typeof value === "boolean" ? value : undefined;
}

function firstString(record: Record<string, unknown>, keys: string[]): string | undefined {
  for (const key of keys) {
    const value = stringValue(record[key]);
    if (value !== undefined) return value;
  }
  return undefined;
}

function firstNumber(record: Record<string, unknown>, keys: string[]): number | undefined {
  for (const key of keys) {
    const value = numberValue(record[key]);
    if (value !== undefined) return value;
  }
  return undefined;
}

function firstBoolean(record: Record<string, unknown>, keys: string[]): boolean | undefined {
  for (const key of keys) {
    const value = booleanValue(record[key]);
    if (value !== undefined) return value;
  }
  return undefined;
}

function createProtoSchema<Shape extends Message>(
  schema: GenMessage<Shape>,
  label: string,
): ProtoSchema<Shape> {
  return z.unknown().transform<Shape>((value, ctx) => {
    if (!isJsonValue(value)) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, message: `Invalid ${label} response` });
      return z.NEVER;
    }
    try {
      const message = fromJson(schema, value, { ignoreUnknownFields: true });
      const validation = validator.validate(schema, message);
      if (validation.kind !== "valid") {
        console.error(`[ecosystem-manager] ${label} validation failed`, validation.error);
        ctx.addIssue({ code: z.ZodIssueCode.custom, message: `Invalid ${label} response` });
        return z.NEVER;
      }
      return message;
    } catch (error) {
      const msg = error instanceof Error ? error.message : String(error);
      console.error(`[ecosystem-manager] ${label} validation failed`, msg);
      ctx.addIssue({ code: z.ZodIssueCode.custom, message: `Invalid ${label} response` });
      return z.NEVER;
    }
  });
}

// ---------------------------------------------------------------------------
// Public utilities
// ---------------------------------------------------------------------------

export function parseProtoResponse<Shape extends Message>(
  schema: ProtoSchema<Shape>,
  data: unknown,
  label: string,
): Shape {
  const result = schema.safeParse(data);
  if (!result.success) throw new Error(`Invalid ${label} response`);
  return result.data;
}

export function toProtoJson<Shape extends Message>(
  schema: GenMessage<Shape>,
  message: Shape,
): JsonValue {
  return toJson(schema, message, { useProtoFieldName: true, alwaysEmitImplicit: true });
}

export function buildMessage<Shape extends Message>(
  schema: GenMessage<Shape>,
  value?: MessageInitShape<GenMessage<Shape>>,
): Shape {
  return create(schema, value);
}

// ---------------------------------------------------------------------------
// Response schemas (zod validators)
// ---------------------------------------------------------------------------

export const taskListResponseSchema = createProtoSchema(TaskListResponseSchema, "task list");
export const taskDetailResponseSchema = createProtoSchema(TaskDetailResponseSchema, "task detail");
export const settingsResponseSchema = createProtoSchema(SettingsResponseSchema, "settings");
export const queueStatusResponseSchema = createProtoSchema(QueueStatusResponseSchema, "queue status");
export const processListResponseSchema = createProtoSchema(ProcessListResponseSchema, "process list");

// Domain-level schemas for direct validation
export const taskProtoSchema = createProtoSchema(TaskSchema, "task");
export const executionRecordProtoSchema = createProtoSchema(ExecutionRecordSchema, "execution record");
export const queueStatusProtoSchema = createProtoSchema(QueueStatusSchema, "queue status");
export const settingsProtoSchema = createProtoSchema(SettingsSchema, "settings");

// ---------------------------------------------------------------------------
// Type guards
// ---------------------------------------------------------------------------

const TASK_STATUSES = new Set(["pending", "in-progress", "completed", "failed", "completed-finalized", "failed-blocked", "archived"]);

function isTaskStatus(v: unknown): v is TaskStatus {
  return typeof v === "string" && TASK_STATUSES.has(v);
}

const isTheme = (v: unknown): v is Settings["display"]["theme"] =>
  v === "light" || v === "dark" || v === "auto";

const isEnabledFor = (v: unknown): v is Settings["recycler"]["enabled_for"] =>
  v === "off" || v === "resources" || v === "scenarios" || v === "both";

const isModelProvider = (v: unknown): v is Settings["recycler"]["model_provider"] =>
  v === "ollama" || v === "openrouter";

const isRunnerType = (v: unknown): v is Settings["agent"]["runner_type"] =>
  v === "claude-code" || v === "codex" || v === "opencode";

// ---------------------------------------------------------------------------
// Mapping functions: proto → UI types
// ---------------------------------------------------------------------------

/**
 * Maps a proto Task to the UI Task shape (snake_case fields).
 * Handles the target field normalization that previously required
 * normalizeTaskTargets().
 */
export function mapProtoTask(proto: ProtoTask, runtime?: {
  currentProcess?: ProtoProcessInfo;
  autoSteerPhaseIndex?: number;
}): Task {
  const targets = proto.targets.length > 0
    ? proto.targets
    : proto.target
      ? [proto.target]
      : [];

  return {
    id: proto.id,
    title: proto.title,
    type: proto.type as Task["type"],
    operation: proto.operation as Task["operation"],
    priority: proto.priority as Task["priority"],
    status: isTaskStatus(proto.status) ? proto.status : "pending",
    target: targets.filter(Boolean),
    notes: proto.notes || undefined,
    steer_set: proto.steerSet.length > 0 ? proto.steerSet : undefined,
    auto_steer_profile_id: proto.autoSteerProfileId || undefined,
    auto_steer_phase_index: runtime?.autoSteerPhaseIndex,
    steering_queue: proto.steeringQueue.length > 0
      ? proto.steeringQueue.map((step) => step.skillIds)
      : undefined,
    auto_requeue: proto.processorAutoRequeue,
    created_at: proto.createdAt,
    updated_at: proto.updatedAt,
    completion_count: proto.completionCount,
    last_completed_at: proto.lastCompletedAt || undefined,
    cooldown_until: proto.cooldownUntil || undefined,
    current_process: runtime?.currentProcess
      ? mapProtoProcessInfoToTaskProcess(runtime.currentProcess)
      : undefined,
  };
}

function mapProtoProcessInfoToTaskProcess(proto: ProtoProcessInfo): ProcessInfo {
  return {
    process_id: proto.runId || proto.taskId,
    agent_id: proto.agentTag,
    start_time: proto.startedAt,
  };
}

/**
 * Maps a proto QueueStatus to the UI QueueStatus shape.
 * Replaces normalizeQueueStatus().
 */
export function mapProtoQueueStatus(proto: ProtoQueueStatus): QueueStatus {
  const extras = proto as unknown as Record<string, unknown>;
  return {
    active: proto.isActive,
    slots_used: proto.maxSlots - proto.availableSlots,
    max_concurrent: proto.maxSlots,
    available_slots: proto.availableSlots,
    tasks_remaining: proto.pendingCount + proto.inProgressCount,
    cooldown_seconds: proto.cooldownSeconds,
    rate_limited: proto.isRateLimitPaused,
    rate_limit_retry_after: 0,
    rate_limit_pause_until: proto.rateLimitResumeAt || undefined,
    executions_completed: numberValue(extras.executionsCompleted) ?? 0,
    execution_limit: numberValue(extras.executionLimit) ?? 0,
    execution_limit_reached: booleanValue(extras.executionLimitReached) ?? false,
  };
}

/**
 * Maps a proto ProcessInfo to the UI RunningProcess shape.
 * Replaces normalizeRunningProcess().
 */
export function mapProtoProcessInfoToRunning(proto: ProtoProcessInfo, extra?: {
  taskTitle?: string;
  processType?: string;
}): RunningProcess {
  const startMs = proto.startedAt ? new Date(proto.startedAt).getTime() : 0;
  const elapsed = startMs > 0 ? Math.floor((Date.now() - startMs) / 1000) : 0;

  return {
    task_id: proto.taskId,
    task_title: extra?.taskTitle ?? "",
    process_id: proto.runId || proto.taskId,
    process_type: (extra?.processType ?? "task") as RunningProcess["process_type"],
    agent_id: proto.agentTag,
    start_time: proto.startedAt,
    elapsed_seconds: elapsed,
  };
}

/**
 * Maps a proto ExecutionRecord to the UI ExecutionHistory shape.
 * Replaces normalizeExecution().
 */
export function mapProtoExecutionRecord(proto: ProtoExecutionRecord): ExecutionHistory {
  const rateLimited = proto.rateLimited || proto.exitReason === "rate_limited";
  const status: ExecutionHistory["status"] =
    rateLimited ? "rate_limited"
    : proto.success ? "completed"
    : proto.exitReason === "timeout" ? "failed"
    : proto.endTime ? "failed"
    : "running";

  return {
    id: proto.executionId || proto.startTime || "",
    task_id: proto.taskId,
    task_title: proto.taskTitle || undefined,
    task_type: proto.taskType as ExecutionHistory["task_type"],
    task_operation: proto.taskOperation as ExecutionHistory["task_operation"],
    agent_tag: proto.agentTag || undefined,
    process_id: proto.processId || undefined,
    start_time: proto.startTime,
    end_time: proto.endTime || undefined,
    duration: proto.duration || undefined,
    status,
    exit_reason: proto.exitReason || undefined,
    prompt_size: proto.promptSize || undefined,
    prompt_path: proto.promptPath || undefined,
    output_path: proto.outputPath || undefined,
    clean_output_path: proto.cleanOutputPath || undefined,
    last_message_path: proto.lastMessagePath || undefined,
    transcript_path: proto.transcriptPath || undefined,
    auto_steer_profile_id: proto.autoSteerProfileId || undefined,
    auto_steer_iteration: proto.autoSteerIteration || undefined,
    steer_skill_ids: proto.steerSkillIds.length > 0 ? proto.steerSkillIds : undefined,
    steer_set_label: proto.steerSetLabel || undefined,
    steer_phase_index: proto.steerPhaseIndex || undefined,
    steer_phase_iteration: proto.steerPhaseIteration || undefined,
    steering_source: proto.steeringSource || undefined,
    timeout_allowed: proto.timeoutAllowed || undefined,
    rate_limited: rateLimited,
    retry_after: proto.retryAfter || undefined,
    success: proto.success,
  };
}

// Default settings fallback
const DEFAULT_SETTINGS: Settings = {
  processor: { concurrent_slots: 1, cooldown_seconds: 30, execution_limit: 0, active: false },
  agent: {
    max_turns: 60,
    allowed_tools: "Read,Write,Edit,Bash,LS,Glob,Grep",
    skip_permissions: true,
    task_timeout_minutes: 30,
    idle_timeout_cap_minutes: 30,
    runner_type: "claude-code",
  },
  display: { theme: "dark", condensed_mode: false },
  recycler: {
    enabled_for: "off",
    recycle_interval: 60,
    max_retries: 3,
    retry_delay_seconds: 2,
    model_provider: "ollama",
    model_name: "chat.default",
    completion_threshold: 3,
    failure_threshold: 5,
  },
};

/**
 * Maps a proto Settings (flat) to the UI Settings (nested) shape.
 * Replaces mapApiSettingsToUi().
 */
export function mapProtoSettings(proto: ProtoSettings): Settings {
  const recycler = proto.recycler;
  const extras = proto as unknown as Record<string, unknown>;
  return {
    processor: {
      concurrent_slots: proto.slots || DEFAULT_SETTINGS.processor.concurrent_slots,
      cooldown_seconds: proto.cooldownSeconds || DEFAULT_SETTINGS.processor.cooldown_seconds,
      execution_limit: numberValue(extras.executionLimit) ?? DEFAULT_SETTINGS.processor.execution_limit,
      active: proto.active,
    },
    agent: {
      max_turns: proto.maxTurns || DEFAULT_SETTINGS.agent.max_turns,
      allowed_tools: proto.allowedTools || DEFAULT_SETTINGS.agent.allowed_tools,
      skip_permissions: proto.skipPermissions,
      task_timeout_minutes: proto.taskTimeout || DEFAULT_SETTINGS.agent.task_timeout_minutes,
      idle_timeout_cap_minutes: proto.idleTimeoutCap || DEFAULT_SETTINGS.agent.idle_timeout_cap_minutes,
      runner_type: isRunnerType(proto.runnerType) ? proto.runnerType : DEFAULT_SETTINGS.agent.runner_type,
    },
    display: {
      theme: isTheme(proto.theme) ? proto.theme : DEFAULT_SETTINGS.display.theme,
      condensed_mode: proto.condensedMode,
    },
    recycler: {
      enabled_for: recycler && isEnabledFor(recycler.enabledFor)
        ? recycler.enabledFor
        : DEFAULT_SETTINGS.recycler.enabled_for,
      recycle_interval: recycler?.intervalSeconds ?? DEFAULT_SETTINGS.recycler.recycle_interval,
      max_retries: recycler?.maxRetries ?? DEFAULT_SETTINGS.recycler.max_retries,
      retry_delay_seconds: recycler?.retryDelaySeconds ?? DEFAULT_SETTINGS.recycler.retry_delay_seconds,
      model_provider: recycler && isModelProvider(recycler.modelProvider)
        ? recycler.modelProvider
        : DEFAULT_SETTINGS.recycler.model_provider,
      model_name: recycler?.modelName || DEFAULT_SETTINGS.recycler.model_name,
      completion_threshold: recycler?.completionThreshold ?? DEFAULT_SETTINGS.recycler.completion_threshold,
      failure_threshold: recycler?.failureThreshold ?? DEFAULT_SETTINGS.recycler.failure_threshold,
    },
  };
}

/**
 * Maps UI Settings (nested) to proto-compatible flat JSON for API requests.
 * Replaces mapUiSettingsToApi().
 */
export function mapUiSettingsToProtoJson(settings: Settings): Record<string, unknown> {
  return {
    theme: settings.display.theme,
    condensed_mode: settings.display.condensed_mode,
    slots: settings.processor.concurrent_slots,
    cooldown_seconds: settings.processor.cooldown_seconds,
    execution_limit: settings.processor.execution_limit,
    active: settings.processor.active,
    max_turns: settings.agent.max_turns,
    allowed_tools: settings.agent.allowed_tools,
    skip_permissions: settings.agent.skip_permissions,
    task_timeout: settings.agent.task_timeout_minutes,
    idle_timeout_cap: settings.agent.idle_timeout_cap_minutes,
    runner_type: settings.agent.runner_type,
    recycler: {
      enabled_for: settings.recycler.enabled_for,
      interval_seconds: settings.recycler.recycle_interval,
      max_retries: settings.recycler.max_retries,
      retry_delay_seconds: settings.recycler.retry_delay_seconds,
      model_provider: settings.recycler.model_provider,
      model_name: settings.recycler.model_name,
      completion_threshold: settings.recycler.completion_threshold,
      failure_threshold: settings.recycler.failure_threshold,
    },
  };
}

/**
 * Maps a proto Resource to the UI Resource shape.
 */
/**
 * Maps a proto ActiveTarget to the UI ActiveTarget shape.
 */
export function mapProtoActiveTarget(proto: ProtoActiveTarget): ActiveTarget {
  return {
    target: proto.target,
    task_id: proto.taskId,
    status: isTaskStatus(proto.status) ? proto.status : "pending",
    title: proto.title || undefined,
  };
}

// Default constraints matching current API constants.go values
const DEFAULT_CONSTRAINTS: SettingsConstraints = {
  slots: { min: 1, max: 5 },
  cooldown_seconds: { min: 5, max: 300 },
  execution_limit: { min: 0, max: 10000 },
  max_turns: { min: 5, max: 500 },
  task_timeout: { min: 5, max: 240 },
  idle_timeout_cap: { min: 2, max: 240 },
  recycler: {
    interval_seconds: { min: 30, max: 1800 },
    max_retries: { min: 0, max: 10 },
    retry_delay_seconds: { min: 1, max: 300 },
    completion_threshold: { min: 1, max: 10 },
    failure_threshold: { min: 1, max: 10 },
  },
};

function parseConstraintRange(raw: unknown, fallback: ConstraintRange): ConstraintRange {
  if (!raw || typeof raw !== "object") return fallback;
  const r = raw as Record<string, unknown>;
  return {
    min: typeof r.min === "number" ? r.min : fallback.min,
    max: typeof r.max === "number" ? r.max : fallback.max,
  };
}

function parseConstraints(raw: unknown): SettingsConstraints {
  if (!raw || typeof raw !== "object") return { ...DEFAULT_CONSTRAINTS };
  const r = raw as Record<string, unknown>;
  const recyclerRaw = r.recycler as Record<string, unknown> | undefined;
  return {
    slots: parseConstraintRange(r.slots, DEFAULT_CONSTRAINTS.slots),
    cooldown_seconds: parseConstraintRange(r.cooldown_seconds, DEFAULT_CONSTRAINTS.cooldown_seconds),
    execution_limit: parseConstraintRange(r.execution_limit, DEFAULT_CONSTRAINTS.execution_limit),
    max_turns: parseConstraintRange(r.max_turns, DEFAULT_CONSTRAINTS.max_turns),
    task_timeout: parseConstraintRange(r.task_timeout, DEFAULT_CONSTRAINTS.task_timeout),
    idle_timeout_cap: parseConstraintRange(r.idle_timeout_cap, DEFAULT_CONSTRAINTS.idle_timeout_cap),
    recycler: {
      interval_seconds: parseConstraintRange(recyclerRaw?.interval_seconds, DEFAULT_CONSTRAINTS.recycler.interval_seconds),
      max_retries: parseConstraintRange(recyclerRaw?.max_retries, DEFAULT_CONSTRAINTS.recycler.max_retries),
      retry_delay_seconds: parseConstraintRange(recyclerRaw?.retry_delay_seconds, DEFAULT_CONSTRAINTS.recycler.retry_delay_seconds),
      completion_threshold: parseConstraintRange(recyclerRaw?.completion_threshold, DEFAULT_CONSTRAINTS.recycler.completion_threshold),
      failure_threshold: parseConstraintRange(recyclerRaw?.failure_threshold, DEFAULT_CONSTRAINTS.recycler.failure_threshold),
    },
  };
}

/**
 * Gracefully parse any raw JSON as a proto Settings, with fallback defaults.
 * Used at the GET /api/settings boundary.
 * Returns both settings and constraints (if present in the API response).
 */
export function parseSettingsResponse(raw: unknown): { settings: Settings; constraints: SettingsConstraints } {
  const wrapper = raw as Record<string, unknown> | undefined;
  const constraints = parseConstraints(wrapper?.constraints);
  try {
    // The API wraps settings in { settings: { ... } }
    const source = wrapper?.settings ?? raw;
    const result = settingsProtoSchema.safeParse(source);
    if (result.success) {
      const settings = mapProtoSettings(result.data);
      // Merge fields not in the proto schema from the raw source
      const s = source as Record<string, unknown>;
      if (typeof s.execution_limit === "number") {
        settings.processor.execution_limit = s.execution_limit;
      }
      return { settings, constraints };
    }
  } catch {
    // fall through to defaults
  }
  return { settings: { ...DEFAULT_SETTINGS }, constraints };
}

/**
 * Gracefully parse raw task JSON. Falls back to pass-through with minimal normalization
 * when proto validation fails (e.g., server returns extra runtime fields).
 */
export function parseTaskResponse(raw: unknown): Task {
  if (!raw || typeof raw !== "object") {
    throw new Error("Invalid task response");
  }
  const result = taskProtoSchema.safeParse(raw);
  if (result.success) {
    const task = mapProtoTask(result.data);
    // Proto parsing ignores runtime fields (current_process, execution_count, etc.)
    // that the Go handler injects beyond the proto schema. Merge them from raw data.
    const r = raw as Record<string, unknown>;
    const rawProcess = r.current_process ?? r.currentProcess;
    const process = normalizeProcessInfo(rawProcess);
    if (process) {
      task.current_process = process;
    }
    const executionCount = numberValue(r.execution_count);
    const queueIndex = numberValue(r.steering_queue_index);
    const queueLabel = stringValue(r.steering_queue_set_label);
    const queueTotal = numberValue(r.steering_queue_total);
    const queueExhausted = booleanValue(r.steering_queue_exhausted);
    const phaseIndex = numberValue(r.auto_steer_phase_index);
    if (executionCount !== undefined) task.execution_count = executionCount;
    if (queueIndex !== undefined) task.steering_queue_index = queueIndex;
    if (queueLabel !== undefined) task.steering_queue_set_label = queueLabel;
    if (queueTotal !== undefined) task.steering_queue_total = queueTotal;
    if (queueExhausted !== undefined) task.steering_queue_exhausted = queueExhausted;
    if (phaseIndex !== undefined) task.auto_steer_phase_index = phaseIndex;
    return task;
  }
  // Fallback: minimal normalization for responses with extra runtime fields
  return fallbackNormalizeTask(raw);
}

/**
 * Minimal fallback normalization when proto parse fails.
 * Handles extra runtime fields the Go handler injects beyond the proto schema.
 */
function normalizeProcessInfo(raw: unknown): ProcessInfo | undefined {
  if (!isRecord(raw)) return undefined;
  return {
    process_id: firstString(raw, ["process_id", "run_id", "processId", "runId"]) ?? "",
    agent_id: firstString(raw, ["agent_id", "agent_tag", "agentId", "agentTag"]) ?? "",
    start_time: firstString(raw, ["start_time", "started_at", "startTime", "startedAt"]) ?? "",
  };
}

function parseSkillSet(value: unknown): string[] | undefined {
  if (!Array.isArray(value)) return undefined;
  const ids = value
    .map((id) => (typeof id === "string" ? id.trim() : ""))
    .filter(Boolean);
  return ids.length > 0 ? ids : undefined;
}

function parseSteeringQueue(value: unknown): string[][] | undefined {
  if (!Array.isArray(value)) return undefined;
  const queue = value
    .map((step) => {
      if (Array.isArray(step)) return parseSkillSet(step);
      if (step && typeof step === "object") {
        const record = step as Record<string, unknown>;
        return parseSkillSet(record.skill_ids ?? record.skillIds);
      }
      return undefined;
    })
    .filter((step): step is string[] => Array.isArray(step) && step.length > 0);
  return queue.length > 0 ? queue : undefined;
}

function fallbackNormalizeTask(raw: unknown): Task {
  if (!isRecord(raw)) {
    throw new Error("Invalid task response");
  }
  const rawTarget = raw.target;
  const targets = Array.isArray(raw.targets) ? raw.targets
    : Array.isArray(rawTarget) ? rawTarget
    : typeof rawTarget === "string" ? [rawTarget]
    : [];

  return {
    id: stringValue(raw.id) ?? "",
    title: stringValue(raw.title) ?? "",
    type: raw.type === "scenario" ? "scenario" : "resource",
    operation: raw.operation === "improver" ? "improver" : "generator",
    priority: raw.priority === "critical" || raw.priority === "high" || raw.priority === "medium" || raw.priority === "low"
      ? raw.priority
      : "medium",
    status: isTaskStatus(raw.status) ? raw.status : "pending",
    target: targets.filter((target): target is string => typeof target === "string" && target.length > 0),
    notes: stringValue(raw.notes),
    steer_set: parseSkillSet(raw.steer_set ?? raw.steerSet),
    auto_steer_profile_id: firstString(raw, ["auto_steer_profile_id", "autoSteerProfileId"]),
    auto_steer_phase_index: firstNumber(raw, ["auto_steer_phase_index", "autoSteerPhaseIndex"]),
    steering_queue: parseSteeringQueue(raw.steering_queue ?? raw.steeringQueue),
    auto_requeue: firstBoolean(raw, ["auto_requeue", "processor_auto_requeue", "processorAutoRequeue"]) ?? true,
    created_at: firstString(raw, ["created_at", "createdAt"]) ?? "",
    updated_at: firstString(raw, ["updated_at", "updatedAt"]) ?? "",
    completion_count: firstNumber(raw, ["completion_count", "completionCount"]) ?? 0,
    last_completed_at: firstString(raw, ["last_completed_at", "lastCompletedAt"]),
    cooldown_until: firstString(raw, ["cooldown_until", "cooldownUntil"]),
    current_process: normalizeProcessInfo(raw.current_process ?? raw.currentProcess),
  };
}

/**
 * Parse raw execution record JSON, with fallback.
 */
export function parseExecutionResponse(raw: unknown): ExecutionHistory {
  if (!raw || typeof raw !== "object") {
    throw new Error("Invalid execution response");
  }
  const result = executionRecordProtoSchema.safeParse(raw);
  if (result.success) {
    return mapProtoExecutionRecord(result.data);
  }
  // Fallback for responses that don't match proto exactly
  return fallbackNormalizeExecution(raw);
}

function fallbackNormalizeExecution(raw: unknown): ExecutionHistory {
  if (!isRecord(raw)) {
    throw new Error("Invalid execution response");
  }
  const id = firstString(raw, ["id", "execution_id", "executionId", "start_time"]) ?? "";
  const startTime = firstString(raw, ["start_time", "startTime"]) ?? "";
  const endTime = firstString(raw, ["end_time", "endTime"]);
  const exitReason = firstString(raw, ["exit_reason", "exitReason"]);
  const rateLimited = firstBoolean(raw, ["rate_limited", "rateLimited"]) ?? exitReason === "rate_limited";
  const success = booleanValue(raw.success);
  const rawStatus = stringValue(raw.status)?.toLowerCase();

  const status: ExecutionHistory["status"] =
    rawStatus === "rate_limited" || rateLimited ? "rate_limited"
    : rawStatus === "completed" || success === true ? "completed"
    : rawStatus === "failed" || success === false ? "failed"
    : "running";

  return {
    id,
    task_id: firstString(raw, ["task_id", "taskId"]) ?? "",
    task_title: firstString(raw, ["task_title", "taskTitle"]),
    task_type: raw.task_type === "resource" || raw.task_type === "scenario" ? raw.task_type : undefined,
    task_operation: raw.task_operation === "generator" || raw.task_operation === "improver" ? raw.task_operation : undefined,
    agent_tag: firstString(raw, ["agent_tag", "agentTag"]),
    process_id: firstNumber(raw, ["process_id", "processId"]),
    start_time: startTime,
    end_time: endTime,
    duration: stringValue(raw.duration),
    status,
    exit_code: firstNumber(raw, ["exit_code", "exitCode"]),
    exit_reason: exitReason,
    prompt_size: firstNumber(raw, ["prompt_size", "promptSize"]),
    prompt_path: firstString(raw, ["prompt_path", "promptPath"]),
    output_path: firstString(raw, ["output_path", "outputPath"]),
    clean_output_path: firstString(raw, ["clean_output_path", "cleanOutputPath"]),
    last_message_path: firstString(raw, ["last_message_path", "lastMessagePath"]),
    transcript_path: firstString(raw, ["transcript_path", "transcriptPath"]),
    auto_steer_profile_id: firstString(raw, ["auto_steer_profile_id", "autoSteerProfileId"]),
    auto_steer_iteration: firstNumber(raw, ["auto_steer_iteration", "autoSteerIteration"]),
    steer_skill_ids: parseSkillSet(raw.steer_skill_ids ?? raw.steerSkillIds),
    steer_set_label: firstString(raw, ["steer_set_label", "steerSetLabel"]),
    steer_phase_index: firstNumber(raw, ["steer_phase_index", "steerPhaseIndex"]),
    steer_phase_iteration: firstNumber(raw, ["steer_phase_iteration", "steerPhaseIteration"]),
    steering_source: firstString(raw, ["steering_source", "steeringSource"]),
    timeout_allowed: firstString(raw, ["timeout_allowed", "timeoutAllowed"]),
    rate_limited: rateLimited,
    retry_after: firstNumber(raw, ["retry_after", "retryAfter"]),
    success,
  };
}

/**
 * Parse raw queue status JSON with proto validation fallback.
 */
export function parseQueueStatusResponse(raw: unknown): QueueStatus {
  if (!raw || typeof raw !== "object") {
    return {
      active: false, slots_used: 0, max_concurrent: 1,
      available_slots: 1, tasks_remaining: 0, cooldown_seconds: 30,
      rate_limited: false, rate_limit_retry_after: 0,
    };
  }
  const result = queueStatusProtoSchema.safeParse(raw);
  if (result.success) {
    const status = mapProtoQueueStatus(result.data);
    // Merge fields not in the proto schema from the raw source
    const r = raw as Record<string, unknown>;
    if (typeof r.executions_completed === "number") status.executions_completed = r.executions_completed;
    if (typeof r.execution_limit === "number") status.execution_limit = r.execution_limit;
    if (typeof r.execution_limit_reached === "boolean") status.execution_limit_reached = r.execution_limit_reached;
    return status;
  }
  // Fallback: direct field mapping for snake_case responses
  const r = raw as Record<string, unknown>;
  const maxSlots = firstNumber(r, ["max_slots", "max_concurrent"]) ?? 1;
  const availableSlots = numberValue(r.available_slots) ?? maxSlots;
  const pendingCount = numberValue(r.pending_count) ?? 0;
  const inProgressCount = numberValue(r.in_progress_count) ?? 0;
  const rateLimitInfo = isRecord(r.rate_limit_info) ? r.rate_limit_info : {};
  return {
    active: firstBoolean(r, ["is_active", "active", "processor_active", "settings_active"]) ?? false,
    slots_used: firstNumber(r, ["slots_used", "executing_count", "running_count"]) ?? (maxSlots - availableSlots),
    max_concurrent: maxSlots,
    available_slots: availableSlots,
    tasks_remaining: numberValue(r.tasks_remaining) ?? pendingCount + inProgressCount,
    cooldown_seconds: numberValue(r.cooldown_seconds) ?? 30,
    rate_limited: firstBoolean(r, ["is_rate_limit_paused", "rate_limited"]) ?? booleanValue(rateLimitInfo.paused) ?? false,
    rate_limit_retry_after: numberValue(r.rate_limit_retry_after) ?? numberValue(rateLimitInfo.remaining_secs) ?? 0,
    rate_limit_pause_until: firstString(r, ["rate_limit_resume_at", "rate_limit_pause_until"]) ?? stringValue(rateLimitInfo.pause_until),
    executions_completed: numberValue(r.executions_completed) ?? 0,
    execution_limit: numberValue(r.execution_limit) ?? 0,
    execution_limit_reached: booleanValue(r.execution_limit_reached) ?? false,
  };
}

/**
 * Parse a raw running process JSON entry with proto validation fallback.
 */
export function parseRunningProcessResponse(raw: unknown): RunningProcess {
  if (!raw || typeof raw !== "object") {
    return {
      task_id: "", task_title: "", process_id: "",
      process_type: "task", agent_id: "", start_time: "", elapsed_seconds: 0,
    };
  }
  const r = raw as Record<string, unknown>;
  const startTime = firstString(r, ["started_at", "start_time"]) ?? "";
  const startMs = startTime ? new Date(startTime).getTime() : 0;
  const elapsed = numberValue(r.elapsed_seconds) ?? (startMs > 0 ? Math.floor((Date.now() - startMs) / 1000) : 0);

  return {
    task_id: firstString(r, ["task_id", "taskId"]) ?? "",
    task_title: firstString(r, ["task_title", "taskTitle"]) ?? "",
    process_id: firstString(r, ["run_id", "process_id", "runId", "processId", "task_id"]) ?? "",
    process_type: "task",
    agent_id: firstString(r, ["agent_tag", "agent_id", "agentTag", "agentId"]) ?? "",
    start_time: startTime,
    elapsed_seconds: elapsed,
  };
}

// ---------------------------------------------------------------------------
// ActiveTarget parse helpers
// (Resource / Scenario moved to Connect-RPC — see ui/src/api/discovery.ts)
// ---------------------------------------------------------------------------

/**
 * Parse a raw active target JSON entry with proto validation fallback.
 */
export function parseActiveTargetResponse(raw: unknown): ActiveTarget | null {
  if (!isRecord(raw)) return null;
  const target = stringValue(raw.target) ?? "";
  const taskId = firstString(raw, ["task_id", "taskId"]) ?? "";
  const status = stringValue(raw.status) ?? "";
  if (!target || !taskId || !status) return null;

  return {
    target,
    task_id: taskId,
    status: isTaskStatus(status) ? status : "pending",
    title: stringValue(raw.title),
  };
}

// Re-export schema descriptors for request building
export { TaskSchema, ExecutionRecordSchema, QueueStatusSchema, SettingsSchema, RecyclerSettingsSchema };
