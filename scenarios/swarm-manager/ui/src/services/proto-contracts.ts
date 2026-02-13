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
  ListBacklogItemsResponseSchema,
  QueueBacklogItemResponseSchema,
  BacklogResearchResponseSchema,
} from "@vrooli/proto-types/swarm-manager/v1/api/backlog_pb";
import { AgentManagerStatusResponseSchema } from "@vrooli/proto-types/swarm-manager/v1/api/agent_manager_pb";
import { SettingsResponseSchema } from "@vrooli/proto-types/swarm-manager/v1/api/settings_pb";
import {
  ListScenariosResponseSchema,
  ScenarioResponseSchema,
  DeleteScenarioResponseSchema,
  ScenarioFilesResponseSchema,
  DeleteScenarioRequestSchema,
  PreserveFilesRequestSchema,
  type DeleteScenarioResponse,
} from "@vrooli/proto-types/swarm-manager/v1/api/scenarios_pb";
import type { ScenarioFile } from "@vrooli/proto-types/swarm-manager/v1/api/scenarios_pb";
import type {
  BacklogItem as BacklogItemDomain,
  BacklogFile as BacklogFileDomain,
  Scenario as ScenarioDomain,
  ScenarioFile as ScenarioFileDomain,
  DeleteScenarioResponse as DeleteScenarioDomain,
  Settings as SettingsDomain,
  ThemePreference,
} from "../types";
import {
  BACKLOG_KINDS,
  BACKLOG_RESEARCH_TARGETS,
  BACKLOG_STATUSES,
  SCENARIO_STATUSES,
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
const backlogResearchTargetSet = new Set<string>(BACKLOG_RESEARCH_TARGETS);
const scenarioStatusSet = new Set<string>(SCENARIO_STATUSES);
const fileTypeSet = new Set<string>(["file", "directory"]);

function isBacklogStatus(value: unknown): value is BacklogItemDomain["status"] {
  return typeof value === "string" && backlogStatusSet.has(value);
}

function isBacklogKind(value: unknown): value is BacklogItemDomain["kind"] {
  return typeof value === "string" && backlogKindSet.has(value);
}

function isBacklogResearchTarget(value: unknown): value is BacklogItemDomain["researchTarget"] {
  return typeof value === "string" && backlogResearchTargetSet.has(value);
}

function isScenarioStatus(value: unknown): value is ScenarioDomain["status"] {
  return typeof value === "string" && scenarioStatusSet.has(value);
}

function isFileType(value: unknown): value is BacklogFileDomain["type"] {
  return typeof value === "string" && fileTypeSet.has(value);
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
export const scenarioFilesResponseSchema = createProtoSchema(
  ScenarioFilesResponseSchema,
  "scenario files"
);
export const settingsResponseSchema = createProtoSchema(
  SettingsResponseSchema,
  "settings"
);

export { DeleteScenarioRequestSchema, PreserveFilesRequestSchema };

export function mapProtoBacklogItem(protoItem: BacklogItem): BacklogItemDomain {
  const status = isBacklogStatus(protoItem.status) ? protoItem.status : "backlog";
  const kind = isBacklogKind(protoItem.kind) ? protoItem.kind : "idea";
  const researchTarget = isBacklogResearchTarget(protoItem.researchTarget) ? protoItem.researchTarget : undefined;
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
    ...(researchTarget ? { researchTarget } : {}),
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

export function mapProtoSettings(protoSettings: Settings): SettingsDomain {
  return {
    theme: normalizeThemePreference(protoSettings.theme),
    customFocus: protoSettings.customFocus ?? "",
    insightsEnabled: protoSettings.insightsEnabled ?? false,
    insightsAutoAnalyze: protoSettings.insightsAutoAnalyze ?? false,
  };
}
