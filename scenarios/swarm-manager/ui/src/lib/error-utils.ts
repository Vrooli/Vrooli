/**
 * Error Utilities - Observability and Diagnosis Helpers
 *
 * This module provides structured error logging and diagnosis utilities
 * that make errors observable without exposing sensitive information.
 *
 * DOC: docs/internal/ERROR-SEMANTICS.md
 *
 * ╔════════════════════════════════════════════════════════════════╗
 * ║  ERROR CATEGORIES - Read before modifying                      ║
 * ║                                                                ║
 * ║  Each category has a specific recovery path. Changing these    ║
 * ║  affects UI error states and automated retry logic.            ║
 * ║                                                                ║
 * ║  Categories:                                                   ║
 * ║  - NETWORK: Connection failures → Retry with backoff           ║
 * ║  - TIMEOUT: Request timed out → Retry with backoff             ║
 * ║  - AUTH: Session expired/forbidden → Re-authenticate           ║
 * ║  - NOT_FOUND: Resource missing → Navigate away, don't retry    ║
 * ║  - SERVER: Server error (5xx) → Retry with backoff             ║
 * ║  - VALIDATION: Bad input → Fix input, don't retry              ║
 * ║  - PARSE: Invalid response → Report bug, don't retry           ║
 * ║  - RUNTIME: Unexpected error → Refresh page                    ║
 * ╚════════════════════════════════════════════════════════════════╝
 *
 * Key principles:
 * - Structured logging for machine parsing
 * - No sensitive data in logs (URLs sanitized, no tokens)
 * - Correlation IDs for tracing errors across systems
 */

import { isApiError, type ApiError } from "./api-client";

/**
 * Error categories that map to distinct recovery paths.
 * See the comment block above for recovery path documentation.
 */
export type ErrorCategory =
  | "NETWORK"    // Network/connectivity issues → retry
  | "TIMEOUT"    // Request timed out → retry
  | "AUTH"       // Authentication/authorization → re-auth
  | "NOT_FOUND"  // Resource doesn't exist → navigate away
  | "SERVER"     // Server-side error → retry later
  | "VALIDATION" // Client-side input error → fix and resubmit
  | "PARSE"      // Response parsing failed → report bug
  | "RUNTIME";   // Unexpected runtime error → refresh

/**
 * Structured error log entry for observability.
 * Machine-parseable format for log aggregation and alerting.
 */
export interface ErrorLogEntry {
  /** ISO timestamp */
  timestamp: string;
  /** High-level error category */
  category: ErrorCategory;
  /** Human-readable summary */
  message: string;
  /** Correlation ID for tracing */
  correlationId: string;
  /** HTTP status if applicable */
  status?: number;
  /** Whether the error is recoverable by retry */
  retryable: boolean;
  /** Source component or module */
  source: string;
  /** Additional context (no sensitive data) */
  context?: Record<string, string | number | boolean>;
}

/**
 * Maps ApiErrorType to ErrorCategory.
 */
function mapApiErrorToCategory(error: ApiError): ErrorCategory {
  switch (error.type) {
    case "network":
      return "NETWORK";
    case "timeout":
      return "TIMEOUT";
    case "http":
      if (error.status === 401 || error.status === 403) return "AUTH";
      if (error.status === 404) return "NOT_FOUND";
      if (error.status === 400 || error.status === 422) return "VALIDATION";
      if (error.isServerError) return "SERVER";
      return "VALIDATION"; // Default for other 4xx
    case "parse":
      return "PARSE";
    default:
      return "RUNTIME";
  }
}

/**
 * Maps any error to an ErrorCategory.
 */
export function categorizeError(error: unknown): ErrorCategory {
  if (isApiError(error)) {
    return mapApiErrorToCategory(error);
  }

  if (error instanceof Error) {
    const msg = error.message.toLowerCase();
    if (msg.includes("network") || msg.includes("fetch")) return "NETWORK";
    if (msg.includes("timeout") || msg.includes("abort")) return "TIMEOUT";
  }

  return "RUNTIME";
}

/**
 * Generates a unique ID with the given prefix.
 * Format: {prefix}_{timestamp_base36}_{random_6chars}
 *
 * This is a pure utility for generating correlation IDs, error IDs, etc.
 * Not cryptographically secure - suitable for logging and debugging correlation.
 *
 * @param prefix - The prefix for the ID (e.g., "err", "page_err", "trace")
 * @returns A unique ID string
 *
 * @example
 * generateUniqueId("err")       // "err_m1abc23_f7gh9j"
 * generateUniqueId("page_err")  // "page_err_m1abc23_k3mn5p"
 * generateUniqueId("trace")     // "trace_m1abc23_r2st4v"
 */
export function generateUniqueId(prefix: string): string {
  return `${prefix}_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 8)}`;
}

/**
 * Generates a correlation ID for error tracing.
 * Format: err_{timestamp}_{random}
 *
 * This is a convenience wrapper around generateUniqueId for API error logging.
 */
export function generateCorrelationId(): string {
  return generateUniqueId("err");
}

/**
 * Creates a structured error log entry.
 * Safe for logging - no sensitive data included.
 *
 * @param error - The error to log
 * @param source - Component or module name
 * @param context - Additional context (no sensitive data)
 * @returns Structured log entry
 */
