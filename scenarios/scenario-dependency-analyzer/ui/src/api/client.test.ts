import { beforeEach, describe, expect, it, vi } from "vitest";

import { fetchAnalysisHealth, fetchDependencyGraph } from "./client";

function jsonResponse(payload: unknown, status = 200): Response {
  return new Response(JSON.stringify(payload), {
    status,
    headers: { "Content-Type": "application/json" }
  });
}

beforeEach(() => {
  vi.spyOn(globalThis, "fetch").mockResolvedValue(
    jsonResponse({
      id: "graph",
      graph_type: "combined",
      nodes: [],
      edges: []
    })
  );
});

describe("API client", () => {
  it("loads dependency graphs through the shared URL builder", async () => {
    await expect(fetchDependencyGraph("combined")).resolves.toMatchObject({
      id: "graph",
      graph_type: "combined"
    });

    expect(fetch).toHaveBeenCalledWith(expect.stringContaining("/graph/combined"), {
      signal: undefined
    });
  });

  it("surfaces non-2xx responses as errors", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(jsonResponse({ error: "down" }, 503));

    await expect(fetchAnalysisHealth()).rejects.toThrow("Scenario API request failed (503)");
  });
});
