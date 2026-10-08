/**
 * useAnalyze lifecycle tests. The hook drives one analysis op through
 * (model-backed) select → install-gate → run, or (pure-Go probe) run directly,
 * via the injected AnalyzeClient seam — so the whole synchronous state machine
 * is exercised without a network. Covers the probe happy path, the
 * model-install gate, failure + retry, and clear.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";

import { isAnalyzeActive, useAnalyze } from "./useAnalyze";
import { makeAnalyzeClient, makeOcrResult, makeProbeResult } from "./mocks/analysis";
import { makeSelectedModel } from "./mocks/ai";

const PNG = new File(["bytes"], "in.png", { type: "image/png" });

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("isAnalyzeActive", () => {
  it("treats installing/running as active and the rest as inactive", () => {
    expect(isAnalyzeActive("installing")).toBe(true);
    expect(isAnalyzeActive("running")).toBe(true);
    expect(isAnalyzeActive("idle")).toBe(false);
    expect(isAnalyzeActive("needs-install")).toBe(false);
    expect(isAnalyzeActive("done")).toBe(false);
    expect(isAnalyzeActive("failed")).toBe(false);
  });
});

describe("useAnalyze", () => {
  it("runs the pure-Go probe directly (no model gate) and exposes the result", async () => {
    const client = makeAnalyzeClient();
    const { result } = renderHook(() => useAnalyze({ client }));

    act(() => result.current.run("probe", PNG, false));

    await waitFor(() => expect(result.current.phase).toBe("done"));
    expect(client.selectModel).not.toHaveBeenCalled();
    expect(client.analyze).toHaveBeenCalledWith("probe", PNG);
    expect(result.current.result?.kind).toBe("probe");
  });

  it("opens the install gate for a model-backed op, then runs after install", async () => {
    const client = makeAnalyzeClient({
      selectModel: vi.fn(() =>
        Promise.resolve(makeSelectedModel({ id: "tesseract", name: "tesseract", installed: false })),
      ),
      analyze: vi.fn(() => Promise.resolve(makeOcrResult())),
    });
    const { result } = renderHook(() => useAnalyze({ client }));

    act(() => result.current.run("ocr", PNG, true));
    await waitFor(() => expect(result.current.phase).toBe("needs-install"));
    expect(result.current.model?.installed).toBe(false);
    expect(client.analyze).not.toHaveBeenCalled();

    act(() => result.current.installAndRun());
    await waitFor(() => expect(result.current.phase).toBe("done"));
    expect(client.install).toHaveBeenCalledWith("tesseract");
    expect(client.analyze).toHaveBeenCalledWith("ocr", PNG);
    expect(result.current.result?.kind).toBe("ocr");
  });

  it("surfaces a failed run and retries", async () => {
    const analyze = vi
      .fn()
      .mockRejectedValueOnce(new Error("ocr engine missing"))
      .mockResolvedValue(makeProbeResult());
    const client = makeAnalyzeClient({ analyze });
    const { result } = renderHook(() => useAnalyze({ client }));

    act(() => result.current.run("probe", PNG, false));
    await waitFor(() => expect(result.current.phase).toBe("failed"));
    expect(result.current.error).toBe("ocr engine missing");

    act(() => result.current.retry());
    await waitFor(() => expect(result.current.phase).toBe("done"));
    expect(result.current.result?.kind).toBe("probe");
  });

  it("surfaces a selectModel failure (Error.message) for a model-backed op", async () => {
    const client = makeAnalyzeClient({
      selectModel: vi.fn(() => Promise.reject(new Error("registry offline"))),
    });
    const { result } = renderHook(() => useAnalyze({ client }));

    act(() => result.current.run("ocr", PNG, true));

    await waitFor(() => expect(result.current.phase).toBe("failed"));
    expect(result.current.error).toBe("registry offline");
    expect(client.analyze).not.toHaveBeenCalled();
  });

  it("stringifies a non-Error analyze rejection for the failure text", async () => {
    const client = makeAnalyzeClient({
      // eslint-disable-next-line @typescript-eslint/prefer-promise-reject-errors -- intentional non-Error rejection to cover the String(err) branch
      analyze: vi.fn(() => Promise.reject("probe crashed")),
    });
    const { result } = renderHook(() => useAnalyze({ client }));

    act(() => result.current.run("probe", PNG, false));

    await waitFor(() => expect(result.current.phase).toBe("failed"));
    expect(result.current.error).toBe("probe crashed");
  });

  it("installAndRun is a no-op before any run (no pending op/model)", () => {
    const client = makeAnalyzeClient();
    const { result } = renderHook(() => useAnalyze({ client }));

    act(() => result.current.installAndRun());

    expect(result.current.phase).toBe("idle");
    expect(client.install).not.toHaveBeenCalled();
  });

  it("surfaces an install-job failure from the gate", async () => {
    const client = makeAnalyzeClient({
      selectModel: vi.fn(() =>
        Promise.resolve(makeSelectedModel({ id: "tesseract", installed: false })),
      ),
      waitJob: vi.fn(() => Promise.resolve({ ok: false, error: "model download failed" })),
    });
    const { result } = renderHook(() => useAnalyze({ client }));

    act(() => result.current.run("ocr", PNG, true));
    await waitFor(() => expect(result.current.phase).toBe("needs-install"));

    act(() => result.current.installAndRun());
    await waitFor(() => expect(result.current.phase).toBe("failed"));
    expect(result.current.error).toBe("model download failed");
    expect(client.analyze).not.toHaveBeenCalled();
  });

  it("surfaces a thrown install error from the gate", async () => {
    const client = makeAnalyzeClient({
      selectModel: vi.fn(() =>
        Promise.resolve(makeSelectedModel({ id: "tesseract", installed: false })),
      ),
      install: vi.fn(() => Promise.reject(new Error("no network"))),
      analyze: vi.fn(() => Promise.resolve(makeOcrResult())),
    });
    const { result } = renderHook(() => useAnalyze({ client }));

    act(() => result.current.run("ocr", PNG, true));
    await waitFor(() => expect(result.current.phase).toBe("needs-install"));

    act(() => result.current.installAndRun());
    await waitFor(() => expect(result.current.phase).toBe("failed"));
    expect(result.current.error).toBe("no network");
  });

  it("surfaces a thrown waitJob error from the gate", async () => {
    const client = makeAnalyzeClient({
      selectModel: vi.fn(() =>
        Promise.resolve(makeSelectedModel({ id: "tesseract", installed: false })),
      ),
      waitJob: vi.fn(() => Promise.reject(new Error("wait stream broke"))),
      analyze: vi.fn(() => Promise.resolve(makeOcrResult())),
    });
    const { result } = renderHook(() => useAnalyze({ client }));

    act(() => result.current.run("ocr", PNG, true));
    await waitFor(() => expect(result.current.phase).toBe("needs-install"));

    act(() => result.current.installAndRun());
    await waitFor(() => expect(result.current.phase).toBe("failed"));
    expect(result.current.error).toBe("wait stream broke");
  });

  it("skips the install wait when the model is already installed", async () => {
    const client = makeAnalyzeClient({
      selectModel: vi.fn(() =>
        Promise.resolve(makeSelectedModel({ id: "tesseract", installed: false })),
      ),
      install: vi.fn(() => Promise.resolve({ jobId: "", alreadyInstalled: true })),
      analyze: vi.fn(() => Promise.resolve(makeOcrResult())),
    });
    const { result } = renderHook(() => useAnalyze({ client }));

    act(() => result.current.run("ocr", PNG, true));
    await waitFor(() => expect(result.current.phase).toBe("needs-install"));

    act(() => result.current.installAndRun());
    await waitFor(() => expect(result.current.phase).toBe("done"));
    expect(client.waitJob).not.toHaveBeenCalled();
    expect(result.current.result?.kind).toBe("ocr");
  });

  it("cancel() returns an in-flight run to idle and ignores its late result", async () => {
    let resolveAnalyze: ((value: ReturnType<typeof makeProbeResult>) => void) | undefined;
    const client = makeAnalyzeClient({
      analyze: vi.fn(
        () =>
          new Promise<ReturnType<typeof makeProbeResult>>((resolve) => {
            resolveAnalyze = resolve;
          }),
      ),
    });
    const { result } = renderHook(() => useAnalyze({ client }));

    act(() => result.current.run("probe", PNG, false));
    await waitFor(() => expect(result.current.phase).toBe("running"));

    act(() => result.current.cancel());
    expect(result.current.phase).toBe("idle");

    // The superseded analyze resolves late; the run-id guard drops it.
    await act(async () => {
      await Promise.resolve();
      resolveAnalyze?.(makeProbeResult());
    });
    expect(result.current.phase).toBe("idle");
    expect(result.current.result).toBeNull();
  });

  it("clear() resets a terminal result back to idle", async () => {
    const client = makeAnalyzeClient();
    const { result } = renderHook(() => useAnalyze({ client }));

    act(() => result.current.run("probe", PNG, false));
    await waitFor(() => expect(result.current.phase).toBe("done"));

    act(() => result.current.clear());
    expect(result.current.phase).toBe("idle");
    expect(result.current.result).toBeNull();
  });
});
