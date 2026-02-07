import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });
export const buildUrl = (path: string) => buildApiUrl(path, { baseUrl: API_BASE });

/**
 * Recovery actions that indicate what a client should do to recover from an error.
 * These map to the backend RecoveryAction type.
 */
export type RecoveryAction =
  | "retry"                  // Transient failure, can retry immediately
  | "retry_with_backoff"     // Rate limiting or temporary overload, retry after delay
  | "fix_input"              // Client should correct input and resubmit
  | "provide_credentials"    // Missing authentication or secrets
  | "wait_for_resource"      // Resource is being prepared, try again soon
  | "install_dependency"     // System dependency must be installed
  | "contact_support"        // Unrecoverable error requiring human intervention
  | "none";                  // No recovery possible

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
    return this.recovery === "fix_input" || this.recovery === "provide_credentials";
  }

  /** Check if this is a transient error that may resolve on its own */
  isTransient(): boolean {
    return this.recovery === "retry" || this.recovery === "retry_with_backoff" || this.recovery === "wait_for_resource";
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
 * Parse an API response and throw an ApiError if the response indicates an error.
 * Handles both structured error responses and fallback to simple error messages.
 */
async function parseApiError(response: Response): Promise<ApiError> {
  try {
    const data: unknown = await response.json();
    if (data && typeof data === "object" && "error" in data) {
      return new ApiError(data as ApiErrorResponse, response.status);
    }
    // Fallback for non-structured error responses
    const message = data && typeof data === "object" && "message" in data
      ? (data as { message: string }).message
      : response.statusText;
    return new ApiError({
      error: message,
      code: "UNKNOWN_ERROR",
      recovery: "none",
    }, response.status);
  } catch {
    // JSON parsing failed, use status text
    return new ApiError({
      error: response.statusText,
      code: "UNKNOWN_ERROR",
      recovery: response.status >= 500 ? "retry" : "none",
    }, response.status);
  }
}

/**
 * Helper to throw an ApiError from a response if not OK.
 * Use this in API functions for consistent error handling.
 */
export async function throwIfNotOk(response: Response): Promise<void> {
  if (!response.ok) {
    throw await parseApiError(response);
  }
}
