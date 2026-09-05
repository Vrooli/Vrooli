import { create } from '@bufbuild/protobuf';
import { resolveApiBase, buildApiUrl, createScenarioConnectTransport } from '@vrooli/api-base';
import {
  JsonValueSchema,
  JsonObjectSchema,
  JsonListSchema,
} from '@vrooli/proto-types/common/v1/types_pb';
import type {
  JsonValue as ProtoJsonValue,
  JsonObject as ProtoJsonObject,
  JsonList as ProtoJsonList,
} from '@vrooli/proto-types/common/v1/types_pb';

// Connect transport base (host root; the transport appends the RPC path).
export const API_BASE = resolveApiBase();
// REST base (…/api/v1) for the deliberate non-RPC exceptions: multipart asset
// upload and static /uploads asset serving.
export const REST_API_BASE = resolveApiBase({ appendSuffix: true });

// Admin surfaces authenticate with an HMAC session cookie set by
// AdminAuthService.Login. Send credentials on every RPC so authenticated
// reads/writes carry the cookie, matching the browser's same-origin behavior.
const credentialedFetch: typeof fetch = (input, init) =>
  fetch(input, { ...init, credentials: 'include' });

export const transport = createScenarioConnectTransport({
  baseUrl: API_BASE,
  fetch: credentialedFetch,
});

/**
 * Typed error thrown by the REST-exception helpers (multipart asset upload,
 * static asset serving). Connect RPCs surface their own ConnectError; this
 * covers only the endpoints that cannot be RPCs.
 */
export class ApiError extends Error {
  readonly status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

export async function decodeApiError(res: Response): Promise<ApiError> {
  let message = res.statusText || `request failed with status ${res.status}`;
  try {
    const text = await res.text();
    if (text) {
      message = text;
    }
  } catch {
    // Keep the status-derived message when the body is unreadable.
  }
  return new ApiError(message, res.status);
}

/**
 * uploadFile posts multipart form data to a REST-exception endpoint. The
 * browser sets the multipart Content-Type (with boundary) from the FormData,
 * so no headers are passed. Credentials ride along for admin-only uploads.
 */
export async function uploadFile(path: string, formData: FormData): Promise<Response> {
  return fetch(buildApiUrl(path, { baseUrl: REST_API_BASE }), {
    method: 'POST',
    body: formData,
    credentials: 'include',
    cache: 'no-store',
  });
}

/**
 * jsonValueToJs unwraps a common.v1.JsonValue message into a plain JavaScript
 * value. Proto int64 collapses to number for display ergonomics; callers that
 * need full precision should read the proto field directly.
 */
export function jsonValueToJs(value: ProtoJsonValue | undefined): unknown {
  if (!value || value.kind.case === undefined) {
    return undefined;
  }
  switch (value.kind.case) {
    case 'boolValue':
    case 'doubleValue':
    case 'stringValue':
      return value.kind.value;
    case 'intValue':
      return Number(value.kind.value);
    case 'nullValue':
      return null;
    case 'bytesValue':
      return value.kind.value;
    case 'objectValue':
      return jsonObjectToRecord(value.kind.value);
    case 'listValue':
      return jsonListToArray(value.kind.value);
    default:
      return undefined;
  }
}

function jsonObjectToRecord(obj: ProtoJsonObject | undefined): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  if (!obj) {
    return out;
  }
  for (const [key, val] of Object.entries(obj.fields)) {
    out[key] = jsonValueToJs(val);
  }
  return out;
}

function jsonListToArray(list: ProtoJsonList | undefined): unknown[] {
  if (!list) {
    return [];
  }
  return list.values.map((v) => jsonValueToJs(v));
}

/**
 * jsonMapToRecord converts a proto `map<string, JsonValue>` into a plain object
 * of unwrapped JS values — the shape UI components read metadata as.
 */
export function jsonMapToRecord(
  map: Record<string, ProtoJsonValue> | undefined,
): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  if (!map) {
    return out;
  }
  for (const [key, val] of Object.entries(map)) {
    out[key] = jsonValueToJs(val);
  }
  return out;
}

/**
 * jsToJsonValue builds a common.v1.JsonValue message from a plain JavaScript
 * value — the inverse of jsonValueToJs. Used when the UI needs to author a
 * proto `map<string, JsonValue>` (e.g. plan display metadata for demo plans).
 */
export function jsToJsonValue(value: unknown): ProtoJsonValue {
  if (value === null || value === undefined) {
    return create(JsonValueSchema, { kind: { case: 'nullValue', value: 0 } });
  }
  if (typeof value === 'boolean') {
    return create(JsonValueSchema, { kind: { case: 'boolValue', value } });
  }
  if (typeof value === 'number') {
    return create(JsonValueSchema, { kind: { case: 'doubleValue', value } });
  }
  if (typeof value === 'bigint') {
    return create(JsonValueSchema, { kind: { case: 'intValue', value } });
  }
  if (typeof value === 'string') {
    return create(JsonValueSchema, { kind: { case: 'stringValue', value } });
  }
  if (Array.isArray(value)) {
    const list = create(JsonListSchema, { values: value.map(jsToJsonValue) });
    return create(JsonValueSchema, { kind: { case: 'listValue', value: list } });
  }
  const fields: Record<string, ProtoJsonValue> = {};
  for (const [key, val] of Object.entries(value as Record<string, unknown>)) {
    fields[key] = jsToJsonValue(val);
  }
  const obj = create(JsonObjectSchema, { fields });
  return create(JsonValueSchema, { kind: { case: 'objectValue', value: obj } });
}

/** recordToJsonMap builds a proto `map<string, JsonValue>` from a plain object. */
export function recordToJsonMap(record: Record<string, unknown>): Record<string, ProtoJsonValue> {
  const out: Record<string, ProtoJsonValue> = {};
  for (const [key, val] of Object.entries(record)) {
    out[key] = jsToJsonValue(val);
  }
  return out;
}

export type { ProtoJsonValue };
