import { describe, expect, it, vi } from "vitest";
import { createTransitionService, type TransitionClient } from "./transition-service";

describe("transition service", () => {
  it("starts a catalog-only transition without a UI action registry", async () => {
    const listTransitions = vi.fn().mockResolvedValue({
      transitions: [{ key: "temporary.catalog-transition", subject: "temporary_subject" }],
    });
    const startTransition = vi.fn().mockResolvedValue({ executionId: "exec-temporary" });
    const service = createTransitionService({ listTransitions, startTransition } as unknown as TransitionClient);

    await expect(service.start("temporary.catalog-transition", "temporary-1")).resolves.toEqual({ executionId: "exec-temporary" });
    await service.list();

    expect(listTransitions).toHaveBeenCalledTimes(1);
    expect(startTransition).toHaveBeenCalledWith({
      transitionKey: "temporary.catalog-transition",
      subjectRef: { subject: "temporary_subject", value: "temporary-1" },
    });
  });
});
