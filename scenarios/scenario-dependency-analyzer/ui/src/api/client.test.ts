import { beforeEach, describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { AnalysisOperationState, StartOperationResponseSchema, WaitOperationResponseSchema } from "@vrooli/proto-types/scenario-dependency-analyzer/v1/analysis_operations/analysis_operations_pb";

import { analyzeAllScenarios, analysisOperationsClient, fetchAnalysisHealth, fetchDependencyGraph, fetchDeploymentReport } from "./client";

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
  it("reattaches after an observer failure using the same admission key", async () => {
    const start = vi.spyOn(analysisOperationsClient, "startOperation").mockResolvedValue(create(StartOperationResponseSchema, { operation: { id: "persisted" } }));
    const wait = vi.spyOn(analysisOperationsClient, "waitOperation")
      .mockRejectedValueOnce(new Error("connection lost"))
      .mockResolvedValue(create(WaitOperationResponseSchema, { operation: { id: "persisted", state: AnalysisOperationState.SUCCEEDED } }));
    await expect(analyzeAllScenarios()).rejects.toThrow("connection lost");
    await expect(analyzeAllScenarios()).resolves.toBeUndefined();
    expect(start).toHaveBeenCalledTimes(2);
    expect(start.mock.calls[0]?.[0]?.clientOperationId).toBeTruthy();
    expect(start.mock.calls[1]?.[0]?.clientOperationId).toBe(start.mock.calls[0]?.[0]?.clientOperationId);
    expect(wait).toHaveBeenCalledTimes(2);
    expect(wait).toHaveBeenLastCalledWith({ operationId: "persisted" });
    expect(fetch).not.toHaveBeenCalled();
    start.mockRestore(); wait.mockRestore();
  });

  it("reports partial analysis and requires a new key for a deliberate rerun", async () => {
    const start = vi.spyOn(analysisOperationsClient, "startOperation").mockResolvedValue(create(StartOperationResponseSchema, { operation: { id: "partial" } }));
    const wait = vi.spyOn(analysisOperationsClient, "waitOperation")
      .mockResolvedValueOnce(create(WaitOperationResponseSchema, { operation: { id: "partial", state: AnalysisOperationState.PARTIAL, error: "alpha could not be read" } }))
      .mockResolvedValue(create(WaitOperationResponseSchema, { operation: { id: "next", state: AnalysisOperationState.SUCCEEDED } }));
    await expect(analyzeAllScenarios()).rejects.toThrow("alpha could not be read");
    await expect(analyzeAllScenarios()).resolves.toBeUndefined();
    expect(start.mock.calls[0]?.[0]?.clientOperationId).toBeTruthy();
    expect(start.mock.calls[1]?.[0]?.clientOperationId).not.toBe(start.mock.calls[0]?.[0]?.clientOperationId);
    start.mockRestore(); wait.mockRestore();
  });

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
