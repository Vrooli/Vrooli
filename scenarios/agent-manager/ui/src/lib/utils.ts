import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";
import { resolveAgentManagerApiBase } from "./api";
import { JsonObject, JsonValue, NetworkAccess, RunnerType, SandboxMode } from "../types";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function getApiBaseUrl(): string {
  // Use api-base resolution which handles localhost/proxy scenarios correctly
  // The UI server proxies /api/* to the actual API server
  return resolveAgentManagerApiBase(true);
}

export function formatDuration(ms: number): string {
  if (ms < 1000) return ms + "ms";
  if (ms < 60000) return (ms / 1000).toFixed(1) + "s";
  if (ms < 3600000) return Math.floor(ms / 60000) + "m " + Math.floor((ms % 60000) / 1000) + "s";
  return Math.floor(ms / 3600000) + "h " + Math.floor((ms % 3600000) / 60000) + "m";
}

export function runnerTypeToSlug(type?: RunnerType): string {
  switch (type) {
    case RunnerType.CLAUDE_CODE:
      return "claude-code";
    case RunnerType.CODEX:
      return "codex";
    case RunnerType.OPENCODE:
      return "opencode";
    case RunnerType.GROK:
      return "grok";
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
    case "grok":
      return RunnerType.GROK;
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
    case RunnerType.GROK:
      return "Grok";
    default:
      return "Unknown";
  }
}

export function networkAccessLabel(na?: NetworkAccess): string {
  switch (na) {
    case NetworkAccess.NONE:
      return "None";
    case NetworkAccess.LOCALHOST:
      return "Localhost";
    case NetworkAccess.FULL:
      return "Full";
    default:
      return "Localhost";
  }
}

// sandboxModeLabel renders the per-run SandboxMode for the UI.
// Replaces the older "Sandbox Required" boolean badge — see
// scenarios/agent-manager/docs/internal/SEAMS.md (RunMode decision boundary).
export function sandboxModeLabel(mode?: SandboxMode): string {
  switch (mode) {
    case SandboxMode.OFF:
      return "Off";
    case SandboxMode.TRACKING:
      return "Tracking";
    case SandboxMode.PROTECTED:
      return "Protected";
    default:
      return "Default";
  }
}

// profileSandboxModeFormValue extracts the sandbox-mode form-string
// ("off"/"tracking"/"protected") from a profile. Falls back to
// "protected" for profiles with no SandboxConfig — matches the
// agent-manager DefaultSandboxConfig.
export function profileSandboxModeFormValue(profile: {
  sandboxConfig?: { mode?: SandboxMode };
}): "off" | "tracking" | "protected" {
  switch (profile.sandboxConfig?.mode) {
    case SandboxMode.OFF:
      return "off";
    case SandboxMode.TRACKING:
      return "tracking";
    case SandboxMode.PROTECTED:
      return "protected";
    default:
      return "protected";
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
