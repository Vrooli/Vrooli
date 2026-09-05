import { afterEach, describe, expect, it, vi } from "vitest";

import { onProfilerRender } from "./profiler";

describe("onProfilerRender", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("emits a user-timing measure and swallows unsupported implementations", () => {
    const now = vi.spyOn(performance, "now").mockReturnValue(100);
    const measure = vi.spyOn(performance, "measure").mockImplementation(() => undefined as PerformanceMeasure);

    onProfilerRender("Panel", "mount", 12, 0, 0, 0);

    expect(now).toHaveBeenCalled();
    expect(measure).toHaveBeenCalledWith("⚛ Panel (mount)", {
      start: 88,
      duration: 12,
    });

    measure.mockImplementation(() => {
      throw new Error("unsupported");
    });
    expect(() => onProfilerRender("Panel", "update", 5, 0, 0, 0)).not.toThrow();
  });
});
