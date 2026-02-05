import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  categorizeError,
  createErrorLogEntry,
  createSuccessLogEntry,
  generateCorrelationId,
  generateUniqueId,
  getRecoveryGuidance,
  logError,
  logSuccess,
  RECOVERY_PATHS,
  type ErrorCategory,
  type OperationOutcome,
} from "./error-utils";
import { ApiError } from "./api-client";

/**
 * Error Utilities Tests
 *
 * These tests verify:
 * - Error categorization is accurate
 * - Structured logging works correctly
 * - Sensitive information is sanitized
 * - Recovery paths are appropriate
 *
 * [REQ:PHASE6] Test error categorization and recovery paths
 */

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null;

const isErrorLogPayload = (value: unknown): value is { category: string; source: string } =>
  isRecord(value) && typeof value.category === "string" && typeof value.source === "string";

const isSuccessLogPayload = (value: unknown): value is { outcome: string; source: string; message: string } =>
  isRecord(value) &&
  typeof value.outcome === "string" &&
  typeof value.source === "string" &&
  typeof value.message === "string";

describe("categorizeError", () => {
  describe("ApiError categorization", () => {
    it("categorizes network error as NETWORK", () => {
      const error = new ApiError("network", "Failed to fetch");
      expect(categorizeError(error)).toBe("NETWORK");
    });

    it("categorizes timeout error as TIMEOUT", () => {
      const error = new ApiError("timeout", "Request timed out");
      expect(categorizeError(error)).toBe("TIMEOUT");
    });

    it("categorizes 401 as AUTH", () => {
      const error = new ApiError("http", "Unauthorized", { status: 401 });
      expect(categorizeError(error)).toBe("AUTH");
    });

    it("categorizes 403 as AUTH", () => {
      const error = new ApiError("http", "Forbidden", { status: 403 });
      expect(categorizeError(error)).toBe("AUTH");
    });

    it("categorizes 404 as NOT_FOUND", () => {
      const error = new ApiError("http", "Not found", { status: 404 });
      expect(categorizeError(error)).toBe("NOT_FOUND");
    });

    it("categorizes 400 as VALIDATION", () => {
      const error = new ApiError("http", "Bad request", { status: 400 });
      expect(categorizeError(error)).toBe("VALIDATION");
    });

    it("categorizes 422 as VALIDATION", () => {
      const error = new ApiError("http", "Unprocessable entity", { status: 422 });
      expect(categorizeError(error)).toBe("VALIDATION");
    });

    it("categorizes 500 as SERVER", () => {
      const error = new ApiError("http", "Internal server error", { status: 500 });
      expect(categorizeError(error)).toBe("SERVER");
    });

    it("categorizes 503 as SERVER", () => {
      const error = new ApiError("http", "Service unavailable", { status: 503 });
      expect(categorizeError(error)).toBe("SERVER");
    });

    it("categorizes parse error as PARSE", () => {
      const error = new ApiError("parse", "Failed to parse");
      expect(categorizeError(error)).toBe("PARSE");
    });
  });

  describe("generic Error categorization", () => {
    it("categorizes network-like errors as NETWORK", () => {
      expect(categorizeError(new Error("network request failed"))).toBe("NETWORK");
      expect(categorizeError(new Error("Failed to fetch"))).toBe("NETWORK");
    });

    it("categorizes timeout-like errors as TIMEOUT", () => {
      expect(categorizeError(new Error("Request timeout"))).toBe("TIMEOUT");
      expect(categorizeError(new Error("The operation was aborted"))).toBe("TIMEOUT");
    });

    it("categorizes unknown errors as RUNTIME", () => {
      expect(categorizeError(new Error("Something unexpected"))).toBe("RUNTIME");
      expect(categorizeError("string error")).toBe("RUNTIME");
      expect(categorizeError(null)).toBe("RUNTIME");
    });
  });
});

describe("generateUniqueId", () => {
  it("generates unique IDs with given prefix", () => {
    const id1 = generateUniqueId("test");
    const id2 = generateUniqueId("test");
    expect(id1).not.toBe(id2);
    expect(id1).toMatch(/^test_[a-z0-9]+_[a-z0-9]+$/);
    expect(id2).toMatch(/^test_[a-z0-9]+_[a-z0-9]+$/);
  });

  it("uses the exact prefix provided", () => {
    expect(generateUniqueId("err")).toMatch(/^err_/);
    expect(generateUniqueId("page_err")).toMatch(/^page_err_/);
    expect(generateUniqueId("trace")).toMatch(/^trace_/);
    expect(generateUniqueId("custom_prefix")).toMatch(/^custom_prefix_/);
  });

  it("generates IDs with timestamp and random suffix", () => {
    const id = generateUniqueId("test");
    const parts = id.split("_");
    // Format: prefix_timestamp_random (could be more parts if prefix has underscores)
    expect(parts.length).toBeGreaterThanOrEqual(3);
    // Last part should be random alphanumeric
    const lastPart = parts[parts.length - 1];
    expect(lastPart).toBeDefined();
    expect(lastPart).toMatch(/^[a-z0-9]+$/);
    expect(lastPart?.length).toBe(6);
  });
});

