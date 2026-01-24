/**
 * Tests for error utility functions.
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { ApiError, type RecoveryAction } from "./api";
import {
  getRecoveryActionLabel,
  getErrorStyling,
  getErrorMessage,
  getErrorCode,
  isApiError,
  formatErrorDetails,
  logError,
  getSuggestedRetryDelay,
  createErrorInfo,
} from "./error-utils";

// Helper to create ApiError instances
function makeApiError(opts: {
  error?: string;
  code?: string;
  recovery?: RecoveryAction;
  recoveryHint?: string;
  details?: Record<string, unknown>;
} = {}): ApiError {
  return new ApiError({
    error: opts.error ?? "Test error",
    code: opts.code,
    recovery: opts.recovery,
    recovery_hint: opts.recoveryHint,
    details: opts.details,
  });
}

// ============================================================================
// Recovery Action Labels
// ============================================================================

describe("getRecoveryActionLabel", () => {
  it("returns correct label for retry", () => {
    expect(getRecoveryActionLabel("retry")).toBe("Try again");
  });

  it("returns correct label for retry_with_backoff", () => {
    expect(getRecoveryActionLabel("retry_with_backoff")).toBe("Wait and try again");
  });

  it("returns correct label for fix_input", () => {
    expect(getRecoveryActionLabel("fix_input")).toBe("Check your input");
  });

  it("returns correct label for provide_credentials", () => {
    expect(getRecoveryActionLabel("provide_credentials")).toBe("Provide credentials");
  });

  it("returns correct label for wait_for_resource", () => {
    expect(getRecoveryActionLabel("wait_for_resource")).toBe("Resource being prepared");
  });

  it("returns correct label for install_dependency", () => {
    expect(getRecoveryActionLabel("install_dependency")).toBe("Install required dependency");
  });

  it("returns correct label for contact_support", () => {
    expect(getRecoveryActionLabel("contact_support")).toBe("Contact support");
  });

  it("returns empty string for none", () => {
    expect(getRecoveryActionLabel("none")).toBe("");
  });
});

// ============================================================================
// Error Styling
// ============================================================================

describe("getErrorStyling", () => {
  it("returns warning severity for transient errors", () => {
    const error = makeApiError({ recovery: "retry" });
    const styling = getErrorStyling(error);
    expect(styling.severity).toBe("warning");
    expect(styling.showRetry).toBe(true);
    expect(styling.autoDismiss).toBe(false);
  });

  it("returns warning severity for backoff errors", () => {
    const error = makeApiError({ recovery: "retry_with_backoff" });
    const styling = getErrorStyling(error);
    expect(styling.severity).toBe("warning");
    expect(styling.showRetry).toBe(true);
  });

  it("returns warning severity for wait_for_resource", () => {
    const error = makeApiError({ recovery: "wait_for_resource" });
    const styling = getErrorStyling(error);
    expect(styling.severity).toBe("warning");
    expect(styling.showRetry).toBe(false);
  });

  it("returns info severity for fix_input errors", () => {
    const error = makeApiError({ recovery: "fix_input" });
    const styling = getErrorStyling(error);
    expect(styling.severity).toBe("info");
    expect(styling.showRetry).toBe(false);
  });

  it("returns info severity for provide_credentials errors", () => {
    const error = makeApiError({ recovery: "provide_credentials" });
    const styling = getErrorStyling(error);
    expect(styling.severity).toBe("info");
    expect(styling.showRetry).toBe(false);
  });

  it("returns error severity for unrecoverable errors", () => {
    const error = makeApiError({ recovery: "none" });
    const styling = getErrorStyling(error);
    expect(styling.severity).toBe("error");
    expect(styling.showRetry).toBe(false);
  });

  it("returns error severity for regular Error", () => {
    const error = new Error("Regular error");
    const styling = getErrorStyling(error);
    expect(styling.severity).toBe("error");
    expect(styling.showRetry).toBe(false);
  });

  it("returns error severity for non-error values", () => {
    const styling = getErrorStyling("string error");
    expect(styling.severity).toBe("error");
  });
});

// ============================================================================
// Error Message Extraction
// ============================================================================

describe("getErrorMessage", () => {
  it("returns user message from ApiError", () => {
    const error = makeApiError({
      error: "Base error",
      recoveryHint: "Try this",
    });
    expect(getErrorMessage(error)).toBe("Base error. Try this");
  });

  it("returns message from regular Error", () => {
    const error = new Error("Regular error message");
    expect(getErrorMessage(error)).toBe("Regular error message");
  });

  it("returns string error directly", () => {
    expect(getErrorMessage("String error")).toBe("String error");
  });

  it("returns default message for unknown types", () => {
    expect(getErrorMessage(null)).toBe("An unknown error occurred");
    expect(getErrorMessage(undefined)).toBe("An unknown error occurred");
    expect(getErrorMessage(123)).toBe("An unknown error occurred");
    expect(getErrorMessage({})).toBe("An unknown error occurred");
  });
});

// ============================================================================
// Error Code Extraction
// ============================================================================

describe("getErrorCode", () => {
  it("returns code from ApiError", () => {
    const error = makeApiError({ code: "VALIDATION_ERROR" });
    expect(getErrorCode(error)).toBe("VALIDATION_ERROR");
  });

  it("returns undefined for regular Error", () => {
    expect(getErrorCode(new Error("test"))).toBeUndefined();
  });

  it("returns undefined for non-error values", () => {
    expect(getErrorCode("string")).toBeUndefined();
    expect(getErrorCode(null)).toBeUndefined();
  });
});

// ============================================================================
// Type Guard
// ============================================================================

describe("isApiError", () => {
  it("returns true for ApiError", () => {
    const error = makeApiError();
    expect(isApiError(error)).toBe(true);
  });

  it("returns false for regular Error", () => {
    expect(isApiError(new Error("test"))).toBe(false);
  });

  it("returns false for non-error values", () => {
    expect(isApiError("string")).toBe(false);
    expect(isApiError(null)).toBe(false);
    expect(isApiError(undefined)).toBe(false);
  });
});

// ============================================================================
// Error Details Formatting
// ============================================================================

describe("formatErrorDetails", () => {
  it("formats ApiError details as JSON", () => {
    const error = makeApiError({
      details: { field: "name", issue: "required" },
    });
    const result = formatErrorDetails(error);
    expect(result).toContain('"field"');
    expect(result).toContain('"name"');
    expect(result).toContain('"issue"');
  });

  it("returns empty string for ApiError without details", () => {
    const error = makeApiError();
    expect(formatErrorDetails(error)).toBe("");
  });

  it("returns empty string for regular Error", () => {
    expect(formatErrorDetails(new Error("test"))).toBe("");
  });

  it("returns empty string for non-error values", () => {
    expect(formatErrorDetails("string")).toBe("");
  });
});

// ============================================================================
// Error Logging
// ============================================================================

describe("logError", () => {
  let consoleSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
  });

  it("logs ApiError with structured fields", () => {
    const error = makeApiError({
      error: "Test error",
      code: "TEST_ERROR",
      recovery: "retry",
      recoveryHint: "Try again",
    });
    logError("Test context", error);
    expect(consoleSpy).toHaveBeenCalled();
    const args = consoleSpy.mock.calls[0];
    expect(args[0]).toContain("Test context");
    expect(args[1]).toMatchObject({
      message: "Test error",
      code: "TEST_ERROR",
      recovery: "retry",
      recoveryHint: "Try again",
    });
  });

  it("logs regular Error with name and message", () => {
    const error = new Error("Regular error");
    logError("Context", error);
    expect(consoleSpy).toHaveBeenCalled();
    const args = consoleSpy.mock.calls[0];
    expect(args[1]).toMatchObject({
      message: "Regular error",
      name: "Error",
    });
  });

  it("logs non-error values directly", () => {
    logError("Context", "string error");
    expect(consoleSpy).toHaveBeenCalled();
    const args = consoleSpy.mock.calls[0];
    expect(args[1]).toBe("string error");
  });
});

// ============================================================================
// Retry Delay
// ============================================================================

describe("getSuggestedRetryDelay", () => {
  it("returns 1000ms for retry action", () => {
    const error = makeApiError({ recovery: "retry" });
    expect(getSuggestedRetryDelay(error)).toBe(1000);
  });

  it("returns 5000ms for retry_with_backoff action", () => {
    const error = makeApiError({ recovery: "retry_with_backoff" });
    expect(getSuggestedRetryDelay(error)).toBe(5000);
  });

  it("returns 3000ms for wait_for_resource action", () => {
    const error = makeApiError({ recovery: "wait_for_resource" });
    expect(getSuggestedRetryDelay(error)).toBe(3000);
  });

  it("returns 0 for other recovery actions", () => {
    expect(getSuggestedRetryDelay(makeApiError({ recovery: "none" }))).toBe(0);
    expect(getSuggestedRetryDelay(makeApiError({ recovery: "fix_input" }))).toBe(0);
    expect(getSuggestedRetryDelay(makeApiError({ recovery: "contact_support" }))).toBe(0);
  });

  it("returns 0 for non-ApiError", () => {
    expect(getSuggestedRetryDelay(new Error("test"))).toBe(0);
    expect(getSuggestedRetryDelay("string")).toBe(0);
  });
});

// ============================================================================
// Error Info Creation
// ============================================================================

describe("createErrorInfo", () => {
  it("creates ErrorInfo from ApiError", () => {
    const error = makeApiError({
      error: "Validation failed",
      code: "VALIDATION_ERROR",
      recovery: "fix_input",
      recoveryHint: "Check field values",
    });
    const info = createErrorInfo(error);
    expect(info).toEqual({
      message: "Validation failed",
      code: "VALIDATION_ERROR",
      canRetry: false,
      requiresInputFix: true,
      recoveryHint: "Check field values",
    });
  });

  it("creates ErrorInfo from retryable ApiError", () => {
    const error = makeApiError({ recovery: "retry" });
    const info = createErrorInfo(error);
    expect(info.canRetry).toBe(true);
    expect(info.requiresInputFix).toBe(false);
  });

  it("creates ErrorInfo from regular Error", () => {
    const error = new Error("Something went wrong");
    const info = createErrorInfo(error);
    expect(info).toEqual({
      message: "Something went wrong",
      canRetry: false,
      requiresInputFix: false,
    });
  });

  it("creates ErrorInfo from string error", () => {
    const info = createErrorInfo("String error message");
    expect(info).toEqual({
      message: "String error message",
      canRetry: false,
      requiresInputFix: false,
    });
  });

  it("creates ErrorInfo from unknown error type", () => {
    const info = createErrorInfo(null);
    expect(info.message).toBe("An unknown error occurred");
    expect(info.canRetry).toBe(false);
  });
});
