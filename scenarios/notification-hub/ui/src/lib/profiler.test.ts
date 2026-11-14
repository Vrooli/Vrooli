import { describe, expect, it, vi } from "vitest";

import { onProfilerRender } from "./profiler";

const commit = (id: string, duration: number) => onProfilerRender(id, "update", duration, duration, 0, duration);

describe("onProfilerRender", () => {
  it("records each commit as a user-timing measure named for the subtree and phase", () => {
    const measure = vi.spyOn(performance, "measure").mockImplementation(() => ({}) as PerformanceMeasure);
    commit("Inbox", 12);
    expect(measure).toHaveBeenCalledWith("⚛ Inbox (update)", expect.objectContaining({ duration: 12 }));
    measure.mockRestore();
  });

  it("never lets a browser that rejects the call surface an error", () => {
    const measure = vi.spyOn(performance, "measure").mockImplementation(() => { throw new TypeError("unsupported"); });
    expect(() => commit("Inbox", 5)).not.toThrow();
    measure.mockRestore();
  });
});
