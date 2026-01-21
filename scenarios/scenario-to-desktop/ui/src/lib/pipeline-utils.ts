/**
 * Shared utility functions for pipeline status mapping and log parsing.
 */

/** UI-friendly build status */
export type MappedBuildStatus = "building" | "ready" | "partial" | "failed";

/**
 * Maps raw pipeline status strings to UI-friendly build status.
 * Used by components that poll pipeline status and display results.
 */
export function mapPipelineStatus(status: string): MappedBuildStatus {
  switch (status) {
    case "pending":
    case "running":
      return "building";
    case "completed":
      return "ready";
    case "failed":
    case "cancelled":
      return "failed";
    default:
      return "building";
  }
}

// ============================================================================
// Structured Log Parsing
// ============================================================================

/** Log severity levels matching backend LogLevel */
export type LogLevel = "INFO" | "WARN" | "ERROR" | "DEBUG";

/** Parsed structured log entry */
export interface ParsedLogEntry {
  /** ISO timestamp string */
  timestamp: string;
  /** Severity level */
  level: LogLevel;
  /** Log message content */
  message: string;
  /** Original raw log string */
  raw: string;
}

/**
 * Regex pattern for structured log entries.
 * Format: "[TIMESTAMP] [LEVEL] message"
 * Example: "[2024-01-15T10:30:00Z] [INFO] Stage bundle starting"
 */
const STRUCTURED_LOG_PATTERN = /^\[(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z?)\]\s*\[(INFO|WARN|ERROR|DEBUG)\]\s*(.*)$/;

/**
 * Parses a raw log string into a structured log entry.
 * If the log doesn't match the structured format, returns a fallback entry.
 */
export function parseLogEntry(raw: string): ParsedLogEntry {
  const match = raw.match(STRUCTURED_LOG_PATTERN);
  if (match) {
    return {
      timestamp: match[1],
      level: match[2] as LogLevel,
      message: match[3],
      raw,
    };
  }
  // Fallback for unstructured logs (legacy format)
  return {
    timestamp: new Date().toISOString(),
    level: "INFO",
    message: raw,
    raw,
  };
}

/**
 * Parses an array of raw log strings into structured entries.
 */
export function parseLogs(logs: string[]): ParsedLogEntry[] {
  return logs.map(parseLogEntry);
}

/**
 * Filters parsed logs by severity level.
 * @param logs Parsed log entries
 * @param minLevel Minimum severity to include (DEBUG < INFO < WARN < ERROR)
 */
export function filterLogsByLevel(logs: ParsedLogEntry[], minLevel: LogLevel): ParsedLogEntry[] {
  const levelOrder: Record<LogLevel, number> = {
    DEBUG: 0,
    INFO: 1,
    WARN: 2,
    ERROR: 3,
  };
  const minOrder = levelOrder[minLevel];
  return logs.filter((log) => levelOrder[log.level] >= minOrder);
}

/**
 * Returns styling hints for a log level (for UI display).
 */
export function getLogLevelStyle(level: LogLevel): { color: string; bg: string } {
  switch (level) {
    case "ERROR":
      return { color: "text-red-400", bg: "bg-red-950/30" };
    case "WARN":
      return { color: "text-yellow-400", bg: "bg-yellow-950/30" };
    case "INFO":
      return { color: "text-slate-300", bg: "" };
    case "DEBUG":
      return { color: "text-slate-500", bg: "" };
    default:
      return { color: "text-slate-300", bg: "" };
  }
}

/**
 * Formats a timestamp for display (relative time if recent, absolute if older).
 */
export function formatLogTimestamp(timestamp: string): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  const diffMin = Math.floor(diffSec / 60);

  if (diffSec < 60) {
    return `${diffSec}s ago`;
  }
  if (diffMin < 60) {
    return `${diffMin}m ago`;
  }
  // For older entries, show time only
  return date.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit", second: "2-digit" });
}

/**
 * Extracts the most recent significant event from logs.
 * Prioritizes ERROR > WARN > INFO for summary display.
 */
export function getLatestSignificantLog(logs: ParsedLogEntry[]): ParsedLogEntry | null {
  if (logs.length === 0) return null;

  // Look for errors first
  const errors = logs.filter((l) => l.level === "ERROR");
  if (errors.length > 0) return errors[errors.length - 1];

  // Then warnings
  const warnings = logs.filter((l) => l.level === "WARN");
  if (warnings.length > 0) return warnings[warnings.length - 1];

  // Otherwise return the most recent entry
  return logs[logs.length - 1];
}

// ============================================================================
// Idempotency Key Generation
// ============================================================================

/**
 * Generates a stable idempotency key for pipeline requests.
 * The key is based on:
 * - Scenario name (required)
 * - Target stage (if stop_after_stage is set)
 * - A timestamp-based session ID (prevents stale keys from old sessions)
 *
 * This ensures that:
 * 1. Identical requests within the same session return the same key
 * 2. Rapid double-clicks don't create duplicate pipelines
 * 3. New sessions get fresh keys to allow intentional retries
 *
 * @param scenarioName The scenario being deployed
 * @param stage Optional target stage (for stop_after_stage)
 * @param sessionId Optional session identifier (defaults to page load time)
 * @returns A deterministic idempotency key
 */
export function generateIdempotencyKey(
  scenarioName: string,
  stage?: string,
  sessionId?: string
): string {
  // Use the provided session ID or generate one from page load time
  const session = sessionId ?? getSessionId();
  const parts = [scenarioName, session];
  if (stage) {
    parts.push(stage);
  }
  // Create a simple hash from the parts
  return parts.join("-");
}

// Session ID is generated once per page load to scope idempotency keys
let _sessionId: string | null = null;

/**
 * Gets or creates a session-scoped identifier.
 * This ensures idempotency keys are unique per page session but stable within it.
 */
export function getSessionId(): string {
  if (!_sessionId) {
    // Use a combination of timestamp and random value for uniqueness
    const timestamp = Date.now().toString(36);
    const random = Math.random().toString(36).substring(2, 8);
    _sessionId = `${timestamp}-${random}`;
  }
  return _sessionId;
}

/**
 * Resets the session ID, forcing new idempotency keys for future requests.
 * Useful when the user explicitly wants to retry a failed operation.
 */
export function resetSessionId(): void {
  _sessionId = null;
}

/**
 * Generates a unique idempotency key that will never match previous keys.
 * Use this when you explicitly want to create a new pipeline, not reuse existing.
 */
export function generateUniqueIdempotencyKey(scenarioName: string, stage?: string): string {
  const uniqueId = `${Date.now().toString(36)}-${Math.random().toString(36).substring(2, 8)}`;
  return generateIdempotencyKey(scenarioName, stage, uniqueId);
}
