/**
 * Utility functions for handling and displaying API errors in the UI.
 * Works with the ApiError class from ./api.ts
 */

import type { RecoveryAction } from "./api";
import { ApiError } from "./api";

/**
 * Get a user-friendly message for a recovery action.
 */
export function getRecoveryActionLabel(action: RecoveryAction): string {
  switch (action) {
    case "retry":
      return "Try again";
    case "retry_with_backoff":
      return "Wait and try again";
    case "fix_input":
      return "Check your input";
    case "provide_credentials":
      return "Provide credentials";
    case "wait_for_resource":
      return "Resource being prepared";
    case "install_dependency":
      return "Install required dependency";
    case "contact_support":
      return "Contact support";
    case "none":
    default:
      return "";
  }
}

/**
 * Get styling hints for error display based on recovery action.
 */
export interface ErrorStyling {
  /** Severity level for color/icon selection */
  severity: "error" | "warning" | "info";
  /** Whether to show a retry button */
  showRetry: boolean;
  /** Whether to auto-dismiss after a delay */
  autoDismiss: boolean;
}

export function getErrorStyling(error: unknown): ErrorStyling {
  if (error instanceof ApiError) {
    // Transient errors that can be retried are less severe
    if (error.isTransient()) {
      return {
        severity: "warning",
        showRetry: error.canRetry(),
        autoDismiss: false,
      };
    }
    // Errors requiring user action are informational
    if (error.requiresInputFix()) {
      return {
        severity: "info",
        showRetry: false,
        autoDismiss: false,
      };
    }
  }
  // Default for unrecoverable errors
  return {
    severity: "error",
    showRetry: false,
    autoDismiss: false,
  };
}

/**
 * Extract a displayable message from any error type.
 * Prefers structured ApiError messages with recovery hints.
 */
export function getErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    return error.getUserMessage();
  }
  if (error instanceof Error) {
    return error.message;
  }
  if (typeof error === "string") {
    return error;
  }
  return "An unknown error occurred";
}

/**
 * Extract error code if available.
 */
export function getErrorCode(error: unknown): string | undefined {
  if (error instanceof ApiError) {
    return error.code;
  }
  return undefined;
}

/**
 * Check if an error is an ApiError instance.
 */
export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}

/**
 * Format error details as a readable string (for debugging or detailed views).
 */
export function formatErrorDetails(error: unknown): string {
  if (error instanceof ApiError && error.details) {
    return JSON.stringify(error.details, null, 2);
  }
  return "";
}

/**
 * Log an error for observability, including structured fields if available.
 * This can be extended to send to a logging service.
 */
export function logError(context: string, error: unknown): void {
  const timestamp = new Date().toISOString();

  if (error instanceof ApiError) {
    console.error(`[${timestamp}] ${context}:`, {
      message: error.message,
      code: error.code,
      recovery: error.recovery,
      recoveryHint: error.recoveryHint,
      statusCode: error.statusCode,
      details: error.details,
    });
  } else if (error instanceof Error) {
    console.error(`[${timestamp}] ${context}:`, {
      message: error.message,
      name: error.name,
      stack: error.stack,
    });
  } else {
    console.error(`[${timestamp}] ${context}:`, error);
  }
}

/**
 * Suggested retry delay based on recovery action (in milliseconds).
 */
export function getSuggestedRetryDelay(error: unknown): number {
  if (error instanceof ApiError) {
    switch (error.recovery) {
      case "retry":
        return 1000; // 1 second for immediate retry
      case "retry_with_backoff":
        return 5000; // 5 seconds for backoff
      case "wait_for_resource":
        return 3000; // 3 seconds for resource wait
      default:
        return 0;
    }
  }
  return 0;
}

/**
 * Structured error information for store/component consumption.
 * This interface decouples error representation from the ApiError class.
 */
export interface ErrorInfo {
  /** Human-readable error message */
  message: string;
  /** Machine-readable error code */
  code?: string;
  /** Whether the error can be retried */
  canRetry: boolean;
  /** Whether user input needs correction */
  requiresInputFix: boolean;
  /** Human-readable recovery hint */
  recoveryHint?: string;
}

/**
 * Create structured error info from any error type.
 * This is a pure function that transforms errors into a data structure
 * suitable for UI consumption without coupling to the ApiError class methods.
 */
export function createErrorInfo(err: unknown): ErrorInfo {
  if (err instanceof ApiError) {
    return {
      message: err.message,
      code: err.code,
      canRetry: err.canRetry(),
      requiresInputFix: err.requiresInputFix(),
      recoveryHint: err.recoveryHint,
    };
  }
  // Fallback for non-ApiError errors
  const message = getErrorMessage(err);
  return {
    message,
    canRetry: false,
    requiresInputFix: false,
  };
}
