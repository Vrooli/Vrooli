import { describe, expect, it, vi } from "vitest";
import { onProfilerRender } from "./profiler";

describe("onProfilerRender", () => {
  it("records a commit measurement", () => {
    const measure = vi.spyOn(performance, "measure").mockImplementation(() => undefined as never);

    onProfilerRender("vault", "update", 12, 12, 0, 12);

    expect(measure).toHaveBeenCalledWith("⚛ vault (update)", expect.objectContaining({ duration: 12 }));
    measure.mockRestore();
  });

  it("does not allow measurement failures to escape", () => {
    const measure = vi.spyOn(performance, "measure").mockImplementation(() => {
      throw new Error("measurement unavailable");
    });

    expect(() => onProfilerRender("vault", "mount", 4, 4, 0, 4)).not.toThrow();
    measure.mockRestore();
  });
});
