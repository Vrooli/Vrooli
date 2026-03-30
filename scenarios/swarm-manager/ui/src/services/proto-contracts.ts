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
import type {
  BacklogItem,
  BacklogFile,
} from "@vrooli/proto-types/swarm-manager/v1/domain/backlog_pb";
import type { Settings } from "@vrooli/proto-types/swarm-manager/v1/domain/settings_pb";
import type { Scenario } from "@vrooli/proto-types/swarm-manager/v1/domain/scenario_pb";
import {
  BacklogItemResponseSchema,
  BacklogFilesResponseSchema,
  BacklogFileResponseSchema,
  BacklogFileOperationResponseSchema,
  ListBacklogItemsResponseSchema,
  QueueBacklogItemResponseSchema,
  BacklogResearchResponseSchema,
} from "@vrooli/proto-types/swarm-manager/v1/api/backlog_pb";
import { AgentManagerStatusResponseSchema } from "@vrooli/proto-types/swarm-manager/v1/api/agent_manager_pb";
import { GraphResponseSchema } from "@vrooli/proto-types/swarm-manager/v1/api/graph_pb";
import { SettingsResponseSchema } from "@vrooli/proto-types/swarm-manager/v1/api/settings_pb";
import {
  ListAgentActivitiesResponseSchema,
  AgentActivityResponseSchema,
} from "@vrooli/proto-types/swarm-manager/v1/api/agent_activity_pb";
import {
  ListScenariosResponseSchema,
  ScenarioResponseSchema,
  DeleteScenarioResponseSchema,
  ScenarioFilesResponseSchema,
  DeleteScenarioRequestSchema,
  PreserveFilesRequestSchema,
  SpecSyncArchiveRequestSchema,
  SpecSyncArchiveResponseSchema,
  type DeleteScenarioResponse,
  type SpecSyncArchiveResponse,
} from "@vrooli/proto-types/swarm-manager/v1/api/scenarios_pb";
import type { ScenarioFile } from "@vrooli/proto-types/swarm-manager/v1/api/scenarios_pb";
import {
  ListExecutionResponseSchema,
  ExecutionResponseSchema,
  CreateExecutionRequestSchema,
  FollowUpExecutionRequestSchema,
} from "@vrooli/proto-types/swarm-manager/v1/api/execution_pb";
import type {
  ExecutionRecord as ProtoExecutionRecord,
  Finalization as ProtoFinalization,
  ReviewResult as ProtoReviewResult,
} from "@vrooli/proto-types/swarm-manager/v1/domain/execution_pb";
import type { AgentActivity as ProtoAgentActivity } from "@vrooli/proto-types/swarm-manager/v1/domain/agent_activity_pb";
import type {
  AgentActivity,
  AgentActivityInteractionType,
  AgentActivityOwnerType,
  AgentActivityPurpose,
  AgentActivityStatus,
  BacklogItem as BacklogItemDomain,
  BacklogFile as BacklogFileDomain,
  Scenario as ScenarioDomain,
  ScenarioFile as ScenarioFileDomain,
  DeleteScenarioResponse as DeleteScenarioDomain,
  Settings as SettingsDomain,
  ThemePreference,
  ExecutionRecord as ExecutionRecordDomain,
  ExecutionStatus,
  ExecutionMode,
  Finalization,
  FinalizationAggregateClassification,
  FinalizationPhase,
  FinalizationScopeSource,
  FinalizationStatus,
  ReviewClassification,
} from "../types";
import {
  BACKLOG_KINDS,
  BACKLOG_STATUSES,
  SCENARIO_STATUSES,
  EXECUTION_STATUSES,
  EXECUTION_MODES,
} from "../types";

const validator = createValidator();

type ProtoSchema<Shape extends Message> = z.ZodType<Shape, z.ZodTypeDef, unknown>;

function isJsonValue(value: unknown): value is JsonValue {
  if (value === null) {
    return true;
  }
  const valueType = typeof value;
  if (valueType === "string" || valueType === "number" || valueType === "boolean") {
    return true;
  }
  if (Array.isArray(value)) {
    return value.every(isJsonValue);
  }
  if (valueType === "object") {
    return Object.values(value as Record<string, unknown>).every(isJsonValue);
  }
  return false;
}

