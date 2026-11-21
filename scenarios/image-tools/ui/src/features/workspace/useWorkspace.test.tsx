/**
 * useWorkspace store tests — the non-destructive op stack plus the metadata
 * branch. Exercises the hook directly with an injected fake runner so the
 * store logic is verified without the network. The central guarantees:
 * applying image ops never mutates the base, undo/redo move between cached
 * steps (no recompute), and a metadata read shows JSON without pushing a step.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, renderHook } from "@testing-library/react";

import { runOp, type RunOpImageResult } from "../../api/ops";
import { liveRunner, useWorkspace, type WorkspaceRunner } from "./useWorkspace";

// Mock only the network-touching `runOp`; keep the rest of api/ops intact so
// the typed re-exports and helpers stay real. `liveRunner` is the one path that
// crosses the wire, so its branches are tested against the mock + a fake fetch.
vi.mock("../../api/ops", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/ops")>();
  return { ...actual, runOp: vi.fn() };
});

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

  it("pushes an externally-produced image result as a composable step (AI path)", () => {
    const { result } = setup(makeImageRunner());
    act(() => result.current.setBase(PNG));

    const aiResult: RunOpImageResult = {
      kind: "image",
      url: "blob:enhanced",
      width: 2048,
      height: 1536,
      format: "png",
      jobId: "ai-job-1",
    };
    const outputFile = new File(["enhanced"], "enhanced.png", { type: "image/png" });

    act(() => result.current.applyImageResult("upscale", { scale: 2 }, aiResult, outputFile));

    expect(result.current.entries).toHaveLength(1);
    expect(result.current.entries[0]?.operation).toBe("upscale");
    expect(result.current.currentResult).toBe(aiResult);
    expect(result.current.previewUrl).toBe("blob:enhanced");
    expect(result.current.canUndo).toBe(true);

    // It composes into the same history — undo removes it like any other step.
    act(() => result.current.undo());
    expect(result.current.entries).toHaveLength(0);
    expect(result.current.previewUrl).toBe("blob:base");
  });

  it("setMode, setOperation, setParam and setParams mutate the controlled state", () => {
    const { result } = setup(makeImageRunner());

    act(() => result.current.setMode("enhance"));
    expect(result.current.mode).toBe("enhance");

    // setOperation seeds the form from the op's spec defaults.
    act(() => result.current.setOperation("resize"));
    expect(result.current.params).toEqual({ width: 256, height: 0, fit: "fit", gravity: "" });

    // setParam replaces a single key, leaving the rest intact.
    act(() => result.current.setParam("width", 512));
    expect(result.current.params.width).toBe(512);
    expect(result.current.params.height).toBe(0);

    // setParams merges several keys in one change (e.g. a crop drag).
    act(() => result.current.setParams({ height: 128, fit: "fill" }));
    expect(result.current.params).toMatchObject({ width: 512, height: 128, fit: "fill" });
  });

  it("apply is a no-op when no base image is loaded (no input branch)", async () => {
    const runner = makeImageRunner();
    const { result } = setup(runner);

    act(() => result.current.setOperation("resize"));
    await act(async () => {
      await result.current.apply();
    });

    expect(runner).not.toHaveBeenCalled();
    expect(result.current.entries).toHaveLength(0);
  });

  it("apply is a no-op when no operation is selected", async () => {
    const runner = makeImageRunner();
    const { result } = setup(runner);

    act(() => result.current.setBase(PNG));
    // operation stays "" — the falsy-operation guard short-circuits.
    await act(async () => {
      await result.current.apply();
    });

    expect(runner).not.toHaveBeenCalled();
    expect(result.current.entries).toHaveLength(0);
  });

  it("captures a runner rejection in error and clears applying", async () => {
    const boom = new Error("op failed");
    const runner = vi.fn<WorkspaceRunner>(() => Promise.reject(boom));
    const { result } = setup(runner);

    act(() => result.current.setBase(PNG));
    act(() => result.current.setOperation("resize"));
    await act(async () => {
      await result.current.apply();
    });

    expect(result.current.error).toBe(boom);
    expect(result.current.applying).toBe(false);
    expect(result.current.entries).toHaveLength(0);
  });

  it("forwards an overlay to the runner only for overlay-accepting ops", async () => {
    const runner = makeImageRunner();
    const { result } = setup(runner);
    const overlay = new File(["wm"], "wm.png", { type: "image/png" });

    act(() => result.current.setBase(PNG));
    act(() => result.current.setOperation("overlay"));
    await act(async () => {
      await result.current.apply(overlay);
    });
    // overlay op declares acceptsOverlay → the overlay is threaded through.
    expect(runner.mock.calls[0]?.[3]).toEqual({ overlay });

    // A non-overlay op drops the overlay (empty opts branch).
    act(() => result.current.setOperation("resize"));
    await act(async () => {
      await result.current.apply(overlay);
    });
    expect(runner.mock.calls[1]?.[3]).toEqual({});
  });

  it("undo and redo are no-ops on empty stacks", () => {
    const { result } = setup(makeImageRunner());
    act(() => result.current.setBase(PNG));

    act(() => result.current.undo());
    expect(result.current.entries).toHaveLength(0);
    expect(result.current.canUndo).toBe(false);

    act(() => result.current.redo());
    expect(result.current.entries).toHaveLength(0);
    expect(result.current.canRedo).toBe(false);
  });

  it("revokes the prior base object URL when a new base replaces it", () => {
    const { result } = setup(makeImageRunner());
    const url = vi.mocked(URL);

    act(() => result.current.setBase(PNG));
    const replacement = new File(["other"], "other.png", { type: "image/png" });
    act(() => result.current.setBase(replacement));

    expect(url.revokeObjectURL).toHaveBeenCalledWith("blob:base");
    expect(result.current.base?.file).toBe(replacement);
  });
});

describe("liveRunner", () => {
  beforeEach(() => {
    vi.stubGlobal("URL", {
      ...URL,
      createObjectURL: vi.fn(() => "blob:live"),
      revokeObjectURL: vi.fn(),
    });
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
    vi.unstubAllGlobals();
  });

  it("passes a metadata result straight through (metadata branch)", async () => {
    vi.mocked(runOp).mockResolvedValueOnce({ kind: "metadata", json: '{"k":1}' });

    const out = await liveRunner("metadata", {}, PNG, {});
    expect(out).toEqual({ kind: "metadata", json: '{"k":1}' });
  });

  it("materializes an image result's bytes as a File using the result format", async () => {
    vi.mocked(runOp).mockResolvedValueOnce({
      kind: "image",
      url: "blob:remote",
      width: 10,
      height: 20,
      format: "webp",
      jobId: "j1",
    });
    const blob = new Blob(["png-bytes"], { type: "image/webp" });
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve({ blob: () => Promise.resolve(blob) })));

    const out = await liveRunner("resize", { width: 10 }, PNG, {});
    expect(out.kind).toBe("image");
    if (out.kind !== "image") throw new Error("expected image result");
    expect(out.outputFile.name).toBe("step.webp");
    expect(out.outputFile.type).toBe("image/webp");
    expect(out.result.url).toBe("blob:remote");
  });

  it("falls back to png + image/png when the result format and blob type are blank", async () => {
    vi.mocked(runOp).mockResolvedValueOnce({
      kind: "image",
      url: "blob:remote",
      width: 0,
      height: 0,
      format: "",
      jobId: "",
    });
    // A blob with an empty type exercises the `blob.type || "image/png"` fallback.
    const blob = new Blob(["bytes"]);
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve({ blob: () => Promise.resolve(blob) })));

    const out = await liveRunner("rotate", {}, PNG, {});
    if (out.kind !== "image") throw new Error("expected image result");
    expect(out.outputFile.name).toBe("step.png");
    expect(out.outputFile.type).toBe("image/png");
  });

  it("threads overlay opts into runOp", async () => {
    vi.mocked(runOp).mockResolvedValueOnce({ kind: "metadata", json: "{}" });
    const overlay = new File(["wm"], "wm.png", { type: "image/png" });

    await liveRunner("overlay", { text: "hi" }, PNG, { overlay });
    expect(vi.mocked(runOp)).toHaveBeenCalledWith(
      "overlay",
      PNG,
      expect.objectContaining({ text: "hi" }),
      { overlay },
    );
  });
});
