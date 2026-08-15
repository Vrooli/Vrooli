import { afterEach, describe, expect, it, vi } from "vitest";
import { onProfilerRender } from "./profiler";

describe("onProfilerRender", () => {
  afterEach(() => vi.restoreAllMocks());

  it("records a measurement using the render duration", () => {
    const measure = vi.spyOn(performance, "measure").mockImplementation(() => undefined as never);

    onProfilerRender("App", "update", 12.5, 100, 112.5, 0);

    expect(measure).toHaveBeenCalledWith(
      "⚛ App (update)",
      expect.objectContaining({ duration: 12.5 }),
    );
  });

  it("swallows measurement failures so profiling cannot affect the app", () => {
    vi.spyOn(performance, "measure").mockImplementation(() => {
      throw new Error("unsupported performance entry");
    });

    expect(() => onProfilerRender("App", "mount", 1, 0, 1, 0)).not.toThrow();
  });
});
