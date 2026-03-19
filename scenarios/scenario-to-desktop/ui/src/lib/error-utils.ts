/**
 * Utility functions for handling and displaying API errors in the UI.
 * Works with the ApiError class from ./api.ts
 */

import { ApiError } from "./api";

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
