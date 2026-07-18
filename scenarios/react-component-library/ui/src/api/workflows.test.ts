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
});
