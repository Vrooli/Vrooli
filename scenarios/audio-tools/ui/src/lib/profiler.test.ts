import { describe, it, expect, vi, afterEach } from "vitest";

import { onProfilerRender } from "./profiler";

afterEach(() => {
  vi.restoreAllMocks();
});

describe("onProfilerRender", () => {
  it("calls performance.measure with the correct id/phase label and timing", () => {
    const measureSpy = vi.spyOn(performance, "measure").mockImplementation(() => ({
      name: "dummy",
      entryType: "measure",
      startTime: 0,
      duration: 0,
      detail: null,
      toJSON() { return {}; },
    }));

    onProfilerRender(
      "AppRoot",
      "mount",
      12.5,  // actualDuration
      50,    // baseDuration (unused by the callback)
      100,   // startTime (unused)
      200,   // commitTime (unused)
    );

    expect(measureSpy).toHaveBeenCalledTimes(1);
    const [label, opts] = measureSpy.mock.calls[0] as [string, { start: number; duration: number }];
    expect(label).toBe("⚛ AppRoot (mount)");
    // duration should equal actualDuration
    expect(opts.duration).toBe(12.5);
    // start = performance.now() - actualDuration; we can't predict the exact
    // value but it should be a non-negative number
    expect(opts.start).toBeGreaterThanOrEqual(0);
  });

  it("catches errors from performance.measure without throwing", () => {
    vi.spyOn(performance, "measure").mockImplementation(() => {
      throw new RangeError("bad measure options");
    });
    // Must not throw
    expect(() =>
      onProfilerRender("Comp", "update", 5, 10, 0, 0),
    ).not.toThrow();
  });

  it("uses the phase argument in the label string", () => {
    const measureSpy = vi.spyOn(performance, "measure").mockImplementation(() => ({
      name: "",
      entryType: "measure",
      startTime: 0,
      duration: 0,
      detail: null,
      toJSON() { return {}; },
    }));
    onProfilerRender("Nav", "update", 3, 3, 0, 0);
    const [label] = measureSpy.mock.calls[0] as [string];
    expect(label).toContain("update");
  });
});
