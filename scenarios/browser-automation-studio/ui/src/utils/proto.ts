import { fromJson, toJson, type JsonReadOptions, type JsonWriteOptions, type Message } from '@bufbuild/protobuf';

const readOptions: Partial<JsonReadOptions> = { ignoreUnknownFields: false };
const writeOptions: Partial<JsonWriteOptions> = { useProtoFieldName: true };

const isPlainObject = (value: unknown): value is Record<string, unknown> =>
  Boolean(value) && typeof value === 'object' && !Array.isArray(value);

const toLowerCamel = (key: string): string =>
  key.replace(/_([a-z0-9])/g, (_, char: string) => char.toUpperCase());

// normalizeProtoJsonInput converts protojson "UseProtoNames" payloads (snake_case)
// into standard protobuf JSON names (lowerCamel) so bufbuild's fromJson() can parse them.
//
// It intentionally avoids touching keys that look like selectors/paths/etc.
const normalizeProtoJsonInput = (value: unknown): unknown => {
  if (Array.isArray(value)) {
    return value.map(normalizeProtoJsonInput);
  }
  if (!isPlainObject(value)) {
    return value;
  }

  const out: Record<string, unknown> = {};
  for (const [rawKey, rawVal] of Object.entries(value)) {
    const key = (() => {
      if (rawKey.startsWith('_')) return rawKey;
      if (!rawKey.includes('_')) return rawKey;
      if (!/^[a-z][a-z0-9_]*$/.test(rawKey)) return rawKey;
      return toLowerCamel(rawKey);
    })();

    if (!(key in out)) {
      out[key] = normalizeProtoJsonInput(rawVal);
    }
  }
  return out;
};

export const parseProtoStrict = <T>(schema: any, raw: unknown): T =>
  fromJson(schema, normalizeProtoJsonInput(raw) as any, readOptions) as T;

export const protoMessageToJson = (schema: any, message: Message): Record<string, unknown> => {
  try {
    return toJson(schema, message, writeOptions) as Record<string, unknown>;
  } catch {
    return {};
  }
};
