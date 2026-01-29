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
import type { Scenario } from "@vrooli/proto-types/swarm-manager/v1/domain/scenario_pb";
import {
  BacklogItemResponseSchema,
  BacklogFilesResponseSchema,
  BacklogFileResponseSchema,
  ListBacklogItemsResponseSchema,
  QueueBacklogItemResponseSchema,
} from "@vrooli/proto-types/swarm-manager/v1/api/backlog_pb";
import {
  ListScenariosResponseSchema,
  ScenarioResponseSchema,
  DeleteScenarioResponseSchema,
  ScenarioFilesResponseSchema,
  DeleteScenarioRequestSchema,
  PreserveFilesRequestSchema,
} from "@vrooli/proto-types/swarm-manager/v1/api/scenarios_pb";
import type { ScenarioFile } from "@vrooli/proto-types/swarm-manager/v1/api/scenarios_pb";
import type {
  BacklogItem as BacklogItemDomain,
  BacklogFile as BacklogFileDomain,
  Scenario as ScenarioDomain,
  ScenarioFile as ScenarioFileDomain,
  DeleteScenarioResponse as DeleteScenarioDomain,
} from "../types";

const validator = createValidator();

type ProtoSchema<Shape extends Message> = z.ZodType<Shape, z.ZodTypeDef, unknown>;

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
    try {
      const jsonValue = value as JsonValue;
      const message = fromJson(schema, jsonValue, {
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
export const listScenariosResponseSchema = createProtoSchema(
  ListScenariosResponseSchema,
  "scenarios list"
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

export { DeleteScenarioRequestSchema, PreserveFilesRequestSchema };

export function mapProtoBacklogItem(protoItem: BacklogItem): BacklogItemDomain {
  return {
    name: protoItem.name ?? "",
    title: protoItem.title ?? "",
    description: protoItem.description ?? "",
    status: (protoItem.status as BacklogItemDomain["status"]) || "backlog",
    priority: protoItem.priority ?? 0,
    tags: protoItem.tags ?? [],
    created: protoItem.created ?? "",
    updated: protoItem.updated ?? "",
    kind: (protoItem.kind as BacklogItemDomain["kind"]) || "idea",
    ...(protoItem.researchTarget ? { researchTarget: protoItem.researchTarget as BacklogItemDomain["researchTarget"] } : {}),
  };
}

export function mapProtoBacklogFile(protoFile: BacklogFile): BacklogFileDomain {
  const size = toFiniteNumber(protoFile.size);
  const children = protoFile.children?.map(mapProtoBacklogFile) ?? [];
  return {
    name: protoFile.name ?? "",
    path: protoFile.path ?? "",
    type: (protoFile.type as BacklogFileDomain["type"]) || "file",
    ...(size !== undefined ? { size } : {}),
    ...(children.length > 0 ? { children } : {}),
  };
}

export function mapProtoScenario(protoScenario: Scenario): ScenarioDomain {
  return {
    name: protoScenario.name ?? "",
    displayName: protoScenario.displayName ?? "",
    description: protoScenario.description ?? "",
    status: (protoScenario.status as ScenarioDomain["status"]) || "unknown",
    priority: protoScenario.priority ?? 0,
    completenessScore:
      typeof protoScenario.completenessScore === "number"
        ? protoScenario.completenessScore
        : undefined,
    isGreenfield: protoScenario.isGreenfield ?? false,
    tags: protoScenario.tags ?? [],
    recommendationsEnabled: protoScenario.recommendationsEnabled ?? true,
  };
}

export function mapProtoScenarioFile(protoFile: ScenarioFile): ScenarioFileDomain {
  const size = toFiniteNumber(protoFile.size);
  const children = protoFile.children?.map(mapProtoScenarioFile) ?? [];
  return {
    name: protoFile.name ?? "",
    path: protoFile.path ?? "",
    type: (protoFile.type as ScenarioFileDomain["type"]) || "file",
    ...(size !== undefined ? { size } : {}),
    ...(children.length > 0 ? { children } : {}),
  };
}

export function mapDeleteScenarioResponse(
  protoResponse: {
    name: string;
    archived: boolean;
    message: string;
    backlogIdeaName?: string;
    preservedFiles?: string[];
  }
): DeleteScenarioDomain {
  return {
    name: protoResponse.name,
    archived: protoResponse.archived,
    message: protoResponse.message,
    ...(protoResponse.backlogIdeaName ? { backlogIdeaName: protoResponse.backlogIdeaName } : {}),
    ...(protoResponse.preservedFiles?.length ? { preservedFiles: protoResponse.preservedFiles } : {}),
  };
}
