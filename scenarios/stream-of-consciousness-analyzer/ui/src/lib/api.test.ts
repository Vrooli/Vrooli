import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { ApiRequestError } from "./api";

// Mock @vrooli/api-base
vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:3000/api/v1",
  buildApiUrl: (path: string) => `http://localhost:3000/api/v1${path}`,
}));

// [REQ:P0-001] API client error handling tests
describe("ApiRequestError", () => {
  it("carries status, category, and retryable fields", () => {
    const err = new ApiRequestError(503, {
      category: "dependency",
      message: "service down",
      retryable: true,
    });
    expect(err.status).toBe(503);
    expect(err.category).toBe("dependency");
    expect(err.retryable).toBe(true);
    expect(err.message).toBe("service down");
    expect(err.name).toBe("ApiRequestError");
  });

  it("extends Error for standard error handling", () => {
    const err = new ApiRequestError(400, {
      category: "validation",
      message: "bad",
      retryable: false,
    });
    expect(err instanceof Error).toBe(true);
    expect(err instanceof ApiRequestError).toBe(true);
  });
});

// [REQ:P0-001] apiFetch error mapping tests
describe("apiFetch error mapping", () => {
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("wraps network errors as dependency category", async () => {
    globalThis.fetch = vi.fn().mockRejectedValue(new TypeError("Failed to fetch"));

    // Dynamic import to get apiFetch with our mock
    const { listSchemes } = await import("./api");

    try {
      await listSchemes();
      expect.unreachable("should have thrown");
    } catch (e) {
      expect(e).toBeInstanceOf(ApiRequestError);
      if (!(e instanceof ApiRequestError)) throw e;
      const err = e;
      expect(err.category).toBe("dependency");
      expect(err.retryable).toBe(true);
    }
  });

  it("parses structured API error responses", async () => {
    const apiError = { category: "not_found", message: "scheme not found", retryable: false };
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 404,
      json: () => Promise.resolve(apiError),
    });

    const { getScheme } = await import("./api");

    try {
      await getScheme("missing-id");
      expect.unreachable("should have thrown");
    } catch (e) {
      if (!(e instanceof ApiRequestError)) throw e;
      const err = e;
      expect(err.status).toBe(404);
      expect(err.category).toBe("not_found");
      expect(err.message).toBe("scheme not found");
    }
  });

  it("falls back gracefully for non-JSON error responses", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 502,
      json: () => Promise.reject(new Error("not json")),
    });

    const { listSchemes } = await import("./api");

    try {
      await listSchemes();
      expect.unreachable("should have thrown");
    } catch (e) {
      if (!(e instanceof ApiRequestError)) throw e;
      const err = e;
      expect(err.status).toBe(502);
      expect(err.category).toBe("internal");
      expect(err.retryable).toBe(true);
    }
  });
});