describe("generateCorrelationId", () => {
  it("generates unique IDs", () => {
    const id1 = generateCorrelationId();
    const id2 = generateCorrelationId();
    expect(id1).not.toBe(id2);
  });

  it("generates IDs with correct format (uses err prefix)", () => {
    const id = generateCorrelationId();
    expect(id).toMatch(/^err_[a-z0-9]+_[a-z0-9]+$/);
  });

  it("is equivalent to generateUniqueId with err prefix", () => {
    // Both should produce the same format
    const correlationId = generateCorrelationId();
    const uniqueId = generateUniqueId("err");
    expect(correlationId).toMatch(/^err_[a-z0-9]+_[a-z0-9]+$/);
    expect(uniqueId).toMatch(/^err_[a-z0-9]+_[a-z0-9]+$/);
  });
});

describe("createErrorLogEntry", () => {
  it("creates structured entry for ApiError", () => {
    const error = new ApiError("http", "Server error", { status: 500 });
    const entry = createErrorLogEntry(error, "TestComponent");

    expect(entry.category).toBe("SERVER");
    expect(entry.status).toBe(500);
    expect(entry.retryable).toBe(true);
    expect(entry.source).toBe("TestComponent");
    expect(entry.correlationId).toMatch(/^err_/);
    expect(entry.timestamp).toBeDefined();
  });

  it("creates structured entry for generic Error", () => {
    const error = new Error("Something went wrong");
    const entry = createErrorLogEntry(error, "AnotherComponent");

    expect(entry.category).toBe("RUNTIME");
    expect(entry.status).toBeUndefined();
    expect(entry.retryable).toBe(false);
    expect(entry.source).toBe("AnotherComponent");
  });

  it("includes context when provided", () => {
    const error = new Error("Test");
    const entry = createErrorLogEntry(error, "Test", { action: "load", retry: 2 });

    expect(entry.context).toEqual({ action: "load", retry: 2 });
  });

  it("sanitizes URLs from error messages", () => {
    const error = new Error("Failed at https://api.example.com/secret/path");
    const entry = createErrorLogEntry(error, "Test");

    expect(entry.message).not.toContain("api.example.com");
    expect(entry.message).toContain("[URL]");
  });

  it("sanitizes file paths from error messages", () => {
    const error = new Error("Error in /home/user/secret/file.ts");
    const entry = createErrorLogEntry(error, "Test");

    expect(entry.message).not.toContain("/home/user");
    expect(entry.message).toContain("[PATH]");
  });

  it("truncates long messages", () => {
    const longMessage = "a".repeat(300);
    const error = new Error(longMessage);
    const entry = createErrorLogEntry(error, "Test");

    expect(entry.message.length).toBeLessThanOrEqual(200);
    expect(entry.message).toContain("...");
  });
});

describe("logError", () => {
  let consoleSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    consoleSpy.mockRestore();
  });

  it("logs structured JSON to console", () => {
    const error = new ApiError("network", "Failed to fetch");
    logError(error, "TestComponent");

    expect(consoleSpy).toHaveBeenCalled();
    const firstCall = consoleSpy.mock.calls[0];
    expect(firstCall).toBeDefined();
    const [prefix, json] = firstCall ?? [];
    expect(prefix).toBe("[NETWORK]");

    const parsed: unknown = JSON.parse(String(json));
    if (!isErrorLogPayload(parsed)) {
      throw new Error("Expected error log payload.");
    }
    expect(parsed.category).toBe("NETWORK");
    expect(parsed.source).toBe("TestComponent");
  });

  it("returns the log entry", () => {
    const error = new Error("Test");
    const entry = logError(error, "Test");

    expect(entry.category).toBeDefined();
    expect(entry.correlationId).toBeDefined();
  });
});

