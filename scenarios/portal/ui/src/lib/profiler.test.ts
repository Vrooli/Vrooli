import { afterEach, describe, expect, it, vi } from "vitest";

import { onProfilerRender } from "./profiler";

describe("onProfilerRender", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("emits a user-timing measure for profiler commits", () => {
    const now = vi.spyOn(performance, "now").mockReturnValue(50);
    const measure = vi.spyOn(performance, "measure").mockImplementation(() => ({} as PerformanceMeasure));

    onProfilerRender("chat", "update", 12, 14, 40, 50);

    expect(now).toHaveBeenCalled();
    expect(measure).toHaveBeenCalledWith("⚛ chat (update)", {
      start: 38,
      duration: 12,
    });
  });

  it("swallows browsers that reject object-form performance.measure", () => {
    vi.spyOn(performance, "now").mockReturnValue(50);
    vi.spyOn(performance, "measure").mockImplementation(() => {
      throw new Error("unsupported");
    });

    expect(() => onProfilerRender("chat", "mount", 4, 4, 46, 50)).not.toThrow();
  });
});
