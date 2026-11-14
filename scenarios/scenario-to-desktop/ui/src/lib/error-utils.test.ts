/**
 * Tests for error utility functions.
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { Code, ConnectError } from "@connectrpc/connect";
import { ApiError, type RecoveryAction } from "./api";
import { getErrorMessage, logError, createErrorInfo } from "./error-utils";

// Helper to create ApiError instances
function makeApiError(
  opts: {
    error?: string;
    code?: string;
    recovery?: RecoveryAction;
    recoveryHint?: string;
    details?: Record<string, unknown>;
  } = {},
): ApiError {
  return new ApiError({
    error: opts.error ?? "Test error",
    code: opts.code,
    recovery: opts.recovery,
    recovery_hint: opts.recoveryHint,
    details: opts.details,
  });
}

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
    expect(args?.[0]).toContain("Test context");
    expect(args?.[1]).toMatchObject({
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
    expect(args?.[1]).toMatchObject({
      message: "Regular error",
      name: "Error",
    });
  });

  it("logs non-error values directly", () => {
    logError("Context", "string error");
    expect(consoleSpy).toHaveBeenCalled();
    const args = consoleSpy.mock.calls[0];
    expect(args?.[1]).toBe("string error");
  });
});

// ============================================================================
// Error Info Creation
// ============================================================================

describe("createErrorInfo", () => {
  it("decodes the shared Connect remediation envelope", () => {
    const connectError = new ConnectError("pipeline missing", Code.NotFound);
    connectError.details.push({
      type: "vrooli.scenario_to_desktop.v1.shared.ErrorEnvelope",
      value: new Uint8Array(),
      debug: {
        code: "PIPELINE_NOT_FOUND",
        category: "resource_missing",
        recovery: "fix_input",
        recoveryHint: "Start a new pipeline or check the ID",
        details: { pipeline_id: '"pipe-42"' },
        manualSteps: ["Run scenario-to-desktop pipeline list"],
      },
    });

    expect(createErrorInfo(connectError)).toMatchObject({
      code: "PIPELINE_NOT_FOUND",
      requiresInputFix: true,
      recoveryHint: "Start a new pipeline or check the ID",
      details: {
        pipeline_id: "pipe-42",
        category: "resource_missing",
      },
    });
  });

  it("falls back safely when an envelope has incomplete or malformed fields", () => {
    const connectError = new ConnectError(
      "temporary failure",
      Code.Unavailable,
    );
    connectError.details.push(
      {
        type: "unrelated.detail",
        value: new Uint8Array(),
        debug: { recovery: "retry" },
      },
      {
        type: "vrooli.scenario_to_desktop.v1.shared.ErrorEnvelope",
        value: new Uint8Array(),
        debug: {
          recovery: "not-a-recovery-action",
          details: { machine_value: "not-json" },
        },
      },
    );

    expect(createErrorInfo(connectError)).toMatchObject({
      code: "Unavailable",
      canRetry: true,
      recoveryHint: undefined,
      details: {
        machine_value: "not-json",
      },
    });
  });

  it("ignores malformed envelope debug payloads and retains generic Connect diagnostics", () => {
    const connectError = new ConnectError("bad input", Code.InvalidArgument);
    connectError.details.push({
      type: "vrooli.scenario_to_desktop.v1.shared.ErrorEnvelope",
      value: new Uint8Array(),
      debug: [],
    });

    expect(createErrorInfo(connectError)).toMatchObject({
      code: "InvalidArgument",
      requiresInputFix: true,
      details: { connectCode: Code.InvalidArgument },
    });
  });

  it("preserves Connect code, metadata, and details for recovery UI", () => {
    const connectError = new ConnectError(
      "pipeline service is temporarily unavailable",
      Code.Unavailable,
      { "x-request-id": "request-123" },
    );
    connectError.details.push({
      type: "type.googleapis.com/vrooli.scenario_to_desktop.v1.ErrorDetail",
      value: new Uint8Array([1, 2, 3]),
    });

    const info = createErrorInfo(connectError);

    expect(info).toMatchObject({
      message: "pipeline service is temporarily unavailable",
      code: "Unavailable",
      canRetry: true,
      details: {
        connectCode: Code.Unavailable,
        connectMetadata: { "x-request-id": "request-123" },
      },
    });
    expect(info.details?.connectDetails).toHaveLength(1);
  });

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
