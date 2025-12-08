import { fromJson, toJson, type JsonReadOptions, type JsonWriteOptions, type Message } from '@bufbuild/protobuf';

const readOptions: Partial<JsonReadOptions> = { ignoreUnknownFields: false };
const writeOptions: Partial<JsonWriteOptions> = { useProtoFieldName: true };

export const parseProtoStrict = <T>(schema: any, raw: unknown): T =>
  fromJson(schema, raw as any, readOptions) as T;

export const protoMessageToJson = (schema: any, message: Message): Record<string, unknown> => {
  try {
    return toJson(schema, message, writeOptions) as Record<string, unknown>;
  } catch {
    return {};
  }
};
