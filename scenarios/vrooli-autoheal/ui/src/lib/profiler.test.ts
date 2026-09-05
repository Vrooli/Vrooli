import { describe, expect, it, vi } from "vitest";
import { onProfilerRender } from "./profiler";

describe("profiler instrumentation", () => {
  it("does not let measurement failures affect rendering", () => {
    const measure = vi.spyOn(performance, "measure").mockImplementation(() => {
      throw new Error("measurement unavailable");
    });
    expect(() => onProfilerRender("test", "mount", 1, 0, 0, 0)).not.toThrow();
    expect(measure).toHaveBeenCalled();
    measure.mockRestore();
  });
});
