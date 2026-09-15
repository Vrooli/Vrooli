import { afterEach, describe, expect, it, vi } from "vitest";
import { onProfilerRender } from "./profiler";

describe("onProfilerRender", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("records a portable browser performance measure for each React commit", () => {
    const measure = vi.spyOn(performance, "measure");

    onProfilerRender("generator-form", "update", 14.5, 19.2, 100, 120);

    expect(measure).toHaveBeenCalledWith("⚛ generator-form update", {
      start: 100,
      end: 120,
      detail: { actualDuration: 14.5, baseDuration: 19.2 },
    });
  });

  it("does not fail a desktop UI when the Performance Timeline is unavailable", () => {
    const original = performance.measure;
    Object.defineProperty(performance, "measure", {
      configurable: true,
      value: undefined,
    });

    expect(() => {
      onProfilerRender("generator-form", "mount", 1, 1, 0, 1);
    }).not.toThrow();

    Object.defineProperty(performance, "measure", {
      configurable: true,
      value: original,
    });
  });
});
