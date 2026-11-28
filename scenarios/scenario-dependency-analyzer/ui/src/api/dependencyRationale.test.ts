import { afterEach, describe, expect, it, vi } from "vitest";

import { searchDependencyRationale } from "./dependencyRationale";

describe("searchDependencyRationale", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("sends an exact one-scenario scope and caps the response count", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      results: [], scan: { basis: "canonical_service_manifest", scenariosScanned: 1, scenariosFailed: 0, complete: true, runtimeStatusKnown: false }, degradations: [],
    }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    const response = await searchDependencyRationale("runtime dependency purpose", "web-console", 100);
    expect(response.scenario).toBe("web-console");
    expect(response.scan.runtimeStatusKnown).toBe(false);
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(String(url)).toContain("/dependencies/rationale/search");
    expect(JSON.parse(String(init.body))).toEqual({ query: "runtime dependency purpose", scenario: "web-console", limit: 10 });
  });
});
