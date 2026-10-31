import { describe, expect, it, vi } from "vitest";

import { AuthClient } from "./AuthClient";

describe("AuthClient", () => {
  it("adds the bearer token without replacing caller headers", async () => {
    const fetchImpl = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
      expect(new Headers(init?.headers).get("Authorization")).toBe("Bearer token");
      expect(new Headers(init?.headers).get("X-Request-ID")).toBe("request-1");
      return new Response(JSON.stringify({ ok: true }), { status: 200 });
    });
    const client = new AuthClient({ baseURL: "https://example.test", getToken: () => "token", fetchImpl });

    await expect(client.request("/status", { headers: { "X-Request-ID": "request-1" } })).resolves.toEqual({ ok: true });
    expect(fetchImpl).toHaveBeenCalledOnce();
  });

  it("rejects non-success responses", async () => {
    const client = new AuthClient({
      baseURL: "https://example.test",
      fetchImpl: vi.fn(async () => new Response(null, { status: 401 })),
    });

    await expect(client.request("status")).rejects.toThrow("authenticated request failed: 401");
  });
});