describe("RECOVERY_PATHS", () => {
  it("has recovery path for each category", () => {
    const categories: ErrorCategory[] = [
      "NETWORK", "TIMEOUT", "AUTH", "NOT_FOUND",
      "SERVER", "VALIDATION", "PARSE", "RUNTIME",
    ];

    for (const category of categories) {
      expect(RECOVERY_PATHS[category]).toBeDefined();
      expect(RECOVERY_PATHS[category].action).toBeTruthy();
      expect(typeof RECOVERY_PATHS[category].canRetry).toBe("boolean");
    }
  });

  it("marks retryable categories correctly", () => {
    expect(RECOVERY_PATHS.NETWORK.canRetry).toBe(true);
    expect(RECOVERY_PATHS.TIMEOUT.canRetry).toBe(true);
    expect(RECOVERY_PATHS.SERVER.canRetry).toBe(true);

    expect(RECOVERY_PATHS.AUTH.canRetry).toBe(false);
    expect(RECOVERY_PATHS.NOT_FOUND.canRetry).toBe(false);
    expect(RECOVERY_PATHS.VALIDATION.canRetry).toBe(false);
    expect(RECOVERY_PATHS.PARSE.canRetry).toBe(false);
    expect(RECOVERY_PATHS.RUNTIME.canRetry).toBe(false);
  });
});

describe("getRecoveryGuidance", () => {
  it("returns correct guidance for ApiError", () => {
    const error = new ApiError("network", "Failed");
    const guidance = getRecoveryGuidance(error);

    expect(guidance.category).toBe("NETWORK");
    expect(guidance.canRetry).toBe(true);
    expect(guidance.buttonLabel).toBe("Try Again");
  });

  it("returns correct guidance for generic error", () => {
    const error = new Error("Unknown error");
    const guidance = getRecoveryGuidance(error);

    expect(guidance.category).toBe("RUNTIME");
    expect(guidance.canRetry).toBe(false);
    expect(guidance.buttonLabel).toBe("Refresh Page");
  });
});

// ============================================================================
// Success Logging Tests (Phase 20: Signal & Feedback Surface Design)
// ============================================================================

describe("createSuccessLogEntry", () => {
  it("creates structured entry with all required fields", () => {
    const entry = createSuccessLogEntry("created", "Backlog item created successfully", "BacklogPage");

    expect(entry.outcome).toBe("created");
    expect(entry.message).toBe("Backlog item created successfully");
    expect(entry.source).toBe("BacklogPage");
    expect(entry.correlationId).toMatch(/^op_[a-z0-9]+_[a-z0-9]+$/);
    expect(entry.timestamp).toBeDefined();
    expect(new Date(entry.timestamp).getTime()).not.toBeNaN();
  });

  it("includes context when provided", () => {
    const entry = createSuccessLogEntry("updated", "Backlog item updated", "BacklogPage", {
      backlogName: "test-idea",
      priority: 1,
    });

    expect(entry.context).toEqual({ backlogName: "test-idea", priority: 1 });
  });

  it("supports all operation outcomes", () => {
    const outcomes: OperationOutcome[] = ["created", "updated", "deleted", "fetched", "completed"];

    for (const outcome of outcomes) {
      const entry = createSuccessLogEntry(outcome, `Test ${outcome}`, "Test");
      expect(entry.outcome).toBe(outcome);
    }
  });
});

describe("logSuccess", () => {
  let consoleSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    consoleSpy = vi.spyOn(console, "info").mockImplementation(() => {});
  });

  afterEach(() => {
    consoleSpy.mockRestore();
  });

  it("logs structured JSON to console.info", () => {
    logSuccess("created", "Backlog item created", "BacklogPage");

    expect(consoleSpy).toHaveBeenCalled();
    const firstCall = consoleSpy.mock.calls[0];
    expect(firstCall).toBeDefined();
    const [prefix, json] = firstCall ?? [];
    expect(prefix).toBe("[CREATED]");

    const parsed: unknown = JSON.parse(String(json));
    if (!isSuccessLogPayload(parsed)) {
      throw new Error("Expected success log payload.");
    }
    expect(parsed.outcome).toBe("created");
    expect(parsed.source).toBe("BacklogPage");
    expect(parsed.message).toBe("Backlog item created");
  });

  it("returns the log entry", () => {
    const entry = logSuccess("deleted", "Backlog item deleted", "BacklogPage", { backlogName: "test" });

    expect(entry.outcome).toBe("deleted");
    expect(entry.correlationId).toBeDefined();
    expect(entry.context).toEqual({ backlogName: "test" });
  });

  it("uses correct prefix for different outcomes", () => {
    logSuccess("fetched", "Data loaded", "DataService");
    const firstCall = consoleSpy.mock.calls[0];
    expect(firstCall?.[0]).toBe("[FETCHED]");

    logSuccess("completed", "Operation complete", "TaskRunner");
    const secondCall = consoleSpy.mock.calls[1];
    expect(secondCall?.[0]).toBe("[COMPLETED]");
  });
});
