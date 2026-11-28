import { afterEach, describe, expect, it, vi } from "vitest";

import { assessChangeIntent } from "./changeIntent";

describe("assessChangeIntent", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("keeps the request scenario bounded and caps the response size", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ results: [], limitations: [] }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    const response = await assessChangeIntent("change request", 100);
    expect(response.scenario).toBe("business-health");
    expect(response.results).toEqual([]);
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(String(url)).toContain("/api/v1/change-intent/assess");
    expect(JSON.parse(String(init.body))).toEqual({ scenario: "business-health", intent: "change request", limit: 20 });
  });
});
