/**
 * Telemetry Domain Logic
 *
 * Pure functions for telemetry-related operations.
 * This module handles JSONL file parsing, telemetry path generation,
 * and event validation. No side effects - all testable in isolation.
 */

import type { OperatingSystem, TelemetryEvent, TelemetryFilePath } from "./types";

// Re-export types for convenience
export type { OperatingSystem, TelemetryEvent, TelemetryFilePath };

// ============================================================================
// Types
// ============================================================================

export interface TelemetryParseResult {
  success: boolean;
  events: TelemetryEvent[];
  errors: TelemetryParseError[];
}

export interface TelemetryParseError {
  lineNumber: number;
  line: string;
  error: string;
}

export interface TelemetryValidationResult {
  valid: boolean;
  errors: string[];
  warnings: string[];
  eventCount: number;
}

// ============================================================================
// Constants
// ============================================================================

/**
 * Default telemetry file name used by desktop wrappers.
 */
export const TELEMETRY_FILE_NAME = "deployment-telemetry.jsonl";

/**
 * Standard telemetry event types recorded by the runtime.
 */
export const STANDARD_EVENTS = [
  "runtime_start",
  "runtime_shutdown",
  "runtime_error",
  "app_session_succeeded",
  "app_session_failed",
  "service_start",
  "service_ready",
  "service_not_ready",
  "service_exit",
  "service_blocked",
  "gpu_status",
  "secrets_missing",
  "secrets_updated",
  "migration_start",
  "migration_applied",
  "migration_failed",
  "asset_missing",
  "asset_checksum_mismatch",
  "asset_size_exceeded",
  "api_unreachable",
  "smoke_test_started",
  "smoke_test_passed",
  "smoke_test_failed",
  "runtime_telemetry_uploaded"
] as const;

export type StandardEventType = (typeof STANDARD_EVENTS)[number];

// ============================================================================
// Path Generation
// ============================================================================

/**
 * Generate platform-specific telemetry file paths.
 * The app name is used as the subdirectory within platform-specific app data locations.
 */
export function generateTelemetryPaths(appName: string): TelemetryFilePath[] {
  if (!appName) {
    throw new Error("appName is required to generate telemetry paths");
  }

  return [
    {
      os: "Windows",
      path: `%APPDATA%/${appName}/${TELEMETRY_FILE_NAME}`
    },
    {
      os: "macOS",
      path: `~/Library/Application Support/${appName}/${TELEMETRY_FILE_NAME}`
    },
    {
      os: "Linux",
      path: `~/.config/${appName}/${TELEMETRY_FILE_NAME}`
    }
  ];
}

/**
 * Get telemetry path for a specific operating system.
 */
export function getTelemetryPathForOS(appName: string, os: OperatingSystem): string {
  const paths = generateTelemetryPaths(appName);
  const match = paths.find((p) => p.os === os);
  return match?.path ?? "";
}

// ============================================================================
// JSONL Parsing
// ============================================================================

/**
 * Parse a JSONL (JSON Lines) string into telemetry events.
 * Each line is expected to be a valid JSON object.
 * Returns both successful parses and errors for each failed line.
 */
export function parseJsonlContent(content: string): TelemetryParseResult {
  const events: TelemetryEvent[] = [];
  const errors: TelemetryParseError[] = [];

  // Split and filter empty lines
  const lines = content
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean);

  if (lines.length === 0) {
    return {
      success: false,
      events: [],
      errors: [{ lineNumber: 0, line: "", error: "File is empty" }]
    };
  }

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const lineNumber = i + 1;

    try {
      const parsed = JSON.parse(line);
      if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
        errors.push({
          lineNumber,
          line: truncateLine(line),
          error: "Expected a JSON object"
        });
        continue;
      }
      events.push(parsed as TelemetryEvent);
    } catch (e) {
      errors.push({
        lineNumber,
        line: truncateLine(line),
        error: e instanceof Error ? e.message : "Invalid JSON"
      });
    }
  }

  return {
    success: errors.length === 0,
    events,
    errors
  };
}

