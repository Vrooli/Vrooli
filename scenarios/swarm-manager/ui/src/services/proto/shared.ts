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

const validator = createValidator();

export type ProtoSchema<Shape extends Message> = z.ZodType<Shape, z.ZodTypeDef, unknown>;

export function isJsonValue(value: unknown): value is JsonValue {
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

export function toFiniteNumber(value: number | bigint | undefined): number | undefined {
  if (typeof value === "bigint") {
    const asNumber = Number(value);
    return Number.isFinite(asNumber) ? asNumber : undefined;
  }
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

export function createProtoSchema<Shape extends Message>(
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

export const fileTypeSet = new Set<string>(["file", "directory"]);

export function isFileType(value: unknown): value is "file" | "directory" {
  return typeof value === "string" && fileTypeSet.has(value);
}
