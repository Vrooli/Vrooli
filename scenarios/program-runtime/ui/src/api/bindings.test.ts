import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { fetchBindings, fetchUnbound } from "./bindings";

describe("api/bindings", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("decodes the governed registry response", async () => {
    fetchSpy.mockResolvedValueOnce(new Response(JSON.stringify({
      bindings: [{ id: "fixture/list", effect: "read", signature: "FixtureService.ListItems()" }],
    }), { status: 200 }));

    await expect(fetchBindings()).resolves.toHaveLength(1);
    expect(fetchSpy.mock.calls[0]?.[1]).toMatchObject({ method: "POST", cache: "no-store" });
    expect(fetchSpy.mock.calls[0]?.[0]).toMatch(/BindingRegistryService\/ListBindings/);
  });

  it("decodes unbound capabilities", async () => {
    fetchSpy.mockResolvedValueOnce(new Response(JSON.stringify({
      capabilities: [{ scenario: "legacy", command: "run", reason: "UNBOUND_REASON_NO_MANIFEST" }],
    }), { status: 200 }));

    await expect(fetchUnbound()).resolves.toHaveLength(1);
    expect(fetchSpy.mock.calls[0]?.[0]).toMatch(/BindingRegistryService\/ListUnbound/);
  });

  it("returns the typed server error for failed requests", async () => {
    fetchSpy.mockResolvedValueOnce(new Response(JSON.stringify({
      code: "denied",
      message: "governance refused",
    }), { status: 403 }));

    await expect(fetchBindings()).rejects.toMatchObject({ code: "denied", status: 403 });
  });

  it("uses a deterministic fallback for malformed error envelopes", async () => {
    fetchSpy.mockResolvedValueOnce(new Response("not-json", { status: 502 }));

    await expect(fetchUnbound()).rejects.toMatchObject({
      code: "internal",
      status: 502,
    });
  });
});