/**
 * Truncate a line for error display.
 */
function truncateLine(line: string, maxLength = 50): string {
  if (line.length <= maxLength) {
    return line;
  }
  return line.slice(0, maxLength - 3) + "...";
}

// ============================================================================
// Event Validation
// ============================================================================

/**
 * Validate parsed telemetry events.
 * Checks for structural correctness and provides warnings for non-standard events.
 */
export function validateTelemetryEvents(events: TelemetryEvent[]): TelemetryValidationResult {
  const errors: string[] = [];
  const warnings: string[] = [];

  if (events.length === 0) {
    return {
      valid: false,
      errors: ["No events to validate"],
      warnings: [],
      eventCount: 0
    };
  }

  const unknownEventTypes = new Set<string>();

  for (let i = 0; i < events.length; i++) {
    const event = events[i];
    const eventNum = i + 1;

    // Check for event type
    if (!event.event) {
      warnings.push(`Event ${eventNum}: Missing 'event' field`);
    } else if (!isStandardEvent(event.event)) {
      unknownEventTypes.add(event.event);
    }
  }

  // Aggregate warnings for unknown event types
  if (unknownEventTypes.size > 0) {
    const types = Array.from(unknownEventTypes).slice(0, 5).join(", ");
    const suffix = unknownEventTypes.size > 5 ? ` and ${unknownEventTypes.size - 5} more` : "";
    warnings.push(`Non-standard event types detected: ${types}${suffix}`);
  }

  return {
    valid: errors.length === 0,
    errors,
    warnings,
    eventCount: events.length
  };
}

/**
 * Check if an event type is a standard runtime event.
 */
export function isStandardEvent(eventType: string): eventType is StandardEventType {
  return STANDARD_EVENTS.includes(eventType as StandardEventType);
}

// ============================================================================
// File Processing (Pure - operates on content, not File objects)
// ============================================================================

/**
 * Process telemetry file content for upload.
 * Parses JSONL, validates events, and returns prepared payload or error.
 */
export function processTelemetryContent(content: string): {
  success: boolean;
  events?: TelemetryEvent[];
  error?: string;
  warnings?: string[];
} {
  // Parse JSONL
  const parseResult = parseJsonlContent(content);

  if (!parseResult.success && parseResult.events.length === 0) {
    const firstError = parseResult.errors[0];
    return {
      success: false,
      error: firstError ? `Line ${firstError.lineNumber}: ${firstError.error}` : "Failed to parse file"
    };
  }

  // Validate events
  const validation = validateTelemetryEvents(parseResult.events);

  if (!validation.valid) {
    return {
      success: false,
      error: validation.errors.join("; ")
    };
  }

  return {
    success: true,
    events: parseResult.events,
    warnings: validation.warnings.length > 0 ? validation.warnings : undefined
  };
}

// ============================================================================
// Event Formatting (for display)
// ============================================================================

/**
 * Format an event for display preview.
 */
export function formatEventPreview(event: TelemetryEvent): string {
  const parts: string[] = [];

  if (event.event) {
    parts.push(`event: "${event.event}"`);
  }
  if (event.level) {
    parts.push(`level: "${event.level}"`);
  }
  if (event.timestamp) {
    parts.push(`timestamp: "${event.timestamp}"`);
  }
  if (event.session_id) {
    parts.push(`session: "${event.session_id}"`);
  }
  if (event.detail) {
    parts.push(`detail: "${truncateLine(String(event.detail), 30)}"`);
  }

  return `{ ${parts.join(", ")} }`;
}

/**
 * Generate an example telemetry JSON line for documentation.
 */
export function generateExampleEvent(): string {
  return JSON.stringify({
    event: "api_unreachable",
    detail: "https://example.com",
    timestamp: new Date().toISOString()
  });
}