const backlogStatusSet = new Set<string>(BACKLOG_STATUSES);
const backlogKindSet = new Set<string>(BACKLOG_KINDS);
const scenarioStatusSet = new Set<string>(SCENARIO_STATUSES);
const fileTypeSet = new Set<string>(["file", "directory"]);
const executionStatusSet = new Set<string>(EXECUTION_STATUSES);
const executionModeSet = new Set<string>(EXECUTION_MODES);
const agentActivityStatusSet = new Set<string>([
  "pending",
  "starting",
  "running",
  "needs_review",
  "complete",
  "failed",
  "cancelled",
  "unspecified",
]);
const agentActivityPurposeSet = new Set<string>([
  "initialize",
  "workshop",
  "finalize",
  "research",
  "process",
  "fixup",
  "followup",
  "spec_sync",
  "classify",
]);
const agentActivityInteractionTypeSet = new Set<string>(["spawn", "continue"]);
const agentActivityOwnerTypeSet = new Set<string>(["backlog", "capture", "scenario"]);

function isBacklogStatus(value: unknown): value is BacklogItemDomain["status"] {
  return typeof value === "string" && backlogStatusSet.has(value);
}

function isBacklogKind(value: unknown): value is BacklogItemDomain["kind"] {
  return typeof value === "string" && backlogKindSet.has(value);
}

function isScenarioStatus(value: unknown): value is ScenarioDomain["status"] {
  return typeof value === "string" && scenarioStatusSet.has(value);
}

function isFileType(value: unknown): value is BacklogFileDomain["type"] {
  return typeof value === "string" && fileTypeSet.has(value);
}

function isExecutionStatus(value: unknown): value is ExecutionStatus {
  return typeof value === "string" && executionStatusSet.has(value);
}

function isExecutionMode(value: unknown): value is ExecutionMode {
  return typeof value === "string" && executionModeSet.has(value);
}

function isAgentActivityStatus(value: unknown): value is AgentActivityStatus {
  return typeof value === "string" && agentActivityStatusSet.has(value);
}

function isAgentActivityPurpose(value: unknown): value is AgentActivityPurpose {
  return typeof value === "string" && agentActivityPurposeSet.has(value);
}

function isAgentActivityInteractionType(value: unknown): value is AgentActivityInteractionType {
  return typeof value === "string" && agentActivityInteractionTypeSet.has(value);
}

function isAgentActivityOwnerType(value: unknown): value is AgentActivityOwnerType {
  return typeof value === "string" && agentActivityOwnerTypeSet.has(value);
}

function isThemePreference(value: unknown): value is ThemePreference {
  return value === "dark" || value === "light" || value === "system";
}

function normalizeThemePreference(value?: string): ThemePreference {
  return isThemePreference(value) ? value : "dark";
}

function toFiniteNumber(value: number | bigint | undefined): number | undefined {
  if (typeof value === "bigint") {
    const asNumber = Number(value);
    return Number.isFinite(asNumber) ? asNumber : undefined;
  }
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

function createProtoSchema<Shape extends Message>(
  schema: GenMessage<Shape>,
  label: string
): ProtoSchema<Shape> {
  return z.unknown().transform<Shape>((value, ctx) => {
    if (!isJsonValue(value)) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: `Invalid ${label} response`,
      });
      return z.NEVER;
    }
    try {
      // fromJson accepts both proto field names and JSON names; only ignoreUnknownFields needed.
      const message = fromJson(schema, value, {
        ignoreUnknownFields: true,
      });
      const validation = validator.validate(schema, message);
      if (validation.kind !== "valid") {
        console.error(`[swarm-manager] ${label} response validation failed`, validation.error);
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: `Invalid ${label} response`,
        });
        return z.NEVER;
      }
      return message;
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      console.error(`[swarm-manager] ${label} response validation failed`, message);
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: `Invalid ${label} response`,
      });
      return z.NEVER;
    }
  });
}

export function parseProtoResponse<Shape extends Message>(
  schema: ProtoSchema<Shape>,
  data: unknown,
  label: string
): Shape {
  const result = schema.safeParse(data);
  if (!result.success) {
    throw new Error(`Invalid ${label} response`);
  }
  return result.data;
}

export function requireProtoField<T>(value: T | undefined, label: string): T {
  if (value === undefined || value === null) {
    throw new Error(`Invalid ${label} response`);
  }
  return value;
}

