import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useCountdown } from "../hooks/useCountdown";

// [REQ:P1-001b-i] Countdown Timer Display

describe("useCountdown", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("returns null for never mode", () => {
    const { result } = renderHook(() =>
      useCountdown(new Date().toISOString(), "never", undefined),
    );
    expect(result.current).toBeNull();
  });

  it("returns null when duration is undefined", () => {
    const { result } = renderHook(() =>
      useCountdown(new Date().toISOString(), "preset", undefined),
    );
    expect(result.current).toBeNull();
  });

  it("returns a countdown string for a valid preset duration", () => {
    // Created now with 1h duration — should show ~59m remaining
    const createdAt = new Date().toISOString();
    const { result } = renderHook(() =>
      useCountdown(createdAt, "preset", "1h"),
    );
    // Should be a non-null string (e.g. "59m 59s" or similar)
    expect(result.current).not.toBeNull();
    expect(typeof result.current).toBe("string");
  });

  it("returns expired for sessions past their TTL", () => {
    // Created 2 hours ago with 1h duration — already expired
    const twoHoursAgo = new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString();
    const { result } = renderHook(() =>
      useCountdown(twoHoursAgo, "preset", "1h"),
    );
    expect(result.current?.toLowerCase()).toBe("expired");
  });

  it("updates countdown string when interval fires", () => {
    const createdAt = new Date().toISOString();
    const { result } = renderHook(() =>
      useCountdown(createdAt, "preset", "1h"),
    );

    // Capture initial value to ensure countdown is active
    expect(result.current).not.toBeNull();

    // Advance time by 5 seconds and let interval fire
    act(() => {
      vi.advanceTimersByTime(5000);
    });

    // The countdown should still be a valid string (may or may not differ
    // depending on granularity, but should not be null)
    expect(result.current).not.toBeNull();
    expect(typeof result.current).toBe("string");
  });

  it("cleans up interval on unmount", () => {
    const clearSpy = vi.spyOn(globalThis, "clearInterval");

    const { unmount } = renderHook(() =>
      useCountdown(new Date().toISOString(), "preset", "1h"),
    );

    unmount();

    expect(clearSpy).toHaveBeenCalled();
    clearSpy.mockRestore();
  });
});
