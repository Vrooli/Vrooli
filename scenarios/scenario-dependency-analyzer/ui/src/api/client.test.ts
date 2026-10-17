import { beforeEach, describe, expect, it, vi } from "vitest";

import { fetchAnalysisHealth, fetchDependencyGraph, fetchDeploymentReport } from "./client";

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

  it("forwards deployment cancellation and does not report a deliberate abort as an API outage", async () => {
    const controller = new AbortController();
    controller.abort();
    vi.mocked(fetch).mockRejectedValueOnce(new DOMException("cancelled", "AbortError"));
    const errors = vi.spyOn(console, "error").mockImplementation(() => {});
    await expect(fetchDeploymentReport("alpha", controller.signal)).resolves.toBeNull();
    expect(fetch).toHaveBeenCalledWith(expect.stringContaining("/scenarios/alpha/deployment"), { signal: controller.signal });
    expect(errors).not.toHaveBeenCalled();
    errors.mockRestore();
  });

  it("retains diagnostics and the unavailable-report result for an actual deployment failure", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(jsonResponse({ error: "down" }, 503));
    const errors = vi.spyOn(console, "error").mockImplementation(() => {});
    await expect(fetchDeploymentReport("alpha")).resolves.toBeNull();
    expect(errors).toHaveBeenCalledWith(expect.stringContaining("alpha"), expect.objectContaining({ message: "Scenario API request failed (503)" }));
    errors.mockRestore();
  });
});
