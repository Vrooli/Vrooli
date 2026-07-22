import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { workflowsClient } from "./workflows";

describe("api/workflows", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("normalizes an absent workflows field to an empty list", async () => {
    fetchSpy.mockResolvedValueOnce(
      new Response("{}", {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );

    await expect(workflowsClient.listWorkflows({ activeOnly: false })).resolves.toMatchObject({ workflows: [] });
  });

  it("forwards workflow mutations and promotion reads through the Connect client", async () => {
    for (let index = 0; index < 4; index += 1) {
      fetchSpy.mockResolvedValueOnce(new Response("{}", { status: 200, headers: { "content-type": "application/json" } }));
    }

    await workflowsClient.startWorkflow({ kind: 2, assetId: "asset-1", targetScenario: "demo", idempotencyKey: "start-1" });
    await workflowsClient.stopWorkflow({ id: "workflow-1" });
    await workflowsClient.retryWorkflow({ id: "workflow-1", idempotencyKey: "retry-1" });
    await workflowsClient.getPromotionReadiness({ assetId: "asset-1", originScenario: "demo" });

    expect(fetchSpy).toHaveBeenCalledTimes(4);
  });
});
