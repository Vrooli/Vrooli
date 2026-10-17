import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";
import { Code, ConnectError } from "@connectrpc/connect";

const API_BASE = resolveApiBase({ appendSuffix: true });
export const buildUrl = (path: string) =>
  buildApiUrl(path, { baseUrl: API_BASE });

/**
 * Recovery actions that indicate what a client should do to recover from an error.
 * These map to the backend RecoveryAction type.
 */
export type RecoveryAction =
  | "retry" // Transient failure, can retry immediately
  | "retry_with_backoff" // Rate limiting or temporary overload, retry after delay
  | "fix_input" // Client should correct input and resubmit
  | "provide_credentials" // Missing authentication or secrets
  | "wait_for_resource" // Resource is being prepared, try again soon
  | "install_dependency" // System dependency must be installed
  | "contact_support" // Unrecoverable error requiring human intervention
  | "none"; // No recovery possible

/**
 * Structured API error response from the backend.
 * Includes machine-readable error code, recovery action, and optional details.
 */
export interface ApiErrorResponse {
  error: string;
  code?: string;
  details?: Record<string, unknown>;
  recovery?: RecoveryAction;
  recovery_hint?: string;
}

/**
 * Custom error class for API errors with structured information.
 * Provides recovery guidance for both UI display and agent consumption.
 */
export class ApiError extends Error {
  /** Machine-readable error code (e.g., "VALIDATION_ERROR", "PIPELINE_NOT_FOUND") */
  readonly code: string;
  /** Optional details about the error (e.g., field-specific validation errors) */
  readonly details?: Record<string, unknown>;
  /** Suggested recovery action for the client */
  readonly recovery: RecoveryAction;
  /** Human-readable hint about how to recover from the error */
  readonly recoveryHint?: string;
  /** HTTP status code if available */
  readonly statusCode?: number;

  constructor(response: ApiErrorResponse, statusCode?: number) {
    super(response.error);
    this.name = "ApiError";
    this.code = response.code ?? "UNKNOWN_ERROR";
    this.details = response.details;
    this.recovery = response.recovery ?? "none";
    this.recoveryHint = response.recovery_hint;
    this.statusCode = statusCode;
  }

  /** Check if this error suggests the client should retry */
  canRetry(): boolean {
    return this.recovery === "retry" || this.recovery === "retry_with_backoff";
  }

  /** Check if this error requires user input correction */
  requiresInputFix(): boolean {
    return (
      this.recovery === "fix_input" || this.recovery === "provide_credentials"
    );
  }

  /** Check if this is a transient error that may resolve on its own */
  isTransient(): boolean {
    return (
      this.recovery === "retry" ||
      this.recovery === "retry_with_backoff" ||
      this.recovery === "wait_for_resource"
    );
  }

  /**
   * Get a user-friendly message combining the error message and recovery hint.
   */
  getUserMessage(): string {
    if (this.recoveryHint) {
      return `${this.message}. ${this.recoveryHint}`;
    }
    return this.message;
  }
}

/**
 * Converts a typed Connect failure into the UI's shared error representation.
 * Connect metadata and Any details are retained verbatim for recovery UI and
 * diagnostics instead of being collapsed to a message string.
 */
export function apiErrorFromConnect(error: ConnectError): ApiError {
  const envelope = connectErrorEnvelope(error);
  if (envelope) {
    const recovery = envelope.recovery;
    return new ApiError({
      error: error.rawMessage,
      code: envelope.code || Code[error.code],
      recovery: isRecoveryAction(recovery)
        ? recovery
        : connectRecovery(error.code),
      recovery_hint: envelope.recoveryHint || undefined,
      details: {
        ...decodeDetails(envelope.details ?? {}),
        category: envelope.category,
        retryStrategy: envelope.retryStrategy,
        autoFix: envelope.autoFix,
        manualSteps: envelope.manualSteps,
        diagnostic: envelope.diagnostic,
      },
    });
  }
  const recovery = connectRecovery(error.code);
  return new ApiError({
    error: error.rawMessage,
    code: Code[error.code],
    recovery,
    details: {
      connectCode: error.code,
      connectMetadata: Object.fromEntries(error.metadata.entries()),
      connectDetails: error.details,
    },
  });
}

type ConnectEnvelope = {
  code?: string;
  category?: string;
  recovery?: string;
  recoveryHint?: string;
  details?: Record<string, string>;
  retryStrategy?: string;
  autoFix?: string;
  manualSteps?: string[];
  diagnostic?: string;
};

// Error details are self-describing Any messages. This narrow decoder keeps
// the client forward-compatible while the generated package is refreshed by
// the workspace lifecycle: unknown detail types remain in connectDetails.
function connectErrorEnvelope(
  error: ConnectError,
): ConnectEnvelope | undefined {
  for (const detail of error.details) {
    const incoming = detail as { type?: unknown; debug?: unknown };
    if (
      incoming.type !== "vrooli.scenario_to_desktop.v1.shared.ErrorEnvelope" ||
      !isRecord(incoming.debug)
    ) {
      continue;
    }
    return incoming.debug as ConnectEnvelope;
  }
  return undefined;
}

function decodeDetails(
  details: Record<string, string>,
): Record<string, unknown> {
  return Object.fromEntries(
    Object.entries(details).map(([key, value]) => {
      try {
        return [key, JSON.parse(value) as unknown];
      } catch {
        return [key, value];
      }
    }),
  );
}

function isRecoveryAction(value: unknown): value is RecoveryAction {
  return (
    typeof value === "string" &&
    [
      "retry",
      "retry_with_backoff",
      "fix_input",
      "provide_credentials",
      "wait_for_resource",
      "install_dependency",
      "contact_support",
      "none",
    ].includes(value)
  );
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function connectRecovery(code: Code): RecoveryAction {
  switch (code) {
    case Code.InvalidArgument:
    case Code.OutOfRange:
      return "fix_input";
    case Code.DeadlineExceeded:
    case Code.Unavailable:
    case Code.ResourceExhausted:
    case Code.Aborted:
      return "retry_with_backoff";
    case Code.FailedPrecondition:
      return "wait_for_resource";
    case Code.Unauthenticated:
    case Code.PermissionDenied:
      return "provide_credentials";
    default:
      return "none";
  }
}
