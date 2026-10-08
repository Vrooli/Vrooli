// Responsibility: provide shared HTTP, protobuf and form conversion helpers.
import { useState } from "react";

import { fromJson, toJson, type DescMessage, type MessageShape, type JsonValue } from "@bufbuild/protobuf";

import { durationFromMs } from "@bufbuild/protobuf/wkt";

import { getApiBaseUrl, jsonObjectToPlain } from "../lib/utils";

import { ErrorResponseSchema } from "@vrooli/proto-types/common/v1/types_pb";

import { NetworkAccess, SandboxMode } from "@vrooli/proto-types/agent-manager/v1/domain/types_pb";

// sandboxModeFromForm parses the UI form-string to the proto enum.
// Empty/unknown maps to UNSPECIFIED so agent-manager applies its
// DefaultSandboxConfig.
export function sandboxModeFromForm(s?: "off" | "tracking" | "protected"): SandboxMode {
  switch (s) {
    case "off":
      return SandboxMode.OFF;
    case "tracking":
      return SandboxMode.TRACKING;
    case "protected":
      return SandboxMode.PROTECTED;
    default:
      return SandboxMode.UNSPECIFIED;
  }
}

export function networkAccessToProto(na: "none" | "localhost" | "full"): NetworkAccess {
  switch (na) {
    case "none":
      return NetworkAccess.NONE;
    case "localhost":
      return NetworkAccess.LOCALHOST;
    case "full":
      return NetworkAccess.FULL;
    default:
      return NetworkAccess.LOCALHOST;
  }
}

interface ApiState<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
}

export function useApiState<T>(initialData: T | null = null): ApiState<T> & {
  setData: (data: T | null) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
} {
  const [data, setData] = useState<T | null>(initialData);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  return { data, loading, error, setData, setLoading, setError };
}

export const protoReadOptions = { ignoreUnknownFields: true, protoFieldName: true };

const protoWriteOptions = { useProtoFieldName: true };

const jsonValueKeys = [
  { snake: "bool_value", camel: "boolValue" },
  { snake: "int_value", camel: "intValue" },
  { snake: "double_value", camel: "doubleValue" },
  { snake: "string_value", camel: "stringValue" },
  { snake: "object_value", camel: "objectValue" },
  { snake: "list_value", camel: "listValue" },
  { snake: "null_value", camel: "nullValue" },
  { snake: "bytes_value", camel: "bytesValue" },
];

export function parseProto<Desc extends DescMessage>(schema: Desc, raw: unknown): MessageShape<Desc> {
  return fromJson(schema, raw as JsonValue, protoReadOptions);
}

export function toProtoJson<Desc extends DescMessage>(schema: Desc, message: MessageShape<Desc>): Record<string, unknown> {
  return toJson(schema, message, protoWriteOptions) as Record<string, unknown>;
}

export function normalizeJsonValueInput(value: unknown): Record<string, unknown> {
  if (value === null) {
    return { null_value: "NULL_VALUE" };
  }
  if (typeof value === "boolean") {
    return { bool_value: value };
  }
  if (typeof value === "number") {
    if (Number.isInteger(value)) {
      return { int_value: value };
    }
    return { double_value: value };
  }
  if (typeof value === "string") {
    return { string_value: value };
  }
  if (Array.isArray(value)) {
    return { list_value: { values: value.map(normalizeJsonValueInput) } };
  }
  if (typeof value === "object" && value !== null) {
    const obj = value as Record<string, unknown>;
    for (const key of jsonValueKeys) {
      if (key.snake in obj || key.camel in obj) {
        const raw = (key.snake in obj ? obj[key.snake] : obj[key.camel]) as unknown;
        if (key.snake === "object_value") {
          const rawObj = raw as Record<string, unknown> | undefined;
          const rawFields = rawObj && typeof rawObj === "object" ? (rawObj.fields as Record<string, unknown> | undefined) : undefined;
          const fieldsSource = rawFields ?? (rawObj && !Array.isArray(rawObj) ? rawObj : {});
          const fields: Record<string, unknown> = {};
          for (const [fieldKey, fieldValue] of Object.entries(fieldsSource ?? {})) {
            fields[fieldKey] = normalizeJsonValueInput(fieldValue);
          }
          return { object_value: { fields } };
        }
        if (key.snake === "list_value") {
          const rawList = Array.isArray(raw) ? raw : (raw as Record<string, unknown>)?.values;
          const values = Array.isArray(rawList) ? rawList.map(normalizeJsonValueInput) : [];
          return { list_value: { values } };
        }
        if (key.snake === "null_value") {
          return { null_value: "NULL_VALUE" };
        }
        return { [key.snake]: raw };
      }
    }

    const fields: Record<string, unknown> = {};
    for (const [fieldKey, fieldValue] of Object.entries(obj)) {
      fields[fieldKey] = normalizeJsonValueInput(fieldValue);
    }
    return { object_value: { fields } };
  }

  return { string_value: String(value) };
}

function normalizeJsonValueMap(value: unknown): Record<string, unknown> | undefined {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return undefined;
  }
  const normalized: Record<string, unknown> = {};
  for (const [key, entry] of Object.entries(value as Record<string, unknown>)) {
    normalized[key] = normalizeJsonValueInput(entry);
  }
  return normalized;
}

export function normalizeHealthResponseJson(raw: unknown): unknown {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    return raw;
  }
  const obj = raw as Record<string, unknown>;
  const normalized: Record<string, unknown> = { ...obj };
  const dependencies = normalizeJsonValueMap(obj.dependencies);
  const metrics = normalizeJsonValueMap(obj.metrics);
  if (dependencies) {
    normalized.dependencies = dependencies;
  }
  if (metrics) {
    normalized.metrics = metrics;
  }
  return normalized;
}

function extractErrorMessage(raw: unknown, fallback: string): string {
  try {
    const parsed = parseProto(ErrorResponseSchema, raw);
    const details = jsonObjectToPlain(parsed.details);
    const userMessage = details?.user_message;
    if (typeof userMessage === "string" && userMessage.trim() !== "") {
      return userMessage;
    }
    if (parsed.message) {
      return parsed.message;
    }
  } catch {
    // ignore
  }
  return fallback;
}

export async function apiRequest<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const baseUrl = getApiBaseUrl();
  const url = endpoint.startsWith("http") ? endpoint : baseUrl + endpoint;

  const response = await fetch(url, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options.headers,
    },
  });

  if (!response.ok) {
    const errorData: unknown = await response.json().catch(() => ({}));
    throw new Error(extractErrorMessage(errorData, "Request failed: " + response.status));
  }

  if (response.status === 204) {
    return {} as T;
  }

  const json: unknown = await response.json();
  return json as T;
}

export function durationFromMinutes(minutes?: number) {
  if (typeof minutes !== "number" || Number.isNaN(minutes) || minutes <= 0) {
    return undefined;
  }
  return durationFromMs(minutes * 60_000);
}
