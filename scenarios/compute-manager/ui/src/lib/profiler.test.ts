import { afterEach, describe, expect, it, vi } from "vitest";

import { onProfilerRender } from "./profiler";

describe("onProfilerRender", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("emits a user-timing measurement aligned to the commit", () => {
    const measure = vi.spyOn(performance, "measure").mockImplementation(() => undefined as unknown as PerformanceMeasure);
    vi.spyOn(performance, "now").mockReturnValue(120);

    onProfilerRender("dashboard", "mount", 20, 0, 0, 0);

    expect(measure).toHaveBeenCalledWith("⚛ dashboard (mount)", { start: 100, duration: 20 });
  });

  it("swallows browsers that reject the measurement options", () => {
    const measure = vi.spyOn(performance, "measure").mockImplementation(() => { throw new Error("unsupported"); });

    expect(() => onProfilerRender("dashboard", "update", 1, 0, 0, 0)).not.toThrow();
    expect(measure).toHaveBeenCalledTimes(1);
  });
});
