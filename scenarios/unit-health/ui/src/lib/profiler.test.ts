import { afterEach, describe, expect, it, vi } from "vitest";

import { onProfilerRender } from "./profiler";

afterEach(() => vi.unstubAllGlobals());

describe("profiler timing boundary", () => {
  it("records the component and phase with the observed render duration", () => {
    const measure = vi.fn();
    vi.stubGlobal("performance", { now: () => 120, measure });

    onProfilerRender("validation", "update", 7, 9, 100, 120);

    expect(measure).toHaveBeenCalledTimes(1);
    expect(measure).toHaveBeenCalledWith("⚛ validation (update)", {
      start: 113,
      duration: 7,
    });
  });

  it("does not break rendering when the browser rejects measurement options", () => {
    const failure = new TypeError("unsupported measurement options");
    const measure = vi.fn(() => { throw failure; });
    vi.stubGlobal("performance", { now: () => 120, measure });

    expect(() => onProfilerRender("shell", "mount", 3, 9, 100, 120)).not.toThrow();
    expect(measure).toHaveBeenCalledTimes(1);
    expect(measure).toHaveBeenCalledWith("⚛ shell (mount)", {
      start: 117,
      duration: 3,
    });
  });
});
