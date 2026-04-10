import { describe, it, expect, vi, beforeEach } from "vitest";
import { apiBaseMock } from "../test-utils";

// Mock api-base before importing api module
vi.mock("@vrooli/api-base", () => apiBaseMock());

// [REQ:P0-002a] PTY Session Backend - API client
// [REQ:P0-004a] api-base HTTP Integration
describe("api module", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("createSession sends POST and returns session info", async () => {
    const mockSession = {
      id: "test-123",
      shell: "/bin/bash",
      created_at: "2026-01-01T00:00:00Z",
      cols: 80,
      rows: 24,
    };

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockSession),
    }) as typeof fetch;

    const { createSession } = await import("../lib/api");
    const result = await createSession({ cols: 80, rows: 24 });

    expect(result).toEqual(mockSession);
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/sessions"),
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("listSessions sends GET and returns array", async () => {
    const mockSessions = [
      { id: "s1", shell: "/bin/bash", created_at: "2026-01-01T00:00:00Z", cols: 80, rows: 24 },
    ];

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockSessions),
    }) as typeof fetch;

    const { listSessions } = await import("../lib/api");
    const result = await listSessions();

    expect(result).toEqual(mockSessions);
  });

  it("deleteSession sends DELETE", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({ ok: true }) as typeof fetch;

    const { deleteSession } = await import("../lib/api");
    await deleteSession("test-123");

    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/sessions/test-123"),
      expect.objectContaining({ method: "DELETE" }),
    );
  });

  // Failure path: structured JSON error is extracted from response
  it("createSession extracts structured error message from JSON response", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 429,
      json: () => Promise.resolve({
        error: "Maximum number of concurrent sessions reached. Close an existing session and try again.",
        code: "session_limit_reached",
      }),
    }) as typeof fetch;

    const { createSession } = await import("../lib/api");
    await expect(createSession()).rejects.toThrow(
      "Maximum number of concurrent sessions reached",
    );
  });

  // Failure path: falls back to status code when response is not JSON
  it("createSession falls back to status code when response is not JSON", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
      json: () => Promise.reject(new Error("not json")),
    }) as typeof fetch;

    const { createSession } = await import("../lib/api");
    await expect(createSession()).rejects.toThrow("500");
  });

  // Failure path: deleteSession extracts structured error message
  it("deleteSession extracts structured error from JSON response", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 404,
      json: () => Promise.resolve({
        error: "Session abc123 not found",
        code: "session_not_found",
      }),
    }) as typeof fetch;

    const { deleteSession } = await import("../lib/api");
    await expect(deleteSession("abc123")).rejects.toThrow(
      "Session abc123 not found",
    );
  });

  // Failure path: APIError carries structured category, recovery, and retry fields
  it("createSession throws APIError with category and recovery", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 429,
      json: () => Promise.resolve({
        error: "Session limit reached",
        code: "session_limit_reached",
        category: "resource_limit",
        recovery: "Close an unused terminal session, then retry",
        retry: true,
      }),
    }) as typeof fetch;

    const { createSession, APIError } = await import("../lib/api");
    try {
      await createSession();
      expect.fail("should have thrown");
    } catch (err) {
      expect(err).toBeInstanceOf(APIError);
      const apiErr = err as InstanceType<typeof APIError>;
      expect(apiErr.code).toBe("session_limit_reached");
      expect(apiErr.category).toBe("resource_limit");
      expect(apiErr.recovery).toBe("Close an unused terminal session, then retry");
      expect(apiErr.retry).toBe(true);
      expect(apiErr.status).toBe(429);
    }
  });

  // Failure path: non-JSON response gives APIError with fallback fields
  it("non-JSON response creates APIError with fallback recovery", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 502,
      json: () => Promise.reject(new Error("not json")),
    }) as typeof fetch;

    const { createSession, APIError } = await import("../lib/api");
    try {
      await createSession();
      expect.fail("should have thrown");
    } catch (err) {
      expect(err).toBeInstanceOf(APIError);
      const apiErr = err as InstanceType<typeof APIError>;
      expect(apiErr.status).toBe(502);
      expect(apiErr.retry).toBe(true);
      expect(apiErr.recovery).toBeTruthy();
    }
  });

  // [REQ:P0-004b] api-base WebSocket Integration
  it("buildSessionWsUrl constructs correct WebSocket URL", async () => {
    const { buildSessionWsUrl } = await import("../lib/api");
    const url = buildSessionWsUrl("session-abc");

    expect(url).toContain("/sessions/session-abc/ws");
    expect(url).toMatch(/^ws/);
  });
});

describe("toErrorInfo", () => {
  it("extracts fields from APIError", async () => {
    const { APIError, toErrorInfo } = await import("../lib/api");
    const err = new APIError(429, {
      error: "Limit reached",
      code: "session_limit_reached",
      category: "resource_limit",
      recovery: "Close a session",
      retry: true,
    });
    const info = toErrorInfo(err);
    expect(info.message).toBe("Limit reached");
    expect(info.recovery).toBe("Close a session");
    expect(info.retry).toBe(true);
  });

  it("handles plain Error (no recovery/retry)", async () => {
    const { toErrorInfo } = await import("../lib/api");
    const info = toErrorInfo(new Error("network failed"));
    expect(info.message).toBe("network failed");
    expect(info.recovery).toBeUndefined();
    expect(info.retry).toBeUndefined();
  });

  it("handles non-Error values", async () => {
    const { toErrorInfo } = await import("../lib/api");
    const info = toErrorInfo("string error");
    expect(info.message).toBe("Unknown error");
  });

  it("omits empty recovery/retry from APIError", async () => {
    const { APIError, toErrorInfo } = await import("../lib/api");
    const err = new APIError(400, {
      error: "Bad input",
      code: "invalid_body",
    });
    const info = toErrorInfo(err);
    expect(info.message).toBe("Bad input");
    expect(info.recovery).toBeUndefined();
    expect(info.retry).toBeUndefined();
  });
});
