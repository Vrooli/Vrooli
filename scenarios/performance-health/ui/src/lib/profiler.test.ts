import { describe, expect, it, vi, afterEach } from "vitest";

import { onProfilerRender } from "./profiler";

describe("onProfilerRender", () => {
  afterEach(() => vi.restoreAllMocks());

  it("emits a ⚛-prefixed user-timing measure aligned to the commit window", () => {
    const measure = vi.spyOn(performance, "measure").mockReturnValue({} as PerformanceMeasure);
    onProfilerRender("List", "mount", 12, 12, 0, 12);
    expect(measure).toHaveBeenCalledWith(
      "⚛ List (mount)",
      expect.objectContaining({ duration: 12 }),
    );
  });

  it("never throws when performance.measure rejects the object form", () => {
    vi.spyOn(performance, "measure").mockImplementation(() => {
      throw new Error("legacy browser");
    });
    expect(() => onProfilerRender("List", "update", 4, 4, 0, 4)).not.toThrow();
  });
});
