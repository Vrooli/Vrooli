import { afterEach, describe, expect, it, vi } from "vitest";

import { onProfilerRender } from "./profiler";

describe("onProfilerRender", () => {
  afterEach(() => vi.restoreAllMocks());

  it("emits a user_timing measure aligned to the commit window", () => {
    const measure = vi
      .spyOn(performance, "measure")
      .mockImplementation(() => ({}) as PerformanceMeasure);
    vi.spyOn(performance, "now").mockReturnValue(1000);

    // Signature: (id, phase, actualDuration, baseDuration, startTime, commitTime).
    onProfilerRender("AppShell", "update", 12, 12, 0, 0);

    expect(measure).toHaveBeenCalledWith("⚛ AppShell (update)", {
      start: 988, // now() - actualDuration
      duration: 12,
    });
  });

  it("never throws when performance.measure rejects the object form", () => {
    vi.spyOn(performance, "measure").mockImplementation(() => {
      throw new Error("legacy browser");
    });

    // The catch block must swallow the error so a measurement failure never
    // surfaces in the running app.
    expect(() => onProfilerRender("AppShell", "mount", 5, 5, 0, 0)).not.toThrow();
  });
});
