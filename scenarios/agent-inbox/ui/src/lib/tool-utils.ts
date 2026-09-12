/**
 * Utilities for parsing and displaying tool call data.
 *
 * Tool calls may include _context_attachments which contain skills and other
 * context injected by the system. This module extracts and formats that data
 * for display in the UI.
 */

/**
 * Skill attachment extracted from _context_attachments.
 * These are knowledge modules injected into the tool call context.
 */
export interface SkillAttachment {
  type: "skill";
  key: string;
  label: string;
  content: string;
  tags?: string[];
}

/**
 * Generic context attachment that may be included in tool calls.
 * Skills are the primary type, but the system supports other types.
 */
export interface ContextAttachment {
  type: string;
  key: string;
  label: string;
  content: string;
  tags?: string[];
}

/**
 * Parsed tool input separating arguments from context attachments.
 */
export interface ParsedToolInput {
  /** Tool arguments without _context_attachments */
  arguments: Record<string, unknown>;
  /** Skills extracted from _context_attachments */
  skills: SkillAttachment[];
  /** All context attachments (including non-skill types) */
  allAttachments: ContextAttachment[];
  /** Raw JSON string for display */
  rawArguments: string;
  /** Formatted JSON without _context_attachments for cleaner display */
  cleanedArgumentsJson: string;
}

/**
 * Parse tool input JSON and extract context attachments.
 *
 * @param argsJson - Raw JSON string from tool call arguments
 * @returns Parsed input with separated arguments and skill attachments
 */
export function parseToolInput(argsJson: string | undefined | null): ParsedToolInput {
  const emptyResult: ParsedToolInput = {
    arguments: {},
    skills: [],
    allAttachments: [],
    rawArguments: argsJson || "{}",
    cleanedArgumentsJson: "{}",
  };

  if (!argsJson) {
    return emptyResult;
  }

  try {
    const parsed: unknown = JSON.parse(argsJson);

    if (typeof parsed !== "object" || parsed === null) {
      return {
        ...emptyResult,
        arguments: { value: parsed },
        cleanedArgumentsJson: JSON.stringify({ value: parsed }, null, 2),
      };
    }

    const { _context_attachments, ...restArgs } = parsed as {
      _context_attachments?: ContextAttachment[];
      [key: string]: unknown;
    };

    const allAttachments: ContextAttachment[] = Array.isArray(_context_attachments)
      ? _context_attachments
      : [];

    const skills: SkillAttachment[] = allAttachments.filter(
      (att): att is SkillAttachment => att.type === "skill"
    );

    return {
      arguments: restArgs,
      skills,
      allAttachments,
      rawArguments: argsJson,
      cleanedArgumentsJson: JSON.stringify(restArgs, null, 2),
    };
  } catch {
    return {
      ...emptyResult,
      cleanedArgumentsJson: argsJson,
    };
  }
}

/**
 * Format tool result for display.
 * Handles both string and object results.
 */
export function formatToolResult(result: unknown): string {
  if (result === null || result === undefined) {
    return "";
  }

  if (typeof result === "string") {
    try {
      const parsed: unknown = JSON.parse(result);
      return JSON.stringify(parsed, null, 2);
    } catch {
      return result;
    }
  }

  if (typeof result === "object") {
    return JSON.stringify(result, null, 2);
  }

  if (typeof result === "number" || typeof result === "boolean" || typeof result === "bigint") {
    return String(result);
  }

  return JSON.stringify(result);
}

/**
 * Truncate text to a maximum length with ellipsis.
 */
export function truncateText(text: string, maxLength: number = 100): string {
  if (text.length <= maxLength) {
    return text;
  }
  return text.slice(0, maxLength - 3) + "...";
}

/**
 * Get a brief summary of a tool result for inline display.
 */
export function getResultSummary(result: unknown, maxLength: number = 50): string | null {
  if (result === null || result === undefined) {
    return null;
  }

  if (typeof result === "string") {
    return truncateText(result, maxLength);
  }

  if (typeof result === "object") {
    const obj = result as Record<string, unknown>;

    if (typeof obj.message === "string") {
      return truncateText(obj.message, maxLength);
    }
    if (typeof obj.summary === "string") {
      return truncateText(obj.summary, maxLength);
    }
    if (typeof obj.status === "string") {
      return truncateText(obj.status, maxLength);
    }

    if (Array.isArray(obj.files)) {
      return `${obj.files.length} file${obj.files.length !== 1 ? "s" : ""}`;
    }
    if (typeof obj.files_created === "number") {
      return `${obj.files_created} file${obj.files_created !== 1 ? "s" : ""} created`;
    }
    if (Array.isArray(obj.results)) {
      return `${obj.results.length} result${obj.results.length !== 1 ? "s" : ""}`;
    }
  }

  return "Result available";
}

/**
 * Check if a tool status indicates completion (success or failure).
 */
export function isTerminalStatus(
  status: string
): status is "completed" | "failed" | "rejected" | "cancelled" {
  return ["completed", "failed", "rejected", "cancelled", "error", "timeout"].includes(status);
}

/**
 * Check if a tool status indicates failure.
 */
export function isFailedStatus(
  status: string
): status is "failed" | "rejected" | "cancelled" | "error" | "timeout" {
  return ["failed", "rejected", "cancelled", "error", "timeout"].includes(status);
}
