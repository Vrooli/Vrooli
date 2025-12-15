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

import { create, fromJson, toJson, type MessageShape, type DescMessage } from '@bufbuild/protobuf';
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
  const jsonData = typeof data === 'string' ? JSON.parse(data) : data;
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
  const jsonData = typeof data === 'string' ? JSON.parse(data) : data;
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
export function dateToTimestamp(date: Date) {
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

import type { JsonValue, JsonObject, JsonList } from '@vrooli/proto-types/common/v1/types_pb';

/**
 * Convert a proto JsonValue to a plain JavaScript value.
 *
 * @param value - Proto JsonValue
 * @returns Plain JavaScript value (string, number, boolean, object, array, or null)
 */
export function jsonValueToPlain(value: JsonValue | undefined): unknown {
  if (!value || !value.kind) {
    return undefined;
  }

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
      return undefined;
  }
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
export function jsonValueMapToPlain(map: { [key: string]: JsonValue } | undefined): Record<string, unknown> {
  if (!map) {
    return {};
  }

  const result: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(map)) {
    result[key] = jsonValueToPlain(value);
  }
  return result;
}