export function toProtoJson<Shape extends Message>(
  schema: GenMessage<Shape>,
  message: Shape
): JsonValue {
  return toJson(schema, message, {
    useProtoFieldName: true,
    alwaysEmitImplicit: true,
  });
}

export function buildMessage<Shape extends Message>(
  schema: GenMessage<Shape>,
  value?: MessageInitShape<GenMessage<Shape>>
): Shape {
  return create(schema, value);
}

export const listBacklogResponseSchema = createProtoSchema(
  ListBacklogItemsResponseSchema,
  "backlog list"
);
export const backlogItemResponseSchema = createProtoSchema(
  BacklogItemResponseSchema,
  "backlog item"
);
export const backlogFilesResponseSchema = createProtoSchema(
  BacklogFilesResponseSchema,
  "backlog files"
);
export const backlogFileResponseSchema = createProtoSchema(
  BacklogFileResponseSchema,
  "backlog file"
);
export const backlogFileOperationResponseSchema = createProtoSchema(
  BacklogFileOperationResponseSchema,
  "backlog file operation"
);
export const queueBacklogResponseSchema = createProtoSchema(
  QueueBacklogItemResponseSchema,
  "backlog queue"
);
export const backlogResearchResponseSchema = createProtoSchema(
  BacklogResearchResponseSchema,
  "backlog research"
);
export const listScenariosResponseSchema = createProtoSchema(
  ListScenariosResponseSchema,
  "scenarios list"
);
export const agentManagerStatusResponseSchema = createProtoSchema(
  AgentManagerStatusResponseSchema,
  "agent-manager status"
);
export const scenarioResponseSchema = createProtoSchema(
  ScenarioResponseSchema,
  "scenario"
);
export const deleteScenarioResponseSchema = createProtoSchema(
  DeleteScenarioResponseSchema,
  "scenario delete"
);
export const specSyncArchiveResponseSchema = createProtoSchema(
  SpecSyncArchiveResponseSchema,
  "spec-sync-archive"
);
export const scenarioFilesResponseSchema = createProtoSchema(
  ScenarioFilesResponseSchema,
  "scenario files"
);
export const settingsResponseSchema = createProtoSchema(
  SettingsResponseSchema,
  "settings"
);
export const graphResponseSchema = createProtoSchema(
  GraphResponseSchema,
  "graph"
);
export const listAgentActivitiesResponseSchema = createProtoSchema(
  ListAgentActivitiesResponseSchema,
  "agent activities"
);
export const agentActivityResponseSchema = createProtoSchema(
  AgentActivityResponseSchema,
  "agent activity"
);

export { DeleteScenarioRequestSchema, PreserveFilesRequestSchema, SpecSyncArchiveRequestSchema };

export function mapProtoBacklogItem(protoItem: BacklogItem): BacklogItemDomain {
  const status = isBacklogStatus(protoItem.status) ? protoItem.status : "backlog";
  const kind = isBacklogKind(protoItem.kind) ? protoItem.kind : "idea";
  return {
    name: protoItem.name ?? "",
    title: protoItem.title ?? "",
    description: protoItem.description ?? "",
    status,
    priority: protoItem.priority ?? 0,
    tags: protoItem.tags ?? [],
    created: protoItem.created ?? "",
    updated: protoItem.updated ?? "",
    kind,
    ...(protoItem.dependsOn?.length ? { dependsOn: protoItem.dependsOn } : {}),
    ...(protoItem.initiative ? { initiative: protoItem.initiative } : {}),
    ...(protoItem.acceptanceAllow?.length ? { acceptanceAllow: protoItem.acceptanceAllow } : {}),
    ...(protoItem.acceptanceDeny?.length ? { acceptanceDeny: protoItem.acceptanceDeny } : {}),
  };
}

export function mapProtoBacklogFile(protoFile: BacklogFile): BacklogFileDomain {
  const size = toFiniteNumber(protoFile.size);
  const children = protoFile.children?.map(mapProtoBacklogFile) ?? [];
  const fileType = isFileType(protoFile.type) ? protoFile.type : "file";
  return {
    name: protoFile.name ?? "",
    path: protoFile.path ?? "",
    type: fileType,
    ...(size !== undefined ? { size } : {}),
    ...(children.length > 0 ? { children } : {}),
  };
}