export function createErrorLogEntry(
  error: unknown,
  source: string,
  context?: Record<string, string | number | boolean>
): ErrorLogEntry {
  const category = categorizeError(error);
  const correlationId = generateCorrelationId();

  let message = "An unexpected error occurred";
  let status: number | undefined;
  let retryable = false;

  if (isApiError(error)) {
    message = error.userMessage;
    status = error.status;
    retryable = error.isRetryable;
  } else if (error instanceof Error) {
    // Sanitize error message - don't include full technical details
    message = sanitizeErrorMessage(error.message);
    retryable = category === "NETWORK" || category === "TIMEOUT" || category === "SERVER";
  }

  return {
    timestamp: new Date().toISOString(),
    category,
    message,
    correlationId,
    status,
    retryable,
    source,
    context,
  };
}

/**
 * Sanitizes an error message to remove potentially sensitive information.
 * - Removes URLs (may contain tokens or internal paths)
 * - Removes file paths
 * - Truncates long messages
 */
function sanitizeErrorMessage(message: string): string {
  // Remove URLs
  let sanitized = message.replace(/https?:\/\/[^\s]+/gi, "[URL]");
  // Remove file paths (Unix and Windows)
  sanitized = sanitized.replace(/(?:\/[\w.-]+)+|(?:[A-Z]:\\[\w.-\\]+)/gi, "[PATH]");
  // Truncate if too long
  if (sanitized.length > 200) {
    sanitized = sanitized.slice(0, 197) + "...";
  }
  return sanitized;
}

/**
 * Logs an error with structured metadata.
 * In production, this could send to an external logging service.
 *
 * @param error - The error to log
 * @param source - Component or module name
 * @param context - Additional context
 */
export function logError(
  error: unknown,
  source: string,
  context?: Record<string, string | number | boolean>
): ErrorLogEntry {
  const entry = createErrorLogEntry(error, source, context);

  // Log as structured JSON for machine parsing
  console.error(`[${entry.category}]`, JSON.stringify(entry));

  return entry;
}

/**
 * Recovery path guidance for each error category.
 * Used by UI components to show appropriate recovery actions.
 */
export const RECOVERY_PATHS: Record<ErrorCategory, {
  action: string;
  buttonLabel?: string;
  canRetry: boolean;
}> = {
  NETWORK: {
    action: "Check your internet connection and try again",
    buttonLabel: "Try Again",
    canRetry: true,
  },
  TIMEOUT: {
    action: "The server is taking too long - try again in a moment",
    buttonLabel: "Try Again",
    canRetry: true,
  },
  AUTH: {
    action: "Your session has expired - please refresh the page",
    buttonLabel: "Refresh",
    canRetry: false,
  },
  NOT_FOUND: {
    action: "This resource no longer exists",
    buttonLabel: "Go Back",
    canRetry: false,
  },
  SERVER: {
    action: "The server encountered an error - try again later",
    buttonLabel: "Try Again",
    canRetry: true,
  },
  VALIDATION: {
    action: "Please check your input and try again",
    buttonLabel: undefined, // No button - user needs to fix input
    canRetry: false,
  },
  PARSE: {
    action: "Received an invalid response from the server",
    buttonLabel: "Report Issue",
    canRetry: false,
  },
  RUNTIME: {
    action: "Something went wrong - please refresh the page",
    buttonLabel: "Refresh Page",
    canRetry: false,
  },
};

/**
 * Gets recovery guidance for an error.
 */
export function getRecoveryGuidance(error: unknown) {
  const category = categorizeError(error);
  return {
    category,
    ...RECOVERY_PATHS[category],
  };
}

// ============================================================================
// Success Logging (Observability for happy paths)
// ============================================================================

/**
 * Operation outcome types for success logging.
 */
export type OperationOutcome = "created" | "updated" | "deleted" | "fetched" | "completed";

/**
 * Structured success log entry for observability.
 * Mirrors ErrorLogEntry but for successful operations.
 */
export interface SuccessLogEntry {
  /** ISO timestamp */
  timestamp: string;
  /** Type of operation completed */
  outcome: OperationOutcome;
  /** Human-readable summary */
  message: string;
  /** Correlation ID for tracing */
  correlationId: string;
  /** Source component or module */
  source: string;
  /** Additional context (no sensitive data) */
  context?: Record<string, string | number | boolean>;
}

/**
 * Creates a structured success log entry.
 * Safe for logging - no sensitive data included.
 *
 * @param outcome - The type of operation completed
 * @param message - Human-readable description
 * @param source - Component or module name
 * @param context - Additional context (no sensitive data)
 * @returns Structured log entry
 */
export function createSuccessLogEntry(
  outcome: OperationOutcome,
  message: string,
  source: string,
  context?: Record<string, string | number | boolean>
): SuccessLogEntry {
  return {
    timestamp: new Date().toISOString(),
    outcome,
    message,
    correlationId: generateUniqueId("op"),
    source,
    context,
  };
}

/**
 * Logs a successful operation with structured metadata.
 * Useful for observability and debugging happy paths.
 *
 * @param outcome - The type of operation completed
 * @param message - Human-readable description
 * @param source - Component or module name
 * @param context - Additional context
 */
export function logSuccess(
  outcome: OperationOutcome,
  message: string,
  source: string,
  context?: Record<string, string | number | boolean>
): SuccessLogEntry {
  const entry = createSuccessLogEntry(outcome, message, source, context);

  // Log as structured JSON for machine parsing (use info level via console.info)
  console.info(`[${entry.outcome.toUpperCase()}]`, JSON.stringify(entry));

  return entry;
}
