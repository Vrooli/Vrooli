import { afterEach, describe, expect, it, vi } from "vitest";

import { onProfilerRender } from "./profiler";

describe("onProfilerRender", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("emits a user_timing measure aligned to the commit window", () => {
    const measure = vi.spyOn(performance, "measure").mockImplementation(() => ({}) as PerformanceMeasure);
    vi.spyOn(performance, "now").mockReturnValue(1000);

    onProfilerRender("Subtree", "update", 12, 12, 0, 12);

    expect(measure).toHaveBeenCalledWith("⚛ Subtree (update)", {
      start: 1000 - 12,
      duration: 12,
    });
  });

  it("swallows errors from browsers that reject the object form", () => {
    vi.spyOn(performance, "measure").mockImplementation(() => {
      throw new Error("legacy browser");
    });

    expect(() => onProfilerRender("X", "mount", 4, 4, 0, 4)).not.toThrow();
  });
});
