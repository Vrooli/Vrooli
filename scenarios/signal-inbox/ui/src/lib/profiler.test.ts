import { afterEach, describe, expect, it, vi } from "vitest";

import { onProfilerRender } from "./profiler";

describe("onProfilerRender", () => {
  afterEach(() => vi.restoreAllMocks());

  it("emits a commit measurement", () => {
    const measure = vi.spyOn(performance, "measure").mockImplementation(() => ({}) as PerformanceMeasure);
    vi.spyOn(performance, "now").mockReturnValue(100);
    onProfilerRender("SignalList", "mount", 12, 0, 0, 0);
    expect(measure).toHaveBeenCalledWith("⚛ SignalList (mount)", { start: 88, duration: 12 });
  });

  it("swallows measurement failures", () => {
    vi.spyOn(performance, "measure").mockImplementation(() => { throw new Error("unsupported"); });
    expect(() => onProfilerRender("SignalList", "update", 1, 0, 0, 0)).not.toThrow();
  });
});
