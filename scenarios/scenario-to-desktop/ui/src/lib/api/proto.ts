/**
 * Proto parsing utilities for scenario-to-desktop.
 *
 * This module provides utilities for parsing JSON API responses into
 * protobuf message types, handling the snake_case to camelCase conversion
 * that occurs between Go/server responses and TypeScript proto types.
 */

import {
  fromJson,
  toJson,
  type JsonReadOptions,
  type JsonWriteOptions,
  type Message,
  type DescMessage,
  type JsonValue,
} from "@bufbuild/protobuf";

// Allow unknown fields to handle cases where API returns fields with different names
// than what the proto schema expects (e.g., execution_id -> executionId vs json_name="id")
const readOptions: Partial<JsonReadOptions> = { ignoreUnknownFields: true };
const writeOptions: Partial<JsonWriteOptions> = { useProtoFieldName: true };

const isPlainObject = (value: unknown): value is Record<string, unknown> =>
  Boolean(value) && typeof value === "object" && !Array.isArray(value);

const toLowerCamel = (key: string): string =>
  key.replace(/_([a-z0-9])/g, (_, char: string) => char.toUpperCase());

/**
 * normalizeProtoJsonInput converts protojson "UseProtoNames" payloads (snake_case)
 * into standard protobuf JSON names (lowerCamel) so bufbuild's fromJson() can parse them.
 *
 * It intentionally avoids touching keys that look like selectors/paths/etc.
 */
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
      if (rawKey.startsWith("_")) return rawKey;
      if (!rawKey.includes("_")) return rawKey;
      if (!/^[a-z][a-z0-9_]*$/.test(rawKey)) return rawKey;
      return toLowerCamel(rawKey);
    })();

    if (!(key in out)) {
      out[key] = normalizeProtoJsonInput(rawVal);
    }
  }
  return out;
};

/**
 * Parse a raw JSON value into a protobuf message type.
 * Throws if parsing fails.
 *
 * @param schema - The protobuf message schema (e.g., PipelineStatusSchema)
 * @param raw - The raw JSON data to parse
 * @returns The parsed protobuf message
 *
 * @example
 * ```ts
 * import { PipelineStatusSchema, PipelineStatus } from "@vrooli/proto-types/scenario-to-desktop/v1/pipeline/types_pb";
 * const status = parseProtoStrict<PipelineStatus>(PipelineStatusSchema, apiResponse);
 * ```
 */
export const parseProtoStrict = <T>(schema: DescMessage, raw: unknown): T =>
  fromJson(schema, normalizeProtoJsonInput(raw) as JsonValue, readOptions) as T;

/**
 * Parse a raw JSON value into a protobuf message type with a fallback default.
 * Returns the default value if parsing fails instead of throwing.
 *
 * @param schema - The protobuf message schema
 * @param raw - The raw JSON data to parse
 * @param defaultValue - The value to return if parsing fails
 * @returns The parsed protobuf message or the default value
 *
 * @example
 * ```ts
 * const status = parseProtoSafe(PipelineStatusSchema, apiResponse, null);
 * if (status) { ... }
 * ```
 */
export const parseProtoSafe = <T>(
  schema: DescMessage,
  raw: unknown,
  defaultValue: T
): T => {
  try {
    return parseProtoStrict<T>(schema, raw);
  } catch {
    return defaultValue;
  }
};

/**
 * Convert a protobuf message back to JSON.
 * Returns an empty object if conversion fails.
 *
 * @param schema - The protobuf message schema
 * @param message - The protobuf message to convert
 * @returns A JSON object representation of the message
 *
 * @example
 * ```ts
 * const json = protoMessageToJson(PipelineConfigSchema, config);
 * await fetch('/api/pipeline', { body: JSON.stringify(json) });
 * ```
 */
export const protoMessageToJson = (
  schema: DescMessage,
  message: Message
): Record<string, unknown> => {
  try {
    return toJson(schema, message, writeOptions) as Record<string, unknown>;
  } catch {
    return {};
  }
};

/**
 * Convert a protobuf Timestamp to a JavaScript Date.
 * Handles both proto Timestamp objects and ISO string values.
 *
 * @param timestamp - The timestamp to convert
 * @returns A Date object, or undefined if the timestamp is invalid
 */
export const timestampToDate = (
  timestamp: { seconds?: bigint; nanos?: number } | string | undefined | null
): Date | undefined => {
  if (!timestamp) return undefined;

  if (typeof timestamp === "string") {
    const date = new Date(timestamp);
    return isNaN(date.getTime()) ? undefined : date;
  }

  if (typeof timestamp === "object" && "seconds" in timestamp) {
    const seconds = Number(timestamp.seconds ?? 0n);
    const nanos = timestamp.nanos ?? 0;
    return new Date(seconds * 1000 + nanos / 1_000_000);
  }

  return undefined;
};

/**
 * Convert a JavaScript Date to a protobuf-compatible timestamp string.
 *
 * @param date - The date to convert
 * @returns An ISO string suitable for proto timestamps
 */
export const dateToTimestamp = (date: Date): string => date.toISOString();
