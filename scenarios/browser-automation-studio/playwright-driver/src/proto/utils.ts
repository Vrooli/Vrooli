/**
 * Proto Utilities
 *
 * Utilities for working with proto-generated types, including JSON
 * serialization/deserialization and type conversion helpers.
 *
 * WIRE FORMAT:
 *   The Go API uses protojson with UseProtoNames:true, which produces
 *   snake_case JSON field names. The @bufbuild/protobuf library handles
 *   this automatically when using fromJson/toJson with the appropriate options.
 */

import { create, fromJson, toJson, type MessageShape, type DescMessage, type JsonValue } from '@bufbuild/protobuf';
import { timestampFromDate, timestampDate, type Timestamp } from '@bufbuild/protobuf/wkt';

// =============================================================================
// JSON SERIALIZATION OPTIONS
// =============================================================================

/**
 * JSON parsing options for incoming data from the Go API.
 *
 * - ignoreUnknownFields: false - Strict mode, reject unknown fields
 *   This catches schema mismatches early rather than silently ignoring them
 */
export const PARSE_OPTIONS = {
  ignoreUnknownFields: false,
} as const;

/**
 * JSON parsing options for lenient parsing (e.g., user input).
 *
 * - ignoreUnknownFields: true - Accept unknown fields
 */
export const PARSE_OPTIONS_LENIENT = {
  ignoreUnknownFields: true,
} as const;

// =============================================================================
// PARSING UTILITIES
// =============================================================================

/**
 * Parse JSON data into a proto message with strict validation.
 *
 * @param schema - The proto message schema (e.g., StepOutcomeSchema)
 * @param data - JSON data (object or string)
 * @returns Parsed proto message
 * @throws Error if data doesn't match schema
 *
 * @example
 * ```ts
 * import { parseProto, CompiledInstructionSchema } from '../proto';
 *
 * const instruction = parseProto(CompiledInstructionSchema, jsonData);
 * ```
 */
export function parseProto<T extends DescMessage>(
  schema: T,
  data: unknown
): MessageShape<T> {
  const jsonData: unknown = typeof data === 'string' ? JSON.parse(data) : data;
  if (!isJsonValue(jsonData)) {
    throw new Error('Invalid JSON data for proto parsing');
  }
  return fromJson(schema, jsonData, PARSE_OPTIONS);
}

/**
 * Parse JSON data into a proto message with lenient validation.
 * Unknown fields are ignored rather than causing an error.
 *
 * @param schema - The proto message schema
 * @param data - JSON data (object or string)
 * @returns Parsed proto message
 */
export function parseProtoLenient<T extends DescMessage>(
  schema: T,
  data: unknown
): MessageShape<T> {
  const jsonData: unknown = typeof data === 'string' ? JSON.parse(data) : data;
  if (!isJsonValue(jsonData)) {
    throw new Error('Invalid JSON data for proto parsing');
  }
  return fromJson(schema, jsonData, PARSE_OPTIONS_LENIENT);
}

/**
 * Try to parse JSON data into a proto message, returning null on failure.
 *
 * @param schema - The proto message schema
 * @param data - JSON data (object or string)
 * @returns Parsed proto message or null
 */
export function tryParseProto<T extends DescMessage>(
  schema: T,
  data: unknown
): MessageShape<T> | null {
  try {
    return parseProto(schema, data);
  } catch {
    return null;
  }
}

// =============================================================================
// SERIALIZATION UTILITIES
// =============================================================================

/**
 * Serialize a proto message to JSON object.
 *
 * The output uses snake_case field names to match the Go API wire format.
 *
 * @param schema - The proto message schema
 * @param message - The proto message to serialize
 * @returns JSON object
 */
export function toJsonObject<T extends DescMessage>(
  schema: T,
  message: MessageShape<T>
): Record<string, unknown> {
  return toJson(schema, message) as Record<string, unknown>;
}

