/**
 * useEnhance lifecycle tests. The hook drives one AI enhancement op through
 * select → (install gate) → submit → watch → fetch-result via the injected
 * EnhanceClient seam, so the whole state machine is exercised without a
 * network. Covers the happy path, the model-install gate, failure + retry,
 * and cancel.
 *
 * The production `liveEnhanceClient` (default seam wrapping the AI submit edge +
 * the Jobs/Models Connect clients) is covered in `useEnhance.live.test.tsx`,
 * which mocks the Connect clients + AI edge to assert the proto→seam mapping.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";

import { isEnhanceActive, useEnhance } from "./useEnhance";
import { makeEnhanceClient, makeSelectedModel } from "./mocks/ai";

const PNG = new File(["bytes"], "in.png", { type: "image/png" });

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("isEnhanceActive", () => {
  it("treats the spinner phases as active and the terminal/idle phases as inactive", () => {
    expect(isEnhanceActive("submitting")).toBe(true);
    expect(isEnhanceActive("installing")).toBe(true);
    expect(isEnhanceActive("running")).toBe(true);
    expect(isEnhanceActive("idle")).toBe(false);
    expect(isEnhanceActive("needs-install")).toBe(false);
    expect(isEnhanceActive("succeeded")).toBe(false);
    expect(isEnhanceActive("failed")).toBe(false);
  });
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

  it("surfaces a selectModel failure (Error.message)", async () => {
    const onResult = vi.fn();
    const client = makeEnhanceClient({
      selectModel: vi.fn(() => Promise.reject(new Error("registry offline"))),
    });
    const { result } = renderHook(() => useEnhance({ onResult, client }));

    act(() => result.current.start("upscale", { scale: 2 }, PNG));

    await waitFor(() => expect(result.current.phase).toBe("failed"));
    expect(result.current.error).toBe("registry offline");
    expect(client.submit).not.toHaveBeenCalled();
  });

  it("surfaces a thrown submit error and stringifies a non-Error throw", async () => {
    const onResult = vi.fn();
    const client = makeEnhanceClient({
      // eslint-disable-next-line @typescript-eslint/prefer-promise-reject-errors -- intentional non-Error rejection to cover the String(err) branch
      submit: vi.fn(() => Promise.reject("submit edge crashed")),
    });
    const { result } = renderHook(() => useEnhance({ onResult, client }));

    act(() => result.current.start("denoise", {}, PNG));

    await waitFor(() => expect(result.current.phase).toBe("failed"));
    expect(result.current.error).toBe("submit edge crashed");
  });

  it("fails when reading the terminal job state throws", async () => {
    const onResult = vi.fn();
    const client = makeEnhanceClient({
      result: vi.fn(() => Promise.reject(new Error("getJob unreachable"))),
    });
    const { result } = renderHook(() => useEnhance({ onResult, client }));

    act(() => result.current.start("denoise", {}, PNG));

    await waitFor(() => expect(result.current.phase).toBe("failed"));
    expect(result.current.error).toBe("getJob unreachable");
    expect(onResult).not.toHaveBeenCalled();
  });

  it("fails with a null error when the terminal job is not-ok but carries no message", async () => {
    const onResult = vi.fn();
    const client = makeEnhanceClient({
      result: vi.fn(() => Promise.resolve({ ok: false, resultRef: "", error: "" })),
    });
    const { result } = renderHook(() => useEnhance({ onResult, client }));

    act(() => result.current.start("denoise", {}, PNG));

    await waitFor(() => expect(result.current.phase).toBe("failed"));
    expect(result.current.error).toBeNull();
  });

  it("fails when fetching the result blob throws", async () => {
    const onResult = vi.fn();
    const client = makeEnhanceClient({
      fetchResult: vi.fn(() => Promise.reject(new Error("blob 404"))),
    });
    const { result } = renderHook(() => useEnhance({ onResult, client }));

    act(() => result.current.start("denoise", {}, PNG));

    await waitFor(() => expect(result.current.phase).toBe("failed"));
    expect(result.current.error).toBe("blob 404");
    expect(onResult).not.toHaveBeenCalled();
  });

  it("installAndRun is a no-op before any start (no pending run/model)", () => {
    const onResult = vi.fn();
    const client = makeEnhanceClient();
    const { result } = renderHook(() => useEnhance({ onResult, client }));

    act(() => result.current.installAndRun());

    expect(result.current.phase).toBe("idle");
    expect(client.install).not.toHaveBeenCalled();
  });

  it("surfaces a thrown install error from the gate", async () => {
    const onResult = vi.fn();
    const client = makeEnhanceClient({
      selectModel: vi.fn(() => Promise.resolve(makeSelectedModel({ installed: false }))),
      install: vi.fn(() => Promise.reject(new Error("no network"))),
    });
    const { result } = renderHook(() => useEnhance({ onResult, client }));

    act(() => result.current.start("upscale", { scale: 2 }, PNG));
    await waitFor(() => expect(result.current.phase).toBe("needs-install"));

    act(() => result.current.installAndRun());
    await waitFor(() => expect(result.current.phase).toBe("failed"));
    expect(result.current.error).toBe("no network");
    expect(client.submit).not.toHaveBeenCalled();
  });

  it("surfaces a thrown waitJob error from the gate", async () => {
    const onResult = vi.fn();
    const client = makeEnhanceClient({
      selectModel: vi.fn(() => Promise.resolve(makeSelectedModel({ installed: false }))),
      waitJob: vi.fn(() => Promise.reject(new Error("wait stream broke"))),
    });
    const { result } = renderHook(() => useEnhance({ onResult, client }));

    act(() => result.current.start("upscale", { scale: 2 }, PNG));
    await waitFor(() => expect(result.current.phase).toBe("needs-install"));

    act(() => result.current.installAndRun());
    await waitFor(() => expect(result.current.phase).toBe("failed"));
    expect(result.current.error).toBe("wait stream broke");
  });

  it("fails with a null error when the install job fails without a message", async () => {
    const onResult = vi.fn();
    const client = makeEnhanceClient({
      selectModel: vi.fn(() => Promise.resolve(makeSelectedModel({ installed: false }))),
      waitJob: vi.fn(() => Promise.resolve({ ok: false, error: "" })),
    });
    const { result } = renderHook(() => useEnhance({ onResult, client }));

    act(() => result.current.start("upscale", { scale: 2 }, PNG));
    await waitFor(() => expect(result.current.phase).toBe("needs-install"));

    act(() => result.current.installAndRun());
    await waitFor(() => expect(result.current.phase).toBe("failed"));
    expect(result.current.error).toBeNull();
    expect(client.submit).not.toHaveBeenCalled();
  });

  it("skips the install wait when the model is already installed", async () => {
    const onResult = vi.fn();
    const client = makeEnhanceClient({
      selectModel: vi.fn(() => Promise.resolve(makeSelectedModel({ installed: false }))),
      install: vi.fn(() => Promise.resolve({ jobId: "", alreadyInstalled: true })),
    });
    const { result } = renderHook(() => useEnhance({ onResult, client }));

    act(() => result.current.start("upscale", { scale: 2 }, PNG));
    await waitFor(() => expect(result.current.phase).toBe("needs-install"));

    act(() => result.current.installAndRun());
    await waitFor(() => expect(result.current.phase).toBe("succeeded"));
    expect(client.waitJob).not.toHaveBeenCalled();
    expect(onResult).toHaveBeenCalledTimes(1);
  });

  it("preview fetches the model badge without running, and swallows its failures", async () => {
    const onResult = vi.fn();
    const client = makeEnhanceClient({
      selectModel: vi
        .fn()
        .mockResolvedValueOnce(makeSelectedModel({ id: "rembg", name: "rembg" }))
        .mockRejectedValueOnce(new Error("badge lookup failed")),
    });
    const { result } = renderHook(() => useEnhance({ onResult, client }));

    act(() => result.current.preview("background_removal"));
    await waitFor(() => expect(result.current.model?.id).toBe("rembg"));
    expect(client.submit).not.toHaveBeenCalled();
    expect(result.current.phase).toBe("idle");

    // A failing badge lookup is swallowed (no error surfaced until a real run).
    act(() => result.current.preview("upscale"));
    await Promise.resolve();
    expect(result.current.error).toBeNull();
    expect(result.current.phase).toBe("idle");
  });

  it("dismiss clears a terminal/error state back to idle (keeping the model badge)", async () => {
    const onResult = vi.fn();
    const client = makeEnhanceClient();
    const { result } = renderHook(() => useEnhance({ onResult, client }));

    act(() => result.current.start("background_removal", {}, PNG));
    await waitFor(() => expect(result.current.phase).toBe("succeeded"));

    act(() => result.current.dismiss());
    expect(result.current.phase).toBe("idle");
    expect(result.current.error).toBeNull();
    expect(result.current.model).not.toBeNull();
  });

  it("retry is a no-op when nothing has been started", () => {
    const onResult = vi.fn();
    const client = makeEnhanceClient();
    const { result } = renderHook(() => useEnhance({ onResult, client }));

    act(() => result.current.retry());

    expect(result.current.phase).toBe("idle");
    expect(client.selectModel).not.toHaveBeenCalled();
  });

  it("cancel is a best-effort no-op (no job yet) and tolerates a rejecting cancel", () => {
    const onResult = vi.fn();
    // No job has been submitted yet, so jobIdRef is empty → cancel() must not
    // call client.cancel; it just resets to idle.
    const client = makeEnhanceClient();
    const { result } = renderHook(() => useEnhance({ onResult, client }));

    act(() => result.current.cancel());
    expect(result.current.phase).toBe("idle");
    expect(client.cancel).not.toHaveBeenCalled();
  });

  it("swallows a rejecting cancel call for an in-flight job", async () => {
    const onResult = vi.fn();
    const client = makeEnhanceClient({
      watch: vi.fn(
        (_jobId, signal, onEvent) =>
          new Promise<void>((resolve) => {
            onEvent({ percent: 20, message: "", state: "running" });
            signal.addEventListener("abort", () => resolve());
          }),
      ),
      cancel: vi.fn(() => Promise.reject(new Error("cancel race"))),
    });
    const { result } = renderHook(() => useEnhance({ onResult, client }));

    act(() => result.current.start("background_removal", {}, PNG));
    await waitFor(() => expect(result.current.phase).toBe("running"));

    act(() => result.current.cancel());
    await waitFor(() => expect(result.current.phase).toBe("idle"));
    expect(client.cancel).toHaveBeenCalledWith("job-1");
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
