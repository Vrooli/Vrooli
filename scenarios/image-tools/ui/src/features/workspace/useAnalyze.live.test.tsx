/**
 * Production `liveAnalyzeClient` test. Analyze shares its model methods with
 * `liveEnhanceClient`; the only method it defines itself is `analyze`, which
 * delegates to the sync analysis edge (`api/analysis.analyze`). This test mocks
 * that edge to assert the delegation. (The hook's state machine is covered with
 * a fully-stubbed seam in `useAnalyze.test.tsx`.)
 */
import { afterEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  analyze: vi.fn(),
}));

vi.mock("../../api/analysis", async (importActual) => {
  const actual = await importActual<typeof import("../../api/analysis")>();
  return { ...actual, analyze: mocks.analyze };
});

import { liveAnalyzeClient } from "./useAnalyze";

const PNG_FILE = new File(["bytes"], "in.png", { type: "image/png" });

afterEach(() => {
  vi.clearAllMocks();
});

describe("liveAnalyzeClient", () => {
  it("analyze delegates op + file to the sync analysis edge and returns its result", async () => {
    const probe = {
      kind: "probe" as const,
      jobId: "j1",
      width: 640,
      height: 480,
      format: "png",
      colorModel: "rgba",
      hasAlpha: true,
      frameCount: 1,
      megapixels: 0.31,
      sizeBytes: 12_345,
      hasExif: false,
      hasGps: false,
      orientation: 0,
      dominantColors: [],
    };
    mocks.analyze.mockResolvedValueOnce(probe);

    const out = await liveAnalyzeClient.analyze("probe", PNG_FILE);

    expect(mocks.analyze).toHaveBeenCalledWith("probe", PNG_FILE);
    expect(out).toBe(probe);
  });

  it("reuses the shared liveEnhanceClient model methods", () => {
    expect(typeof liveAnalyzeClient.selectModel).toBe("function");
    expect(typeof liveAnalyzeClient.install).toBe("function");
    expect(typeof liveAnalyzeClient.waitJob).toBe("function");
  });
});
