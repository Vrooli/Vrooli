/**
 * useEnhance lifecycle tests. The hook drives one AI enhancement op through
 * select → (install gate) → submit → watch → fetch-result via the injected
 * EnhanceClient seam, so the whole state machine is exercised without a
 * network. Covers the happy path, the model-install gate, failure + retry,
 * and cancel.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";

import { useEnhance } from "./useEnhance";
import { makeEnhanceClient, makeSelectedModel } from "./mocks/ai";

const PNG = new File(["bytes"], "in.png", { type: "image/png" });

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("useEnhance", () => {
  it("runs an installed op to success and hands the result back", async () => {
    const onResult = vi.fn();
    const client = makeEnhanceClient();
    const { result } = renderHook(() => useEnhance({ onResult, client }));

    act(() => result.current.start("background_removal", {}, PNG));

    await waitFor(() => expect(result.current.phase).toBe("succeeded"));
    expect(result.current.tier).toBe("local-cpu");
    expect(onResult).toHaveBeenCalledWith(
      expect.objectContaining({
        op: "background_removal",
        result: expect.objectContaining({ kind: "image", jobId: "job-1", width: 200 }),
      }),
    );
  });

  it("opens the install gate when the model isn't installed, then runs after install", async () => {
    const onResult = vi.fn();
    const client = makeEnhanceClient({
      selectModel: vi.fn(() => Promise.resolve(makeSelectedModel({ installed: false }))),
    });
    const { result } = renderHook(() => useEnhance({ onResult, client }));

    act(() => result.current.start("upscale", { scale: 2 }, PNG));
    await waitFor(() => expect(result.current.phase).toBe("needs-install"));
    expect(result.current.model?.installed).toBe(false);
    expect(client.submit).not.toHaveBeenCalled();

    act(() => result.current.installAndRun());
    await waitFor(() => expect(result.current.phase).toBe("succeeded"));
    expect(client.install).toHaveBeenCalledWith("rembg");
    expect(client.submit).toHaveBeenCalledWith("upscale", { scale: 2 }, PNG);
    expect(onResult).toHaveBeenCalledTimes(1);
  });

  it("surfaces a failed terminal state and retries", async () => {
    const onResult = vi.fn();
    const result_ = vi
      .fn()
      .mockResolvedValueOnce({ ok: false, resultRef: "", error: "backend exploded" })
      .mockResolvedValue({ ok: true, resultRef: "out/ok.png", error: "" });
    const client = makeEnhanceClient({ result: result_ });
    const { result } = renderHook(() => useEnhance({ onResult, client }));

    act(() => result.current.start("denoise", {}, PNG));
    await waitFor(() => expect(result.current.phase).toBe("failed"));
    expect(result.current.error).toBe("backend exploded");
    expect(onResult).not.toHaveBeenCalled();

    act(() => result.current.retry());
    await waitFor(() => expect(result.current.phase).toBe("succeeded"));
    expect(onResult).toHaveBeenCalledTimes(1);
  });

  it("cancels an in-flight job and returns to idle without landing a result", async () => {
    const onResult = vi.fn();
    const client = makeEnhanceClient({
      // Watch that stays open until the run is aborted.
      watch: vi.fn(
        (_jobId, signal, onEvent) =>
          new Promise<void>((resolve) => {
            onEvent({ percent: 20, message: "", state: "running" });
            signal.addEventListener("abort", () => resolve());
          }),
      ),
    });
    const { result } = renderHook(() => useEnhance({ onResult, client }));

    act(() => result.current.start("background_removal", {}, PNG));
    await waitFor(() => expect(result.current.phase).toBe("running"));

    act(() => result.current.cancel());
    await waitFor(() => expect(result.current.phase).toBe("idle"));
    expect(client.cancel).toHaveBeenCalledWith("job-1");
    expect(onResult).not.toHaveBeenCalled();
  });
});
