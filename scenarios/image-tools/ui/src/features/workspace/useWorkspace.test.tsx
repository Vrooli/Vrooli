/**
 * useWorkspace store tests — the non-destructive op stack plus the metadata
 * branch. Exercises the hook directly with an injected fake runner so the
 * store logic is verified without the network. The central guarantees:
 * applying image ops never mutates the base, undo/redo move between cached
 * steps (no recompute), and a metadata read shows JSON without pushing a step.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, renderHook } from "@testing-library/react";

import type { RunOpImageResult } from "../../api/ops";
import { useWorkspace, type WorkspaceRunner } from "./useWorkspace";

const PNG = new File(["base-bytes"], "base.png", { type: "image/png" });

/** Deterministic image runner: each call yields a uniquely-named output. */
const makeImageRunner = () => {
  let n = 0;
  return vi.fn<WorkspaceRunner>(() => {
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
      kind: "image",
      result,
      outputFile: new File([`out-${n}`], `out-${n}.png`, { type: "image/png" }),
    });
  });
};

const setup = (runner: WorkspaceRunner) => {
  const hook = renderHook(() => useWorkspace(runner));
  return hook;
};

describe("useWorkspace", () => {
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

  it("applies image ops onto a stack, composing each on the previous output", async () => {
    const runner = makeImageRunner();
    const { result } = setup(runner);

    act(() => result.current.setBase(PNG));
    expect(result.current.previewUrl).toBe("blob:base");

    act(() => result.current.setOperation("resize"));
    await act(async () => {
      await result.current.apply();
    });
    expect(result.current.entries).toHaveLength(1);
    expect(result.current.previewUrl).toBe("blob:step-1");

    act(() => result.current.setOperation("crop"));
    await act(async () => {
      await result.current.apply();
    });
    expect(result.current.entries).toHaveLength(2);
    expect(result.current.previewUrl).toBe("blob:step-2");

    // First op ran against the base; second composed on the first output.
    expect(runner.mock.calls[0]?.[2]).toBe(PNG);
    expect((runner.mock.calls[1]?.[2] as File).name).toBe("out-1.png");
  });

  it("a metadata read shows JSON without pushing a step", async () => {
    const runner = vi.fn<WorkspaceRunner>(() =>
      Promise.resolve({ kind: "metadata", json: '{"format":"png"}' }),
    );
    const { result } = setup(runner);

    act(() => result.current.setBase(PNG));
    act(() => result.current.setOperation("metadata"));
    await act(async () => {
      await result.current.apply();
    });

    expect(result.current.metadata).toBe('{"format":"png"}');
    expect(result.current.entries).toHaveLength(0);
    expect(result.current.previewUrl).toBe("blob:base");
  });

  it("undo is non-destructive: the base and prior results are preserved", async () => {
    const runner = makeImageRunner();
    const { result } = setup(runner);

    act(() => result.current.setBase(PNG));
    act(() => result.current.setOperation("resize"));
    await act(async () => {
      await result.current.apply();
    });
    act(() => result.current.setOperation("crop"));
    await act(async () => {
      await result.current.apply();
    });
    expect(runner).toHaveBeenCalledTimes(2);

    act(() => result.current.undo());
    expect(result.current.entries).toHaveLength(1);
    expect(result.current.previewUrl).toBe("blob:step-1");
    expect(runner).toHaveBeenCalledTimes(2);
    expect(result.current.base?.file).toBe(PNG);

    act(() => result.current.undo());
    expect(result.current.entries).toHaveLength(0);
    expect(result.current.previewUrl).toBe("blob:base");
    expect(result.current.canUndo).toBe(false);
    expect(result.current.base?.file).toBe(PNG);
  });

  it("redo replays an undone step from cache (no recompute)", async () => {
    const runner = makeImageRunner();
    const { result } = setup(runner);

    act(() => result.current.setBase(PNG));
    act(() => result.current.setOperation("resize"));
    await act(async () => {
      await result.current.apply();
    });

    act(() => result.current.undo());
    expect(result.current.canRedo).toBe(true);

    act(() => result.current.redo());
    expect(result.current.entries).toHaveLength(1);
    expect(result.current.previewUrl).toBe("blob:step-1");
    expect(runner).toHaveBeenCalledTimes(1);
  });

  it("applying a new op after undo clears the redo stack", async () => {
    const runner = makeImageRunner();
    const { result } = setup(runner);

    act(() => result.current.setBase(PNG));
    act(() => result.current.setOperation("resize"));
    await act(async () => {
      await result.current.apply();
    });
    act(() => result.current.setOperation("crop"));
    await act(async () => {
      await result.current.apply();
    });

    act(() => result.current.undo());
    expect(result.current.canRedo).toBe(true);

    act(() => result.current.setOperation("rotate"));
    await act(async () => {
      await result.current.apply();
    });
    expect(result.current.canRedo).toBe(false);
    expect(result.current.entries).toHaveLength(2);
  });

  it("resets the stack while leaving the base loaded", async () => {
    const runner = makeImageRunner();
    const { result } = setup(runner);

    act(() => result.current.setBase(PNG));
    act(() => result.current.setOperation("resize"));
    await act(async () => {
      await result.current.apply();
    });
    expect(result.current.entries).toHaveLength(1);

    act(() => result.current.reset());
    expect(result.current.entries).toHaveLength(0);
    expect(result.current.base?.file).toBe(PNG);
    expect(result.current.previewUrl).toBe("blob:base");
  });
});
