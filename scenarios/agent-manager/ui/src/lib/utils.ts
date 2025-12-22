import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";
import { resolveApiBase } from "@vrooli/api-base";
import { timestampDate, type Timestamp } from "@bufbuild/protobuf/wkt";
import { JsonObject, JsonValue, RunnerType } from "../types";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function getApiBaseUrl(): string {
  // Use api-base resolution which handles localhost/proxy scenarios correctly
  // The UI server proxies /api/* to the actual API server
  return resolveApiBase({ appendSuffix: true });
}

function toDateValue(date: string | Date | Timestamp | undefined): Date | undefined {
  if (!date) return undefined;
  if (date instanceof Date) return date;
  if (typeof date === "string") {
    const parsed = new Date(date);
    return Number.isNaN(parsed.getTime()) ? undefined : parsed;
  }
  if (typeof date === "object" && "seconds" in date) {
    return timestampDate(date as Timestamp);
  }
  return undefined;
}

export function formatDate(date: string | Date | Timestamp | undefined): string {
  const d = toDateValue(date);
  if (!d) return "N/A";
  return d.toLocaleString();
}

export function formatDuration(ms: number): string {
  if (ms < 1000) return ms + "ms";
  if (ms < 60000) return (ms / 1000).toFixed(1) + "s";
  if (ms < 3600000) return Math.floor(ms / 60000) + "m " + Math.floor((ms % 60000) / 1000) + "s";
  return Math.floor(ms / 3600000) + "h " + Math.floor((ms % 3600000) / 60000) + "m";
}

export function formatRelativeTime(date: string | Date | Timestamp | undefined): string {
  const d = toDateValue(date);
  if (!d) return "N/A";
  const now = new Date();
  const diffMs = now.getTime() - d.getTime();
  
  if (diffMs < 60000) return "just now";
  if (diffMs < 3600000) return Math.floor(diffMs / 60000) + "m ago";
  if (diffMs < 86400000) return Math.floor(diffMs / 3600000) + "h ago";
  return Math.floor(diffMs / 86400000) + "d ago";
}

export function runnerTypeToSlug(type?: RunnerType): string {
  switch (type) {
    case RunnerType.CLAUDE_CODE:
      return "claude-code";
    case RunnerType.CODEX:
      return "codex";
    case RunnerType.OPENCODE:
      return "opencode";
    default:
      return "claude-code";
  }
}

export function runnerTypeFromSlug(value?: string): RunnerType | undefined {
  switch (value) {
    case "claude-code":
      return RunnerType.CLAUDE_CODE;
    case "codex":
      return RunnerType.CODEX;
    case "opencode":
      return RunnerType.OPENCODE;
    default:
      return undefined;
  }
}

export function runnerTypeLabel(type?: RunnerType): string {
  switch (type) {
    case RunnerType.CLAUDE_CODE:
      return "Claude Code";
    case RunnerType.CODEX:
      return "Codex";
    case RunnerType.OPENCODE:
      return "OpenCode";
    default:
      return "Unknown";
  }
}

export function jsonValueToPlain(value?: JsonValue): unknown {
  if (!value) return undefined;
  const kind = value.kind;
  switch (kind.case) {
    case "boolValue":
    case "doubleValue":
    case "stringValue":
    case "bytesValue":
      return kind.value;
    case "intValue":
      return Number(kind.value);
    case "nullValue":
      return null;
    case "objectValue":
      return jsonObjectToPlain(kind.value);
    case "listValue":
      return kind.value.values.map(jsonValueToPlain);
    default:
      return undefined;
  }
}

export function jsonObjectToPlain(value?: JsonObject): Record<string, unknown> | undefined {
  if (!value?.fields) return undefined;
  const result: Record<string, unknown> = {};
  for (const [key, field] of Object.entries(value.fields)) {
    result[key] = jsonValueToPlain(field);
  }
  return result;
}

export function truncate(str: string, maxLen: number): string {
  if (str.length <= maxLen) return str;
  return str.slice(0, maxLen - 3) + "...";
}
