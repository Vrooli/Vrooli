import { describe, it, expect, vi, beforeEach } from "vitest";

// [REQ:P1-001a] Expiration Policy Engine - client types
// [REQ:P1-001b] Policy Configuration UI

describe("Session Policy API client", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("getSessionPolicy calls correct endpoint", async () => {
    const mockResponse = {
      session_id: "test-123",
      policy: { mode: "never" },
      expires_at: null,
      ttl_seconds: null,
    };

    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      }),
    );

    const { getSessionPolicy } = await import("../lib/api");
    const result = await getSessionPolicy("test-123");

    expect(result.session_id).toBe("test-123");
    expect(result.policy.mode).toBe("never");
    expect(fetch).toHaveBeenCalledOnce();
  });

  it("updateSessionPolicy sends PUT with policy body", async () => {
    const mockResponse = {
      session_id: "test-123",
      policy: { mode: "preset", duration: "8h" },
      expires_at: "2026-02-20T00:00:00Z",
      ttl_seconds: 28800,
    };

    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      }),
    );

    const { updateSessionPolicy } = await import("../lib/api");
    const result = await updateSessionPolicy("test-123", {
      mode: "preset",
      duration: "8h",
    });

    expect(result.policy.mode).toBe("preset");
    expect(result.policy.duration).toBe("8h");
    expect(result.ttl_seconds).toBe(28800);

    const call = vi.mocked(fetch).mock.calls[0];
    expect(call).toBeDefined();
    const [, opts] = call ?? [];
    expect(opts?.method).toBe("PUT");
    const body = JSON.parse(opts?.body as string) as Record<string, unknown>;
    expect(body.mode).toBe("preset");
    expect(body.duration).toBe("8h");
  });

  it("updateSessionPolicy throws APIError on failure", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 400,
        json: () =>
          Promise.resolve({
            error: "Invalid policy",
            code: "invalid_policy",
            category: "validation",
            recovery: "Use a valid mode",
          }),
      }),
    );

    const { updateSessionPolicy, APIError } = await import("../lib/api");

    await expect(
      updateSessionPolicy("test-123", { mode: "bad" }),
    ).rejects.toThrow(APIError);
  });

  it("getSessionPolicy throws APIError on 404", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 404,
        json: () =>
          Promise.resolve({
            error: "Session not found",
            code: "session_not_found",
            category: "validation",
          }),
      }),
    );

    const { getSessionPolicy, APIError } = await import("../lib/api");

    await expect(getSessionPolicy("nonexistent")).rejects.toThrow(APIError);
  });
});

describe("Policy types", () => {
  it("SessionInfo includes policy field", async () => {
    // Construct a valid SessionInfo to verify shape includes policy
    const session: import("../lib/api").SessionInfo = {
      id: "test",
      shell: "/bin/sh",
      created_at: "2026-01-01T00:00:00Z",
      cols: 80,
      rows: 24,
      policy: { mode: "never" },
    };
    expect(session.policy.mode).toBe("never");
  });

  it("PolicyResponse includes TTL fields", async () => {
    const resp: import("../lib/api").PolicyResponse = {
      session_id: "test",
      policy: { mode: "preset", duration: "1h" },
      expires_at: "2026-01-01T01:00:00Z",
      ttl_seconds: 3600,
    };
    expect(resp.ttl_seconds).toBe(3600);
    expect(resp.expires_at).toBeDefined();
  });
});

describe("SessionsModal policy controls", () => {
  it("sessions modal module exports default component", async () => {
    const mod = await import("../components/SessionsModal");
    expect(typeof mod.default).toBe("function");
  });
});
