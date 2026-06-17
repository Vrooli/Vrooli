/**
 * Op-stack tests (req 21) — proves the non-destructive editor pipeline.
 *
 * Exercises the `useOpStack` hook directly with an injected fake runner so the
 * stack logic is verified without the network. The central guarantee: applying
 * ops never mutates the base image, and undo/redo move between cached steps
 * rather than recomputing — so undo cannot corrupt earlier results.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, renderHook } from "@testing-library/react";

import type { OpParamValues, RunOpImageResult } from "../../api/ops";
import { useOpStack, type OpRunner } from "./useOpStack";

const PNG = new File(["base-bytes"], "base.png", { type: "image/png" });

/** Deterministic runner: each call yields a uniquely-named output File. */
const makeRunner = () => {
  let n = 0;
  return vi.fn<OpRunner>((_operation: string, _params: OpParamValues, _input: File) => {
    n += 1;
    const result: RunOpImageResult = {
      kind: "image",
      url: `blob:step-${n}`,
      width: n,
      height: n,
      format: "png",
      jobId: `job-${n}`,
    };
    return Promise.resolve({
      result,
      outputFile: new File([`out-${n}`], `out-${n}.png`, { type: "image/png" }),
    });
  });
};

describe("useOpStack", () => {
  beforeEach(() => {
    vi.stubGlobal("URL", {
      ...URL,
      createObjectURL: vi.fn(() => "blob:base"),
      revokeObjectURL: vi.fn(),
    });
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
    vi.unstubAllGlobals();
  });

  it("applies ops onto a stack, composing each op on the previous output", async () => {
    const runner = makeRunner();
    const { result } = renderHook(() => useOpStack(runner));

    act(() => result.current.setBase(PNG));
    expect(result.current.previewUrl).toBe("blob:base");

    await act(async () => {
      await result.current.apply("resize", { width: 100 });
    });
    expect(result.current.entries).toHaveLength(1);
    expect(result.current.previewUrl).toBe("blob:step-1");

    await act(async () => {
      await result.current.apply("crop", { x: 0 });
    });
    expect(result.current.entries).toHaveLength(2);
    expect(result.current.previewUrl).toBe("blob:step-2");

    // First op ran against the base; second op composed on the first output.
    expect(runner.mock.calls[0]?.[2]).toBe(PNG);
    expect(runner.mock.calls[1]?.[2]).toBeInstanceOf(File);
    expect((runner.mock.calls[1]?.[2] as File).name).toBe("out-1.png");
  });

  it("undo is non-destructive: the base image and prior results are preserved", async () => {
    const runner = makeRunner();
    const { result } = renderHook(() => useOpStack(runner));

    act(() => result.current.setBase(PNG));
    await act(async () => {
      await result.current.apply("resize", { width: 100 });
    });
    await act(async () => {
      await result.current.apply("crop", { x: 0 });
    });
    expect(runner).toHaveBeenCalledTimes(2);

    // Undo drops to the resize result WITHOUT re-running anything.
    act(() => result.current.undo());
    expect(result.current.entries).toHaveLength(1);
    expect(result.current.previewUrl).toBe("blob:step-1");
    expect(runner).toHaveBeenCalledTimes(2);

    // Base image is untouched throughout (non-destructive).
    expect(result.current.base?.file).toBe(PNG);

    // Undo again returns to the original; canUndo flips off.
    act(() => result.current.undo());
    expect(result.current.entries).toHaveLength(0);
    expect(result.current.previewUrl).toBe("blob:base");
    expect(result.current.canUndo).toBe(false);
    expect(result.current.base?.file).toBe(PNG);
  });

  it("redo replays an undone step from cache (no recompute)", async () => {
    const runner = makeRunner();
    const { result } = renderHook(() => useOpStack(runner));

    act(() => result.current.setBase(PNG));
    await act(async () => {
      await result.current.apply("resize", { width: 100 });
    });

    act(() => result.current.undo());
    expect(result.current.canRedo).toBe(true);

    act(() => result.current.redo());
    expect(result.current.entries).toHaveLength(1);
    expect(result.current.previewUrl).toBe("blob:step-1");
    // Redo reuses the cached step — runner still called exactly once.
    expect(runner).toHaveBeenCalledTimes(1);
  });

  it("applying a new op after undo clears the redo stack", async () => {
    const runner = makeRunner();
    const { result } = renderHook(() => useOpStack(runner));

    act(() => result.current.setBase(PNG));
    await act(async () => {
      await result.current.apply("resize", { width: 100 });
    });
    await act(async () => {
      await result.current.apply("crop", { x: 0 });
    });

    act(() => result.current.undo());
    expect(result.current.canRedo).toBe(true);

    await act(async () => {
      await result.current.apply("rotate", { angle: 90 });
    });
    expect(result.current.canRedo).toBe(false);
    expect(result.current.entries).toHaveLength(2);
  });
});
