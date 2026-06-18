/**
 * useAnalyze lifecycle tests. The hook drives one analysis op through
 * (model-backed) select → install-gate → run, or (pure-Go probe) run directly,
 * via the injected AnalyzeClient seam — so the whole synchronous state machine
 * is exercised without a network. Covers the probe happy path, the
 * model-install gate, failure + retry, and clear.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";

import { useAnalyze } from "./useAnalyze";
import { makeAnalyzeClient, makeOcrResult, makeProbeResult } from "./mocks/analysis";
import { makeSelectedModel } from "./mocks/ai";

const PNG = new File(["bytes"], "in.png", { type: "image/png" });

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
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
