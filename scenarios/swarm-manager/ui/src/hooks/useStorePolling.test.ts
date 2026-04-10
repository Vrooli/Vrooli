import { renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useStorePolling } from "./useStorePolling";

describe("useStorePolling", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("polls at the specified interval when enabled", () => {
    const pollFn = vi.fn();

    renderHook(() =>
      useStorePolling({ enabled: true, intervalMs: 5000, pollFn }),
    );

    expect(pollFn).not.toHaveBeenCalled();

    vi.advanceTimersByTime(5000);
    expect(pollFn).toHaveBeenCalledTimes(1);

    vi.advanceTimersByTime(5000);
    expect(pollFn).toHaveBeenCalledTimes(2);
  });

  it("calls pollFn immediately when immediate is true", () => {
    const pollFn = vi.fn();

    renderHook(() =>
      useStorePolling({ enabled: true, intervalMs: 5000, pollFn, immediate: true }),
    );

    expect(pollFn).toHaveBeenCalledTimes(1);

    vi.advanceTimersByTime(5000);
    expect(pollFn).toHaveBeenCalledTimes(2);
  });

  it("does not poll when disabled", () => {
    const pollFn = vi.fn();

    renderHook(() =>
      useStorePolling({ enabled: false, intervalMs: 5000, pollFn }),
    );

    vi.advanceTimersByTime(20_000);
    expect(pollFn).not.toHaveBeenCalled();
  });

  it("stops polling when enabled changes to false", () => {
    const pollFn = vi.fn();

    const { rerender } = renderHook(
      ({ enabled }: { enabled: boolean }) =>
        useStorePolling({ enabled, intervalMs: 5000, pollFn }),
      { initialProps: { enabled: true } },
    );

    vi.advanceTimersByTime(5000);
    expect(pollFn).toHaveBeenCalledTimes(1);

    rerender({ enabled: false });

    vi.advanceTimersByTime(15_000);
    expect(pollFn).toHaveBeenCalledTimes(1);
  });

  it("cleans up interval on unmount", () => {
    const pollFn = vi.fn();

    const { unmount } = renderHook(() =>
      useStorePolling({ enabled: true, intervalMs: 5000, pollFn }),
    );

    vi.advanceTimersByTime(5000);
    expect(pollFn).toHaveBeenCalledTimes(1);

    unmount();

    vi.advanceTimersByTime(15_000);
    expect(pollFn).toHaveBeenCalledTimes(1);
  });

  it("handles async pollFn without errors", () => {
    const pollFn = vi.fn().mockResolvedValue(undefined);

    renderHook(() =>
      useStorePolling({ enabled: true, intervalMs: 3000, pollFn }),
    );

    vi.advanceTimersByTime(3000);
    expect(pollFn).toHaveBeenCalledTimes(1);
  });
});