/**
 * Serialize a proto message to JSON string.
 *
 * @param schema - The proto message schema
 * @param message - The proto message to serialize
 * @param pretty - Whether to format with indentation (default: false)
 * @returns JSON string
 */
export function toJsonString<T extends DescMessage>(
  schema: T,
  message: MessageShape<T>,
  pretty = false
): string {
  const obj = toJson(schema, message);
  return pretty ? JSON.stringify(obj, null, 2) : JSON.stringify(obj);
}

// =============================================================================
// MESSAGE CREATION UTILITIES
// =============================================================================

/**
 * Create a new proto message with the given initial values.
 *
 * @param schema - The proto message schema
 * @param init - Optional initial values
 * @returns New proto message
 *
 * @example
 * ```ts
 * import { createMessage, StepOutcomeSchema } from '../proto';
 *
 * const outcome = createMessage(StepOutcomeSchema, {
 *   schemaVersion: 'automation-step-outcome-v1',
 *   success: true,
 * });
 * ```
 */
export function createMessage<T extends DescMessage>(
  schema: T,
  init?: Partial<MessageShape<T>>
): MessageShape<T> {
  return create(schema, init as MessageShape<T>);
}

// =============================================================================
// TIMESTAMP UTILITIES
// =============================================================================

/**
 * Convert a JavaScript Date to a proto Timestamp.
 *
 * @param date - JavaScript Date object
 * @returns Proto Timestamp
 */
export function dateToTimestamp(date: Date): Timestamp {
  return timestampFromDate(date);
}

/**
 * Convert a proto Timestamp to a JavaScript Date.
 *
 * @param timestamp - Proto Timestamp
 * @returns JavaScript Date object
 */
export function timestampToDate(timestamp: Timestamp): Date {
  return timestampDate(timestamp);
}

/**
 * Convert a proto Timestamp to an ISO 8601 string.
 *
 * @param timestamp - Proto Timestamp
 * @returns ISO 8601 date string
 */
export function timestampToIso(timestamp: Timestamp): string {
  return timestampDate(timestamp).toISOString();
}

// =============================================================================
// SCHEMA VERSION CONSTANTS
// =============================================================================

/**
 * Schema version for StepOutcome messages.
 * Must match the Go constant in contracts.go.
 */
export const STEP_OUTCOME_SCHEMA_VERSION = 'automation-step-outcome-v1';

/**
 * Payload version for StepOutcome messages.
 */
export const PAYLOAD_VERSION = '1';

/**
 * Schema version for ExecutionPlan messages.
 */
export const EXECUTION_PLAN_SCHEMA_VERSION = 'automation-plan-v1';

// =============================================================================
// JSON VALUE CONVERSION UTILITIES
// =============================================================================

import type { JsonValue as ProtoJsonValue, JsonObject, JsonList } from '@vrooli/proto-types/common/v1/types_pb';

const isJsonValue = (value: unknown): value is JsonValue => {
  if (
    value === null ||
    typeof value === 'string' ||
    typeof value === 'number' ||
    typeof value === 'boolean'
  ) {
    return true;
  }

  if (Array.isArray(value)) {
    return value.every(isJsonValue);
  }

  if (typeof value === 'object' && value !== null) {
    return Object.values(value as Record<string, unknown>).every(isJsonValue);
  }

  return false;
};

/**
 * Convert a proto JsonValue to a plain JavaScript value.
 *
 * Handles both:
 * 1. Typed bufbuild proto objects with value.kind.case
 * 2. Raw JSON objects from protojson (fallback for incomplete deserialization)
 *
 * @param value - Proto JsonValue or raw JSON object
 * @returns Plain JavaScript value (string, number, boolean, object, array, or null)
 */
