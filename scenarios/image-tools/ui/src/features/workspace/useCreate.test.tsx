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

import { isCreateActive, useCreate } from "./useCreate";
import { makeCreateClient, makeSelectedModel } from "./mocks/ai";

const PNG = new File(["bytes"], "in.png", { type: "image/png" });

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("isCreateActive", () => {
  it("aliases isEnhanceActive: spinner phases active, terminal/idle inactive", () => {
    expect(isCreateActive("submitting")).toBe(true);
    expect(isCreateActive("running")).toBe(true);
    expect(isCreateActive("installing")).toBe(true);
    expect(isCreateActive("idle")).toBe(false);
    expect(isCreateActive("succeeded")).toBe(false);
    expect(isCreateActive("failed")).toBe(false);
    expect(isCreateActive("needs-install")).toBe(false);
  });
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

  it("surfaces a thrown submit error (Error.message) as the failure text", async () => {
    const client = makeCreateClient({
      submit: vi.fn(() => Promise.reject(new Error("queue is full"))),
    });
    const { result } = renderHook(() => useCreate({ client }));

    act(() => result.current.start("text_to_image", { prompt: "a lake", variations: 1 }));

    await waitFor(() => expect(result.current.phase).toBe("failed"));
    expect(result.current.error).toBe("queue is full");
    expect(result.current.results).toHaveLength(0);
  });

  it("stringifies a non-Error rejection (selectModel) for the failure text", async () => {
    const client = makeCreateClient({
      // eslint-disable-next-line @typescript-eslint/prefer-promise-reject-errors -- intentional non-Error rejection to cover the String(err) branch
      selectModel: vi.fn(() => Promise.reject("model registry down")),
    });
    const { result } = renderHook(() => useCreate({ client }));

    act(() => result.current.start("text_to_image", { prompt: "a lake", variations: 1 }));

    await waitFor(() => expect(result.current.phase).toBe("failed"));
    expect(result.current.error).toBe("model registry down");
  });

  it("installAndRun is a no-op before any start (no pending run/model)", () => {
    const client = makeCreateClient();
    const { result } = renderHook(() => useCreate({ client }));

    act(() => result.current.installAndRun());

    expect(result.current.phase).toBe("idle");
    expect(client.install).not.toHaveBeenCalled();
  });

  it("fails the run when the install job itself fails", async () => {
    const client = makeCreateClient({
      selectModel: vi.fn(() => Promise.resolve(makeSelectedModel({ id: "sd-1.5", installed: false }))),
      waitJob: vi.fn(() => Promise.resolve({ ok: false, error: "download failed" })),
    });
    const { result } = renderHook(() => useCreate({ client }));

    act(() => result.current.start("text_to_image", { prompt: "a lake", variations: 1 }));
    await waitFor(() => expect(result.current.phase).toBe("needs-install"));

    act(() => result.current.installAndRun());
    await waitFor(() => expect(result.current.phase).toBe("failed"));
    expect(result.current.error).toBe("download failed");
    expect(client.submit).not.toHaveBeenCalled();
  });

  it("skips the install wait when the model is already installed", async () => {
    const client = makeCreateClient({
      selectModel: vi.fn(() => Promise.resolve(makeSelectedModel({ id: "sd-1.5", installed: false }))),
      install: vi.fn(() => Promise.resolve({ jobId: "", alreadyInstalled: true })),
    });
    const { result } = renderHook(() => useCreate({ client }));

    act(() => result.current.start("text_to_image", { prompt: "a lake", variations: 1 }));
    await waitFor(() => expect(result.current.phase).toBe("needs-install"));

    act(() => result.current.installAndRun());
    await waitFor(() => expect(result.current.phase).toBe("succeeded"));
    expect(client.waitJob).not.toHaveBeenCalled();
    expect(result.current.results).toHaveLength(1);
  });

  it("dismiss clears results + error back to idle", async () => {
    const client = makeCreateClient();
    const { result } = renderHook(() => useCreate({ client }));

    act(() => result.current.start("text_to_image", { prompt: "a lake", variations: 1 }));
    await waitFor(() => expect(result.current.phase).toBe("succeeded"));
    expect(result.current.results).toHaveLength(1);

    act(() => result.current.dismiss());
    expect(result.current.phase).toBe("idle");
    expect(result.current.results).toHaveLength(0);
    expect(result.current.error).toBeNull();
  });

  it("preview fetches the model badge without running", async () => {
    const client = makeCreateClient();
    const { result } = renderHook(() => useCreate({ client }));

    act(() => result.current.preview("text_to_image"));

    await waitFor(() => expect(result.current.model?.id).toBe("sd-1.5"));
    expect(client.submit).not.toHaveBeenCalled();
    expect(result.current.phase).toBe("idle");
  });

  it("retry is a no-op when nothing has been started", () => {
    const client = makeCreateClient();
    const { result } = renderHook(() => useCreate({ client }));

    act(() => result.current.retry());

    expect(result.current.phase).toBe("idle");
    expect(client.selectModel).not.toHaveBeenCalled();
  });

  it("fails the run when fetching a variation blob rejects", async () => {
    const client = makeCreateClient({
      fetchResult: vi.fn(() => Promise.reject(new Error("blob 404"))),
    });
    const { result } = renderHook(() => useCreate({ client }));

    act(() => result.current.start("text_to_image", { prompt: "a lake", variations: 1 }));
    await waitFor(() => expect(result.current.phase).toBe("failed"));
    expect(result.current.error).toBe("blob 404");
  });

  it("fails the run when reading the terminal job state throws", async () => {
    const client = makeCreateClient({
      result: vi.fn(() => Promise.reject(new Error("getJob unreachable"))),
    });
    const { result } = renderHook(() => useCreate({ client }));

    act(() => result.current.start("text_to_image", { prompt: "a lake", variations: 1 }));
    await waitFor(() => expect(result.current.phase).toBe("failed"));
    expect(result.current.error).toBe("getJob unreachable");
  });

  it("fails with a null error when the terminal job is not-ok but carries no message", async () => {
    const client = makeCreateClient({
      result: vi.fn(() =>
        Promise.resolve({ ok: false, resultRef: "", message: "", error: "" }),
      ),
    });
    const { result } = renderHook(() => useCreate({ client }));

    act(() => result.current.start("text_to_image", { prompt: "a lake", variations: 1 }));
    await waitFor(() => expect(result.current.phase).toBe("failed"));
    expect(result.current.error).toBeNull();
  });

  it("defaults the requested count to 1 when variations is unset", async () => {
    const client = makeCreateClient({
      watch: vi.fn(
        (_jobId, signal, onEvent) =>
          new Promise<void>((resolve) => {
            onEvent({ percent: 10, message: "", state: "running" });
            signal.addEventListener("abort", () => resolve());
          }),
      ),
    });
    const { result } = renderHook(() => useCreate({ client }));

    act(() => result.current.start("text_to_image", { prompt: "a lake" }));
    await waitFor(() => expect(result.current.phase).toBe("running"));
    expect(result.current.requestedCount).toBe(1);
  });

  it("surfaces a thrown install error from the gate", async () => {
    const client = makeCreateClient({
      selectModel: vi.fn(() => Promise.resolve(makeSelectedModel({ id: "sd-1.5", installed: false }))),
      install: vi.fn(() => Promise.reject(new Error("no network"))),
    });
    const { result } = renderHook(() => useCreate({ client }));

    act(() => result.current.start("text_to_image", { prompt: "a lake", variations: 1 }));
    await waitFor(() => expect(result.current.phase).toBe("needs-install"));

    act(() => result.current.installAndRun());
    await waitFor(() => expect(result.current.phase).toBe("failed"));
    expect(result.current.error).toBe("no network");
  });

  it("surfaces a thrown waitJob error from the gate", async () => {
    const client = makeCreateClient({
      selectModel: vi.fn(() => Promise.resolve(makeSelectedModel({ id: "sd-1.5", installed: false }))),
      waitJob: vi.fn(() => Promise.reject(new Error("wait stream broke"))),
    });
    const { result } = renderHook(() => useCreate({ client }));

    act(() => result.current.start("text_to_image", { prompt: "a lake", variations: 1 }));
    await waitFor(() => expect(result.current.phase).toBe("needs-install"));

    act(() => result.current.installAndRun());
    await waitFor(() => expect(result.current.phase).toBe("failed"));
    expect(result.current.error).toBe("wait stream broke");
  });

  it("fails with a null error when the install job fails without a message", async () => {
    const client = makeCreateClient({
      selectModel: vi.fn(() => Promise.resolve(makeSelectedModel({ id: "sd-1.5", installed: false }))),
      waitJob: vi.fn(() => Promise.resolve({ ok: false, error: "" })),
    });
    const { result } = renderHook(() => useCreate({ client }));

    act(() => result.current.start("text_to_image", { prompt: "a lake", variations: 1 }));
    await waitFor(() => expect(result.current.phase).toBe("needs-install"));

    act(() => result.current.installAndRun());
    await waitFor(() => expect(result.current.phase).toBe("failed"));
    expect(result.current.error).toBeNull();
    expect(client.submit).not.toHaveBeenCalled();
  });

  it("preview swallows a failing badge lookup without surfacing an error", async () => {
    const client = makeCreateClient({
      selectModel: vi.fn(() => Promise.reject(new Error("badge lookup failed"))),
    });
    const { result } = renderHook(() => useCreate({ client }));

    act(() => result.current.preview("text_to_image"));
    await Promise.resolve();

    expect(result.current.error).toBeNull();
    expect(result.current.phase).toBe("idle");
    expect(client.submit).not.toHaveBeenCalled();
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
