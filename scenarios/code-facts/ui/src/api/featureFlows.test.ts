import { afterEach, describe, expect, it, vi } from "vitest";

import { searchFeatureFlows } from "./featureFlows";

describe("searchFeatureFlows", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("defaults omitted optional degradations and preserves the explicit scope", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      query: "request handler",
      scenario: "search-hub",
      results: [],
      // The Go response uses omitempty for degradations on success.
    }), { status: 200, headers: { "Content-Type": "application/json" } }));
    vi.stubGlobal("fetch", fetchMock);

    const response = await searchFeatureFlows("request handler", "search-hub", "api", 10);

    expect(response.degradations).toEqual([]);
    expect(response.results).toEqual([]);
    expect(response.path).toBe("api");
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toContain("/api/v1/flows/search");
    expect(JSON.parse(String(init.body))).toEqual({ query: "request handler", scenario: "search-hub", path: "api", limit: 5 });
  });
});
