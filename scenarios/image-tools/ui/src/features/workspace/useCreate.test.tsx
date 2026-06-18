/**
 * useCreate lifecycle tests. The hook drives one generation op through select →
 * (install gate) → submit → watch → parse-variations → fetch-each-blob via the
 * injected CreateClient seam, so the whole state machine (including the
 * N-variation fan-out) is exercised without a network. Covers the single- and
 * multi-variation happy paths, the install gate, failure + retry, cancel, and
 * the optional input image (text-to-image takes none; img2img passes one).
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";

import { useCreate } from "./useCreate";
import { makeCreateClient, makeSelectedModel } from "./mocks/ai";

const PNG = new File(["bytes"], "in.png", { type: "image/png" });

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("useCreate", () => {
  it("runs text-to-image (no input) to one variation", async () => {
    const client = makeCreateClient();
    const { result } = renderHook(() => useCreate({ client }));

    act(() => result.current.start("text_to_image", { prompt: "a lake", variations: 1 }));

    await waitFor(() => expect(result.current.phase).toBe("succeeded"));
    expect(result.current.results).toHaveLength(1);
    expect(result.current.results[0]?.result.kind).toBe("image");
    // No input image was submitted for text-to-image.
    expect(client.submit).toHaveBeenCalledWith(
      "text_to_image",
      expect.objectContaining({ prompt: "a lake" }),
      undefined,
      undefined,
    );
  });

  it("fans out every key from a multi-variation job message", async () => {
    const client = makeCreateClient({
      result: vi.fn(() =>
        Promise.resolve({
          ok: true,
          resultRef: "out/0.png",
          message: "variations: [out/0.png out/1.png out/2.png]",
          error: "",
        }),
      ),
    });
    const { result } = renderHook(() => useCreate({ client }));

    act(() => result.current.start("text_to_image", { prompt: "a lake", variations: 3 }));

    await waitFor(() => expect(result.current.phase).toBe("succeeded"));
    expect(result.current.results).toHaveLength(3);
    expect(client.fetchResult).toHaveBeenCalledTimes(3);
    expect(result.current.results.map((v) => v.index)).toEqual([0, 1, 2]);
  });

  it("passes the input image through for img2img", async () => {
    const client = makeCreateClient();
    const { result } = renderHook(() => useCreate({ client }));

    act(() => result.current.start("image_to_image", { prompt: "x", variations: 1 }, PNG));

    await waitFor(() => expect(result.current.phase).toBe("succeeded"));
    expect(client.submit).toHaveBeenCalledWith(
      "image_to_image",
      expect.objectContaining({ prompt: "x" }),
      PNG,
      undefined,
    );
  });

  it("opens the install gate when the model isn't installed, then runs after install", async () => {
    const client = makeCreateClient({
      selectModel: vi.fn(() => Promise.resolve(makeSelectedModel({ id: "sd-1.5", installed: false }))),
    });
    const { result } = renderHook(() => useCreate({ client }));

    act(() => result.current.start("text_to_image", { prompt: "a lake", variations: 1 }));
    await waitFor(() => expect(result.current.phase).toBe("needs-install"));
    expect(client.submit).not.toHaveBeenCalled();

    act(() => result.current.installAndRun());
    await waitFor(() => expect(result.current.phase).toBe("succeeded"));
    expect(client.install).toHaveBeenCalledWith("sd-1.5");
    expect(result.current.results).toHaveLength(1);
  });

  it("surfaces a failed terminal state and retries", async () => {
    const result_ = vi
      .fn()
      .mockResolvedValueOnce({ ok: false, resultRef: "", message: "", error: "backend exploded" })
      .mockResolvedValue({ ok: true, resultRef: "out/ok.png", message: "produced 1/1", error: "" });
    const client = makeCreateClient({ result: result_ });
    const { result } = renderHook(() => useCreate({ client }));

    act(() => result.current.start("text_to_image", { prompt: "a lake", variations: 1 }));
    await waitFor(() => expect(result.current.phase).toBe("failed"));
    expect(result.current.error).toBe("backend exploded");
    expect(result.current.results).toHaveLength(0);

    act(() => result.current.retry());
    await waitFor(() => expect(result.current.phase).toBe("succeeded"));
    expect(result.current.results).toHaveLength(1);
  });

  it("cancels an in-flight job and returns to idle without landing variations", async () => {
    const client = makeCreateClient({
      watch: vi.fn(
        (_jobId, signal, onEvent) =>
          new Promise<void>((resolve) => {
            onEvent({ percent: 20, message: "produced 1/3", state: "running" });
            signal.addEventListener("abort", () => resolve());
          }),
      ),
    });
    const { result } = renderHook(() => useCreate({ client }));

    act(() => result.current.start("text_to_image", { prompt: "a lake", variations: 3 }));
    await waitFor(() => expect(result.current.phase).toBe("running"));

    act(() => result.current.cancel());
    await waitFor(() => expect(result.current.phase).toBe("idle"));
    expect(client.cancel).toHaveBeenCalledWith("gen-1");
    expect(result.current.results).toHaveLength(0);
  });

  it("tracks the requested variation count for the skeleton grid", async () => {
    const client = makeCreateClient({
      watch: vi.fn(
        (_jobId, signal, onEvent) =>
          new Promise<void>((resolve) => {
            onEvent({ percent: 10, message: "produced 1/4", state: "running" });
            signal.addEventListener("abort", () => resolve());
          }),
      ),
    });
    const { result } = renderHook(() => useCreate({ client }));

    act(() => result.current.start("text_to_image", { prompt: "a lake", variations: 4 }));
    await waitFor(() => expect(result.current.phase).toBe("running"));
    expect(result.current.requestedCount).toBe(4);
  });
});
