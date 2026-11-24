import { afterEach, describe, expect, it, vi } from "vitest";

import { searchInvariantCandidates } from "./invariantCandidates";

describe("searchInvariantCandidates", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("sends an explicit scenario subtree and caps results at ten", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      query: "cache bound",
      scenario: "search-hub",
      scope: "api/cache",
      results: [],
      degradations: [],
    }), { status: 200, headers: { "Content-Type": "application/json" } }));
    vi.stubGlobal("fetch", fetchMock);

    const result = await searchInvariantCandidates("cache bound", "search-hub", "api/cache", 50);

    expect(result.results).toEqual([]);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(String(url)).toContain("/vrooli.code_facts.v1.facts.CodeFactsService/SearchInvariantCandidates");
    expect(JSON.parse(String(init.body))).toMatchObject({ query: "cache bound", scenario: "search-hub", path: "api/cache", limit: 10 });
  });
});
