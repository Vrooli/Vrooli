import { describe, expect, it, vi } from "vitest";
import { onProfilerRender } from "./profiler";

describe("onProfilerRender", () => {
  it("records a measurement for a completed render", () => {
    const measure = vi.spyOn(performance, "measure").mockImplementation(() => undefined as never);

    onProfilerRender("wizard", "update", 12, 8, 4, 1);

    expect(measure).toHaveBeenCalledWith(
      "⚛ wizard (update)",
      expect.objectContaining({ duration: 12 }),
    );
    measure.mockRestore();
  });

  it("does not let measurement failures affect onboarding", () => {
    const measure = vi.spyOn(performance, "measure").mockImplementation(() => {
      throw new Error("unsupported");
    });

    expect(() => onProfilerRender("wizard", "mount", 4, 4, 2, 1)).not.toThrow();
    measure.mockRestore();
  });
});