export function jsonValueToPlain(value: ProtoJsonValue | undefined): unknown {
  if (!value) {
    return undefined;
  }

  // Handle typed bufbuild proto objects (with kind.case discriminator)
  if (value.kind && value.kind.case) {
    switch (value.kind.case) {
      case 'boolValue':
        return value.kind.value;
      case 'intValue':
        // Convert bigint to number (safe for most values)
        return Number(value.kind.value);
      case 'doubleValue':
        return value.kind.value;
      case 'stringValue':
        return value.kind.value;
      case 'objectValue':
        return jsonObjectToPlain(value.kind.value);
      case 'listValue':
        return jsonListToPlain(value.kind.value);
      case 'nullValue':
        return null;
      case 'bytesValue':
        // Return as base64 string for compatibility
        return Buffer.from(value.kind.value).toString('base64');
      default:
        // Unknown case - fall through to raw JSON handling
        break;
    }
  }

  // Fallback: Handle raw JSON objects from protojson serialization
  // This handles cases where nested JsonValue fields aren't fully deserialized
  const raw = value as Record<string, unknown>;

  // Check for snake_case field names (from UseProtoNames: true)
  if ('string_value' in raw) return raw.string_value;
  if ('bool_value' in raw) return raw.bool_value;
  if ('int_value' in raw) return Number(raw.int_value);
  if ('double_value' in raw) return raw.double_value;
  if ('null_value' in raw) return null;
  if ('bytes_value' in raw) {
    const bytes = raw.bytes_value;
    if (typeof bytes === 'string') return Buffer.from(bytes, 'base64').toString('base64');
    if (bytes instanceof Uint8Array) return Buffer.from(bytes).toString('base64');
    return bytes;
  }

  // Check for camelCase field names (default protojson)
  if ('stringValue' in raw) return raw.stringValue;
  if ('boolValue' in raw) return raw.boolValue;
  if ('intValue' in raw) return Number(raw.intValue);
  if ('doubleValue' in raw) return raw.doubleValue;
  if ('nullValue' in raw) return null;
  if ('bytesValue' in raw) {
    const bytes = raw.bytesValue;
    if (typeof bytes === 'string') return Buffer.from(bytes, 'base64').toString('base64');
    if (bytes instanceof Uint8Array) return Buffer.from(bytes).toString('base64');
    return bytes;
  }

  // Handle nested objects/lists
  if ('object_value' in raw || 'objectValue' in raw) {
    const objVal = (raw.object_value ?? raw.objectValue) as { fields?: Record<string, unknown> };
    if (objVal?.fields) {
      const result: Record<string, unknown> = {};
      for (const [key, fieldValue] of Object.entries(objVal.fields)) {
        result[key] = jsonValueToPlain(fieldValue as ProtoJsonValue);
      }
      return result;
    }
    return {};
  }

  if ('list_value' in raw || 'listValue' in raw) {
    const listVal = (raw.list_value ?? raw.listValue) as { values?: unknown[] };
    if (listVal?.values) {
      return listVal.values.map((v) => jsonValueToPlain(v as ProtoJsonValue));
    }
    return [];
  }

  return undefined;
}

/**
 * Convert a proto JsonObject to a plain JavaScript object.
 */
export function jsonObjectToPlain(obj: JsonObject | undefined): Record<string, unknown> {
  if (!obj || !obj.fields) {
    return {};
  }

  const result: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(obj.fields)) {
    result[key] = jsonValueToPlain(value);
  }
  return result;
}

/**
 * Convert a proto JsonList to a plain JavaScript array.
 */
export function jsonListToPlain(list: JsonList | undefined): unknown[] {
  if (!list || !list.values) {
    return [];
  }

  return list.values.map(jsonValueToPlain);
}

/**
 * Convert a map of proto JsonValue to a plain JavaScript object.
 * Used for CompiledInstruction.params conversion.
 */
export function jsonValueMapToPlain(map: { [key: string]: ProtoJsonValue } | undefined): Record<string, unknown> {
  if (!map) {
    return {};
  }

  const result: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(map)) {
    result[key] = jsonValueToPlain(value);
  }
  return result;
}