export function mapProtoScenario(protoScenario: Scenario): ScenarioDomain {
  const status = isScenarioStatus(protoScenario.status) ? protoScenario.status : "unknown";
  return {
    name: protoScenario.name ?? "",
    displayName: protoScenario.displayName ?? "",
    description: protoScenario.description ?? "",
    status,
    priority: protoScenario.priority ?? 0,
    completenessScore:
      typeof protoScenario.completenessScore === "number"
        ? protoScenario.completenessScore
        : undefined,
    isGreenfield: protoScenario.isGreenfield ?? false,
    tags: protoScenario.tags ?? [],
  };
}

export function mapProtoScenarioFile(protoFile: ScenarioFile): ScenarioFileDomain {
  const size = toFiniteNumber(protoFile.size);
  const children = protoFile.children?.map(mapProtoScenarioFile) ?? [];
  const fileType = isFileType(protoFile.type) ? protoFile.type : "file";
  return {
    name: protoFile.name ?? "",
    path: protoFile.path ?? "",
    type: fileType,
    ...(size !== undefined ? { size } : {}),
    ...(children.length > 0 ? { children } : {}),
  };
}

export function mapDeleteScenarioResponse(
  protoResponse: DeleteScenarioResponse
): DeleteScenarioDomain {
  return {
    name: protoResponse.name,
    archived: protoResponse.archived,
    message: protoResponse.message,
    backlogIdeaName: protoResponse.backlogIdeaName,
    preservedFiles: protoResponse.preservedFiles,
  };
}

export function mapSpecSyncArchiveResponse(
  protoResponse: SpecSyncArchiveResponse
): { executionId: string; status: string; message: string } {
  return {
    executionId: protoResponse.executionId,
    status: protoResponse.status,
    message: protoResponse.message,
  };
}

export function mapProtoSettings(protoSettings: Settings): SettingsDomain {
  const mode = isExecutionMode(protoSettings.defaultMode) ? protoSettings.defaultMode : "manual";
  return {
    theme: normalizeThemePreference(protoSettings.theme),
    defaultMode: mode,
    autoFixup: protoSettings.autoFixup ?? false,
    maxFixupAttempts: protoSettings.maxFixupAttempts ?? 0,
    maxAutoRounds: protoSettings.maxAutoRounds ?? 10,
    autoInitializeWorkshop: protoSettings.autoInitializeWorkshop ?? true,
    autoAdvanceWorkshop: protoSettings.autoAdvanceWorkshop ?? true,
    autoCascadeWorkshop: protoSettings.autoCascadeWorkshop ?? true,
    agentMaxTurns: protoSettings.agentMaxTurns ?? 60,
    agentTimeoutSeconds: protoSettings.agentTimeoutSeconds ?? 900,
    agentRequiresApproval: protoSettings.agentRequiresApproval ?? true,
    searchDebounceMs: protoSettings.searchDebounceMs ?? 300,
    toastDurationMs: protoSettings.toastDurationMs ?? 5000,
    confirmDestructiveActions: protoSettings.confirmDestructiveActions ?? true,
    reviewCodeQualityMinScore: protoSettings.reviewCodeQualityMinScore ?? 60,
    reviewTestMinPassRate: protoSettings.reviewTestMinPassRate ?? 1.0,
    reviewMaxBlockingViolations: protoSettings.reviewMaxBlockingViolations ?? 0,
    reviewMaxWarnings: protoSettings.reviewMaxWarnings ?? -1,
    reviewRequireScreenshots: protoSettings.reviewRequireScreenshots ?? true,
    reviewRequireTests: protoSettings.reviewRequireTests ?? true,
  };
}

export const listExecutionResponseSchema = createProtoSchema(
  ListExecutionResponseSchema,
  "execution list"
);
export const executionResponseSchema = createProtoSchema(
  ExecutionResponseSchema,
  "execution"
);
export { CreateExecutionRequestSchema, FollowUpExecutionRequestSchema };

export function mapProtoAgentActivity(proto: ProtoAgentActivity): AgentActivity {
  return {
    activityId: proto.activityId ?? "",
    ownerType: isAgentActivityOwnerType(proto.ownerType) ? proto.ownerType : "backlog",
    ownerKind: proto.ownerKind,
    ownerName: proto.ownerName ?? "",
    ownerTitle: proto.ownerTitle,
    executionId: proto.executionId,
    purpose: isAgentActivityPurpose(proto.purpose) ? proto.purpose : "process",
    interactionType: isAgentActivityInteractionType(proto.interactionType)
      ? proto.interactionType
      : "spawn",
    taskId: proto.taskId,
    runId: proto.runId,
    status: isAgentActivityStatus(proto.status) ? proto.status : "unspecified",
    requestedAt: proto.requestedAt ?? "",
    startedAt: proto.startedAt,
    finishedAt: proto.finishedAt,
    failureReason: proto.failureReason,
    requestedBy: proto.requestedBy,
    metadata: proto.metadata ?? {},
    updatedAt: proto.updatedAt ?? "",
  };
}

