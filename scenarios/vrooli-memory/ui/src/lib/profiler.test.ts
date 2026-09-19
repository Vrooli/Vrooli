import { describe, expect, it, vi } from "vitest";

import { onProfilerRender } from "./profiler";

describe("onProfilerRender", () => {
  it("emits a timing measure and tolerates unsupported browsers", () => {
    const measure = vi.spyOn(performance, "measure").mockImplementation(() => ({}) as PerformanceMeasure);
    onProfilerRender("journal", "mount", 4, 10, 0, 10);
    expect(measure).toHaveBeenCalledWith("⚛ journal (mount)", expect.objectContaining({ duration: 4 }));
    measure.mockImplementation(() => { throw new Error("unsupported"); });
    expect(() => onProfilerRender("journal", "update", 2, 12, 10, 12)).not.toThrow();
  });
});
