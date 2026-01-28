import { z } from "zod";
import {
  create,
  fromJson,
  toJson,
  type DescMessage,
  type JsonValue,
  type MessageInitShape,
  type MessageShape,
} from "@bufbuild/protobuf";
import { createValidator } from "@bufbuild/protovalidate";
import type {
  Idea,
  IdeaFile,
} from "@vrooli/proto-types/swarm-manager/v1/domain/idea_pb";
import type { Scenario } from "@vrooli/proto-types/swarm-manager/v1/domain/scenario_pb";
import {
  IdeaResponseSchema,
  IdeaFilesResponseSchema,
  IdeaFileResponseSchema,
  ListIdeasResponseSchema,
  QueueIdeaResponseSchema,
} from "@vrooli/proto-types/swarm-manager/v1/api/ideas_pb";
import {
  ListScenariosResponseSchema,
  ScenarioResponseSchema,
  DeleteScenarioResponseSchema,
} from "@vrooli/proto-types/swarm-manager/v1/api/scenarios_pb";
import type {
  Idea as IdeaDomain,
  IdeaFile as IdeaFileDomain,
  Scenario as ScenarioDomain,
  DeleteScenarioResponse as DeleteScenarioDomain,
} from "../types";

const validator = createValidator();

function toFiniteNumber(value: number | bigint | undefined): number | undefined {
  if (typeof value === "bigint") {
    const asNumber = Number(value);
    return Number.isFinite(asNumber) ? asNumber : undefined;
  }
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

function createProtoSchema<Desc extends DescMessage>(
  schema: Desc,
  label: string
): z.ZodType<MessageShape<Desc>> {
  return z.unknown().transform((value, ctx) => {
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
      return message as MessageShape<Desc>;
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      console.error(`[swarm-manager] ${label} response validation failed`, message);
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: `Invalid ${label} response`,
      });
      return z.NEVER;
    }
  }) as unknown as z.ZodType<MessageShape<Desc>>;
}

export function parseProtoResponse<T>(schema: z.ZodType<T>, data: unknown, label: string): T {
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

export function toProtoJson<Desc extends DescMessage>(
  schema: Desc,
  message: MessageShape<Desc>
): JsonValue {
  return toJson(schema, message, {
    useProtoFieldName: true,
    alwaysEmitImplicit: true,
  });
}

export function buildMessage<Desc extends DescMessage>(
  schema: Desc,
  value?: MessageInitShape<Desc>
): MessageShape<Desc> {
  return create(schema, value);
}

export const listIdeasResponseSchema = createProtoSchema(
  ListIdeasResponseSchema,
  "ideas list"
);
export const ideaResponseSchema = createProtoSchema(IdeaResponseSchema, "idea");
export const ideaFilesResponseSchema = createProtoSchema(
  IdeaFilesResponseSchema,
  "idea files"
);
export const ideaFileResponseSchema = createProtoSchema(
  IdeaFileResponseSchema,
  "idea file"
);
export const queueIdeaResponseSchema = createProtoSchema(
  QueueIdeaResponseSchema,
  "idea queue"
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

export function mapProtoIdea(protoIdea: Idea): IdeaDomain {
  return {
    name: protoIdea.name ?? "",
    title: protoIdea.title ?? "",
    description: protoIdea.description ?? "",
    status: (protoIdea.status as IdeaDomain["status"]) || "backlog",
    priority: protoIdea.priority ?? 0,
    tags: protoIdea.tags ?? [],
    created: protoIdea.created ?? "",
    updated: protoIdea.updated ?? "",
  };
}

export function mapProtoIdeaFile(protoFile: IdeaFile): IdeaFileDomain {
  const size = toFiniteNumber(protoFile.size);
  const children = protoFile.children?.map(mapProtoIdeaFile) ?? [];
  return {
    name: protoFile.name ?? "",
    path: protoFile.path ?? "",
    type: (protoFile.type as IdeaFileDomain["type"]) || "file",
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

export function mapDeleteScenarioResponse(
  protoResponse: {
    name: string;
    archived: boolean;
    message: string;
  }
): DeleteScenarioDomain {
  return {
    name: protoResponse.name,
    archived: protoResponse.archived,
    message: protoResponse.message,
  };
}
