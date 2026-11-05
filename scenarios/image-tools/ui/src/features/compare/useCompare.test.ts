import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook, waitFor } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import { DiffResultSchema, type DiffResult } from "@vrooli/proto-types/image-tools/v1/diff/diff_pb";

import { DiffMode } from "../../api/diff";
import { useCompare, type CompareClient } from "./useCompare";

const diffResult = (over: Partial<DiffResult> = {}): DiffResult =>
  create(DiffResultSchema, {
    jobId: "job-diff",
    verdict: "different",
    dimensionsMatch: true,
    baseWidth: 100,
    baseHeight: 80,
    compareWidth: 100,
    compareHeight: 80,
    changedPixels: 1234n,
    totalPixels: 8000n,
    changedFraction: 0.154,
    mae: 12.5,
    rmse: 20.1,
    psnr: 28.3,
    phashDistance: 7,
    phashSimilarity: 0.89,
    ssim: 0.74,
    heatmapRef: "out/heat.png",
    warnings: [],
    ...over,
  });

const fakeClient = (over: Partial<CompareClient> = {}): CompareClient => ({
  compare: vi.fn().mockResolvedValue(diffResult()),
  blobUrl: (key: string) => `/api/v1/blobs/${key}`,
  ...over,
});

const BASE = new File(["base"], "base.png", { type: "image/png" });
const COMPARE = new File(["compare"], "compare.png", { type: "image/png" });

beforeEach(() => {
  // jsdom doesn't implement object URLs.
  vi.stubGlobal("URL", {
    ...URL,
    createObjectURL: vi.fn(() => "blob:fake"),
    revokeObjectURL: vi.fn(),
  });
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.clearAllMocks();
});

describe("useCompare", () => {
  it("starts idle and cannot compare until both slots are filled", () => {
    const { result } = renderHook(() => useCompare(fakeClient()));
    expect(result.current.phase).toBe("idle");
    expect(result.current.canCompare).toBe(false);
    expect(result.current.mode).toBe(DiffMode.PIXEL);

    act(() => result.current.setImage("base", BASE));
    expect(result.current.canCompare).toBe(false);
    act(() => result.current.setImage("compare", COMPARE));
    expect(result.current.canCompare).toBe(true);
    expect(result.current.baseUrl).toBe("blob:fake");
    expect(result.current.compareUrl).toBe("blob:fake");
  });

  it("runs a comparison and exposes the result + heat-map url", async () => {
    const client = fakeClient();
    const { result } = renderHook(() => useCompare(client));
    act(() => result.current.setImage("base", BASE));
    act(() => result.current.setImage("compare", COMPARE));

    act(() => result.current.runCompare());
    await waitFor(() => expect(result.current.phase).toBe("ready"));
    expect(result.current.result?.verdict).toBe("different");
    expect(result.current.heatmapUrl).toBe("/api/v1/blobs/out/heat.png");
    expect(client.compare).toHaveBeenCalledWith(
      expect.objectContaining({ base: BASE, compare: COMPARE, mode: DiffMode.PIXEL }),
    );
  });

  it("threads the chosen mode + tolerance into the compare call", async () => {
    const client = fakeClient();
    const { result } = renderHook(() => useCompare(client));
    act(() => result.current.setImage("base", BASE));
    act(() => result.current.setImage("compare", COMPARE));
    act(() => result.current.setMode(DiffMode.PERCEPTUAL));
    act(() => result.current.setTolerance(0.2));

    act(() => result.current.runCompare());
    await waitFor(() => expect(client.compare).toHaveBeenCalled());
    expect(client.compare).toHaveBeenCalledWith(
      expect.objectContaining({ mode: DiffMode.PERCEPTUAL, tolerance: 0.2 }),
    );
  });

  it("does nothing when runCompare is called without both images", () => {
    const client = fakeClient();
    const { result } = renderHook(() => useCompare(client));
    act(() => result.current.setImage("base", BASE));
    act(() => result.current.runCompare());
    expect(client.compare).not.toHaveBeenCalled();
    expect(result.current.phase).toBe("idle");
  });

  it("surfaces a failure as an error and returns to idle", async () => {
    const client = fakeClient({ compare: vi.fn().mockRejectedValue(new Error("boom")) });
    const { result } = renderHook(() => useCompare(client));
    act(() => result.current.setImage("base", BASE));
    act(() => result.current.setImage("compare", COMPARE));
    act(() => result.current.runCompare());
    await waitFor(() => expect(result.current.error).toBe("boom"));
    expect(result.current.phase).toBe("idle");
  });

  it("ignores a superseded run — only the latest result commits", async () => {
    let resolveFirst: (v: DiffResult) => void = () => {};
    const first = new Promise<DiffResult>((res) => {
      resolveFirst = res;
    });
    const client = fakeClient({
      compare: vi
        .fn()
        .mockReturnValueOnce(first)
        .mockResolvedValueOnce(diffResult({ verdict: "similar" })),
    });
    const { result } = renderHook(() => useCompare(client));
    act(() => result.current.setImage("base", BASE));
    act(() => result.current.setImage("compare", COMPARE));

    act(() => result.current.runCompare()); // run 1 (pending)
    act(() => result.current.runCompare()); // run 2 supersedes run 1
    await waitFor(() => expect(result.current.result?.verdict).toBe("similar"));

    // Resolving the stale first run must not clobber the newer verdict.
    act(() => resolveFirst(diffResult({ verdict: "different" })));
    await Promise.resolve();
    expect(result.current.result?.verdict).toBe("similar");
  });

  it("reset clears the verdict and ignores an in-flight run", async () => {
    let resolveRun: (v: DiffResult) => void = () => {};
    const pending = new Promise<DiffResult>((res) => {
      resolveRun = res;
    });
    const client = fakeClient({ compare: vi.fn().mockReturnValue(pending) });
    const { result } = renderHook(() => useCompare(client));
    act(() => result.current.setImage("base", BASE));
    act(() => result.current.setImage("compare", COMPARE));
    act(() => result.current.runCompare());
    expect(result.current.phase).toBe("comparing");

    act(() => result.current.reset());
    expect(result.current.phase).toBe("idle");
    act(() => resolveRun(diffResult()));
    await Promise.resolve();
    expect(result.current.result).toBeNull();
  });

  it("replacing an input invalidates the prior verdict", async () => {
    const client = fakeClient();
    const { result } = renderHook(() => useCompare(client));
    act(() => result.current.setImage("base", BASE));
    act(() => result.current.setImage("compare", COMPARE));
    act(() => result.current.runCompare());
    await waitFor(() => expect(result.current.result).not.toBeNull());

    act(() => result.current.setImage("compare", new File(["new"], "new.png", { type: "image/png" })));
    expect(result.current.result).toBeNull();
    expect(result.current.phase).toBe("idle");
  });
});
