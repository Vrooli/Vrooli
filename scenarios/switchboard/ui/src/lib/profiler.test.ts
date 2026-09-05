import { describe, expect, it, vi } from "vitest";

import { onProfilerRender } from "./profiler";

describe("onProfilerRender", () => {
  it("records a user-timing measure", () => {
    const measure = vi.spyOn(performance, "measure").mockReturnValue({} as PerformanceMeasure);
    onProfilerRender("surface", "mount", 4, 4, 0, 4);
    expect(measure).toHaveBeenCalledWith("⚛ surface (mount)", expect.objectContaining({ duration: 4 }));
    measure.mockRestore();
  });

  it("swallows browsers that reject the measure options", () => {
    const measure = vi.spyOn(performance, "measure").mockImplementation(() => {
      throw new Error("unsupported");
    });
    expect(() => onProfilerRender("surface", "update", 1, 1, 0, 1)).not.toThrow();
    measure.mockRestore();
  });
});
