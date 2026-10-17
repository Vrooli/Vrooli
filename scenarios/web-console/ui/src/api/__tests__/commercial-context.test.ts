import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  _resetCommercialContextCache,
  deleteSubscriptionSession,
  getCommercialContext,
  getConnections,
  getSubscriptionSession,
  getSubscriptionSummary,
  isCommercialContentVisible,
  provisionOpenRouterKey,
  provisionSubscriptionSession,
  removeOpenRouterKey,
  safeCommercialDestination,
  testOpenRouterKey,
} from "../monetization";

const context = {
  account: { subscription_status: "active", plan_tier: "free", credit_balance: 4, entitlement_ids: [], evaluated_at: "now" },
  content: [{ content_id: "connect", placement: "integrations", title: "Connect", description: "Connect an account", priority: "contextual", eligible: true, cta_label: "Manage account", cta_destination: "/account", expires_at: "later", dismissible: true }],
  generated_at: "now",
  stale_after: "later",
  source: "landing-page-business-suite",
};

function signIn(token: string): void {
  sessionStorage.setItem("vrooli.web.access-token", JSON.stringify({ accessToken: token, expiresAt: "2099-01-01T00:00:00Z" }));
}

describe("commercial context client", () => {
  beforeEach(() => {
    _resetCommercialContextCache();
    sessionStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("deduplicates concurrent requests and caches safe content", async () => {
    signIn("token-a");
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(context), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    const [first, second] = await Promise.all([
      getCommercialContext("integrations", "audio-tools"),
      getCommercialContext("integrations", "audio-tools"),
    ]);
    await getCommercialContext("integrations", "audio-tools");

    expect(first.content[0]?.content_id).toBe("connect");
    expect(second).toEqual(first);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("does not reuse a cached response after identity changes", async () => {
    signIn("token-a");
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify(context), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify(context), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    await getCommercialContext();
    signIn("token-b");
    await getCommercialContext();
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("deduplicates by request key rather than sharing different capabilities", async () => {
    signIn("token-a");
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ ...context, source: "capability-a" }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ ...context, source: "capability-b" }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    const [first, second] = await Promise.all([
      getCommercialContext("integrations", "capability-a"),
      getCommercialContext("integrations", "capability-b"),
    ]);

    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(first.source).toBe("capability-a");
    expect(second.source).toBe("capability-b");
  });

  it("fails closed without an account", async () => {
    await expect(getCommercialContext()).rejects.toThrow("requires an account");
  });

  it("fails closed for expired, invalid, misplaced, or actively dismissed content", () => {
    const now = Date.parse("2026-01-01T00:00:00Z");
    const valid = { ...context.content[0]!, expires_at: "2026-01-01T00:05:00Z" };
    expect(isCommercialContentVisible(valid, "integrations", now)).toBe(true);
    expect(isCommercialContentVisible({ ...valid, expires_at: "2025-12-31T23:59:00Z" }, "integrations", now)).toBe(false);
    expect(isCommercialContentVisible({ ...valid, expires_at: "not-a-date" }, "integrations", now)).toBe(false);
    expect(isCommercialContentVisible({ ...valid, placement: "account" }, "integrations", now)).toBe(false);
    expect(isCommercialContentVisible({ ...valid, dismissed_until: "2026-01-01T00:10:00Z" }, "integrations", now)).toBe(false);
  });

  it("accepts only same-origin or owned LPBS destinations", () => {
    expect(safeCommercialDestination("/account")).toBe(`${window.location.origin}/account`);
    expect(safeCommercialDestination("https://vrooli.com/account")).toBe("https://vrooli.com/account");
    expect(safeCommercialDestination("https://evil.example/phish")).toBeNull();
    expect(safeCommercialDestination("javascript:alert(1)")).toBeNull();
  });
});

describe("subscription and credential REST wrappers", () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  function stubFetch(responder: (path: string, init?: RequestInit) => Response): ReturnType<typeof vi.fn> {
    const mock = vi.fn((path: string, init?: RequestInit) => Promise.resolve(responder(path, init)));
    vi.stubGlobal("fetch", mock);
    return mock;
  }

  it("fetches subscription session status with an auth header when signed in", async () => {
    sessionStorage.setItem(
      "vrooli.web.access-token",
      JSON.stringify({ accessToken: "tok", expiresAt: "2099-01-01T00:00:00Z" }),
    );
    const mock = stubFetch(() => new Response(JSON.stringify({ configured: true, provider_state: "ok" }), { status: 200 }));
    await expect(getSubscriptionSession()).resolves.toEqual({ configured: true, provider_state: "ok" });
    expect(mock).toHaveBeenCalledWith("/api/v1/auth/subscription/session", expect.anything());
    const headers = mock.mock.calls[0]![1]!.headers as Headers;
    expect(headers.get("Authorization")).toBe("Bearer tok");
    expect(headers.get("Content-Type")).toBe("application/json");
  });

  it("normalizes a 401 summary into a truthful signed-out shape", async () => {
    stubFetch(() => new Response("unauthorized", { status: 401 }));
    await expect(getSubscriptionSummary()).resolves.toEqual({
      configured: false,
      status: "signed_out",
      plan_tier: "free",
      pending_sync: 0,
    });
  });

  it("returns the server summary when authorized", async () => {
    stubFetch(() => new Response(JSON.stringify({ configured: true, status: "active", plan_tier: "pro" }), { status: 200 }));
    await expect(getSubscriptionSummary()).resolves.toMatchObject({ status: "active", plan_tier: "pro" });
  });

  it("surfaces non-401 summary failures to the caller", async () => {
    stubFetch(() => new Response("boom", { status: 500 }));
    await expect(getSubscriptionSummary()).rejects.toThrow("Request failed (500)");
  });

  it("provisions and deletes the subscription session with correct verbs and body", async () => {
    const mock = stubFetch(() => new Response(null, { status: 204 }));
    await expect(provisionSubscriptionSession("refresh-1")).resolves.toEqual({});
    expect(mock).toHaveBeenLastCalledWith(
      "/api/v1/auth/subscription/session",
      expect.objectContaining({ method: "POST", body: JSON.stringify({ refresh_token: "refresh-1" }) }),
    );
    await expect(deleteSubscriptionSession()).resolves.toEqual({});
    expect(mock).toHaveBeenLastCalledWith(
      "/api/v1/auth/subscription/session",
      expect.objectContaining({ method: "DELETE" }),
    );
  });

  it("provisions, removes, and tests the OpenRouter key", async () => {
    const mock = stubFetch((path) =>
      path === "/api/v1/credentials/test"
        ? new Response(JSON.stringify({ valid: true, source: "live", checked_at: "now" }), { status: 200 })
        : new Response(null, { status: 204 }),
    );
    await provisionOpenRouterKey("sk-abc");
    expect(mock).toHaveBeenLastCalledWith(
      "/api/v1/credentials/provision",
      expect.objectContaining({ method: "POST", body: JSON.stringify({ identity: "vrooli/openrouter", field: "api-key", value: "sk-abc" }) }),
    );
    await removeOpenRouterKey();
    expect(mock).toHaveBeenLastCalledWith(
      "/api/v1/credentials/provision",
      expect.objectContaining({ method: "DELETE", body: JSON.stringify({ identity: "vrooli/openrouter", field: "api-key" }) }),
    );
    await expect(testOpenRouterKey()).resolves.toEqual({ valid: true, source: "live", checked_at: "now" });
  });

  it("lists managed integration connections", async () => {
    stubFetch(() => new Response(JSON.stringify({ connections: [{ id: "c1", provider: "google", connection_name: "Gmail", status: "connected" }] }), { status: 200 }));
    await expect(getConnections()).resolves.toMatchObject({ connections: [{ id: "c1", status: "connected" }] });
  });
});