export function mapProtoExecutionRecord(proto: ProtoExecutionRecord): ExecutionRecordDomain {
  const status = isExecutionStatus(proto.status) ? proto.status : "pending";
  const mode = isExecutionMode(proto.mode) ? proto.mode : "manual";
  const record: ExecutionRecordDomain = {
    executionId: proto.executionId ?? "",
    backlogKind: (proto.backlogKind ?? "idea") as ExecutionRecordDomain["backlogKind"],
    backlogName: proto.backlogName ?? "",
    taskId: proto.taskId,
    runId: proto.runId,
    status,
    mode,
    startedAt: proto.startedAt,
    finishedAt: proto.finishedAt,
    failureReason: proto.failureReason,
    startedBy: proto.startedBy,
    operation: proto.operation as ExecutionRecordDomain["operation"],
    parentExecutionId: proto.parentExecutionId,
    fixupAttempt: proto.fixupAttempt ?? 0,
    createdAt: proto.createdAt ?? "",
    updatedAt: proto.updatedAt ?? "",
  };
  if (proto.finalization) {
    record.finalization = mapProtoFinalization(proto.finalization);
  }
  return record;
}

function mapProtoReviewResult(
  proto: ProtoReviewResult,
): NonNullable<NonNullable<ExecutionRecordDomain["finalization"]>["scenarios"][number]["review"]["result"]> {
  return {
    jobId: proto.jobId ?? "",
    classification: (proto.classification ?? "not_assessable") as ReviewClassification,
    dimensions: (proto.dimensions ?? []).map((dim) => ({
      name: dim.name ?? "",
      status: dim.status ?? "",
      details: dim.details,
    })),
    summary: proto.summary ?? "",
    reviewedAt: proto.reviewedAt ?? "",
  };
}

function mapProtoFinalization(proto: ProtoFinalization): Finalization {
  return {
    eligible: proto.eligible ?? false,
    status: (proto.status ?? "pending") as FinalizationStatus,
    phase: (proto.phase ?? "scope_detection") as FinalizationPhase,
    scopeSource: (proto.scopeSource ?? "none") as FinalizationScopeSource,
    skipReason: proto.skipReason,
    startedAt: proto.startedAt,
    completedAt: proto.completedAt,
    warnings: (proto.warnings ?? []).map((warning) => ({
      code: warning.code ?? "",
      scenarioName: warning.scenarioName,
      message: warning.message ?? "",
      retryable: warning.retryable ?? false,
      createdAt: warning.createdAt ?? "",
    })),
    affectedScenarios: proto.affectedScenarios ?? [],
    aggregateClassification: (proto.aggregateClassification ?? "not_assessable") as FinalizationAggregateClassification,
    aggregateSummary: proto.aggregateSummary,
    scenarios: (proto.scenarios ?? []).map((scenario) => ({
      scenarioName: scenario.scenarioName ?? "",
      changedPaths: scenario.changedPaths ?? [],
      restart: {
        status: (scenario.restart?.status ?? "pending") as FinalizationStatus,
        attempts: scenario.restart?.attempts ?? 0,
        lastError: scenario.restart?.lastError,
        startedAt: scenario.restart?.startedAt,
        finishedAt: scenario.restart?.finishedAt,
      },
      health: {
        status: (scenario.health?.status ?? "pending") as FinalizationStatus,
        scenarioStatus: scenario.health?.scenarioStatus,
        healthStatus: scenario.health?.healthStatus,
        schemaValid: scenario.health?.schemaValid ?? false,
        details: scenario.health?.details,
        checkedAt: scenario.health?.checkedAt,
      },
      review: {
        status: (scenario.review?.status ?? "pending") as FinalizationStatus,
        jobId: scenario.review?.jobId,
        skipReason: scenario.review?.skipReason,
        result: scenario.review?.result ? mapProtoReviewResult(scenario.review.result) : undefined,
      },
    })),
  };
}
