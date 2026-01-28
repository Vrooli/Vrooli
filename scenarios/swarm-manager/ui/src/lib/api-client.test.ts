import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { ApiError, isApiError, ApiClient } from "./api-client";

/**
 * API Client Tests - Failure Handling
 *
 * These tests verify the structured error handling and graceful degradation
 * patterns in the API client. They ensure:
 * - Errors are properly differentiated by type
 * - User messages are helpful and non-technical
 * - Retryable vs non-retryable errors are correctly identified
 * - Timeouts are properly enforced
 */

// [REQ:PHASE5] Test error differentiation for graceful degradation
describe("ApiError", () => {
  describe("construction", () => {
    it("creates network error with correct properties", () => {
      const error = new ApiError("network", "Network request failed");

      expect(error.type).toBe("network");
      expect(error.message).toBe("Network request failed");
      expect(error.status).toBeUndefined();
      expect(error.isClientError).toBe(false);
      expect(error.isServerError).toBe(false);
      expect(error.isRetryable).toBe(true);
    });

    it("creates timeout error with correct properties", () => {
      const error = new ApiError("timeout", "Request timed out");

      expect(error.type).toBe("timeout");
      expect(error.isRetryable).toBe(true);
    });

    it("creates HTTP 4xx error as client error", () => {
      const error = new ApiError("http", "Not found", { status: 404 });

      expect(error.type).toBe("http");
      expect(error.status).toBe(404);
      expect(error.isClientError).toBe(true);
      expect(error.isServerError).toBe(false);
      expect(error.isRetryable).toBe(false);
    });

    it("creates HTTP 5xx error as server error", () => {
      const error = new ApiError("http", "Internal server error", { status: 500 });

      expect(error.type).toBe("http");
      expect(error.status).toBe(500);
      expect(error.isClientError).toBe(false);
      expect(error.isServerError).toBe(true);
      expect(error.isRetryable).toBe(true);
    });

    it("creates parse error as non-retryable", () => {
      const error = new ApiError("parse", "Failed to parse response");

      expect(error.type).toBe("parse");
      expect(error.isRetryable).toBe(false);
    });

    it("preserves cause for debugging", () => {
      const cause = new Error("Original error");
      const error = new ApiError("network", "Wrapped error", { cause });

      expect(error.cause).toBe(cause);
    });
  });

  describe("userMessage", () => {
    it("provides friendly message for network error", () => {
      const error = new ApiError("network", "fetch failed");
      expect(error.userMessage).toContain("Unable to connect");
      expect(error.userMessage).toContain("internet connection");
    });

    it("provides friendly message for timeout error", () => {
      const error = new ApiError("timeout", "aborted");
      expect(error.userMessage).toContain("timed out");
      expect(error.userMessage).toContain("try again");
    });

    it("provides session expired message for 401", () => {
      const error = new ApiError("http", "Unauthorized", { status: 401 });
      expect(error.userMessage).toContain("session has expired");
    });

    it("provides permission message for 403", () => {
      const error = new ApiError("http", "Forbidden", { status: 403 });
      expect(error.userMessage).toContain("permission");
    });

    it("provides not found message for 404", () => {
      const error = new ApiError("http", "Not found", { status: 404 });
      expect(error.userMessage).toContain("not found");
    });

    it("provides server error message for 5xx", () => {
      const error = new ApiError("http", "Internal error", { status: 500 });
      expect(error.userMessage).toContain("server encountered an error");
    });

    it("provides parse error message", () => {
      const error = new ApiError("parse", "JSON parse failed");
      expect(error.userMessage).toContain("invalid response");
    });

    it("does not expose technical details", () => {
      const error = new ApiError("network", "TypeError: Failed to fetch at http://localhost:15000/api/v1/ideas");
      expect(error.userMessage).not.toContain("localhost");
      expect(error.userMessage).not.toContain("TypeError");
      expect(error.userMessage).not.toContain("/api/");
    });
  });
});

describe("isApiError", () => {
  it("returns true for ApiError instances", () => {
    const error = new ApiError("network", "test");
    expect(isApiError(error)).toBe(true);
  });

  it("returns false for regular Error instances", () => {
    const error = new Error("test");
    expect(isApiError(error)).toBe(false);
  });

  it("returns false for null/undefined", () => {
    expect(isApiError(null)).toBe(false);
    expect(isApiError(undefined)).toBe(false);
  });
});

// [REQ:PHASE5] Test API client error handling behavior
describe("ApiClient", () => {
  let fetchMock: ReturnType<typeof vi.fn>;
  let originalFetch: typeof globalThis.fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    fetchMock = vi.fn();
    globalThis.fetch = fetchMock;
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  it("throws HttpError on non-2xx response", async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 404,
      headers: new Headers({ "content-type": "application/json" }),
    });

    const client = new ApiClient("http://localhost:15000", 5000);

    await expect(client.get("/test")).rejects.toMatchObject({
      type: "http",
      status: 404,
    });
  });

  it("throws ParseError on invalid JSON", async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      headers: new Headers({ "content-type": "application/json" }),
      json: () => Promise.reject(new SyntaxError("Unexpected token")),
    });

    const client = new ApiClient("http://localhost:15000", 5000);

    await expect(client.get("/test")).rejects.toMatchObject({
      type: "parse",
    });
  });

  it("handles non-JSON responses gracefully", async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      headers: new Headers({ "content-type": "text/plain" }),
    });

    const client = new ApiClient("http://localhost:15000", 5000);
    const result = await client.get("/health");

    expect(result).toBeUndefined();
  });

  it("throws NetworkError on fetch failure", async () => {
    fetchMock.mockRejectedValue(new TypeError("Failed to fetch"));

    const client = new ApiClient("http://localhost:15000", 5000);

    await expect(client.get("/test")).rejects.toMatchObject({
      type: "network",
    });
  });

  it("throws TimeoutError when request is aborted", async () => {
    // Directly simulate what happens when AbortController triggers
    const abortError = new DOMException("The operation was aborted.", "AbortError");
    fetchMock.mockRejectedValue(abortError);

    const client = new ApiClient("http://localhost:15000", 50); // 50ms timeout

    await expect(client.get("/test")).rejects.toMatchObject({
      type: "timeout",
    });
  });

  it("clears timeout on successful response", async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      headers: new Headers({ "content-type": "application/json" }),
      json: () => Promise.resolve({ data: "test" }),
    });

    const client = new ApiClient("http://localhost:15000", 30000);
    const result = await client.get("/test");

    expect(result).toEqual({ data: "test" });
  });
});
