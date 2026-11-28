import { afterEach, describe, expect, it, vi } from "vitest";

import { onProfilerRender } from "./profiler";

describe("onProfilerRender", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("records a user timing measure for React commits", () => {
    const nowSpy = vi.spyOn(performance, "now").mockReturnValue(100);
    const measureSpy = vi.spyOn(performance, "measure").mockImplementation(() => ({} as PerformanceMeasure));

    onProfilerRender("dashboard", "mount", 12, 0, 0, 0);

    expect(nowSpy).toHaveBeenCalled();
    expect(measureSpy).toHaveBeenCalledWith("⚛ dashboard (mount)", {
      start: 88,
      duration: 12,
    });
  });

  it("suppresses browser measurement incompatibilities", () => {
    vi.spyOn(performance, "now").mockReturnValue(100);
    vi.spyOn(performance, "measure").mockImplementation(() => {
      throw new Error("unsupported");
    });

    expect(() => onProfilerRender("dashboard", "update", 3, 0, 0, 0)).not.toThrow();
  });
});
