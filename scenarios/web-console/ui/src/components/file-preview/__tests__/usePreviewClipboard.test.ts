import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { copyText } from "../../../lib/clipboard";
import { usePreviewClipboard } from "../usePreviewClipboard";

vi.mock("../../../lib/clipboard", () => ({ copyText: vi.fn() }));
afterEach(() => { cleanup(); vi.useRealTimers(); vi.resetAllMocks(); });

describe("preview clipboard feedback", () => {
  it("copies the exact original value and clears truthful success/failure feedback", async () => {
    vi.useFakeTimers();
    vi.mocked(copyText).mockResolvedValueOnce(true).mockResolvedValueOnce(false);
    const { result } = renderHook(usePreviewClipboard);
    await act(async () => { await result.current.copy("001.20\noriginal"); });
    expect(copyText).toHaveBeenCalledWith("001.20\noriginal");
    expect(result.current.status).toBe("copied");
    act(() => { vi.advanceTimersByTime(2000); });
    expect(result.current.status).toBeNull();
    await act(async () => { await result.current.copy("denied"); });
    expect(result.current.status).toBe("failed");
  });
  it("ignores an older write that completes after a newer write", async () => {
    let complete: (value: boolean) => void = () => {};
    vi.mocked(copyText).mockImplementationOnce(() => new Promise((resolve) => { complete = resolve; })).mockResolvedValueOnce(false);
    const { result, unmount } = renderHook(usePreviewClipboard);
    let first: Promise<void> | undefined;
    act(() => { first = result.current.copy("first"); });
    await act(async () => { await result.current.copy("second"); });
    await act(async () => { complete(true); await first; });
    expect(result.current.status).toBe("failed");
    unmount();
  });
  it("does not publish feedback or schedule timers after the preview closes", async () => {
    vi.useFakeTimers();
    let complete: (value: boolean) => void = () => {};
    vi.mocked(copyText).mockImplementationOnce(() => new Promise((resolve) => { complete = resolve; }));
    const { result, unmount } = renderHook(usePreviewClipboard);
    let pending: Promise<void> | undefined;
    act(() => { pending = result.current.copy("value"); });
    unmount();
    await act(async () => { complete(true); await pending; });
    expect(vi.getTimerCount()).toBe(0);
    expect(result.current.status).toBeNull();
  });
});
