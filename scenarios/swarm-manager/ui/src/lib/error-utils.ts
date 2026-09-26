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
 * ║  - STALE_CHUNK: Deploy replaced this tab's chunks → Reload     ║
 * ║  - RUNTIME: Unexpected error → Refresh page                    ║
 * ╚════════════════════════════════════════════════════════════════╝
 *
 * Key principles:
 * - Structured logging for machine parsing
 * - No sensitive data in logs (URLs sanitized, no tokens)
 * - Correlation IDs for tracing errors across systems
 */

import { isStaleChunkError } from "@vrooli/api-base";
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
  | "PARSE"       // Response parsing failed → report bug
  | "STALE_CHUNK" // Lazy chunk gone after a deploy → reload to update
  | "RUNTIME";    // Unexpected runtime error → refresh

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

  // Must run before the generic "fetch" → NETWORK match: a stale-chunk
  // failure ("Failed to fetch dynamically imported module") is a deploy
  // artifact whose recovery is a reload, not a retry.
  if (isStaleChunkError(error)) {
    return "STALE_CHUNK";
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

  // Same derivation the UI shows the operator. A log that says "The request
  // failed. Please try again." while the toast says "milestone has no
  // acceptance criteria" makes a bug report impossible to act on.
  const message = describeError(error).message;
  let status: number | undefined;
  let retryable = false;

  if (isApiError(error)) {
    status = error.status;
    retryable = error.isRetryable;
  } else if (error instanceof Error) {
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
 * Absolute POSIX path, anchored so it cannot start mid-token.
 *
 * The anchor is the whole point. The previous pattern was `(?:\/[\w.-]+)+`,
 * which matches any run of `/segment` *wherever it appears* — so the domain
 * refs this app is built on (`goal/release-1`, `execute/ship-workspace`) came
 * out as `goal[PATH]`, destroying the most actionable part of a message.
 *
 * Requiring a leading boundary (start of string, whitespace, or an opening
 * delimiter) plus at least two segments keeps `/home/op/.ssh/id_rsa` matched
 * and `kind/name` untouched. Written with an explicit captured prefix rather
 * than a lookbehind: this bundle ships to embedded WebViews old enough to
 * need the runtime polyfills in main.tsx, and lookbehind is not universal
 * there.
 */
const ABSOLUTE_POSIX_PATH = /(^|[\s"'`([{<,;:])(\/(?:[\w.-]+\/)+[\w.-]*)/g;

/**
 * Windows drive-letter path.
 *
 * The previous class was `[\w.-\\]`, in which `.-\` is read as a character
 * *range* spanning 0x2E–0x5C — quietly including digits, `/`, `:`, `?`, `@`
 * and every uppercase letter. Putting the dash last makes it a literal, which
 * confines the class to the three characters that were intended.
 */
const WINDOWS_PATH = /[A-Za-z]:\\(?:[\w.-]+\\)*[\w.-]*/g;

/** Longest sanitized message we keep; the tail of a stack adds nothing. */
const MAX_SANITIZED_LENGTH = 200;

/**
 * Sanitizes an error message to remove potentially sensitive information.
 * - Removes URLs (may contain tokens or internal paths)
 * - Removes absolute filesystem paths, which can leak a home directory layout
 * - Truncates long messages
 *
 * It deliberately does NOT touch relative `kind/name` entity refs. Those are
 * domain identifiers, not filesystem locations, and they carry no secret —
 * redacting them only makes the message useless.
 */
export function sanitizeErrorMessage(message: string): string {
  // URLs first: they contain slashes that would otherwise be partly matched
  // by the path patterns below.
  let sanitized = message.replace(/https?:\/\/[^\s]+/gi, "[URL]");
  sanitized = sanitized.replace(ABSOLUTE_POSIX_PATH, "$1[PATH]");
  sanitized = sanitized.replace(WINDOWS_PATH, "[PATH]");
  if (sanitized.length > MAX_SANITIZED_LENGTH) {
    sanitized = sanitized.slice(0, MAX_SANITIZED_LENGTH - 3) + "...";
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
  STALE_CHUNK: {
    action: "A new version was deployed while this tab was open - reload to update",
    buttonLabel: "Reload",
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
// User-Facing Error Description (the one derivation every surface shares)
// ============================================================================

/** Longest server message we will place in a toast before eliding. */
const MAX_DESCRIBED_MESSAGE = 240;

/**
 * A failed operation, described for a human, in one shape.
 *
 * Every async surface in the app derives its wording from this so a 404 reads
 * the same in a toast, a banner, and a dialog.
 */
export interface ErrorDescription {
  category: ErrorCategory;
  /** What went wrong, as one user-safe sentence. Never a stack or a URL. */
  message: string;
  /** What the operator can do about it. */
  recovery: string;
  /** Whether repeating the identical request could plausibly succeed. */
  canRetry: boolean;
  /** Server's machine-readable code ("plan_stale") when present, else "". */
  code: string;
  /** HTTP status when the failure reached the server, else undefined. */
  status?: number;
}

/**
 * True when a server-authored message is worth showing verbatim.
 *
 * The API's own 4xx bodies carry the actual reason ("milestone has no
 * acceptance criteria"), which is far more useful than a generic retry
 * prompt. But `normalizeErrorDetail` also synthesizes placeholder text for
 * empty and HTML bodies, and those say nothing — they must not reach a user.
 */
function isMeaningfulServerMessage(message: string): boolean {
  const trimmed = message.trim();
  if (!trimmed) return false;
  if (/^Request failed with status \d+$/i.test(trimmed)) return false;
  if (/^The server returned an HTML error page/i.test(trimmed)) return false;
  return true;
}

/**
 * Derives the single user-facing description of any thrown value.
 *
 * Precedence is deliberate:
 *  1. A meaningful server message on a *client* error (4xx). These are our own
 *     validation refusals and they name the actual problem. Note this is the
 *     one case where `ApiError.userMessage` is the wrong choice — it collapses
 *     every 4xx that isn't 401/403/404 into "The request failed."
 *  2. `ApiError.userMessage` for transport-shaped failures, where the server
 *     text is either absent or an implementation detail.
 *  3. A sanitized `Error.message` for non-API throws, since those can carry
 *     internal paths.
 *
 * Server-authored text is NOT run through `sanitizeErrorMessage`, and it is
 * worth being precise about why. The helper's job is to keep incidental
 * filesystem detail out of logs from arbitrary JS throws. Our own 4xx bodies
 * are the opposite: written for this operator, on this machine, and when they
 * name a path ("config at /etc/vrooli/service.json is invalid") that path is
 * the actionable content. Redacting it would leave a sentence that says
 * something is wrong somewhere. It is length-capped instead.
 */
export function describeError(error: unknown): ErrorDescription {
  const category = categorizeError(error);
  const guidance = RECOVERY_PATHS[category];

  if (isApiError(error)) {
    const serverMessage = error.message;
    const preferServer = error.isClientError
      && error.status !== 401
      && error.status !== 403
      && isMeaningfulServerMessage(serverMessage);
    const message = preferServer ? elide(serverMessage) : error.userMessage;
    return {
      category,
      message,
      recovery: guidance.action,
      canRetry: guidance.canRetry && error.isRetryable,
      code: error.code,
      status: error.status,
    };
  }

  if (error instanceof Error) {
    return {
      category,
      message: sanitizeErrorMessage(error.message) || "An unexpected error occurred.",
      recovery: guidance.action,
      canRetry: guidance.canRetry,
      code: "",
    };
  }

  return {
    category,
    message: "An unexpected error occurred.",
    recovery: guidance.action,
    canRetry: guidance.canRetry,
    code: "",
  };
}

/**
 * Convenience for call sites that only need the sentence.
 *
 * `fallback` covers the case where an operation failed with no usable detail
 * and the caller knows a better domain-specific phrasing.
 */
export function errorMessageOf(error: unknown, fallback?: string): string {
  const described = describeError(error);
  if (fallback && described.category === "RUNTIME" && !(error instanceof Error)) {
    return fallback;
  }
  return described.message;
}

function elide(message: string): string {
  const trimmed = message.trim();
  return trimmed.length > MAX_DESCRIBED_MESSAGE
    ? `${trimmed.slice(0, MAX_DESCRIBED_MESSAGE - 1)}…`
    : trimmed;
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
