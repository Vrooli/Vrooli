import { z } from "zod";
import { fromJson, type GenMessage } from "@bufbuild/protobuf";
import { createValidator } from "@bufbuild/protovalidate";
import {
  type InfrastructureHealthResponse,
  type KnowledgeHealthResponse,
  type SearchResponse,
  InfrastructureHealthResponseSchema,
  KnowledgeHealthResponseSchema,
  SearchResponseSchema,
} from "@vrooli/proto-types/knowledge-observatory/v1/api_pb";

const validator = createValidator();

function createProtoSchema<T>(schema: GenMessage<T>, label: string): z.ZodType<T> {
  return z.unknown().transform((value, ctx) => {
    try {
      const message = fromJson(schema, value, {
        jsonOptions: { useProtoNames: true, ignoreUnknownFields: true },
      });
      const validation = validator.validate(schema, message);
      if (validation.kind !== "valid") {
        console.error(`[knowledge-observatory] ${label} response validation failed`, validation.error);
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: `Invalid ${label} response`,
        });
        return z.NEVER;
      }
      return message;
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      console.error(`[knowledge-observatory] ${label} response validation failed`, message);
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: `Invalid ${label} response`,
      });
      return z.NEVER;
    }
  }) as z.ZodType<T>;
}

export const infrastructureHealthResponseSchema: z.ZodType<InfrastructureHealthResponse> = createProtoSchema(
  InfrastructureHealthResponseSchema,
  "health"
);

export const searchResponseSchema: z.ZodType<SearchResponse> = createProtoSchema(
  SearchResponseSchema,
  "search"
);

export const knowledgeHealthResponseSchema: z.ZodType<KnowledgeHealthResponse> = createProtoSchema(
  KnowledgeHealthResponseSchema,
  "knowledge health"
);

export type InfrastructureHealthResponseProto = z.infer<typeof infrastructureHealthResponseSchema>;
export type SearchResponseProto = z.infer<typeof searchResponseSchema>;
export type KnowledgeHealthResponseProto = z.infer<typeof knowledgeHealthResponseSchema>;

export function parseProtoResponse<T>(schema: z.ZodType<T>, data: unknown, label: string): T {
  const result = schema.safeParse(data);
  if (!result.success) {
    throw new Error(`Invalid ${label} response`);
  }
  return result.data;
}
