import { afterEach, describe, expect, it, vi } from "vitest";

import { matchInvariantCandidateReceipt, searchInvariantCandidates } from "./invariantCandidates";

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
    expect(JSON.parse(new TextDecoder().decode(init.body as Uint8Array))).toMatchObject({ query: "cache bound", scenario: "search-hub", path: "api/cache", limit: 10 });
  });
});

describe("matchInvariantCandidateReceipt", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("sends one receipt and the exact candidate source identity to the owner proxy", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      status: "EXACT_MATCH",
      receiptId: "receipt-1",
      candidate: { candidateId: "code-facts:invariant-candidate:demo:runtime_guard:api/server.go", scenario: "demo", relativePath: "api/server.go", sourceLine: 42, sourceFileSha256: "a".repeat(64) },
      reason: "exact identity match",
    }), { status: 200, headers: { "Content-Type": "application/json" } }));
    vi.stubGlobal("fetch", fetchMock);

    const candidate = { candidateId: "code-facts:invariant-candidate:demo:runtime_guard:api/server.go", scenario: "demo", relativePath: "api/server.go", sourceLine: 42, sourceFileSha256: "a".repeat(64) };
    const result = await matchInvariantCandidateReceipt("receipt-1", candidate);

    expect(result.status).toBe("EXACT_MATCH");
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(String(url)).toContain("/api/v1/invariants/candidate-receipt-match");
    expect(JSON.parse(String(init.body))).toEqual({ receiptId: "receipt-1", candidate });
  });
});
