import { describe, expect, it, vi } from "vitest";
import { createCaptureService } from "./capture-service";

describe("capture service classification", () => {
  it("uses the generic transition service and reloads the applied capture", async () => {
    const apiClient = {
      get: vi.fn().mockResolvedValue({
        capture: { id: "capture-1", text: "A useful observation", status: "classified" },
      }),
    };
    const transitions = {
      list: vi.fn(),
      start: vi.fn().mockResolvedValue({ executionId: "exec-capture-1" }),
      apply: vi.fn().mockResolvedValue(undefined),
    };
    const service = createCaptureService(apiClient as never, transitions);

    await expect(service.classify("capture-1")).resolves.toEqual({ workflowExecutionId: "exec-capture-1" });
    await expect(service.applyClassification("capture-1", "exec-capture-1")).resolves.toMatchObject({
      id: "capture-1",
      status: "classified",
    });

    expect(transitions.start).toHaveBeenCalledWith("capture.classify", "capture-1");
    expect(transitions.apply).toHaveBeenCalledWith("capture.classify", "exec-capture-1");
    expect(apiClient.get).toHaveBeenCalledWith("/captures/capture-1");
  });
});
