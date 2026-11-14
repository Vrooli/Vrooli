import { describe, expect, it, vi } from "vitest";
import { onProfilerRender } from "./profiler";

describe("onProfilerRender", () => {
  it("emits a user timing measurement without surfacing browser failures", () => {
    const measure = vi.spyOn(performance, "measure").mockImplementation(() => undefined as never);
    onProfilerRender("planner", "mount", 12, 0, 0, 0);
    expect(measure).toHaveBeenCalledWith("⚛ planner (mount)", expect.objectContaining({ duration: 12 }));
    measure.mockRestore();
  });

  it("swallows browsers that reject the options-form measurement", () => {
    const measure = vi.spyOn(performance, "measure").mockImplementation(() => { throw new TypeError("unsupported"); });
    expect(() => onProfilerRender("planner", "update", 4, 0, 0, 0)).not.toThrow();
    measure.mockRestore();
  });
});
